<template>
  <div data-testid="key-usage-page" class="flex min-h-screen flex-col bg-background-full text-text-primary">
    <header class="border-b border-separator-border bg-background-primary-default">
      <nav :aria-label="t('home.redesign.navigation')" class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-5 py-4 sm:px-8">
        <router-link to="/home" class="flex min-w-0 items-center gap-2.5 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring">
          <img :src="siteLogo || '/logo.svg'" alt="" class="h-8 w-8 shrink-0 rounded-lg object-contain" />
          <span class="max-w-40 truncate text-headline-semibold sm:max-w-64">{{ siteName }}</span>
        </router-link>
        <div class="flex items-center gap-1 sm:gap-3">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="usage-link hidden text-body-medium sm:inline-flex">{{ t('home.docs') }}</a>
          <LocaleSwitcher />
          <Button variant="ghost" icon-only :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="toggleTheme"><Icon :name="isDark ? 'sun' : 'moon'" size="md" aria-hidden="true" /></Button>
          <router-link to="/home" class="usage-back"><Icon name="arrowLeft" size="sm" aria-hidden="true" />{{ t('keyUsage.design.backHome') }}</router-link>
        </div>
      </nav>
    </header>

    <main class="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-6 px-5 py-8 sm:px-8 sm:py-12">
      <div class="flex flex-col items-start gap-3">
        <Badge tone="accent"><span class="inline-flex items-center gap-1.5"><Icon name="chart" size="xs" aria-hidden="true" />{{ t('keyUsage.design.eyebrow') }}</span></Badge>
        <h1 class="text-display-4-semibold">{{ t('keyUsage.title') }}</h1>
        <p class="max-w-2xl text-headline-regular text-text-secondary">{{ t('keyUsage.subtitle') }}</p>
      </div>

      <section class="usage-card p-5 sm:p-6" :aria-label="t('keyUsage.design.queryHeading')">
        <form class="flex flex-col gap-4 sm:flex-row sm:items-end" @submit.prevent="queryKey">
          <Input id="usage-api-key" v-model="apiKey" type="password" revealable :label="t('keyUsage.design.keyLabel')" :placeholder="t('keyUsage.placeholder')" :disabled="isQuerying" autocomplete="off" autocapitalize="off" spellcheck="false" class="min-w-0 flex-1">
            <template #prefix><Icon name="key" size="md" aria-hidden="true" /></template>
          </Input>
          <Button type="submit" :loading="isQuerying" class="sm:min-w-28" data-testid="key-usage-query"><Icon v-if="!isQuerying" name="search" size="sm" aria-hidden="true" />{{ isQuerying ? t('keyUsage.querying') : t('keyUsage.query') }}</Button>
        </form>
        <p class="mt-3 flex items-start gap-2 text-caption-1-regular text-text-secondary"><Icon name="shield" size="sm" class="shrink-0" aria-hidden="true" />{{ t('keyUsage.privacyNote') }}</p>
        <div v-if="showDatePicker" class="mt-5 flex flex-col gap-3 border-t border-separator-border pt-5" data-testid="key-usage-range">
          <div class="flex flex-wrap items-center gap-2">
            <span class="me-2 text-body-2-medium text-text-secondary">{{ t('keyUsage.dateRange') }}</span>
            <Button v-for="range in dateRanges" :key="range.key" :variant="currentRange === range.key ? 'secondary' : 'ghost'" size="small" :aria-pressed="currentRange === range.key" :disabled="isQuerying" @click="setDateRange(range.key)">{{ range.label }}</Button>
          </div>
          <div v-if="currentRange === 'custom'" class="grid items-end gap-3 sm:grid-cols-[1fr_1fr_auto]">
            <Input id="usage-start-date" v-model="customStartDate" type="date" :label="t('keyUsage.design.startDate')" :disabled="isQuerying" />
            <Input id="usage-end-date" v-model="customEndDate" type="date" :label="t('keyUsage.design.endDate')" :disabled="isQuerying" />
            <Button :disabled="isQuerying" @click="queryKey">{{ t('keyUsage.apply') }}</Button>
          </div>
        </div>
      </section>

      <div v-if="queryError" role="alert" data-testid="key-usage-error" class="flex items-start gap-3 rounded-2xl border border-border-error-default bg-background-error-subtle p-4 text-body-regular text-text-error-primary"><Icon name="exclamationTriangle" size="md" class="shrink-0" aria-hidden="true" /><p class="min-w-0 break-words">{{ queryError }}</p></div>

      <section v-if="showLoading" :aria-label="t('keyUsage.querying')" aria-busy="true" data-testid="key-usage-loading" class="grid gap-4 md:grid-cols-3">
        <div v-for="index in 3" :key="index" class="usage-card flex flex-col gap-5 p-6"><Skeleton width="45%" :height="16" /><Skeleton width="65%" :height="32" /><Skeleton :height="8" /></div>
      </section>
      <section v-else-if="!resultData && !queryError" class="usage-card flex flex-col items-center gap-4 px-5 py-16 text-center" data-testid="key-usage-empty">
        <div class="flex h-14 w-14 items-center justify-center rounded-2xl border border-border-button-default bg-background-secondary-default"><Icon name="chart" size="xl" class="text-foreground-icon-secondary" aria-hidden="true" /></div>
        <h2 class="text-title-3-semibold">{{ t('keyUsage.design.emptyTitle') }}</h2>
        <p class="max-w-md text-body-regular text-text-secondary">{{ t('keyUsage.design.emptyDescription') }}</p>
        <div class="mt-3 flex flex-wrap justify-center gap-2"><Badge>{{ t('keyUsage.walletBalance') }}</Badge><Badge>{{ t('keyUsage.totalQuota') }}</Badge><Badge>{{ t('keyUsage.dailyDetail') }}</Badge></div>
      </section>

      <div v-else-if="resultData" class="flex min-w-0 flex-col gap-6" data-testid="key-usage-results">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-baseline gap-3"><h2 class="text-title-2-semibold">{{ t('keyUsage.design.overview') }}</h2><p v-if="resultData.quota?.remaining != null" class="text-body-medium text-text-secondary">{{ t('keyUsage.remainingQuota') }} <span class="text-text-primary">{{ usd(resultData.quota.remaining) }}</span></p></div>
          <Badge v-if="statusInfo" :tone="statusInfo.isActive ? 'success' : 'danger'" dot>{{ statusInfo.label }} · {{ statusInfo.statusText }}</Badge>
        </div>
        <div v-if="ringItems.length" :class="ringGridClass" data-testid="key-usage-quotas">
          <article v-for="(ring, index) in ringItems" :key="ring.title" class="usage-card flex min-w-0 items-center justify-between gap-4 p-5 sm:p-6">
            <div class="flex min-w-0 flex-col gap-3">
              <p class="flex items-center gap-2 text-body-medium text-text-secondary"><Icon :name="ring.iconType === 'clock' ? 'clock' : ring.iconType === 'calendar' ? 'calendar' : 'creditCard'" size="sm" aria-hidden="true" />{{ ring.title }}</p>
              <p :class="cx('break-words tabular-nums text-text-primary', ring.isBalance ? 'text-display-4-medium' : 'text-title-2-semibold')">{{ ring.amount }}</p>
              <p v-if="ring.resetAt" class="text-caption-1-regular text-text-secondary">{{ t('keyUsage.design.resetsIn') }} {{ formatResetTime(ring.resetAt) }}</p>
              <span v-else class="text-caption-1-regular text-text-tertiary">{{ ring.isBalance ? t('keyUsage.design.availableBalance') : t('keyUsage.design.usedOfLimit') }}</span>
            </div>
            <div v-if="!ring.isBalance" class="relative h-20 w-20 shrink-0" role="img" :aria-label="`${ring.title}: ${ring.pct}% ${t('keyUsage.used')}`">
              <svg viewBox="0 0 160 160" class="h-full w-full -rotate-90" aria-hidden="true"><circle cx="80" cy="80" r="68" fill="none" stroke="rgb(var(--boardui-chart-track))" stroke-width="10" /><circle cx="80" cy="80" r="68" fill="none" :stroke="ringColors[index % ringColors.length]" stroke-width="10" stroke-linecap="round" :stroke-dasharray="CIRCUMFERENCE" :stroke-dashoffset="CIRCUMFERENCE * (1 - ring.pct / 100)" /></svg>
              <span class="absolute inset-0 flex items-center justify-center text-headline-semibold tabular-nums">{{ ring.pct }}%</span>
            </div>
            <span v-else class="flex h-12 w-12 shrink-0 items-center justify-center rounded-2xl bg-background-accent-subtle text-text-accent"><Icon name="creditCard" size="lg" aria-hidden="true" /></span>
          </article>
        </div>

        <div v-if="resultData.usage" class="grid gap-4 sm:grid-cols-3" data-testid="usage-headlines">
          <StatCard :label="t('keyUsage.todayCost')" :value="usd(resultData.usage.today?.actual_cost)" icon="dollar">{{ t('keyUsage.dateRangeToday') }}</StatCard>
          <StatCard :label="t('keyUsage.todayRequests')" :value="fmtNum(resultData.usage.today?.requests)" icon="sync">{{ t('keyUsage.dateRangeToday') }}</StatCard>
          <StatCard :label="t('keyUsage.totalCost')" :value="usd(resultData.usage.total?.actual_cost)" icon="chart">{{ t('keyUsage.charts.lifetime') }}</StatCard>
        </div>
        <KeyUsageCharts :daily="dailyUsageRows" :models="modelStats" :usage="resultData.usage" :days="dailyUsageDays" :trend-loading="dailyLoading" :trend-error="dailyError" @update:days="setDailyUsageDays" />
        <details class="usage-card min-w-0" data-testid="usage-details" @toggle="detailsOpen = ($event.target as HTMLDetailsElement).open">
          <summary class="cursor-pointer rounded-3xl p-5 text-body-medium outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring sm:px-6">{{ t('keyUsage.charts.details') }}<span class="ms-2 text-caption-1-regular text-text-secondary">{{ t('keyUsage.charts.detailsHint') }}</span></summary>
          <div v-if="detailsOpen" class="flex min-w-0 flex-col gap-4 border-t border-separator-border p-3 sm:p-5">
        <section v-if="detailRows.length" class="usage-card overflow-hidden" aria-labelledby="usage-detail-title">
          <div class="px-5 py-5 sm:px-6"><h2 id="usage-detail-title" class="text-headline-semibold">{{ t('keyUsage.detailInfo') }}</h2></div>
          <dl class="grid divide-y divide-separator-border border-t border-separator-border md:grid-cols-2 md:divide-y-0">
            <div v-for="(row, index) in detailRows" :key="index" class="flex min-w-0 items-center justify-between gap-4 px-5 py-4 sm:px-6">
              <dt class="flex shrink-0 items-center gap-2 text-body-regular text-text-secondary"><svg class="h-4 w-4 shrink-0 text-foreground-icon-secondary" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" v-html="row.iconSvg" />{{ row.label }}</dt>
              <dd :class="cx('min-w-0 break-words text-end text-body-medium tabular-nums', row.valueClass || 'text-text-primary')">{{ row.value }}</dd>
            </div>
          </dl>
        </section>

        <section v-if="usageStatCells.length" class="usage-card overflow-hidden" aria-labelledby="usage-token-title">
          <div class="px-5 py-5 sm:px-6"><h2 id="usage-token-title" class="text-headline-semibold">{{ t('keyUsage.tokenStats') }}</h2></div>
          <dl class="grid grid-cols-2 gap-px border-t border-separator-border bg-separator-border md:grid-cols-4">
            <div v-for="cell in usageStatCells" :key="cell.label" class="flex min-w-0 flex-col gap-2 bg-background-primary-default p-4 sm:px-6"><dt class="text-caption-1-regular text-text-secondary">{{ cell.label }}</dt><dd class="break-words text-title-3-medium tabular-nums">{{ cell.value }}</dd></div>
          </dl>
        </section>

        <section v-if="showDailyUsage" class="usage-card min-w-0 overflow-hidden" aria-labelledby="usage-daily-title">
          <h2 id="usage-daily-title" class="p-5 text-headline-semibold sm:px-6">{{ t('keyUsage.dailyDetail') }}</h2>
          <div v-if="dailyUsageRows.length" class="overflow-x-auto" tabindex="0" :aria-label="t('keyUsage.dailyDetail')">
            <Table size="sm"><caption class="sr-only">{{ t('keyUsage.dailyDetail') }}</caption><thead><tr><th v-for="column in dailyColumns" :key="column" scope="col">{{ t(`keyUsage.${column}`) }}</th></tr></thead><tbody><tr v-for="row in dailyUsageRows" :key="row.date"><td class="whitespace-nowrap">{{ row.date }}</td><td>{{ fmtNum(row.requests) }}</td><td>{{ fmtNum(row.input_tokens) }}</td><td>{{ fmtNum(row.output_tokens) }}</td><td>{{ fmtNum(row.cache_read_tokens) }}</td><td>{{ fmtNum(row.cache_write_tokens) }}</td><td>{{ usd(row.actual_cost != null ? row.actual_cost : row.cost) }}</td></tr></tbody></Table>
          </div>
          <p v-else class="border-t border-separator-border px-5 py-10 text-center text-body-regular text-text-secondary">{{ t('keyUsage.noDailyUsage') }}</p>
        </section>

        <section v-if="modelStats.length" class="usage-card min-w-0 overflow-hidden" aria-labelledby="usage-model-title">
          <h2 id="usage-model-title" class="p-5 text-headline-semibold sm:px-6">{{ t('keyUsage.modelStats') }}</h2>
          <div class="overflow-x-auto" tabindex="0" :aria-label="t('keyUsage.modelStats')"><Table size="sm"><caption class="sr-only">{{ t('keyUsage.modelStats') }}</caption><thead><tr><th v-for="column in modelColumns" :key="column" scope="col">{{ t(`keyUsage.${column}`) }}</th></tr></thead><tbody><tr v-for="(model, index) in modelStats" :key="index"><td class="max-w-64 break-words">{{ model.model || '-' }}</td><td>{{ fmtNum(model.requests) }}</td><td>{{ fmtNum(model.input_tokens) }}</td><td>{{ fmtNum(model.output_tokens) }}</td><td>{{ fmtNum(model.cache_creation_tokens) }}</td><td>{{ fmtNum(model.cache_read_tokens) }}</td><td>{{ fmtNum(model.total_tokens) }}</td><td>{{ usd(model.actual_cost != null ? model.actual_cost : model.cost) }}</td></tr></tbody></Table></div>
        </section>
          </div>
        </details>
      </div>
    </main>
    <footer class="border-t border-separator-border px-5 py-6 sm:px-8"><div class="mx-auto flex max-w-6xl flex-wrap justify-between gap-4 text-caption-1-regular text-text-secondary"><p class="break-all">&copy; {{ currentYear }} {{ siteName }}</p><span>{{ t('home.redesign.footer') }}</span></div></footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import Button from '@/components/boardui/Button.vue'
