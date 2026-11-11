// Decode the real Vue checkout QR using the browser's native barcode reader.
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const phase = process.argv[2] || 'after'
if (!['before', 'after'].includes(phase)) throw new Error('Use before or after')
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/opcpay-qr', phase)
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true })
const [original] = fixture.listeners('request')
fixture.removeAllListeners('request')
let pending
let purchaseURL = 'https://pay.example.test/direct?intent=card-10'
let purchaseQR = 'https://qr.example.test/rendered-payment.png'
fixture.on('request', async (req, res) => {
  const path = new URL(req.url, 'http://fixture.invalid').pathname.replace(/^\/api\/v1/, '')
  if (!path.startsWith('/payment/opcpay')) return original(req, res)
  let raw = ''; for await (const chunk of req) raw += chunk
  const body = raw ? JSON.parse(raw) : {}
  requests.push({ method: req.method, path, kind: body.kind })
  const reply = data => {
    res.writeHead(200, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ code: 0, message: 'OK', data }))
  }
  if (path.endsWith('/__fixture')) {
    purchaseURL = body.purchaseURL
    purchaseQR = body.purchaseQR
    reply({ ready: true }); return
  }
  if (path === '/payment/opcpay') {
    reply({ configured: true, enabled: true, provider_name: 'Synthetic provider', provider_id: 1,
      capabilities: { provider_configured: true, capabilities: { purchase_entry: true, consumption: true } },
      product: { capability_status: 'ENABLED', product_name: 'Fixture', billing_mode: 'QUANTITY',
        charging_options: [{ sku_id: 'trial', display_name: '体验包', price: '10.00', currency: 'CNY', quota_amount: 10 }] },
      credit_usd_per_unit: 1, access: { remaining: 0 }, pending })
    return
  }
  if (path === '/payment/opcpay/operations') {
    if (body.kind !== 'purchase') throw new Error('QR acceptance must not consume credit')
    pending ||= { business_id: 'fixture-' + body.action_id, action_id: body.action_id, sku_id: body.sku_id,
      kind: 'purchase', status: 'READY', amount: 10, credit_usd: 10,
      result: { status: 'READY', purchase_url: purchaseURL, purchase_qr_code: purchaseQR } }
    reply(pending); return
  }
  if (path.endsWith('/close')) { pending = undefined; reply({ closed: true }); return }
  reply(pending)
})
await new Promise(resolve => fixture.listen(0, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13042', '--strictPort'], {
  cwd: root, env: { ...process.env, VITE_DEV_PROXY_TARGET: `http://127.0.0.1:${fixture.address().port}` },
  stdio: ['ignore', 'pipe', 'pipe']
})
let log = ''
vite.stdout.on('data', data => { log += data })
vite.stderr.on('data', data => { log += data })
try {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch('http://127.0.0.1:13042')).ok) break } catch { /* own server startup */ }
    if (vite.exitCode !== null || i === 99) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'opcpay-qr.browser.mjs'), 'utf8'))
    .replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output))
    .replace('"__REPOSITORY_ROOT__"', JSON.stringify(resolve(root, '..')))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Direct payment QR acceptance failed')
} finally {
  vite.kill('SIGTERM')
  fixture.closeAllConnections(); fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
