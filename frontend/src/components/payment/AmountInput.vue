<template>
  <div class="space-y-5">
    <div class="space-y-3"><h3 class="text-body-medium text-text-primary">{{ t('payment.quickAmounts') }}</h3>
      <div class="grid gap-3 sm:grid-cols-3" role="group" :aria-label="t('payment.quickAmounts')">
        <BoardButton v-for="amt in filteredAmounts" :key="amt" variant="secondary" :aria-pressed="modelValue === amt" :data-testid="`quick-amount-${amt}`" :class="cx('h-auto justify-start whitespace-normal p-4 text-start', modelValue === amt && 'border-border-focus-ring bg-background-accent-subtle')" @click="selectAmount(amt)">
          <span class="space-y-1"><span class="block text-body-semibold">{{ t('console.amountPay', {amount: formatPaymentAmount(quoteFor(amt).payBase, currency)}) }}</span><span class="block text-caption-1-regular text-text-secondary" data-testid="quick-amount-credited">{{ t('console.amountCredit', {amount: '$' + quoteFor(amt).credited.toFixed(2)}) }}</span></span>
        </BoardButton>
      </div>
    </div>
    <BoardInput :model-value="customText" :label="t('payment.customAmount')" :placeholder="placeholderText" inputmode="decimal" @input="handleInput"><template #prefix><span>{{ currency || 'USD' }}</span></template></BoardInput>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardInput from '@/components/boardui/Input.vue'
import { cx } from '@/utils/cx'
import { useI18n } from 'vue-i18n'
import type { RechargeBonusTier } from '@/types/payment'
import { quoteRechargeBonus, type RechargeBonusMode } from '@/utils/rechargeBonus'
import { formatPaymentAmount } from './currency'

const props = withDefaults(defineProps<{
  amounts?: number[]
  modelValue: number | null
  min?: number
  max?: number
  /** 充值优惠阶梯（按 min_amount 升序）；为空时不显示价签与第二行 */
  bonusTiers?: RechargeBonusTier[]
  /** 阶梯模式：bonus 赠金 / discount 折扣 */
  bonusMode?: RechargeBonusMode
  /** 充值倍率（1 支付币种 = multiplier USD），用于计算到账金额 */
  multiplier?: number
  /** 支付币种（折扣模式第二行实付金额的币种与精度） */
  currency?: string
}>(), {
  amounts: () => [10, 20, 50, 100, 200, 500, 1000, 2000, 5000],
  min: 0,
  max: 0,
  bonusTiers: () => [],
  bonusMode: 'bonus',
  multiplier: 1,
  currency: undefined,
})

const emit = defineEmits<{
  'update:modelValue': [value: number | null]
}>()

const { t } = useI18n()

const customText = ref('')

// 0 = no limit
const filteredAmounts = computed(() =>
  props.amounts.filter((a) => (props.min <= 0 || a >= props.min) && (props.max <= 0 || a <= props.max))
)


function currencyDigits(): number {
  if (!props.currency) return 2
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: props.currency }).resolvedOptions().maximumFractionDigits ?? 2
  } catch {
    return 2
  }
}

function quoteFor(amt: number) {
  return quoteRechargeBonus(props.bonusTiers, amt, {
    multiplier: props.multiplier,
    mode: props.bonusMode,
    currencyDigits: currencyDigits(),
  })
}

const placeholderText = computed(() => {
  if (props.min > 0 && props.max > 0) return `${props.min} - ${props.max}`
  if (props.min > 0) return `≥ ${props.min}`
  if (props.max > 0) return `≤ ${props.max}`
  return t('payment.enterAmount')
})

const AMOUNT_PATTERN = /^\d*(\.\d{0,2})?$/

function selectAmount(amt: number) {
  customText.value = String(amt)
  emit('update:modelValue', amt)
}

function handleInput(e: Event) {
  const input = e.target as HTMLInputElement
  const val = input.value
  if (!AMOUNT_PATTERN.test(val)) {
    input.value = customText.value
    return
  }
  customText.value = val
  if (val === '') {
    emit('update:modelValue', null)
    return
  }
  const num = parseFloat(val)
  if (!isNaN(num) && num > 0) {
    emit('update:modelValue', num)
  } else {
    emit('update:modelValue', null)
  }
}

watch(() => props.modelValue, (v) => {
  if (v !== null && String(v) !== customText.value) {
    customText.value = String(v)
  }
}, { immediate: true })
</script>
