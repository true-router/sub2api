// Browser regression against the existing local development server on port 3000.
import { readFile, mkdir } from 'node:fs/promises'
import { resolve } from 'node:path'
import { spawn } from 'node:child_process'
const root = resolve(import.meta.dirname, '../..')
const output = resolve(root, '.planning/2026-11-11-balance-sources/artifacts/tooltip')
await mkdir(output, { recursive: true })
const source = (await readFile(resolve(import.meta.dirname, 'balance-tooltip.browser.mjs'), 'utf8'))
  .replace('"__REPOSITORY_ROOT__"', JSON.stringify(root))
  .replace('"__ARTIFACT_DIRECTORY__"', JSON.stringify(output))
const child = spawn('ego-browser', ['nodejs'], { stdio: ['pipe', 'inherit', 'inherit'] })
child.stdin.end(source)
child.once('error', () => { process.exitCode = 1 })
child.once('close', code => { process.exitCode = code ?? 1 })
