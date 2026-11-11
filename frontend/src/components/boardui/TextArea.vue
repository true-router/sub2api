<script setup lang="ts">
import { computed, nextTick, ref, useAttrs, useId, watch } from 'vue'
import { cx } from '@/utils/cx'
defineOptions({ inheritAttrs: false })
const attrs = useAttrs()
const props = withDefaults(defineProps<{ modelValue: string | null | undefined; label?: string; placeholder?: string; hint?: string; error?: string; disabled?: boolean; readonly?: boolean; required?: boolean; id?: string; rows?: number | string; maxLength?: number; autoResize?: boolean }>(), { rows: 3, autoResize: true })
const emit = defineEmits<{ 'update:modelValue': [value: string]; change: [value: string]; blur: [event: FocusEvent]; focus: [event: FocusEvent] }>()
const generatedId = useId()
const fieldId = computed(() => props.id || generatedId)
const textarea = ref<HTMLTextAreaElement>()
watch(() => props.modelValue, async () => {
  await nextTick()
  if (props.autoResize && textarea.value) { textarea.value.style.height = 'auto'; textarea.value.style.height = `${textarea.value.scrollHeight}px` }
}, { immediate: true })
defineExpose({ focus: () => textarea.value?.focus(), select: () => textarea.value?.select() })
</script>

<template>
  <div :class="cx('flex w-full flex-col gap-1.5', attrs.class as string)" data-boardui="textarea">
    <label v-if="label" :for="fieldId" class="text-body-2-medium text-text-primary">{{ label }}<span v-if="required" aria-hidden="true" class="ms-1 text-text-error-primary">*</span></label>
    <textarea v-bind="{...attrs,class:undefined}" :id="fieldId" ref="textarea" :value="modelValue" :rows="rows" :maxlength="maxLength" :disabled="disabled" :readonly="readonly" :required="required" :placeholder="placeholder" :aria-invalid="error ? true : undefined" :aria-describedby="hint || error ? `${fieldId}-help` : undefined" :class="cx('min-h-20 w-full rounded-[10px] border bg-background-primary-default px-3 py-2 text-body-regular text-text-primary shadow-sm outline-none transition-colors duration-150 placeholder:text-text-placeholder focus:ring-2 disabled:cursor-not-allowed disabled:opacity-50', error ? 'border-border-error-default focus:ring-border-error-default/20' : 'border-border-button-default focus:border-border-focus-ring focus:ring-border-focus-ring/20', autoResize ? 'resize-none overflow-hidden' : 'resize-y')" @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)" @change="emit('change', ($event.target as HTMLTextAreaElement).value)" @blur="emit('blur',$event)" @focus="emit('focus',$event)" />
    <div v-if="hint || error || maxLength" class="flex justify-between gap-3 text-caption-1-regular text-text-secondary"><p :id="`${fieldId}-help`" :class="{ 'text-text-error-primary': error }">{{ error || hint }}</p><span v-if="maxLength" class="ms-auto tabular-nums">{{ (modelValue || '').length }}/{{ maxLength }}</span></div>
  </div>
</template>
