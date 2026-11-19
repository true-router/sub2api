const task = await taskSpace('Seat terminal stream acceptance')
console.log('EGO space', task.spaceId)
const page = task.page('p1')
try {
  await page.goto(fixture.base + '/login')
  await page.evaluate(async ({ email, password }) => {
    const reply = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })
    const { data } = await reply.json()
    localStorage.setItem('auth_token', data.access_token)
    localStorage.setItem('auth_user', JSON.stringify(data.user))
    localStorage.setItem('sub2api_locale', 'zh')
    localStorage.setItem(`user_guide_${data.user.id}_user_v4_interactive`, 'true')
  }, { email: fixture.email, password: fixture.password })
  await page.goto(fixture.base + '/profile')
  await page.waitForSelector('a[href="/subscriptions/seat-terminals"]', { timeout: 10000 })
  await page.click('a[href="/subscriptions/seat-terminals"]', { label: 'Open seat terminal page' })
  await page.waitForSelector('[data-boardui="table"] tbody tr:nth-child(2)', { timeout: 12000 })
  await page.evaluate(() => {
    const consolePage = document.querySelector('.boardui-admin-layout [data-testid="console-page"]')
    if (consolePage?.querySelector('h1')?.textContent !== '席位终端') {
      throw new Error('seat terminals must use the account workspace and shared console heading')
    }
    const rows = Array.from(consolePage.querySelectorAll('[data-boardui="table"] tbody tr[data-row-id]'))
    if (rows.length !== 2 || !rows.every(row =>
      row.cells[0].textContent.trim() === 'original-user-session-7f52cef9-5830' &&
      row.querySelector('[data-boardui="badge"]'))) throw new Error('original session IDs or BoardUI badges missing')
    const tableFrame = rows[0].closest('table').parentElement.getBoundingClientRect()
    if (!rows.every(row => row.querySelector('button').getBoundingClientRect().right <= tableFrame.right)) {
      throw new Error('seat actions are outside the visible table')
    }
    if (document.body.innerText.includes('ba579c2d') || document.body.innerText.includes('unassigned-original-session')) {
      throw new Error('internal seat ID or unassigned session rendered')
    }
  })
  await page.screenshot({ path: fixture.listScreenshot })
  const slots = new Set()
  for (const row of [1, 2]) {
    const cell = row === 1 ? 'td:first-child' : 'button'
    await page.click(`[data-boardui="table"] tbody tr:nth-child(${row}) ${cell}`, { label: 'Open selected session terminal' })
    await page.waitForSelector('[data-boardui="drawer"]', { timeout: 10000 })
    await page.waitForFunction(() => Array.from(document.querySelectorAll('.xterm-rows > div'))
      .some(row => row.textContent.includes('Live update received')), undefined, { timeout: 10000 })
    slots.add(await page.evaluate(() => Array.from(document.querySelectorAll('.xterm-rows > div'))
      .find(row => row.textContent.includes('Fixture slot'))?.textContent.trim()))
    if (row === 2) {
      await page.waitForFunction(() => {
        const drawer = document.querySelector('[data-boardui="drawer"]')
        const panel = drawer?.querySelector('.modal-content')
        return drawer && panel && getComputedStyle(drawer).opacity === '1' &&
          getComputedStyle(panel).opacity === '1' && !drawer.classList.contains('modal-enter-active')
      }, undefined, { timeout: 5000 })
      await page.screenshot({ path: fixture.screenshot })
    }
    await page.keyboard.press('Escape')
    await page.waitForSelector('[data-boardui="drawer"]', { state: 'hidden', timeout: 5000 })
  }
  if (slots.size !== 2 || slots.has(undefined)) throw new Error('same session ID opened the wrong terminal')
  console.log('PASS browser seat list and incremental terminal output')
} catch (error) {
  console.log('EGO acceptance failed', error.message, await page.url())
  console.log((await page.snapshot()).slice(-1800))
  throw error
}
