# 经典抢占算法观测实践

本目录包含用于观测 Kueue 经典抢占算法工作流程的 YAML 文件和文档。

## 实验场景

我们将创建以下场景来观测经典抢占的不同情况：

### 场景 1：队列内抢占 (WithinClusterQueue)

- 同一个 ClusterQueue 内的高优先级工作负载抢占低优先级工作负载

### 场景 2：Cohort 内回收 (ReclaimWithinCohort)

- 工作负载符合名义配额，回收被其他队列借用的资源

### 场景 3：借用时抢占 (BorrowWithinCohort)

- 工作负载需要借用资源时，抢占其他队列的低优先级工作负载

## 文件说明

- `namespace.yaml` - 创建测试命名空间
- `resource-flavor.yaml` - 定义资源类型 (v1beta1)
- `cohort.yaml` - 创建 Cohort (v1alpha1)
- `cluster-queues.yaml` - 创建多个 ClusterQueue (v1beta1)
- `local-queues.yaml` - 创建 LocalQueue (v1beta1)
- `workloads/` - 包含不同优先级的测试工作负载

**注意**: Cohort 资源目前使用 `v1alpha1` API 版本，其他 Kueue 资源使用 `v1beta1`。

## 使用方法

1. 启动 Kueue 并启用详细日志：

```bash
make run
```

注意：Makefile 已配置为启用调试级别日志 (`--zap-log-level=debug --zap-devel`)

2. 应用基础配置：

```bash
kubectl apply -f examples/classical-preemption/namespace.yaml
kubectl apply -f examples/classical-preemption/resource-flavor.yaml
kubectl apply -f examples/classical-preemption/cohort.yaml
kubectl apply -f examples/classical-preemption/cluster-queues.yaml
kubectl apply -f examples/classical-preemption/local-queues.yaml
```

3. 运行不同场景的测试：

```bash
# 场景 1：队列内抢占
kubectl apply -f examples/classical-preemption/workloads/scenario1/

# 场景 2：Cohort 内回收
kubectl apply -f examples/classical-preemption/workloads/scenario2/

# 场景 3：借用时抢占
kubectl apply -f examples/classical-preemption/workloads/scenario3/
```

4. 或者运行快速测试：

```bash
cd examples/classical-preemption
./quick-test.sh
```

## 观测要点

通过日志观测以下关键信息：

1. **抢占候选者分类**：
   - `classifyPreemptionVariant` 函数的输出
   - 候选者被分类为哪种抢占类型

2. **抢占策略选择**：
   - 是否允许借用 (`allowBorrowing`)
   - 尝试的抢占策略顺序

3. **抢占过程**：
   - 候选者评估顺序
   - 每个候选者的抢占原因
   - 工作负载是否能够放入

4. **回填过程**：
   - 哪些工作负载被重新加回
   - 最终的抢占集合

## 日志级别

- `V(1)`: 主要的抢占流程和决策
- `V(2)`: 详细的候选者评估过程
- `Info`: 重要的抢占结果和错误

使用以下命令查看详细日志：

```bash
# 查看所有抢占相关日志
kubectl logs -f deployment/kueue-controller-manager -n kueue-system | grep "classical-preemption"

# 查看特定级别的日志
kubectl logs -f deployment/kueue-controller-manager -n kueue-system | grep -E "(classical-preemption|preemption)"
```
