import { pathToFileURL } from 'node:url'
import { join } from 'node:path'
import { mkdir, writeFile } from 'node:fs/promises'
import { createServer } from 'node:http'

const [root, project, port, database] = process.argv.slice(2)
const file = (path) => pathToFileURL(join(root, path)).href
const { seedVm, startGateway } = await import(file('test/harness.mjs'))
const token = 'isolated-kernel-token'
const sources = []
const seats = [
  'ba579c2d7e541beca399402d86f9e5b3',
  'bb579c2d7e541beca399402d86f9e5b3',
  'bc579c2d7e541beca399402d86f9e5b3',
]
for (const [index, seat] of seats.entries()) {
  const vm = seedVm({ project, id: 'private-fixture-slot-' + (index + 1) })
  vm.dataplane = 'hermit'
  vm.runtime.kernel_socket = join(project, 'kernel-' + index + '.sock')
  await mkdir(vm.runtime.worker_run_dir, { recursive: true })
  await writeFile(vm.runtime.worker_token_file, token)
  await writeFile(join(project, 'vms', vm.id + '.json'), JSON.stringify(vm))

  let streams = 0
  const source = createServer((req, res) => {
    if (req.headers['x-kin-internal-token'] !== token) return res.writeHead(401).end()
    if (req.url === '/internal/v1/seats') {
      return res.writeHead(200, { 'Content-Type': 'application/json' }).end(JSON.stringify({
        seats: [{
          id: seat, state: 'active', private_marker: vm.id,
          session_id: index < 2 ? 'original-user-session-7f52cef9-5830' : 'unassigned-original-session',
        }],
      }))
    }
    if (req.url !== `/internal/v1/seats/${seat}/terminal`) return res.writeHead(404).end()
    res.writeHead(200, {
      'Content-Type': 'application/octet-stream',
      'X-Terminal-Columns': '140',
      'X-Terminal-Rows': '40',
      'X-Private-Fixture': vm.id,
      Location: 'http://fixture.invalid/private',
    })
    res.write('\u001b[2J\u001b[H\u001b[36mClaude Code\u001b[0m\r\n\r\n> Terminal ready\r\n')
    res.write(`Fixture slot ${index + 1}\r\n`)
    const update = setTimeout(() => res.write('\u001b[32m● Live update received\u001b[0m\r\n'), 750)
    streams++
    const burst = index === 0 && (streams === 2 || streams === 3)
      ? setTimeout(() => res.write(Buffer.alloc(384 * 1024, 65)), 4000)
      : null
    res.once('close', () => {
      clearTimeout(update)
      clearTimeout(burst)
      void writeFile(join(project, 'terminal-detached'), 'yes')
    })
  })
  await new Promise((resolve, reject) =>
    source.once('error', reject).listen(vm.runtime.kernel_socket, resolve),
  )
  sources.push(source)
}
const gateway = await startGateway({
  project,
  port: Number(port),
  operations: false,
  env: {
    VM2API_DATABASE_URL: database,
    KIN_POSTGRES_TEST_URL: database,
    VM2API_DB_SECRET: 'isolated-fixture',
    KIN_TEST_MODELS_DEV: '1',
    NODE_OPTIONS: '--import=' + file('test/fixtures/postgresql-proxy-pool-gateway.mjs'),
  },
})
process.once('SIGTERM', async () => {
  await gateway.stop()
  for (const source of sources) source.close()
  process.exit()
})
