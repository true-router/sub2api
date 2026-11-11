/** Isolated dashboard E2E: node e2e/user-dashboard.mjs. Artifacts: output/user-dashboard. */
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { readFile, mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/user-dashboard')
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true, dashboardPreview: true })
await new Promise(resolve => fixture.listen(18085, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13035', '--strictPort'], { cwd: root, env: { ...process.env, VITE_ACCOUNT_PREVIEW: 'true', VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18085' }, stdio: ['ignore', 'pipe', 'pipe'] })
let log = ''
vite.stdout.on('data', chunk => { log += chunk })
vite.stderr.on('data', chunk => { log += chunk })
try {
  for (let i = 0; i < 120; i++) {
    try { if ((await fetch('http://127.0.0.1:13035')).ok) break } catch { /* server startup */ }
    if (vite.exitCode !== null || i === 119) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'user-dashboard.browser.mjs'), 'utf8')).replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Dashboard browser acceptance failed')
  const trend = requests.filter(r => r.path === '/usage/dashboard/trend')
  const recent = requests.filter(r => r.path === '/usage')
  const hourly = trend.some(r => new URLSearchParams(r.search).get('granularity') === 'hour')
  const updatedRange = recent.some(r => { const p = new URLSearchParams(r.search); return p.get('start_date') === p.get('end_date') })
  if (!hourly || !updatedRange) throw new Error('Dashboard filters did not reach the API')
  console.log('PASS Dashboard range and granularity reach the API')
} finally {
  vite.kill('SIGTERM')
  fixture.closeAllConnections()
  fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
