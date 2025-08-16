/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package classical

import (
	"k8s.io/apimachinery/pkg/util/sets"
	ctrl "sigs.k8s.io/controller-runtime"

	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/kueue/pkg/cache"
	"sigs.k8s.io/kueue/pkg/resources"
	"sigs.k8s.io/kueue/pkg/util/priority"
	"sigs.k8s.io/kueue/pkg/workload"
)

type preemptionVariant int

const (
	// Cannot be preempted
	Never preemptionVariant = iota
	// Candidate within the same CQ as the preemptor
	WithinCQ
	// Preemptor has preferential access to the resources needing preemption
	// over the candidate, because of its CQ position in the cohort topology.
	HiearchicalReclaim
	// Can only be preempted if preemptor CQ (after all preemptions and the
	// admission of the incoming workload) would not be borrowing any quota
	ReclaimWithoutBorrowing
	// Can be preemped even if preemptor CQ would be borrowing
	ReclaimWhileBorrowing
)

func (m preemptionVariant) PreemptionReason() string {
	switch m {
	case WithinCQ:
		return kueue.InClusterQueueReason
	case HiearchicalReclaim:
		return kueue.InCohortReclamationReason
	case ReclaimWhileBorrowing:
		return kueue.InCohortReclaimWhileBorrowingReason
	case ReclaimWithoutBorrowing:
		return kueue.InCohortReclamationReason
	}
	return "Unknown"
}

type HierarchicalPreemptionCtx struct {
	Wl                *kueue.Workload
	Cq                *cache.ClusterQueueSnapshot
	FrsNeedPreemption sets.Set[resources.FlavorResource]
	Requests          resources.FlavorResourceQuantities
	WorkloadOrdering  workload.Ordering
}

// 检查是否禁止在 cohort 内借用时抢占
func IsBorrowingWithinCohortForbidden(cq *cache.ClusterQueueSnapshot) (bool, *int32) {
	borrowWithinCohort := cq.Preemption.BorrowWithinCohort
	if borrowWithinCohort == nil || borrowWithinCohort.Policy == kueue.BorrowWithinCohortPolicyNever {
		return true, nil
	}
	return false, borrowWithinCohort.MaxPriorityThreshold
}

// classifyPreemptionVariant evaluates, based on config and priorities, the
// preemption type for a given candidate
func classifyPreemptionVariant(ctx *HierarchicalPreemptionCtx, wl *workload.Info, haveHierarchicalAdvantage bool) preemptionVariant {

	setupLog := ctrl.Log.WithName("classical-preemption").WithValues(
		"preemptor", ctx.Wl.Name,
		"preemptorCQ", ctx.Cq.Name,
		"candidate", wl.Obj.Name,
		"candidateCQ", wl.ClusterQueue,
	)

	if !WorkloadUsesResources(wl, ctx.FrsNeedPreemption) {
		setupLog.V(2).Info("Candidate doesn't use resources needed for preemption", "result", "Never")
		return Never
	}

	incomingPriority := priority.Priority(ctx.Wl)
	candidatePriority := priority.Priority(wl.Obj)
	setupLog.V(2).Info("Priority comparison",
		"incomingPriority", incomingPriority,
		"candidatePriority", candidatePriority,
		"hierarchicalAdvantage", haveHierarchicalAdvantage)

	if !satisfiesPreemptionPolicy(ctx, wl, incomingPriority, candidatePriority) {
		setupLog.V(2).Info("Candidate doesn't satisfy preemption policy", "result", "Never")
		return Never
	}

	if wl.ClusterQueue == ctx.Cq.Name {
		setupLog.V(2).Info("Same ClusterQueue preemption", "result", "WithinCQ")
		return WithinCQ
	}

	if haveHierarchicalAdvantage {
		setupLog.V(2).Info("Hierarchical advantage preemption", "result", "HierarchicalReclaim")
		return HiearchicalReclaim
	}

	borrowWithinCohortForbidden, borrowWithinCohortThreshold := IsBorrowingWithinCohortForbidden(ctx.Cq)
	setupLog.V(2).Info("BorrowWithinCohort check",
		"forbidden", borrowWithinCohortForbidden,
		"threshold", borrowWithinCohortThreshold)

	if borrowWithinCohortForbidden {
		setupLog.V(2).Info("Borrowing within cohort forbidden", "result", "ReclaimWithoutBorrowing")
		return ReclaimWithoutBorrowing
	}

	if isAboveBorrowingThreshold(candidatePriority, incomingPriority, borrowWithinCohortThreshold) {
		setupLog.V(2).Info("Above borrowing threshold", "result", "ReclaimWithoutBorrowing")
		return ReclaimWithoutBorrowing
	}

	setupLog.V(2).Info("Can reclaim while borrowing", "result", "ReclaimWhileBorrowing")
	return ReclaimWhileBorrowing
}

