/** Isolated key-usage E2E: node e2e/key-usage.mjs. */
import { serveKeyUsageFixture } from './fixtures/key-usage.mjs'
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { readFile, mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/key-usage-boardui')
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true })
const [originalHandler] = fixture.listeners('request')
fixture.removeAllListeners('request')
fixture.on('request', async (req, res) => {
  if (!await serveKeyUsageFixture(req, res, requests)) await originalHandler(req, res)
})
await new Promise(resolve => fixture.listen(18088, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13038', '--strictPort'], { cwd: root, env: { ...process.env, VITE_ACCOUNT_PREVIEW: 'true', VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18088' }, stdio: ['ignore', 'pipe', 'pipe'] })
let log = ''
vite.stdout.on('data', chunk => { log += chunk })
vite.stderr.on('data', chunk => { log += chunk })
try {
  for (let i = 0; i < 120; i++) {
    try { if ((await fetch('http://127.0.0.1:13038')).ok) break } catch { /* wait for startup */ }
    if (vite.exitCode !== null || i === 119) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'key-usage.browser.mjs'), 'utf8')).replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Key usage browser acceptance failed')
  const usage = requests.filter(item => item.path === '/v1/usage')
  if (!usage.some(item => { const p = new URLSearchParams(item.search); return p.get('start_date') === '2026-11-01' && p.get('end_date') === '2026-11-05' && !!p.get('timezone') }) || !usage.some(item => new URLSearchParams(item.search).get('days') === '7')) throw new Error('Date filters did not reach the usage API')
  console.log('PASS Dates, daily period and timezone reach the usage API')
} finally {
  vite.kill('SIGTERM')
  fixture.closeAllConnections()
  fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
