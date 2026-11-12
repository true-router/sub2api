// The usage table talks to real Go handlers and disposable PostgreSQL.
import { createServer } from 'node:http'
import { createServer as createNetServer } from 'node:net'
import { spawn } from 'node:child_process'
import { readFile, mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import XLSX from 'xlsx'

const root = resolve(import.meta.dirname, '../..')
const output = resolve(root, 'output/usage-session')
await mkdir(output, { recursive: true })
async function freePort() {
  const server = createNetServer()
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve))
  const port = server.address().port
  await new Promise(resolve => server.close(resolve))
  return port
}
const backendPort = await freePort()
const vitePort = await freePort()
const fixture = createAdminFixtureServer()
await new Promise(resolve => fixture.listen(0, '127.0.0.1', resolve))
const fixtureOrigin = `http://127.0.0.1:${fixture.address().port}`
const backendOrigin = `http://127.0.0.1:${backendPort}`
const logs = []
const children = []
function start(command, args, options = {}) {
  const child = spawn(command, args, { cwd: root, detached: true, stdio: ['ignore', 'pipe', 'pipe'], ...options })
  child.stdout.on('data', chunk => logs.push(String(chunk)))
  child.stderr.on('data', chunk => logs.push(String(chunk)))
  children.push(child)
  return child
}
async function ready(origin, child) {
  for (let i = 0; i < 240; i++) {
    if (child.exitCode != null) throw new Error(logs.join(''))
    try { if ((await fetch(origin)).ok) return } catch { /* wait for startup */ }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  throw new Error(`Startup timed out: ${logs.join('')}`)
}
const requests = []
const proxy = createServer(async (req, res) => {
  try {
    const path = new URL(req.url, 'http://fixture.invalid').pathname
    requests.push({ method: req.method, url: req.url })
    const actual = path === '/api/v1/admin/usage' || /^\/api\/v1\/admin\/users\/\d+$/.test(path)
    let body = ''
    for await (const chunk of req) body += chunk
    const response = await fetch((actual ? backendOrigin : fixtureOrigin) + req.url, {
      method: req.method, headers: { 'content-type': 'application/json' }, ...(body ? { body } : {})
    })
    res.writeHead(response.status, { 'content-type': 'application/json' })
    res.end(await response.text())
  } catch (error) { res.writeHead(500); res.end(JSON.stringify({ message: String(error) })) }
})
try {
  const backend = start('sh', ['backend/tools/usage-session-e2e/run.sh', '--listen', `127.0.0.1:${backendPort}`])
  await ready(backendOrigin + '/health', backend)
  await new Promise(resolve => proxy.listen(0, '127.0.0.1', resolve))
  const vite = start('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', String(vitePort), '--strictPort'], {
    cwd: resolve(root, 'frontend'),
    env: { ...process.env, VITE_DEV_PROXY_TARGET: `http://127.0.0.1:${proxy.address().port}` }
  })
  const origin = `http://127.0.0.1:${vitePort}`
  await ready(origin, vite)
  const source = (await readFile(resolve(import.meta.dirname, 'usage-session.browser.mjs'), 'utf8'))
    .replace('"__ORIGIN__"', JSON.stringify(origin)).replace('"__OUTPUT__"', JSON.stringify(output))
  const browser = spawn('ego-browser', ['nodejs'], { stdio: ['pipe', 'inherit', 'inherit'] })
  browser.stdin.end(source)
  const code = await new Promise((resolve, reject) => { browser.once('error', reject); browser.once('exit', resolve) })
  if (code !== 0) throw new Error(`Browser acceptance exited ${code}`)
  const workbook = XLSX.readFile(resolve(output, 'session.xlsx'))
  const rows = XLSX.utils.sheet_to_json(workbook.Sheets[workbook.SheetNames[0]])
  const exportValid = rows.length === 2 && rows.every(row => row['会话 ID'] === 'session-shared' && row['用户'] === 'session-one@example.test')
  const receiptPath = resolve(output, 'receipt.json')
  const receipt = JSON.parse(await readFile(receiptPath, 'utf8'))
  receipt.checks.push({ name: 'Downloaded workbook contains exactly the selected user and session', passed: exportValid })
  await writeFile(receiptPath, JSON.stringify(receipt, null, 2))
  if (!exportValid) throw new Error('Filtered workbook has incorrect contents')
  console.log('PASS Downloaded workbook contains exactly the selected user and session')
} finally {
  for (const child of children.reverse()) { try { process.kill(-child.pid, 'SIGTERM') } catch { /* already stopped */ } }
  proxy.closeAllConnections()
  proxy.close()
  fixture.closeAllConnections()
  fixture.close()
  await writeFile(resolve(output, 'server.log'), logs.join(''))
  await writeFile(resolve(output, 'requests.json'), JSON.stringify(requests, null, 2))
}
