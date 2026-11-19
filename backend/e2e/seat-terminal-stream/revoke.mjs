import assert from 'node:assert/strict'

const [base, id, token] = process.argv.slice(2)
const address = new URL('/api/v1/seat-terminals/' + id + '/stream', base)
address.protocol = 'ws:'
const viewer = new WebSocket(address, ['sub2api-terminal', 'jwt.' + token])
viewer.binaryType = 'arraybuffer'
let received = 0
const timer = setTimeout(() => { throw new Error('revoked stream remained open') }, 12000)
viewer.onerror = () => { throw new Error('viewer failed before revocation') }
viewer.onmessage = ({ data }) => {
  if (!(data instanceof ArrayBuffer)) return
  if (received === 0) console.log('ready')
  received += data.byteLength
  viewer.send(String(received))
}
await new Promise((resolve) => { viewer.onclose = resolve })
clearTimeout(timer)
assert(received > 0)
