<template>
  <BaseDialog
    :show="show"
    :title="t('payment.admin.refundOrder')"
    width="normal"
    @close="emit('cancel')"
  >
    <form id="refund-form" @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Refund Request Info -->
      <div
        v-if="order?.refund_requested_at || order?.refund_request_reason"
        class="rounded-lg border border-text-feature bg-background-feature-subtle p-3 "
      >
        <div class="flex items-center gap-2 text-body-medium text-text-feature ">
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {{ t('payment.admin.refundRequestInfo') }}
        </div>
        <div v-if="order?.refund_requested_at" class="mt-2 flex justify-between text-body-regular">
          <span class="text-text-feature ">{{ t('payment.admin.refundRequestedAt') }}</span>
          <span class="text-text-feature ">{{ formatDateTime(order.refund_requested_at) }}</span>
        </div>
        <div v-if="order?.refund_request_reason" class="mt-1 text-body-regular">
          <span class="text-text-feature ">{{ t('payment.admin.refundRequestReason') }}:</span>
          <span class="ml-1 text-text-feature ">{{ order.refund_request_reason }}</span>
        </div>
      </div>

      <!-- Order Info -->
      <div class="rounded-lg bg-background-secondary-default p-3 ">
        <div class="flex justify-between text-body-regular">
          <span class="text-text-secondary ">{{ t('payment.orders.orderId') }}</span>
          <span class="font-mono text-text-primary ">#{{ order?.id }}</span>
        </div>
        <div class="mt-1 flex justify-between text-body-regular">
          <span class="text-text-secondary ">{{ t('payment.orders.creditedAmount') }}</span>
          <span class="font-medium text-text-primary ">{{ creditedAmountSymbol }}{{ order?.amount?.toFixed(2) }}</span>
        </div>
        <div class="mt-1 flex justify-between text-body-regular">
          <span class="text-text-secondary ">{{ t('payment.orders.payAmount') }}</span>
          <span class="font-medium text-text-primary ">{{ paymentAmountSymbol }}{{ order?.pay_amount?.toFixed(2) }}</span>
        </div>
        <div v-if="actuallyRefunded > 0" class="mt-1 flex justify-between text-body-regular">
          <span class="text-text-secondary ">{{ t('payment.admin.alreadyRefunded') }}</span>
          <span class="font-medium text-text-error-primary ">{{ creditedAmountSymbol }}{{ actuallyRefunded.toFixed(2) }}</span>
        </div>
      </div>

      <!-- Deduct Balance -->
      <div>
        <div class="flex items-center gap-2">
          <input
            id="deduct-balance"
            v-model="form.deduct_balance"
            type="checkbox"
            class="h-4 w-4 rounded border-border-button-hover text-text-accent focus:ring-border-focus-ring"
          />
          <label for="deduct-balance" class="text-body-regular text-text-primary ">
            {{ t('payment.admin.deductBalance') }}
          </label>
          <span class="text-caption-1-regular text-text-secondary ">{{ t('payment.admin.deductBalanceHint') }}</span>
        </div>

        <!-- User Balance Info (when deduct_balance is checked) -->
        <div v-if="form.deduct_balance && userBalance != null" class="mt-3 grid grid-cols-2 gap-3">
          <div class="rounded-lg bg-background-secondary-default p-3 text-body-regular ">
            <div class="text-text-secondary ">{{ t('payment.admin.userBalance') }}</div>
            <div class="mt-1 font-semibold text-text-primary ">{{ creditedAmountSymbol }}{{ userBalance.toFixed(2) }}</div>
          </div>
          <div class="rounded-lg bg-background-secondary-default p-3 text-body-regular ">
            <div class="text-text-secondary ">{{ t('payment.admin.orderAmount') }}</div>
            <div class="mt-1 font-semibold text-text-primary ">{{ creditedAmountSymbol }}{{ order?.amount?.toFixed(2) }}</div>
          </div>
        </div>

        <!-- Insufficient balance warning -->
        <div
          v-if="form.deduct_balance && balanceInsufficient"
          class="mt-2 rounded-lg bg-background-warning-subtle p-3 text-body-regular text-text-warning "
        >
          {{ t('payment.admin.insufficientBalance') }}
        </div>

        <!-- No deduction info -->
        <div
          v-if="!form.deduct_balance"
          class="mt-2 rounded-lg bg-background-info-subtle p-3 text-body-regular text-text-info "
        >
          {{ t('payment.admin.noDeduction') }}
        </div>
      </div>

      <!-- Refund Amount -->
      <div>
        <label class="input-label">{{ t('payment.admin.refundAmount') }}</label>
        <div class="relative">
          <span class="absolute left-3 top-1/2 -translate-y-1/2 text-text-secondary">{{ creditedAmountSymbol }}</span>
          <input
            v-model.number="form.amount"
            type="number"
            step="0.01"
            min="0.01"
            :max="maxRefundable"
            class="input pl-7"
            required
          />
        </div>
        <p class="mt-1 text-caption-1-regular text-text-secondary ">
          {{ t('payment.admin.maxRefundable') }}: {{ creditedAmountSymbol }}{{ maxRefundable.toFixed(2) }}
        </p>
      </div>

      <!-- Reason -->
      <div>
        <label class="input-label">{{ t('payment.admin.refundReason') }}</label>
        <textarea
          v-model="form.reason"
          rows="3"
          class="input"
          :placeholder="t('payment.admin.refundReasonPlaceholder')"
          required
        ></textarea>
      </div>

      <!-- Warning -->
      <div
        v-if="warning"
        class="rounded-lg bg-background-warning-subtle p-3 text-body-regular text-text-warning "
      >
        {{ warning }}
      </div>

      <!-- Force Refund -->
      <div v-if="requireForce" class="flex items-center gap-2">
        <input
          id="force-refund"
          v-model="form.force"
          type="checkbox"
          class="h-4 w-4 rounded border-border-button-hover text-text-error-primary focus:ring-border-error-default"
        />
        <label for="force-refund" class="text-body-medium text-text-error-primary ">
          {{ t('payment.admin.forceRefund') }}
        </label>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <BoardButton type="button" @click="emit('cancel')" variant="secondary" size="medium">
          {{ t('common.cancel') }}
        </BoardButton>
        <button
          type="submit"
          form="refund-form"
          :disabled="submitting || form.amount <= 0 || (requireForce && !form.force)"
          class="rounded-md bg-background-error-default px-4 py-2 text-body-medium text-text-white hover:bg-background-error-default focus:outline-none focus:ring-2 focus:ring-border-error-default focus:ring-offset-2 disabled:opacity-50 dark:focus:ring-offset-dark-800"
        >
          {{ submitting ? t('common.processing') : t('payment.admin.confirmRefund') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { PaymentOrder } from '@/types/payment'
import { formatOrderDateTime } from '@/components/payment/orderUtils'
import { currencySymbol } from '@/components/payment/currency'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  order: PaymentOrder | null
  submitting?: boolean
  userBalance?: number | null
  requireForce?: boolean
  warning?: string
}>()

const emit = defineEmits<{
  (e: 'confirm', data: { amount: number; reason: string; deduct_balance: boolean; force: boolean }): void
  (e: 'cancel'): void
}>()

const creditedAmountSymbol = currencySymbol('USD')

const paymentAmountSymbol = computed(() => currencySymbol(props.order?.currency))

const form = reactive({
  amount: 0,
  reason: '',
  deduct_balance: true,
  force: false,
})

// In REFUND_REQUESTED / REFUND_PENDING status, refund_amount is requested/pending, not actually refunded.
// Only PARTIALLY_REFUNDED / REFUNDED have real refund amounts.
const actuallyRefunded = computed(() => {
  if (!props.order) return 0
  const s = props.order.status
  if (s === 'PARTIALLY_REFUNDED' || s === 'REFUNDED') return props.order.refund_amount || 0
  return 0
})

const maxRefundable = computed(() => {
  if (!props.order) return 0
  return props.order.amount - actuallyRefunded.value
})

const balanceInsufficient = computed(() => {
  if (props.userBalance == null || !props.order) return false
  return props.userBalance < form.amount
})

watch(() => props.show, (val) => {
  if (val && props.order) {
    // For REFUND_REQUESTED, pre-fill with the requested amount
    if (props.order.status === 'REFUND_REQUESTED' && props.order.refund_amount) {
      form.amount = props.order.refund_amount
    } else {
      form.amount = maxRefundable.value
    }
    form.reason = props.order.refund_request_reason || ''
    form.deduct_balance = true
    form.force = false
  }
})

function formatDateTime(dateStr: string): string {
  return formatOrderDateTime(dateStr)
}

function handleSubmit() {
  if (form.amount <= 0 || form.amount > maxRefundable.value) return
  if (props.requireForce && !form.force) return
  emit('confirm', { ...form })
}
</script>
