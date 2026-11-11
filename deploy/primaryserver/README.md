# primaryserver 内网调用配置

本目录适用于已有 `truerouter` Compose 项目。应用服务名必须为 `app`。
配置让 app 把 `vm2api.truerouter.cc` 解析到 Docker 的 `host-gateway`。
账号上游继续使用 `https://vm2api.truerouter.cc`。

调用路径：app → 宿主机内网 443 → OpenResty → `127.0.0.1:8787` → vm2api。
请求保留 HTTPS 主机名与证书校验。
本方案依赖宿主机 443 端口允许 app 访问，并保留该域名的 OpenResty 反向代理。

## 准备配置

警告：已有部署文件可能包含密钥。保留私有备份，不要提交或公开备份文件。

1. 把本目录的脚本和配置片段一起复制到部署主机。
2. 对已有项目运行准备脚本：

   ```sh
   sh deploy/primaryserver/prepare-internal-route.sh \
     /opt/1panel/docker/compose/truerouter
   ```

3. 脚本通过 Docker Compose 合并配置并校验结果。
   脚本在同一目录保存 `.docker-compose.before-vm2api.*` 原始备份。
   备份和生成配置的权限均为 0600。
   脚本不修改 `.env`，不启动或重建容器。

脚本只依赖 POSIX shell、Docker Compose 和系统文件命令。
Compose 必须支持 `--no-env-resolution` 和 `--no-path-resolution`。
已核实的 primaryserver Compose v5.6.1 支持这些参数。

生成配置会规范化 YAML 并移除注释。变量插值表达式、相对路径和 `env_file` 引用保留。
已有其他主机映射会保留。重复执行不会重复备份或改变配置。
服务名不匹配或配置校验失败时，脚本不替换原文件。

## 稍后部署

警告：重建 app 会中断该容器中的在途请求。先排空请求，再执行以下操作。

1. 在已有部署目录运行：

   ```sh
   cd /opt/1panel/docker/compose/truerouter
   docker compose config --quiet
   docker compose up -d --no-deps app
   ```

2. 核对 app 中的域名映射：

   ```sh
   docker compose exec -T app getent hosts vm2api.truerouter.cc
   docker compose exec -T app wget -q -T 8 -O - \
     https://vm2api.truerouter.cc/health
   ```

3. 确认解析结果为宿主机内网地址，并确认健康检查成功。
4. 通过 TrueRouter 执行已授权的模型请求，核对鉴权和流式响应。
5. 核对 PostgreSQL、Redis、vm2api 和动态内核容器的身份没有变化。

## 后续更新与回滚

内网映射已写入主 `docker-compose.yml`。普通 `docker compose up`、重建 app 和仅替换镜像的发布会继续应用它。
如果部署流程重新生成或覆盖主 Compose 文件，必须在重建 app 前再次运行准备脚本。
不要使用通用 `docker-deploy.sh` 覆盖已有 primaryserver 配置。该脚本用于通用安装，并下载上游模板。

Docker 在创建容器时解析 `host-gateway`。宿主机网关地址变化后，需要重建 app 来刷新映射。
普通重启不会刷新映射。

回滚时，恢复本次准备脚本保存的主 Compose 备份，再按相同的请求排空流程重建 app。
不恢复数据库或凭据。

## 本地端到端验证

测试使用本机 Docker 和 `alpine:latest`，创建独立 Compose 项目。
测试验证配置合并、幂等性、原始备份、变量与数据保留、两次 app 重建和依赖容器身份保留。
测试结束后删除隔离容器和网络，并生成 JSON 收据。

在仓库根目录运行：

```sh
python3 deploy/tests/primaryserver-internal-route.e2e.py \
  --receipt /tmp/primaryserver-internal-route-e2e.json
```

该测试不代表生产 HTTPS 或真实模型请求已经验收。
