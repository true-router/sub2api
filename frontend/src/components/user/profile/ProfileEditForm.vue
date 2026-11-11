<template>
  <div :class="props.embedded ? 'space-y-4' : 'card'">
    <div
      v-if="!props.embedded"
      class="border-b border-border-button-default px-6 py-4"
    >
      <h2 class="text-lg font-medium text-text-primary">
        {{ t('profile.editProfile') }}
      </h2>
    </div>
    <div :class="props.embedded ? '' : 'px-6 py-6'">
      <form @submit.prevent="handleUpdateProfile" class="space-y-4">
        <div v-if="props.embedded">
          <p class="text-sm font-semibold text-text-primary">
            {{ t('profile.editProfile') }}
          </p>
        </div>
        <BoardInput
          v-model="username"
          :label="t('profile.username')"
          id="username"
          :placeholder="t('profile.enterUsername')"
        />

        <div class="flex justify-end pt-4">
          <BoardButton type="submit" :disabled="loading" variant="primary" size="medium">
            {{ loading ? t('profile.updating') : t('profile.updateProfile') }}
          </BoardButton>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import BoardInput from '@/components/boardui/Input.vue'
import BoardButton from '@/components/boardui/Button.vue'
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { userAPI } from '@/api'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = withDefaults(defineProps<{
  initialUsername: string
  embedded?: boolean
}>(), {
  embedded: false,
})

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const username = ref(props.initialUsername)
let savedUsername = props.initialUsername
const loading = ref(false)

watch(() => props.initialUsername, (val) => {
  if (username.value === savedUsername) username.value = val
  savedUsername = val
})

const handleUpdateProfile = async () => {
  if (!username.value.trim()) {
    appStore.showError(t('profile.usernameRequired'))
    return
  }

  loading.value = true
  const submittedUsername = username.value
  try {
    const updatedUser = await userAPI.updateProfile({
      username: submittedUsername
    })
    if (username.value === submittedUsername) username.value = updatedUser.username
    savedUsername = updatedUser.username
    authStore.user = updatedUser
    appStore.showSuccess(t('profile.updateSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('profile.updateFailed')))
  } finally {
    loading.value = false
  }
}
</script>
