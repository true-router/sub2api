import assert from 'node:assert/strict'

const [base, seat, otherSeat, unassignedSeat, admin, member] = process.argv.slice(2)
const address = new URL(`/api/v1/admin/seat-terminals/${seat}/stream`, base)
address.protocol = 'ws:'
const protocols = (token) => ['sub2api-admin', `jwt.${token}`]
const viewer = new WebSocket(address, protocols(admin))
viewer.binaryType = 'arraybuffer'
const deadline = setTimeout(() => { throw new Error('terminal did not update') }, 10000)
await new Promise((resolve, reject) => {
  let text = ''
  let bytes = 0
  viewer.onerror = reject
  viewer.onclose = () => reject(new Error('terminal closed before update'))
  viewer.onmessage = ({ data }) => {
    if (typeof data === 'string') {
      assert.deepEqual(JSON.parse(data), { columns: 140, rows: 40 })
      return
    }
    text += Buffer.from(data).toString()
    bytes += data.byteLength
    viewer.send(String(bytes))
    if (!text.includes('Live update received')) return
    assert(text.includes('Terminal ready') && text.includes('\u001b[36m'))
    assert(!text.includes('private-fixture-slot') && !text.includes('fixture.invalid'))
    assert.equal(viewer.protocol, 'sub2api-admin')
    viewer.close()
    resolve()
  }
})
for (const protocol of [[], protocols(member)]) {
  await new Promise((resolve, reject) => {
    const unauthorized = new WebSocket(address, protocol)
    unauthorized.onopen = () => reject(new Error('unauthorized terminal connection accepted'))
    unauthorized.onerror = resolve
  })
}
clearTimeout(deadline)

const slow = new WebSocket(address, protocols(admin))
slow.binaryType = 'arraybuffer'
const fast = new WebSocket(address, protocols(admin))
fast.binaryType = 'arraybuffer'
let slowBytes = 0
let fastBytes = 0
const maxWait = setTimeout(() => { throw new Error('slow viewer was not bounded') }, 13000)
const throughput = await new Promise((resolve, reject) => {
  slow.onmessage = ({ data }) => {
    if (data instanceof ArrayBuffer) slowBytes += data.byteLength
  }
  fast.onmessage = ({ data }) => {
    if (!(data instanceof ArrayBuffer)) return
    fastBytes += data.byteLength
    fast.send(String(fastBytes))
  }
  slow.onerror = reject
  fast.onerror = reject
  slow.onclose = () => {
    assert(slowBytes <= 256 * 1024)
    assert(fastBytes >= 384 * 1024)
    resolve({ slowBytes, fastBytes })
  }
})
clearTimeout(maxWait)
fast.close()
function checkSeat(id, token, adminView, canView) {
  return new Promise((resolve, reject) => {
    const prefix = adminView ? '/api/v1/admin/seat-terminals/' : '/api/v1/seat-terminals/'
    const url = new URL(prefix + id + '/stream', base)
    url.protocol = 'ws:'
    const expectedProtocol = adminView ? 'sub2api-admin' : 'sub2api-terminal'
    const client = new WebSocket(url, [expectedProtocol, `jwt.${token}`])
    client.binaryType = 'arraybuffer'
    const timer = setTimeout(() => { client.close(); reject(new Error('seat check timed out')) }, 5000)
    let hasData = false
    client.onmessage = ({ data }) => {
      assert(canView)
      assert.equal(client.protocol, expectedProtocol)
      if (data instanceof ArrayBuffer) {
        assert(Buffer.from(data).includes('Terminal ready'))
        hasData = true
        client.close()
      }
    }
    client.onclose = () => {
      clearTimeout(timer)
      assert.equal(hasData, canView)
      resolve()
    }
    client.onerror = () => { clearTimeout(timer); reject(new Error('seat check failed')) }
  })
}
await checkSeat(seat, member, false, true)
await checkSeat(otherSeat, member, false, false)
await checkSeat(unassignedSeat, member, false, false)
await checkSeat(otherSeat, admin, true, true)
await checkSeat(unassignedSeat, admin, true, false)
console.log(JSON.stringify(throughput))
