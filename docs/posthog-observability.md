# sub2api 的 PostHog 全链路观测契约

日期：2026-11-12。状态：核心接入已实现。已完成隔离验收和 TrueRouter 项目回读；生产部署与完整指标、告警尚未完成。

## 目的与职责

本端负责 `客户 → sub2api → vm2api` 的入口追踪、路由与重试、上下文传播及入口最终结果。日志和追踪经宿主 Collector 进入同一个 PostHog 项目。

总决策位于 clearcc 根目录的 `POSTHOG_OBSERVABILITY_DECISION.md`。本文保存 sub2api 独立实施所需的本端契约。共享语义变更需同步总决策与下游仓库。

## 输入与输出

- 公网入口新建受控 trace。客户传入的请求 ID 保留现有诊断语义，不用于授权或控制采样。
- 内部可信来源只有经过确认后才能接续父上下文，普通客户 API 鉴权不代表内部信任。
- 向受控 vm2api 发送请求时，由 SDK 注入 `traceparent`，按需保留 `tracestate`。初期不传播 `baggage`。
- 每次实际出站与重试建立独立 client span。使用当前尝试的 span 注入上下文，不能重复使用上一次尝试的 span ID。
- 保留本地 `request_id`、客户端请求 ID 和下游响应的 `upstream_request_id`。它们不必与下游本地请求 ID 相等。
- 受控链路以同一个 `trace_id` 关联。直接发送到其他官方上游时，不自动附带内部追踪头。

## 当前能力与修改位置

本节基于 `5208257af72ed138ef239cf5165fda59a1fc2f50` 的源码核对。实施前重新确认函数与部署身份。

| 位置 | 已有能力 | 本次实施要求 |
|---|---|---|
| [request_logger.go](../backend/internal/server/middleware/request_logger.go) | 校验或生成 X-Request-ID，创建请求日志上下文 | 增加入口 span；向现有 zap 上下文加入 trace/span |
| [logger.go](../backend/internal/server/middleware/logger.go) | 请求完成访问日志 | 关联入口最终结果；区分 HTTP 状态和流完成 |
| [gateway_service.go](../backend/internal/service/gateway_service.go) | 出站头白名单 | 确认受控传播路径；不能只扩大白名单就宣称贯通 |
| [gateway_anthropic_passthrough.go](../backend/internal/service/gateway_anthropic_passthrough.go) | 构造 Anthropic 实际出站请求 | 在过滤、鉴权和覆写完成后注入当前尝试上下文 |
| [gateway_forward.go](../backend/internal/service/gateway_forward.go) | 转发与重试相关处理 | 核对每次尝试、协议转换与错误返回边界 |
| [日志包](../backend/internal/pkg/logger/logger.go) | zap 与结构化输出 | 保留日志库和已有日志用途，补齐关联字段 |
| [go.mod](../backend/go.mod)及服务启动入口 | 已有间接 OTel 依赖 | 核对实际初始化，再补齐必要 SDK、exporter 和关闭刷新 |

先核实线上实际使用的路由。先打通一条路由，再覆盖实际启用的 Messages、Chat Completions、Responses 路径。每条路径均检查重试后请求是否重建或覆盖追踪头。

## 日志与结果

资源使用 `service.namespace=truerouter`、`service.name=sub2api`，补充版本、实例和环境。日志与 span 的资源身份必须一致。

日志携带 `trace_id` 和 `span_id`。Collector 将它们解析为 OTLP 专用 TraceId、SpanId 字段。普通 JSON 字段存在不等于 PostHog 已完成关联。

请求阶段日志记录 UTC 时间、严重级别、事件、阶段、结果、trace/span、本地请求 ID。出站日志补充尝试标识、受控目标、不透明账号标识、响应请求 ID、错误类别和耗时。避免记录含密钥的完整 URL。

入口只产生一次最终结果。下游尝试与分类调用不计为新的客户入口请求。HTTP 200 后流出错仍记录失败或结果未知。只有观察到协议终止才记录 `stream_complete=true`。

首个字节、首个协议事件和首个内容输出分开。入口“首个输出耗时”从接收请求计到向客户发送首个有效内容；无法观测时缺省。不能用下游相对计时替代入口耗时。

保留用量数据库和本地运维日志。PostHog 上传使用字段白名单。禁止上传鉴权头、Cookie、token、提示词、完整输出和原始工具参数。原始上游错误也要过滤。

## 运行与兼容

应用只配置内部 Collector 地址。PostHog 项目 token 由 Collector 管理。导出失败、队列满和关闭刷新超时不能阻断业务或改变原有重试策略。

缺少或无效追踪头不影响请求处理。停用观测不改变鉴权、计费、路由和响应协议。新增观测不改变业务请求 ID、去重语义和下游取消协议。

## 端到端验收

实施前固定以下失败场景，再使用端到端测试验证：

1. 非流式和流式请求：入口与下游共享 trace，日志定位到正确 span。
2. 实际启用的每条协议路由：头过滤、转换和请求重建不丢失上下文。
3. 重试与下游失败：尝试分开，最终结果只记录一次。
4. 同请求 ID、同会话并发：按 trace 区分，日志不串请求。
5. 首个响应后错误与客户端取消：不误报流完成，保留原有业务行为。
6. 缺省、无效和伪造追踪头：公网新建根，采样与鉴权不受外部操控。
7. Collector 不可用：业务继续，采集故障可见，平台恢复后按容量回读数据。
8. 目标 PostHog 项目：可以通过入口请求 ID 找到 trace，并读到 vm2api 与 Hermit 关联日志。

每次验收保留命令、summary.json、源码和构建摘要、脱敏配置摘要、平台回读查询与结果、trace/log 样本和清理回执。仅本地日志或 HTTP 上传成功不能证明跨仓库验收通过。

核心接入和隔离平台验收结果见下节。生产部署、真实上游和完整协议路由验收仍未执行。

## 本次已实现的接入与验收边界

总验收回执位于 clearcc 的 `artifacts/posthog-observability-acceptance-20261112/README.md`。回执记录平台查询、构建摘要、清理结果和未验收范围。

应用向内部 Collector 发送标准 OTLP，保留现有请求与取消标识。PostHog 目标为 US 区 TrueRouter，ProjectID 730627。上传凭据只由宿主 Collector 管理。

请求阶段使用专用结构化事件，资源身份与 span 一致。Collector 只上传批准事件，过滤敏感字段，并按服务、实例和环境分组。宿主配置、模板、复现命令和总验收记录位于 clearcc 的 observability/README.md。

隔离链路使用真实三端中间件、传输与槽位入口。普通响应和流式响应经过真实 Hermit 原生运行时与固定 CLI，上游仅为本机模拟服务。sub2api 数据库、计费和账号路由不在该夹具范围内。故障、取消和并发使用受控原生运行时夹具，不能称为真实 CLI 故障验收。

已单独验证 Collector 的服务分组、trace/span 关联、字段过滤、重启读取位置、后端失败后的持久化队列恢复和日志轮转。目标 PostHog 项目已回读三端完整追踪与关联日志。生产放置、真实账号、真实上游、全量协议、指标总览和告警仍需部署验收。

本端配置 TRUEROUTER_TRACE_ORIGINS 和至少 32 字节的 TRUEROUTER_TRACE_TOKEN。出站仅在受控 origin 注入内部上下文。响应的固定 X-Observability-Outcome 元数据在本端被读取并移除，用于保留下游结果未知语义；不改变业务重放规则。
