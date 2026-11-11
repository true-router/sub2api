<template>
  <div class="grid min-w-0 gap-4 lg:grid-cols-3" data-testid="key-usage-charts">
    <section :aria-busy="trendLoading || undefined" class="chart-card lg:col-span-2" aria-labelledby="usage-trend-title">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="flex flex-col gap-1"><h2 id="usage-trend-title" class="text-headline-semibold">{{ t('keyUsage.charts.trendTitle') }}</h2><p class="text-caption-1-regular text-text-secondary">{{ t('keyUsage.charts.dailyScope', { days }) }}</p></div>
        <SegmentedControl v-model="trendMetric" :label="t('keyUsage.charts.trendMetric')" :items="trendMetrics" />
      </div>
      <div class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-title-1-semibold tabular-nums" data-testid="usage-trend-total">{{ formatValue(trendTotal, trendMetric) }}<span class="ms-2 text-caption-1-regular text-text-secondary">{{ t('keyUsage.charts.periodTotal') }}</span></p>
        <div class="flex gap-1"><Button v-for="period in [7, 30, 90] as const" :key="period" size="small" :variant="days === period ? 'secondary' : 'ghost'" :aria-pressed="days === period" :disabled="trendLoading" @click="emit('update:days', period)">{{ t(`keyUsage.dateRange${period}d`) }}</Button></div>
      </div>
      <p v-if="trendError" role="alert" data-testid="usage-trend-error" class="text-body-regular text-text-error-primary">{{ trendError }}</p>
      <div class="relative">
      <div v-if="orderedDays.length" class="relative h-56 min-w-0 sm:h-64"><Line :data="trendData" :options="trendOptions" role="img" :aria-label="t('keyUsage.charts.trendAccessible', { metric: trendMetricLabel, total: formatValue(trendTotal, trendMetric) })" /></div>
      <p v-else class="chart-empty">{{ t('keyUsage.noDailyUsage') }}</p>
      <div v-if="trendLoading" role="status" :aria-label="t('keyUsage.querying')" data-testid="usage-trend-loading" class="absolute inset-0 flex flex-col justify-center gap-5 bg-background-primary-default/95 p-4"><Skeleton width="70%" :height="16" /><Skeleton :height="100" /><Skeleton width="85%" :height="12" /></div>
      </div>
    </section>

    <section class="chart-card" aria-labelledby="usage-model-chart-title">
      <div class="flex flex-wrap items-start justify-between gap-3"><div class="flex flex-col gap-1"><h2 id="usage-model-chart-title" class="text-headline-semibold">{{ t('keyUsage.charts.modelsTitle') }}</h2><p class="text-caption-1-regular text-text-secondary">{{ t('keyUsage.charts.selectedScope') }}</p></div><SegmentedControl v-model="modelMetric" :label="t('keyUsage.charts.modelMetric')" :items="modelMetrics" /></div>
      <template v-if="modelTotal > 0">
        <div class="relative mx-auto h-44 w-44"><Doughnut :data="modelData" :options="modelOptions" role="img" :aria-label="t('keyUsage.charts.modelsAccessible', { metric: modelMetricLabel, total: formatValue(modelTotal, modelMetric) })" /><div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-1"><span class="text-title-2-semibold tabular-nums" data-testid="usage-model-total">{{ formatValue(modelTotal, modelMetric) }}</span><span class="text-caption-1-regular text-text-secondary">{{ modelMetricLabel }}</span></div></div>
        <ul class="flex flex-col gap-3" data-testid="usage-model-legend"><li v-for="(item, index) in groupedModels" :key="index" class="flex min-w-0 items-center gap-2 text-body-2-regular"><span class="h-2 w-2 shrink-0 rounded-full" :style="{ backgroundColor: chartColors[index % chartColors.length] }" aria-hidden="true" /><span class="min-w-0 flex-1 truncate" :title="item.label">{{ item.label }}</span><span class="tabular-nums text-text-secondary">{{ Math.round(item.value / modelTotal * 100) }}%</span></li></ul>
      </template>
      <p v-else class="chart-empty">{{ t('keyUsage.charts.noModelUsage') }}</p>
    </section>

    <section v-if="usage" class="chart-card lg:col-span-3" aria-labelledby="usage-token-chart-title">
      <div class="flex flex-wrap items-start justify-between gap-4"><div class="flex flex-col gap-1"><h2 id="usage-token-chart-title" class="text-headline-semibold">{{ t('keyUsage.charts.tokensTitle') }}</h2><p class="text-caption-1-regular text-text-secondary">{{ t('keyUsage.charts.tokensDescription') }}</p></div><SegmentedControl v-model="tokenScope" :label="t('keyUsage.charts.tokenScope')" :items="[{ value: 'today', label: t('keyUsage.dateRangeToday') }, { value: 'total', label: t('keyUsage.charts.lifetime') }]" /></div>
      <div v-if="tokenValues.some(value => value > 0)" class="relative h-52 min-w-0"><Bar :data="tokenData" :options="tokenOptions" role="img" :aria-label="t('keyUsage.charts.tokensAccessible', { scope: tokenScope === 'today' ? t('keyUsage.dateRangeToday') : t('keyUsage.charts.lifetime') })" /></div>
      <p v-else class="chart-empty">{{ t('keyUsage.charts.noTokenUsage') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, ArcElement, BarElement, Tooltip, Filler, type ChartOptions } from 'chart.js'
import { Line, Doughnut, Bar } from 'vue-chartjs'
import Button from '@/components/boardui/Button.vue'
import Skeleton from '@/components/boardui/Skeleton.vue'
import SegmentedControl from '@/components/boardui/SegmentedControl.vue'
import { useBoardUIChartTheme } from '@/composables/useBoardUIChartTheme'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, ArcElement, BarElement, Tooltip, Filler)
interface TokenCounts { input_tokens?: number; output_tokens?: number; cache_creation_tokens?: number; cache_read_tokens?: number }
interface DailyUsage { date: string; requests?: number; input_tokens?: number; output_tokens?: number; cost?: number; actual_cost?: number }
interface ModelUsage { model?: string; requests?: number; cost?: number; actual_cost?: number }
const props = defineProps<{ daily: DailyUsage[]; models: ModelUsage[]; days: 7 | 30 | 90; usage?: { today?: TokenCounts; total?: TokenCounts }; trendLoading?: boolean; trendError?: string }>()
const emit = defineEmits<{ 'update:days': [days: 7 | 30 | 90] }>()
const { t, locale } = useI18n()
const theme = useBoardUIChartTheme()
const chartColors = computed(() => [theme.value.series[3], theme.value.series[0], theme.value.series[4], theme.value.series[2], theme.value.series[1], theme.value.neutral])
const trendMetric = ref('cost')
const modelMetric = ref('cost')
const tokenScope = ref('today')
const trendMetrics = computed(() => [{ value: 'cost', label: t('keyUsage.cost') }, { value: 'requests', label: t('keyUsage.requests') }, { value: 'tokens', label: 'Tokens' }])
const modelMetrics = computed(() => trendMetrics.value.slice(0, 2))
const trendMetricLabel = computed(() => trendMetrics.value.find(item => item.value === trendMetric.value)?.label || '')
const modelMetricLabel = computed(() => modelMetrics.value.find(item => item.value === modelMetric.value)?.label || '')
const numeric = (value: unknown) => typeof value === 'number' && Number.isFinite(value) ? Math.max(0, value) : 0
const compact = (value: number) => new Intl.NumberFormat(locale.value, { notation: 'compact', maximumFractionDigits: 1 }).format(value)
function formatValue(value: number, metric: string) {
  return metric === 'cost' ? new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: 6 }).format(value) : new Intl.NumberFormat(locale.value).format(value)
}
const orderedDays = computed(() => [...props.daily].sort((a, b) => a.date.localeCompare(b.date)))
const trendValues = computed(() => orderedDays.value.map(row => trendMetric.value === 'cost' ? numeric(row.actual_cost ?? row.cost) : trendMetric.value === 'requests' ? numeric(row.requests) : numeric(row.input_tokens) + numeric(row.output_tokens)))
const trendTotal = computed(() => trendValues.value.reduce((sum, value) => sum + value, 0))
const trendData = computed(() => ({ labels: orderedDays.value.map(row => row.date), datasets: [{ label: trendMetricLabel.value, data: trendValues.value, borderColor: chartColors.value[0], backgroundColor: chartColors.value[0] + '18', fill: true, tension: 0, pointRadius: orderedDays.value.length < 10 ? 4 : 0, pointHoverRadius: 6, borderWidth: 2 }] }))
const trendOptions = computed<ChartOptions<'line'>>(() => ({
  responsive: true, maintainAspectRatio: false, animation: false,
  interaction: { intersect: false, mode: 'index' },
  plugins: { legend: { display: false }, tooltip: { callbacks: { label: context => `${context.dataset.label}: ${formatValue(Number(context.raw), trendMetric.value)}` } } },
  scales: { x: { grid: { display: false }, border: { display: false }, ticks: { color: theme.value.text, maxTicksLimit: 6, maxRotation: 0, callback: (_value, index) => orderedDays.value[index]?.date.slice(5) } }, y: { beginAtZero: true, grid: { color: theme.value.grid }, border: { display: false }, ticks: { color: theme.value.text, maxTicksLimit: 5, precision: trendMetric.value === 'requests' ? 0 : undefined, callback: value => trendMetric.value === 'cost' ? '$' + compact(Number(value)) : compact(Number(value)) } } },
}))
const groupedModels = computed(() => {
  const totals = new Map<string, number>()
  for (const row of props.models) {
    const label = row.model || t('keyUsage.design.unknown')
    const value = modelMetric.value === 'cost' ? numeric(row.actual_cost ?? row.cost) : numeric(row.requests)
    totals.set(label, (totals.get(label) || 0) + value)
  }
  const rows = [...totals].map(([label, value]) => ({ label, value })).filter(row => row.value > 0).sort((a, b) => b.value - a.value)
  if (rows.length <= 6) return rows
  return [...rows.slice(0, 5), { label: t('keyUsage.charts.otherModels'), value: rows.slice(5).reduce((sum, row) => sum + row.value, 0) }]
})
const modelTotal = computed(() => groupedModels.value.reduce((sum, row) => sum + row.value, 0))
const modelData = computed(() => ({ labels: groupedModels.value.map(row => row.label), datasets: [{ data: groupedModels.value.map(row => row.value), backgroundColor: groupedModels.value.map((_row, index) => chartColors.value[index % chartColors.value.length]), borderWidth: 0, hoverOffset: 3 }] }))
const modelOptions = computed<ChartOptions<'doughnut'>>(() => ({ responsive: true, maintainAspectRatio: false, animation: false, cutout: '78%', plugins: { legend: { display: false }, tooltip: { callbacks: { label: context => `${context.label}: ${formatValue(Number(context.raw), modelMetric.value)} (${Math.round(Number(context.raw) / modelTotal.value * 100)}%)` } } } }))
const tokenLabels = computed(() => [t('keyUsage.inputTokens'), t('keyUsage.outputTokens'), t('keyUsage.cacheCreationTokens'), t('keyUsage.cacheReadTokens')])
const tokenValues = computed(() => {
  const stats = tokenScope.value === 'today' ? props.usage?.today : props.usage?.total
  return [stats?.input_tokens, stats?.output_tokens, stats?.cache_creation_tokens, stats?.cache_read_tokens].map(numeric)
})
const tokenData = computed(() => ({ labels: tokenLabels.value, datasets: [{ label: 'Tokens', data: tokenValues.value, backgroundColor: chartColors.value.slice(0, 4), borderRadius: 6, barThickness: 16 }] }))
const tokenOptions = computed<ChartOptions<'bar'>>(() => ({ responsive: true, maintainAspectRatio: false, animation: false, indexAxis: 'y', plugins: { legend: { display: false }, tooltip: { callbacks: { label: context => formatValue(Number(context.raw), 'tokens') + ' Tokens' } } }, scales: { x: { beginAtZero: true, grid: { color: theme.value.grid }, border: { display: false }, ticks: { color: theme.value.text, maxTicksLimit: 4, callback: value => compact(Number(value)) } }, y: { grid: { display: false }, border: { display: false }, ticks: { color: theme.value.text, autoSkip: false } } } }))
</script>

<style scoped>
.chart-card { @apply flex min-w-0 flex-col gap-5 rounded-3xl border border-border-button-default bg-background-primary-default p-5 sm:p-6; }
.chart-empty { @apply flex min-h-52 items-center justify-center text-center text-body-regular text-text-secondary; }
</style>
