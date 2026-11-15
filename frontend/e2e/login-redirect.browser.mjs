/** Run with the account preview: node e2e/check-preview.mjs login-redirect. */
/* global taskSpace */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-login-redirect')
const page = task.page('p1')
const origin = 'http://127.0.0.1:13032'
const output = "__BOARDUI_ARTIFACT_DIRECTORY__"
await fs.mkdir(output, { recursive: true })
const results = []
try {
  for (const scenario of [
    { name: 'Admin default', email: 'admin', password: 'admin', login: '/login', expected: '/admin/dashboard' },
    { name: 'Member default', email: 'member1@example.test', password: 'BoardUI-Preview-2026', login: '/login', expected: '/dashboard' },
    { name: 'Admin explicit destination', email: 'admin', password: 'admin', login: '/login?redirect=/admin/settings', expected: '/admin/settings' },
    { name: 'Member explicit destination', email: 'member1@example.test', password: 'BoardUI-Preview-2026', login: '/login?redirect=/keys', expected: '/keys' }
  ]) {
    await page.goto(origin)
    await page.evaluate(() => {
      for (const key of ['auth_token', 'auth_user', 'refresh_token', 'token_expires_at']) localStorage.removeItem(key)
      localStorage.setItem('sub2api_locale', 'zh')
    })
    await page.goto(origin + scenario.login)
    await page.waitForSelector('form button[type="submit"]:not(:disabled)')
    await page.fill('#email', scenario.email)
    await page.fill('#password', scenario.password)
    await page.press('#password', 'Enter')
    await page.waitForFunction(() => !location.pathname.startsWith('/login'))
    const actual = await page.evaluate(() => location.pathname)
    const result = { name: scenario.name, expected: scenario.expected, actual, passed: actual === scenario.expected }
    results.push(result)
    console.log(JSON.stringify(result))
  }
} catch (error) {
  results.push({ name: 'Login redirect acceptance', passed: false, error: error.message })
}
await fs.writeFile(output + '/checks.json', JSON.stringify(results, null, 2))
const failed = results.some(result => !result.passed)
await task.finish({ keep: [] })
if (failed) throw new Error('Login redirect acceptance failed: ' + output + '/checks.json')
