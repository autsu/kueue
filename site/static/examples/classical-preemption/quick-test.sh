#!/bin/bash

set -e

echo "=== 快速测试经典抢占日志 ==="

# 检查 kubectl 是否可用
if ! command -v kubectl &>/dev/null; then
    echo "错误: kubectl 未安装或不在 PATH 中"
    exit 1
fi

echo "1. 清理之前的资源..."
kubectl delete namespace preemption-test --ignore-not-found=true
sleep 5

echo "2. 应用基础配置..."
kubectl apply -f namespace.yaml
kubectl apply -f resource-flavor.yaml
kubectl apply -f cohort.yaml
kubectl apply -f cluster-queues.yaml
kubectl apply -f local-queues.yaml

echo "等待资源创建完成..."
sleep 10

echo "3. 检查 ClusterQueue 状态..."
kubectl get clusterqueue

echo ""
echo "4. 运行简单的队列内抢占测试..."

# 提交低优先级工作负载
echo "提交低优先级工作负载..."
kubectl apply -f workloads/scenario1/low-priority-workload.yaml

echo "等待工作负载被调度..."
sleep 15

echo "当前工作负载状态:"
kubectl get workloads -n preemption-test

echo ""
echo "提交高优先级工作负载触发抢占..."
kubectl apply -f workloads/scenario1/high-priority-workload.yaml

echo "等待抢占发生..."
sleep 20

echo ""
echo "最终工作负载状态:"
kubectl get workloads -n preemption-test

echo ""
echo "=== 测试完成 ==="
echo "请查看 Kueue 日志以观测抢占算法的工作过程："
echo "kubectl logs deployment/kueue-controller-manager -n kueue-system | grep 'classical-preemption'"
