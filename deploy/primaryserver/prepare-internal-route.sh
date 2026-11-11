#!/bin/sh
# 把内网映射合入已有配置。此脚本不启动或重建容器。
set -eu
umask 077

if [ "$#" -ne 1 ]; then
    echo "用法：sh prepare-internal-route.sh <部署目录>" >&2
    exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
target_dir=$(CDPATH= cd -- "$1" && pwd)
base="$target_dir/docker-compose.yml"
overlay="$script_dir/compose.vm2api-internal.yml"

if [ ! -f "$base" ]; then
    echo "找不到已有 docker-compose.yml。" >&2
    exit 1
fi

rendered=$(mktemp "$target_dir/.docker-compose.vm2api.XXXXXX")
trap 'rm -f "$rendered"' EXIT
trap 'exit 1' HUP INT TERM

# 保留变量表达式、相对路径和 env_file 引用。不要导出生产密钥。
docker compose --project-directory "$target_dir" -f "$base" -f "$overlay" \
    config --no-interpolate --no-env-resolution --no-path-resolution > "$rendered"
docker compose --project-directory "$target_dir" -f "$rendered" config --quiet

if cmp -s "$base" "$rendered"; then
    echo "内网映射已存在，配置无需更新。"
    exit 0
fi

backup=$(mktemp "$target_dir/.docker-compose.before-vm2api.XXXXXX")
cat "$base" > "$backup"
mv "$rendered" "$base"
echo "已合入内网映射，并保存权限为 0600 的原始配置备份。"
echo "尚未重建 app。请在请求排空后执行部署。"
