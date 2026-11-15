const fs = await import('node:fs/promises')
const task = await taskSpace('Sub2API landing page')
const page = task.page('p1')
const output = "__ARTIFACT_DIRECTORY__"
const origin = 'http://127.0.0.1:13037'
const checks = []
const check = (name, passed) => { checks.push({ name, passed: Boolean(passed) }); if (!passed) throw new Error(name); console.log('PASS ' + name) }
const settings = async value => {
  const response = await page.fetch('/api/v1/__home_fixture', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(value) })
  if (!response.ok) throw new Error('Fixture update failed')
  await page.goto(origin + '/home')
  await page.waitForSelector('.landing, #custom-home, iframe')
}
const shot = async name => {
  const result = await page.cdp('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true })
  await fs.writeFile(output + '/' + name + '.png', Buffer.from(result.data, 'base64'))
}
const overflow = () => document.documentElement.scrollWidth <= innerWidth
const revealAll = () => page.evaluate(() => document.querySelectorAll('.rv').forEach(n => n.classList.add('in')))
try {
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1050, deviceScaleFactor: 1, mobile: false })
  await page.goto(origin + '/home')
  await page.evaluate(() => { localStorage.removeItem('auth_token'); localStorage.removeItem('auth_user'); localStorage.setItem('sub2api_locale', 'zh'); localStorage.setItem('theme', 'light') })
  await page.reload()
  await page.waitForSelector('[data-testid="landing-home"]')
  check('Fidelity card shows metric consistency and the report link', await page.evaluate(() => {
    const card = document.querySelector('#fidelity .span7')
    const report = card.querySelector('[data-testid="fidelity-report"]')
    return card.querySelector('h3').textContent === '多项指标百分百一致'
      && card.querySelector('.val .num').textContent.endsWith('%')
      && !card.textContent.includes('测试机构')
      && [...card.querySelectorAll('[data-testid="fidelity-metrics"] dt')].map(n => n.textContent).join('|') === '知识准确度|PDF 文档识别|结构化输出|模型家族校验'
      && [...card.querySelectorAll('[data-testid="fidelity-metrics"] dd')].map(n => n.textContent).join('|') === '通过|通过|通过|与官方模型家族一致'
      && report.textContent.includes('查看检测报告')
      && report.href === 'https://www.hvoyai.com/report/CJ26erY31J4'
      && report.target === '_blank'
      && report.rel.includes('noopener')
  }))
  for (const language of ['zh', 'en']) {
    if (language === 'en') await page.click('[data-testid="home-locale-toggle"]')
    await page.waitForFunction(code => document.documentElement.lang === code, language)
    for (const theme of ['light', 'dark']) {
      if (theme === 'dark') await page.click('[data-testid="home-theme-toggle"]')
      const expected = { language, theme }
      check(`${language}/${theme} controls, surfaces and text match the selected preferences`, await page.evaluate(({ language, theme }) => {
        const dark = theme === 'dark'
        const landing = document.querySelector('.landing')
        const root = getComputedStyle(landing)
        const surface = getComputedStyle(document.querySelector('.card'))
        const rows = [...document.querySelectorAll('[data-testid="fidelity-metrics"] > div')]
        return document.documentElement.classList.contains('dark') === dark
          && localStorage.getItem('theme') === theme
          && document.documentElement.lang === language
          && (language === 'zh' ? landing.textContent.includes('多项指标百分百一致') : landing.textContent.includes('100% match across multiple metrics'))
          && root.backgroundColor === (dark ? 'rgb(18, 18, 18)' : 'rgb(250, 250, 250)')
          && surface.backgroundColor === (dark ? 'rgb(26, 26, 26)' : 'rgb(255, 255, 255)')
          && rows.every(row => getComputedStyle(row).backgroundColor === (dark ? 'rgb(31, 31, 31)' : 'rgb(247, 247, 247)'))
          && root.color === (dark ? 'rgb(250, 250, 250)' : 'rgb(10, 10, 10)')
      }, expected))
      await page.reload()
      await page.waitForSelector('[data-testid="home-theme-toggle"]')
      check(`${language}/${theme} persists after reload`, await page.evaluate(({ language, theme }) => document.documentElement.lang === language && document.documentElement.classList.contains('dark') === (theme === 'dark'), expected))
      await page.click('.nav-r a[href="/login"]')
      await page.waitForSelector('[data-testid="auth-shell"]')
      check(`${language}/${theme} preference survives navigation to login`, await page.evaluate(({ language, theme }) => document.documentElement.lang === language && localStorage.getItem('theme') === theme, expected))
      await page.click('.home-link[href="/home"]')
      await page.waitForSelector('[data-testid="home-theme-toggle"]')
      check(`${language}/${theme} restores after returning from login`, await page.evaluate(({ language, theme }) => document.documentElement.lang === language && document.documentElement.classList.contains('dark') === (theme === 'dark'), expected))
      await page.evaluate(() => document.querySelector('.gauge').scrollIntoView({ block: 'center' }))
      await page.waitForFunction(() => document.querySelector('.gauge .val b').textContent === '100%')
      await revealAll()
      await page.waitForFunction(() => [...document.querySelectorAll('.rv')].every(node => getComputedStyle(node).opacity === '1'))
      await page.evaluate(() => window.scrollTo(0, 0))
      await shot(`landing-${language}-${theme}`)
      for (const width of [390, 320]) {
        await page.cdp('Emulation.setDeviceMetricsOverride', { width, height: 844, deviceScaleFactor: 1, mobile: true })
        check(`${language}/${theme} mobile navigation is one flat row with only logo and menu button at ${width}px`, await page.evaluate(() => {
          const nav = document.querySelector('.nav'); const style = getComputedStyle(nav)
          const visible = [...document.querySelectorAll('.nav-r > *')].filter(n => n.getBoundingClientRect().width > 0)
          return document.documentElement.scrollWidth <= innerWidth && nav.getBoundingClientRect().height <= 64 && style.borderRadius === '0px' && visible.length === 1 && visible[0].dataset.testid === 'home-menu-toggle'
        }))
        await page.click('[data-testid="home-menu-toggle"]')
        check(`${language}/${theme} menu keeps login, language, theme and claim visible at ${width}px`, await page.evaluate(() => {
          const rects = ['home-menu-login', 'home-menu-locale', 'home-menu-theme', 'home-menu-primary'].map(id => document.querySelector(`[data-testid="${id}"]`)?.getBoundingClientRect())
          return rects.every(r => r && r.width > 0 && r.left >= 0 && r.right <= innerWidth) && document.querySelector('[data-testid="home-menu-login"]').getAttribute('href') === '/login' && document.querySelector('[data-testid="home-menu-toggle"]').getAttribute('aria-expanded') === 'true'
        }))
        await shot(`landing-menu-${language}-${theme}-${width}`)
        await page.click('[data-testid="home-menu"] a[href="#pricing"]')
        check(`${language}/${theme} menu closes after choosing a link at ${width}px`, await page.evaluate(() => !document.querySelector('[data-testid="home-menu"]')))
        await page.evaluate(() => { history.replaceState(null, '', location.pathname); window.scrollTo(0, 0) })
      }
      await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1050, deviceScaleFactor: 1, mobile: false })
    }
    await page.click('[data-testid="home-theme-toggle"]')
  }
  await page.click('[data-testid="home-locale-toggle"]')
  await page.waitForFunction(() => document.documentElement.lang === 'zh')
  check('Desktop navigation is a flat full-width bar, not a floating capsule', await page.evaluate(() => { const bar = document.querySelector('.navbar').getBoundingClientRect(); const nav = getComputedStyle(document.querySelector('.nav')); return Math.round(bar.width) === document.documentElement.clientWidth && nav.borderRadius === '0px' && nav.boxShadow === 'none' && getComputedStyle(document.querySelector('[data-testid="home-menu-toggle"]')).display === 'none' }))
  check('Anonymous promo bar sits above the navigation and opens registration', await page.evaluate(() => { const promo = document.querySelector('[data-testid="home-promo"]'); const nav = document.querySelector('.navbar'); return promo.getAttribute('href') === '/register' && promo.getBoundingClientRect().bottom <= nav.getBoundingClientRect().top }))
  check('Anonymous primary action opens registration', await page.evaluate(() => document.querySelector('[data-testid="home-primary-action"]').getAttribute('href') === '/register'))
  check('Hero shows four price rows and a 1/10 price', await page.evaluate(() => document.querySelectorAll('.pc .pr').length === 4 && document.querySelector('.pc .pr .o strong').textContent === '$1.5'))
  check('Brand icons render as inline SVG', await page.evaluate(() => document.querySelectorAll('.li svg').length > 20))
  await page.waitForSelector('.nav-links a[href="/model-plaza"]')
  check('Model plaza and documentation links come from settings', await page.evaluate(() => !!document.querySelector('.nav-links a[href="/model-plaza"]') && !!document.querySelector('.nav-links a[href^="https://docs.example.test"]')))
  check('Desktop has no horizontal overflow', await page.evaluate(overflow))
  check('Landing shows the configured site name and no hard-coded brand', await page.evaluate(() => { const t = document.querySelector('.landing').textContent; return t.includes('TrueRouter') && !t.includes('Sub2API') }))
  await page.click('.seg span:nth-child(2)')
  check('Output tab switches price rows', await page.evaluate(() => document.querySelector('.pc .pr .o strong').textContent === '$7.5'))
  await page.evaluate(() => document.querySelector('.gauge').scrollIntoView({ block: 'center' }))
  await page.waitForFunction(() => document.querySelector('.gauge .val b').textContent === '100%')
  check('Fidelity gauge counts up to 100%', true)
  await page.evaluate(() => { const r = document.querySelector('input[type=range]'); r.value = 100; r.dispatchEvent(new Event('input')) })
  check('Calculator updates official price, our price and savings', await page.evaluate(() => { const v = [...document.querySelectorAll('.cmp .v, .save b')].map(n => n.textContent); return v.join() === '$900,$90,$810' }))
  await page.evaluate(() => document.querySelector('.env-btn').scrollIntoView({ block: 'center', behavior: 'instant' }))
  await page.waitForSelector('.env-btn')
  await page.click('.env-btn')
  await page.waitForFunction(() => document.querySelector('.gift-sec').dataset.state === 'open')
  check('Opening the gift reveals the amount and register action', true)
  await page.waitForFunction(() => document.querySelector('.gift-sec .ec-v').textContent === '5.00')
  await shot('landing-desktop')
  await page.click('.gift-sec .btn-p')
  await page.waitForURL('**/register')
  check('Open gift action navigates to registration', true)
  await page.evaluate(() => localStorage.setItem('sub2api_locale', 'en'))
  await page.goto(origin + '/home')
  await page.waitForSelector('[data-testid="landing-home"]')
  check('English locale translates the landing page', await page.evaluate(() => { const t = document.querySelector('.landing').textContent; return t.includes('at 10% of the price') && t.includes('Pay as you go') && !t.includes('多项指标百分百一致') }))
  await page.goto(origin + '/register')
  await page.waitForSelector('[data-testid="auth-shell"]')
  check('English locale translates the gift card on the register page', await page.evaluate(() => document.querySelector('[data-testid="auth-gift"]').textContent.includes('credit')))
  await page.evaluate(() => localStorage.setItem('sub2api_locale', 'zh'))
  await page.goto(origin + '/home')
  await page.waitForSelector('[data-testid="landing-home"]')
  await page.evaluate(() => sessionStorage.setItem('gift', 'opened'))
  await page.goto(origin + '/register')
  await page.waitForSelector('[data-testid="auth-shell"]')
  check('Register page shows the gift card with the opened-gift title', await page.evaluate(() => document.querySelector('[data-testid="auth-gift"] h3').textContent === '你拆开的 $5 赠金在这里' && document.querySelector('[data-testid="auth-gift"] .card b').textContent === '$5.00'))
  check('Register form keeps email, password and confirmation fields', await page.evaluate(() => !!document.querySelector('#email') && !!document.querySelector('#password') && !!document.querySelector('#confirmPassword') && document.querySelector('button[type=submit]').textContent.includes('$5')))
  check('Register page links back to login and home', await page.evaluate(() => !!document.querySelector('.foot a[href="/login"]') && !!document.querySelector('.home-link[href="/home"]')))
  check('Register page never scrolls vertically on desktop', await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight && document.querySelector('.pane').scrollHeight <= document.querySelector('.pane').clientHeight))
  await shot('register-desktop')
  for (const [w, h] of [[1366, 640], [390, 844], [360, 640]]) {
    await page.cdp('Emulation.setDeviceMetricsOverride', { width: w, height: h, deviceScaleFactor: 1, mobile: w < 600 })
    await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth)
    check(`Register page fits ${w}x${h} without scrolling`, await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight && document.querySelector('.pane').scrollHeight <= document.querySelector('.pane').clientHeight))
    await shot(`register-${w}x${h}`)
  }
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1050, deviceScaleFactor: 1, mobile: false })
  await page.evaluate(() => sessionStorage.removeItem('gift'))
  await page.goto(origin + '/home')
  await page.waitForSelector('[data-testid="landing-home"]')
  await settings({ site_logo: '/logo.svg' })
  check('Landing replaces the logo mark with the configured site logo', await page.evaluate(() => document.querySelectorAll('.landing .logo-mark img[src="/logo.svg"]').length === 3 && !document.querySelector('.landing .logo-mark svg')))
  for (const path of ['/register', '/login']) {
    await page.goto(origin + path)
    await page.waitForSelector('[data-testid="auth-shell"]')
    check(`${path} uses the configured site name and logo`, await page.evaluate(() => { const t = document.querySelector('[data-testid="auth-shell"]').textContent; return t.includes('TrueRouter') && !t.includes('Sub2API') && document.querySelectorAll('.logo-mark img[src="/logo.svg"]').length === 2 }))
  }
  await settings({ model_plaza_require_auth: true })
  check('Anonymous visitors cannot see the private model plaza', await page.evaluate(() => !document.querySelector('a[href="/model-plaza"]')))
  await settings({ model_plaza_enabled: false, doc_url: '' })
  check('Disabled plaza and missing documentation hide their links', await page.evaluate(() => !document.querySelector('a[href="/model-plaza"]') && !document.querySelector('a[href="https://docs.example.test"]')))
  for (const width of [390, 320]) {
    await page.cdp('Emulation.setDeviceMetricsOverride', { width, height: 844, deviceScaleFactor: 1, mobile: true })
    await page.waitForFunction(() => document.documentElement.scrollWidth <= innerWidth)
    check(`Mobile landing at ${width}px has no horizontal overflow`, await page.evaluate(overflow))
    await revealAll()
    await shot('landing-mobile-' + width)
  }
  await page.cdp('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1050, deviceScaleFactor: 1, mobile: false })
  await settings({ home_content: '<section id="custom-home">Custom home</section>' })
  await page.waitForSelector('#custom-home')
  check('Custom HTML takes precedence over the landing page', await page.evaluate(() => !document.querySelector('.landing')))
  await settings({ home_content: origin + '/logo.svg' })
  await page.waitForSelector('iframe')
  check('Custom URL takes precedence and has an accessible title', await page.evaluate(() => document.querySelector('iframe').title === 'TrueRouter' && !document.querySelector('.landing')))
  await settings({})
  await page.evaluate(() => { localStorage.setItem('auth_token', 'local-member-preview'); localStorage.setItem('auth_user', JSON.stringify({ id: 2, role: 'user', email: 'member1@example.test' })) })
  await page.reload()
  await page.waitForSelector('[data-testid="home-primary-action"][href="/dashboard"]')
  check('Members do not see the sign-up promo bar', await page.evaluate(() => !document.querySelector('[data-testid="home-promo"]')))
  check('Members see the claimed gift and their dashboard destination', await page.evaluate(() => document.querySelector('.gift-sec').dataset.state === 'claimed' && !document.querySelector('.nav-r a[href="/login"]')))
  await page.evaluate(() => { localStorage.setItem('auth_token', 'local-preview-only'); localStorage.setItem('auth_user', JSON.stringify({ id: 1, role: 'admin', email: 'admin@example.test' })) })
  await page.reload()
  await page.waitForSelector('[data-testid="home-primary-action"][href="/admin/dashboard"]')
  check('Administrators receive their dashboard destination', true)
  await page.evaluate(() => { localStorage.removeItem('auth_token'); localStorage.removeItem('auth_user') })
  await page.cdp('Emulation.clearDeviceMetricsOverride')
  await page.reload()
  await task.finish({ keep: [] })
} finally {
  await fs.writeFile(output + '/checks.json', JSON.stringify({ checks, fixture: true, origin, rerun: 'cd frontend && node e2e/home.mjs' }, null, 2))
}
