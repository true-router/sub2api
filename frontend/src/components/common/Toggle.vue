<script setup lang="ts">
import BoardSwitch from '@/components/boardui/Switch.vue'
defineProps<{ modelValue: boolean; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<template>
  <BoardSwitch :model-value="modelValue" :disabled="disabled" @update:model-value="emit('update:modelValue', $event)" />
</template>
