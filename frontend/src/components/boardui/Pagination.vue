<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardButton from './Button.vue'
import Icon from '@/components/icons/Icon.vue'
const props = defineProps<{ page: number; totalPages: number; items: (number | string)[] }>()
const emit = defineEmits<{ 'update:page': [value: number] }>()
const { t } = useI18n()
const nav = ref<HTMLElement>()
function select(value: number) { if (value > 0 && value <= props.totalPages && value !== props.page) emit('update:page', value) }
async function keydown(event: KeyboardEvent) {
  if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
  event.preventDefault()
  select(event.key === 'Home' ? 1 : event.key === 'End' ? props.totalPages : props.page + (event.key === 'ArrowRight' ? 1 : -1))
  await nextTick()
  nav.value?.querySelector<HTMLButtonElement>('[aria-current="page"]')?.focus()
}
</script>

<template>
  <nav ref="nav" data-boardui="pagination" :aria-label="t('boardui.pagination')" class="flex items-center gap-1" @keydown="keydown">
    <BoardButton variant="secondary" size="small" :disabled="page <= 1" :aria-label="t('pagination.previous')" @click="select(page - 1)"><Icon name="arrowLeft" size="sm" /><span class="hidden lg:inline">{{ t('pagination.previous') }}</span></BoardButton>
    <template v-for="(item,index) in items" :key="index">
      <BoardButton v-if="typeof item === 'number'" size="small" :variant="item === page ? 'secondary' : 'ghost'" :aria-current="item === page ? 'page' : undefined" :aria-label="t('pagination.goToPage', {page:item})" class="w-8 px-0" @click="select(item)">{{ item }}</BoardButton>
      <span v-else class="px-1 text-body-regular text-text-tertiary" aria-hidden="true">…</span>
    </template>
    <BoardButton variant="secondary" size="small" :disabled="page >= totalPages" :aria-label="t('pagination.next')" @click="select(page + 1)"><span class="hidden lg:inline">{{ t('pagination.next') }}</span><Icon name="arrowRight" size="sm" /></BoardButton>
  </nav>
</template>