import Input from '@/components/boardui/Input.vue'
import Badge from '@/components/boardui/Badge.vue'
import Skeleton from '@/components/boardui/Skeleton.vue'
import Table from '@/components/boardui/Table.vue'
import StatCard from '@/components/boardui/StatCard.vue'
import KeyUsageCharts from '@/components/key-usage/KeyUsageCharts.vue'
import { cx } from '@/utils/cx'
import { buildGatewayUrl } from '@/api/client'
import { formatDateLocalInput } from '@/utils/format'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const appStore = useAppStore()
const subscriptionFeatureEnabled = computed(() => resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.subscription))

// ==================== Site Settings (same as HomeView) ====================

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))

// ==================== Theme (same as HomeView) ====================

const isDark = ref(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

const currentYear = computed(() => new Date().getFullYear())

// ==================== Key Query State ====================

const detailsOpen = ref(false)
const queryError = ref('')
const apiKey = ref('')
const isQuerying = ref(false)
const showLoading = ref(false)
const showDatePicker = ref(false)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const resultData = ref<any>(null)
const now = ref(new Date())
let resetTimer: ReturnType<typeof setInterval> | null = null

// ==================== Date Range State ====================

type DateRangeKey = 'today' | '7d' | '30d' | 'custom'
const currentRange = ref<DateRangeKey>('today')
const customStartDate = ref('')
const customEndDate = ref('')
const dailyUsageDays = ref<7 | 30 | 90>(30)
const dailyLoading = ref(false)
const dailyError = ref('')
let loadedDailyDays: 7 | 30 | 90 = 30
let dailyRequestVersion = 0
let dailyController: AbortController | null = null
let lastQuery: { key: string; params: string } | null = null

const dateRanges = computed(() => [
  { key: 'today' as const, label: t('keyUsage.dateRangeToday') },
  { key: '7d' as const, label: t('keyUsage.dateRange7d') },
  { key: '30d' as const, label: t('keyUsage.dateRange30d') },
  { key: 'custom' as const, label: t('keyUsage.dateRangeCustom') },
])

function setDateRange(key: DateRangeKey) {
  currentRange.value = key
  if (key !== 'custom') {
    queryKey()
  }
}

function getDateParams(): string {
  const now = new Date()
  const params = new URLSearchParams()

  if (currentRange.value === 'custom') {
    if (customStartDate.value && customEndDate.value) {
      params.set('start_date', customStartDate.value)
      params.set('end_date', customEndDate.value)
    }
  } else {
    const end = formatDateLocalInput(now)
    let start: string
    switch (currentRange.value) {
      case 'today': start = end; break
      case '7d': start = formatDateLocalInput(new Date(now.getTime() - 7 * 86400000)); break
      case '30d': start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000)); break
      default: start = formatDateLocalInput(new Date(now.getTime() - 30 * 86400000))
    }
    params.set('start_date', start)
    params.set('end_date', end)
  }
  params.set('days', String(dailyUsageDays.value))
  params.set('timezone', getBrowserTimezone())
  return params.toString()
}

