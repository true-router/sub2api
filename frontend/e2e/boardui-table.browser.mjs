/** Run with the account preview: node e2e/check-preview.mjs table. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-boardui-table-acceptance')
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
    localStorage.setItem('table-page-size', '20')
    localStorage.removeItem('admin-users-table-sort')
    localStorage.removeItem('user-hidden-columns')
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
  }, data)
  await desktop()
  await page.goto(origin + '/admin/users')
  const table = 'table[data-boardui="table"]'
  const row = table + ' tbody tr[data-row-id]'
  await page.waitForSelector(row)
  check('The shared data table renders native cells with compact dimensions', await page.evaluate(() => {
    const table = document.querySelector('table[data-boardui="table"]')
    const row = table.querySelector('tbody tr[data-row-id]')
    return table.dataset.size === 'sm' && row.getBoundingClientRect().height >= 48 &&
      getComputedStyle(row.cells[0]).paddingTop === '6px' &&
      getComputedStyle(row.cells[0]).borderBottomWidth === '1px'
  }))

  const sort = table + ' [data-sort-key="email"]'
  await page.press(sort, 'Enter')
  await page.waitForFunction(() => document.querySelector('[data-sort-key="email"]').closest('th').getAttribute('aria-sort') === 'ascending' &&
    window.previewRequests.some(url => url.includes('/api/v1/admin/users?') && url.includes('sort_by=email') && url.includes('sort_order=asc')))
  await page.press(sort, 'Space')
  await page.waitForFunction(() => document.querySelector('[data-sort-key="email"]').closest('th').getAttribute('aria-sort') === 'descending' &&
    window.previewRequests.some(url => url.includes('/api/v1/admin/users?') && url.includes('sort_by=email') && url.includes('sort_order=desc')))
  check('Enter and Space sort the column and send the correct server query', true)

  await page.press(row + ':first-child input[data-test="select-row"]', 'Space')
  check('Selecting a row paints it and marks the header checkbox indeterminate', await page.evaluate(() =>
    !!document.querySelector('table[data-boardui="table"] tr[data-selected]') &&
    document.querySelector('input[data-test="select-all"]').indeterminate
  ))
  await page.press('input[data-test="select-all"]', 'Space')
  check('The header checkbox selects all rows on the current page', await page.evaluate(() =>
    [...document.querySelectorAll('table[data-boardui="table"] input[data-test="select-row"]')].every(input => input.checked)
  ))
  await page.press('input[data-test="select-all"]', 'Space')

  const first = await page.evaluate(() => document.querySelector('table[data-boardui="table"] tbody tr[data-row-id]').dataset.rowId)
  await page.click('[data-boardui="pagination"] button:last-child')
  await page.waitForFunction(first => {
    const row = document.querySelector('table[data-boardui="table"] tbody tr[data-row-id]')
    return row && row.dataset.rowId !== first
  }, first)
  check('Pagination loads a different set of rows', true)
  await desktop()
  await page.goto(origin + '/admin/users')
  await page.waitForSelector(row)
  await page.screenshot({ path: output + '/users.png' })
  await page.evaluate(() => {
    localStorage.setItem('user-hidden-columns', '[]')
    localStorage.setItem('user-column-settings-version', '3')
  })
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1100, height: 720, deviceScaleFactor: 1, mobile: false })
  await page.goto(origin + '/admin/users')
  await page.waitForSelector(row)
  const sticky = await page.evaluate(() => {
    const scroller = document.querySelector('.table-wrapper')
    const header = scroller.querySelector('thead')
    const left = scroller.querySelector('tbody .sticky-col-left')
    const right = scroller.querySelector('tbody .sticky-col-right')
    const viewport = [scroller.clientWidth, scroller.clientHeight]
    const content = [scroller.scrollWidth, scroller.scrollHeight]
    if (!left || !right) return { viewport, content, left: !!left, right: !!right, passed: false }
    // Scroll past the selection column before comparing the pinned positions.
    scroller.scrollTo(64, 0)
    const before = [header.getBoundingClientRect().top, left.getBoundingClientRect().left, right.getBoundingClientRect().right]
    scroller.scrollTo(144, 80)
    const after = [header.getBoundingClientRect().top, left.getBoundingClientRect().left, right.getBoundingClientRect().right]
    const offset = [scroller.scrollLeft, scroller.scrollTop]
    const passed = offset[0] > 64 && offset[1] > 0 && before.every((value, index) => Math.abs(value - after[index]) < 2)
    scroller.scrollTo(0, 0)
    return { viewport, content, offset, before, after, passed }
  })
  await fs.writeFile(output + '/sticky.json', JSON.stringify(sticky, null, 2))
  check('Fixed header and columns retain their position when the table scrolls', sticky.passed)
  await desktop()
  await page.goto(origin + '/keys')
  await page.waitForSelector(table + ' [data-sort-key="name"]')
  check('API keys use the same Table primitive', true)
  await page.goto(origin + '/available-channels')
  await page.waitForSelector('table[data-boardui="table"][data-testid="desktop-channels"] th')
  check('Available channels use Table without the data-table adapter', true)

  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true })
  await page.goto(origin + '/admin/users')
  await page.waitForSelector('input[data-test="select-all-mobile"]')
  check('Narrow screens retain the selectable card list', await page.evaluate(() => !document.querySelector('table[data-boardui="table"]')))
} catch (error) {
  results.push({ name: 'Table scenario', passed: false, error: error.message })
}
await fs.writeFile(output + '/checks.json', JSON.stringify(results, null, 2))
if (results.some(result => !result.passed)) throw new Error('Table acceptance failed: ' + output + '/checks.json')
await task.finish({ keep: [] })