func satisfiesPreemptionPolicy(ctx *HierarchicalPreemptionCtx, wl *workload.Info, incomingPriority, candidatePriority int32) bool {
	var preemptionPolicy kueue.PreemptionPolicy
	if wl.ClusterQueue == ctx.Cq.Name {
		preemptionPolicy = ctx.Cq.Preemption.WithinClusterQueue
	} else {
		preemptionPolicy = ctx.Cq.Preemption.ReclaimWithinCohort
	}
	lowerPriority := incomingPriority > candidatePriority
	if preemptionPolicy == kueue.PreemptionPolicyLowerPriority {
		return lowerPriority
	}
	if preemptionPolicy == kueue.PreemptionPolicyLowerOrNewerEqualPriority {
		preemptorTS := ctx.WorkloadOrdering.GetQueueOrderTimestamp(ctx.Wl)
		newerEqualPriority := (incomingPriority == candidatePriority) && preemptorTS.Before(ctx.WorkloadOrdering.GetQueueOrderTimestamp(wl.Obj))
		return (lowerPriority || newerEqualPriority)
	}
	return preemptionPolicy == kueue.PreemptionPolicyAny
}

func isAboveBorrowingThreshold(candidatePriority, incomingPriority int32, borrowWithinCohortThreshold *int32) bool {
	if candidatePriority >= incomingPriority {
		return true
	}
	if borrowWithinCohortThreshold == nil {
		return false
	}
	return candidatePriority > *borrowWithinCohortThreshold
}

func collectSameQueueCandidates(ctx *HierarchicalPreemptionCtx) []*candidateElem {
	if ctx.Cq.Preemption.WithinClusterQueue == kueue.PreemptionPolicyNever {
		return []*candidateElem{}
	}
	return getCandidatesFromCQ(ctx.Cq, nil, ctx, false)
}

func getCandidatesFromCQ(cq *cache.ClusterQueueSnapshot, lca *cache.CohortSnapshot, ctx *HierarchicalPreemptionCtx, hasHiearchicalAdvantage bool) []*candidateElem {
	candidates := []*candidateElem{}
	for _, candidateWl := range cq.Workloads {
		preemptionVariant := classifyPreemptionVariant(ctx, candidateWl, hasHiearchicalAdvantage)
		if preemptionVariant == Never {
			continue
		}
		candidates = append(candidates,
			&candidateElem{
				wl:                candidateWl,
				lca:               lca,
				preemptionVariant: preemptionVariant,
			})
	}
	return candidates
}

func collectCandidatesForHierarchicalReclaim(ctx *HierarchicalPreemptionCtx) ([]*candidateElem, []*candidateElem) {
	hierarchyCandidates := []*candidateElem{}
	priorityCandidates := []*candidateElem{}
	if !ctx.Cq.HasParent() || ctx.Cq.Preemption.ReclaimWithinCohort == kueue.PreemptionPolicyNever {
		return hierarchyCandidates, priorityCandidates
	}
	var previousSubtreeRoot *cache.CohortSnapshot
	var candidateList *[]*candidateElem
	var fits bool
	hasHierarchicalAdvantage, remainingRequests := cache.QuantitiesFitInQuota(ctx.Cq, ctx.Requests)
	for currentSubtreeRoot := range ctx.Cq.PathParentToRoot() {
		if hasHierarchicalAdvantage {
			candidateList = &hierarchyCandidates
		} else {
			candidateList = &priorityCandidates
		}
		collectCandidatesInSubtree(ctx, currentSubtreeRoot, currentSubtreeRoot, previousSubtreeRoot, hasHierarchicalAdvantage, candidateList)
		fits, remainingRequests = cache.QuantitiesFitInQuota(currentSubtreeRoot, remainingRequests)
		// Once we find a subtree sT that fits the requests, we will look for workloads that use quota
		// of that subtree. The preemptor will have hierarchical advantage over all such workloads
		// because it belongs to subtree sT. For that reason variable hasHierarchicalAdvantage
		// remains true in subsequent iterations of the loop.
		hasHierarchicalAdvantage = hasHierarchicalAdvantage || fits
		previousSubtreeRoot = currentSubtreeRoot
	}
	return hierarchyCandidates, priorityCandidates
}

