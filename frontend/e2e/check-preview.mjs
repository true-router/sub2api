/** Run against the local preview: node e2e/check-preview.mjs components|routes|layout|account|table|settings. */
import { spawn } from 'node:child_process'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'

const scenarios = {
  components: ['boardui-components', 'admin-components'],
  routes: ['preview-routes', 'admin-preview'],
  layout: ['boardui-layout', 'admin-layout'],
  account: ['boardui-account', 'account-boardui'],
  table: ['boardui-table', 'boardui-table'],
  settings: ['preview-settings', 'preview-settings']
}
const name = process.argv[2] || 'components'
const scenario = scenarios[name]
if (!scenario) throw new Error('Expected components, routes, layout, account, table, or settings')
const source = (await readFile(resolve(import.meta.dirname, scenario[0] + '.browser.mjs'), 'utf8'))
  .replace('"__BOARDUI_ARTIFACT_DIRECTORY__"', JSON.stringify(resolve(import.meta.dirname, '../../output', scenario[1])))
const browser = spawn('ego-browser', ['nodejs', '-e', source], { stdio: 'inherit' })
process.exitCode = await new Promise((resolve, reject) => {
  browser.once('error', reject)
  browser.once('exit', code => resolve(code ?? 1))
})
