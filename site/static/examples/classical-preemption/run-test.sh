#!/bin/bash

set -e

echo "=== 经典抢占算法观测实践 ==="

# 检查 kubectl 是否可用
if ! command -v kubectl &> /dev/null; then
    echo "错误: kubectl 未安装或不在 PATH 中"
    exit 1
fi

# 应用基础配置
echo "1. 应用基础配置..."
kubectl apply -f namespace.yaml
kubectl apply -f resource-flavor.yaml
kubectl apply -f cohort.yaml
kubectl apply -f cluster-queues.yaml
kubectl apply -f local-queues.yaml

echo "等待资源创建完成..."
sleep 5

# 检查资源状态
echo "2. 检查 ClusterQueue 状态..."
kubectl get clusterqueue

echo ""
echo "3. 开始测试场景..."

# 场景选择
echo "请选择要测试的场景："
echo "1) 队列内抢占 (WithinClusterQueue)"
echo "2) Cohort 内回收 (ReclaimWithinCohort)"
echo "3) 借用时抢占 (BorrowWithinCohort)"
echo "4) 运行所有场景"
read -p "请输入选择 (1-4): " choice

run_scenario() {
    local scenario=$1
    echo ""
    echo "=== 运行场景 $scenario ==="
    
    # 清理之前的工作负载
    kubectl delete jobs -n preemption-test --all --ignore-not-found=true
    sleep 3
    
    case $scenario in
        1)
            echo "场景 1: 队列内抢占"
            echo "提交低优先级工作负载..."
            kubectl apply -f workloads/scenario1/low-priority-workload.yaml
            sleep 10
            echo "提交高优先级工作负载..."
            kubectl apply -f workloads/scenario1/high-priority-workload.yaml
            ;;
        2)
            echo "场景 2: Cohort 内回收"
            echo "提交借用资源的工作负载..."
            kubectl apply -f workloads/scenario2/borrowing-workload.yaml
            sleep 10
            echo "提交回收资源的工作负载..."
            kubectl apply -f workloads/scenario2/reclaiming-workload.yaml
            ;;
        3)
            echo "场景 3: 借用时抢占"
            echo "提交受害者工作负载..."
            kubectl apply -f workloads/scenario3/victim-workload.yaml
            sleep 10
            echo "提交借用抢占者工作负载..."
            kubectl apply -f workloads/scenario3/borrowing-preemptor.yaml
            ;;
    esac
    
    echo "等待抢占发生..."
    sleep 15
    
    echo "工作负载状态:"
    kubectl get workloads -n preemption-test
    echo ""
    echo "Job 状态:"
    kubectl get jobs -n preemption-test
}

case $choice in
    1)
        run_scenario 1
        ;;
    2)
        run_scenario 2
        ;;
    3)
        run_scenario 3
        ;;
    4)
        run_scenario 1
        sleep 30
        run_scenario 2
        sleep 30
        run_scenario 3
        ;;
    *)
        echo "无效选择"
        exit 1
        ;;
esac

echo ""
echo "=== 测试完成 ==="
echo "请查看 Kueue 日志以观测抢占算法的工作过程："
echo "kubectl logs -f deployment/kueue-controller-manager -n kueue-system | grep 'classical-preemption'"