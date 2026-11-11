<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.tempUnschedulable.statusTitle')"
    width="normal"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div v-if="loading" class="flex items-center justify-center py-8">
        <svg class="h-6 w-6 animate-spin text-text-tertiary" fill="none" viewBox="0 0 24 24">
          <circle
            class="opacity-25"
            cx="12"
            cy="12"
            r="10"
            stroke="currentColor"
            stroke-width="4"
          ></circle>
          <path
            class="opacity-75"
            fill="currentColor"
            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
          ></path>
        </svg>
      </div>

      <div v-else-if="!isActive" class="rounded-lg border border-border-button-default p-4 text-body-regular text-text-secondary ">
        {{ t('admin.accounts.tempUnschedulable.notActive') }}
      </div>

      <div v-else class="space-y-4">
        <div class="rounded-lg border border-text-success bg-background-success-subtle p-3 text-body-regular text-text-success ">
          {{ t('admin.accounts.recoverStateHint') }}
        </div>

        <div class="rounded-lg border border-border-button-default p-4 ">
          <p class="text-caption-1-regular text-text-secondary ">
            {{ t('admin.accounts.tempUnschedulable.accountName') }}
          </p>
          <p class="mt-1 text-body-medium text-text-primary ">
            {{ account?.name || '-' }}
          </p>
        </div>

        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="rounded-lg border border-border-button-default p-3 ">
            <p class="text-caption-1-regular text-text-secondary ">
              {{ t('admin.accounts.tempUnschedulable.triggeredAt') }}
            </p>
            <p class="mt-1 text-body-medium text-text-primary ">
              {{ triggeredAtText }}
            </p>
          </div>
          <div class="rounded-lg border border-border-button-default p-3 ">
            <p class="text-caption-1-regular text-text-secondary ">
              {{ t('admin.accounts.tempUnschedulable.until') }}
            </p>
            <p class="mt-1 text-body-medium text-text-primary ">
              {{ untilText }}
            </p>
          </div>
          <div class="rounded-lg border border-border-button-default p-3 ">
            <p class="text-caption-1-regular text-text-secondary ">
              {{ t('admin.accounts.tempUnschedulable.remaining') }}
            </p>
            <p class="mt-1 text-body-medium text-text-primary ">
              {{ remainingText }}
            </p>
          </div>
          <div class="rounded-lg border border-border-button-default p-3 ">
            <p class="text-caption-1-regular text-text-secondary ">
              {{ t('admin.accounts.tempUnschedulable.errorCode') }}
            </p>
            <p class="mt-1 text-body-medium text-text-primary ">
              {{ state?.status_code || '-' }}
            </p>
          </div>
          <div class="rounded-lg border border-border-button-default p-3 ">
            <p class="text-caption-1-regular text-text-secondary ">
              {{ t('admin.accounts.tempUnschedulable.matchedKeyword') }}
            </p>
            <p class="mt-1 text-body-medium text-text-primary ">
              {{ state?.matched_keyword || '-' }}
            </p>
          </div>
          <div class="rounded-lg border border-border-button-default p-3 ">
            <p class="text-caption-1-regular text-text-secondary ">
              {{ t('admin.accounts.tempUnschedulable.ruleOrder') }}
            </p>
            <p class="mt-1 text-body-medium text-text-primary ">
              {{ ruleIndexDisplay }}
            </p>
          </div>
        </div>

        <div class="rounded-lg border border-border-button-default p-3 ">
          <p class="text-caption-1-regular text-text-secondary ">
            {{ t('admin.accounts.tempUnschedulable.errorMessage') }}
          </p>
          <div class="mt-2 rounded bg-background-secondary-default p-2 text-caption-1-regular text-text-primary ">
            {{ state?.error_message || '-' }}
          </div>
        </div>

        <div
          v-if="hasThresholdEvidence"
          class="rounded-lg border border-text-info bg-background-info-subtle p-3 text-body-regular text-text-info "
          data-testid="temp-unsched-trigger-evidence"
        >
          {{ triggerEvidenceText }}
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <BoardButton type="button" variant="secondary" size="medium" @click="handleClose">
          {{ t('common.close') }}
        </BoardButton>
        <BoardButton
          type="button"
          variant="primary" size="medium"
          :disabled="!isActive || resetting"
          @click="handleReset"
        >
          <svg
            v-if="resetting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          {{ t('admin.accounts.recoverState') }}
        </BoardButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Account, TempUnschedulableStatus } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{
  show: boolean
  account: Account | null
}>()

