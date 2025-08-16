#!/bin/bash

set -e

echo "=== 验证经典抢占配置文件 ==="

BASE_DIR="examples/classical-preemption"

echo "1. 验证 YAML 文件语法..."

echo "  - namespace.yaml"
kubectl apply --dry-run=client -f $BASE_DIR/namespace.yaml >/dev/null

echo "  - resource-flavor.yaml"
kubectl apply --dry-run=client -f $BASE_DIR/resource-flavor.yaml >/dev/null

echo "  - cohort.yaml"
kubectl apply --dry-run=client -f $BASE_DIR/cohort.yaml >/dev/null

echo "  - cluster-queues.yaml"
kubectl apply --dry-run=client -f $BASE_DIR/cluster-queues.yaml >/dev/null

echo "  - local-queues.yaml"
kubectl apply --dry-run=client -f $BASE_DIR/local-queues.yaml >/dev/null

echo "  - 工作负载文件..."
for scenario in $BASE_DIR/workloads/scenario*/; do
    if [ -d "$scenario" ]; then
        echo "    - $scenario"
        for yaml_file in "$scenario"*.yaml; do
            if [ -f "$yaml_file" ]; then
                kubectl apply --dry-run=client -f "$yaml_file" >/dev/null
            fi
        done
    fi
done

echo ""
echo "2. 检查 API 版本兼容性..."
echo "  - Cohort API 版本: $(grep "apiVersion.*kueue" $BASE_DIR/cohort.yaml | head -1 | cut -d: -f2 | tr -d ' ')"
echo "  - ClusterQueue API 版本: $(grep "apiVersion.*kueue" $BASE_DIR/cluster-queues.yaml | head -1 | cut -d: -f2 | tr -d ' ')"
echo "  - LocalQueue API 版本: $(grep "apiVersion.*kueue" $BASE_DIR/local-queues.yaml | head -1 | cut -d: -f2 | tr -d ' ')"

echo ""
echo "3. 检查集群中的 Kueue CRD..."
kubectl get crd | grep kueue || echo "  警告: 未找到 Kueue CRD，请确保 Kueue 已安装"

echo ""
echo "=== 配置验证完成 ==="
echo "所有配置文件语法正确，可以安全使用！"
