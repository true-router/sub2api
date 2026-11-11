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
const clickTheme = async () => {
  const point = (await page.cdp('Runtime.evaluate', { expression: `(() => {
    const rect = document.querySelector('[data-testid="login-theme-toggle"]').getBoundingClientRect()
    return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
  })()`, returnByValue: true })).result.value
  await page.cdp('Input.dispatchMouseEvent', { type: 'mousePressed', ...point, button: 'left', clickCount: 1 })
  await page.cdp('Input.dispatchMouseEvent', { type: 'mouseReleased', ...point, button: 'left', clickCount: 1 })
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
  await page.waitForSelector('[data-boardui="auth-card"] form button[type="submit"]:not(:disabled)')
  check('Login uses AuthCard with labelled BoardUI inputs', await page.evaluate(() =>
    document.querySelectorAll('[data-boardui="auth-card"] [data-boardui="input"]').length === 2 &&
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
  await page.waitForSelector('[data-boardui="auth-card"] [role="alert"]')
  check('Failed login keeps the user on the form and displays the API error', await page.evaluate(() =>
    location.pathname === '/login' && document.querySelector('[data-boardui="auth-card"] [role="alert"]').textContent.includes('local preview')
  ))

  await page.screenshot({ path: output + '/login-error.png' })
  await page.goto(origin + '/login?redirect=/admin/settings')
  await page.waitForSelector('form button[type="submit"]:not(:disabled)')
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true })
  check('Narrow login has no horizontal overflow', await page.evaluate(() => innerWidth === 390 && document.documentElement.scrollWidth <= innerWidth))
  await page.screenshot({ path: output + '/login-mobile.png' })

  await desktop()
  await clickTheme()
  check('Login theme toggle enables dark semantic tokens', await page.evaluate(() => {
    const card = document.querySelector('[data-boardui="auth-card"]')
    return document.documentElement.classList.contains('dark') &&
      getComputedStyle(card).backgroundColor !== 'rgb(255, 255, 255)'
  }))
  await desktop()
  await page.waitForFunction(() => {
    const surface = document.querySelector('[data-boardui="input"] > div')
    const token = getComputedStyle(document.documentElement).getPropertyValue('--boardui-background-primary-default').trim().split(/\s+/).join(', ')
    return getComputedStyle(surface).backgroundColor === 'rgb(' + token + ')'
  })
  await page.screenshot({ path: output + '/login-dark.png' })
  await clickTheme()
  await desktop()
  await page.goto(origin + '/login?redirect=/admin/settings')
  await page.waitForSelector('form button[type="submit"]:not(:disabled)')
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
