/** Run from frontend: node e2e/admin-boardui.mjs. Requires pnpm and ego-browser. */
import { createAdminFixtureServer, PREVIEW_USER as user } from './fixtures/admin-api.mjs'
import { spawn } from 'node:child_process'
import { mkdir, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const artifacts = resolve(root, '../output/admin-boardui')
await mkdir(artifacts, { recursive: true })
const requests = []
const fixture = createAdminFixtureServer({ requests })
await new Promise(resolve => fixture.listen(18081, '127.0.0.1', resolve))
const vite = spawn('pnpm', ['exec', 'vite', '--host', '127.0.0.1', '--port', '13031', '--strictPort'], { cwd: root, env: { ...process.env, VITE_DEV_PROXY_TARGET: 'http://127.0.0.1:18081' }, stdio: ['ignore', 'pipe', 'pipe'] })
let serverLog = ''
vite.stdout.on('data', chunk => { serverLog += chunk })
vite.stderr.on('data', chunk => { serverLog += chunk })
const stop = () => { vite.kill('SIGTERM'); fixture.close() }
process.on('SIGINT', stop)
process.on('SIGTERM', stop)
try {
  let ready = false
  for (let attempt = 0; attempt < 120; attempt++) {
    if (vite.exitCode !== null) throw new Error(`Vite exited: ${serverLog}`)
    try { if ((await fetch('http://127.0.0.1:13031')).ok) { ready = true; break } } catch { /* wait for Vite */ }
    await new Promise(resolve => setTimeout(resolve, 250))
  }
  if (!ready) throw new Error(`Vite did not start: ${serverLog}`)
  const browserScript = `
const fs = await import('node:fs/promises');
const task = await taskSpace('sub2api-boardui-acceptance');
const page = task.page('p1');
const readShell = async (fn) => {
  const { result, exceptionDetails } = await page.cdp('Runtime.evaluate', { expression: '(' + fn.toString() + ')()', returnByValue: true });
  if (exceptionDetails) throw new Error(exceptionDetails.text);
  return result.value;
};
const waitShell = async (fn, name) => {
  for (let attempt = 0; attempt < 200; attempt++) {
    if (await readShell(fn)) return;
    await new Promise(resolve => setTimeout(resolve, 50));
  }
  throw new Error((name || 'Responsive shell condition timed out') + ': ' + JSON.stringify(await readShell(() => ({ width: innerWidth, dark: document.documentElement.className, table: document.querySelector('.table-scroll-container') && getComputedStyle(document.querySelector('.table-scroll-container')).backgroundColor, sidebar: document.querySelector('.sidebar')?.className }))));
};
const checks = [];
// Native Ego actions restore the host viewport. Use CDP pointer input for the
// responsive shell checks so the explicit emulated viewport remains active.
const clickShell = async (selector) => {
  const { result } = await page.cdp('Runtime.evaluate', {
    expression: '(() => { const e = document.querySelector(' + JSON.stringify(selector) + '); const b = e.getBoundingClientRect(); return { x: b.left + b.width / 2, y: b.top + b.height / 2 }; })()',
    returnByValue: true
  });
  const point = result.value;
  await page.cdp('Input.dispatchMouseEvent', { type: 'mousePressed', ...point, button: 'left', clickCount: 1 });
  await page.cdp('Input.dispatchMouseEvent', { type: 'mouseReleased', ...point, button: 'left', clickCount: 1 });
};
const check = (condition, name) => { if (!condition) throw new Error(name); checks.push(name); console.log('PASS: ' + name); };
await page.goto('http://127.0.0.1:13031/login');
await page.evaluate((user) => {
  localStorage.setItem('auth_token', 'local-e2e-fixture');
  localStorage.setItem('auth_user', JSON.stringify(user));
  localStorage.setItem('sub2api_locale', 'en');
  localStorage.setItem('theme', 'light');
  localStorage.setItem('admin_guide_1_admin_v4_interactive', 'true');
}, ${JSON.stringify(user)});
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
await page.goto('http://127.0.0.1:13031/admin/dashboard');
await page.waitForSelector('[data-testid="admin-metric-card"]');
check(await page.evaluate(() => document.documentElement.dataset.adminUi === 'boardui'), 'Admin theme activates');
check(await page.evaluate(() => document.querySelectorAll('[data-testid="admin-metric-card"]').length === 8), 'Eight metric cards render');
check(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'Desktop has no horizontal overflow');
await page.screenshot({ path: ${JSON.stringify(resolve(artifacts, 'dashboard-light.png'))} });
console.log('Dashboard ready');
await page.click('loc=css:.sidebar a[href="/admin/users"]');
await page.waitForSelector('input[placeholder="Search by email, username, notes, or API key"]');
await page.waitForFunction(() => document.body.textContent.includes('member1@example.test'));
await page.fill('input[placeholder="Search by email, username, notes, or API key"]', 'member28');
await page.waitForFunction(() => document.body.textContent.includes('member28@example.test') && !document.body.textContent.includes('member1@example.test'));
check(true, 'User search reaches API and filters rows');
await page.click('loc=role:button[name="Create User"]');
await page.waitForSelector('[role="dialog"]');
check(await page.evaluate(() => getComputedStyle(document.querySelector('.modal-content')).borderRadius === '24px'), 'Teleported dialog uses admin theme');
await page.press('input[type="email"]', 'Escape');
await page.waitForSelector('[role="dialog"]', { state: 'hidden' });
check(true, 'Escape closes dialog');
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
await waitShell(() => matchMedia('(min-width: 1024px)').matches, 'Desktop table layout');
await clickShell('[data-testid="sidebar-theme-dark"]');
check(await readShell(() => document.documentElement.classList.contains('dark')), 'Theme toggle enables dark mode');
await waitShell(() => { const expected = 'rgb(' + getComputedStyle(document.documentElement).getPropertyValue('--boardui-background-secondary-default').trim().split(' ').join(', ') + ')'; return getComputedStyle(document.querySelector('.table-scroll-container')).backgroundColor === expected; }, 'Dark table surface');
await page.screenshot({ path: ${JSON.stringify(resolve(artifacts, 'users-dark.png'))} });
await clickShell('[data-testid="sidebar-collapse"]');
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
await waitShell(() => document.querySelector('.sidebar').getBoundingClientRect().width < 100, 'Sidebar width after collapse');
check(true, 'Sidebar collapses');
await clickShell('[data-testid="sidebar-collapse"]');
await page.cdp('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
await clickShell('[data-testid="mobile-menu-toggle"]');
await waitShell(() => document.querySelector('.sidebar').getBoundingClientRect().left >= 0, 'Mobile drawer opens');
await page.screenshot({ path: ${JSON.stringify(resolve(artifacts, 'mobile-navigation.png'))} });
await clickShell('.sidebar a[href="/admin/dashboard"]');
await waitShell(() => document.querySelectorAll('[data-testid="admin-metric-card"]').length === 8, 'Mobile metric cards');
await waitShell(() => document.querySelector('.sidebar').getBoundingClientRect().right <= 0, 'Mobile drawer closes');
check(await readShell(() => innerWidth === 390 && document.documentElement.scrollWidth <= innerWidth), 'Mobile dashboard has no horizontal overflow');
await page.screenshot({ path: ${JSON.stringify(resolve(artifacts, 'dashboard-mobile.png'))} });
await page.goto('http://127.0.0.1:13031/dashboard');
await page.waitForFunction(() => !document.documentElement.hasAttribute('data-admin-ui'));
check(true, 'User route clears admin theme');
await fs.writeFile(${JSON.stringify(resolve(artifacts, 'browser-result.json'))}, JSON.stringify({ checks, fixture: true, checkedAt: new Date().toISOString() }, null, 2));
await task.finish({ keep: [] });
console.log(JSON.stringify({ checks }));
`
  const browser = spawn('ego-browser', ['nodejs', '-e', browserScript], { stdio: 'inherit' })
  const code = await new Promise(resolve => browser.on('exit', resolve))
  await writeFile(resolve(artifacts, 'requests.json'), JSON.stringify(requests, null, 2))
  if (code !== 0) throw new Error(`Browser E2E exited with code ${code}`)
  console.log(`Artifacts: ${artifacts}`)
} finally {
  stop()
  await writeFile(resolve(artifacts, 'vite.log'), serverLog)
}
