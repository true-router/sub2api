/* global taskSpace */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-usage-session-acceptance')
const page = task.page('p1')
const origin = "__ORIGIN__"
const output = "__OUTPUT__"
const checks = []
function check(name, passed) {
  checks.push({ name, passed: Boolean(passed) })
  console.log(JSON.stringify(checks.at(-1)))
  if (!passed) throw new Error(name)
}
try {
  await page.cdp('Page.addScriptToEvaluateOnNewDocument', { source: `
    window.usageRequests=[];
    const open=XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open=function(method,url,...args){window.usageRequests.push(String(url));return open.call(this,method,url,...args)};
    Object.defineProperty(navigator,'clipboard',{value:{writeText:async value=>{window.copiedSession=value}}});
  ` })
  await page.goto(origin + '/login')
  await page.evaluate(() => {
    localStorage.setItem('auth_token', 'local-preview-only')
    localStorage.setItem('auth_user', JSON.stringify({ id: 1, email: 'admin@example.test', role: 'admin', status: 'active' }))
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
    localStorage.setItem('theme', 'light')
  })
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1100, deviceScaleFactor: 1, mobile: false })
  await page.goto(origin + '/admin/usage')
  const row = 'table tbody tr[data-row-id]'
  const session = '[data-testid="usage-session-filter"]'
  await page.waitForSelector(row)
  await fs.writeFile(output+'/initial-snapshot.txt',await page.snapshot({ scope: 'full_page' }))
  check('Session IDs are visible by default', await page.evaluate(() => document.querySelectorAll('[data-testid="usage-session-link"]').length === 5))
  check('Requests without a session show an explicit absent label', await page.evaluate(() => document.querySelector('[data-testid="usage-session-missing"]')?.textContent.includes('未提供')))
  await page.click('[data-testid="usage-session-copy"] >> nth=0')
  check('Copy uses the complete session ID without applying a filter', await page.evaluate(() => window.copiedSession === 'session-shared' && !document.querySelector('[data-testid="usage-session-filter"]').value))
  await page.click('[data-testid="usage-session-link"] >> nth=0')
  await page.waitForFunction(() => document.querySelectorAll('table tbody tr[data-row-id]').length===1)
  check('Session click scopes records to the clicked user', await page.evaluate(() => {
    const urls=window.usageRequests.map(u=>new URL(u,location.origin))
    return urls.some(u=>u.pathname==='/api/v1/admin/usage' && u.searchParams.get('session_id')==='session-shared' && u.searchParams.get('user_id')==='2')
  }))
  check('The selected session remains visible in the filter', await page.evaluate(() => document.querySelector('[data-testid="usage-session-filter"]').value==='session-shared'))
  check('Cleanup is disabled while the displayed list is session-scoped', await page.evaluate(() => document.querySelector('[data-testid="usage-cleanup"]').disabled))
  await page.fill(session, 'session-other')
  await page.press(session, 'Enter')
  await page.waitForSelector('[data-testid="usage-session-clear"]')
  await page.waitForFunction(() => document.querySelectorAll('table tbody tr[data-row-id]').length===0)
  check('Manual session changes retain user scope and exclude unrelated users', true)
  await page.click('[data-testid="usage-session-clear"]')
  await page.waitForFunction(() => document.querySelectorAll('table tbody tr[data-row-id]').length===1)
  check('Clearing only the session retains the selected user', await page.evaluate(() => !document.querySelector('[data-testid="usage-session-filter"]').value))
  await page.click('[data-testid="usage-reset"]')
  await page.waitForFunction(() => document.querySelectorAll('table tbody tr[data-row-id]').length===6)
  check('Reset clears the session and user filters', true)
  await page.fill(session, 'session-shared')
  await page.press(session, 'Enter')
  await page.waitForFunction(() => document.querySelectorAll('table tbody tr[data-row-id]').length===3)
  check('Manual exact filtering excludes prefixes and missing IDs', true)
  await page.click('[data-testid="usage-session-link"] >> nth=1')
  await page.waitForFunction(() => document.querySelectorAll('table tbody tr[data-row-id]').length===2)
  await page.click('[data-sort-key="created_at"]')
  await page.waitForFunction(() => {
    const urls=window.usageRequests.map(u=>new URL(u,location.origin))
    return urls.some(u=>u.pathname==='/api/v1/admin/usage' && u.searchParams.get('session_id')==='session-shared' && u.searchParams.get('sort_order')==='asc')
  })
  check('Time sorting retains session scope', true)
  const downloaded=page.waitForEvent('download',{timeout:30000})
  await page.click('[data-testid="usage-export"]')
  const download=await downloaded
  await download.saveAs(output+'/session.xlsx')
  check('Filtered export requests an exact total with user and session scope',await page.evaluate(()=>window.usageRequests.map(u=>new URL(u,location.origin)).some(u=>u.pathname==='/api/v1/admin/usage' && u.searchParams.get('exact_total')==='true' && u.searchParams.get('session_id')==='session-shared' && u.searchParams.get('user_id')==='1')))
  await page.cdp('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true})
  await page.waitForSelector(session)
  check('Session filter remains usable on narrow screens',await page.evaluate(()=>document.querySelector('[data-testid="usage-session-filter"]').getBoundingClientRect().width>100))
  await fs.writeFile(output+'/snapshot.txt',await page.snapshot({scope:'full_page'}))
  await fs.writeFile(output+'/receipt.json',JSON.stringify({checks,backend:'Real HTTP handler/service/repository with disposable PostgreSQL',clipboard:'Instrumented clipboard writeText; system clipboard unchanged',production_modified:false},null,2))
  await task.finish({keep:[]})
} catch(error) {
  await fs.writeFile(output+'/receipt.json',JSON.stringify({checks,error:String(error)},null,2))
  console.log(await page.snapshot({scope:'full_page'}))
  throw error
}
