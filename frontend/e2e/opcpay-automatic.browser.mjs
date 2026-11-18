const fs = await import('node:fs/promises'), crypto = await import('node:crypto')
const task = await taskSpace('sub2api-opcpay-automatic'), page = task.page('p1')
console.log({ taskSpaceId: task.spaceId })
const output = "__OUTPUT__", root = "__ROOT__"
const checks = []
const check = (name, passed) => { checks.push({ name, passed }); if (!passed) throw Error(name); console.log('PASS ' + name) }
const fixture = async paid => {
  const result = await page.fetch('/api/v1/payment/opcpay/__fixture', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ paid }) })
  return (typeof result.body === 'string' ? JSON.parse(result.body) : result.body).data
}
const idle = () => page.waitForFunction(() => document.querySelector('[data-testid="opcpay-panel"] button') && !document.querySelector('[data-testid="opcpay-panel"] button').disabled)
const open = async () => {
  await page.click('[data-testid="opcpay-card"]')
  await page.click('loc=role:button[name="支付宝购买 · CNY 9.90"]')
  await page.waitForSelector('img[alt="支付宝购买入口二维码"]'); await idle()
  return (await fixture()).operations.at(-1).business_id
}
const close = async () => {
  await page.click('.modal-header button[aria-label="关闭"]')
  await page.waitForSelector('[data-testid="opcpay-modal"]', { state: 'hidden' }); await idle()
}
await page.goto('http://127.0.0.1:13044/login')
await page.evaluate(() => {
  localStorage.setItem('auth_token', 'local-member-preview')
  localStorage.setItem('auth_user', JSON.stringify({ id: 2, role: 'user', email: 'member1@example.test', balance: 100 }))
  localStorage.setItem('sub2api_locale', 'zh'); localStorage.setItem('theme', 'light')
  localStorage.setItem('user_guide_2_user_v4_interactive', 'true')
})
await page.goto('http://127.0.0.1:13044/purchase')
await page.waitForSelector('[data-testid="opcpay-card"]')
console.log(await page.snapshot())
try {
  const first = await open()
  check('Checkout explains automatic delivery', await page.evaluate(() => document.querySelector('[data-testid="opcpay-modal"]').innerText.includes('自动到账')))
  check('Checkout has no payment-confirmation button', !(await page.evaluate(() => [...document.querySelectorAll('button')].some(button => button.textContent.includes('我已支付')))))
  await fixture(first)
  await page.waitForFunction(() => document.querySelector('[data-testid="opcpay-modal"]')?.innerText.includes('充值成功，已到账 10 积分。'), undefined, { timeout: 10000 })
  check('The page displays automatic delivery without a confirmation click', true)
  await close()
  const second = await open()
  await close()
  check('Closing checkout explains payment still settles', await page.evaluate(() => document.querySelector('[data-testid="opcpay-panel"]').innerText.includes('付款成功后仍会自动到账')))
  await fixture(second)
  await page.waitForFunction(() => document.querySelector('[data-testid="opcpay-panel"]')?.innerText.includes('充值成功，已到账 10 积分。'), undefined, { timeout: 10000 })
  check('A closed checkout reports later delivery without reopening the dialog', !(await page.evaluate(() => !!document.querySelector('[data-testid="opcpay-modal"]'))))
  const final = await fixture()
  check('Two purchases receive exactly two credits', final.credits === 20 && final.deductions === 2)
  await page.screenshot({ path: output + '/automatic-credit.png' })
} finally {
  const report = { passed: checks.length === 6 && checks.every(item => item.passed), checks, mode: 'real Vue; simulated merchant, session and money', source_sha256: {} }
  for (const name of ['frontend/src/components/payment/OPCPayPanel.vue', 'frontend/src/api/opcpay.ts', 'frontend/e2e/opcpay-automatic.mjs', 'frontend/e2e/opcpay-automatic.browser.mjs']) report.source_sha256[name] = crypto.createHash('sha256').update(await fs.readFile(root + '/' + name)).digest('hex')
  await fs.writeFile(output + '/checks.json', JSON.stringify(report, null, 2))
}
// The caller finishes this task space after inspecting the receipt.
