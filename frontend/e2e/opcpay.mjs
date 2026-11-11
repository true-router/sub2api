import { readFile, mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { spawn } from 'node:child_process'
const output = resolve(import.meta.dirname, '../../output/opcpay-browser')
await mkdir(output, { recursive: true })
const source = (await readFile(resolve(import.meta.dirname, 'opcpay.browser.mjs'), 'utf8')).replace('"__OPCPAY_ARTIFACT_DIRECTORY__"', JSON.stringify(output))
const child = spawn('ego-browser', ['nodejs'], { stdio: ['pipe', 'inherit', 'inherit'] })
child.stdin.end(source)
child.once('error', () => { process.exitCode = 1 })
child.once('close', code => { process.exitCode = code ?? 1 })
