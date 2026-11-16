<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore, useAppStore } from '@/stores'
import { FeatureFlags, resolveFeatureFlag } from '@/utils/featureFlags'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import BillingCheckout from '@/components/user/console/BillingCheckout.vue'
import BalanceBreakdown from '@/components/user/BalanceBreakdown.vue'
import BoardPillTabs from '@/components/boardui/PillTabs.vue'
import BillingSubscriptions from '@/components/user/console/BillingSubscriptions.vue'
const RedeemView = defineAsyncComponent(() => import('./RedeemView.vue'))
const UserOrdersView = defineAsyncComponent(() => import('./UserOrdersView.vue'))
const { t } = useI18n()
const auth = useAuthStore()
const app = useAppStore()
const route = useRoute()
const router = useRouter()
type Tab = 'recharge' | 'subscription' | 'redeem' | 'orders'
const validTabs: Tab[] = ['recharge', 'subscription', 'redeem', 'orders']
const initial = validTabs.includes(route.query.tab as Tab) ? route.query.tab as Tab : 'recharge'
const tab = ref<Tab>(initial)
const mode = ref<'recharge' | 'subscription'>(initial === 'subscription' ? 'subscription' : 'recharge')
const busy = ref(false)
function setBusy(value: boolean) { busy.value = value; if (value) tab.value = mode.value }
const paymentsEnabled = computed(() => resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.payment))
const items = computed(() => [
  ...(paymentsEnabled.value ? [{ key: 'recharge' as Tab, label: t('console.addFunds') }] : []),
  ...(resolveFeatureFlag(app.cachedPublicSettings, FeatureFlags.subscription) ? [{key: 'subscription' as Tab, label: t('console.plans')}] : []),
  {key: 'redeem' as Tab, label: t('console.redeem')}, ...(paymentsEnabled.value ? [{key: 'orders' as Tab, label: t('console.orders')}] : []),
])
watch(items, available => { if (!busy.value && !available.some(item => item.key === tab.value)) { tab.value = available[0]?.key ?? 'redeem'; if (tab.value === 'recharge' || tab.value === 'subscription') mode.value = tab.value } }, { immediate: true })
function changeTab(value: Tab) {
  if (busy.value || !items.value.some(item => item.key === value)) return
  tab.value = value
  if (value === 'recharge' || value === 'subscription') mode.value = value
  void router.replace({query:{...route.query,tab:value}})
}
watch(mode, value => { if (tab.value === 'recharge' || tab.value === 'subscription') tab.value = value })
watch(() => route.query.tab, value => { if (!busy.value && items.value.some(item => item.key === value)) { tab.value = value as Tab; if(value === 'recharge' || value === 'subscription') mode.value = value } })
onMounted(() => { void auth.refreshUser() })
</script>
<template>
  <ConsolePage :title="t('console.billing')" :description="t('console.billingDescription')">
    <section class="card space-y-3 p-5" data-testid="billing-balance"><h2 class="text-body-medium text-text-secondary">{{ t('console.availableBalance') }}</h2><p class="text-title-1-semibold tabular-nums text-text-primary">P {{ (auth.user?.balance ?? 0).toFixed(2) }}</p><BalanceBreakdown :user="auth.user" /></section>
    <div class="space-y-2 rounded-3xl border border-border-button-default bg-background-secondary-default p-4"><p class="text-body-regular text-text-secondary">{{ t('apiAccess.billingRule') }}</p><router-link to="/account/model-plaza" class="inline-block text-body-medium text-text-accent underline">{{ t('apiAccess.models') }}</router-link></div>
    <section class="space-y-5" data-testid="billing-tabs">
      <BoardPillTabs :model-value="tab" :items="items" :label="t('console.billing')" id-prefix="billing-tab" panel-id="billing-panel" @update:model-value="changeTab" />
      <p v-if="busy" role="status" class="text-body-regular text-text-warning">{{ t('console.paymentInProgress') }}</p>
      <div id="billing-panel" role="tabpanel" :aria-labelledby="`billing-tab-${tab}`" class="space-y-5">
        <BillingSubscriptions v-if="tab === 'subscription'" />
        <!-- Keep checkout mounted so pending payments and provider recovery survive tab changes. -->
        <BillingCheckout v-if="paymentsEnabled" v-show="tab === 'recharge' || tab === 'subscription'" v-model:mode="mode" @busy="setBusy" />
        <RedeemView v-if="tab === 'redeem'" embedded />
        <UserOrdersView v-if="tab === 'orders'" embedded />
      </div>
    </section>
  </ConsolePage>
</template>
