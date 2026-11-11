<script setup lang="ts">
import { ref, useAttrs, watchEffect } from 'vue'
import { cx } from '@/utils/cx'
defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const props = defineProps<{ modelValue: boolean; indeterminate?: boolean; disabled?: boolean; label?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; change: [event: Event] }>()
const input = ref<HTMLInputElement>()
watchEffect(() => { if (input.value) input.value.indeterminate = !!props.indeterminate })
function change(event: Event) { emit('update:modelValue', (event.target as HTMLInputElement).checked); emit('change', event) }
</script>

<template>
  <label :class="cx('relative inline-flex cursor-pointer items-center gap-2 align-middle', disabled && 'cursor-not-allowed opacity-50', attrs.class as string)" data-boardui="checkbox" @click.stop>
    <input v-bind="{...attrs, class:undefined}" ref="input" type="checkbox" :checked="modelValue" :disabled="disabled" :aria-label="(attrs['aria-label'] as string) || label" class="peer sr-only" @change="change" />
    <span aria-hidden="true" :class="cx('flex h-4 w-4 shrink-0 items-center justify-center rounded border shadow-sm transition-colors duration-150 peer-focus-visible:ring-2 peer-focus-visible:ring-border-focus-ring peer-focus-visible:ring-offset-2', modelValue || indeterminate ? 'border-background-accent-default bg-gradient-to-b from-background-accent-default to-background-accent-hover text-text-white' : 'border-border-button-hover bg-background-primary-default')">
      <svg v-if="modelValue || indeterminate" viewBox="0 0 16 16" fill="none" class="h-3 w-3" stroke="currentColor" stroke-width="2"><path v-if="indeterminate" d="M3 8h10" /><path v-else class="boardui-checkbox-tick" d="m3 8 3 3 7-7" /></svg>
    </span>
    <span v-if="label" class="text-body-regular text-text-primary">{{ label }}</span>
  </label>
</template>