async function setDailyUsageDays(days: 7 | 30 | 90) {
  if (dailyUsageDays.value === days || dailyLoading.value || isQuerying.value || !resultData.value || !lastQuery) return
  const version = ++dailyRequestVersion
  const controller = new AbortController()
  dailyController = controller
  const query = lastQuery
  const params = new URLSearchParams(query.params)
  params.set('days', String(days))
  dailyUsageDays.value = days
  dailyLoading.value = true
  dailyError.value = ''
  try {
    const data = await fetchUsage(query.key, params.toString(), controller.signal)
    if (version !== dailyRequestVersion) return
    if (!Array.isArray(data.daily_usage)) throw new Error(t('keyUsage.queryFailedRetry'))
    resultData.value = { ...resultData.value, daily_usage: data.daily_usage }
    loadedDailyDays = days
  } catch (err) {
    if (version !== dailyRequestVersion) return
    dailyUsageDays.value = loadedDailyDays
    dailyError.value = (err as Error).message || t('keyUsage.queryFailedRetry')
  } finally {
    if (version === dailyRequestVersion) {
      dailyLoading.value = false
      dailyController = null
    }
  }
}

// ==================== Quota Rings ====================

const CIRCUMFERENCE = 2 * Math.PI * 68
const ringColors = ['rgb(var(--boardui-background-accent-default))', 'rgb(var(--boardui-chart-1))', 'rgb(var(--boardui-chart-5))', 'rgb(var(--boardui-chart-4))']

