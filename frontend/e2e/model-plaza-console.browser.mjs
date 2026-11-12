const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-model-plaza-console')
const page = task.page('p1')
const output = "__ARTIFACT_DIRECTORY__"
const origin = 'http://127.0.0.1:13037'
const checks = []
const check = (name, passed) => { checks.push({ name, passed: Boolean(passed) }); if (!passed) throw new Error(name); console.log('PASS ' + name) }
const fixture = query => fetch('http://127.0.0.1:18087/api/v1/__model-plaza-fixture?' + query)
async function login(role) {
  await page.goto(origin + '/login')
  await page.evaluate(role => {
    const admin = role === 'admin'
    localStorage.setItem('auth_token', admin ? 'local-admin-preview' : 'local-member-preview')
    localStorage.setItem('auth_user', JSON.stringify({ id: admin ? 1 : 2, role, email: admin ? 'admin@example.test' : 'member1@example.test', balance: 100 }))
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('user_guide_2_user_v4_interactive', 'true')
    localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
  }, role)
}
try {
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1050, deviceScaleFactor: 1, mobile: false })
  await fixture('enabled=true')
  await login('user')
  await page.goto(origin + '/dashboard')
  await page.waitForSelector('.sidebar a[href="/account/model-plaza"]')
  check('User sidebar exposes model plaza', true)
  await page.click('.sidebar a[href="/account/model-plaza"]')
  await page.waitForSelector('[data-model="console-e2e-model"]')
  check('Catalog opens inside account console with active menu and one heading', await page.evaluate(() => location.pathname === '/account/model-plaza' && !!document.querySelector('.sidebar a[href="/account/model-plaza"][aria-current="page"]') && document.documentElement.dataset.adminUi === 'boardui' && document.querySelectorAll('main h1').length === 1))
  await page.fill('input[type="search"]', 'no-such-model')
  check('Embedded catalog retains search', await page.evaluate(() => !document.querySelector('[data-model="console-e2e-model"]')))
  await page.goto(origin + '/account/model-plaza')
  await page.waitForSelector('[data-model="console-e2e-model"]')
  check('Direct navigation retains console layout', await page.evaluate(() => !!document.querySelector('.sidebar')))
  await login('admin')
  await page.goto(origin + '/account/model-plaza')
  await page.waitForSelector('[data-model="console-e2e-model"]')
  check('Administrator personal menu exposes model plaza', await page.evaluate(() => !!document.querySelector('.sidebar a[href="/account/model-plaza"]')))
  await fixture('enabled=false')
  await login('user')
  await page.goto(origin + '/account/model-plaza')
  await page.waitForURL('**/dashboard')
  check('Disabled feature hides menu and blocks direct navigation', await page.evaluate(() => !document.querySelector('.sidebar a[href="/account/model-plaza"]')))
  await page.evaluate(() => { localStorage.removeItem('auth_token'); localStorage.removeItem('auth_user') })
  await fixture('enabled=true')
  await page.goto(origin + '/account/model-plaza')
  await page.waitForURL('**/login?redirect=**')
  check('Console route requires authentication and retains return path', await page.evaluate(() => new URLSearchParams(location.search).get('redirect') === '/account/model-plaza'))
  await page.goto(origin + '/model-plaza')
  await page.waitForSelector('[data-model="console-e2e-model"]')
  check('Anonymous public catalog retains standalone layout', await page.evaluate(() => !document.querySelector('.sidebar') && document.querySelectorAll('main h1').length === 1))
  await fixture('enabled=true&require_auth=true')
  await page.goto(origin + '/model-plaza')
  await page.waitForURL('**/login?redirect=**')
  check('Public catalog retains optional authentication requirement', true)
  await task.finish({ keep: [] })
} finally {
  await fs.writeFile(output + '/checks.json', JSON.stringify({ generated_at: new Date().toISOString(), scope: 'Actual frontend with deterministic API fixtures; not production backend acceptance', checks }, null, 2))
}
