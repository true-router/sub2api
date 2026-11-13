import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import net from 'node:net'
import crypto from 'node:crypto'

// 失败方式：新请求未停止、旧流被中断、未鉴权停服、错误实例、计数过早归零、进程残留。
const binary = process.argv[2], out = path.resolve(process.argv[3])
fs.mkdirSync(out, { recursive: true })
const secret = path.join(out, 'private-token')
fs.writeFileSync(secret, 'isolated-rollout-control-token-32-bytes', { mode: 0o600 })
const listener = net.createServer(); await new Promise(r => listener.listen(0, '127.0.0.1', r)); const port = listener.address().port; await new Promise(r => listener.close(r))
const child = spawn(binary, [], { env: { ...process.env, E2E_LISTEN: `127.0.0.1:${port}`, TRUEROUTER_ROLLOUT_TOKEN_FILE: secret, TRUEROUTER_INSTANCE_ID: 'isolated-sub-old' }, stdio: ['ignore', 'ignore', 'pipe'] })
let error = ''; child.stderr.on('data', b => { error += b })
const base = `http://127.0.0.1:${port}`, checks = []
const check = (name, ok) => { checks.push({ name, result: ok ? 'PASS' : 'FAIL' }); assert.ok(ok, name + ': ' + error.slice(-400)) }
let result = 'FAIL'
try {
  for (let i = 0; i < 100; i++) { try { if ((await fetch(base + '/health')).ok) break } catch {} await new Promise(r => setTimeout(r, 30)) }
  const health = await fetch(base + '/health')
  check('entrypoint_identity_not_cached', health.headers.get('x-truerouter-instance') === 'isolated-sub-old' && health.headers.get('cache-control') === 'no-store')
  await fetch(base + '/billing')
  const stream = await fetch(base + '/slow'); const reader = stream.body.getReader(); const first = await reader.read()
  check('stream_started', Buffer.from(first.value).toString().includes('started'))
  check('unauthorized_control', (await fetch(base + '/_rollout/drain', { method: 'POST' })).status === 401)
  const headers = { authorization: 'Bearer isolated-rollout-control-token-32-bytes' }
  const draining = await fetch(base + '/_rollout/drain', { method: 'POST', headers }).then(r => r.json())
  check('active_stream_blocks_drain', draining.instance === 'isolated-sub-old' && draining.active_requests === 1 && draining.drained === false)
  check('new_request_rejected', (await fetch(base + '/normal')).status === 503)
  check('readiness_degraded', (await fetch(base + '/health')).status === 503)
  let tail = ''; for (;;) { const row = await reader.read(); if (row.done) break; tail += Buffer.from(row.value).toString() }
  check('old_stream_completed', tail.includes('completed'))
  let drained = await fetch(base + '/_rollout/status', { headers }).then(r => r.json())
  check('billing_queue_blocks_drain', drained.drained === false && drained.active_requests === 0 && drained.pending_usage_records > 0)
  for (let i=0;i<100 && !drained.drained;i++) { await new Promise(r=>setTimeout(r,30)); drained = await fetch(base + '/_rollout/status', { headers }).then(r=>r.json()) }
  check('drain_confirmed', drained.drained === true && drained.active_requests === 0)
  result = 'PASS'
} finally {
  if (child.exitCode === null && child.signalCode === null) { child.kill('SIGTERM'); await once(child, 'exit') }
  fs.rmSync(secret, { force: true })
  fs.writeFileSync(path.join(out, 'summary.json'), JSON.stringify({ result, checks, binary_sha256: crypto.createHash('sha256').update(fs.readFileSync(binary)).digest('hex'), boundary: '真实 Gin、HTTP/SSE 与异步计费任务池；无外部数据库或模型上游', cleanup: child.exitCode !== null || child.signalCode !== null }, null, 2))
}