// visit the nodes in the hierarchy and collect the ones that exceed quota
// avoid subtrees that are within quota and the skipped subtree
func collectCandidatesInSubtree(ctx *HierarchicalPreemptionCtx, currentCohort *cache.CohortSnapshot, subtreeRoot *cache.CohortSnapshot, skipSubtree *cache.CohortSnapshot, hasHierarchicalAdvantage bool, result *[]*candidateElem) {
	// we already processed this subtree
	for _, childCohort := range currentCohort.ChildCohorts() {

		if childCohort == skipSubtree {
			continue
		}
		// don't look for candidates in subtrees that are not exceeding their quotas
		if cache.IsWithinNominalInResources(childCohort, ctx.FrsNeedPreemption) {
			continue
		}
		collectCandidatesInSubtree(ctx, childCohort, subtreeRoot, skipSubtree, hasHierarchicalAdvantage, result)
	}
	for _, childCq := range currentCohort.ChildCQs() {
		if childCq == ctx.Cq {
			continue
		}
		if !cache.IsWithinNominalInResources(childCq, ctx.FrsNeedPreemption) {
			*result = append(*result, getCandidatesFromCQ(childCq, subtreeRoot, ctx, hasHierarchicalAdvantage)...)
		}
	}
}

// getNodeHeight calculates the distance to the furthest leaf
func getNodeHeight(node *cache.CohortSnapshot) int {
	maxHeight := min(node.ChildCount(), 1)
	for _, childCohort := range node.ChildCohorts() {
		maxHeight = max(maxHeight, getNodeHeight(childCohort)+1)
	}
	return maxHeight
}

// FindHeightOfLowestSubtreeThatFits returns height of a lowest subtree in the cohort
// that fits additional val of resource fr. If no such subtree exists, it returns
// height the whole cohort hierarchy. Note that height of a trivial subtree
// with only one node is 0. It also returns if the returned subtree is smaller than the whole cohort tree.
//
// 这个函数的目的是找到能够容纳额外资源请求的最小子树，并返回该子树的高度。换句话说，
// 它要回答："我需要向上借用到第几层才能满足资源需求？"
func FindHeightOfLowestSubtreeThatFits(c *cache.ClusterQueueSnapshot, fr resources.FlavorResource, val int64) (int, bool) {
	// !c.BorrowingWith(fr, val) 当前使用量 + 请求量 <= 名义配额，则不需要借用
	// !c.HasParent 代表都没父节点，无处可以借用

	if !c.BorrowingWith(fr, val) || !c.HasParent() {
		// 返回 (0, c.HasParent())，表示高度为0（本地资源足够）
		return 0, c.HasParent()
	}

	// LocalAvailable 的含义：
	// func LocalAvailable(node flatResourceNode, fr resources.FlavorResource) int64 {
	//     return max(0, node.getResourceNode().guaranteedQuota(fr)-node.getResourceNode().Usage[fr])
	// }
	// 本地保证配额中未使用的部分
	// guaranteedQuota - Usage 的正数部分
	// 请求: 500 CPU
	// LocalAvailable: 200 CPU (本地可用)
	// remaining = 500 - 200 = 300 CPU (需要从父节点借用)
	remaining := val - cache.LocalAvailable(c, fr)
	// 向上遍历寻找合适的子树
	for trackingNode := range c.PathParentToRoot() {
		// 找到资源满足的节点了
		// return c.ResourceNode.Usage[fr]+val > c.ResourceNode.SubtreeQuota[fr]
		if !trackingNode.BorrowingWith(fr, remaining) {
			// 返回高度
			// 走到这里说明 c 必定不是根节点，所以第二个返回值必定是 true？
			return getNodeHeight(trackingNode), trackingNode.HasParent()
		}
		// 还需要继续借用
		remaining -= cache.LocalAvailable(trackingNode, fr)
	}
	// no fit found
	return getNodeHeight(c.Parent().Root()), false
}
