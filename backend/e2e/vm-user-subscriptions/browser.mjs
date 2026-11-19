// All logins below are disposable users created by check.py.
import { spawnSync } from "node:child_process";
const [origin, evidence, panel] = process.argv.slice(2);
function run(code) {
	const r = spawnSync("ego-browser", ["nodejs", "-e", code], {
		encoding: "utf8",
		timeout: 90000,
	});
	const text = r.stdout + "\n" + r.stderr;
	if (r.status !== 0 || r.error) throw Error(text || String(r.error));
	return text;
}
const created = run(
	'const t=await taskSpace("专属订阅隔离验收"); console.log("SPACE_ID="+t.spaceId)',
);
const id = Number(created.match(/SPACE_ID=(\d+)/)?.[1]);
if (!id) throw Error("No browser task space");
const setup = `const t=await taskSpace(${id});const p=t.page("p1"); const origin=${JSON.stringify(origin)},out=${JSON.stringify(evidence)};`;
function step(script) {
	run(setup + script);
}
function login(email, path) {
	step(`await p.goto(origin+"/login");await p.evaluate(async ({email})=>{
    const r=await fetch('/api/v1/auth/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email,password:'Isolated-vm-user-123!'})});
    const {data}=await r.json(); localStorage.setItem('auth_token',data.access_token);localStorage.setItem('auth_user',JSON.stringify(data.user));localStorage.setItem('sub2api_locale','zh');
    localStorage.setItem((data.user.role==='admin'?'admin':'user')+'_guide_'+data.user.id+'_'+data.user.role+'_v4_interactive','true');
  },{email:${JSON.stringify(email)}});await p.goto(origin+${JSON.stringify(path)});`);
}
try {
    step(`await p.goto(${JSON.stringify(panel)}+'/#/login');
    await p.fill('#login-username','admin');await p.fill('#login-password','testpass');
    await p.click('button[type=submit]');await p.waitForFunction(()=>!location.hash.includes('/login'));
    await p.evaluate(async()=>{const res=await fetch('/api/panel/onboarding',{method:'PUT',headers:{Authorization:'Bearer '+localStorage.getItem('kin_console_token'),'Content-Type':'application/json'},body:JSON.stringify({action:'dismiss'})});if(!res.ok)throw Error('fixture onboarding '+res.status)});
    await p.goto('about:blank');await p.goto(${JSON.stringify(panel)}+'/#/users');
    await p.waitForSelector('text="changed@fixture.invalid"');
    await p.click('loc=role:button[name="新建用户"]');await p.waitForSelector('#new-email');
    await p.fill('#new-email','new-user@fixture.invalid');
    await p.screenshot({path:out+'/vm-user-email.png'});`);
	login("admin@fixture.invalid", "/admin/subscriptions");
	step(`await p.click('loc=role:button[name="分配订阅"]');await p.waitForSelector('#subscription-assign-mode');
 await p.fill('[data-assign-user-search] input','ui-member');await p.click('text="ui-member@fixture.invalid"');
 await p.waitForFunction(()=>document.querySelector('#assign-subscription-form')?.innerText.includes('请核对两端身份'));
 await p.screenshot({path:out+'/confirm-identities.png'});
 await p.click('#subscription-assign-mode');await p.click('[role=option]:has-text("选择已有分组")');
 await p.waitForSelector('#assign-subscription-form .select-trigger:nth-of-type(1)');
 await p.waitForFunction(()=>document.querySelector('#subscription-assign-mode')?.textContent.includes('选择已有分组') && document.querySelectorAll('#assign-subscription-form .select-trigger').length===2);
 await p.screenshot({path:out+'/existing-group.png'});
 await p.click('#subscription-assign-mode');await p.click('[role=option]:has-text("创建专属分组")');await p.waitForFunction(()=>document.querySelector('#assign-subscription-form')?.innerText.includes('请核对两端身份'));
 await p.click('button[form=assign-subscription-form]');await p.waitForSelector('#assign-subscription-form',{state:'hidden'});`);
	login("member-a@fixture.invalid", "/subscriptions");
	step(`await p.waitForFunction(()=>{const s=document.querySelector('[data-testid=slot-usage]')?.innerText;return s?.includes('12%')&&s.includes('64%')&&s.includes('81%')&&s.includes('9%')});
 await p.waitForSelector('text="稍后领取"');await p.click('text="稍后领取"');
 await p.waitForFunction(()=>{const el=document.querySelector('[data-testid=slot-usage]');const r=el.getBoundingClientRect();return el.contains(document.elementFromPoint(r.left+12,r.top+12))});
 await p.screenshot({path:out+'/user-slot-usage.png'});`);
} finally {
	run("const t=await taskSpace(" + id + ");await t.finish({keep:[]})");
}
