/* global taskSpace */
const fs = await import('node:fs/promises'), crypto = await import('node:crypto')
const task = await taskSpace('sub2api-opcpay-confirmation-recovery'), page = task.page('p1')
const output = "__OUTPUT__", root = "__ROOT__"
const checks = []
const check = (name, passed) => { checks.push({ name, passed }); console.log((passed ? 'PASS ' : 'FAIL ') + name) }
const state = () => page.evaluate(() => JSON.parse(sessionStorage.getItem('opcpay-action:2') || 'null'))
const fixture = async mode => {
  const result = await page.fetch('/api/v1/payment/opcpay/__fixture', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ mode }) })
  return (typeof result.body === 'string' ? JSON.parse(result.body) : result.body).data
}
const idle = () => page.waitForFunction(() => document.querySelector('[data-testid="opcpay-panel"] button') && !document.querySelector('[data-testid="opcpay-panel"] button').disabled)
const open = async () => {
  await page.click('[data-testid="opcpay-card"]')
  await page.click('[data-testid="opcpay-panel"] > button')
  await page.waitForSelector('img[alt="支付宝购买入口二维码"]'); await idle()
}
const confirm = async () => { await page.click('.modal-footer button'); await idle() }
const close = async () => { await page.click('.modal-header button[aria-label="关闭"]'); await page.waitForSelector('[data-testid="opcpay-modal"]', { state: 'hidden' }); await idle() }
await page.goto('http://127.0.0.1:13043/login')
await page.evaluate(() => {
  localStorage.setItem('auth_token', 'local-member-preview')
  localStorage.setItem('auth_user', JSON.stringify({ id: 2, role: 'user', email: 'member1@example.test', balance: 100 }))
  localStorage.setItem('sub2api_locale', 'zh'); localStorage.setItem('theme', 'light')
  localStorage.setItem('user_guide_2_user_v4_interactive', 'true')
  sessionStorage.setItem('opcpay-action:2', JSON.stringify({ id: 'seeded-purchase-0001', sku: 'ten-usd', phase: 'purchase' }))
})
await page.goto('http://127.0.0.1:13043/purchase')
await page.waitForSelector('img[alt="支付宝购买入口二维码"]'); await idle()
console.log(await page.snapshot())
try {
  check('Checkout shows payment amount and credited USD', await page.evaluate(() => document.body.innerText.includes('CNY 9.90') && document.body.innerText.includes('10 USD')))
  check('Checkout hides internal points and redemption terminology', !(await page.evaluate(() => /积分|核销|权益/.test(document.querySelector('[data-testid="opcpay-panel"]').innerText + document.querySelector('[data-testid="opcpay-modal"]').innerText))))
  const original = (await state()).id
  await confirm()
  check('Unpaid confirmation shows the requested retry message', await page.evaluate(() => document.querySelector('.modal-body [role="alert"]')?.textContent === '验证失败，请重试'))
  check('Definitive rejection restores purchase phase and original action', (await state()).phase === 'purchase' && (await state()).id === original)
  check('No unpaid deduction or credit occurs', (await fixture()).deductions === 0 && (await fixture()).credits === 0)
  await confirm()
  check('Repeated unpaid confirmation keeps the same purchase action', (await state()).phase === 'purchase' && (await state()).id === original)
  await page.evaluate(() => { const old = JSON.parse(sessionStorage.getItem('opcpay-action:2')); old.phase = 'consume'; sessionStorage.setItem('opcpay-action:2', JSON.stringify(old)) })
  await page.reload(); await page.waitForSelector('[data-testid="opcpay-card"]'); await idle()
  const recovered = await page.evaluate(() => !!document.querySelector('img[alt="支付宝购买入口二维码"]'))
  check('Previously stuck unpaid session restores the original checkout', recovered && (await state()).phase === 'purchase')
  if (recovered) {
    await close()
    const unlocked = await page.evaluate(() => !document.querySelector('[data-testid="opcpay-card"]').disabled && !sessionStorage.getItem('opcpay-action:2'))
    check('Closing after rejection permits a new payment', unlocked)
    if (unlocked) {
      await open(); check('New checkout gets a new action', (await state()).id !== original)
      await fixture('paid'); await confirm()
      check('Successful credit shows only the credited amount', await page.evaluate(() => document.querySelector('[data-testid="opcpay-modal"]').textContent.includes('充值成功，已到账 10 USD。') && !/积分|核销/.test(document.querySelector('[data-testid="opcpay-modal"]').textContent)))
      await close(); await open()
      await fixture('unknown'); const unknownAction = (await state()).id
      await confirm()
      check('Generic errors retain consume phase despite a pending purchase', (await state()).phase === 'consume' && (await state()).id === unknownAction)
      await close()
      check('Ambiguous consumption cannot be abandoned for a new payment', await page.evaluate(() => document.querySelector('[data-testid="opcpay-card"]').disabled))
      check('Hidden pending confirmation offers a result query', await page.evaluate(() => [...document.querySelectorAll('button')].some(button => button.textContent.trim() === '查询充值结果')))
      await fixture('paid'); await page.click('loc=role:button[name="查询充值结果"]'); await page.waitForSelector('[data-testid="opcpay-modal"]'); await idle()
      check('Unknown attempt recovers with its original action', (await fixture()).credits === 20 && (await fixture()).deductions === 2)
      await close(); await open(); await fixture('confirmed'); const confirmedAction = (await state()).id
      await confirm()
      check('Confirmed consumption keeps recovery state', (await state()).phase === 'consume' && (await state()).id === confirmedAction)
      await fixture('paid'); await page.reload(); await page.waitForSelector('[data-testid="opcpay-modal"]'); await idle()
      const final = await fixture()
      check('Confirmed recovery credits once without repeating a deduction', final.deductions === 3 && final.credits === 30)
    }
  }
  await page.screenshot({ path: output + '/checkout.png' })
} finally {
  const report = { passed: checks.length === 16 && checks.every(item => item.passed), checks, mode: 'real Vue checkout; synthetic provider, session and money', source_sha256: {} }
  for (const name of ['frontend/src/components/payment/OPCPayPanel.vue', 'frontend/e2e/opcpay-confirmation.mjs', 'frontend/e2e/opcpay-confirmation.browser.mjs']) report.source_sha256[name] = crypto.createHash('sha256').update(await fs.readFile(root + '/' + name)).digest('hex')
  await fs.writeFile(output + '/checks.json', JSON.stringify(report, null, 2))
}
if (checks.length !== 16 || !checks.every(item => item.passed)) throw new Error('Unpaid confirmation must remain recoverable')
await task.finish({ keep: [] })
