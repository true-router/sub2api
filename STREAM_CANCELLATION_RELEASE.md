# 文本流式取消候选版本说明

## 行为变更

普通文本 HTTP 流式请求默认在客户端断开后立即取消上游。此前入口会为读取最终用量继续保留上游。该行为会延长原生执行，并使同会话的新请求等待。

新增启动配置：

```yaml
gateway:
  stream_data_interval_timeout: 180
  client_disconnect_drain_timeout: 0
```

环境变量为 `GATEWAY_CLIENT_DISCONNECT_DRAIN_TIMEOUT=0`。默认值是 `0`。接受 `0` 至 `300` 的整数。负数、超限值和小数会使启动失败。

- `0`：客户端断开后立即取消。
- `1` 至 `300`：客户端断开后最多继续读取指定秒数。后续数据不会延长时限。
- 客户端仍连接时，不启动断开时限。
- `stream_data_interval_timeout` 只控制相邻上游数据的间隔。其值为 `0` 时禁用间隔检查。

配置通过 YAML 或环境变量管理。修改 YAML 后重启进程。修改 Compose 的环境变量后重新创建对应服务容器，单纯重启旧容器不会载入新的环境变量。管理界面不提供此启动配置。

目标入口包括 Messages、Chat Completions 和 Responses 的普通文本 HTTP 流。图片、音频、WebSocket 和后台调用保留各自的生命周期。

## 会话与错误

vm2api 和 Hermit 继续使用现有的会话隔离与停止确认机制。逻辑队列释放不表示原生执行已经停止。确认停止后才允许复用会话。

工具交付结果未知时，网关保留结构化 `409`、错误 `code` 和 `error.details.retryable=false`。网关不会透传其余上游详情。调用方不得自动重放该操作。普通 `409` 不触发同账号自动重试。

vm2api 增加 `session_queue_wait_started`、`session_queue_wait_finished` 和 `session_queue_turn_released` 事件。等待耗时使用单调时钟。事件不包含原始会话键。

## 用量与迁移

启动时执行迁移 `260_usage_log_completeness.sql`。迁移为 `usage_logs` 增加可空布尔列 `usage_incomplete`。API 返回同名字段。

| 值 | 含义 |
| --- | --- |
| `true` | 缺少最终用量，或流未完整结束。只保留已经确认的计数。 |
| `false` | 已观察到协议要求的完整用量。零计数有效。 |
| `null` | 历史记录或未测量的路径。不能视为完整。 |

缺失字段与 JSON `null` 均不构成完整用量。累计输出计数取已确认值，不逐事件相加。不推算未观测到的用量。

取消生成与计费写入使用独立生命周期。沿用现有请求去重。下列查询用于区分完整性状态：

```sql
SELECT usage_incomplete, COUNT(*)
FROM usage_logs
GROUP BY usage_incomplete;
```

迁移不回填旧记录。旧版本使用显式列插入，新增可空列允许其继续写入。回滚应用时保留该列。不得在新版本仍运行时删除该列。

## 本地验收与复现

在 `backend` 目录执行：

```sh
go build -o /tmp/sub2api-stream-cancel-e2e ./e2e/stream-cancellation
python3 e2e/stream-cancellation/check.py --binary /tmp/sub2api-stream-cancel-e2e --out /tmp/stream-cancel-anthropic
python3 e2e/stream-cancellation/check.py --binary /tmp/sub2api-stream-cancel-e2e --provider openai --out /tmp/stream-cancel-openai
```

输出目录必须尚未存在。每次运行保存检查结果、转发结果和日志。上游只使用本机合成数据。

真实 PostgreSQL 验收使用独立数据库 `stream_cancellation_e2e`。不得指向业务数据库。设置 `E2E_DATABASE_URL` 后执行：

```sh
/tmp/sub2api-stream-cancel-e2e --db-check
```

数据库验收执行正式迁移、正式用量仓储和正式结算仓储。它验证单条、批量、异步写入、重复结算及旧列插入兼容。该验收需要全新数据库。

跨仓库原生验收驱动和固定工件位于工作区 `.planning/2026-11-17-2026-11-17-online-ttft/implementation/`。执行记录包含具体命令、源码身份和失败工件。它使用真实 Hermit、固定官方 CLI、真实 vm2api 和隔离 OpenResty。上游及账号均为合成数据。该验收不等同于生产鉴权、生产调度或真实订阅计费验收。

## 发布与回滚门槛

本次仅准备本地候选版本。尚未推送、发布或部署。未终止生产会话。

本地故障注入已覆盖取消接口不可达、`202`、超时与连接中断。停止未知保护、独立会话隔离、竞争与共享旧策略均有工件。生产发布前仍须执行目标平台的候选工件验收和发布门禁，并取得生产操作授权。未运行的场景不得标为通过。

已知的基线缺口：OpenAI Responses 透传路径没有数据间隔计时器。本次保留该行为。新增客户端断开时限独立生效。不要将该路径描述为已经受数据间隔配置保护。

回滚顺序：

1. 记录候选提交、镜像摘要和实际配置。
2. 若仅需恢复继续读取行为，将新参数显式设为正整数，并重新载入启动配置。该模式会恢复断开后的资源消耗与会话等待。
3. 若需回滚应用，恢复已记录的旧镜像，保留新增数据库列。
4. 检查正常完成、取消、同会话重发和用量写入。
5. 保留结果未知会话的隔离。不得用配置回滚解除隔离或重放工具。
