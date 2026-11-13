import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createSSRApp, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createI18n } from 'vue-i18n'

// Render actual Vue views using payloads from the disposable PostgreSQL/HTTP
// harness. No browser, fake component, new dependency or production request.
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const receiptPath = resolve(root, '../output/opcpay-e2e/checks.json')
const receiptBytes = await readFile(receiptPath)
const receipt = JSON.parse(receiptBytes)
assert.equal(receipt.failed, 0)
assert.equal(receipt.mode, 'simulated-provider-real-postgresql-http')
for (const [path, expected] of Object.entries(receipt.source_sha256)) {
  const actual = createHash('sha256').update(await readFile(resolve(root, '../backend', path))).digest('hex')
  assert.equal(actual, expected, `Stale HTTP receipt for ${path}; run backend/tools/opcpay-e2e/run.sh first`)
}
const output = resolve(root, '../output/opcpay-order-render')
await mkdir(output, { recursive: true })
const server = await createServer({
  root,
  configFile: false,
  plugins: [vue()],
  resolve: { alias: { '@': resolve(root, 'src') } },
  server: { middlewareMode: true },
  appType: 'custom',
})
const checks = []
const sources = {}
async function load(path) {
  sources[path] = createHash('sha256').update(await readFile(resolve(root, path))).digest('hex')
  return (await server.ssrLoadModule('/' + path)).default
}
try {
  for (const path of ['e2e/opcpay-orders.render.mjs', 'src/components/admin/payment/AdminOrderDetail.vue', 'src/components/admin/payment/AdminOrderTable.vue', 'src/views/admin/orders/AdminOrdersView.vue', 'src/components/payment/OrderTable.vue']) {
    sources[path] = createHash('sha256').update(await readFile(resolve(root, path))).digest('hex')
  }
  const Stats = await load('src/components/admin/payment/OrderStatsCards.vue')
  for (const locale of ['zh', 'en']) {
    const messages = await load(`src/i18n/locales/${locale}/index.ts`)
    const render = async (component, props, name) => {
      const app = createSSRApp({ render: () => h(component, props) })
      app.use(createI18n({ legacy: false, locale, messages: { [locale]: messages } }))
      const html = await renderToString(app)
      await writeFile(resolve(output, `${locale}-${name}.html`), html)
      return html
    }
    const unknown = receipt.unknown_payment_dashboard
    assert.ok(unknown.total_count > 0)
    assert.deepEqual(unknown.avg_amount, {})
    const empty = await render(Stats, { stats: unknown }, 'unknown')
    const noData = messages.payment.admin.noData
    assert.equal(empty.split(noData).length - 1, 3, `${locale}: empty revenue and average cards must show a visible fallback`)
    checks.push(`${locale}: empty monetary statistics remain visible`)

    const known = await render(Stats, { stats: receipt.orders.dashboard }, 'known')
    assert.ok(known.includes('9.90'))
    checks.push(`${locale}: HTTP purchase snapshot price appears in statistics`)
    const corrected = await render(Stats, { stats: receipt.orders.corrected_dashboard }, 'reconciled')
    assert.ok(corrected.includes('38.95'))
    checks.push(`${locale}: reconciled actual payment restores the average`)
    const zero = await render(Stats, { stats: {
      ...unknown, today_amount: { CNY: 0 }, total_amount: { CNY: 0 }, avg_amount: { CNY: 0 },
    } }, 'zero')
    assert.ok(zero.includes('0.00') && !zero.includes(noData))
    checks.push(`${locale}: zero monetary values are not treated as missing`)
  }
  await writeFile(resolve(output, 'summary.json'), JSON.stringify({
    result: 'PASS', checks, source_sha256: sources,
    backend_receipt_sha256: createHash('sha256').update(receiptBytes).digest('hex'),
    boundary: 'Real PostgreSQL/HTTP receipt rendered through actual Vue components; no browser or production traffic.',
  }, null, 2) + '\n')
  console.log(JSON.stringify({ result: 'PASS', checks: checks.length, output }))
} finally {
  await server.close()
}
