// Real Vue confirmation recovery with a synthetic provider and no real money.
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const phase = process.argv[2] || 'after'
if (!['before', 'after'].includes(phase)) throw new Error('Use before or after')
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/opcpay-confirmation', phase)
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true })
const [original] = fixture.listeners('request')
fixture.removeAllListeners('request')
let pending, mode = 'unpaid', credits = 0, deductions = 0
const consumed = new Set(), delivered = new Set()
fixture.on('request', async (req, res) => {
  const path = new URL(req.url, 'http://fixture.invalid').pathname.replace(/^\/api\/v1/, '')
  if (!path.startsWith('/payment/opcpay')) return original(req, res)
  let raw = ''; for await (const chunk of req) raw += chunk
  const body = raw ? JSON.parse(raw) : {}
  requests.push({ path, kind: body.kind, action_id: body.action_id })
  const reply = (data, status = 200, message = 'OK', reason) => {
    res.writeHead(status, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ code: status === 200 ? 0 : status, data, message, reason }))
  }
  if (path.endsWith('/__fixture')) { if (body.mode) mode = body.mode; reply({ mode, pending, credits, deductions }); return }
  if (path === '/payment/opcpay') {
    reply({ configured: true, enabled: true, provider_name: 'Internal provider', provider_id: 1,
      capabilities: { provider_configured: true, capabilities: { purchase_entry: true, consumption: true } },
      product: { capability_status: 'ENABLED', product_name: 'Fixture', billing_mode: 'QUANTITY',
        charging_options: [{ sku_id: 'ten-usd', display_name: '10 积分充值卡', price: '9.90', currency: 'CNY', quota_amount: 10 }] },
      access: { remaining: 0 }, credit_usd_per_unit: 1, pending }); return
  }
  if (path.endsWith('/close')) { pending = undefined; reply({ closed: true }); return }
  if (body.kind === 'purchase') {
    pending = { business_id: 'purchase-' + body.action_id, action_id: body.action_id, sku_id: body.sku_id,
      kind: 'purchase', status: 'READY', amount: 10, credit_usd: 10,
      result: { status: 'READY', purchase_url: 'https://pay.example.test/direct' } }
    reply(pending); return
  }
  if (body.kind === 'consume') {
    if (mode === 'unpaid') { reply(null, 409, '验证失败，请重试', 'OPCPAY_PAYMENT_UNVERIFIED'); return }
    if (mode === 'unknown') { reply(null, 409, '请求未确认，请恢复原操作'); return }
    if (!consumed.has(body.action_id)) { consumed.add(body.action_id); deductions++ }
    pending = { business_id: 'consume-' + body.action_id, action_id: body.action_id, sku_id: body.sku_id,
      kind: 'consume', status: 'CONFIRMED', amount: 10, credit_usd: 10 }
    if (mode === 'confirmed') { reply(null, 409, '交付暂未确认，请重试'); return }
    if (!delivered.has(body.action_id)) { delivered.add(body.action_id); credits += 10 }
    const completed = { ...pending, status: 'COMPLETED' }; pending = undefined
    reply(completed); return
  }
  reply(pending)
})
await new Promise(resolve => fixture.listen(0, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13043', '--strictPort'], {
  cwd: root, env: { ...process.env, VITE_DEV_PROXY_TARGET: `http://127.0.0.1:${fixture.address().port}` }, stdio: ['ignore', 'pipe', 'pipe']
})
let log = ''; vite.stdout.on('data', data => { log += data }); vite.stderr.on('data', data => { log += data })
try {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch('http://127.0.0.1:13043')).ok) break } catch { /* own fixture startup */ }
    if (vite.exitCode !== null || i === 99) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'opcpay-confirmation.browser.mjs'), 'utf8'))
    .replace('"__OUTPUT__"', JSON.stringify(output)).replace('"__ROOT__"', JSON.stringify(resolve(root, '..')))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Checkout confirmation acceptance failed')
} finally {
  vite.kill('SIGTERM'); fixture.closeAllConnections(); fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
