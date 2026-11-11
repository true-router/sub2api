/** Browser program. Invoke through check-preview.mjs to resolve artifact paths. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-preview-settings-and-attributes')
const page = task.page('p1')
const origin = 'http://127.0.0.1:13032'
const output = "__BOARDUI_ARTIFACT_DIRECTORY__"
await fs.mkdir(output, { recursive: true })
await page.cdp('Page.addScriptToEvaluateOnNewDocument', { source: `
  window.__previewErrors = [];
  window.__previewPending = 0;
  window.__previewLastRequest = 0;
  for (const level of ['warn', 'error']) {
    const original = console[level];
    console[level] = function(...args) {
      window.__previewErrors.push({level, message: args.map(value => value instanceof Error ? value.stack : typeof value === 'string' ? value : JSON.stringify(value)).join(' ')});
      original.apply(console, args);
    };
  }
  window.addEventListener('error', event => window.__previewErrors.push({level:'error', message:event.message}));
  window.addEventListener('unhandledrejection', event => window.__previewErrors.push({level:'error', message:String(event.reason)}));
  const send = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.send = function(...args) {
    window.__previewPending++;
    window.__previewLastRequest = performance.now();
    this.addEventListener('loadend', () => { window.__previewPending--; window.__previewLastRequest = performance.now(); }, {once:true});
    return send.apply(this,args);
  };
` })
await page.goto(origin + '/login')
await page.evaluate(() => {
  localStorage.removeItem('auth_token')
  localStorage.removeItem('auth_user')
  localStorage.setItem('sub2api_locale', 'zh')
  // Show the usage column so the preview exercises its secondary data request.
  localStorage.setItem('user-hidden-columns', '[]')
  localStorage.setItem('user-column-settings-version', '3')
  localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
})
await page.goto(origin + '/admin/dashboard')
await page.waitForSelector('#email')
await page.fill('#email', 'admin')
await page.fill('#password', 'admin')
await page.click('form button[type="submit"]')
await page.waitForURL('**/admin/dashboard')
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })

const results = [{ scenario: 'Admin signs in through the preview login form', passed: true, errors: [] }]
const idle = () => page.waitForFunction(() =>
  window.__previewPending === 0 && window.__previewLastRequest > 0 && performance.now() - window.__previewLastRequest > 400
)
await page.goto(origin + '/admin/settings')
await page.waitForSelector('main [data-boardui="content-heading"]')
await idle()
const readSettings = () => page.evaluate(() => ({
  path: location.pathname,
  errors: window.__previewErrors,
  toasts: [...document.querySelectorAll('[data-boardui="toast"][role="alert"]')].map(toast => toast.textContent.trim())
}))
const initial = await readSettings()
results.push({ scenario: 'System settings', ...initial, passed: initial.path === '/admin/settings' && !initial.errors.length && !initial.toasts.length })

const desktopGeometry = await page.evaluate(async () => {
  const rail = document.querySelector('[data-boardui="pill-tabs"]')
  const indicator = rail?.querySelector('[data-boardui="pill-tab-indicator"]')
  const panel = document.querySelector('#settings-panel')
  if (!rail || !indicator || !panel) return { passed: false }
  const originalTop = rail.getBoundingClientRect().top
  panel.scrollTop = 220
  await new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve)))
  const railRect = rail.getBoundingClientRect()
  const panelRect = panel.getBoundingClientRect()
  const opacity = Number(document.querySelector('[data-boardui="scroll-fade"] > div')?.style.opacity)
  const passed = panel.scrollTop > 0 && railRect.top === originalTop &&
    railRect.bottom <= panelRect.top && opacity === 1
  return { passed, scrollTop: panel.scrollTop, railBottom: railRect.bottom, panelTop: panelRect.top, opacity }
})
results.push({ scenario: 'PillTab stays above the desktop scroll region', ...desktopGeometry, errors: [] })

