/** Actual account route with deterministic catalog/FX fixtures. */
import { createAdminFixtureServer } from './fixtures/admin-api.mjs'
import { createServer } from 'node:http'
import { spawn } from 'node:child_process'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createHash } from 'node:crypto'
const root = resolve(import.meta.dirname,'..')
const output = resolve(process.env.COMPARISON_E2E_OUTPUT || '/tmp/sub2api-account-comparison-e2e')
await mkdir(output,{recursive:true})
const requests=[]
const base=createAdminFixtureServer({requests,accountPreview:true,dashboardPreview:true,modelProvidersPreview:true})
let scenario='standard'
const server=createServer((req,res)=>{
 const path=new URL(req.url,'http://localhost')
 let data
 if(path.pathname==='/__scenario'){scenario=path.searchParams.get('name');data={scenario}}
 if(path.pathname==='/api/v1/user/api-access/pricing-context'){
  if(scenario==='context-error'){res.writeHead(503);res.end('{}');return}
  data={cny_per_credit:scenario==='recharge-two'?2:1,exchange_rate:scenario==='no-fx'?null:{usd_to_cny:7,date:scenario==='old-fx'?'2000-01-01':new Date().toISOString().slice(0,10),source:'E2E fixture',stale:scenario==='cached-fx'}}
 }
 if(path.pathname==='/api/v1/user/api-access/models'){
  const pricing={billing_mode:'token',input_price:scenario==='zero'?0:0.000002,output_price:0.00002,cache_read_price:0.0000002}
  const official=scenario==='missing-official'?null:{input_price:0.00001,output_price:0.00002,cache_read_price:0.000001}
  if(scenario==='tiers'){
   pricing.intervals=[{min_tokens:0,max_tokens:200000,input_price:0.000002,output_price:0.00002},{min_tokens:200000,max_tokens:null,input_price:0.000004,output_price:0.00004}]
   official.intervals=[{min_tokens:0,max_tokens:200000,input_price:0.00001,output_price:0.00002},{min_tokens:200000,max_tokens:null,input_price:0.00002,output_price:0.00004}]
  }
  if(scenario==='per-request'){pricing.billing_mode='per_request';pricing.per_request_price=0.5}
  const offer={endpoint:'chat_completions',rate_multiplier:0.5,pricing,official_pricing:official,long_context_basis:scenario==='marginal'?'marginal':'whole_request'}
  data=[{id:'gpt-comparison-fixture',platform:'openai',subscription_available:false,wallet_offers:[offer,{...offer,endpoint:'responses'}]}]
 }
 if(data!==undefined){requests.push({method:req.method,path:path.pathname,scenario});res.setHeader('Content-Type','application/json');res.end(JSON.stringify({code:0,data}));return}
 base.emit('request',req,res)
})
await new Promise(resolve=>server.listen(18110,'127.0.0.1',resolve))
const vite=spawn('pnpm',['exec','vite','--host','127.0.0.1','--port','13110','--strictPort'],{cwd:root,env:{...process.env,VITE_ACCOUNT_PREVIEW:'true',VITE_DEV_PROXY_TARGET:'http://127.0.0.1:18110'},stdio:['ignore','pipe','pipe']})
let log='';vite.stdout.on('data',x=>{log+=x});vite.stderr.on('data',x=>{log+=x})
try{
 for(let i=0;i<120;i++){try{if((await fetch('http://127.0.0.1:13110')).ok)break}catch{};if(vite.exitCode!==null||i===119)throw Error(log);await new Promise(r=>setTimeout(r,250))}
 const source=(await readFile(resolve(import.meta.dirname,'model-price-comparison.browser.mjs'),'utf8')).replace('"__OUTPUT__"',JSON.stringify(output))
 const child=spawn('ego-browser',['nodejs','-e',source],{stdio:'inherit'})
 const code=await new Promise((resolve,reject)=>{child.once('error',reject);child.once('exit',resolve)})
 if(code)throw Error('Account model price comparison failed')
}finally{
 vite.kill('SIGTERM');server.closeAllConnections();server.close()
 await writeFile(resolve(output,'requests.json'),JSON.stringify(requests,null,2))
 await writeFile(resolve(output,'vite.log'),log)
 const hashes={}
 for(const p of ['e2e/model-price-comparison.mjs','e2e/model-price-comparison.browser.mjs','src/views/user/UnifiedModelsView.vue','src/components/modelPlaza/UnifiedPriceComparison.vue','src/components/modelPlaza/plazaPricing.ts','src/api/apiAccess.ts']){try{hashes[p]=createHash('sha256').update(await readFile(resolve(root,p))).digest('hex')}catch{hashes[p]=null}}
 await writeFile(resolve(output,'source-sha256.json'),JSON.stringify(hashes,null,2))
}
