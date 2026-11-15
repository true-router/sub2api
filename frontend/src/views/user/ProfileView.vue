<template>
  <ConsolePage :title="t('console.account')" :description="t('console.accountDescription')">
    <div class="space-y-5" data-testid="profile-shell">
      <section class="card space-y-4 p-5" data-testid="account-profile"><div><h2 class="text-title-3-semibold text-text-primary">{{ t('console.profile') }}</h2><p class="break-all text-body-regular text-text-secondary">{{ user?.email }}</p></div><ProfileEditForm :initial-username="user?.username || ''" embedded /></section>
      <section class="space-y-3" data-testid="account-security"><div><h2 class="text-title-3-semibold text-text-primary">{{ t('console.security') }}</h2><p class="text-body-regular text-text-secondary">{{ t('console.securityDescription') }}</p></div>
        <details class="rounded-3xl border border-border-button-default p-5"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.password') }}</summary><div class="mt-4"><ProfilePasswordForm /></div></details>
        <details class="rounded-3xl border border-border-button-default p-5"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.twoFactor') }}</summary><div class="mt-4"><ProfileTotpCard /></div></details>
        <details v-if="passkeyEnabled" class="rounded-3xl border border-border-button-default p-5"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.passkeys') }}</summary><div class="mt-4"><ProfilePasskeyCard enabled /></div></details>
        <details class="rounded-3xl border border-border-button-default p-5"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.bindings') }}</summary><div class="mt-4"><ProfileIdentityBindingsSection :user="user" :linuxdo-enabled="linuxdoOAuthEnabled" :dingtalk-enabled="dingtalkOAuthEnabled" :oidc-enabled="oidcOAuthEnabled" :oidc-provider-name="oidcOAuthProviderName" :wechat-enabled="wechatOAuthEnabled" :wechat-open-enabled="wechatOAuthOpenEnabled" :wechat-mp-enabled="wechatOAuthMPEnabled" embedded compact /></div></details>
      </section>
      <details class="rounded-3xl border border-border-button-default p-5" data-testid="account-optional"><summary class="cursor-pointer text-body-medium text-text-primary">{{ t('console.optionalFeatures') }}</summary><div class="mt-5 space-y-5">
        <ProfileAvatarCard :user="user" />
        <ProfileBalanceNotifyCard v-if="user && balanceLowNotifyEnabled" :enabled="user.balance_notify_enabled ?? true" :threshold="user.balance_notify_threshold" :extra-emails="user.balance_notify_extra_emails ?? []" :system-default-threshold="systemDefaultThreshold" :user-email="user.email" />
        <div class="flex flex-wrap gap-4 text-body-medium text-text-accent">
          <router-link v-if="canUseBatchImage" to="/batch-image" class="underline">{{ t('nav.batchImage') }}</router-link>
          <router-link v-if="resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.availableChannels)" to="/available-channels" class="underline">{{ t('nav.availableChannels') }}</router-link>
          <router-link v-if="resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.affiliate)" to="/affiliate" class="underline">{{ t('nav.affiliate') }}</router-link>
          <router-link v-if="resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.mcp)" to="/settings/mcp" class="underline">{{ t('mcp.title') }}</router-link>
          <router-link v-if="resolveFeatureFlag(appStore.cachedPublicSettings, FeatureFlags.channelMonitor)" to="/monitor" class="underline">{{ t('console.serviceStatus') }}</router-link>
          <router-link v-for="item in appStore.cachedPublicSettings?.custom_menu_items?.filter(item => item.visibility === 'user') || []" :key="item.id" :to="`/custom/${item.id}`" class="underline">{{ item.label }}</router-link>
        </div>
      </div></details>
      <section v-if="contactInfo" class="card flex items-start gap-3 p-5"><Icon name="chat" size="md" class="text-foreground-icon-secondary" /><div><h2 class="text-body-medium text-text-primary">{{ t('common.contactSupport') }}</h2><p class="break-all text-body-regular text-text-secondary">{{ contactInfo }}</p></div></section>
    </div>
  </ConsolePage>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Icon } from '@/components/icons'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import ProfileBalanceNotifyCard from '@/components/user/profile/ProfileBalanceNotifyCard.vue'
import ProfileEditForm from '@/components/user/profile/ProfileEditForm.vue'
import ProfileAvatarCard from '@/components/user/profile/ProfileAvatarCard.vue'
import ProfileIdentityBindingsSection from '@/components/user/profile/ProfileIdentityBindingsSection.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import ProfilePasswordForm from '@/components/user/profile/ProfilePasswordForm.vue'
import ProfileTotpCard from '@/components/user/profile/ProfileTotpCard.vue'
import ProfilePasskeyCard from '@/components/user/profile/ProfilePasskeyCard.vue'
import { isWeChatWebOAuthEnabled } from '@/api/auth'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const user = computed(() => authStore.user)

const contactInfo = ref('')
const balanceLowNotifyEnabled = ref(false)
const systemDefaultThreshold = ref(0)
const linuxdoOAuthEnabled = ref(false)
const dingtalkOAuthEnabled = ref(false)
const wechatOAuthEnabled = ref(false)
const wechatOAuthOpenEnabled = ref<boolean | undefined>(undefined)
const wechatOAuthMPEnabled = ref<boolean | undefined>(undefined)
const oidcOAuthEnabled = ref(false)
const oidcOAuthProviderName = ref('OIDC')
const passkeyEnabled = ref(false)

onMounted(async () => {
  void refreshBatchImageAccess()
  const profileRefresh = authStore.refreshUser().catch((error) => {
    console.error('Failed to refresh profile:', error)
  })

  const settingsLoad = appStore.fetchPublicSettings()
    .then((settings) => {
      if (!settings) {
        return
      }
      contactInfo.value = settings.contact_info || ''
      balanceLowNotifyEnabled.value = settings.balance_low_notify_enabled ?? false
      systemDefaultThreshold.value = settings.balance_low_notify_threshold ?? 0
      linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled ?? false
      dingtalkOAuthEnabled.value = settings.dingtalk_oauth_enabled ?? false
      wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
      wechatOAuthOpenEnabled.value = typeof settings.wechat_oauth_open_enabled === 'boolean'
        ? settings.wechat_oauth_open_enabled
        : undefined
      wechatOAuthMPEnabled.value = typeof settings.wechat_oauth_mp_enabled === 'boolean'
        ? settings.wechat_oauth_mp_enabled
        : undefined
      oidcOAuthEnabled.value = settings.oidc_oauth_enabled ?? false
      oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
      passkeyEnabled.value = settings.passkey_enabled === true
    })
    .catch((error) => {
      console.error('Failed to load settings:', error)
    })

  await Promise.all([profileRefresh, settingsLoad])
})
</script>
