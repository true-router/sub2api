<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import { cx } from '@/utils/cx'
const props = defineProps<{ modelValue: string; items: { value: string; label: string; disabled?: boolean }[]; label: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const id = useId()
const root = ref<HTMLElement>()
const indicator = ref<Record<string, string>>({ opacity: '0' })
async function measure() {
  await nextTick()
  const button = root.value?.querySelector<HTMLElement>('[aria-checked="true"]')
  if (button) indicator.value = { left: `${button.offsetLeft}px`, top: `${button.offsetTop}px`, width: `${button.offsetWidth}px`, height: `${button.offsetHeight}px`, opacity: '1' }
}
watch(() => [props.modelValue, props.items], measure, { immediate: true })
useResizeObserver(root, measure)
const enabled = computed(() => props.items.filter(item => !item.disabled))
function keydown(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const items = enabled.value
  if (!items.length) return
  const current = items.findIndex(item => item.value === props.modelValue)
  const index = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (current + (event.key === 'ArrowRight' ? 1 : -1) + items.length) % items.length
  emit('update:modelValue', items[index].value)
  document.getElementById(`${id}-${props.items.findIndex(item => item.value === items[index].value)}`)?.focus()
}
</script>

<template>
  <div ref="root" role="radiogroup" :aria-label="label" data-boardui="segmented-control" class="relative inline-flex max-w-full items-center gap-1 rounded-xl border border-border-button-default bg-background-secondary-default p-1" @keydown="keydown">
    <span aria-hidden="true" :style="indicator" class="pointer-events-none absolute rounded-lg bg-background-primary-default shadow-sm transition-all duration-200 motion-reduce:transition-none" />
    <button v-for="(item, index) in items" :id="`${id}-${index}`" :key="item.value" type="button" role="radio" :disabled="item.disabled" :aria-checked="modelValue === item.value" :tabindex="modelValue === item.value ? 0 : -1" :class="cx('relative z-10 rounded-lg px-3 py-1.5 text-body-2-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-border-focus-ring disabled:opacity-50', modelValue === item.value ? 'text-text-primary' : 'text-text-secondary hover:text-text-primary')" @click="emit('update:modelValue', item.value)">{{ item.label }}</button>
  </div>
</template>
