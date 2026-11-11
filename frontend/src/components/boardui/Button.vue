<script setup lang="ts">
import { computed, ref, useAttrs } from 'vue'
import { cx } from '@/utils/cx'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const button = ref<HTMLButtonElement>()
defineExpose({ focus: () => button.value?.focus(), click: () => button.value?.click() })
const props = withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'success' | 'warning'
  size?: 'medium' | 'small' | 'xs'
  iconOnly?: boolean
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), { variant: 'primary', size: 'medium', type: 'button' })
const variants = {
  primary: 'border-transparent bg-gradient-to-b from-background-accent-default to-background-accent-hover text-text-white shadow-sm hover:brightness-105 active:brightness-95',
  secondary: 'border-border-button-default bg-background-primary-default text-text-primary shadow-sm hover:border-border-button-hover hover:bg-background-primary-hover active:bg-background-primary-active',
  ghost: 'border-transparent bg-transparent text-text-secondary hover:bg-background-secondary-default active:bg-background-secondary-hover',
  danger: 'border-transparent bg-background-error-default text-text-white shadow-sm hover:brightness-105 active:brightness-95',
  success: 'border-transparent bg-background-success-default text-text-white hover:brightness-105 active:brightness-95',
  warning: 'border-transparent bg-background-warning-default text-text-white hover:brightness-105 active:brightness-95'
}
const sizes = { medium: 'h-9 rounded-[10px] px-3 py-2 text-body-medium', small: 'h-8 rounded-lg px-2.5 py-1.5 text-body-medium', xs: 'h-6 rounded px-2 text-caption-1-semibold' }
const classes = computed(() => cx('inline-flex shrink-0 items-center justify-center gap-1.5 whitespace-nowrap border outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-border-focus-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', variants[props.variant], sizes[props.size], props.iconOnly && 'aspect-square px-0', attrs.class as string))
</script>

<template>
  <button v-bind="attrs" ref="button" :class="classes" :type="type" :disabled="disabled || loading" :aria-busy="loading || undefined" data-boardui="button">
    <span v-if="loading" aria-hidden="true" class="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent motion-reduce:animate-none" />
    <slot name="leading" />
    <slot />
    <slot name="trailing" />
  </button>
</template>
