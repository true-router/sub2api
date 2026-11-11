const fs = await import('node:fs/promises')
const crypto = await import('node:crypto')
const task = await taskSpace('sub2api-opcpay-modal-close')
const page = task.page('p1')
const output = "__ARTIFACT_DIRECTORY__"
await page.goto('http://127.0.0.1:13041/login')
await page.evaluate(() => {
  localStorage.setItem('auth_token', 'local-member-preview')
  localStorage.setItem('auth_user', JSON.stringify({ id: 2, role: 'user', email: 'member1@example.test', balance: 100 }))
  localStorage.setItem('sub2api_locale', 'zh')
  localStorage.setItem('theme', 'light')
  localStorage.setItem('user_guide_2_user_v4_interactive', 'true')
})
await page.goto('http://127.0.0.1:13041/purchase')
await page.waitForSelector('[data-testid="opcpay-card"]')
console.log(await page.snapshot())
await page.click('[data-testid="opcpay-card"]')
await page.click('loc=role:button[name="支付宝购买 · CNY 10.00"]')
await page.waitForSelector('img[alt="支付宝购买入口二维码"]')
await page.waitForFunction(() => !document.querySelector('.modal-footer button').disabled)
await page.evaluate(() => {
  const open = XMLHttpRequest.prototype.open, send = XMLHttpRequest.prototype.send
  XMLHttpRequest.prototype.open = function(method, url, ...rest) { this.__closeURL = url; return open.call(this, method, url, ...rest) }
  XMLHttpRequest.prototype.send = function(...args) {
    if (String(this.__closeURL).endsWith('/close')) window.__closeProbe.closeSentWithVisibleModal.push(!!document.querySelector('[data-testid="opcpay-modal"]'))
    return send.apply(this, args)
  }
  window.__startCloseProbe = () => {
    const probe = window.__closeProbe = { emptyFrames: 0, frames: 0, done: false, closeSentWithVisibleModal: [] }
    function frame() {
      const body = document.querySelector('[data-testid="opcpay-modal"]')
      if (!body) { probe.done = true; return }
      probe.frames++
      if (getComputedStyle(body.closest('.modal-overlay')).opacity > 0.01 && !body.textContent.trim() && !body.querySelector('img')) probe.emptyFrames++
      requestAnimationFrame(frame)
    }
    requestAnimationFrame(frame)
  }
  window.__startCloseProbe()
})
await page.click('.modal-header button[aria-label="关闭"]')
await page.waitForFunction(() => window.__closeProbe.emptyFrames > 0 || window.__closeProbe.done)
const probe = await page.evaluate(() => window.__closeProbe)
await page.screenshot({ path: output + (probe.emptyFrames ? '/before.png' : '/after.png') })
const checks = [{ name: 'Slow post-close refresh never exposes an empty modal', passed: probe.emptyFrames === 0 }]
const check = (name, passed) => { checks.push({ name, passed }); if (!passed) throw new Error(name) }
if (!probe.emptyFrames) {
  await page.waitForFunction(() => !document.querySelector('[data-testid="opcpay-card"]').disabled)
  const finalProbe = await page.evaluate(() => window.__closeProbe)
  check('Purchase cleanup starts after the modal leaves', finalProbe.closeSentWithVisibleModal.length === 1 && finalProbe.closeSentWithVisibleModal[0] === false)
  const openPurchase = async () => {
    await page.click('[data-testid="opcpay-card"]')
    await page.click('loc=role:button[name="支付宝购买 · CNY 10.00"]')
    await page.waitForSelector('img[alt="支付宝购买入口二维码"]')
    await page.waitForFunction(() => !document.querySelector('.modal-footer button').disabled)
  }
  await openPurchase()
  const originalCode = await page.evaluate(() => document.querySelector('[data-testid="opcpay-modal"]').textContent.match(/业务编号：([^\s]+)/)[1])
  await page.fetch('/api/v1/payment/opcpay/__fixture', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ failClose: true }) })
  await page.click('.modal-header button[aria-label="关闭"]')
  await page.waitForSelector('.modal-body [role="alert"]')
  check('Failed close restores the original QR and business ID', await page.evaluate(code => !!document.querySelector('img[alt="支付宝购买入口二维码"]') && document.querySelector('[data-testid="opcpay-modal"]').textContent.includes(code), originalCode))
  await page.screenshot({ path: output + '/close-failure-restored.png' })
  await page.evaluate(() => window.__startCloseProbe())
  await page.click('.modal-header button[aria-label="关闭"]')
  await page.waitForFunction(() => window.__closeProbe.done)
  await page.waitForFunction(() => !document.querySelector('[data-testid="opcpay-card"]').disabled)
  check('Retry closes without an empty frame', await page.evaluate(() => window.__closeProbe.emptyFrames === 0 && window.__closeProbe.closeSentWithVisibleModal.length === 1))
  await openPurchase()
  await page.evaluate(() => window.__startCloseProbe())
  await page.keyboard.press('Escape')
  await page.keyboard.press('Escape')
  await page.waitForFunction(() => window.__closeProbe.done)
  await page.waitForFunction(() => !document.querySelector('[data-testid="opcpay-card"]').disabled)
  check('Repeated Escape closes once without clearing visible content', await page.evaluate(() => window.__closeProbe.emptyFrames === 0 && window.__closeProbe.closeSentWithVisibleModal.length === 1 && window.__closeProbe.closeSentWithVisibleModal[0] === false))
}
const root = "__REPOSITORY_ROOT__"
const receipt = { passed: checks.every(item => item.passed), checks, probe, mode: 'real Vue payment page; synthetic provider only', source_sha256: {} }
for (const name of ['frontend/src/components/payment/OPCPayPanel.vue', 'frontend/src/components/common/BaseDialog.vue']) receipt.source_sha256[name] = crypto.createHash('sha256').update(await fs.readFile(root + name)).digest('hex')
await fs.writeFile(output + (receipt.passed ? '/after.json' : '/before.json'), JSON.stringify(receipt, null, 2))
console.log(JSON.stringify(receipt))
if (!receipt.passed) throw new Error('Visible payment modal became empty before it closed')
await task.finish({ keep: [] })