interface RingItem {
  title: string
  pct: number
  amount: string
  isBalance?: boolean
  iconType: 'clock' | 'calendar' | 'dollar'
  resetAt?: string | null
}

// ==================== Computed Data ====================

const statusInfo = computed(() => {
  const data = resultData.value
  if (!data) return null

  if (data.mode === 'quota_limited') {
    const isValid = data.isValid !== false
    const statusMap: Record<string, string> = {
      active: t('keyUsage.design.active'),
      quota_exhausted: t('keyUsage.design.exhausted'),
      expired: t('keyUsage.design.expired'),
    }
    return {
      label: t('keyUsage.quotaMode'),
      statusText: statusMap[data.status] || data.status || t('keyUsage.design.unknown'),
      isActive: isValid && data.status === 'active',
    }
  }

  return {
    label: data.planName || t('keyUsage.walletBalance'),
    statusText: t('keyUsage.design.active'),
    isActive: true,
  }
})

const ringItems = computed<RingItem[]>(() => {
  const data = resultData.value
  if (!data) return []

  const items: RingItem[] = []

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const pct = data.quota.limit > 0 ? Math.min(Math.round((data.quota.used / data.quota.limit) * 100), 100) : 0
      items.push({ title: t('keyUsage.totalQuota'), pct, amount: `${usd(data.quota.used)} / ${usd(data.quota.limit)}`, iconType: 'dollar' })
    }
    if (data.rate_limits) {
      const windowLabels: Record<string, string> = { '5h': t('keyUsage.limit5h'), '1d': t('keyUsage.limitDaily'), '7d': t('keyUsage.limit7d') }
      const windowIcons: Record<string, 'clock' | 'calendar'> = { '5h': 'clock', '1d': 'calendar', '7d': 'calendar' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? Math.min(Math.round((rl.used / rl.limit) * 100), 100) : 0
        items.push({
          title: windowLabels[rl.window] || rl.window,
          pct,
          amount: `${usd(rl.used)} / ${usd(rl.limit)}`,
          iconType: windowIcons[rl.window] || 'clock',
          resetAt: rl.reset_at,
        })
      }
    }
  } else {
    if (data.subscription) {
      const sub = data.subscription
      const limits = [
        { label: t('keyUsage.limitDaily'), usage: sub.daily_usage_usd, limit: sub.daily_limit_usd },
        { label: t('keyUsage.limitWeekly'), usage: sub.weekly_usage_usd, limit: sub.weekly_limit_usd },
        { label: t('keyUsage.limitMonthly'), usage: sub.monthly_usage_usd, limit: sub.monthly_limit_usd },
      ]
      for (const l of limits) {
        if (l.limit != null && l.limit > 0) {
          const pct = Math.min(Math.round((l.usage / l.limit) * 100), 100)
          items.push({ title: l.label, pct, amount: `${usd(l.usage)} / ${usd(l.limit)}`, iconType: 'calendar' })
        }
      }
    }
    if (!data.subscription && data.balance != null) {
      items.push({ title: t('keyUsage.walletBalance'), pct: 0, amount: usd(data.balance), isBalance: true, iconType: 'dollar' })
    }
  }

  return items
})

