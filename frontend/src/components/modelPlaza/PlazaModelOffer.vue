<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardTable from '@/components/boardui/Table.vue'
import Badge from '@/components/boardui/Badge.vue'
import type { ModelPlazaGroup, PlazaExchangeRate, PlazaModel } from '@/api/modelPlaza'
import { effectiveRate, priceRows, type DisplayCurrency } from './plazaPricing'
import PlazaPriceRow from './PlazaPriceRow.vue'
import { formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import { useAppStore } from '@/stores/app'
const props = defineProps<{ group: ModelPlazaGroup; model: PlazaModel; currency: DisplayCurrency; cnyPerCredit: number; quote?: PlazaExchangeRate | null }>()
const { t } = useI18n()
const appStore = useAppStore()
const rows = computed(() => priceRows(props.model))
const firstTier = computed(() => rows.value[0]?.tier)
const primary = computed(() => rows.value.filter(row => ['input', 'output'].includes(row.label) && row.tier === firstTier.value))
const details = computed(() => rows.value.filter(row => !primary.value.includes(row)))
const rate = computed(() => effectiveRate(props.group, props.model))
const subscription = computed(() => props.group.subscription_type === 'subscription')
const mediaMode = computed(() => props.model.pricing?.billing_mode || 'token')
const periods = computed(() => props.model.time_pricing?.periods ?? [])
const peakWindow = computed(() => props.group.peak_rate_enabled ? formatPeakRateWindow(props.group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset)) : '')
</script>

<template>
  <section class="min-w-0 border-t border-separator-border p-4 sm:p-5" :data-group-id="group.id">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h3 class="break-words text-body-medium text-text-primary">{{ group.name }}</h3>
        <Badge v-if="group.user_rate_multiplier != null" tone="accent">{{ t('modelPlaza.compare.personal') }}</Badge>
        <Badge v-if="subscription">{{ t('modelPlaza.badges.subscription') }}</Badge>
        <Badge v-if="group.is_exclusive">{{ t('modelPlaza.badges.exclusive') }}</Badge>
      </div>
      <span class="text-caption-1-regular text-text-tertiary">{{ t(mediaMode === 'token' ? 'modelPlaza.compare.perMillion' : `modelPlaza.compare.units.${mediaMode}`) }}<span v-if="firstTier"> · {{ firstTier }}</span></span>
    </div>
    <p v-if="subscription" class="mb-3 text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.subscriptionNote') }}</p>
    <BoardTable class="plaza-offer-table table-fixed">
      <thead><tr><th scope="col">{{ t('modelPlaza.compare.item') }}</th><th scope="col">{{ t(subscription ? 'modelPlaza.compare.quota' : 'modelPlaza.compare.ourPrice') }}</th><th scope="col">{{ t('modelPlaza.compare.official') }}</th><th scope="col">{{ t('modelPlaza.compare.advantage') }}</th></tr></thead>
      <tbody>
        <PlazaPriceRow v-for="row in (primary.length ? primary : rows)" :key="row.key" :row="row" :rate="rate" :currency="currency" :cny-per-credit="cnyPerCredit" :quote="quote" :subscription="subscription" />
      </tbody>
    </BoardTable>
    <details class="mt-4" data-testid="offer-details">
      <summary class="cursor-pointer rounded-md text-body-2-medium text-text-accent outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring">{{ t('modelPlaza.compare.details') }}</summary>
      <div class="mt-4 flex flex-col gap-4">
        <p v-if="group.description" class="text-body-2-regular text-text-secondary">{{ group.description }}</p>
        <div v-if="details.length && primary.length">
          <BoardTable class="plaza-offer-table table-fixed"><tbody><PlazaPriceRow v-for="row in details" :key="row.key" :row="row" :rate="rate" :currency="currency" :cny-per-credit="cnyPerCredit" :quote="quote" :subscription="subscription" /></tbody></BoardTable>
        </div>
        <p v-if="rows.some(row => row.tier)" class="text-caption-1-regular text-text-secondary">{{ t(model.long_context_basis === 'marginal' ? 'modelPlaza.table.tierHintMarginal' : 'modelPlaza.table.tierHint') }}</p>
        <p v-if="group.long_context_pricing_enabled === false && (model.official_pricing?.intervals?.length ?? 0) > 1" class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.detail.longContextDisabledNote') }}</p>
        <div v-for="period in periods" :key="period.start_time" class="flex flex-col gap-2">
          <p class="text-body-2-medium text-text-primary">{{ period.start_time }}–{{ period.end_time }} · {{ model.time_pricing?.timezone }}<span v-if="model.time_pricing?.weekdays_only"> · {{ t('modelPlaza.table.timePricingWeekdays') }}</span> · ×{{ period.multiplier }}</p>
          <BoardTable class="plaza-offer-table table-fixed"><tbody><PlazaPriceRow v-for="row in rows" :key="row.key" :row="row" :rate="rate * period.multiplier" :currency="currency" :cny-per-credit="cnyPerCredit" :quote="quote" :subscription="subscription" /></tbody></BoardTable>
        </div>
        <p v-if="peakWindow" class="text-caption-1-regular text-text-warning">{{ t('modelPlaza.detail.peakNote', { window: peakWindow, multiplier: group.peak_rate_multiplier }) }} · {{ t('modelPlaza.compare.peakExcluded') }}</p>
        <p class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.groupRate', { value: rate }) }}</p>
        <p class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.creditExplanation') }}</p>
        <p v-for="(multiplier, effort) in model.pricing?.reasoning_effort_multipliers ?? {}" :key="effort" class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.table.reasoningMultiplierHint', { effort, multiplier }) }}</p>
      </div>
    </details>
  </section>
</template>

<style>
.plaza-offer-table > thead > tr > th { @apply whitespace-normal px-1.5 text-caption-1-medium sm:px-3; }
.plaza-offer-table > tbody > tr > td { @apply px-1.5 sm:px-3; overflow-wrap: anywhere; }
</style>
