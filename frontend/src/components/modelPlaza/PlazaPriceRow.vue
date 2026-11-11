<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { PlazaExchangeRate } from '@/api/modelPlaza'
import { money, savings, savingsPercent, validQuote, type DisplayCurrency, type PriceRow } from './plazaPricing'
const props = defineProps<{ row: PriceRow; rate: number; currency: DisplayCurrency; cnyPerCredit: number; quote?: PlazaExchangeRate | null; subscription: boolean }>()
const { t } = useI18n()
const credits = computed(() => props.row.credits == null ? null : props.row.credits * props.rate * props.row.scale)
const officialUSD = computed(() => props.row.official == null ? null : props.row.official * props.row.scale)
const own = computed(() => {
  if (props.subscription) return credits.value == null ? '—' : t('modelPlaza.compare.credits', { value: Number(credits.value.toPrecision(10)).toLocaleString('en-US', { maximumFractionDigits: 12 }) })
  if (credits.value == null) return '—'
  const cny = credits.value * props.cnyPerCredit
  return money(props.currency === 'CNY' ? cny : validQuote(props.quote) ? cny / props.quote.usd_to_cny : null, props.currency)
})
const reference = computed(() => props.currency === 'CNY' && validQuote(props.quote) ? money(officialUSD.value == null ? null : officialUSD.value * props.quote.usd_to_cny, 'CNY') : money(officialUSD.value, 'USD'))
const saving = computed(() => props.subscription ? null : savings(credits.value, officialUSD.value, props.cnyPerCredit, props.quote))
const percent = computed(() => saving.value == null ? '' : savingsPercent(saving.value))
</script>

<template>
  <tr :data-price="row.key">
    <td><span class="text-body-2-medium">{{ t(`modelPlaza.compare.labels.${row.label}`) }}</span><span v-if="row.tier" class="mt-1 block text-caption-1-regular text-text-tertiary">{{ row.tier }}</span></td>
    <td><span class="break-words text-body-medium tabular-nums text-text-primary" data-testid="own-price">{{ own }}</span></td>
    <td><span class="break-words text-body-2-regular tabular-nums text-text-secondary">{{ reference }}</span><span v-if="currency === 'CNY' && validQuote(quote) && officialUSD != null" class="mt-1 block text-caption-1-regular tabular-nums text-text-tertiary">{{ money(officialUSD, 'USD') }}</span></td>
    <td><span v-if="saving != null && saving > 0" class="text-body-2-medium tabular-nums text-text-success" data-testid="savings">{{ t('modelPlaza.compare.save', { value: percent }) }}</span><span v-else class="text-caption-1-regular text-text-tertiary">{{ saving != null ? t('modelPlaza.compare.noSaving') : '—' }}</span></td>
  </tr>
</template>
