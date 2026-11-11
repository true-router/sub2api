import {writeFile, mkdir} from 'node:fs/promises'
import {resolve} from 'node:path'
const output = resolve(import.meta.dirname, '../../output/admin-preview')
await mkdir(output, {recursive:true})
const checks = [
 ['/admin/groups/usage-summary', data => Array.isArray(data), 'group usage summary must be iterable'],
 ['/admin/groups/capacity-summary', data => Array.isArray(data), 'group capacity summary must be iterable'],
 ['/admin/announcements', data => Array.isArray(data.items), 'announcement data table requires items array'],
 ['/admin/settings', data => typeof data.backend_mode_enabled === 'boolean', 'backend mode Toggle requires a boolean'],
 ['/admin/ops/runtime/logging', data => Number.isInteger(data.retention_days), 'LogRetentionSelect requires numeric retention_days']
]
const results=[]
for(const [path,valid,message] of checks){
 const response=await fetch('http://127.0.0.1:13031/api/v1'+path)
 const {data}=await response.json()
 const passed=valid(data)
 results.push({path,passed,message,data})
 console.log(`${passed?'PASS':'FAIL'} ${path}: ${message}`)
}
await writeFile(resolve(output, 'contracts.json'),JSON.stringify(results,null,2))
if(results.some(result=>!result.passed))process.exitCode=1
