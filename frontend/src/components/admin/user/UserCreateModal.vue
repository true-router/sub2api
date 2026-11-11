<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.createUser')"
    width="normal"
    @close="$emit('close')"
  >
    <form id="create-user-form" @submit.prevent="submit" class="space-y-5 lg:space-y-4">
      <BoardInput v-model="form.email" type="email" required autofocus :label="t('admin.users.email')" :placeholder="t('admin.users.enterEmail')" autocomplete="off" />
      <div class="flex items-end gap-2">
        <BoardInput v-model="form.password" type="password" required revealable :label="t('admin.users.password')" :placeholder="t('admin.users.enterPassword')" autocomplete="new-password" />
        <BoardButton type="button" @click="generateRandomPassword" class="px-3" variant="secondary" size="medium" :aria-label="t('admin.users.password')"><Icon name="refresh" size="md" /></BoardButton>
      </div>
      <BoardInput v-model="form.username" :label="t('admin.users.username')" :placeholder="t('admin.users.enterUsername')" />
      <div>
        <label class="input-label">{{ t('admin.users.form.roleLabel') }}</label>
        <Select :model-value="form.role" :aria-label="t('admin.users.form.roleLabel')" :options="[{ value: 'user', label: t('admin.users.roles.user') }, { value: 'admin', label: t('admin.users.roles.admin') }]" @update:model-value="form.role = $event as 'user' | 'admin'" />
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <BoardNumberField :model-value="form.balance" @update:model-value="form.balance = String($event)" :label="t('admin.users.columns.balance')" step="any" />
        <BoardNumberField :model-value="form.concurrency" @update:model-value="form.concurrency = Number($event)" :label="t('admin.users.columns.concurrency')" :min="1" :step="1" />
      </div>
      <BoardNumberField :model-value="form.rpm_limit" @update:model-value="form.rpm_limit = Number($event)" :label="t('admin.users.form.rpmLimit')" :min="0" :step="1" :hint="t('admin.users.form.rpmLimitHint')" />
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <BoardButton @click="$emit('close')" type="button" variant="secondary" size="medium">{{ t('common.cancel') }}</BoardButton>
        <BoardButton type="submit" form="create-user-form" :disabled="loading" variant="primary" size="medium">
          {{ loading ? t('admin.users.creating') : t('common.create') }}
        </BoardButton>
      </div>
    </template>
  </BaseDialog>

  <!-- 创建管理员账号时后端要求 step-up 2FA，弹出 TOTP 验证后自动重试 -->
  <TotpStepUpDialog :controller="stepUp" />
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import BoardInput from '@/components/boardui/Input.vue'
import Select from '@/components/common/Select.vue'
import BoardNumberField from '@/components/boardui/NumberField.vue'
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'; import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useStepUp, isStepUpBlocked, isStepUpCancelled, stepUpBlockReason } from '@/composables/useStepUp'
import TotpStepUpDialog from '@/components/auth/TotpStepUpDialog.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'success']); const { t } = useI18n()
const appStore = useAppStore()

const form = reactive({ email: '', password: '', username: '', notes: '', role: 'user' as 'user' | 'admin', balance: '', concurrency: 1, rpm_limit: 0 })

const stepUp = useStepUp()
const loading = ref(false)

const submit = async () => {
  if (loading.value) return
  loading.value = true
  try {
    const { balance: rawBalance, ...rest } = { ...form }
    const balance = String(rawBalance).trim()
    const payload: typeof rest & { balance?: number } = { ...rest }
    if (balance !== '') {
      payload.balance = Number(balance)
    }
    // 创建管理员属敏感操作：后端返回 STEP_UP_REQUIRED 时弹 TOTP 验证并重试
    await stepUp.run(() => adminAPI.users.create(payload))
    appStore.showSuccess(t('admin.users.userCreated'))
    emit('success'); emit('close')
  } catch (e: any) {
    if (isStepUpCancelled(e)) {
      // 用户主动取消二次验证：静默返回，表单保持打开。
    } else if (isStepUpBlocked(e)) {
      appStore.showError(
        stepUpBlockReason(e) === 'STEP_UP_ADMIN_API_KEY_FORBIDDEN'
          ? t('stepUp.adminApiKeyForbidden')
          : t('stepUp.notEnabled')
      )
    } else {
      appStore.showError(e?.message || t('admin.users.failedToCreate'))
    }
  } finally { loading.value = false }
}

watch(() => props.show, (v) => { if(v) Object.assign(form, { email: '', password: '', username: '', notes: '', role: 'user', balance: '', concurrency: 1, rpm_limit: 0 }) })

const generateRandomPassword = () => {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let p = ''; for (let i = 0; i < 16; i++) p += chars.charAt(Math.floor(Math.random() * chars.length))
  form.password = p
}
</script>
