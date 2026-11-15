<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { prepareAPIAccess, type APIAccess } from '@/api/apiAccess'
import BoardButton from '@/components/boardui/Button.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'

const { t } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const { copyToClipboard } = useClipboard()
const access = ref<APIAccess | null>(null)
const loading = ref(false)
const failed = ref(false)
const revealed = ref(false)
const gift = computed(() => auth.user?.bonus_balance ?? 0)
const baseURL = computed(() => (app.cachedPublicSettings?.api_base_url || window.location.origin).replace(/\/v1\/?$/, '').replace(/\/+$/, ''))
const key = computed(() => access.value?.default_key)
async function prepare() {
  loading.value = true
  failed.value = false
  try { access.value = await prepareAPIAccess() } catch { failed.value = true } finally { loading.value = false }
}
onMounted(prepare)
</script>

<template>
  <section class="space-y-4" data-testid="default-api-access">
    <div v-if="gift > 0" class="rounded-3xl border border-border-button-default bg-background-success-subtle p-4">
      <h2 class="text-title-3-semibold text-text-success">{{ t('apiAccess.gift', { amount: gift.toFixed(2) }) }}</h2>
      <p class="text-body-regular text-text-secondary">{{ t('apiAccess.giftDescription') }}</p>
    </div>
    <div class="card space-y-4 p-4 sm:p-6">
      <div>
        <h2 class="text-title-2-medium text-text-primary">{{ t('apiAccess.title') }}</h2>
        <p class="text-body-regular text-text-secondary">{{ t('apiAccess.description') }}</p>
      </div>
      <BoardSkeleton v-if="loading && !access" height="5rem" />
      <div v-else-if="failed" role="alert" class="flex flex-wrap items-center justify-between gap-3">
        <p class="text-body-regular text-text-error-primary">{{ t('apiAccess.failed') }}</p>
        <BoardButton :loading="loading" @click="prepare">{{ t('apiAccess.retry') }}</BoardButton>
      </div>
      <div v-else-if="!key" class="space-y-2">
        <p class="text-body-regular text-text-secondary">{{ t('apiAccess.revoked') }}</p>
        <router-link to="/keys" class="text-body-medium text-text-accent underline">{{ t('apiAccess.manageKeys') }}</router-link>
      </div>
      <div v-else class="space-y-2">
        <h3 class="text-body-medium text-text-primary">{{ t('apiAccess.defaultKey') }}</h3>
        <div class="flex flex-wrap items-center gap-2 rounded-xl border border-border-button-default p-3">
          <code class="min-w-0 basis-full break-all text-body-regular text-text-primary sm:flex-1 sm:basis-auto">{{ revealed ? key.key : 'sk-••••••••••••••••••••' }}</code>
          <BoardButton variant="secondary" :aria-pressed="revealed" @click="revealed = !revealed">{{ revealed ? t('apiAccess.hide') : t('apiAccess.reveal') }}</BoardButton>
          <BoardButton @click="copyToClipboard(key.key, t('apiAccess.keyCopied'))">{{ t('apiAccess.copyKey') }}</BoardButton>
        </div>
        <p v-if="key.status !== 'active'" role="status" class="text-body-regular text-text-warning">{{ t('apiAccess.keyInactive') }}</p>
      </div>
      <div class="space-y-2">
        <h3 class="text-body-medium text-text-primary">Base URL</h3>
        <div class="flex flex-wrap items-center gap-2 rounded-xl border border-border-button-default p-3">
          <code class="min-w-0 basis-full break-all text-body-regular text-text-primary sm:flex-1 sm:basis-auto">{{ baseURL }}</code>
          <BoardButton variant="secondary" @click="copyToClipboard(baseURL, t('apiAccess.baseCopied'))">{{ t('apiAccess.copyBase') }}</BoardButton>
        </div>
      </div>
      <p class="text-body-regular text-text-secondary">{{ t('apiAccess.billingRule') }}</p>

      <div class="flex flex-wrap gap-4 text-body-medium">
        <router-link to="/account/model-plaza" class="text-text-accent underline">{{ t('apiAccess.models') }}</router-link>
        <router-link to="/purchase" class="text-text-accent underline">{{ t('apiAccess.billing') }}</router-link>
      </div>
    </div>
  </section>
</template>
