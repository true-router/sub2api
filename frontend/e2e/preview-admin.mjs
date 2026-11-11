/** Run from frontend: node e2e/preview-admin.mjs [--account]. Requires pnpm. */
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const accountPreview = process.argv.includes('--account')
const port = accountPreview ? 13032 : 13031
const apiPort = accountPreview ? 18082 : 18081
const origin = `http://127.0.0.1:${port}`
const artifacts = resolve(root, accountPreview ? '../output/account-boardui' : '../output/admin-boardui')
await mkdir(artifacts, { recursive: true })
const fixture = createAdminFixtureServer({ accountPreview })
await new Promise(resolve => fixture.listen(apiPort, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', String(port), '--strictPort'], { cwd: root, env: { ...process.env, VITE_ACCOUNT_PREVIEW: String(accountPreview), VITE_DEV_PROXY_TARGET: `http://127.0.0.1:${apiPort}` }, stdio: ['ignore', 'pipe', 'pipe'] })
let serverLog = ''
vite.stdout.on('data', chunk => { serverLog += chunk })
vite.stderr.on('data', chunk => { serverLog += chunk })
const stop = () => { vite.kill('SIGTERM'); fixture.close() }
process.on('SIGINT', stop)
process.on('SIGTERM', stop)
try {
  let ready = false
  for (let attempt = 0; attempt < 120; attempt++) {
    if (vite.exitCode !== null) throw new Error(`Vite exited: ${serverLog}`)
    try { if ((await fetch(origin)).ok) { ready = true; break } } catch { /* wait for Vite */ }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  if (!ready) throw new Error(`Vite did not start: ${serverLog}`)
  console.log(`Local preview ready: ${origin}${accountPreview ? '/keys' : '/admin/dashboard'}`)
  await new Promise(resolve => {
    process.once('SIGINT', resolve)
    process.once('SIGTERM', resolve)
    vite.once('exit', resolve)
  })
} finally {
  stop()
  await writeFile(resolve(artifacts, 'preview-vite.log'), serverLog)
}
