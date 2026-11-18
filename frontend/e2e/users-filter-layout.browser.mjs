/** Run with the account preview: node e2e/check-preview.mjs users-filters. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-users-filter-layout')
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
const viewport = (width, height, mobile = false) => page.cdp('Emulation.setDeviceMetricsOverride', { width, height, deviceScaleFactor: 1, mobile })
const shot = async name => {
  const { data } = await page.cdp('Page.captureScreenshot', { format: 'png' })
  await fs.writeFile(`${output}/${name}.png`, Buffer.from(data, 'base64'))
}

try {
  await page.cdp('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.previewRequests = [];
    const open = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function (method, url, ...args) {
      window.previewRequests.push(new URL(url, location.origin).href);
      return open.call(this, method, url, ...args);
    };
  ` })
  await page.goto(origin + '/login')
  const login = await page.fetch('/api/v1/auth/login', {
    method: 'POST', headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email: 'admin', password: 'admin' })
  })
  const { data } = typeof login.body === 'string' ? JSON.parse(login.body) : login.body
  await page.evaluate(({ access_token, user }) => {
    localStorage.setItem('auth_token', access_token)
    localStorage.setItem('auth_user', JSON.stringify(user))
    localStorage.setItem('theme', 'light')
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
    localStorage.removeItem('user-visible-filters')
    localStorage.removeItem('user-filter-values')
  }, data)
  await viewport(1440, 900)
  await page.goto(origin + '/admin/users')
  await page.waitForSelector('table[data-boardui="table"] tbody tr[data-row-id]')
  await shot('desktop-light')

  check('The toolbar shows search, recharge, role, and status filters in one row', await page.evaluate(() => {
    const bar = document.querySelector('[data-users-toolbar]')
    if (!bar) return false
    const controls = [...bar.querySelectorAll('[data-users-filter]')]
    const centers = new Set(controls.map(el => { const r = el.getBoundingClientRect(); return Math.round(r.top + r.height / 2) }))
    return controls.length >= 4 && centers.size === 1
  }))
  check('Search, filters, and toolbar buttons share a 36px height', await page.evaluate(() => {
    const bar = document.querySelector('[data-users-toolbar]')
    const boxes = [
      bar.querySelector('[data-boardui="search-field"] > div:last-child'),
      ...bar.querySelectorAll('[data-users-filter] .select-trigger'),
      ...bar.querySelectorAll('[data-boardui="button"]')
    ]
    return boxes.length >= 8 && boxes.every(el => Math.round(el.getBoundingClientRect().height) === 36)
  }))
  check('The toolbar sits inside the table card above the table', await page.evaluate(() => {
    const bar = document.querySelector('[data-users-toolbar]')
    const card = bar?.closest('.table-scroll-container')
    const table = card?.querySelector('table[data-boardui="table"]')
    return Boolean(card && table && bar.getBoundingClientRect().bottom <= table.getBoundingClientRect().top + 1)
  }))
  check('The create action is on the toolbar row, right aligned in the panel', await page.evaluate(() => {
    const create = document.querySelector('[data-testid="create-user-trigger"]').getBoundingClientRect()
    const search = document.querySelector('[data-users-toolbar] [data-boardui="search-field"]').getBoundingClientRect()
    const bar = document.querySelector('[data-users-toolbar]').getBoundingClientRect()
    return Math.abs((create.top + create.height / 2) - (search.top + search.height / 2)) < 2 && bar.right - create.right <= 16
  }))

  // Bulk bar appears for a selection and clears with cancel.
  await page.press('table[data-boardui="table"] tbody tr[data-row-id]:first-child input[data-test="select-row"]', 'Space')
  await page.waitForSelector('[data-users-bulk-bar]')
  await shot('desktop-selection')
  check('Selecting a row shows the bulk bar between toolbar and table', await page.evaluate(() => {
    const bulk = document.querySelector('[data-users-bulk-bar]').getBoundingClientRect()
    const bar = document.querySelector('[data-users-toolbar]').getBoundingClientRect()
    const table = document.querySelector('table[data-boardui="table"]').getBoundingClientRect()
    return bulk.top >= bar.bottom - 1 && bulk.bottom <= table.top + 1
  }))
  await page.evaluate(() => [...document.querySelectorAll('[data-users-bulk-bar] button')].at(-1).click())
  await page.waitForFunction(() => !document.querySelector('[data-users-bulk-bar]'))

  // Recharge filter request
  await page.click('[data-users-filter="recharge"] button')
  await page.waitForSelector('[role="option"]')
  await shot('desktop-recharge-open')
  await page.evaluate(() => [...document.querySelectorAll('[role="option"]')].find(el => el.textContent.includes('未充过值')).click())
  await page.waitForFunction(() => window.previewRequests.some(url => url.includes('/api/v1/admin/users?') && url.includes('has_recharged=false')))
  check('Choosing "never recharged" sends has_recharged=false and resets to page 1', await page.evaluate(() =>
    window.previewRequests.filter(url => url.includes('has_recharged=false')).every(url => url.includes('page=1'))))
  await page.evaluate(() => document.activeElement?.blur()); await new Promise(r => setTimeout(r, 300)); await shot('desktop-filter-active')

  check('An active filter shows a reset action on the same toolbar row', await page.evaluate(() => {
    const reset = document.querySelector('[data-users-reset-filters]')?.getBoundingClientRect()
    const search = document.querySelector('[data-users-toolbar] [data-boardui="search-field"]').getBoundingClientRect()
    return Boolean(reset) && Math.abs((reset.top + reset.height / 2) - (search.top + search.height / 2)) < 2
  }))
  await page.click('[data-users-reset-filters]')
  await page.waitForFunction(() => !document.querySelector('[data-users-reset-filters]'))
  check('Reset sends a request without has_recharged', await page.evaluate(() => {
    const last = window.previewRequests.filter(url => url.includes('/api/v1/admin/users?')).at(-1)
    return !last.includes('has_recharged')
  }))

  await page.evaluate(() => { localStorage.setItem('theme', 'dark') })
  await page.goto(origin + '/admin/users')
  await page.waitForSelector('table[data-boardui="table"] tbody tr[data-row-id]')
  await shot('desktop-dark')

  await viewport(390, 844, true)
  await page.evaluate(() => { localStorage.setItem('theme', 'light') })
  await page.goto(origin + '/admin/users')
  await page.waitForSelector('[data-users-toolbar]')
  check('No horizontal page overflow on narrow screens', await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1))
  await shot('mobile-light')
} finally {
  await fs.writeFile(`${output}/checks.json`, JSON.stringify({ source: 'e2e/users-filter-layout.browser.mjs', origin, results }, null, 2))
}
