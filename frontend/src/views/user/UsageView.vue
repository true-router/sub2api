<script setup lang="ts">
import { defineAsyncComponent, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { usageAPI } from '@/api/usage'
import type { UsageLog, UsageStatsResponse } from '@/types'
import { formatDateLocalInput, formatDateTime } from '@/utils/format'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import UsageCharge from '@/components/user/console/UsageCharge.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardInput from '@/components/boardui/Input.vue'
import BoardTable from '@/components/boardui/Table.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import Pagination from '@/components/common/Pagination.vue'
const UsageExplorer = defineAsyncComponent(() => import('@/components/user/console/UsageExplorer.vue'))
const { t } = useI18n()
const logs = ref<UsageLog[]>([])
const stats = ref<UsageStatsResponse | null>(null)
const start = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const end = ref(formatDateLocalInput(new Date()))
const model = ref('')
const page = ref(1)
const total = ref(0)
const loading = ref(false)
const failed = ref(false)
const advanced = ref(false)
let generation = 0
async function load(reset = false) {
  if (reset) page.value = 1
  const current = ++generation
  loading.value = true; failed.value = false
  try {
    const params = { start_date: start.value, end_date: end.value, model: model.value.trim() || undefined }
    const [list, summary] = await Promise.all([usageAPI.query({ ...params, page: page.value, page_size: 20, sort_by: 'created_at', sort_order: 'desc' }), usageAPI.getStats(params)])
    if (current !== generation) return
    logs.value = list.items; total.value = list.total; stats.value = summary
  } catch { if (current === generation) failed.value = true }
  finally { if (current === generation) loading.value = false }
}
function changePage(value: number) { page.value = value; void load() }
onMounted(() => load())
</script>
<template>
  <ConsolePage :title="t('console.usage')" :description="t('console.usageDescription')">
    <form class="flex flex-wrap items-end gap-3" @submit.prevent="load(true)">
      <BoardInput v-model="start" type="date" :label="t('console.startDate')" class="w-full sm:w-44" required />
      <BoardInput v-model="end" type="date" :label="t('console.endDate')" class="w-full sm:w-44" required :min="start" />
      <BoardInput v-model="model" :label="t('usage.model')" :placeholder="t('console.allModels')" class="w-full sm:w-48" />
      <BoardButton type="submit" :loading="loading">{{ t('console.apply') }}</BoardButton>
    </form>
    <div v-if="failed" role="alert" class="card space-y-3 p-5"><p class="text-body-regular text-text-error-primary">{{ t('console.loadFailed') }}</p><BoardButton variant="secondary" @click="load()">{{ t('apiAccess.retry') }}</BoardButton></div>
    <BoardSkeleton v-else-if="loading && !stats" height="10rem" />
    <template v-else>
      <div v-if="stats" class="grid grid-cols-1 gap-3 sm:grid-cols-3" data-testid="usage-summary">
        <section class="card space-y-2 p-4"><h2 class="text-body-regular text-text-secondary">{{ t('console.requests') }}</h2><p class="text-title-2-medium tabular-nums text-text-primary">{{ stats.total_requests.toLocaleString() }}</p></section>
        <section class="card space-y-2 p-4"><h2 class="text-body-regular text-text-secondary">{{ t('console.tokens') }}</h2><p class="text-title-2-medium tabular-nums text-text-primary">{{ stats.total_tokens.toLocaleString() }}</p></section>
        <section class="card space-y-2 p-4"><h2 class="text-body-regular text-text-secondary">{{ t('console.actualCost') }}</h2><p class="text-title-2-medium tabular-nums text-text-primary">${{ stats.total_actual_cost.toFixed(4) }}</p></section>
      </div>
      <section v-if="!logs.length" class="card space-y-3 p-6" data-testid="usage-empty"><h2 class="text-title-3-semibold text-text-primary">{{ t('console.noRequests') }}</h2><p class="text-body-regular text-text-secondary">{{ model ? t('console.noMatchingRequests') : t('console.usageEmptyNote') }}</p><router-link to="/dashboard" class="text-body-medium text-text-accent underline">{{ t('console.viewAccess') }}</router-link></section>
      <section v-else class="card overflow-hidden" data-testid="usage-list">
        <div class="border-b border-separator-border p-5"><h2 class="text-title-3-semibold text-text-primary">{{ t('console.recentActivity') }}</h2></div>
        <div class="hidden overflow-x-auto sm:block"><BoardTable><thead><tr><th>{{ t('console.time') }}</th><th>{{ t('usage.model') }}</th><th>{{ t('console.tokens') }}</th><th>{{ t('console.charge') }}</th></tr></thead><tbody><tr v-for="record in logs" :key="record.id" :data-row-id="record.id"><td class="whitespace-nowrap">{{ formatDateTime(record.created_at) }}</td><td><code class="break-all">{{ record.model }}</code><p class="text-caption-1-regular text-text-secondary">{{ record.api_key?.name }}</p></td><td>{{ (record.input_tokens + record.output_tokens + record.cache_creation_tokens + record.cache_read_tokens).toLocaleString() }}</td><td><UsageCharge :record="record" /></td></tr></tbody></BoardTable></div>
        <div class="divide-y divide-separator-border sm:hidden"><article v-for="record in logs" :key="record.id" class="space-y-3 p-4"><div class="space-y-1"><code class="break-all text-body-medium text-text-primary">{{ record.model }}</code><p class="text-caption-1-regular text-text-secondary">{{ formatDateTime(record.created_at) }} · {{ record.api_key?.name }}</p></div><UsageCharge :record="record" /></article></div>
      </section>
      <Pagination v-if="total > 20" :page="page" :total="total" :page-size="20" @update:page="changePage" />
    </template>
    <details class="rounded-3xl border border-border-button-default p-5" @toggle="advanced = ($event.target as HTMLDetailsElement).open"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.usageAdvanced') }}</summary><div v-if="advanced" class="mt-5"><UsageExplorer /></div></details>
  </ConsolePage>
</template>
