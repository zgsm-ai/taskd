#!/bin/bash

#
#   shenma-secret.sh将模板文件(template)中的变量引用替换为私密变量文件(envfile)中读取的变量值。
#
#   K8S的YAML可以配置应用程序，但完全固化的配置文件会遇到两个问题：
#   1. 部分配置需要根据环境不同而设置，所以需要根据环境标识进行切换；
#   2. 部分私密信息不适合直接保存在会对外公开的仓库中，需要保存在私有仓库中；
#   解决这两个问题，有一个简单的方案：
#   在项目GIT仓库中只保存YAML模板文件，
#   把需要根据环境变化的内容，以及需要保密的内容以`变量名=变量值`的形式保存在私密变量文件中。
#   模板文件中使用`\$\{\{name ('.' name)*\}\}`形式的变量引用标识引用私密变量文件中的变量值。
#   真正部署的时候，调用shenma-secret.sh对模板文件进行转换，完成变量名到值的替换。
#   
#   变量引用有两种，分两轮进行替换：
#       一种是__special_image_tag,__env_profile形式的，会在第一轮中被替换
#       一种是`\$\{\{name ('.' name)+\}\}`形式的，会在第二轮中被替换
#   形如${{__env_profile.label.pgsql.user}}的变量会在第一轮替换中变成${{prd.label.pgsql.user}},
#   在第二轮替换中变成envfile中prd_label_pgsql_user变量的值。
#

function usage() {
    echo "shenma-secret.sh -d {deployment} [-p profile] [-v version] [-t template] [-e envfile] [-h]"
    echo "  deployment: 需要部署的服务名称"
    echo "  template: YAML模板文件,默认:{deployment}.yaml.template"
    echo "  profile: 环境标识,支持:prd(生产环境),test(测试环境),dev(开发调试环境),默认:prd"
    echo "  version: 版本标识,默认:latest_{timestamp}"
    echo "shenma-secret.sh从envfile中读取变量值,填充模板文件中的变量{{profile.xxx...}},结果输出到文件__{deployment}_{profile}_{version}.yaml中"
    echo "shenma-secret.sh替换变量规则:"
    echo "  __special_image_tag     替换为{version}"
    echo "  __env_profile           替换为{profile}"
    echo "  {{VARIABLE}}...  替换为envfile中VARIABLE的内容"
    echo "  由{{和}}括起来的变量varname，将以'_'替换其中的'.'，然后作为变量名"
    echo "  读取envfile中对应变量的值，替换掉varname以及两侧的{{}}。"
}

while getopts "d:p:v:t:e:h" opt
do
    case $opt in
        d)
        deployment=$OPTARG;;
        p)
        profile=$OPTARG;;
        t)
        template=$OPTARG;;
        v)
        version=$OPTARG;;
        e)
        envfile=$OPTARG;;
        h)
        usage
        exit 0;;
        ?)
        usage
        exit 1;;
    esac
done

if [ $# -eq 0 ]; then
    usage
    exit 0
fi

# 服务名
if [ ""X == "$deployment"X ]; then
    usage
    exit 1
fi
# YAML模板文件
if [ ""X == "$template"X ]; then
    template=${deployment}.yaml.template
fi
# 工作环境：prod,test,dev
if [ ""X == "$profile"X ]; then
    profile=prod
fi
# 版本
if [ ""X == "$version"X ]; then
    timestamp=`date "+%Y%m%d"`
    version=latest_${timestamp}
fi
# 环境变量文件
if [ ""X == "$envfile"X ]; then
    envfile=".env"
fi

# 用户输入参数
echo deployment: $deployment
echo profile: $profile
echo version: $version
echo template: $template
echo envfile: $envfile

workspace=`pwd`
filename="__${deployment}_${profile}_${version}.yaml"

cat $template | sed "s/__special_image_tag/${version}/g" | sed "s/__env_profile/${profile}/g" > $workspace/$filename
sed -i "s/\${{IMAGE_TIMESTAMP}}/$version/g" $workspace/$filename
sed -i "s/\${{ENV_PROFILE}}/$profile/g" $workspace/$filename

source ${envfile}

# 查询所有${{变量}}，并进行值替换
for i in `cat $workspace/$filename | grep -o -w -E "\\\\$\{\{([[:alnum:]]|\.|_|\+)*\}\}"|sort|uniq`; do
    # 提取键名
    content=${i:3:(${#i}-5)};
    echo $content;
    # 把点分键名序列格式化为环境变量名，即，将所有'.'替换为'_'
    content_key=`echo $content | sed  "s#\.#\_#g"`;
    echo $content_key
    # 获取键值
    value="${!content_key}";
    echo $value;
    # 替换文件内容
    if echo "$value" | grep -vq '#'; then
        sed -i "s#$i#$value#g" $workspace/$filename;
    elif echo "$value" | grep -vq '/'; then
        sed -i "s/$i/$value/g" $workspace/$filename;
    elif echo "$value" | grep -vq ','; then
        sed -i "s,$i,$value,g" $workspace/$filename;
    else
        echo "$content_key中同时包含特殊字符“#/,”，无法执行替换"
    fi
    # sed -i "s#$i#$value#g" $workspace/$filename;
done

# 部署
echo apply:
echo kubectl apply -f $workspace/$filename