const emit = defineEmits<{
  close: []
  reset: [account: Account]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const resetting = ref(false)
const status = ref<TempUnschedulableStatus | null>(null)
let requestVersion = 0

const state = computed(() => status.value?.state || null)

const isActive = computed(() => {
  if (!status.value?.active || !state.value) return false
  return state.value.until_unix * 1000 > Date.now()
})

const ruleIndexDisplay = computed(() => {
  if (!state.value || !state.value.matched_keyword || state.value.rule_index < 0) return '-'
  return state.value.rule_index + 1
})

const hasThresholdEvidence = computed(() => (state.value?.trigger_count || 0) > 1)

const triggerEvidenceText = computed(() => {
  const count = state.value?.trigger_count || 0
  const threshold = state.value?.trigger_threshold || 0
  const minutes = state.value?.trigger_window_minutes || 0
  if (threshold > 0 && minutes > 0) {
    return t('admin.accounts.tempUnschedulable.multipleErrorTrigger', { count, threshold, minutes })
  }
  if (threshold > 0) {
    return t('admin.accounts.tempUnschedulable.multipleErrorTriggerNoWindow', { count, threshold })
  }
  if (minutes > 0) {
    return t('admin.accounts.tempUnschedulable.multipleErrorCountInWindow', { count, minutes })
  }
  return t('admin.accounts.tempUnschedulable.multipleErrorCount', { count })
})

const triggeredAtText = computed(() => {
  if (!state.value?.triggered_at_unix) return '-'
  return formatDateTime(new Date(state.value.triggered_at_unix * 1000))
})

const untilText = computed(() => {
  if (!state.value?.until_unix) return '-'
  return formatDateTime(new Date(state.value.until_unix * 1000))
})

const remainingText = computed(() => {
  if (!state.value) return '-'
  const remainingMs = state.value.until_unix * 1000 - Date.now()
  if (remainingMs <= 0) {
    return t('admin.accounts.tempUnschedulable.expired')
  }
  const minutes = Math.ceil(remainingMs / 60000)
  if (minutes < 60) {
    return t('admin.accounts.tempUnschedulable.remainingMinutes', { minutes })
  }
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  if (rest === 0) {
    return t('admin.accounts.tempUnschedulable.remainingHours', { hours })
  }
  return t('admin.accounts.tempUnschedulable.remainingHoursMinutes', { hours, minutes: rest })
})

const loadStatus = async () => {
  if (!props.account) return
  const version = ++requestVersion
  status.value = null
  loading.value = true
  try {
    const result = await adminAPI.accounts.getTempUnschedulableStatus(props.account.id)
    if (version === requestVersion) status.value = result
  } catch (error: any) {
    if (version !== requestVersion) return
    appStore.showError(error?.message || t('admin.accounts.tempUnschedulable.failedToLoad'))
    status.value = null
  } finally {
    if (version === requestVersion) loading.value = false
  }
}

const handleClose = () => {
  emit('close')
}

const handleReset = async () => {
  if (!props.account) return
  resetting.value = true
  try {
    const updated = await adminAPI.accounts.recoverState(props.account.id)
    appStore.showSuccess(t('admin.accounts.recoverStateSuccess'))
    emit('reset', updated)
    handleClose()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.recoverStateFailed'))
  } finally {
    resetting.value = false
  }
}

watch(
  () => [props.show, props.account?.id],
  ([visible], _, onCleanup) => {
    onCleanup(() => { requestVersion++ })
    if (visible && props.account) {
      loadStatus()
      return
    }
    status.value = null
  }
)
</script>
