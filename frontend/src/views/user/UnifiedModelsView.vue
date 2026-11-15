<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import SearchField from '@/components/boardui/SearchField.vue'
import PlazaProviderIcon from '@/components/modelPlaza/PlazaProviderIcon.vue'
import { modelProvider } from '@/components/modelPlaza/plazaProviders'
import { getUnifiedModels, type UnifiedModel, type UnifiedWalletOffer } from '@/api/apiAccess'
import { useClipboard } from '@/composables/useClipboard'
const { t } = useI18n()
const { copyToClipboard } = useClipboard()
const models = ref<UnifiedModel[]>([])
const search = ref('')
const selectedProvider = ref('all')
const loading = ref(false)
const failed = ref(false)
const catalog = computed(() => models.value.map(model => ({ ...model, provider: modelProvider(model.id, model.platform) })))
const providers = computed(() => {
  const result = new Map<string, { id: string; name: string; icon: string; count: number }>()
  for (const { provider } of catalog.value) {
    const item = result.get(provider.id)
    if (item) item.count++
    else result.set(provider.id, { ...provider, count: 1 })
  }
  return [...result.values()].sort((a, b) => a.id === 'other' ? 1 : b.id === 'other' ? -1 : a.name.localeCompare(b.name))
})
const filtered = computed(() => catalog.value.filter(model => (selectedProvider.value === 'all' || model.provider.id === selectedProvider.value)
  && model.id.toLowerCase().includes(search.value.trim().toLowerCase())))
const providerName = computed(() => selectedProvider.value === 'all' ? t('modelPlaza.filters.allProviders')
  : providers.value.find(provider => provider.id === selectedProvider.value)?.name || t('modelPlaza.filters.otherProvider'))
