<template>
  <BaseDialog
    :show="show"
    :title="t('payment.admin.orderDetail')"
    width="wide"
    @close="emit('close')"
  >
    <div v-if="order" class="space-y-4">
      <div class="grid grid-cols-2 gap-4">
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.orderId') }}</p>
          <p class="font-mono text-body-medium text-text-primary ">#{{ order.id }}</p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.status') }}</p>
          <span :class="['badge', statusBadgeClass(order.status)]">
            {{ t('payment.status.' + order.status.toLowerCase(), order.status) }}
          </span>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.baseAmount') }}</p>
          <p class="text-body-medium text-text-primary ">{{ order.pay_amount_known === false ? t('common.unknown') : paymentAmountSymbol + baseAmount.toFixed(2) }}</p>
        </div>
        <div v-if="order.fee_rate > 0">
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.fee') }} ({{ order.fee_rate }}%)</p>
          <p class="text-body-medium text-text-primary ">{{ paymentAmountSymbol }}{{ feeAmount.toFixed(2) }}</p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t(order.payment_type === 'opcpay' && order.pay_amount_known !== false ? 'payment.orders.catalogAmount' : 'payment.orders.payAmount') }}</p>
          <p class="text-body-medium text-text-primary ">{{ order.pay_amount_known === false ? t('common.unknown') : paymentAmountSymbol + order.pay_amount.toFixed(2) }}</p>
        </div>
        <div v-if="(order.bonus_amount ?? 0) > 0">
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.bonusAmount') }}</p>
          <p class="text-body-medium text-text-warning ">+{{ creditedAmountSymbol }}{{ (order.bonus_amount ?? 0).toFixed(2) }}</p>
        </div>
        <div v-if="order.amount !== order.pay_amount || (order.bonus_amount ?? 0) > 0">
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.creditedAmount') }}</p>
          <p class="text-body-medium text-text-primary ">{{ creditedAmountSymbol }}{{ order.amount.toFixed(2) }}</p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.paymentMethod') }}</p>
          <p class="text-body-regular text-text-primary ">
            {{ t('payment.methods.' + order.payment_type, order.payment_type) }}
          </p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.admin.orderType') }}</p>
          <p class="text-body-regular text-text-primary ">
            {{ t('payment.admin.' + order.order_type + 'Order', order.order_type) }}
          </p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.userId') }}</p>
          <p class="text-body-regular text-text-primary ">#{{ order.user_id }}</p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.orders.createdAt') }}</p>
          <p class="text-body-regular text-text-primary ">{{ formatDateTime(order.created_at) }}</p>
        </div>
        <div>
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.admin.expiresAt') }}</p>
          <p class="text-body-regular text-text-primary ">{{ formatDateTime(order.expires_at) }}</p>
        </div>
        <div v-if="order.paid_at">
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.admin.paidAt') }}</p>
          <p class="text-body-regular text-text-primary ">{{ formatDateTime(order.paid_at) }}</p>
        </div>
        <div v-if="order.completed_at">
          <p class="text-caption-1-regular text-text-secondary ">{{ t('payment.admin.completedAt') }}</p>
          <p class="text-body-regular text-text-primary ">{{ formatDateTime(order.completed_at) }}</p>
        </div>
      </div>

      <div
        v-if="order.refund_amount"
        class="rounded-lg border border-border-error-default bg-background-error-subtle p-3 "
      >
        <h4 class="mb-2 text-body-semibold text-text-error-primary ">
          {{ t('payment.admin.refundInfo') }}
        </h4>
        <div class="grid grid-cols-2 gap-2 text-body-regular">
          <div>
            <span class="text-text-error-primary ">{{ t('payment.admin.refundAmount') }}:</span>
            <span class="ml-1 font-medium text-text-error-primary ">{{ creditedAmountSymbol }}{{ order.refund_amount.toFixed(2) }}</span>
          </div>
          <div v-if="order.refund_reason" class="col-span-2">
            <span class="text-text-error-primary ">{{ t('payment.admin.refundReason') }}:</span>
            <span class="ml-1 text-text-error-primary ">{{ order.refund_reason }}</span>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-end gap-2 border-t border-border-button-default pt-4 ">
        <button
          v-if="order.status === 'PENDING'"
          @click="emit('cancel', order)"
          class="btn btn-sm rounded-md bg-background-warning-subtle px-3 py-1.5 text-body-regular text-text-warning hover:bg-background-warning-subtle "
        >
          {{ t('payment.orders.cancel') }}
        </button>
        <BoardButton type="submit"
          v-if="order.status === 'FAILED'"
          @click="emit('retry', order)"
          variant="secondary" size="small"
        >
          {{ t('payment.admin.retry') }}
        </BoardButton>
        <button
          v-if="canRefund(order)"
          @click="emit('refund', order)"
          class="btn btn-sm rounded-md bg-background-error-subtle px-3 py-1.5 text-body-regular text-text-error-primary hover:bg-background-error-subtle "
        >
          {{ t('payment.admin.refund') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { PaymentOrder } from '@/types/payment'
import { statusBadgeClass, canRefund as canRefundStatus, formatOrderDateTime } from '@/components/payment/orderUtils'
import { currencySymbol } from '@/components/payment/currency'

const { t } = useI18n()

const props = defineProps<{
  show: boolean
  order: PaymentOrder | null
}>()

const creditedAmountSymbol = currencySymbol('USD')

const paymentAmountSymbol = computed(() => currencySymbol(props.order?.currency))

/** 充值金额 (base amount before fee) = pay_amount - fee = pay_amount / (1 + fee_rate/100) */
const baseAmount = computed(() => {
  if (!props.order) return 0
  const feeRate = Number(props.order.fee_rate) || 0
  if (feeRate <= 0) return props.order.pay_amount
  return props.order.pay_amount / (1 + feeRate / 100)
})

/** 手续费 = pay_amount - baseAmount */
const feeAmount = computed(() => {
  if (!props.order) return 0
  const feeRate = Number(props.order.fee_rate) || 0
  if (feeRate <= 0) return 0
  return props.order.pay_amount - baseAmount.value
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'cancel', order: PaymentOrder): void
  (e: 'retry', order: PaymentOrder): void
  (e: 'refund', order: PaymentOrder): void
}>()

function canRefund(order: PaymentOrder): boolean {
  return order.payment_type !== 'opcpay' && canRefundStatus(order.status)
}

function formatDateTime(dateStr: string): string {
  return formatOrderDateTime(dateStr)
}
</script>
