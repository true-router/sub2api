<template>
  <div :class="props.embedded ? 'space-y-4' : 'card'">
    <div
      v-if="!props.embedded"
      class="border-b border-border-button-default px-6 py-4"
    >
      <h2 class="text-lg font-medium text-text-primary">
        {{ t('profile.changePassword') }}
      </h2>
    </div>
    <div :class="props.embedded ? '' : 'px-6 py-6'">
      <form @submit.prevent="handleChangePassword" class="space-y-4">
        <div v-if="props.embedded">
          <p class="text-sm font-semibold text-text-primary">
            {{ t('profile.changePassword') }}
          </p>
        </div>
        <BoardInput
          id="old_password"
          v-model="form.old_password"
          :label="t('profile.currentPassword')"
          type="password"
          autocomplete="current-password"
          required
        />

        <BoardInput
          id="new_password"
          v-model="form.new_password"
          :label="t('profile.newPassword')"
          :hint="t('profile.passwordHint')"
          type="password"
          autocomplete="new-password"
          required
        />

        <BoardInput
          id="confirm_password"
          v-model="form.confirm_password"
          :label="t('profile.confirmNewPassword')"
          type="password"
          autocomplete="new-password"
          required
        />

        <div class="flex justify-end pt-4">
          <BoardButton type="submit" :disabled="loading" variant="primary" size="medium">
            {{ loading ? t('profile.changingPassword') : t('profile.changePasswordButton') }}
          </BoardButton>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import BoardInput from '@/components/boardui/Input.vue'
import BoardButton from '@/components/boardui/Button.vue'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { userAPI } from '@/api'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const props = withDefaults(defineProps<{
  embedded?: boolean
}>(), {
  embedded: false,
})

const loading = ref(false)
const form = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})

const handleChangePassword = async () => {
  if (form.value.new_password !== form.value.confirm_password) {
    appStore.showError(t('profile.passwordsNotMatch'))
    return
  }

  if (form.value.new_password.length < 8) {
    appStore.showError(t('profile.passwordTooShort'))
    return
  }

  loading.value = true
  try {
    await userAPI.changePassword(form.value.old_password, form.value.new_password)
    form.value = { old_password: '', new_password: '', confirm_password: '' }
    appStore.showSuccess(t('profile.passwordChangeSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('profile.passwordChangeFailed')))
  } finally {
    loading.value = false
  }
}
</script>
