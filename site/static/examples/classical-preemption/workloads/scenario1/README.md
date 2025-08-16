# 场景 1：队列内抢占 (WithinClusterQueue)

这个场景演示同一个 ClusterQueue 内的高优先级工作负载抢占低优先级工作负载。

## 执行步骤

1. 首先提交低优先级工作负载占用资源：

```bash
kubectl apply -f low-priority-workload.yaml
```

2. 等待工作负载被调度，然后提交高优先级工作负载：

```bash
kubectl apply -f high-priority-workload.yaml
```

## 预期结果

- 低优先级工作负载会被抢占
- 高优先级工作负载会被调度
- 日志中会显示 `result=WithinCQ` 的抢占分类
