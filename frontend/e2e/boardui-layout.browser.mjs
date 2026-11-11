/** Browser program. Invoke through check-preview.mjs to resolve artifact paths. */
const fs = await import('node:fs/promises')
const task = await taskSpace('sub2api-boardui-layout-acceptance')
const page = task.page('p1')
const output = "__BOARDUI_ARTIFACT_DIRECTORY__"
await fs.mkdir(output,{recursive:true})
const checks=[]
const check=(value,name)=>{if(!value)throw new Error(name);checks.push(name);console.log('PASS: '+name)}
await page.goto('http://127.0.0.1:13031/login')
const login=await page.fetch('/api/v1/auth/login',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({email:'admin@example.test',password:'BoardUI-Preview-2026'})})
const data=(typeof login.body==='string'?JSON.parse(login.body):login.body).data
await page.evaluate(data=>{localStorage.setItem('auth_token',data.access_token);localStorage.setItem('auth_user',JSON.stringify(data.user));localStorage.setItem('sub2api_locale','zh');localStorage.setItem('theme','light');localStorage.setItem('admin_guide_1_admin_v4_interactive','true')},data)
await page.cdp('Emulation.setDeviceMetricsOverride',{width:1440,height:1000,deviceScaleFactor:1,mobile:false})
await page.goto('http://127.0.0.1:13031/admin/dashboard')
await page.waitForSelector('[data-testid="admin-metric-card"]')
check(await page.evaluate(()=>!document.querySelector('.app-content > .app-header')),'The content area has no global top header')
check(await page.evaluate(()=>!!document.querySelector('.sidebar [data-testid="command-trigger"]')&&!!document.querySelector('.sidebar [data-testid="sidebar-account-menu"]')),'Search and account controls live in the sidebar')
check(await page.evaluate(()=>!!document.querySelector('main .admin-content-heading [data-boardui="breadcrumb"]')),'Breadcrumb and page title belong to the content region')
check(await page.evaluate(()=>document.querySelector('[data-testid="sidebar-available-balance"]')?.textContent.includes('$100.00')&&!document.querySelector('[data-testid="sidebar-account-popover"]')),'Available balance is visible on the closed sidebar account card')
check(await page.evaluate(()=>{
  const selected=document.querySelector('.sidebar .sidebar-link-active')
  if(!selected)return false
  const style=getComputedStyle(selected)
  return style.backgroundImage.startsWith('linear-gradient(')&&style.boxShadow.includes('inset')&&style.borderRadius==='10px'&&Math.abs(selected.getBoundingClientRect().height-36)<1
}),'Selected navigation uses the registry gradient, edge highlight, and compact shape')
check(await page.evaluate(()=>{
  const main=document.querySelector('.app-main')
  const sidebar=document.querySelector('.sidebar')
  const metric=document.querySelector('[data-testid="admin-metric-card"]')
  return innerWidth===1440 && getComputedStyle(main).paddingTop==='24px' && sidebar.getBoundingClientRect().width===264 && metric.getBoundingClientRect().height<=190
}),'Desktop spacing and metric cards use the compact dimensions')
await page.screenshot({path:output+'/dashboard-desktop.png'})
await page.click('[data-testid="sidebar-account-menu"]')
await page.waitForSelector('[data-testid="sidebar-account-popover"]')
check(await page.evaluate(()=>document.querySelector('[data-testid="sidebar-account-popover"]').textContent.includes('admin@example.test')),'Sidebar account menu exposes current account information')
await page.keyboard.press('Escape')
await page.cdp('Emulation.setDeviceMetricsOverride',{width:1440,height:1000,deviceScaleFactor:1,mobile:false})
await page.goto('http://127.0.0.1:13031/admin/users')
await page.waitForSelector('.table-wrapper tbody tr[data-index] td')
check(await page.evaluate(()=>innerWidth===1440 && getComputedStyle(document.querySelector('.table-wrapper tbody tr[data-index] td')).paddingTop==='6px'),'Desktop table rows use the compact padding')
await page.screenshot({path:output+'/users-desktop.png'})
await page.cdp('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true})
const raw = async expression => (await page.cdp('Runtime.evaluate',{expression,returnByValue:true})).result.value
const click = async selector => {
 const point=await raw('(()=>{const r=document.querySelector('+JSON.stringify(selector)+').getBoundingClientRect();return{x:r.left+r.width/2,y:r.top+r.height/2}})()')
 await page.cdp('Input.dispatchMouseEvent',{type:'mousePressed',...point,button:'left',clickCount:1})
 await page.cdp('Input.dispatchMouseEvent',{type:'mouseReleased',...point,button:'left',clickCount:1})
}
await click('[data-testid="mobile-menu-toggle"]')
for(let i=0;i<100;i++){if(await raw('document.querySelector(".sidebar").getBoundingClientRect().left>=0'))break;await new Promise(resolve=>setTimeout(resolve,20))}
check(await raw('document.querySelector(".sidebar").getBoundingClientRect().left>=0'),'Mobile button opens the same sidebar')
check(await raw('innerWidth===390 && document.querySelector(".sidebar").getBoundingClientRect().width===280 && document.querySelector("[data-testid=command-trigger]").getBoundingClientRect().height===40'),'Mobile retains the original drawer and search sizes')
await page.screenshot({path:output+'/sidebar-mobile.png'})
await fs.writeFile(output+'/result.json',JSON.stringify({checks,fixture:true},null,2))
await task.finish({keep:[]})
