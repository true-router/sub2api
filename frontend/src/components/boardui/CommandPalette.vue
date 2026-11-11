<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { cx } from '@/utils/cx'

export interface CommandItem { id: string; label: string; section: string; keywords?: string }
const props = defineProps<{ show: boolean; items: CommandItem[] }>()
const emit = defineEmits<{ close: []; select: [id: string] }>()
const { t } = useI18n()
const query = ref('')
const active = ref(0)
const input = ref<HTMLInputElement>()
const id = useId()
const filtered = computed(() => {
  const value = query.value.trim().toLocaleLowerCase()
  return props.items.filter(item => !value || `${item.label} ${item.section} ${item.keywords || ''}`.toLocaleLowerCase().includes(value))
})
watch(query, () => { active.value = 0 })
watch(() => props.show, async show => { if (show) { query.value = ''; active.value = 0; await nextTick(); input.value?.focus() } })
function move(event: KeyboardEvent) {
  if (!['ArrowDown','ArrowUp','Home','End','Enter'].includes(event.key)) return
  event.preventDefault()
  if (!filtered.value.length) return
  if (event.key === 'Enter') { emit('select', filtered.value[active.value].id); return }
  active.value = event.key === 'Home' ? 0 : event.key === 'End' ? filtered.value.length - 1 : (active.value + (event.key === 'ArrowDown' ? 1 : -1) + filtered.value.length) % filtered.value.length
  document.getElementById(`${id}-${active.value}`)?.scrollIntoView({block:'nearest'})
}
</script>

<template>
  <BaseDialog :show="show" :title="t('boardui.commandTitle')" width="normal" @close="emit('close')">
    <div data-boardui="command-palette" class="-mx-6 -my-4 flex flex-col">
      <div class="flex items-center gap-3 border-b border-separator-border px-5 py-4">
        <Icon name="search" size="md" class="text-foreground-icon-secondary" />
        <input autofocus ref="input" v-model="query" role="combobox" aria-autocomplete="list" aria-expanded="true" :aria-controls="`${id}-list`" :aria-activedescendant="filtered.length ? `${id}-${active}` : undefined" :aria-label="t('boardui.commandTitle')" :placeholder="t('boardui.commandPlaceholder')" class="min-w-0 flex-1 bg-transparent text-headline-regular text-text-primary outline-none placeholder:text-text-placeholder" @keydown="move" />
        <kbd class="rounded-md border border-border-button-default px-1.5 py-0.5 text-caption-2-medium text-text-tertiary">Esc</kbd>
      </div>
      <div :id="`${id}-list`" role="listbox" :aria-label="t('boardui.commandTitle')" class="max-h-80 overflow-y-auto p-2">
        <template v-for="(item,index) in filtered" :key="item.id">
          <p v-if="!index || item.section !== filtered[index - 1].section" class="px-3 pb-2 pt-3 text-caption-1-medium text-text-tertiary">{{ item.section }}</p>
          <button :id="`${id}-${index}`" type="button" role="option" :aria-selected="index === active" tabindex="-1" :class="cx('flex w-full items-center justify-between gap-3 rounded-xl px-3 py-2.5 text-body-regular outline-none', index === active ? 'bg-background-secondary-default text-text-primary' : 'text-text-secondary')" @pointermove="active = index" @click="emit('select', item.id)">
            <span>{{ item.label }}</span><Icon v-if="index === active" name="arrowRight" size="sm" class="text-foreground-icon-tertiary" />
          </button>
        </template>
        <p v-if="!filtered.length" class="p-8 text-center text-body-regular text-text-secondary">{{ t('empty.noData') }}</p>
      </div>
      <div class="flex items-center justify-between border-t border-separator-border bg-background-secondary-default px-5 py-3 text-caption-1-regular text-text-secondary"><span>↑ ↓ {{ t('boardui.commandNavigate') }}</span><span>↵ {{ t('boardui.commandSelect') }}</span></div>
    </div>
  </BaseDialog>
</template>
