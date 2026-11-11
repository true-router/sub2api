// Synthetic sessions and a simulated payment provider for browser acceptance.
import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'

const root = resolve(import.meta.dirname, '..')
const backend = JSON.parse(await readFile(resolve(root, '../output/opcpay-e2e/preview.json'), 'utf8')).api_url
const base = createAdminFixtureServer({ accountPreview: true })
await new Promise(resolve => base.listen(0, '127.0.0.1', resolve))
const fixtureOrigin = `http://127.0.0.1:${base.address().port}`
const proxy = createServer(async (req, res) => {
  try {
    const url = new URL(req.url, 'http://127.0.0.1')
    const body = []
    for await (const part of req) body.push(part)
    const direct = /^\/api\/v1\/(admin\/payment\/(providers|config)|payment\/opcpay)/.test(url.pathname)
    const origin = direct ? backend : fixtureOrigin
    const path = direct ? req.url.replace(/^\/api\/v1/, '') : req.url
    const payload = Buffer.concat(body)
    const response = await fetch(origin + path, { method: req.method, headers: { 'content-type': 'application/json', authorization: req.headers.authorization || '', 'X-Fixture-Session': 'one' }, ...(payload.length ? { body: payload } : {}) })
    let data = await response.text()
    if (['/api/v1/settings/public', '/api/v1/admin/settings'].includes(url.pathname)) {
      const envelope = JSON.parse(data)
      Object.assign(envelope.data, { payment_enabled: true, payment_enabled_types: ['opcpay'], payment_balance_disabled: false })
      data = JSON.stringify(envelope)
    }
    res.writeHead(response.status, { 'content-type': 'application/json', 'cache-control': 'no-store' })
    res.end(data)
  } catch {
    res.writeHead(502, { 'content-type': 'application/json' })
    res.end(JSON.stringify({ code: 502, message: '模拟验收接口不可用' }))
  }
})
await new Promise(resolve => proxy.listen(18084, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13034', '--strictPort'], { cwd: root, env: { ...process.env, VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18084' }, stdio: 'inherit' })
const close = () => { vite.kill('SIGTERM'); proxy.close(); base.close() }
process.on('SIGINT', close)
process.on('SIGTERM', close)
console.log('独立模拟验收：http://127.0.0.1:13034')
