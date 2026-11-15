<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import { getUnifiedModels, type UnifiedModel } from '@/api/apiAccess'
import { useClipboard } from '@/composables/useClipboard'
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const models = ref<UnifiedModel[]>([])
const search = ref('')
const loading = ref(false)
const failed = ref(false)
const filtered = computed(() => models.value.filter(m => m.id.toLowerCase().includes(search.value.toLowerCase())))
const price = (n: number | null | undefined, rate: number, tokens = true) => n == null ? '—' : `$${(n * rate * (tokens ? 1_000_000 : 1)).toFixed(6)}`
async function load() {
  loading.value = true
  failed.value = false
  try { models.value = await getUnifiedModels() } catch { failed.value = true } finally { loading.value = false }
}
onMounted(load)
</script>
<template>
  <ConsolePage :title="t('console.models')" :description="t('console.modelsDescription')">
    <div class="space-y-4">
      <p class="rounded-3xl border border-border-button-default bg-background-secondary-default p-4 text-body-regular text-text-secondary">{{ t('apiAccess.billingRule') }}</p>
      <SearchInput v-model="search" :placeholder="t('console.searchModels')" />
      <BoardSkeleton v-if="loading" height="10rem" />
      <div v-else-if="failed" role="alert" class="card flex flex-wrap items-center gap-3 p-4"><p class="text-body-regular text-text-error-primary">{{ t('apiAccess.modelsFailed') }}</p><BoardButton @click="load">{{ t('apiAccess.retry') }}</BoardButton></div>
      <p v-else-if="!filtered.length" class="card p-6 text-body-regular text-text-secondary">{{ search ? t('console.noMatchingModels') : t('apiAccess.noModels') }}</p>
      <section v-else class="card divide-y divide-separator-border px-5"><article v-for="model in filtered" :key="model.id" class="space-y-4 py-5" data-testid="unified-model">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="min-w-0"><h2 class="break-all text-title-3-semibold text-text-primary">{{ model.id }}</h2><p class="text-caption-1-regular text-text-secondary">{{ model.platform }}</p></div>
          <BoardButton variant="secondary" @click="copyToClipboard(model.id, t('keys.copied'))">{{ t('apiAccess.copyModel') }}</BoardButton>
        </div>
        <p class="break-all text-caption-1-regular text-text-secondary">{{ model.wallet_offers.map(offer => offer.endpoint === 'chat_completions' ? '/v1/chat/completions' : `/v1/${offer.endpoint}`).join(' · ') }}</p>
        <BoardBadge :tone="model.subscription_available ? 'success' : 'neutral'">{{ model.subscription_available ? t('apiAccess.subscriptionAvailable') : t('console.sameKey') }}</BoardBadge>
        <details class="rounded-xl border border-border-button-default p-3"><summary class="cursor-pointer text-body-medium text-text-secondary">{{ t('console.walletPrices') }}</summary>
        <div class="mt-3 space-y-3"><section v-for="offer in model.wallet_offers" :key="offer.endpoint" class="space-y-2 rounded-xl border border-border-button-default p-3">
          <h3 class="text-body-medium text-text-primary">{{ offer.endpoint }} · {{ t('apiAccess.walletPrice') }}</h3>
          <p v-if="offer.pricing?.billing_mode === 'per_request'" class="text-body-regular text-text-primary">{{ price(offer.pricing.per_request_price, offer.rate_multiplier, false) }} / {{ t('apiAccess.perRequest') }}</p>
          <dl v-else class="grid gap-2 text-body-regular sm:grid-cols-2">
            <div><dt class="text-text-secondary">{{ t('apiAccess.inputPrice') }}</dt><dd>{{ price(offer.pricing?.input_price, offer.rate_multiplier) }}</dd></div>
            <div><dt class="text-text-secondary">{{ t('apiAccess.outputPrice') }}</dt><dd>{{ price(offer.pricing?.output_price, offer.rate_multiplier) }}</dd></div>
            <div v-if="offer.pricing?.cache_read_price != null"><dt class="text-text-secondary">{{ t('apiAccess.cacheReadPrice') }}</dt><dd>{{ price(offer.pricing.cache_read_price, offer.rate_multiplier) }}</dd></div>
            <div v-if="offer.pricing?.cache_write_price != null"><dt class="text-text-secondary">{{ t('apiAccess.cacheWritePrice') }}</dt><dd>{{ price(offer.pricing.cache_write_price, offer.rate_multiplier) }}</dd></div>
          </dl>
          <details v-if="offer.pricing?.intervals?.length || offer.time_pricing" class="rounded-xl border border-border-button-default p-3 text-body-regular">
            <summary class="cursor-pointer text-text-secondary">{{ t('apiAccess.pricingDetails') }}</summary>
            <p v-if="offer.long_context_basis" class="text-caption-1-regular text-text-secondary">{{ offer.long_context_basis }}</p>
            <p v-for="(tier, i) in offer.pricing?.intervals ?? []" :key="i" class="text-caption-1-regular text-text-secondary">{{ tier.min_tokens }}–{{ tier.max_tokens ?? '∞' }} · {{ t('dashboard.input') }} {{ price(tier.input_price, offer.rate_multiplier) }} · {{ t('dashboard.output') }} {{ price(tier.output_price, offer.rate_multiplier) }}</p>
            <p v-if="offer.time_pricing" class="text-caption-1-regular text-text-secondary">{{ offer.time_pricing.timezone }} · {{ t('apiAccess.timePrice') }}</p>
            <p v-for="period in offer.time_pricing?.periods ?? []" :key="period.start_time" class="text-caption-1-regular text-text-secondary">{{ period.start_time }}–{{ period.end_time }} · ×{{ period.multiplier }}</p>
          </details>
        </section>
        </div></details>
      </article></section>
    </div>
  </ConsolePage>
</template>
