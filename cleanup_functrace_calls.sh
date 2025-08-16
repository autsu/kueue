#!/bin/bash

# 查找所有包含 functrace 调用的 Go 文件
files=$(grep -l "functrace" --include="*.go" -r .)

# 遍历每个文件并移除 functrace 调用行
for file in $files; do
  echo "处理文件: $file"
  # 使用 sed 移除包含 functrace 的行
  sed -i '' '/functrace/d' "$file"
done

echo "完成清理 functrace 调用！"