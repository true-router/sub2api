// Run with node frontend/e2e/balance-tooltip.mjs from the repository root. This uses the existing
// isolated local development account and never changes application data.
const fs = await import('node:fs/promises')
const crypto = await import('node:crypto')
const root = "__REPOSITORY_ROOT__"
const out = "__ARTIFACT_DIRECTORY__"
await fs.mkdir(out, { recursive: true })
const task = await taskSpace('sub2api-balance-tooltip-fix')
const page = task.page('p1')
await page.goto('http://127.0.0.1:3000/admin/users')
await page.waitForSelector('.table-body [data-testid="balance-breakdown"]')
const target = '.table-body td:has([data-testid="balance-breakdown"]) .group > button'
await page.hover(target)
await page.waitForFunction(() => {
  const cell = document.querySelector('.table-body [data-testid="balance-breakdown"]').closest('td')
  const tip = [...document.querySelectorAll('[role="tooltip"]')].find(el => getComputedStyle(el).display !== 'none') || cell.querySelector('.pointer-events-none.absolute')
  return tip && getComputedStyle(tip).opacity === '1'
})
const result = await page.evaluate(() => {
  const cell = document.querySelector('.table-body [data-testid="balance-breakdown"]').closest('td')
  const tip = [...document.querySelectorAll('[role="tooltip"]')].find(el => getComputedStyle(el).display !== 'none') || cell.querySelector('.pointer-events-none.absolute')
  const rect = tip.getBoundingClientRect()
  const original = tip.style.pointerEvents
  tip.style.pointerEvents = 'auto'
  const samples = [0.2, 0.5, 0.8].flatMap(x => [0.2, 0.5, 0.8].map(y => {
    const hit = document.elementFromPoint(rect.left + rect.width * x, rect.top + rect.height * y)
    return !!hit && (hit === tip || tip.contains(hit))
  }))
  tip.style.pointerEvents = original
  return { passed: samples.every(Boolean), text: tip.textContent.trim(), samples, tooltip: rect.toJSON(), header: document.querySelector('.table-header').getBoundingClientRect().toJSON() }
})
if (result.passed) {
  result.viewportChecks = []
  for (const dark of [false, true]) {
    await page.evaluate(dark => document.documentElement.classList.toggle('dark', dark), dark)
    await page.evaluate(() => window.scrollTo(0, Math.min(40, document.documentElement.scrollHeight - innerHeight)))
    await page.hover(target)
    await page.waitForFunction(() => [...document.querySelectorAll('[role="tooltip"]')].some(el => getComputedStyle(el).display !== 'none'))
    await page.evaluate(() => { document.querySelector('.table-wrapper').scrollLeft += 24 })
    await page.hover(target)
    await page.waitForFunction(() => {
      const trigger = document.querySelector('.table-body td:has([data-testid="balance-breakdown"]) .group')
      const tip = [...document.querySelectorAll('[role="tooltip"]')].find(el => getComputedStyle(el).display !== 'none')
      if (!trigger || !tip) return false
      const a = trigger.getBoundingClientRect(), b = tip.getBoundingClientRect()
      return Math.abs(a.top - b.bottom - 8) < 1 && Math.abs((a.left + a.width / 2) - (b.left + b.width / 2)) < 1
    })
    result.viewportChecks.push(await page.evaluate(dark => {
      const tip = [...document.querySelectorAll('[role="tooltip"]')].find(el => getComputedStyle(el).display !== 'none')
      const a = document.querySelector('.table-body td:has([data-testid="balance-breakdown"]) .group').getBoundingClientRect()
      const b = tip.getBoundingClientRect()
      return { theme: dark ? 'dark' : 'light', scrollY, scrollLeft: document.querySelector('.table-wrapper').scrollLeft, gap: a.top - b.bottom, centerOffset: (a.left + a.width / 2) - (b.left + b.width / 2), portaledToBody: tip.parentElement === document.body }
    }, dark))
    await page.screenshot({ path: out + (dark ? '/after-dark.png' : '/after-light.png') })
  }
}
const source = await fs.readFile(root + '/frontend/src/views/admin/UsersView.vue')
result.source_sha256 = crypto.createHash('sha256').update(source).digest('hex')
result.tooltip_source_sha256 = crypto.createHash('sha256').update(await fs.readFile(root + '/frontend/src/components/common/HelpTooltip.vue')).digest('hex')
await page.screenshot({ path: out + (result.passed ? '/after.png' : '/before.png') })
await fs.writeFile(out + (result.passed ? '/after.json' : '/before.json'), JSON.stringify(result, null, 2) + '\n')
console.log(JSON.stringify(result))
if (!result.passed) throw new Error('Balance tooltip is obscured by table content or header')
