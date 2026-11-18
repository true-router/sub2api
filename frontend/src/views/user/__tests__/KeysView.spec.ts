import { createPinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { apiClient } from '@/api/client'
import type { ApiKey } from '@/types'
import KeysView from '../KeysView.vue'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))

const createApiKey = (): ApiKey => ({
  id: 1,
  user_id: 1,
  key: 'sk-test-key',
  name: 'test-key',
  group_id: null,
  status: 'active',
  ip_whitelist: [],
  ip_blacklist: [],
  last_used_at: null,
  last_used_ip: null,
  quota: 0,
  quota_used: 0,
  expires_at: null,
  created_at: '2026-06-27T00:00:00Z',
  updated_at: '2026-06-27T00:00:00Z',
  current_concurrency: 3,
  rate_limit_5h: 0,
  rate_limit_1d: 0,
  rate_limit_7d: 0,
  usage_5h: 0,
  usage_1d: 0,
  usage_7d: 0,
  window_5h_start: null,
  window_1d_start: null,
  window_7d_start: null,
  reset_5h_at: null,
  reset_1d_at: null,
  reset_7d_at: null,
})

const originalAdapter = apiClient.defaults.adapter
let rows: ApiKey[]
let wrapper: VueWrapper
const requests: string[] = []
let failList = false
let nextID = 2

async function openPage() {
  wrapper = mount(KeysView, {
    global: { plugins: [createPinia()], stubs: { AppLayout: { template: '<main><slot /></main>' }, Teleport: true } },
  })
  await flushPromises()
}

async function click(label: string, scope = 'main') {
  const button = wrapper.get(scope).findAll('button').find(button => button.text() === label)
  expect(button, label).toBeDefined()
  await button!.trigger('click')
  await flushPromises()
}

beforeEach(() => {
  rows = [{ ...createApiKey(), key: 'sk-fixture-credential-secret', is_default: true }]
  requests.length = 0
  nextID = 2
  failList = false
  apiClient.defaults.adapter = async config => {
    const path = config.url!
    const request = config.method + ' ' + path
    requests.push(request)
    let data: unknown
    if (request === 'post /user/api-access') {
      data = { default_key: rows[0], state: 'ready' }
    } else if (request === 'get /keys') {
      if (failList) throw new Error('fixture temporarily unavailable')
      const { page, page_size } = config.params
      data = { items: rows.slice((page - 1) * page_size, page * page_size), total: rows.length }
    } else if (request === 'post /user/api-access/keys') {
      expect(JSON.parse(config.data)).toEqual({})
      data = { ...createApiKey(), id: nextID++, name: 'additional', access_mode: 'unified' }
      rows.push(data as ApiKey)
    } else {
      const key = rows.find(item => path === '/keys/' + item.id)
      if (!key) throw new Error('Unexpected request: ' + request)
      if (config.method === 'put') data = Object.assign(key, JSON.parse(config.data))
      else if (config.method === 'delete') rows = rows.filter(item => item.id !== key.id)
      else throw new Error('Unexpected method: ' + request)
    }
    return { config, data: { code: 0, data }, status: 200, statusText: 'OK', headers: {} }
  }
})

afterEach(() => {
  wrapper?.unmount()
  apiClient.defaults.adapter = originalAdapter
})

describe('user key management', () => {
  it('prepares access, masks the credential and creates an additional unified key', async () => {
    await openPage()
    expect(requests).toEqual(['post /user/api-access', 'get /keys'])
    expect(wrapper.get('[data-key-id="1"]').text()).toContain('console.default')
    expect(wrapper.text()).not.toContain(rows[0].key)
    await wrapper.get('[data-testid="create-additional-key"]').trigger('click')
    await flushPromises()
    expect(wrapper.get('[data-key-id="2"]').text()).toContain('additional')
    expect(requests.slice(-2)).toEqual(['post /user/api-access/keys', 'get /keys'])
  })

  it('edits limits, toggles status and confirms deletion on the selected key', async () => {
    await openPage()
    await click('common.edit', '[data-key-id="1"]')
    const form = wrapper.get('#console-key-form')
    await form.get('input').setValue('Updated key')
    await form.get('input[type="number"]').setValue('12')
    await form.trigger('submit')
    await flushPromises()
    expect(rows[0]).toMatchObject({ name: 'Updated key', quota: 12 })
    await click('console.disableKey', '[data-key-id="1"]')
    expect(rows[0].status).toBe('inactive')
    await click('console.enableKey', '[data-key-id="1"]')
    expect(rows[0].status).toBe('active')
    await click('common.delete', '[data-key-id="1"]')
    expect(wrapper.get('[role="dialog"]').text()).toContain('console.deleteDefaultNote')
    await click('common.cancel', '[role="dialog"]')
    expect(rows).toHaveLength(1)
    await click('common.delete', '[data-key-id="1"]')
    await click('common.delete', '[role="dialog"]')
    expect(wrapper.get('[data-testid="keys-empty"]').exists()).toBe(true)
  })

  it('recovers from a failed list request and loads the next page', async () => {
    rows = Array.from({ length: 11 }, (_, i) => ({ ...createApiKey(), id: i + 1 }))
    failList = true
    await openPage()
    expect(wrapper.get('[role="alert"]').text()).toContain('console.loadFailed')
    failList = false
    await click('apiAccess.retry')
    expect(wrapper.findAll('[data-key-id]')).toHaveLength(10)
    const next = wrapper.findAll('button').find(button => button.text() === 'pagination.next' || button.attributes('aria-label') === 'pagination.next')
    expect(next).toBeDefined()
    await next!.trigger('click')
    await flushPromises()
    expect(wrapper.findAll('[data-key-id]').map(item => item.attributes('data-key-id'))).toEqual(['11'])
  })
})
