<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import SegmentedControl from '@/components/boardui/SegmentedControl.vue'
import SearchField from '@/components/boardui/SearchField.vue'
import BoardButton from '@/components/boardui/Button.vue'
import Badge from '@/components/boardui/Badge.vue'
import Skeleton from '@/components/boardui/Skeleton.vue'
import PlazaModelCard from './PlazaModelCard.vue'
import { creditCost, validQuote, type DisplayCurrency } from './plazaPricing'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel } from '@/api/modelPlaza'
import { useAuthStore } from '@/stores/auth'
const props = defineProps<{ response: ModelPlazaResponse | null; loading: boolean; error: boolean; embedded?: boolean }>()
const { t } = useI18n()
const auth = useAuthStore()
const selectedCurrency = ref('CNY')
const platform = ref('all')
const group = ref<number | 'all'>('all')
const search = ref('')
const quote = computed(() => validQuote(props.response?.exchange_rate) ? props.response!.exchange_rate! : null)
const cnyPerCredit = computed(() => creditCost(props.response?.cny_per_credit))
const currency = computed<DisplayCurrency>(() => selectedCurrency.value === 'USD' && quote.value ? 'USD' : 'CNY')
const currencyItems = computed(() => [{ value: 'CNY', label: t('modelPlaza.compare.cny') }, { value: 'USD', label: t('modelPlaza.compare.usd'), disabled: !quote.value }])
const groups = computed(() => props.response?.groups ?? [])
const platforms = computed(() => [...new Set(groups.value.flatMap(g => g.models.map(m => m.platform)))].sort())
const descriptionHtml = computed(() => DOMPurify.sanitize(marked.parse(props.response?.description?.trim() || '') as string))
interface ModelEntry { key: string; name: string; platform: string; offers: { group: ModelPlazaGroup; model: PlazaModel }[] }
const catalog = computed(() => {
  const byModel = new Map<string, ModelEntry>()
  const query = search.value.trim().toLowerCase()
  for (const g of groups.value) {
    if (group.value !== 'all' && group.value !== g.id) continue
    for (const m of g.models) {
      if (platform.value !== 'all' && platform.value !== m.platform) continue
      if (query && !m.name.toLowerCase().includes(query)) continue
      const key = `${m.platform}:${m.name}`
      if (!byModel.has(key)) byModel.set(key, { key, name: m.name, platform: m.platform, offers: [] })
      byModel.get(key)!.offers.push({ group: g, model: m })
    }
  }
  return [...byModel.values()].sort((a, b) => a.platform.localeCompare(b.platform) || b.name.localeCompare(a.name))
})
function reset() { platform.value = 'all'; group.value = 'all'; search.value = '' }
</script>

