<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { getSubscriptionSummary } from '@/api/subscriptions'
import { useAppStore } from '@/stores/app'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import DefaultAPIAccessCard from '@/components/user/dashboard/DefaultAPIAccessCard.vue'
import BalanceBreakdown from '@/components/user/BalanceBreakdown.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
const { t } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const planCount = ref<number | null>(null)
const stats = ref<UserDashboardStats | null>(null)
const failed = ref(false)
const loading = ref(false)
const balance = computed(() => auth.user?.balance ?? 0)
async function load() {
  loading.value = true; failed.value = false
  const results = await Promise.allSettled([auth.refreshUser(), usageAPI.getDashboardStats(), resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription) ? getSubscriptionSummary() : Promise.resolve({active_count:0})])
  if (results[1].status === 'fulfilled') stats.value = results[1].value
  if (results[2].status === 'fulfilled') planCount.value = results[2].value.active_count
  failed.value = results.some(r => r.status === 'rejected')
  loading.value = false
}
onMounted(load)
</script>
<template>
  <ConsolePage :title="t('console.home')" :description="t('console.homeDescription')">
    <section v-if="balance <= 0 && planCount === 0 && !auth.isSimpleMode" class="space-y-2 rounded-3xl border border-border-button-default bg-background-warning-subtle p-4" data-testid="home-funding-needed"><h2 class="text-body-semibold text-text-warning">{{ t('console.fundingNeeded') }}</h2><p class="text-body-regular text-text-secondary">{{ t('console.fundingNote') }}</p><router-link to="/purchase" class="inline-block text-body-medium text-text-accent underline">{{ t('console.addFunds') }}</router-link></section>
    <section v-else-if="planCount && !auth.isSimpleMode" class="space-y-2 rounded-3xl border border-border-button-default bg-background-accent-subtle p-4"><h2 class="text-body-semibold text-text-primary">{{ t('console.activePlans', {count: planCount}) }}</h2><router-link to="/subscriptions" class="inline-block text-body-medium text-text-accent underline">{{ t('console.yourPlans') }}</router-link></section>
    <DefaultAPIAccessCard v-if="!auth.isSimpleMode" />
    <div class="grid gap-4 sm:grid-cols-2">
      <section class="card space-y-3 p-5" data-testid="home-balance">
        <h2 class="text-body-medium text-text-secondary">{{ t('console.availableBalance') }}</h2>
        <p class="text-title-1-semibold tabular-nums text-text-primary">${{ balance.toFixed(2) }}</p>
        <BalanceBreakdown :user="auth.user" />
        <router-link to="/purchase" class="inline-block text-body-medium text-text-accent underline">{{ t('console.manageBilling') }}</router-link>
      </section>
      <section class="card space-y-3 p-5" data-testid="home-activity">
        <h2 class="text-title-3-semibold text-text-primary">{{ t('console.recentActivity') }}</h2>
        <BoardSkeleton v-if="loading && !stats" height="3rem" />
        <template v-else-if="stats && stats.total_requests > 0">
          <p class="text-body-regular text-text-secondary">{{ t('console.todayActivity', { count: stats.today_requests, cost: stats.today_actual_cost.toFixed(4) }) }}</p>
          <router-link to="/usage" class="inline-block text-body-medium text-text-accent underline">{{ t('console.viewUsage') }}</router-link>
        </template>
        <p v-else-if="!failed" class="text-body-regular text-text-secondary">{{ t('console.readyWhenYouAre') }}</p>
        <div v-if="failed" role="alert" class="space-y-2"><p class="text-body-regular text-text-error-primary">{{ t('console.loadFailed') }}</p><BoardButton variant="secondary" :loading="loading" @click="load">{{ t('apiAccess.retry') }}</BoardButton></div>
      </section>
    </div>
  </ConsolePage>
</template>
