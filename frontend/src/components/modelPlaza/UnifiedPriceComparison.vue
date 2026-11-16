<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardTable from '@/components/boardui/Table.vue'
import type { UnifiedPricingContext, UnifiedWalletOffer } from '@/api/apiAccess'
import { formatPoints } from '@/utils/points'
import { money, priceRows, savings, savingsPercent, validQuote } from './plazaPricing'
const props = defineProps<{ offer: UnifiedWalletOffer; context: UnifiedPricingContext | null }>()
const { t } = useI18n()
const rows = computed(() => priceRows({ name: '', platform: '', pricing: props.offer.pricing, official_pricing: props.offer.official_pricing ?? null, long_context_basis: props.offer.long_context_basis }))
const quote = computed(() => validQuote(props.context?.exchange_rate) ? props.context!.exchange_rate : null)
const cnyPerPoint = computed(() => {
  const value = props.context?.cny_per_credit
  return value != null && Number.isFinite(value) && value > 0 ? value : null
})
const values = computed(() => rows.value.map(row => {
  const points = row.credits == null ? null : row.credits * props.offer.rate_multiplier * row.scale
  const official = row.official == null ? null : row.official * row.scale
  const cny = points == null || cnyPerPoint.value == null ? null : points * cnyPerPoint.value
  const equivalent = cny == null || !quote.value ? null : cny / quote.value.usd_to_cny
  const saving = cnyPerPoint.value == null ? null : savings(points, official, cnyPerPoint.value, quote.value)
  const pointsLabel = points == null ? '—' : points > 0 && points < 0.000001
    ? `${points.toExponential(2)} ${t('common.pointsUnit')}`
    : formatPoints(points, { minimumFractionDigits: 6, maximumFractionDigits: 6 })
  return { ...row, points, pointsLabel, official, cny, equivalent, saving }
}))
</script>
<template>
  <div class="min-w-0 overflow-x-auto" tabindex="0" :aria-label="t('modelPlaza.accountComparison.table')">
    <BoardTable size="sm">
      <thead><tr>
        <th scope="col">{{ offer.pricing?.billing_mode === 'token' ? t('apiAccess.perMillion') : t('apiAccess.perRequest') }}</th>
        <th scope="col">{{ t('modelPlaza.accountComparison.officialUSD') }}</th>
        <th scope="col">{{ t('modelPlaza.compare.ourPrice') }}</th>
        <th scope="col">{{ t('modelPlaza.compare.advantage') }}</th>
      </tr></thead>
      <tbody>
        <tr v-for="row in values" :key="row.key" :data-price-row="row.key">
          <th scope="row" class="px-3 py-2.5 text-body-2-regular font-normal text-text-secondary">{{ t(`modelPlaza.compare.labels.${row.label}`) }}<span v-if="row.tier" class="mt-1 block text-caption-1-regular text-text-tertiary">{{ row.tier }}</span></th>
          <td><span class="tabular-nums text-text-secondary" data-testid="official-usd">{{ money(row.official, 'USD') }}</span></td>
          <td>
            <span v-if="row.cny != null" class="block tabular-nums text-text-primary" data-testid="site-cny">{{ money(row.cny, 'CNY') }}</span>
            <span v-if="row.equivalent != null" class="mt-1 block text-caption-1-regular tabular-nums text-text-secondary" data-testid="site-usd">{{ money(row.equivalent, 'USD') }}</span>
            <span class="mt-1 block text-caption-1-regular tabular-nums text-text-tertiary" data-testid="site-points">{{ row.pointsLabel }}</span>
          </td>
          <td><span v-if="row.saving != null && row.saving > 0" class="text-body-2-medium tabular-nums text-text-success" data-testid="price-saving">{{ t('modelPlaza.compare.save', { value: savingsPercent(row.saving) }) }}</span><span v-else class="text-caption-1-regular text-text-tertiary">{{ row.saving != null ? t('modelPlaza.compare.noSaving') : '—' }}</span></td>
        </tr>
      </tbody>
    </BoardTable>
    <p v-if="!offer.official_pricing" class="px-3 py-2 text-caption-1-regular text-text-tertiary">{{ t('modelPlaza.accountComparison.noOfficial') }}</p>
  </div>
</template>