const endpointLabel = (endpoint: string) => endpoint === 'chat_completions' ? '/v1/chat/completions' : `/v1/${endpoint}`
const endpoints = (model: UnifiedModel) => model.wallet_offers.map(offer => endpointLabel(offer.endpoint))
// Endpoints share one backend, so offers with identical rate and pricing collapse into one price block.
const uniqueOffers = (model: UnifiedModel) => {
  const groups = new Map<string, { offer: UnifiedWalletOffer; endpoints: string[] }>()
  for (const offer of model.wallet_offers) {
    const sig = JSON.stringify([offer.rate_multiplier, offer.pricing, offer.time_pricing, offer.long_context_basis])
    const group = groups.get(sig)
    if (group) group.endpoints.push(endpointLabel(offer.endpoint))
    else groups.set(sig, { offer, endpoints: [endpointLabel(offer.endpoint)] })
  }
  return [...groups.values()]
}
const hasTiers = (model: UnifiedModel) => model.wallet_offers.some(offer => offer.pricing?.intervals?.length || offer.time_pricing)
function priceRows(offer: UnifiedWalletOffer) {
  const p = offer.pricing
  if (!p) return []
  if (p.billing_mode === 'per_request') return [{ key: 'requestPrice', value: p.per_request_price, tokens: false }]
  const rows = [{ key: 'inputPrice', value: p.input_price, tokens: true }, { key: 'outputPrice', value: p.output_price, tokens: true }]
  if (p.cache_write_price != null) rows.push({ key: 'cacheWritePrice', value: p.cache_write_price, tokens: true })
  if (p.cache_read_price != null) rows.push({ key: 'cacheReadPrice', value: p.cache_read_price, tokens: true })
  return rows
}
function reset() { selectedProvider.value = 'all'; search.value = '' }
const price = (n: number | null | undefined, rate: number, tokens = true) => n == null ? '—' : `$${(n * rate * (tokens ? 1_000_000 : 1)).toFixed(6)}`
const showBase = (rate: number) => rate !== 1
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
      <section class="flex flex-col gap-4 rounded-3xl border border-border-button-default bg-background-primary-default p-4 sm:p-5" :aria-label="t('modelPlaza.compare.filters')">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div class="space-y-1">
            <h2 class="text-body-medium text-text-primary">{{ t('modelPlaza.filters.providerLabel') }}</h2>
            <p class="text-caption-1-regular text-text-tertiary">{{ t('modelPlaza.filters.providerHint') }}</p>
          </div>
          <SearchField v-model="search" :placeholder="t('console.searchModels')" :clear-label="t('modelPlaza.compare.clearSearch')" class="w-full sm:max-w-xs" />
        </div>
        <div class="flex flex-wrap gap-2" role="group" :aria-label="t('modelPlaza.filters.providerLabel')" data-testid="provider-filters">
          <BoardButton variant="secondary" :aria-pressed="selectedProvider === 'all'" data-provider-filter="all" :disabled="loading || failed" class="h-11 gap-2 rounded-xl px-3 aria-pressed:border-border-focus-ring aria-pressed:bg-background-accent-subtle aria-pressed:text-text-accent" @click="selectedProvider = 'all'">
            <PlazaProviderIcon icon="" />{{ t('modelPlaza.filters.allProviders') }}
            <span class="text-caption-1-medium tabular-nums">{{ models.length }}</span>
          </BoardButton>
          <BoardButton v-for="provider in providers" :key="provider.id" variant="secondary" :aria-pressed="selectedProvider === provider.id" :data-provider-filter="provider.id" :disabled="loading || failed" class="h-11 gap-2 rounded-xl px-3 aria-pressed:border-border-focus-ring aria-pressed:bg-background-accent-subtle aria-pressed:text-text-accent" @click="selectedProvider = provider.id">
            <PlazaProviderIcon :icon="provider.icon" />{{ provider.name || t('modelPlaza.filters.otherProvider') }}
            <span class="text-caption-1-medium tabular-nums">{{ provider.count }}</span>
          </BoardButton>
        </div>
      </section>
      <div v-if="!loading && !failed" class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <h2 class="text-title-3-semibold text-text-primary">{{ providerName }}</h2>
          <p class="text-caption-1-regular text-text-tertiary" role="status" aria-live="polite">{{ t('modelPlaza.compare.modelCount', { count: filtered.length }) }}</p>
        </div>
        <BoardButton v-if="search || selectedProvider !== 'all'" variant="ghost" size="small" @click="reset">{{ t('modelPlaza.compare.reset') }}</BoardButton>
      </div>
      <BoardSkeleton v-if="loading" height="10rem" />
      <div v-else-if="failed" role="alert" class="card flex flex-wrap items-center gap-3 p-4"><p class="text-body-regular text-text-error-primary">{{ t('apiAccess.modelsFailed') }}</p><BoardButton @click="load">{{ t('apiAccess.retry') }}</BoardButton></div>
      <p v-else-if="!filtered.length" class="card p-6 text-body-regular text-text-secondary">{{ search || selectedProvider !== 'all' ? t('console.noMatchingModels') : t('apiAccess.noModels') }}</p>
      <section v-else class="grid min-w-0 items-stretch gap-4 lg:grid-cols-2"><article v-for="model in filtered" :key="model.id" class="flex min-w-0 flex-col gap-4 rounded-3xl border border-border-button-default bg-background-primary-default p-4 transition-colors hover:border-border-focus-ring sm:p-5" data-testid="unified-model" :data-model="model.id" :data-provider="model.provider.id">
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-border-button-default bg-background-secondary-default text-foreground-icon-primary"><PlazaProviderIcon :icon="model.provider.icon" /></span>
            <div class="min-w-0 space-y-1"><p class="text-caption-1-medium text-text-tertiary">{{ model.provider.name || t('modelPlaza.filters.otherProvider') }}</p><h3 class="break-all text-body-medium text-text-primary">{{ model.id }}</h3></div>
          </div>
          <BoardButton variant="secondary" size="small" class="shrink-0" @click="copyToClipboard(model.id, t('keys.copied'))">{{ t('apiAccess.copyModel') }}</BoardButton>
        </div>
        <div class="flex flex-wrap items-center gap-1.5">
          <BoardBadge :tone="model.subscription_available ? 'success' : 'neutral'">{{ model.subscription_available ? t('apiAccess.subscriptionAvailable') : t('console.sameKey') }}</BoardBadge>
          <BoardBadge v-for="endpoint in endpoints(model)" :key="endpoint" tone="neutral"><span class="font-mono">{{ endpoint }}</span></BoardBadge>
        </div>
        <section v-for="{ offer, endpoints: offerEndpoints } in uniqueOffers(model)" :key="offerEndpoints.join()" class="min-w-0 overflow-hidden rounded-2xl border border-border-button-default" data-testid="offer-prices">
          <header class="flex flex-wrap items-center justify-between gap-2 bg-background-secondary-default px-3 py-2">
            <span class="break-all font-mono text-caption-1-medium text-text-secondary">{{ uniqueOffers(model).length > 1 ? offerEndpoints.join(' · ') : t('apiAccess.walletPrice') }}</span>
            <BoardBadge tone="accent">{{ t('apiAccess.rateMultiplier') }} ×{{ offer.rate_multiplier }}</BoardBadge>
          </header>
          <p v-if="!offer.pricing" class="px-3 py-3 text-body-2-regular text-text-tertiary">—</p>
          <table v-else class="w-full table-fixed text-left">
            <thead><tr class="text-caption-1-regular text-text-tertiary"><th scope="col" class="px-3 py-2 font-normal">{{ offer.pricing.billing_mode === 'per_request' ? t('apiAccess.perRequest') : t('apiAccess.perMillion') }}</th><th v-if="showBase(offer.rate_multiplier)" scope="col" class="px-3 py-2 text-right font-normal">{{ t('apiAccess.basePrice') }}</th><th scope="col" class="px-3 py-2 text-right font-normal">{{ t('apiAccess.currentPrice') }}</th></tr></thead>
            <tbody>
              <tr v-for="row in priceRows(offer)" :key="row.key" class="border-t border-separator-border" :data-price-row="row.key">
                <th scope="row" class="px-3 py-2 text-body-2-regular font-normal text-text-secondary">{{ t(`apiAccess.${row.key}`) }}</th>
                <td v-if="showBase(offer.rate_multiplier)" class="break-all px-3 py-2 text-right text-body-2-regular tabular-nums text-text-tertiary line-through">{{ price(row.value, 1, row.tokens) }}</td>
                <td class="break-all px-3 py-2 text-right text-body-medium tabular-nums text-text-primary">{{ price(row.value, offer.rate_multiplier, row.tokens) }}</td>
              </tr>
            </tbody>
          </table>
        </section>
        <details v-if="hasTiers(model)" class="mt-auto rounded-xl border border-border-button-default p-3"><summary class="cursor-pointer text-body-medium text-text-secondary">{{ t('apiAccess.pricingDetails') }}</summary>
          <div class="mt-3 space-y-3">
            <template v-for="{ offer, endpoints: offerEndpoints } in uniqueOffers(model)" :key="offerEndpoints.join()">
              <div v-if="offer.pricing?.intervals?.length || offer.time_pricing" class="space-y-1">
                <p v-if="uniqueOffers(model).length > 1" class="text-body-medium text-text-primary">{{ offerEndpoints.join(' · ') }}</p>
                <p v-if="offer.long_context_basis" class="text-caption-1-regular text-text-secondary">{{ offer.long_context_basis }}</p>
                <p v-for="(tier, i) in offer.pricing?.intervals ?? []" :key="i" class="text-caption-1-regular text-text-secondary">{{ tier.min_tokens }}–{{ tier.max_tokens ?? '∞' }} · {{ t('dashboard.input') }} {{ price(tier.input_price, offer.rate_multiplier) }} · {{ t('dashboard.output') }} {{ price(tier.output_price, offer.rate_multiplier) }}</p>
                <p v-if="offer.time_pricing" class="text-caption-1-regular text-text-secondary">{{ offer.time_pricing.timezone }} · {{ t('apiAccess.timePrice') }}</p>
                <p v-for="period in offer.time_pricing?.periods ?? []" :key="period.start_time" class="text-caption-1-regular text-text-secondary">{{ period.start_time }}–{{ period.end_time }} · ×{{ period.multiplier }}</p>
              </div>
            </template>
          </div>
        </details>
      </article></section>
    </div>
  </ConsolePage>
</template>
