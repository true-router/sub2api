<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import * as keysAPI from '@/api/keys'
import { createUnifiedAPIKey, prepareAPIAccess } from '@/api/apiAccess'
import { useClipboard } from '@/composables/useClipboard'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardInput from '@/components/boardui/Input.vue'
import BoardNumberField from '@/components/boardui/NumberField.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { ApiKey } from '@/types'
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const keys = ref<ApiKey[]>([])
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const busy = ref(false)
const failed = ref(false)
const actionError = ref('')
const editing = ref<ApiKey | null>(null)
const deleting = ref<ApiKey | null>(null)
const form = reactive({ name: '', quota: 0 as number | '', expires_at: '', whitelist: '', blacklist: '', rate5h: 0 as number | '', rate1d: 0 as number | '', rate7d: 0 as number | '' })
async function load() {
  loading.value = true; failed.value = false
  try { const result = await keysAPI.list(page.value, 10); keys.value = result.items; total.value = result.total }
  catch { failed.value = true }
  finally { loading.value = false }
}
async function mutate(action: () => Promise<unknown>) {
  if (busy.value) return
  busy.value = true; actionError.value = ''
  try { await action(); await load() }
  catch { actionError.value = t('console.actionFailed') }
  finally { busy.value = false }
}
async function create() { await mutate(async () => { await createUnifiedAPIKey(); page.value = 1 }) }
function edit(key: ApiKey) {
  editing.value = key
  Object.assign(form, { name: key.name, quota: key.quota || 0, expires_at: key.expires_at ? new Date(new Date(key.expires_at).getTime() - new Date(key.expires_at).getTimezoneOffset() * 60000).toISOString().slice(0,16) : '', whitelist: key.ip_whitelist?.join(', ') || '', blacklist: key.ip_blacklist?.join(', ') || '', rate5h: key.rate_limit_5h || 0, rate1d: key.rate_limit_1d || 0, rate7d: key.rate_limit_7d || 0 })
}
const ips = (value: string) => value.split(/[\s,]+/).map(v => v.trim()).filter(Boolean)
async function save() {
  if (!editing.value || !form.name.trim()) return
  const key = editing.value
  await mutate(async () => {
    await keysAPI.update(key.id, { name: form.name.trim(), quota: Number(form.quota), ...(form.expires_at ? {expires_at: new Date(form.expires_at).toISOString()} : key.expires_at ? { expires_at: '' } : {}), ip_whitelist: ips(form.whitelist), ip_blacklist: ips(form.blacklist), rate_limit_5h: Number(form.rate5h), rate_limit_1d: Number(form.rate1d), rate_limit_7d: Number(form.rate7d) })
    editing.value = null
  })
}
async function remove() {
  if (!deleting.value) return
  const key = deleting.value
  await mutate(async () => { await keysAPI.deleteKey(key.id); deleting.value = null; if (keys.value.length === 1 && page.value > 1) page.value-- })
}
async function changePage(value: number) { page.value = value; await load() }
onMounted(async () => { try { await prepareAPIAccess() } catch { /* Listing still allows existing credentials to be managed. */ } await load() })
</script>
<template>
  <ConsolePage :title="t('console.keys')" :description="t('console.keysDescription')">
    <template #actions><BoardButton :loading="busy" @click="create" data-testid="create-additional-key">{{ t('apiAccess.additionalKey') }}</BoardButton></template>
    <p v-if="actionError" role="alert" class="text-body-regular text-text-error-primary">{{ actionError }}</p>
    <div v-if="failed" role="alert" class="card space-y-3 p-6"><p class="text-body-regular text-text-error-primary">{{ t('console.loadFailed') }}</p><BoardButton variant="secondary" @click="load">{{ t('apiAccess.retry') }}</BoardButton></div>
    <BoardSkeleton v-else-if="loading && !keys.length" height="10rem" />
    <section v-else-if="!keys.length" class="card space-y-3 p-6" data-testid="keys-empty"><h2 class="text-title-3-semibold text-text-primary">{{ t('console.noKeys') }}</h2><p class="text-body-regular text-text-secondary">{{ t('apiAccess.revoked') }}</p></section>
    <section v-else class="card divide-y divide-separator-border px-5" data-testid="key-list">
      <article v-for="key in keys" :key="key.id" :data-key-id="key.id" class="space-y-4 py-5">
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="min-w-0 space-y-1"><h2 class="break-all text-title-3-semibold text-text-primary">{{ key.name }}</h2><p class="text-caption-1-regular text-text-secondary">{{ t(key.access_mode === 'unified' ? 'apiAccess.sharedModels' : 'console.legacyKey') }}</p></div>
          <div class="flex flex-wrap gap-2"><BoardBadge v-if="key.is_default" tone="accent">{{ t('console.default') }}</BoardBadge><BoardBadge :tone="key.status === 'active' ? 'success' : 'neutral'">{{ key.status === 'active' ? t('common.active') : key.status === 'inactive' ? t('common.disabled') : key.status }}</BoardBadge></div>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border-button-default p-3"><code class="min-w-0 break-all text-body-regular text-text-primary" dir="ltr">{{ key.key.slice(0,10) }}••••••••</code><BoardButton variant="secondary" @click="copyToClipboard(key.key, t('apiAccess.keyCopied'))">{{ t('apiAccess.copyKey') }}</BoardButton></div>
        <div class="flex flex-wrap gap-2">
          <BoardButton variant="ghost" :disabled="busy" @click="edit(key)">{{ t('common.edit') }}</BoardButton>
          <BoardButton variant="ghost" :disabled="busy" @click="mutate(() => keysAPI.update(key.id, {status: key.status === 'active' ? 'inactive' : 'active'}))">{{ key.status === 'active' ? t('console.disableKey') : t('console.enableKey') }}</BoardButton>
          <BoardButton variant="ghost" :disabled="busy" @click="deleting = key">{{ t('common.delete') }}</BoardButton>
        </div>
      </article>
    </section>
    <Pagination v-if="total > 10" :page="page" :total="total" :page-size="10" @update:page="changePage" />
    <p class="text-caption-1-regular text-text-secondary">{{ t('console.keysNote') }}</p>
    <BaseDialog :show="!!editing" :title="t('keys.editKey')" @close="!busy && (editing = null)">
      <form id="console-key-form" class="space-y-4" @submit.prevent="save">
        <BoardInput v-model="form.name" :label="t('console.keyName')" required :disabled="busy" />
        <details class="space-y-4 rounded-xl border border-border-button-default p-4"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.keyRestrictions') }}</summary>
          <p class="text-caption-1-regular text-text-secondary">{{ t('console.keyRestrictionsNote') }}</p>
          <BoardNumberField v-model="form.quota" :label="t('console.keyBudget')" :min="0" step="any" :disabled="busy" />
          <BoardInput v-model="form.expires_at" type="datetime-local" :label="t('console.keyExpiry')" :disabled="busy" />
          <BoardInput v-model="form.whitelist" :label="t('console.ipWhitelist')" :disabled="busy" />
          <BoardInput v-model="form.blacklist" :label="t('console.ipBlacklist')" :disabled="busy" />
          <BoardNumberField v-model="form.rate5h" :label="t('console.limit5h')" :min="0" step="any" :disabled="busy" />
          <BoardNumberField v-model="form.rate1d" :label="t('console.limit1d')" :min="0" step="any" :disabled="busy" />
          <BoardNumberField v-model="form.rate7d" :label="t('console.limit7d')" :min="0" step="any" :disabled="busy" />
        </details>
        <p v-if="actionError" role="alert" class="text-body-regular text-text-error-primary">{{ actionError }}</p>
      </form>
      <template #footer><BoardButton :loading="busy" :disabled="!form.name.trim()" type="submit" form="console-key-form">{{ t('common.save') }}</BoardButton></template>
    </BaseDialog>
    <BaseDialog :show="!!deleting" :title="t('console.deleteKeyTitle')" @close="!busy && (deleting = null)">
      <p class="text-body-regular text-text-secondary">{{ deleting?.is_default ? t('console.deleteDefaultNote') : t('console.deleteKeyNote') }}</p>
      <template #footer><BoardButton variant="secondary" :disabled="busy" @click="deleting = null">{{ t('common.cancel') }}</BoardButton><BoardButton variant="danger" :loading="busy" @click="remove">{{ t('common.delete') }}</BoardButton></template>
    </BaseDialog>
  </ConsolePage>
</template>