const ringGridClass = computed(() => {
  const len = ringItems.value.length
  if (len === 1) return 'grid grid-cols-1 gap-4'
  if (len === 2) return 'grid grid-cols-1 md:grid-cols-2 gap-4'
  return 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4'
})

interface DetailRow {
  iconBg: string
  iconColor: string
  iconSvg: string
  label: string
  value: string
  valueClass: string
}

function getUsageColor(pct: number): string {
  if (pct > 90) return 'text-text-error-primary'
  if (pct > 70) return 'text-text-warning'
  return 'text-text-success'
}

const detailRows = computed<DetailRow[]>(() => {
  const data = resultData.value
  if (!data) return []

  const rows: DetailRow[] = []
  const ICON_SHIELD = '<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'
  const ICON_CALENDAR = '<rect x="3" y="4" width="18" height="18" rx="2" ry="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/>'
  const ICON_DOLLAR = '<line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>'
  const ICON_CHECK = '<polyline points="20 6 9 17 4 12"/>'

  if (data.mode === 'quota_limited') {
    if (data.quota) {
      const remainColor = data.quota.remaining <= 0 ? 'text-text-error-primary'
        : data.quota.remaining < data.quota.limit * 0.1 ? 'text-text-warning'
        : 'text-text-success'
      rows.push({
        iconBg: 'bg-background-success-subtle', iconColor: 'text-text-success', iconSvg: ICON_SHIELD,
        label: t('keyUsage.remainingQuota'), value: usd(data.quota.remaining), valueClass: remainColor,
      })
    }
    if (data.expires_at) {
      const daysLeft = data.days_until_expiry
      let expiryStr = formatDate(data.expires_at)
      if (daysLeft != null) {
        expiryStr += daysLeft > 0 ? ` ${t('keyUsage.daysLeft', { days: daysLeft })}` : daysLeft === 0 ? ` ${t('keyUsage.todayExpires')}` : ''
      }
      rows.push({
        iconBg: 'bg-background-warning-subtle', iconColor: 'text-text-warning', iconSvg: ICON_CALENDAR,
        label: t('keyUsage.expiresAt'), value: expiryStr, valueClass: '',
      })
    }
    if (data.rate_limits) {
      const windowMap: Record<string, string> = { '5h': '5H', '1d': locale.value === 'zh' ? '日' : 'D', '7d': '7D' }
      for (const rl of data.rate_limits) {
        const pct = rl.limit > 0 ? (rl.used / rl.limit) * 100 : 0
        let valueStr = `${usd(rl.used)} / ${usd(rl.limit)}`
        const resetStr = formatResetTime(rl.reset_at)
        if (resetStr) {
          valueStr += ` (⟳ ${resetStr})`
        }
        rows.push({
          iconBg: 'bg-background-accent-subtle', iconColor: 'text-text-accent', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${windowMap[rl.window] || rl.window})`,
          value: valueStr,
          valueClass: getUsageColor(pct),
        })
      }
    }
  } else {
    rows.push({
      iconBg: 'bg-background-success-subtle', iconColor: 'text-text-success', iconSvg: ICON_CHECK,
      // 订阅功能关闭后这一行只会是「钱包余额」，标签改用不带「订阅」字样的「计费方式」。
      label: subscriptionFeatureEnabled.value ? t('keyUsage.subscriptionType') : t('keyUsage.billingType'),
      value: data.planName || t('keyUsage.walletBalance'), valueClass: '',
    })

    if (data.subscription) {
      const sub = data.subscription
      if (sub.daily_limit_usd > 0) {
        const pct = (sub.daily_usage_usd / sub.daily_limit_usd) * 100
        rows.push({
          iconBg: 'bg-background-accent-subtle', iconColor: 'text-text-accent', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '日' : 'D'})`, value: `${usd(sub.daily_usage_usd)} / ${usd(sub.daily_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.weekly_limit_usd > 0) {
        const pct = (sub.weekly_usage_usd / sub.weekly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-background-feature-subtle', iconColor: 'text-text-feature', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '周' : 'W'})`, value: `${usd(sub.weekly_usage_usd)} / ${usd(sub.weekly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.monthly_limit_usd > 0) {
        const pct = (sub.monthly_usage_usd / sub.monthly_limit_usd) * 100
        rows.push({
          iconBg: 'bg-background-success-subtle', iconColor: 'text-text-success', iconSvg: ICON_DOLLAR,
          label: `${t('keyUsage.usedQuota')} (${locale.value === 'zh' ? '月' : 'M'})`, value: `${usd(sub.monthly_usage_usd)} / ${usd(sub.monthly_limit_usd)}`, valueClass: getUsageColor(pct),
        })
      }
      if (sub.expires_at) {
        rows.push({
          iconBg: 'bg-background-warning-subtle', iconColor: 'text-text-warning', iconSvg: ICON_CALENDAR,
          label: t('keyUsage.subscriptionExpires'), value: formatDate(sub.expires_at), valueClass: '',
        })
      }
    }

    const remainColor = data.remaining != null
      ? (data.remaining <= 0 ? 'text-text-error-primary' : data.remaining < 10 ? 'text-text-warning' : 'text-text-success')
      : ''
    rows.push({
      iconBg: 'bg-background-success-subtle', iconColor: 'text-text-success', iconSvg: ICON_SHIELD,
      label: t('keyUsage.remainingQuota'), value: data.remaining != null ? usd(data.remaining) : '-', valueClass: remainColor,
    })
  }

  return rows
})

interface StatCell {
  label: string
  value: string
}

const usageStatCells = computed<StatCell[]>(() => {
  const usage = resultData.value?.usage
  if (!usage) return []

  const today = usage.today || {}
  const total = usage.total || {}

  return [
    { label: t('keyUsage.todayRequests'), value: fmtNum(today.requests) },
    { label: t('keyUsage.todayInputTokens'), value: fmtNum(today.input_tokens) },
    { label: t('keyUsage.todayOutputTokens'), value: fmtNum(today.output_tokens) },
    { label: t('keyUsage.todayTokens'), value: fmtNum(today.total_tokens) },
    { label: t('keyUsage.todayCacheCreation'), value: fmtNum(today.cache_creation_tokens) },
    { label: t('keyUsage.todayCacheRead'), value: fmtNum(today.cache_read_tokens) },
    { label: t('keyUsage.todayCost'), value: usd(today.actual_cost) },
    { label: t('keyUsage.rpmTpm'), value: `${usage.rpm || 0} / ${usage.tpm || 0}` },
    { label: t('keyUsage.totalRequests'), value: fmtNum(total.requests) },
    { label: t('keyUsage.totalInputTokens'), value: fmtNum(total.input_tokens) },
    { label: t('keyUsage.totalOutputTokens'), value: fmtNum(total.output_tokens) },
    { label: t('keyUsage.totalTokensLabel'), value: fmtNum(total.total_tokens) },
    { label: t('keyUsage.totalCacheCreation'), value: fmtNum(total.cache_creation_tokens) },
    { label: t('keyUsage.totalCacheRead'), value: fmtNum(total.cache_read_tokens) },
    { label: t('keyUsage.totalCost'), value: usd(total.actual_cost) },
    { label: t('keyUsage.avgDuration'), value: usage.average_duration_ms ? `${Math.round(usage.average_duration_ms)} ms` : '-' },
  ]
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const modelStats = computed<any[]>(() => resultData.value?.model_stats || [])

interface DailyUsageRow {
  date: string
  requests: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  cache_write_tokens: number
  cost: number
  actual_cost?: number
}

const dailyUsageRows = computed<DailyUsageRow[]>(() => {
  const rows = resultData.value?.daily_usage
  return Array.isArray(rows) ? rows : []
})

const showDailyUsage = computed(() => Boolean(resultData.value && Array.isArray(resultData.value.daily_usage)))

const dailyColumns = ['date', 'requests', 'inputTokens', 'outputTokens', 'cacheReadTokens', 'cacheWriteTokens', 'cost']
const modelColumns = ['model', 'requests', 'inputTokens', 'outputTokens', 'cacheCreationTokens', 'cacheReadTokens', 'totalTokens', 'cost']

// ==================== Utility Functions ====================

function usd(value: number | null | undefined): string {
  if (value == null || value < 0) return '-'
  return '$' + Number(value).toFixed(2)
}

function fmtNum(val: number | null | undefined): string {
  if (val == null) return '-'
  return val.toLocaleString()
}

function formatDate(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = new Date(iso)
  const loc = locale.value === 'zh' ? 'zh-CN' : 'en-US'
  return d.toLocaleDateString(loc, { year: 'numeric', month: 'long', day: 'numeric' })
}

function getBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    return 'UTC'
  }
}

// ==================== API Query ====================

async function fetchUsage(key: string, dateParams: string, signal?: AbortSignal) {
  const url = buildGatewayUrl('/v1/usage') + (dateParams ? '?' + dateParams : '')
  const res = await fetch(url, {
    headers: { 'Authorization': 'Bearer ' + key },
    signal,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => null)
    const msg = body?.error?.message || body?.message || `${t('keyUsage.queryFailed')} (${res.status})`
    throw new Error(msg)
  }
  return await res.json()
}

async function queryKey() {
  if (isQuerying.value) return
  const key = apiKey.value.trim()
  if (!key) {
    queryError.value = t('keyUsage.enterApiKey')
    return
  }

  if (currentRange.value === 'custom' && (!customStartDate.value || !customEndDate.value || customStartDate.value > customEndDate.value)) {
    queryError.value = t('keyUsage.design.invalidDates')
    return
  }

  dailyRequestVersion++
  dailyController?.abort()
  dailyController = null
  dailyLoading.value = false
  dailyError.value = ''
  lastQuery = null
  const dateParams = getDateParams()
  detailsOpen.value = false
  isQuerying.value = true
  queryError.value = ''
  showLoading.value = true
  resultData.value = null

  try {
    const data = await fetchUsage(key, dateParams)
    lastQuery = { key, params: dateParams }
    loadedDailyDays = dailyUsageDays.value
    resultData.value = data
    showLoading.value = false
    showDatePicker.value = true
  } catch (err) {
    showLoading.value = false
    queryError.value = (err as Error).message || t('keyUsage.queryFailedRetry')
  } finally {
    isQuerying.value = false
  }
}

// ==================== Lifecycle ====================

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  isDark.value = savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', isDark.value)
}

function formatResetTime(resetAt: string | null | undefined): string {
  if (!resetAt) return ''
  const diff = new Date(resetAt).getTime() - now.value.getTime()
  if (diff <= 0) return t('keyUsage.resetNow')
  const days = Math.floor(diff / 86400000)
  const hours = Math.floor((diff % 86400000) / 3600000)
  const mins = Math.floor((diff % 3600000) / 60000)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

onMounted(() => {
  initTheme()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
  resetTimer = setInterval(() => { now.value = new Date() }, 60000)
})

onUnmounted(() => {
  dailyRequestVersion++
  dailyController?.abort()
  lastQuery = null
  if (resetTimer) clearInterval(resetTimer)
})
</script>

<style scoped>
.usage-card {
  @apply rounded-3xl border border-border-button-default bg-background-primary-default;
}
.usage-link {
  @apply rounded text-text-secondary outline-none transition-colors duration-150 hover:text-text-primary focus-visible:ring-2 focus-visible:ring-border-focus-ring;
}
.usage-back {
  @apply inline-flex min-h-9 items-center justify-center gap-2 rounded-xl border border-border-button-default bg-background-primary-default px-3 py-2 text-body-medium shadow-sm outline-none transition-colors duration-150 hover:bg-background-primary-hover focus-visible:ring-2 focus-visible:ring-border-focus-ring;
}
:deep(.bui-table tbody td:not(:first-child)) {
  @apply whitespace-nowrap tabular-nums;
}
</style>
