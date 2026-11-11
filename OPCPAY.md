# 一人付充值卡

## 配置入口

1. 打开「系统设置 → 支付设置」。
2. 打开「启用支付」。
3. 在「启用的服务商」中选择「一人付 · 充值卡」。
4. 在「服务商管理」中点击「添加服务商」。
5. 选择「一人付 · 充值卡」，填写名称和 API Key，并启用实例。
6. 保存配置。

配置保存在现有服务商表中。不读取一人付环境变量。保存后立即生效，无需重启应用。

API Key 使用现有支付加密主密钥执行 AES-256-GCM 加密。主密钥不可用时拒绝保存，不降级为明文。管理接口不返回原密钥或密文。编辑时留空保留原密钥。审计日志隐藏密钥字段。

回跳网站为可选项。可以填写 `https://example.com` 形式的公开网站来源。服务端固定追加 `/purchase`。留空时不指定回跳地址，用户购买后手动返回本站。

只允许同时启用一个一人付项目。有未完成操作时，系统阻止更换密钥、停用或删除原实例。用户可以结束尚未核销的购买入口。已经核销的操作必须继续恢复交付。

## 用户充值

1. 登录后进入「充值/订阅」页面 `/purchase`。
2. 系统从一人付读取可用充值卡。
3. 用户选择充值卡。
4. 用户点击「支付宝购买」，打开购买入口或扫码购买。
5. 用户完成购买后点击「我已购买，确认充值」。
6. 服务端查询积分权益，核销所选充值卡对应的积分，再增加余额。

兑换规则固定为 **1 积分 = 1 USD 的 sub2api 余额**。例如，充值卡包含 20 积分，确认充值时核销 20 积分并增加 20 USD 余额。人民币售价由一人付套餐决定，不能在前端修改。

只展示上游实际启用的 `QUANTITY/POINT` 套餐。充值卡名称、售价、币种和积分数量来自商品接口。买断、时长和次数套餐不作为余额充值卡展示。能力未开放或没有可用充值卡时显示说明，不提供无效购买按钮。

确认充值也可以使用账号已有的积分。系统不会根据余额增加反推某笔购买已付款。购买入口、二维码和页面返回均不是付款成功凭证。

## 服务端实现

保持 Go、Vue 3、Tailwind 3 和现有 BoardUI Vue 组件。复用现有服务商列表、管理员认证、合规确认、审计和敏感字段编辑规则。

一人付通过充值卡流程工作，不进入自定义金额支付流程。退款、账单和自动续费未接入。充值回执保存在 `opcpay_operations`，不伪装为现有支付订单列表中的已付款订单。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/api/v1/admin/payment/providers` | 读取服务商及密钥配置状态 |
| POST | `/api/v1/admin/payment/providers` | 添加服务商 |
| PUT | `/api/v1/admin/payment/providers/:id` | 编辑和启停服务商 |
| GET | `/api/v1/payment/opcpay` | 读取实际能力、充值卡、当前用户积分及未完成操作 |
| POST | `/api/v1/payment/opcpay/operations` | 保存购买或核销意图，再执行操作 |
| POST | `/api/v1/payment/opcpay/operations/:business_id/resume` | 恢复原操作 |
| POST | `/api/v1/payment/opcpay/operations/:business_id/close` | 结束当前用户尚未核销的购买入口 |

用户编号来自服务端登录态。客户端只提交操作类型、操作编号和 SKU。服务器保存所选充值卡的积分数量、交付余额、原服务商及回跳地址。客户端不能指定用户、价格、扣减量或交付量。

核销请求复用购买操作的 `action_id` 和 `sku_id`。服务端从已保存的购买意图取得原数量，避免后续套餐变化改变原操作。现有积分也可通过固定 1 积分的核销操作兑换，不接受客户端自定义扣减量。

核销确认必须满足：`SUCCEEDED`、`confirmed=true`，且业务编号、数量和 `POINT` 单位全部一致。`UNKNOWN` 和 `PROCESSING` 不交付余额。超时后查询原核销，或以原编号和原参数恢复核销。

确认扣减后持久化 `CONFIRMED`。本地交付失败时只恢复本地交付。余额增加与唯一完成回执在同一事务提交。并发重试不会重复增加余额。完成后使现有余额与认证缓存失效。

操作绑定原服务商，并保存凭证指纹。配置发生并发变化时，恢复流程拒绝使用另一项目的凭证。指纹不返回前端。旧版未绑定服务商的未完成操作不会自动改绑，需要管理员检查。

## 构建与迁移

部署应使用本次源码构建的镜像。上游镜像不包含本次改动。

```sh
docker build -t sub2api-opcpay:local .
```

在现有部署中使用该镜像，并保留数据卷。应用启动时按现有机制执行 `251_opcpay_operations.sql` 和 `252_opcpay_provider_binding.sql`。不需要配置 `opcpay_api_key` 或 `OPCPAY_PUBLIC_ORIGIN` 环境变量。

`tools/opcpay` 锁定 npm 包 `opcpay@0.3.4`，用于保留 SDK 和文档参考。已读取包内 README、`skills/opcpay/SKILL.md` 和集成参考。Go 后端按相同官方 HTTP 契约接入，没有增加 Node.js 常驻服务。

参考：[npm 包](https://www.npmjs.com/package/opcpay)、[Node 示例](https://github.com/Yapie0/opcpay/tree/main/examples/node)、[接口文档](https://ai6666.com/opcpay/docs/)、[OpenAPI](https://ai6666.com/opcpay/api/v1/pay/developer/openapi.json)。

## 可重复验证

在项目根目录执行：

```sh
backend/tools/opcpay-e2e/run.sh
```

脚本启动临时 PostgreSQL 18，运行实际服务商处理器、审计中间件和用户充值处理器。上游为本地模拟接口。结束时自动删除测试容器及数据卷。

工件保存在 `output/opcpay-e2e/checks.json`，包含检查结果与源码哈希。

浏览器验收使用独立模拟环境。在一个终端执行：

```sh
OPCPAY_E2E_PREVIEW=1 backend/tools/opcpay-e2e/run.sh
```

保持该终端运行。在另一个终端执行：

```sh
cd frontend
node e2e/opcpay-preview.mjs
```

模拟页面为 `http://127.0.0.1:13034`。在 frontend 目录运行浏览器脚本：

```sh
node e2e/opcpay.mjs
```

浏览器脚本使用 `ego-browser`。工件保存在 `output/opcpay-browser/`。这些检查使用合成登录态和模拟上游，不证明真实商户收款或真实核销。真实付款、核销和生产部署均未执行。
