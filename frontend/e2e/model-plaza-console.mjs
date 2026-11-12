/** Console model plaza E2E: node e2e/model-plaza-console.mjs. Artifacts: output/model-plaza-console. */
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { readFile, mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
const root = resolve(import.meta.dirname, '..')
const output = resolve(root, '../output/model-plaza-console')
await mkdir(output, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests, accountPreview: true, dashboardPreview: true, modelPlazaPreview: true })
await new Promise(resolve => fixture.listen(18087, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13037', '--strictPort'], { cwd: root, env: { ...process.env, VITE_ACCOUNT_PREVIEW: 'true', VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18087' }, stdio: ['ignore', 'pipe', 'pipe'] })
let log = ''
vite.stdout.on('data', chunk => { log += chunk })
vite.stderr.on('data', chunk => { log += chunk })
try {
  for (let i = 0; i < 120; i++) {
    try { if ((await fetch('http://127.0.0.1:13037')).ok) break } catch { /* server startup */ }
    if (vite.exitCode !== null || i === 119) throw new Error(log)
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  const source = (await readFile(resolve(import.meta.dirname, 'model-plaza-console.browser.mjs'), 'utf8')).replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output))
  const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code) throw new Error('Model plaza console acceptance failed')
  if (!requests.some(r => r.path === '/model-plaza')) throw new Error('Catalog API was not requested')

} finally {
  vite.kill('SIGTERM')
  fixture.closeAllConnections()
  fixture.close()
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
  await writeFile(resolve(output, 'vite.log'), log)
}
