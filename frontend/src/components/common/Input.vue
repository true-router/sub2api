<script setup lang="ts">
import { ref } from 'vue'
import BoardInput from '@/components/boardui/Input.vue'
const props = withDefaults(defineProps<{
  modelValue: string | number | null | undefined
  type?: string
  label?: string
  placeholder?: string
  disabled?: boolean
  required?: boolean
  readonly?: boolean
  error?: string
  hint?: string
  id?: string
  autocomplete?: string
}>(), { type: 'text' })
const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
  blur: [event: FocusEvent]
  focus: [event: FocusEvent]
  enter: [event: KeyboardEvent]
}>()
const field = ref<InstanceType<typeof BoardInput>>()
defineExpose({ focus: () => field.value?.focus(), select: () => field.value?.select() })
</script>

<template>
  <BoardInput v-bind="props" ref="field" @update:model-value="emit('update:modelValue', $event)" @change="emit('change', $event)" @blur="emit('blur', $event)" @focus="emit('focus', $event)" @enter="emit('enter', $event)">
    <template v-if="$slots.prefix" #prefix><slot name="prefix" /></template>
    <template v-if="$slots.suffix" #suffix><slot name="suffix" /></template>
  </BoardInput>
</template>
