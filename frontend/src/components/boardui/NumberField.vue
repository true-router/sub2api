<script setup lang="ts">
import { computed, useAttrs } from 'vue'
import Button from './Button.vue'
import Icon from '@/components/icons/Icon.vue'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const props = withDefaults(defineProps<{ modelValue: number | string | null | undefined; label?: string; hint?: string; min?: number; max?: number; step?: number | 'any'; disabled?: boolean; required?: boolean; name?: string }>(), { step: 1 })
const emit = defineEmits<{ 'update:modelValue': [value: number | ''] }>()
const number = computed(() => props.modelValue === '' || props.modelValue == null ? null : Number(props.modelValue))
function change(value: number) {
  if (!Number.isFinite(value)) { emit('update:modelValue', ''); return }
  emit('update:modelValue', Math.min(props.max ?? Infinity, Math.max(props.min ?? -Infinity, value)))
}
function move(direction: number) { change(Number(((number.value ?? props.min ?? 0) + (typeof props.step === 'number' ? props.step : 1) * direction).toFixed(10))) }
</script>

<template>
  <label class="flex flex-col gap-1.5" data-boardui="number-field">
    <span v-if="label" class="text-body-2-medium text-text-primary">{{ label }}</span>
    <span class="flex h-9 items-center overflow-hidden rounded-[10px] border border-border-button-default bg-background-primary-default shadow-sm focus-within:ring-2 focus-within:ring-border-focus-ring/20" :class="{ 'opacity-50': disabled }">
      <Button variant="ghost" size="small" icon-only data-step="decrease" :aria-label="label ? `${label} −` : '−'" :disabled="disabled || (number !== null && number <= (min ?? -Infinity))" class="h-full rounded-none border-r border-separator-border" @click="move(-1)"><span aria-hidden="true">−</span></Button>
      <input v-bind="attrs" :value="modelValue" :name="name" type="number" :min="min" :max="max" :step="step" :disabled="disabled" :required="required" :aria-label="label" class="w-full min-w-0 border-0 bg-transparent px-3 py-1 text-center text-body-regular tabular-nums text-text-primary outline-none" @input="change(($event.target as HTMLInputElement).valueAsNumber)" />
      <Button variant="ghost" size="small" icon-only data-step="increase" :aria-label="label ? `${label} +` : '+'" :disabled="disabled || (number !== null && number >= (max ?? Infinity))" class="h-full rounded-none border-l border-separator-border" @click="move(1)"><Icon name="plus" size="xs" /></Button>
    </span>
    <span v-if="hint" class="text-caption-1-regular text-text-secondary">{{ hint }}</span>
  </label>
</template>
