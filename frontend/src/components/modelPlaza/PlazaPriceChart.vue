<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, CategoryScale, LinearScale, BarElement, Tooltip, type ChartOptions } from 'chart.js'
import { Bar } from 'vue-chartjs'
import BoardButton from '@/components/boardui/Button.vue'
import { useBoardUIChartTheme } from '@/composables/useBoardUIChartTheme'
import type { ModelPlazaGroup, PlazaExchangeRate, PlazaModel } from '@/api/modelPlaza'
import { effectiveRate, money, priceRows, savings, savingsPercent, validQuote, type DisplayCurrency } from './plazaPricing'
ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip)
const props = defineProps<{ group: ModelPlazaGroup; model: PlazaModel; currency: DisplayCurrency; cnyPerCredit: number; quote?: PlazaExchangeRate | null }>()
const { t } = useI18n()
const theme = useBoardUIChartTheme()
const metric = ref('input')
const row = computed(() => priceRows(props.model).find(item => item.label === metric.value))
const credits = computed(() => row.value?.credits == null ? null : row.value.credits * effectiveRate(props.group, props.model) * row.value.scale)
const officialUSD = computed(() => row.value?.official == null ? null : row.value.official * row.value.scale)
const saving = computed(() => props.group.subscription_type === 'subscription' ? null : savings(credits.value, officialUSD.value, props.cnyPerCredit, props.quote))
const own = computed(() => credits.value == null || !validQuote(props.quote) ? null : credits.value * props.cnyPerCredit / (props.currency === 'USD' ? props.quote.usd_to_cny : 1))
const official = computed(() => officialUSD.value == null || !validQuote(props.quote) ? null : officialUSD.value * (props.currency === 'CNY' ? props.quote.usd_to_cny : 1))
const chartData = computed(() => ({
  labels: [t('modelPlaza.compare.chartOurs'), t('modelPlaza.compare.chartOfficial')],
  datasets: [{
    label: t(`modelPlaza.compare.labels.${metric.value}`),
    data: [own.value ?? 0, official.value ?? 0],
    backgroundColor: [theme.value.series[0], theme.value.neutral],
    borderRadius: 6,
    borderSkipped: false as const,
    barThickness: 24
  }]
}))
const options = computed<ChartOptions<'bar'>>(() => ({
  indexAxis: 'y', responsive: true, maintainAspectRatio: false, animation: false,
  plugins: { legend: { display: false }, tooltip: { callbacks: { label: item => money(item.parsed.x, props.currency) } } },
  scales: {
    x: { beginAtZero: true, grid: { color: theme.value.grid }, border: { display: false }, ticks: { color: theme.value.text, maxTicksLimit: 4, callback: value => money(Number(value), props.currency) } },
    y: { grid: { display: false }, border: { display: false }, ticks: { color: theme.value.text, font: { size: 12 } } }
  }
}))
const chartLabel = computed(() => t('modelPlaza.compare.chartDescription', {
  metric: t(`modelPlaza.compare.labels.${metric.value}`), ours: money(own.value, props.currency), official: money(official.value, props.currency)
}))
</script>

<template>
  <section class="mx-4 mb-4 flex min-w-0 flex-col gap-4 rounded-2xl border border-border-button-default bg-background-secondary-default p-4 sm:mx-5" :aria-label="t('modelPlaza.compare.chartTitle')" data-testid="price-chart" :data-our-value="own" :data-official-value="official">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <h3 class="text-body-2-medium text-text-secondary">{{ t('modelPlaza.compare.chartTitle') }}</h3>
      <div class="flex items-center gap-1" role="group" :aria-label="t('modelPlaza.compare.chartMetric')">
        <BoardButton v-for="key in ['input', 'output']" :key="key" :variant="metric === key ? 'secondary' : 'ghost'" size="xs" :aria-pressed="metric === key" :data-testid="`metric-${key}`" @click="metric = key">{{ t(`modelPlaza.compare.labels.${key}`) }}</BoardButton>
      </div>
    </div>
    <template v-if="saving != null && own != null && official != null">
      <div class="flex flex-col gap-1" aria-live="polite">
        <p class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.metricSaving', { metric: t(`modelPlaza.compare.labels.${metric}`) }) }}<span v-if="row?.tier"> · {{ row.tier }}</span></p>
        <p v-if="saving > 0" class="text-display-4-medium tabular-nums text-text-success" data-testid="chart-saving">{{ savingsPercent(saving) }}<span class="text-title-3-semibold">%</span></p>
        <p v-else class="text-title-3-semibold text-text-secondary">{{ t('modelPlaza.compare.noSaving') }}</p>
        <p v-if="saving > 0" class="text-body-2-regular text-text-secondary">{{ t('modelPlaza.compare.absoluteSaving', { amount: money(official - own, currency) }) }}</p>
      </div>
      <div class="relative h-32 min-w-0"><Bar :data="chartData" :options="options" role="img" :aria-label="chartLabel" /></div>
      <dl class="grid grid-cols-2 gap-3 border-t border-separator-border pt-3">
        <div class="min-w-0"><dt class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.ourPrice') }}</dt><dd class="mt-1 break-words text-title-3-semibold tabular-nums text-text-primary">{{ money(own, currency) }}</dd></div>
        <div class="min-w-0"><dt class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.official') }}</dt><dd class="mt-1 break-words text-title-3-semibold tabular-nums text-text-tertiary">{{ money(official, currency) }}</dd></div>
      </dl>
      <p class="text-caption-1-regular text-text-tertiary">{{ t('modelPlaza.compare.chartBasis') }}</p>
    </template>
    <p v-else class="py-4 text-body-2-regular text-text-tertiary">{{ t('modelPlaza.compare.chartUnavailable') }}</p>
  </section>
</template>