await page.press('#settings-tab-general', 'ArrowRight')
await page.waitForFunction(() => document.activeElement?.id === 'settings-tab-agreement' &&
  document.querySelector('#settings-tab-agreement')?.getAttribute('aria-selected') === 'true' &&
  document.querySelector('#settings-panel')?.getAttribute('aria-labelledby') === 'settings-tab-agreement')
results.push({ scenario: 'PillTab supports keyboard selection and resets content scroll',
  passed: await page.evaluate(() => document.querySelector('#settings-panel').scrollTop === 0), errors: [] })

await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
const waitForPill = id => page.waitForFunction(id => {
  const pill = document.querySelector('[data-boardui="pill-tab-indicator"]')?.getBoundingClientRect()
  const tab = document.getElementById(id)?.getBoundingClientRect()
  return pill && tab && Math.abs(pill.left - tab.left) < 1 && Math.abs(pill.width - tab.width) < 1
}, id)
await page.evaluate(() => {
  document.activeElement?.blur()
  document.getElementById('settings-tab-general').click()
})
await page.waitForFunction(() => document.querySelector('#settings-panel')?.getAttribute('aria-labelledby') === 'settings-tab-general')
await waitForPill('settings-tab-general')
await page.screenshot({ path: output + '/pill-tabs-desktop.png' })

for (const tab of ['agreement', 'features', 'security', 'users', 'gateway', 'payment', 'email', 'backup']) {
  await page.evaluate(tab => document.getElementById('settings-tab-' + tab).click(), tab)
  await page.waitForFunction(tab => document.getElementById('settings-tab-' + tab)?.getAttribute('aria-selected') === 'true', tab)
  await idle()
  const state = await readSettings()
  results.push({ scenario: 'Settings tab: ' + tab, ...state, passed: !state.errors.length && !state.toasts.length })
}

await page.cdp('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true })
await page.goto(origin + '/admin/settings')
await page.waitForSelector('#settings-tab-general')
await page.evaluate(() => document.getElementById('settings-tab-backup').click())
await page.waitForFunction(() => document.querySelector('#settings-tab-backup')?.getAttribute('aria-selected') === 'true')
const mobile = await page.evaluate(() => {
  const rail = document.querySelector('[data-boardui="pill-tabs"]')
  const tab = document.getElementById('settings-tab-backup')
  const rect = rail.getBoundingClientRect()
  const selected = tab.getBoundingClientRect()
  return innerWidth === 390 && document.documentElement.scrollWidth <= innerWidth &&
    rail.scrollWidth > rail.clientWidth && selected.left >= rect.left && selected.right <= rect.right + 1
})
results.push({ scenario: 'Mobile PillTab scrolls the selected label into view', passed: mobile, errors: [] })
await waitForPill('settings-tab-backup')
await page.screenshot({ path: output + '/pill-tabs-mobile.png' })
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })

await page.goto(origin + '/admin/users')
const edit = 'tr[data-row-id="2"] .sticky-col-right button:first-child'
await page.waitForSelector(edit)
await page.click(edit)
await page.waitForSelector('#edit-user-form')
await idle()
const attributes = await page.evaluate(async () => {
  const response = await fetch('/api/v1/admin/users/2/attributes')
  const { data } = await response.json()
  return { array: Array.isArray(data), errors: window.__previewErrors }
})
results.push({ scenario: 'Edit user loads attribute values', ...attributes, passed: attributes.array && attributes.errors.length === 0 })
await fs.writeFile(output + '/checks.json', JSON.stringify(results, null, 2))
console.log(JSON.stringify(results.map(({ scenario, passed, errors, toasts }) => ({
  scenario, passed, toasts, errors: errors.map(error => error.message.split('\n')[0])
}))))
if (results.some(result => !result.passed)) throw new Error('Settings and user attributes regression: ' + output + '/checks.json')
await task.finish({ keep: [] })
