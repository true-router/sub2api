<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import BillingSubscriptions from '@/components/user/console/BillingSubscriptions.vue'
const { t } = useI18n()
const app = useAppStore()
</script>
<template>
  <ConsolePage :title="t('console.yourPlans')" :description="t('console.planKeyNote')">
    <BillingSubscriptions />
    <router-link v-if="resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment)" to="/purchase?tab=subscription" class="inline-block text-body-medium text-text-accent underline">{{ t('console.plans') }}</router-link>
  </ConsolePage>
</template>
