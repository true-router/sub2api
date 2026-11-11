/** Browser program. Invoke through check-preview.mjs to resolve artifact paths. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-preview-contract-regression')
const page = task.page('p1')
const origin = 'http://127.0.0.1:13031'
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
const login = await page.fetch('/api/v1/auth/login', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify({ email: 'admin@example.test', password: 'BoardUI-Preview-2026' }) })
const response = typeof login.body === 'string' ? JSON.parse(login.body) : login.body
await page.evaluate(({token, user}) => {
  localStorage.setItem('auth_token', token)
  localStorage.setItem('auth_user', JSON.stringify(user))
  localStorage.setItem('sub2api_locale', 'zh')
  localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true')
}, { token: response.data.access_token, user: response.data.user })
const routes = ['/admin/groups', '/admin/announcements', '/admin/ops', '/admin/users', '/admin/accounts', '/admin/proxies', '/admin/redeem', '/admin/promo-codes', '/admin/usage', '/admin/audit-logs', '/admin/channels/pricing', '/admin/channels/monitor', '/admin/settings', '/admin/dashboard']
const results = []
for (const route of routes) {
  await page.goto(origin + route)
  let timeout = null
  try {
    await page.waitForFunction(() => document.querySelector('.admin-content-heading') && window.__previewPending === 0 && window.__previewLastRequest > 0 && performance.now() - window.__previewLastRequest > 400, undefined, {timeout:12000})
  } catch(error) { timeout = error.message }
  const state = await page.evaluate(() => ({ path: location.pathname, errors: window.__previewErrors || [], rendered: !!document.querySelector('.app-main') }))
  const passed = state.rendered && state.path === route && state.errors.length === 0 && !timeout
  results.push({route, passed, ...state, timeout})
  console.log(JSON.stringify(results.at(-1)))
  if (['/admin/groups','/admin/announcements','/admin/ops'].includes(route)) await page.screenshot({path: output + '/' + route.split('/').at(-1) + '.png'})
}
await fs.writeFile(output + '/routes.json', JSON.stringify(results,null,2))
if (results.some(result => !result.passed)) throw new Error('Preview route regression failed; see ' + output + '/routes.json')
await task.finish({keep:[]})
console.log('PASS: ' + results.length + ' admin routes have no console warnings or errors')