<template>
  <div class="flex min-w-0 flex-col gap-4" data-testid="plaza-catalog">
    <header class="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
      <div class="flex min-w-0 flex-col gap-2">
        <p class="text-caption-1-medium text-text-accent">{{ t('modelPlaza.compare.eyebrow') }}</p>
        <h1 v-if="!embedded" class="text-title-1-medium text-text-primary">{{ t('modelPlaza.title') }}</h1>
        <p class="max-w-2xl text-body-regular text-text-secondary">{{ t('modelPlaza.compare.intro') }}</p>
      </div>
      <SegmentedControl v-model="selectedCurrency" :items="currencyItems" :label="t('modelPlaza.compare.currency')" data-testid="currency-switch" class="self-start" />
    </header>

    <section class="flex flex-col gap-2 rounded-2xl border border-border-button-default bg-background-primary-default px-4 py-3" :aria-label="t('modelPlaza.compare.basis')">
      <div class="flex flex-wrap items-center gap-x-3 gap-y-2">
        <Badge tone="accent">{{ t('modelPlaza.compare.rechargeIncluded') }}</Badge>
        <span class="text-caption-1-medium tabular-nums text-text-primary">{{ t('modelPlaza.compare.rechargeBasis', { value: cnyPerCredit }) }}</span>
        <span v-if="quote" class="text-caption-1-regular text-text-secondary">US$1 = ¥{{ quote.usd_to_cny }}<span v-if="quote.stale"> · {{ t('modelPlaza.compare.cached') }}</span></span>
      </div>
      <p v-if="!quote" class="text-body-2-regular text-text-warning" role="status">{{ t('modelPlaza.compare.noExchange') }}</p>
      <details>
        <summary class="cursor-pointer rounded-md text-caption-1-regular text-text-tertiary outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring">{{ t('modelPlaza.compare.basis') }}</summary>
        <div class="mt-2 flex flex-col gap-2">
          <p v-if="quote" class="text-caption-1-regular text-text-secondary">{{ t('modelPlaza.compare.exchange', { rate: quote.usd_to_cny, date: quote.date, source: quote.source }) }}</p>
          <p class="text-caption-1-regular text-text-tertiary">{{ t('modelPlaza.compare.disclaimer') }}</p>
          <p v-if="!auth.isAuthenticated" class="text-caption-1-regular text-text-tertiary">{{ t('modelPlaza.anonymousHint') }}</p>
          <div v-if="descriptionHtml" class="plaza-description text-body-2-regular text-text-secondary" v-html="descriptionHtml" />
        </div>
      </details>
    </section>

    <div v-if="loading" class="flex flex-col gap-4" role="status" :aria-label="t('modelPlaza.loading')"><Skeleton height="256px" class="w-full rounded-3xl" /><Skeleton height="256px" class="w-full rounded-3xl" /></div>
    <p v-else-if="error" role="alert" class="rounded-3xl border border-border-error-default bg-background-error-subtle p-6 text-body-regular text-text-error-primary">{{ t('modelPlaza.loadFailed') }}</p>
    <template v-else>
      <section class="flex flex-col gap-3" :aria-label="t('modelPlaza.compare.filters')">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <SearchField v-model="search" :placeholder="t('modelPlaza.filters.searchPlaceholder')" :clear-label="t('modelPlaza.compare.clearSearch')" class="max-w-md" />
        <div class="flex flex-wrap items-center gap-2" role="group" :aria-label="t('modelPlaza.filters.platformLabel')">
          <BoardButton :variant="platform === 'all' ? 'primary' : 'secondary'" size="small" :aria-pressed="platform === 'all'" @click="platform = 'all'">{{ t('modelPlaza.filters.all') }}</BoardButton>
          <BoardButton v-for="p in platforms" :key="p" :variant="platform === p ? 'primary' : 'secondary'" size="small" :aria-pressed="platform === p" @click="platform = p">{{ p }}</BoardButton>
        </div>
        </div>
        <div class="flex flex-wrap items-center gap-2" role="group" :aria-label="t('modelPlaza.filters.groupLabel')">
          <span class="text-caption-1-medium text-text-tertiary">{{ t('modelPlaza.filters.groupLabel') }}</span>
          <BoardButton :variant="group === 'all' ? 'secondary' : 'ghost'" size="small" :aria-pressed="group === 'all'" @click="group = 'all'">{{ t('modelPlaza.filters.all') }}</BoardButton>
          <BoardButton v-for="g in groups" :key="g.id" :variant="group === g.id ? 'secondary' : 'ghost'" size="small" :aria-pressed="group === g.id" @click="group = g.id">{{ g.name }}</BoardButton>
        </div>
      </section>
      <div class="flex items-center justify-between gap-3">
        <p class="text-body-2-regular text-text-secondary" role="status" aria-live="polite">{{ t('modelPlaza.compare.modelCount', { count: catalog.length }) }}</p>
        <BoardButton v-if="search || platform !== 'all' || group !== 'all'" variant="ghost" size="small" @click="reset">{{ t('modelPlaza.compare.reset') }}</BoardButton>
      </div>
      <div v-if="!catalog.length" class="rounded-3xl border border-border-button-default bg-background-primary-default p-8 text-center text-body-regular text-text-secondary">{{ groups.length ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}</div>
      <div class="grid min-w-0 items-start gap-5 xl:grid-cols-2" data-testid="model-card-grid">
        <PlazaModelCard v-for="entry in catalog" :key="entry.key" :name="entry.name" :platform="entry.platform" :offers="entry.offers" :currency="currency" :cny-per-credit="cnyPerCredit" :quote="quote" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.plaza-description :deep(a) { @apply text-text-accent underline; }
.plaza-description :deep(p + p) { @apply mt-3; }
.plaza-description :deep(ul) { @apply list-disc ps-5; }
.plaza-description :deep(ol) { @apply list-decimal ps-5; }
.plaza-description :deep(h1), .plaza-description :deep(h2), .plaza-description :deep(h3) { @apply mb-2 mt-4 text-title-3-semibold text-text-primary; }
</style>
