# 场景 2：Cohort 内回收 (ReclaimWithinCohort)

这个场景演示工作负载符合名义配额时，回收被其他队列借用的资源。

## 执行步骤

1. 首先在 medium-priority-cq 中提交工作负载，让它借用资源：

```bash
kubectl apply -f borrowing-workload.yaml
```

2. 等待工作负载被调度，然后在 high-priority-cq 中提交符合名义配额的工作负载：

```bash
kubectl apply -f reclaiming-workload.yaml
```

## 预期结果

- medium-priority-cq 中的工作负载会被抢占（因为它在借用资源）
- high-priority-cq 中的工作负载会被调度
- 日志中会显示 `result=HierarchicalReclaim` 的抢占分类
