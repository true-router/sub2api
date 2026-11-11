<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardButton from '@/components/boardui/Button.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import PlazaModelOffer from './PlazaModelOffer.vue'
import PlazaPriceChart from './PlazaPriceChart.vue'
import type { ModelPlazaGroup, PlazaExchangeRate, PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'
import type { DisplayCurrency } from './plazaPricing'
const props = defineProps<{ name: string; platform: string; offers: { group: ModelPlazaGroup; model: PlazaModel }[]; currency: DisplayCurrency; cnyPerCredit: number; quote?: PlazaExchangeRate | null }>()
const { t } = useI18n()
const selectedGroup = ref<number | null>(null)
const offer = computed(() => props.offers.find(item => item.group.id === selectedGroup.value) ?? props.offers[0])
</script>

<template>
  <article class="flex min-w-0 flex-col overflow-hidden rounded-3xl border border-border-button-default bg-background-primary-default" :data-model="name">
    <header class="flex min-w-0 items-start gap-3 p-4 sm:p-5">
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-background-secondary-default text-foreground-icon-primary"><PlatformIcon :platform="platform as GroupPlatform" size="sm" /></span>
      <div class="flex min-w-0 flex-col gap-1"><h2 class="break-words text-title-3-semibold text-text-primary">{{ name }}</h2><p class="text-caption-1-regular text-text-tertiary">{{ platform }} · {{ t('modelPlaza.compare.offerCount', { count: offers.length }) }}</p></div>
    </header>
    <div class="mb-4 flex flex-wrap gap-1 px-4 sm:px-5" role="group" :aria-label="t('modelPlaza.compare.selectOffer')">
      <BoardButton v-for="item in offers" :key="item.group.id" :variant="offer?.group.id === item.group.id ? 'secondary' : 'ghost'" size="small" :aria-pressed="offer?.group.id === item.group.id" :data-offer-id="item.group.id" @click="selectedGroup = item.group.id">{{ item.group.name }}</BoardButton>
    </div>
    <template v-if="offer">
      <PlazaPriceChart :group="offer.group" :model="offer.model" :currency="currency" :cny-per-credit="cnyPerCredit" :quote="quote" />
      <PlazaModelOffer :group="offer.group" :model="offer.model" :currency="currency" :cny-per-credit="cnyPerCredit" :quote="quote" />
    </template>
  </article>
</template>
