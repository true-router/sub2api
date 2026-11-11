<template>
  <BaseDialog :show="show" :title="isConversion ? t('wallet.convertToBonus') : operation === 'add' ? t('admin.users.deposit') : t('admin.users.withdraw')" width="narrow" @close="$emit('close')">
    <form v-if="user" id="balance-form" @submit.prevent="handleBalanceSubmit" class="space-y-5">
      <div class="flex items-center gap-3 rounded-xl bg-background-secondary-default p-4 ">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-background-accent-subtle"><span class="text-title-3-medium text-text-accent">{{ user.email.charAt(0).toUpperCase() }}</span></div>
        <div class="flex-1"><p class="font-medium text-text-primary ">{{ user.email }}</p><p class="text-body-regular text-text-secondary ">{{ t('admin.users.currentBalance') }}: ${{ formatBalance(user.balance) }}</p></div>
      </div>
      <div v-if="isConversion" class="space-y-2">
        <BalanceBreakdown :user="user" />
        <p class="text-body-regular text-text-secondary">{{ t('wallet.conversionRule') }}</p>
      </div>
      <div>
        <label class="input-label">{{ isConversion ? t('wallet.conversionAmount') : operation === 'add' ? t('admin.users.depositAmount') : t('admin.users.withdrawAmount') }}</label>
        <div class="relative flex gap-2">
          <div class="relative flex-1"><div class="absolute left-3 top-1/2 -translate-y-1/2 font-medium text-text-secondary">$</div><input v-model.number="form.amount" type="number" step="any" min="0" :max="isConversion ? availablePaid : undefined" required class="input pl-8" /></div>
          <BoardButton v-if="operation === 'subtract'" type="button" @click="fillAllBalance" class="whitespace-nowrap" variant="secondary" size="medium">{{ t('admin.users.withdrawAll') }}</BoardButton>
        </div>
      </div>
      <div v-if="operation === 'add'" class="space-y-2">
        <label class="input-label" for="credit-source">{{ t('wallet.creditSource') }}</label>
        <Select id="credit-source" v-model="form.source" :options="creditSources" />
        <p class="text-caption-1-regular text-text-secondary">{{ form.source === 'paid' ? t('wallet.paidCreditEvidence') : t('wallet.adminCreditRule') }}</p>
      </div>
      <div><label class="input-label">{{ isConversion ? t('wallet.conversionReason') : t('admin.users.notes') }}</label><textarea v-model="form.notes" :required="isConversion" rows="3" class="input"></textarea></div>
      <div v-if="isConversion && form.amount > 0" class="space-y-1 text-body-regular text-text-secondary">
        <p>{{ t('wallet.paidAfterConversion') }}: ${{ formatBalance(availablePaid - form.amount) }}</p>
        <p>{{ t('wallet.bonusAfterConversion') }}: ${{ formatBalance((user.bonus_balance ?? 0) + form.amount) }}</p>
      </div>
      <div v-if="form.amount > 0" class="rounded-xl border border-text-info bg-background-info-subtle p-4 "><div class="flex items-center justify-between text-body-regular"><span class="text-text-primary ">{{ t('admin.users.newBalance') }}:</span><span class="font-bold text-text-primary ">${{ formatBalance(calculateNewBalance()) }}</span></div></div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <BoardButton type="button" @click="$emit('close')" variant="secondary" size="medium">{{ t('common.cancel') }}</BoardButton>
        <BoardButton type="submit" form="balance-form" :loading="submitting" :disabled="!form.amount || (isConversion && (!form.notes.trim() || form.amount > availablePaid))" :variant="isConversion ? 'primary' : operation === 'add' ? 'success' : 'danger'">
          {{ submitting ? t('common.saving') : t('common.confirm') }}
        </BoardButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { computed, reactive, ref, watch } from 'vue'
import Select from '@/components/common/Select.vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import BalanceBreakdown from '@/components/user/BalanceBreakdown.vue'

const props = defineProps<{ show: boolean, user: AdminUser | null, operation: 'add' | 'subtract' | 'convert_paid_to_bonus' }>()
const emit = defineEmits(['close', 'success']); const { t } = useI18n(); const appStore = useAppStore()

const isConversion = computed(() => props.operation === 'convert_paid_to_bonus')
const availablePaid = computed(() => Math.max(0, props.user?.paid_balance ?? 0))
const submitting = ref(false); const form = reactive({ amount: 0, notes: '', source: 'bonus' })
let conversionAttempt: { body: string; key: string } | null = null
watch(() => props.show, (v) => { if(v) { form.amount = 0; form.notes = ''; form.source = 'bonus'; conversionAttempt = null } })

const creditSources = computed(() => [{ value: 'bonus', label: t('wallet.bonusBalance') }, { value: 'paid', label: t('wallet.paidBalance') }])

// 格式化余额：显示完整精度，去除尾部多余的0
const formatBalance = (value: number) => {
  if (value === 0) return '0.00'
  // 最多保留8位小数，去除尾部的0
  const formatted = value.toFixed(8).replace(/\.?0+$/, '')
  // 确保至少有2位小数
  const parts = formatted.split('.')
  if (parts.length === 1) return formatted + '.00'
  if (parts[1].length === 1) return formatted + '0'
  return formatted
}

// 填入全部余额
const fillAllBalance = () => {
  if (props.user) {
    form.amount = props.user.balance
  }
}

const calculateNewBalance = () => {
  if (!props.user) return 0
  if (isConversion.value) return props.user.balance
  const result = props.operation === 'add' ? props.user.balance + form.amount : props.user.balance - form.amount
  // 避免浮点数精度问题导致的 -0.00 显示
  return Math.abs(result) < 1e-10 ? 0 : result
}
const handleBalanceSubmit = async () => {
  if (!props.user || submitting.value) return
  if (!form.amount || form.amount <= 0) {
    appStore.showError(t('admin.users.amountRequired'))
    return
  }
  // 退款时验证金额不超过实际余额
  if (props.operation === 'subtract' && form.amount > props.user.balance) {
    appStore.showError(t('admin.users.insufficientBalance'))
    return
  }
  if (props.operation === 'add' && form.source === 'paid' && !form.notes.trim()) {
    appStore.showError(t('wallet.paidCreditEvidence'))
    return
  }
  if (isConversion.value && (!form.notes.trim() || form.amount > availablePaid.value)) {
    appStore.showError(!form.notes.trim() ? t('wallet.conversionReasonRequired') : t('wallet.conversionInsufficient'))
    return
  }
  submitting.value = true
  try {
    let idempotencyKey: string | undefined
    if (isConversion.value) {
      const body = JSON.stringify({ userId: props.user.id, amount: form.amount, reason: form.notes })
      if (conversionAttempt?.body !== body) {
        const requestId = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
        conversionAttempt = { body, key: `balance-conversion-${requestId}` }
      }
      idempotencyKey = conversionAttempt.key
    }
    await adminAPI.users.updateBalance(props.user.id, form.amount, props.operation === 'add' && form.source === 'paid' ? 'add_paid' : props.operation, form.notes, idempotencyKey)
    appStore.showSuccess(t('common.success')); emit('success'); emit('close')
  } catch (e: any) {
    console.error('Failed to update balance:', e)
    appStore.showError(extractApiErrorMessage(e, t('common.error')))
  } finally { submitting.value = false }
}
</script>
