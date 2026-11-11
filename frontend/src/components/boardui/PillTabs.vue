<script setup lang="ts" generic="Key extends string">
import { ref, watch } from 'vue'
import { useResizeObserver } from '@vueuse/core'
import Icon from '@/components/icons/Icon.vue'
import { cx } from '@/utils/cx'

const props = defineProps<{
  modelValue: Key
  items: { key: Key; label: string; icon?: InstanceType<typeof Icon>['$props']['name'] }[]
  label: string
  idPrefix: string
  panelId: string
}>()
const emit = defineEmits<{ 'update:modelValue': [key: Key] }>()

const viewport = ref<HTMLElement>()
const track = ref<HTMLElement>()
const thumb = ref<{ left: number; top: number; width: number; height: number }>()
const buttons = () => Array.from(track.value?.querySelectorAll<HTMLButtonElement>('[role="tab"]') ?? [])

function syncSelection() {
  const rail = viewport.value
  const selected = buttons().find(button => button.getAttribute('aria-selected') === 'true')
  if (!rail || !selected) return
  thumb.value = {
    left: selected.offsetLeft,
    top: selected.offsetTop,
    width: selected.offsetWidth,
    height: selected.offsetHeight,
  }
  const bounds = rail.getBoundingClientRect()
  const tab = selected.getBoundingClientRect()
  if (tab.left < bounds.left) rail.scrollLeft += tab.left - bounds.left
  else if (tab.right > bounds.right) rail.scrollLeft += tab.right - bounds.right
}

watch(() => [props.modelValue, props.items], syncSelection, { flush: 'post' })
useResizeObserver([viewport, track], syncSelection)

function handleKeydown(event: KeyboardEvent, index: number) {
  const last = props.items.length - 1
  const rtl = track.value && getComputedStyle(track.value).direction === 'rtl'
  const step = event.key === 'ArrowRight' ? (rtl ? -1 : 1) : (rtl ? 1 : -1)
  const next = event.key === 'Home' ? 0
    : event.key === 'End' ? last
    : ['ArrowLeft', 'ArrowRight'].includes(event.key) ? (index + step + last + 1) % (last + 1)
    : undefined
  if (next === undefined) return
  event.preventDefault()
  emit('update:modelValue', props.items[next].key)
  buttons()[next]?.focus({ preventScroll: true })
}
</script>

<template>
  <div ref="viewport" data-boardui="pill-tabs" class="min-w-0 overflow-x-auto overscroll-x-contain [scrollbar-width:thin]">
    <div ref="track" role="tablist" :aria-label="label" class="relative flex w-max items-center gap-1 p-1">
      <span
        v-if="thumb"
        data-boardui="pill-tab-indicator"
        aria-hidden="true"
        class="pointer-events-none absolute left-0 top-0 rounded-[10px] bg-background-tertiary-default transition-[transform,width,height] duration-300 [transition-timing-function:cubic-bezier(0.34,1.2,0.64,1)] motion-reduce:transition-none"
        :style="{
          transform: `translate(${thumb.left}px, ${thumb.top}px)`,
          width: `${thumb.width}px`,
          height: `${thumb.height}px`,
        }"
      />
      <button
        v-for="(item, index) in items"
        :id="`${idPrefix}-${item.key}`"
        :key="item.key"
        type="button"
        role="tab"
        data-boardui="pill-tab"
        :aria-selected="modelValue === item.key"
        :aria-controls="panelId"
        :tabindex="modelValue === item.key ? 0 : -1"
        class="group relative z-10 flex shrink-0 cursor-pointer items-center gap-1 rounded-[10px] px-2 py-[5px] outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring"
        @click="emit('update:modelValue', item.key)"
        @keydown="handleKeydown($event, index)"
      >
        <span
          v-if="modelValue !== item.key"
          aria-hidden="true"
          class="pointer-events-none absolute inset-0 rounded-[10px] bg-background-primary-hover opacity-0 transition-opacity duration-200 ease-out group-hover:opacity-100 motion-reduce:transition-none"
        />
        <Icon
          v-if="item.icon"
          :name="item.icon"
          size="md"
          :class="cx('relative shrink-0', modelValue === item.key ? 'text-foreground-icon-primary' : 'text-foreground-icon-secondary')"
          aria-hidden="true"
        />
        <span :class="cx('relative whitespace-nowrap text-body-medium', modelValue === item.key ? 'text-text-primary' : 'text-text-secondary')">{{ item.label }}</span>
      </button>
    </div>
  </div>
</template>
