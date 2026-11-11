// Isolated browser regression: node frontend/e2e/opcpay-close.mjs.
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { readFile, mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../.planning/2026-11-11-balance-sources/artifacts/opcpay-close')
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true })
const [original] = fixture.listeners('request')
fixture.removeAllListeners('request')
let pending
let refreshDelay = false
let failClose = false
let closeCalls = 0
fixture.on('request', async (req, res) => {
  const path = new URL(req.url, 'http://fixture.invalid').pathname.replace(/^\/api\/v1/, '')
  if (!path.startsWith('/payment/opcpay')) return original(req, res)
  let raw = ''; for await (const chunk of req) raw += chunk
  const body = raw ? JSON.parse(raw) : {}
  requests.push({ method: req.method, path })
  const reply = (data, status = 200) => { res.writeHead(status, { 'content-type': 'application/json' }); res.end(JSON.stringify({ code: status === 200 ? 0 : status, message: status === 200 ? 'OK' : '模拟关闭失败，请重试', data })) }
  if (path.endsWith('/__fixture')) { failClose = body.failClose === true; reply({ closeCalls }); return }
  if (path === '/payment/opcpay') {
    if (refreshDelay) { refreshDelay = false; await new Promise(resolve => setTimeout(resolve, 900)) }
    reply({ configured: true, enabled: true, provider_name: 'Local fixture', provider_id: 1, capabilities: { provider_configured: true, capabilities: { purchase_entry: true, consumption: true } }, product: { capability_status: 'ENABLED', product_name: 'Fixture', billing_mode: 'QUANTITY', charging_options: [{ sku_id: 'trial', display_name: '体验包', price: '10.00', currency: 'CNY', quota_amount: 10 }] }, credit_usd_per_unit: 1, access: { remaining: 10 }, pending }); return
  }
  if (path === '/payment/opcpay/operations') {
    if (body.kind !== 'purchase') throw new Error('Regression must not consume payment credit')
    pending = { business_id: 'fixture-' + body.action_id, action_id: body.action_id, sku_id: 'trial', kind: 'purchase', status: 'READY', amount: 10, credit_usd: 10, result: { status: 'READY', purchase_url: 'https://example.test/synthetic-purchase' } }
    reply(pending); return
  }
  if (path.endsWith('/close')) {
    closeCalls++
    if (failClose) { failClose = false; reply(null, 503); return }
    pending = undefined; refreshDelay = true; reply({ closed: true }); return
  }
  reply(pending)
})
await new Promise(resolve => fixture.listen(0, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13041', '--strictPort'], { cwd: root, env: { ...process.env, VITE_DEV_PROXY_TARGET: `http://127.0.0.1:${fixture.address().port}` }, stdio: ['ignore', 'pipe', 'pipe'] })
let log = ''; vite.stdout.on('data', data => { log += data }); vite.stderr.on('data', data => { log += data })
try {
  for (let i = 0; i < 100; i++) {
    try { if ((await fetch('http://127.0.0.1:13041')).ok) break } catch { /* wait for own fixture */ }
    if (vite.exitCode !== null || i === 99) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 100))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'opcpay-close.browser.mjs'), 'utf8')).replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output)).replace('"__REPOSITORY_ROOT__"', JSON.stringify(resolve(root, '..') + '/'))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Payment close regression failed')
} finally {
  vite.kill('SIGTERM'); fixture.closeAllConnections(); fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
