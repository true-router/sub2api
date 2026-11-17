# OpenCode Go OAuth 接入

状态：本地实现与验收完成。发布或部署尚未执行。最后核验：2026-11-17。

## 目标

OpenCode 新版 CLI 将 Console OAuth 凭据保存在本地 SQLite 数据库。该凭据不是 API key。sub2api 原有 OpenCode 添加入口只接收 API key，不能导入该登录状态。

本次增加设备授权、原生凭据导入、令牌刷新和 Go 推理转发。保留旧 API key 接入路径。

## 官方依据

研究基于 anomalyco/opencode 提交 `1cb0633c8b2ed95ec2af645a4c541c5e219e98ed`。

- [设备授权与刷新实现](https://github.com/anomalyco/opencode/blob/1cb0633c8b2ed95ec2af645a4c541c5e219e98ed/packages/core/src/plugin/provider/opencode.ts)。
- [Console 认证头实现](https://github.com/anomalyco/opencode/blob/1cb0633c8b2ed95ec2af645a4c541c5e219e98ed/packages/core/src/console-access.ts)。
- [第三方客户端与 Go 端点说明](https://github.com/anomalyco/opencode/blob/1cb0633c8b2ed95ec2af645a4c541c5e219e98ed/services/site/src/docs/content/console/harnesses/index.mdx)。
- [推理协议说明](https://github.com/anomalyco/opencode/blob/1cb0633c8b2ed95ec2af645a4c541c5e219e98ed/services/site/src/docs/content/console/api/inference.mdx)。

官方当前也支持创建 personal API key。本机没有保存 API key，不代表控制台不能创建 API key。

## 使用步骤

1. 在管理员账号页面添加 OpenCode 账号。
2. 将授权方式切换为 `OAuth · OpenCode Go`。
3. 选择需要使用的代理。
4. 点击开始授权。
5. 打开 OpenCode 授权页面。
6. 在官方页面登录并确认 Go 工作区。
7. 等待页面显示凭据已就绪。
8. 保存账号。

设备授权使用独立的登录会话。优先使用此方式。

如果已有原生 OAuth 凭据对象，也可选择导入。对象必须包含 `type=oauth`、`access`、`refresh`、毫秒单位的 `expires` 和 `metadata.orgID`。服务端只接受官方 Console 来源。页面不会读取服务端任意文件。

警告：导入会共享原客户端的刷新令牌。后续令牌旋转可能导致原客户端需要重新登录。

账号菜单支持手动刷新和重新授权。重新授权沿用既有原子凭据更新入口，并保留账号配置。

## 实现边界

| 项目 | 行为 |
|---|---|
| 设备授权 | JSON POST `/console/auth/device/code`，使用官方公共客户端 `opencode-cli`，声明组织作用域 |
| 设备轮询 | JSON POST `/console/auth/device/token`，处理等待、减速、拒绝和到期 |
| 刷新 | 相同 token 端点，使用 `grant_type=refresh_token` |
| 组织选择 | 采用授权响应的 `org_id`，并通过官方组织列表核验；无明确组织且存在多个组织时拒绝自动选择 |
| OpenAI 协议 | `https://opencode.ai/inference/go/openai/v1` |
| Anthropic 协议 | `https://opencode.ai/inference/go/anthropic`，追加 `/v1/messages` |
| 身份 | `Authorization: Bearer` 和账号的 `x-org-id` |
| 会话 | 复用既有稳定 `x-opencode-session` 生成逻辑 |
| 刷新互斥 | 复用 `OAuthRefreshAPI` 的进程锁、Redis 锁、数据库重读和持久化 |
| 并发请求 | 使用请求内的账号快照，避免修改调度器共享账号 |
| 组织变化 | 刷新响应可更新组织；后续请求使用更新后的组织 |
| API key 兼容 | 保持原有 `/zen/go` 与 `/zen` 地址 |
| 敏感信息 | 账号响应继续脱敏；导入和轮询请求体不写入审计日志 |

OAuth 账号固定为 Go 模式。自定义 API key 地址不能覆盖 OAuth 的官方推理地址。上游推理请求的认证头和组织头由账号凭据决定。

新版 Console 支持把 Gemini 文本模型转换到标准推理协议。OAuth 账号允许该路径。旧 API key 账号继续使用原有模型限制。`jev-*` 仍需要专有端点。

旧 `/zen/go/v1/usage` 查询只适用于 API key。OAuth 账号不会调用该接口，也不会把 OAuth 凭据发往 Claude 用量接口。本次不增加新版 Console 用量展示。

## 管理接口

| 方法 | 路径 | 输入 |
|---|---|---|
| POST | `/api/v1/admin/opencode/oauth/device` | 可选 `proxy_id` |
| POST | `/api/v1/admin/opencode/oauth/poll` | `device_code`、可选 `proxy_id` |
| POST | `/api/v1/admin/opencode/oauth/import` | 单个原生对象 `credential` |

这些接口使用既有管理员认证、审计和管理员确认约束。

保存账号时使用 `platform=opencode_go`、`type=oauth`。凭据包含 `access_token`、`refresh_token`、秒单位的 `expires_at`、`org_id`、`account_mode=go` 和 `api_protocol=adaptive`。可选字段包括 `org_name`、`account_id` 和 `email`。

## 验收方法

依赖：Docker、Go 1.27.4、Python 3、OpenSSL。端到端脚本使用生产二进制、真实 PostgreSQL 和 Redis，以及持有临时证书的 HTTPS 官方协议模拟端点。脚本不会修改系统证书信任，也不会访问生产数据库。

```sh
python3 tools/opencode-oauth/e2e.py --output /tmp/opencode-oauth-receipt
```

`receipt.json` 记录检查结果和出站路径。脚本在退出时正常停止本次后端和容器，并删除本次临时数据。不会清理共享镜像。

需要浏览器验收时增加 `--hold`。脚本会输出临时 `ready.json` 路径。该文件只包含隔离环境凭据。浏览器结束后，在该文件所在目录创建 `release` 文件，脚本将执行清理。

实际验收结果和源码身份见任务验收回执。分别记录构建、模拟上游端到端验收、真实上游请求和浏览器验收。模拟授权不能证明真实用户已完成设备授权。

## 2026-11-17 验收结果

- 后端生产二进制构建通过。
- 前端类型检查、变更文件 ESLint、既有语言键完整性检查和生产构建通过。
- 独立 PostgreSQL、Redis、HTTPS 模拟上游的 31 项端到端检查通过。
- 真实账号页面的 9 项浏览器检查通过。
- 本机 OAuth 凭据经本次 sub2api 网关完成真实 Go 推理，HTTP 200，存在助手正文，`billing_source=go`，`cost=0`。
- 真实验收未旋转本机刷新令牌。仅在独立测试数据库临时保存凭据。验收后删除临时账号和密钥，并销毁测试数据库。
- 真实用户在官方页面完成全新设备授权：未执行。设备流程以官方源码和模拟上游验收为依据。
- 后台定时周期触发：未单独执行。已注册到既有后台刷新服务；请求触发刷新、手动刷新及并发互斥已验收。
- 新版 Console OAuth 用量展示：未实现。旧用量组件不再对 OAuth 账号发起查询。

源码基线：`aa0190d5dbf31570f80fa09ed4f65442a4b2939a`。实现分支：`feat/opencode-go-oauth`。验收时的变更文件 SHA-256 由端到端回执记录。

浏览器脚本位于 `tools/opencode-oauth/browser_e2e.py`。使用 `ego-browser`，并读取仅属于隔离环境的运行信息文件：

```sh
python3 tools/opencode-oauth/browser_e2e.py --runtime /absolute/path/browser-runtime.json --output /tmp/opencode-oauth-browser.json
```

`browser-runtime.json` 的 `origin` 指向本次前端服务，`ready_file` 指向 `e2e.py --hold` 生成的文件。前端服务的 `VITE_DEV_PROXY_TARGET` 必须指向该独立后端。结束浏览器任务后，使用 ego-browser 的 `task.finish({keep:[]})` 关闭脚本回执中记录的任务空间。
