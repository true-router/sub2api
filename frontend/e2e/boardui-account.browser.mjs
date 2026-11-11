/** Browser program. Invoke through check-preview.mjs to resolve artifact paths. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-boardui-account-acceptance')
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
const results = []
const check = (name, passed, details = {}) => {
  results.push({ name, passed, ...details })
  console.log(JSON.stringify(results.at(-1)))
}
async function signIn(email, role) {
  await page.goto(origin + '/login')
  const login = await page.fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email, password: role === 'admin' ? 'admin' : 'BoardUI-Preview-2026' })
  })
  const { data } = typeof login.body === 'string' ? JSON.parse(login.body) : login.body
  await page.evaluate(({ access_token, user, role }) => {
    localStorage.setItem('auth_token', access_token)
    localStorage.setItem('auth_user', JSON.stringify(user))
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem('theme', 'light')
    localStorage.setItem(role + '_guide_1_' + role + '_v4_interactive', 'true')
  }, { ...data, role })
}
await signIn('admin', 'admin')
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false })
const routes = [
  ['/keys', '[data-boardui="search-field"], [data-tour="keys-create-btn"]'],
  ['/batch-image', '[data-boardui="search-field"]'],
  ['/usage', '[data-boardui="segmented-control"]'],
  ['/available-channels', '[data-boardui="search-field"]'],
  ['/monitor', '[data-boardui="segmented-control"]'],
  ['/subscriptions', '.card'],
  ['/purchase', '[data-boardui="segmented-control"]'],
  ['/orders', '[data-boardui="button"]'],
  ['/redeem', '[data-boardui="input"]'],
  ['/affiliate', '[data-boardui="button"]'],
  ['/profile', '[data-boardui="badge"]'],
  ['/settings/mcp', '[data-boardui="input"]'],
  ['/custom/preview', 'section, .card']
]
for (const [route, selector] of routes) {
  await page.goto(origin + route)
  let timeout = null
  try {
    await page.waitForFunction(() => document.querySelector('main [data-boardui="content-heading"]') && window.__previewPending === 0 && window.__previewLastRequest > 0 && performance.now() - window.__previewLastRequest > 400, undefined, { timeout: 9000 })
  } catch (error) { timeout = error.message }
  const state = await page.evaluate(selector => ({
    path: location.pathname,
    workspace: document.documentElement.dataset.adminUi === 'boardui',
    sidebar: Boolean(document.querySelector('.sidebar [data-testid="sidebar-available-balance"]')),
    breadcrumb: Boolean(document.querySelector('main [data-boardui="breadcrumb"] [aria-current="page"]')),
    control: [...(document.querySelector('main')?.querySelectorAll(selector) ?? [])].some(el => !el.closest('.admin-content-heading')),
    errors: window.__previewErrors,
  }), selector)
  check(route, !timeout && state.path === route && state.workspace && state.sidebar && state.breadcrumb && state.control && !state.errors.length, { ...state, timeout })
  if (['/keys', '/redeem', '/profile', '/monitor'].includes(route)) await page.screenshot({ path: output + route + '.png' })
}

try {
  await page.goto(origin + '/profile')
  await page.waitForFunction(() => {
    const active = document.querySelector('.sidebar [aria-current="page"][href="/profile"]')
    const nav = active?.closest('nav')
    return nav && active.getBoundingClientRect().bottom <= nav.getBoundingClientRect().bottom
  })
  check('Direct account navigation reveals the selected sidebar link', true)

  await page.goto(origin + '/monitor')
  await page.waitForSelector('main [data-boardui="segmented-control"]')
  await page.press('main [data-boardui="segmented-control"] [aria-checked="true"]', 'ArrowRight')
  check('Monitor range switches with the keyboard', await page.evaluate(() =>
    document.querySelector('main [data-boardui="segmented-control"] [aria-checked="true"]')?.textContent.includes('15')
  ))

  await page.goto(origin + '/keys')
  await page.waitForSelector('[data-tour="keys-create-btn"]')
  await page.click('[data-tour="keys-create-btn"]')
  await page.waitForSelector('#key-form [data-boardui="input"] input')
  await page.fill('#key-form [data-boardui="input"] input', 'Preview key')
  check('Key form uses BoardUI input', await page.evaluate(() => document.querySelector('#key-form input[data-tour="key-form-name"]').value === 'Preview key'))
  await page.keyboard.press('Escape')
  await page.waitForSelector('#key-form', { state: 'hidden' })
  await page.keyboard.press('ControlOrMeta+k')
  await page.waitForSelector('[data-boardui="command-palette"] input')
  await page.fill('[data-boardui="command-palette"] input', 'MCP')
  await page.press('[data-boardui="command-palette"] input', 'Enter')
  await page.waitForURL('**/settings/mcp')
  check('Command palette opens an account route', true)

  await page.goto(origin + '/redeem')
  await page.waitForSelector('form:has(#code) button[type="submit"]')
  const redeemButton = 'form:has(#code) button[type="submit"]'
  const initiallyDisabled = await page.evaluate(selector => document.querySelector(selector).disabled, redeemButton)
  await page.fill('#code', 'PREVIEW')
  check('Redeem button follows the form state', initiallyDisabled && await page.evaluate(selector => !document.querySelector(selector).disabled, redeemButton))
} catch (error) { check('Account interactions', false, { error: error.message }) }

try {
  await signIn('member1@example.test', 'user')
  await page.goto(origin + '/profile')
  await page.waitForSelector('#old_password')
  check('Regular user has the account workspace without admin links', await page.evaluate(() =>
    document.documentElement.dataset.adminUi === 'boardui' &&
    !!document.querySelector('main [data-boardui="breadcrumb"] a[href="/dashboard"]') &&
    !document.querySelector('.sidebar a[href^="/admin"]')
  ))
  await page.goto(origin + '/admin/users')
  await page.waitForURL('**/dashboard')
  check('Regular user still cannot access admin routes', await page.evaluate(() => document.documentElement.dataset.adminUi === 'boardui' && !document.querySelector('.sidebar a[href^="/admin"]')))
} catch (error) { check('Regular user workspace and permissions', false, { error: error.message }) }

await fs.writeFile(output + '/checks.json', JSON.stringify(results, null, 2))
if (results.some(result => !result.passed)) throw new Error('Account workspace acceptance failed: ' + output + '/checks.json')
await task.finish({ keep: [] })
