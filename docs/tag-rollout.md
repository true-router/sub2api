# tag 发布接入

状态：代码与隔离验收阶段。生产未启用。

应用从环境变量 `TRUEROUTER_ROLLOUT_TOKEN_FILE` 读取私有令牌。使用 `TRUEROUTER_INSTANCE_ID` 标识实例。不设置令牌文件时保持原运行模式。

`POST /_rollout/drain` 停止业务准入。`POST /_rollout/resume` 恢复准入。`GET /_rollout/status` 返回证明。所有控制接口要求 Bearer 令牌。发布器核对实例身份，不以健康检查替代排空。

Gin 中间件统计请求，流式响应保留到结束。升级连接保留到 Close。状态还读取真实 UsageRecordWorkerPool。只有 HTTP 为零且 `pending_usage_records` 为零，状态才允许退役。排空期间新业务请求被拒绝，控制接口仍可访问。

部署使用既有 PostgreSQL 与 Redis。两个应用目录独立。共享数据库的 SQL 迁移集合必须由运维审核。服务器固定 `approved_migrations_sha256`，摘要变化时自动拒绝。不要通过回滚镜像回滚数据库。

验收入口：`backend/e2e/rollout/main.go` 与 `check.mjs` 验证真实 Gin、流式请求和异步计费。`backend/tools/production-billing-e2e` 支持 `--binary`，对固定发布二进制执行真实 PostgreSQL/Redis 计费验收。

## 配置与边界

本仓库新增 `.github/workflows/production-rollout.yml` 与资格、通知脚本。`production` environment 配置 `ROLLOUT_ENDPOINT` 与 `ROLLOUT_NOTIFICATION_SECRET`。`ROLLOUT_ENABLED=true` 才启用。未准备备用账号与槽位时保持关闭。

CI 发送通知后查询服务器任务状态。只有 completed 才标记 DEPLOYED。通知成功仅表示入队。失败、排空等待或查询超时不能作为发布完成。

完整宿主机配置、初次接入、生产锁和恢复步骤见同一项目 vm2api 仓库的 `docs/operations/tag-rollout.md`。本轮未执行托管 CI 或生产部署。
