<script setup lang="ts">
import { computed, ref, useAttrs, useId } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { cx } from '@/utils/cx'

defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const props = withDefaults(defineProps<{ modelValue: string | number | null | undefined; label?: string; hint?: string; error?: string; type?: string; placeholder?: string; required?: boolean; disabled?: boolean; readonly?: boolean; id?: string; revealable?: boolean }>(), { type: 'text' })
const emit = defineEmits<{ 'update:modelValue': [value: string]; change: [value: string]; blur: [event: FocusEvent]; focus: [event: FocusEvent]; enter: [event: KeyboardEvent] }>()
const generatedId = useId()
const fieldId = computed(() => props.id || generatedId)
const visible = ref(false)
const input = ref<HTMLInputElement>()
defineExpose({ focus: () => input.value?.focus(), select: () => input.value?.select() })
</script>

<template>
  <div :class="cx('flex w-full flex-col gap-1.5', attrs.class as string)" data-boardui="input">
    <label v-if="label" :for="fieldId" class="text-body-2-medium text-text-primary">{{ label }}<span v-if="required" class="ms-1 text-text-error-primary" aria-hidden="true">*</span></label>
    <div :class="cx('flex min-h-9 items-center gap-2 rounded-[10px] border bg-background-primary-default px-3 shadow-sm transition-colors duration-150 focus-within:ring-2', error ? 'border-border-error-default focus-within:ring-border-error-default/20' : 'border-border-button-default hover:border-border-button-hover focus-within:border-border-focus-ring focus-within:ring-border-focus-ring/20', disabled && 'cursor-not-allowed bg-background-secondary-default opacity-50')">
      <span v-if="$slots.prefix" class="shrink-0 text-foreground-icon-secondary"><slot name="prefix" /></span>
      <input v-bind="{...attrs, class:undefined}" :id="fieldId" ref="input" :value="modelValue" :type="revealable && visible ? 'text' : type" :placeholder="placeholder" :disabled="disabled" :readonly="readonly" :required="required" :aria-invalid="error ? true : undefined" :aria-describedby="hint || error ? `${fieldId}-help` : undefined" class="min-w-0 flex-1 border-0 bg-transparent py-2 text-body-regular text-text-primary outline-none placeholder:text-text-placeholder disabled:cursor-not-allowed" @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)" @change="emit('change', ($event.target as HTMLInputElement).value)" @blur="emit('blur', $event)" @focus="emit('focus', $event)" @keydown.enter="emit('enter', $event)" />
      <button v-if="revealable && type === 'password'" type="button" :aria-label="label" :aria-pressed="visible" class="rounded-md text-foreground-icon-secondary outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring" @click="visible = !visible"><Icon :name="visible ? 'eyeOff' : 'eye'" size="sm" /></button>
      <span v-if="$slots.suffix" class="shrink-0 text-foreground-icon-secondary"><slot name="suffix" /></span>
    </div>
    <p v-if="hint || error" :id="`${fieldId}-help`" :class="cx('text-caption-1-regular', error ? 'text-text-error-primary' : 'text-text-secondary')">{{ error || hint }}</p>
  </div>
</template>
