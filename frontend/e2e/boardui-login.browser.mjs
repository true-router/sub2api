/** Run against the account preview: node e2e/check-preview.mjs login. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-boardui-login-acceptance')
const page = task.page('p1')
const origin = 'http://127.0.0.1:13032'
const output = "__BOARDUI_ARTIFACT_DIRECTORY__"
await fs.mkdir(output, { recursive: true })
const results = []
function check(name, passed) {
  results.push({ name, passed: Boolean(passed) })
  console.log(JSON.stringify(results.at(-1)))
  if (!passed) throw new Error(name)
}
const desktop = () => page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
try {
  await page.goto(origin + '/login')
  await page.evaluate(() => {
    localStorage.removeItem('auth_token')
    localStorage.removeItem('auth_user')
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('theme', 'light')
    localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
  })
  await desktop()
  await page.goto(origin + '/login?redirect=/admin/settings')
  await page.waitForSelector('[data-testid="auth-shell"] form button[type="submit"]:not(:disabled)')
  check('Login uses AuthShell with labelled BoardUI inputs', await page.evaluate(() =>
    document.querySelectorAll('[data-testid="auth-shell"] [data-boardui="input"]').length === 2 &&
    document.querySelector('h1')?.textContent.length > 0 &&
    document.querySelector('label[for="email"]') &&
    document.querySelector('label[for="password"]') &&
    !document.documentElement.hasAttribute('data-admin-ui')
  ))
  check('Public registration remains available', await page.evaluate(() => !!document.querySelector('a[href="/register"]')))

  await page.fill('#email', 'admin')
  await page.fill('#password', 'wrong')
  await page.click('[aria-label="显示密码"]')
  check('Password visibility toggles without submitting the form', await page.evaluate(() =>
    document.querySelector('#password').type === 'text' && location.pathname === '/login'
  ))
  await page.click('[aria-label="隐藏密码"]')
  check('Password returns to the masked state', await page.evaluate(() => document.querySelector('#password').type === 'password'))
  await page.click('form button[type="submit"]')
  await page.waitForSelector('[data-testid="auth-shell"] [role="alert"]')
  check('Failed login keeps the user on the form and displays the API error', await page.evaluate(() =>
    location.pathname === '/login' && document.querySelector('[data-testid="auth-shell"] [role="alert"]').textContent.includes('local preview')
  ))

  await page.screenshot({ path: output + '/login-error.png' })
  await page.goto(origin + '/login?redirect=/admin/settings')
  await page.waitForSelector('form button[type="submit"]:not(:disabled)')
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true })
  check('Narrow login has no horizontal or vertical overflow', await page.evaluate(() => innerWidth === 390 && document.documentElement.scrollWidth <= innerWidth && document.documentElement.scrollHeight <= innerHeight))
  await page.screenshot({ path: output + '/login-mobile.png' })

  await desktop()
  await page.goto(origin + '/login?redirect=/admin/settings')
  await page.waitForSelector('form button[type="submit"]:not(:disabled)')
  check('Desktop login never scrolls vertically', await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight && document.querySelector('.pane').scrollHeight <= document.querySelector('.pane').clientHeight))
  await page.screenshot({ path: output + '/login-desktop.png' })

  await page.fill('#email', 'admin')
  await page.fill('#password', 'admin')
  await page.press('#password', 'Enter')
  await page.waitForURL('**/admin/settings')
  await page.waitForSelector('[data-boardui="pill-tabs"]')
  check('Admin signs in with Enter and reaches the requested page', await page.evaluate(() =>
    document.documentElement.dataset.adminUi === 'boardui' && document.querySelector('[data-boardui="pill-tabs"]')
  ))
} catch (error) {
  results.push({ name: 'Login acceptance', passed: false, error: error.message })
}
await fs.writeFile(output + '/checks.json', JSON.stringify(results, null, 2))
if (results.some(result => !result.passed)) throw new Error('Login acceptance failed: ' + output + '/checks.json')
await task.finish({ keep: [] })
