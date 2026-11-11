<template>
  <div class="card p-4">
    <h3 class="mb-4 text-body-semibold text-text-primary ">
      {{ t('payment.admin.paymentDistribution') }}
    </h3>
    <div
      v-if="!methods?.length"
      class="flex h-32 items-center justify-center text-body-regular text-text-secondary "
    >
      {{ t('payment.admin.noData') }}
    </div>
    <div v-else class="space-y-3">
      <div v-for="method in methods" :key="method.type" class="space-y-1">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <span :class="['inline-block h-3 w-3 rounded-full', colorMap[method.type] || 'bg-background-quaternary-default']"></span>
            <span class="text-body-regular text-text-primary ">
              {{ t('payment.methods.' + method.type, method.type) }}
            </span>
          </div>
          <div class="space-y-1 text-right">
            <span v-for="[currency, amount] in sortedAmounts(method.amount)" :key="currency" class="block text-body-medium text-text-primary ">
              {{ formatMoney(currency, amount) }}
            </span>
            <span class="ml-2 text-caption-1-regular text-text-secondary ">
              ({{ method.count }})
            </span>
          </div>
        </div>
        <div v-for="[currency, amount] in sortedAmounts(method.amount)" :key="currency" class="flex items-center gap-2">
          <span class="w-10 text-caption-1-regular text-text-secondary ">{{ currency }}</span>
          <div class="h-2 flex-1 overflow-hidden rounded-full bg-background-secondary-default ">
            <div
              :class="['h-full rounded-full transition-all', barColorMap[method.type] || 'bg-background-quaternary-default']"
              :style="{ width: barWidth(currency, amount) + '%' }"
            ></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { CurrencyAmounts, PaymentMethodStats } from '@/types/payment'

const { t } = useI18n()

const props = defineProps<{
  methods: PaymentMethodStats[]
}>()

const colorMap: Record<string, string> = {
  alipay: 'bg-background-info-default',
  wxpay: 'bg-background-success-default',
  alipay_direct: 'bg-background-info-default',
  wxpay_direct: 'bg-background-success-default',
  stripe: 'bg-background-feature-default',
}

const barColorMap: Record<string, string> = {
  alipay: 'bg-background-info-default',
  wxpay: 'bg-background-success-default',
  alipay_direct: 'bg-background-info-default',
  wxpay_direct: 'bg-background-success-default',
  stripe: 'bg-background-feature-default',
}

const maxAmounts = computed<CurrencyAmounts>(() => {
  return props.methods.reduce<CurrencyAmounts>((maximums, method) => {
    for (const [currency, amount] of Object.entries(method.amount)) {
      maximums[currency] = Math.max(maximums[currency] || 0, amount)
    }
    return maximums
  }, {})
})

function sortedAmounts(amounts: CurrencyAmounts): [string, number][] {
  return Object.entries(amounts).sort(([left], [right]) => left.localeCompare(right))
}

function barWidth(currency: string, amount: number): number {
  return Math.min((amount / (maxAmounts.value[currency] || 1)) * 100, 100)
}

function formatMoney(currency: string, amount: number): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(amount)
}
</script>
