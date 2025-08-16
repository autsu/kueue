#!/bin/bash

# 清理 Kueue 项目中的打桩代码
# 这个脚本会删除由打桩库自动注入的代码，包括：
# 1. defer functrace.Trace([]interface {...})() 形式的代码
# 2. // ([]interface {...})() 形式的注释代码

set -e

ROOT_DIR=$(dirname "$(dirname "$(realpath "$0")")")
cd "$ROOT_DIR"

echo "开始清理打桩代码..."

# 清理 defer functrace.Trace 形式的代码
find . -name "*.go" -type f -exec sed -i '' '/defer functrace\.Trace(\[\]interface {/,/})()/d' {} \;

# 清理 // ([]interface 形式的注释代码
find . -name "*.go" -type f -exec sed -i '' '/\/\/ (\[\]interface {/,/})()/d' {} \;

# 清理 functrace.Trace 形式的代码（非 defer 形式）
find . -name "*.go" -type f -exec sed -i '' '/functrace\.Trace(\[\]interface {/,/})()/d' {} \;

echo "清理完成！"