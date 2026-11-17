#!/usr/bin/env python3
"""Exercise the real account UI with the isolated e2e.py --hold environment."""
import argparse,json,pathlib,subprocess
p=argparse.ArgumentParser()
p.add_argument('--runtime',type=pathlib.Path,required=True)
p.add_argument('--output',type=pathlib.Path,required=True)
p.add_argument('--space-id',type=int)
a=p.parse_args()
a.output.parent.mkdir(parents=True,exist_ok=True)
config={'runtime':str(a.runtime.resolve()),'output':str(a.output.resolve()),'spaceID':a.space_id}
source=r'''
const config=__CONFIG__;
const fs=await import('node:fs/promises');
const runtime=JSON.parse(await fs.readFile(config.runtime,'utf8'));
const ready=JSON.parse(await fs.readFile(runtime.ready_file,'utf8'));
const task=await taskSpace(config.spaceID ?? 'OpenCode Go OAuth browser acceptance');
const page=task.page('p1');
const checks=[];
const check=(name,passed)=>{checks.push({name,passed:!!passed});console.log(JSON.stringify(checks.at(-1)));if(!passed)throw new Error(name)};
const request=async(method,path,body)=>{
 const result=await page.fetch('/api/v1'+path,{method,headers:{authorization:'Bearer '+ready.access_token,'content-type':'application/json'},...(body?{body:JSON.stringify(body)}:{})});
 if(!result.ok)throw new Error('Fixture API returned '+result.status);
 const data=typeof result.body==='string'?JSON.parse(result.body):result.body;return data.data;
};
try {
 await page.goto(runtime.origin+'/login');
 await page.evaluate(({access_token,user})=>{
  localStorage.setItem('auth_token',access_token);localStorage.setItem('auth_user',JSON.stringify(user));
  localStorage.setItem('sub2api_locale','zh');localStorage.setItem('theme','light');
  localStorage.setItem(`admin_guide_${user.id}_${user.role}_v4_interactive`,'true');
 },ready);
 const notifications=await request('GET','/notifications');
 for(const item of notifications){if(item.type==='gift'&&!item.read_at)await request('POST',`/notifications/gift/${item.id}/read`,{});}
 await page.goto(runtime.origin+'/admin/accounts');
 await page.waitForSelector('text="添加账号"',{state:'visible',timeout:20000});
 if(await page.evaluate(()=>[...document.querySelectorAll('button')].some(x=>x.textContent.trim()==='稍后领取')))await page.click('text="稍后领取"');
 await page.click('text="添加账号"');
 await page.click('[data-testid="platform-button-opencode_go"]');
 const name='OpenCode OAuth browser '+Date.now();
 await page.fill('input[placeholder="请输入账号名称"]',name);
 await page.click('[data-testid="opencode-auth-method"] button');
 await page.click('text="OAuth · OpenCode Go"');
 check('OAuth selection hides API-key inputs',await page.evaluate(()=>!document.querySelector('[role=dialog] input[placeholder="sk-..."]')));
 await page.click('text="无代理"');await page.click('text="OAuth fixture"');
 await page.click('[data-testid="opencode-oauth-start"]');
 await page.waitForSelector('[data-testid=opencode-oauth-form] a',{state:'visible',timeout:15000});
 check('Device flow displays official verification URL',await page.evaluate(()=>document.querySelector('[data-testid=opencode-oauth-form] a')?.href.startsWith('https://opencode.ai/console/auth/device')));
 await page.waitForSelector('[data-testid="opencode-oauth-ready"]',{state:'visible',timeout:15000});
 check('Device authorization completes in the real account form',true);
 await page.click('[role=dialog] button[type=submit]');
 await page.waitForSelector('[data-testid="opencode-oauth-form"]',{state:'detached',timeout:15000});
 const listing=await request('GET','/admin/accounts?search='+encodeURIComponent(name));
 const found=listing.items.find(x=>x.name===name);
 const account=await request('GET','/admin/accounts/'+found.id);
 check('Saved UI account persists OAuth type, Go mode and proxy',account?.type==='oauth' && account.credentials.account_mode==='go' && account.proxy_id===ready.proxy_id);
 await page.waitForSelector(`[data-row-id="${account.id}"]`,{state:'visible'});
 await page.click(`[data-row-id="${account.id}"] button:has-text("更多")`);
 await page.click('text="重新授权"');
 check('Reauthorization uses the OpenCode form',await page.evaluate(()=>!!document.querySelector('[data-testid=opencode-oauth-form]') && document.querySelector('[role=dialog]')?.textContent.includes('OpenCode Go OAuth')));
 await page.click('button[aria-label="授权方式"]');await page.click('text="导入 OpenCode 凭据"');
 await page.fill('[data-testid="opencode-oauth-json"]','invalid json');await page.click('text="导入凭据"');
 await page.waitForSelector('[role=alert]',{state:'visible'});
 check('Invalid import JSON produces a visible validation error',await page.evaluate(()=>document.querySelector('[role=alert]')?.textContent.includes('JSON')));
 const credential=ready.fixture_credential ?? {type:'oauth',access:'fixture-access-2',refresh:'fixture-refresh-2',expires:Date.now()+3600000,metadata:{server:'https://opencode.ai/console',orgID:'org-rotated'}};
 credential.metadata.orgName='Browser reauthorized fixture';
 await page.fill('[data-testid="opencode-oauth-json"]',JSON.stringify(credential));await page.click('text="导入凭据"');
 await page.waitForSelector('[data-testid="opencode-oauth-ready"]',{state:'visible'});
 check('Successful credential import clears the secret input',await page.evaluate(()=>document.querySelector('[data-testid=opencode-oauth-json]').value===''));
 await page.click('text="保存"');await page.waitForSelector('[data-testid="opencode-oauth-form"]',{state:'detached'});
 const updated=await request('GET','/admin/accounts/'+account.id);
 check('Reauthorization persists the new workspace label',updated.credentials.org_name==='Browser reauthorized fixture');
 await page.reload();await page.waitForSelector(`[data-row-id="${account.id}"]`,{state:'visible',timeout:15000});
 check('OAuth accounts do not display legacy quota probe failures',await page.evaluate(()=>!document.body.textContent.includes('account is not a kimi/zhipu/minimax')));
 console.log(JSON.stringify({taskSpaceID:task.spaceId,passed:true}));
} finally {
 await fs.writeFile(config.output,JSON.stringify({taskSpaceID:task.spaceId,checks,passed:checks.length===9&&checks.every(x=>x.passed)},null,2)+'\n');
}
'''.replace('__CONFIG__',json.dumps(config))
raise SystemExit(subprocess.run(['ego-browser','nodejs','-e',source]).returncode)
