<script setup lang="ts">
import { ref, useAttrs, useId } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { cx } from '@/utils/cx'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
withDefaults(defineProps<{ modelValue: string; placeholder?: string; label?: string; loading?: boolean; disabled?: boolean; shape?: 'pill' | 'rounded'; clearLabel?: string }>(), { shape: 'pill', clearLabel: 'Clear search' })
const emit = defineEmits<{ 'update:modelValue': [value: string]; search: [value: string]; submit: [value: string]; clear: [] }>()
const input = ref<HTMLInputElement>()
const fieldId = useId()
function update(value: string) { emit('update:modelValue', value); emit('search', value) }
function clear() { update(''); emit('clear'); input.value?.focus() }
</script>

<template>
  <div :class="cx('flex w-full flex-col gap-1', attrs.class as string)" data-boardui="search-field">
    <label v-if="label" :for="(attrs.id as string) || fieldId" class="text-body-2-medium text-text-primary">{{ label }}</label>
    <div :class="cx('relative flex h-9 items-center gap-2 border border-border-button-default bg-background-primary-default px-3 text-text-primary shadow-sm focus-within:border-border-focus-ring focus-within:ring-2 focus-within:ring-border-focus-ring/20', shape === 'pill' ? 'rounded-full' : 'rounded-[10px]', disabled && 'opacity-50')">
      <span v-if="loading" class="h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-border-button-hover border-t-border-focus-ring motion-reduce:animate-none" aria-hidden="true" />
      <Icon v-else name="search" size="sm" class="shrink-0 text-foreground-icon-secondary" />
      <input v-bind="{...attrs, class: undefined}" ref="input" :id="(attrs.id as string) || fieldId" type="search" :value="modelValue" :placeholder="placeholder" :disabled="disabled" :aria-label="(attrs['aria-label'] as string) || label || placeholder" class="min-w-0 flex-1 border-0 bg-transparent p-0 text-body-regular text-text-primary outline-none placeholder:text-text-placeholder focus:outline-none [&::-webkit-search-cancel-button]:hidden" @input="update(($event.target as HTMLInputElement).value)" @keydown.esc.stop.prevent="clear" @keydown.enter.prevent="emit('submit', modelValue)" />
      <button v-if="modelValue" type="button" tabindex="-1" :disabled="disabled" :aria-label="clearLabel" class="flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-foreground-icon-secondary hover:bg-background-secondary-default" @click="clear"><Icon name="x" size="xs" /></button>
    </div>
  </div>
</template>
