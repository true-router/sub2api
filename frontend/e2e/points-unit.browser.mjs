const fs = await import('node:fs/promises')
const output = "__OUTPUT__"
const task = await taskSpace('sub2api internal points display')
console.log({ spaceId: task.spaceId })
const page = task.page('p1')
const receipt = { result: 'FAIL', boundary: 'Actual Vue frontend and browser with local API fixtures; no production or payment-provider writes.', checks: [] }
async function check(name, passed) {
  receipt.checks.push({ name, passed: !!passed })
  await fs.writeFile(output + '/summary.json', JSON.stringify(receipt,null,2))
  if (!passed) throw Error(name)
  console.log('PASS '+name)
}
async function go(path, selector) { await page.goto('http://127.0.0.1:13109'+path); if (selector) await page.waitForSelector(selector) }
try {
  await go('/login')
  await page.evaluate(() => {
    localStorage.setItem('auth_token','local-member-preview')
    localStorage.setItem('auth_user',JSON.stringify({id:2,role:'user',email:'member1@example.test',balance:100}))
    localStorage.setItem('sub2api_locale','zh')
    localStorage.setItem('user_guide_2_user_v4_interactive','true')
  })
  await go('/dashboard','[data-testid="home-balance"]')
  await check('Balance uses points and preserves the numeric value',await page.evaluate(()=>{const t=document.querySelector('[data-testid="home-balance"]').innerText;return /100\.00/.test(t)&&/积分|P\s/.test(t)&&!t.includes('$')}))
  await go('/account/model-plaza','[data-testid="offer-prices"]')
  await check('Model price applies group multiplier and uses points',await page.evaluate(()=>{const t=document.querySelector('[data-testid="site-points"]').innerText;return /0\.500000/.test(t)&&/积分|P\s/.test(t)&&!t.includes('$')}))
  await check('Model pricing distinguishes official USD reference from internal points',await page.evaluate(()=>{const t=document.querySelector('[data-testid="offer-prices"]').innerText;return t.includes('US$1.00')&&t.includes('0.500000')}))
  await go('/usage','[data-testid="billing-source"],main table')
  await check('Usage charges use points',await page.evaluate(()=>{const t=document.querySelector('main').innerText;return /P\s*0\.040000|0\.040000\s*积分/.test(t)&&!t.includes('$0.040000')}))
  await go('/purchase')
  await page.waitForSelector('[aria-label="可选充值积分"]')
  await check('Recharge distinguishes actual CNY payment from credited points',await page.evaluate(()=>{const t=document.querySelector('main').innerText;return /CNY\s*10/.test(t)&&/10\s*积分|P\s*10/.test(t)&&!t.includes('10 USD')}))
  await go('/model-plaza','[data-testid="own-price"]')
  await check('Public price comparison keeps real currency',await page.evaluate(()=>{const t=document.querySelector('[data-testid="own-price"]').innerText;return /¥|￥/.test(t)&&!t.includes('P ')}))
  await go('/account/model-plaza','[data-testid="offer-prices"]')
  await page.evaluate(()=>{localStorage.setItem('sub2api_locale','en')})
  await page.reload()
  await page.waitForSelector('[data-testid="offer-prices"]')
  await check('English model pricing uses points',await page.evaluate(()=>{const t=document.querySelector('[data-testid="site-points"]').innerText;return /points|P\s/.test(t)&&!t.includes('$')}))
  receipt.result='PASS'
  await fs.writeFile(output+'/summary.json',JSON.stringify(receipt,null,2))
  await task.finish({keep:[]})
} finally {
  await fs.writeFile(output+'/summary.json',JSON.stringify(receipt,null,2))
}
