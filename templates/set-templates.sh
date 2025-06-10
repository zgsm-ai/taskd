#!/bin/bash

# 获取ls命令的输出结果，并逐行遍历
tpl_output=$(aim taskd list)
while IFS= read -r file; do
    # 执行命令aim taskd list，并传入当前文件作为参数
   aim taskd set "$file" --template ./$file.template.yaml
done <<< "$tpl_output"

