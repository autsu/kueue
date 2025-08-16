#!/bin/bash

# 查找所有导入了 functrace 包的 Go 文件
files=$(grep -l "github.com/toheart/functrace" --include="*.go" -r .)

# 遍历每个文件并移除 functrace 导入
for file in $files; do
  echo "处理文件: $file"
  # 使用 sed 移除导入行
  sed -i '' '/"github.com\/toheart\/functrace"/d' "$file"
  # 如果导入块变成空的，修复语法
  sed -i '' 's/import (\s*)/import (/g' "$file"
  sed -i '' 's/import (\s*))/import ()/g' "$file"
done

echo "完成清理 functrace 导入！"