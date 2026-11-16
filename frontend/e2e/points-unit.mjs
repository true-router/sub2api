/** Run: node e2e/points-unit.mjs. Actual frontend with deterministic local API fixtures. */
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createHash } from 'node:crypto'
const root = resolve(import.meta.dirname, '..')
const output = resolve(process.env.POINTS_E2E_OUTPUT || '/tmp/sub2api-points-e2e')
await mkdir(output, { recursive: true })
const requests = []
const base = createAdminFixtureServer({ requests, accountPreview: true, dashboardPreview: true, modelPlazaPreview: true, modelProvidersPreview: true })
const fixture = createServer((req, res) => {
  const path = new URL(req.url, 'http://localhost').pathname
  if (path === '/api/v1/payment/opcpay') {
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify({ code: 0, data: { enabled:true, configured:true, provider_name:'OPCPay',provider_id:1,capabilities:{provider_configured:true,capabilities:{purchase_entry:true,consumption:true}},product:{capability_status:'ENABLED',charging_options:[{sku_id:'points-fixture',display_name:'Points fixture',price:'10',currency:'CNY',quota_amount:10}]},credit_usd_per_unit:1 } }))
  } else base.emit('request', req, res)
})
await new Promise(resolve => fixture.listen(18109, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13109', '--strictPort'], { cwd: root, env: { ...process.env, VITE_ACCOUNT_PREVIEW: 'true', VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18109' }, stdio: ['ignore', 'pipe', 'pipe'] })
let log = ''
vite.stdout.on('data', chunk => { log += chunk }); vite.stderr.on('data', chunk => { log += chunk })
try {
  for (let i = 0; i < 120; i++) {
    try { if ((await fetch('http://127.0.0.1:13109')).ok) break } catch { /* Startup */ }
    if (vite.exitCode !== null || i === 119) throw Error(log)
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  const script = (await readFile(resolve(import.meta.dirname, 'points-unit.browser.mjs'), 'utf8')).replace('"__OUTPUT__"', JSON.stringify(output))
  const browser = spawn('ego-browser', ['nodejs', '-e', script], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw Error('Points display acceptance failed')
} finally {
  vite.kill('SIGTERM'); fixture.closeAllConnections(); fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
  const files = ['e2e/points-unit.mjs','e2e/points-unit.browser.mjs','src/utils/points.ts','src/views/user/UnifiedModelsView.vue','src/components/payment/currency.ts']
  const source = {}
  for (const file of files) {
    try { source[file] = createHash('sha256').update(await readFile(resolve(root,file))).digest('hex') } catch { source[file] = null }
  }
  await writeFile(resolve(output, 'source-sha256.json'), JSON.stringify(source, null, 2))
}
