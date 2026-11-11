<script setup lang="ts">
import { cx } from '@/utils/cx'
const props = withDefaults(defineProps<{ modelValue: boolean; disabled?: boolean; label?: string; size?: 'sm' | 'md' | 'lg'; shape?: 'pill' | 'rectangle' }>(), { size: 'md', shape: 'pill' })
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const tracks = { sm: 'h-4 w-7', md: 'h-6 w-[42px]', lg: 'h-8 w-14' }
const thumbs = { sm: 'left-0.5 top-0.5 h-3 w-3', md: 'left-[3px] top-[3px] h-[18px] w-[18px]', lg: 'left-1 top-1 h-6 w-6' }
const travel = { sm: 'translate-x-3', md: 'translate-x-[18px]', lg: 'translate-x-6' }
</script>

<template>
  <button type="button" role="switch" data-boardui="switch" :aria-label="label" :aria-checked="modelValue" :disabled="disabled" :class="cx('inline-flex items-center gap-2 outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50', label && 'text-body-regular text-text-primary')" @click="emit('update:modelValue', !props.modelValue)">
    <span aria-hidden="true" :class="cx('relative shrink-0 transition-colors duration-200 focus-within:ring-2', tracks[size], shape === 'pill' ? 'rounded-full' : 'rounded-md', modelValue ? 'bg-gradient-to-b from-background-accent-default to-background-accent-hover shadow-sm' : 'bg-background-tertiary-default')">
      <span :class="cx('absolute flex items-center justify-center bg-gradient-to-b from-foreground-full to-background-secondary-default shadow-sm transition-transform duration-200 motion-reduce:transition-none', thumbs[size], shape === 'pill' ? 'rounded-full' : 'rounded-sm', modelValue && travel[size])">
        <span :class="cx('h-1.5 w-1.5 border shadow-inner', shape === 'pill' ? 'rounded-full' : 'rounded-sm', modelValue ? 'border-background-accent-hover bg-background-accent-default' : 'border-border-button-default bg-background-secondary-default')" />
      </span>
    </span>
    <span v-if="label">{{ label }}</span>
  </button>
</template>
