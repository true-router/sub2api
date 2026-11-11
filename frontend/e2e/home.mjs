/** Reproducible home E2E. Run: node e2e/home.mjs. No production API calls. */
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { readFile, mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/home-boardui')
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true })
const [originalHandler] = fixture.listeners('request')
fixture.removeAllListeners('request')
const baseSettings = { site_name: 'TrueRouter', site_subtitle: '满血 AI，不掺一滴水', site_logo: '', compact_home_enabled: true, home_content: '', api_base_url: 'https://api.example.test/v1/', model_plaza_enabled: true, model_plaza_require_auth: false, registration_enabled: true, backend_mode_enabled: false, doc_url: 'https://docs.example.test' }
let settings = { ...baseSettings }
fixture.on('request', async (req, res) => {
  const path = new URL(req.url, 'http://127.0.0.1').pathname
  if (path === '/api/v1/__home_fixture' && req.method === 'POST') {
    let body = ''
    for await (const chunk of req) body += chunk
    settings = { ...baseSettings, ...JSON.parse(body) }
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify({ code: 0, data: settings }))
  } else if (path === '/api/v1/settings/public') {
    res.setHeader('Content-Type', 'application/json')
    res.end(JSON.stringify({ code: 0, data: settings }))
  } else await originalHandler(req, res)
})
await new Promise(resolve => fixture.listen(18087, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13037', '--strictPort'], { cwd: root, env: { ...process.env, VITE_ACCOUNT_PREVIEW: 'true', VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18087' }, stdio: ['ignore', 'pipe', 'pipe'] })
let log = ''
vite.stdout.on('data', chunk => { log += chunk })
vite.stderr.on('data', chunk => { log += chunk })
try {
  for (let i = 0; i < 120; i++) {
    try { if ((await fetch('http://127.0.0.1:13037')).ok) break } catch { /* wait for startup */ }
    if (vite.exitCode !== null || i === 119) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'home.browser.mjs'), 'utf8')).replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Home browser acceptance failed')
} finally {
  vite.kill('SIGTERM')
  fixture.closeAllConnections()
  fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
