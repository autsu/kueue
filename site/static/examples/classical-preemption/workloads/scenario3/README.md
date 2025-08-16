# 场景 3：借用时抢占 (BorrowWithinCohort)

这个场景演示工作负载需要借用资源时，抢占其他队列的低优先级工作负载。

## 执行步骤

1. 首先在 low-priority-cq 中提交低优先级工作负载：

```bash
kubectl apply -f victim-workload.yaml
```

2. 等待工作负载被调度，然后在 high-priority-cq 中提交需要借用资源的高优先级工作负载：

```bash
kubectl apply -f borrowing-preemptor.yaml
```

## 预期结果

- low-priority-cq 中的低优先级工作负载会被抢占
- high-priority-cq 中的工作负载会被调度并借用资源
- 日志中会显示 `result=ReclaimWhileBorrowing` 的抢占分类
