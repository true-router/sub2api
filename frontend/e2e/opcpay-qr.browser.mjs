const fs = await import('node:fs/promises')
const crypto = await import('node:crypto')
const task = await taskSpace('sub2api-opcpay-direct-qr')
const page = task.page('p1')
const output = "__ARTIFACT_DIRECTORY__"
const root = "__REPOSITORY_ROOT__"
const checks = [], scans = []
const check = (name, passed) => { checks.push({ name, passed }); console.log((passed ? 'PASS ' : 'FAIL ') + name) }
const desktop = { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false }
const userAgent = await page.evaluate(() => navigator.userAgent)
await page.cdp('Emulation.setDeviceMetricsOverride', desktop)
await page.goto('http://127.0.0.1:13042/login')
await page.evaluate(() => {
  localStorage.setItem('auth_token', 'local-member-preview')
  localStorage.setItem('auth_user', JSON.stringify({ id: 2, role: 'user', email: 'member1@example.test', balance: 100 }))
  localStorage.setItem('sub2api_locale', 'zh')
  localStorage.setItem('theme', 'light')
  localStorage.setItem('user_guide_2_user_v4_interactive', 'true')
})
await page.goto('http://127.0.0.1:13042/purchase')
await page.waitForSelector('[data-testid="opcpay-card"]')
console.log(await page.snapshot())
async function configure(purchaseURL, purchaseQR) {
  await page.fetch('/api/v1/payment/opcpay/__fixture', { method: 'POST',
    headers: { 'content-type': 'application/json' }, body: JSON.stringify({ purchaseURL, purchaseQR }) })
}
async function open() {
  await page.click('[data-testid="opcpay-card"]')
  await page.click('loc=role:button[name="支付宝购买 · CNY 10.00"]')
  await page.waitForFunction(() => !document.querySelector('.modal-footer button')?.disabled)
}
async function close() {
  await page.click('.modal-header button[aria-label="关闭"]')
  await page.waitForFunction(() => !document.querySelector('[data-testid="opcpay-modal"]') && !document.querySelector('[data-testid="opcpay-card"]').disabled)
}
async function decode() {
  await page.waitForSelector('img[alt="支付宝购买入口二维码"]')
  await page.waitForFunction(() => document.querySelector('img[alt="支付宝购买入口二维码"]')?.naturalWidth > 0)
  return page.evaluate(async () => {
    const image = document.querySelector('img[alt="支付宝购买入口二维码"]')
    const codes = await new BarcodeDetector({ formats: ['qr_code'] }).detect(image)
    if (codes.length !== 1) throw new Error('Expected one readable checkout QR')
    return codes[0].rawValue
  })
}
const purchaseURL = 'https://pay.example.test/direct?intent=card-10'
try {
  for (const [name, purchaseQR] of [
    ['QR image address', 'https://qr.example.test/rendered-payment.png'],
    ['QR rendering endpoint', 'https://mobilecodec.alipay.com/show.htm?code=synthetic'],
    ['Missing upstream QR', ''],
    ['Unsafe upstream QR', 'javascript:alert(1)']
  ]) {
    await configure(purchaseURL, purchaseQR)
    await open()
    const decoded = await decode()
    scans.push({ scenario: name, decoded, expected: purchaseURL, upstream_qr_address: purchaseQR })
    check(name + ': one scan resolves to the payment entry', decoded === purchaseURL)
    check(name + ': showing a QR does not report successful credit', !(await page.evaluate(() => document.body.innerText.includes('充值完成。'))))
    if (name === 'QR image address') {
      await page.screenshot({ path: output + '/desktop.png' })
      const business = await page.evaluate(() => document.querySelector('[data-testid="opcpay-modal"]').textContent.match(/业务编号：([^\s]+)/)[1])
      await page.reload()
      const restored = await decode()
      check('Resumed operation keeps the same direct payment entry', restored === purchaseURL)
      check('Resumed purchase retains its business ID', await page.evaluate(id => document.querySelector('[data-testid="opcpay-modal"]').textContent.includes(id), business))
    }
    await close()
  }
  const mobileURL = 'alipays://platformapi/startapp?appId=synthetic&scene=payment'
  await configure(mobileURL, 'https://qr.example.test/mobile-image.png')
  await page.cdp('Network.setUserAgentOverride', { userAgent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 Safari/604.1' })
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true })
  await page.reload()
  await page.waitForSelector('[data-testid="opcpay-card"]')
  await open()
  await page.waitForSelector('[data-testid="opcpay-modal"] a')
  check('Mobile opens the canonical Alipay entry', await page.evaluate(url => document.querySelector('[data-testid="opcpay-modal"] a').getAttribute('href') === url, mobileURL))
  check('Mobile does not introduce a second QR', !(await page.evaluate(() => !!document.querySelector('img[alt="支付宝购买入口二维码"]'))))
  await page.screenshot({ path: output + '/mobile.png' })
  await close()
} finally {
  await page.cdp('Network.setUserAgentOverride', { userAgent })
  await page.cdp('Emulation.setDeviceMetricsOverride', desktop)
  const receipt = { passed: checks.length === 12 && checks.every(check => check.passed), checks, scans,
    mode: 'real Vue checkout and native BarcodeDetector; synthetic provider and session; no real payment or consumption', source_sha256: {} }
  for (const name of ['frontend/src/components/payment/OPCPayPanel.vue', 'frontend/e2e/opcpay-qr.mjs', 'frontend/e2e/opcpay-qr.browser.mjs']) {
    receipt.source_sha256[name] = crypto.createHash('sha256').update(await fs.readFile(root + '/' + name)).digest('hex')
  }
  await fs.writeFile(output + '/checks.json', JSON.stringify(receipt, null, 2))
}
if (!checks.every(check => check.passed) || checks.length !== 12) throw new Error('One scan must reach the payment entry rather than another QR')
await task.finish({ keep: [] })
