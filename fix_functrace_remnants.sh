#!/bin/bash

# 查找所有包含 }{ 模式的 Go 文件
files=$(grep -l "}{"|"defer" --include="*.go" -r .)

# 遍历每个文件并修复语法错误
for file in $files; do
  echo "处理文件: $file"
  # 使用 sed 移除包含 }{ 的行及其前后的行
  sed -i '' -E '/\{[[:space:]]*$/,/\}\{.*\}\)\(\)/d' "$file"
  # 移除孤立的 defer 语句
  sed -i '' -E '/defer[[:space:]]*$/d' "$file"
done

echo "完成修复语法错误！"