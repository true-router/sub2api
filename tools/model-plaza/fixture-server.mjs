// Deterministic API fixture server for the real production frontend route.
import http from 'node:http'
import { readFile } from 'node:fs/promises'
import { resolve, extname } from 'node:path'
const root = resolve(process.argv[2] || 'backend/internal/web/dist')
const port = Number(process.env.PLAZA_E2E_PORT || 4317)
let scenario = process.env.PLAZA_FIXTURE_SCENARIO || 'standard'
const pricing = { billing_mode: 'token', input_price: .000008, output_price: .00004, cache_write_price: .00001, cache_write_1h_price: .000016, cache_read_price: .0000002, image_input_price: null, image_output_price: null, per_request_price: null, intervals: [] }
function fixture() {
  const model = { name: 'claude-fable-5-2', platform: 'anthropic', pricing, official_pricing: { ...pricing }, long_context_basis: 'whole_request' }
  const group = { id: 1, name: 'Claude 标准', platform: 'anthropic', description: 'Standard balance pricing', subscription_type: 'standard', rate_multiplier: .8, peak_rate_enabled: true, peak_start: '18:00', peak_end: '22:00', peak_rate_multiplier: 1.2, is_exclusive: false, image_rate_independent: false, image_rate_multiplier: 1, video_rate_independent: false, video_rate_multiplier: 1, long_context_pricing_enabled: true, models: [model] }
  const data = { description: '', cny_per_credit: 1, exchange_rate: { usd_to_cny: 7, date: new Date().toISOString().slice(0, 10), source: 'Frankfurter', stale: false }, groups: [group] }
  if (scenario === 'demo') {
    data.description = '演示数据：模型、分组、价格与汇率均为固定示例。不会产生扣费。'
    group.description = '标准余额分组，按 ¥1 购买 1 额度，模型价格使用 0.8 倍率。'
    const sonnetPricing = { ...pricing, input_price: .000003, output_price: .000015, cache_write_price: .00000375, cache_write_1h_price: .000006, cache_read_price: .0000003 }
    const gptPricing = { ...pricing, input_price: .000005, output_price: .00003, cache_write_price: null, cache_write_1h_price: null, cache_read_price: .0000005 }
    group.models.push({ name: 'claude-sonnet-demo', platform: 'anthropic', pricing: sonnetPricing, official_pricing: { ...sonnetPricing } })
    data.groups.push({ ...group, id: 2, name: 'Claude 优惠', description: '优惠余额分组，使用 0.6 倍率。可与标准分组直接比较。', rate_multiplier: .6, peak_rate_enabled: false, models: [{ ...model }] })
    data.groups.push({ ...group, id: 3, name: 'OpenAI 标准', platform: 'openai', description: 'OpenAI 演示报价。', rate_multiplier: .7, peak_rate_enabled: false, models: [{ name: 'gpt-demo', platform: 'openai', pricing: gptPricing, official_pricing: { ...gptPricing } }] })
  }
  if (scenario === 'missing-rate') data.exchange_rate = null
  if (scenario === 'stale-rate') data.exchange_rate.stale = true
  if (scenario === 'user-rate') group.user_rate_multiplier = .5
  if (scenario === 'no-discount') group.rate_multiplier = 8
  if (scenario === 'missing-official') model.official_pricing = null
  if (scenario === 'subscription') group.subscription_type = 'subscription'
  if (scenario === 'tiers') {
    model.pricing = { ...pricing, intervals: [{ ...pricing, min_tokens: 0, max_tokens: 200000, tier_label: '≤200K' }, { ...pricing, input_price: .000016, min_tokens: 200000, max_tokens: null, tier_label: '>200K' }] }
    model.time_pricing = { timezone: 'Asia/Shanghai', weekdays_only: true, periods: [{ start_time: '00:00:00', end_time: '08:00:00', multiplier: .5 }] }
  }
  if (scenario === 'different-output') model.pricing = { ...pricing, output_price: .00006 }
  if (scenario === 'tiny') model.pricing = { ...pricing, input_price: 1e-12 }
  if (scenario === 'empty') data.groups = []
  return data
}
const types = { '.js': 'application/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.png': 'image/png', '.woff2': 'font/woff2', '.html': 'text/html' }
http.createServer(async (req, res) => {
  const url = new URL(req.url, `http://127.0.0.1:${port}`)
  res.setHeader('Cache-Control', 'no-store')
  const json = value => { res.setHeader('Content-Type', 'application/json'); res.end(JSON.stringify(value)) }
  if (url.pathname === '/__fixture') { scenario = url.searchParams.get('scenario') || 'standard'; return json({ scenario }) }
  if (url.pathname === '/api/v1/model-plaza') return json({ code: 0, data: fixture() })
  if (url.pathname === '/api/v1/settings/public') return json({ code: 0, data: { site_name: scenario === 'demo' ? '模型广场 · 本地演示' : 'Model Plaza E2E', model_plaza_enabled: true, model_plaza_require_auth: false, backend_mode_enabled: false, registration_enabled: true } })
  if (url.pathname.startsWith('/api/')) return json({ code: 0, data: [] })
  try {
    const file = (url.pathname.startsWith('/assets/') || url.pathname === '/logo.svg') ? resolve(root, `.${url.pathname}`) : resolve(root, 'index.html')
    if (!file.startsWith(root + '/')) { res.writeHead(403); return res.end() }
    let body = await readFile(file)
    if (scenario === 'demo' && extname(file) === '.html') body = Buffer.from(body.toString().replace('<head>', '<head><script>localStorage.setItem("sub2api_locale","zh")</script>'))
    res.setHeader('Content-Type', types[extname(file)] || 'application/octet-stream'); res.end(body)
  } catch { res.writeHead(404); res.end('Not found') }
}).listen(port, '127.0.0.1', () => console.log(`Model Plaza E2E: http://127.0.0.1:${port}/model-plaza`))
