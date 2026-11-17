// Real Vue confirmation recovery with a synthetic provider and no real money.
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const phase = process.argv[2] || 'after'
if (!['before', 'after'].includes(phase)) throw new Error('Use before or after')
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/opcpay-automatic', phase)
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true })
const [original] = fixture.listeners('request')
fixture.removeAllListeners('request')
let pending, credits = 0, deductions = 0
const operations = new Map()
fixture.on('request', async (req, res) => {
  const path = new URL(req.url, 'http://fixture.invalid').pathname.replace(/^\/api\/v1/, '')
  if (!path.startsWith('/payment/opcpay')) return original(req, res)
  let raw = ''; for await (const chunk of req) raw += chunk
  const body = raw ? JSON.parse(raw) : {}
  requests.push({ path, method: req.method, kind: body.kind, action_id: body.action_id })
  const reply = (data, status = 200) => {
    res.writeHead(status, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ code: status === 200 ? 0 : status, data, message: 'OK' }))
  }
  if (path.endsWith('/__fixture')) {
    if (body.paid) {
      const operation = operations.get(body.paid)
      if (!operation) throw Error('Unknown simulated purchase')
      if (operation.status !== 'COMPLETED') { credits += 10; deductions++ }
      operations.set(body.paid, { ...operation, kind: 'consume', status: 'COMPLETED' })
      if (pending?.business_id === body.paid) pending = undefined
    }
    reply({ credits, deductions, operations: [...operations.values()] }); return
  }
  if (path === '/payment/opcpay') {
    reply({ configured: true, enabled: true, provider_name: 'Fixture', provider_id: 1,
      capabilities: { provider_configured: true, capabilities: { purchase_entry: true, consumption: true } },
      product: { capability_status: 'ENABLED', product_name: 'Fixture', billing_mode: 'QUANTITY',
        charging_options: [{ sku_id: 'ten-usd', display_name: '10 积分充值卡', price: '9.90', currency: 'CNY', quota_amount: 10 }] },
      access: { remaining: 0 }, credit_usd_per_unit: 1, pending }); return
  }
  if (req.method === 'GET' && path.startsWith('/payment/opcpay/operations/')) {
    reply(operations.get(path.split('/').pop())); return
  }
  if (path.endsWith('/close')) {
    const id = path.split('/').at(-2), operation = operations.get(id)
    if (operation?.status === 'READY') operations.set(id, { ...operation, status: 'CLOSED' })
    pending = undefined; reply({ closed: true }); return
  }
  if (body.kind === 'purchase') {
    pending = { business_id: 'purchase-' + body.action_id, action_id: body.action_id, sku_id: body.sku_id,
      kind: 'purchase', status: 'READY', amount: 10, credit_usd: 10,
      result: { status: 'READY', purchase_url: 'https://pay.example.test/direct' } }
    operations.set(pending.business_id, pending); reply(pending); return
  }
  if (body.kind === 'consume') throw Error('The UI must not submit a confirmation')
  reply(pending)
})
await new Promise(resolve => fixture.listen(0, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13044', '--strictPort'], {
  cwd: root, env: { ...process.env, VITE_DEV_PROXY_TARGET: `http://127.0.0.1:${fixture.address().port}` }, stdio: ['ignore', 'pipe', 'pipe']
})
let log = ''; vite.stdout.on('data', data => { log += data }); vite.stderr.on('data', data => { log += data })
try {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch('http://127.0.0.1:13044')).ok) break } catch { /* own fixture startup */ }
    if (vite.exitCode !== null || i === 99) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'opcpay-automatic.browser.mjs'), 'utf8'))
    .replace('"__OUTPUT__"', JSON.stringify(output)).replace('"__ROOT__"', JSON.stringify(resolve(root, '..')))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Automatic credit acceptance failed')
} finally {
  vite.kill('SIGTERM'); fixture.closeAllConnections(); fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
