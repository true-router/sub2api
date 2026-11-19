<script setup lang="ts">
import SubscriptionSlotUsage from '@/components/common/SubscriptionSlotUsage.vue'
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import subscriptionsAPI from '@/api/subscriptions'
import type { UserSubscription } from '@/types'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import { formatDateTime } from '@/utils/format'
const { t } = useI18n()
const subscriptions = ref<UserSubscription[]>([])
const loading = ref(false)
const failed = ref(false)
async function load() { loading.value=true; failed.value=false; try { subscriptions.value=await subscriptionsAPI.getMySubscriptions() } catch { failed.value=true } finally { loading.value=false } }
onMounted(load)
</script>
<template>
  <section class="card space-y-4 p-5" data-testid="billing-subscriptions">
    <h2 class="text-title-3-semibold text-text-primary">{{ t('console.yourPlans') }}</h2>
    <BoardSkeleton v-if="loading" height="4rem" />
    <div v-else-if="failed" role="alert" class="space-y-2"><p class="text-body-regular text-text-error-primary">{{ t('console.loadFailed') }}</p><BoardButton variant="secondary" @click="load">{{ t('apiAccess.retry') }}</BoardButton></div>
    <p v-else-if="!subscriptions.length" class="text-body-regular text-text-secondary">{{ t('console.noPlan') }}</p>
    <article v-for="sub in subscriptions" v-else :key="sub.id" class="space-y-3 rounded-xl border border-border-button-default p-4">
      <div class="flex flex-wrap items-start justify-between gap-3"><div><h3 class="text-body-medium text-text-primary">{{ sub.group?.name }}</h3><p class="text-caption-1-regular text-text-secondary">{{ t('console.expires', {date: sub.expires_at ? formatDateTime(sub.expires_at) : t('userSubscriptions.noExpiration')}) }}</p></div><BoardBadge :tone="sub.status === 'active' && (!sub.expires_at || new Date(sub.expires_at).getTime() > Date.now()) ? 'success' : 'neutral'">{{ t(`userSubscriptions.status.${sub.status}`) }}</BoardBadge></div>
      <SubscriptionSlotUsage v-if="sub.slot_usage" :usage="sub.slot_usage" />
            <div v-else class="grid gap-3 sm:grid-cols-3"><template v-for="window in (['daily','weekly','monthly'] as const)" :key="window"><div v-if="sub.group?.[`${window}_limit_usd`] != null" class="space-y-2"><p class="text-caption-1-regular text-text-secondary">{{ t(`console.${window}`) }}</p><p class="text-body-medium text-text-primary">P {{ sub[`${window}_usage_usd`].toFixed(2) }} / P {{ sub.group[`${window}_limit_usd`]?.toFixed(2) }}</p><progress class="h-1.5 w-full accent-background-accent-default" :aria-label="t(`console.${window}`)" :value="sub[`${window}_usage_usd`]" :max="sub.group[`${window}_limit_usd`] || 1" /></div></template></div>
    </article>
    <p class="text-caption-1-regular text-text-secondary">{{ t('console.planKeyNote') }}</p>
  </section>
</template>
