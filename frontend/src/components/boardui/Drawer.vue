<script setup lang="ts">
import BaseDialog from '@/components/common/BaseDialog.vue'
withDefaults(defineProps<{ show: boolean; title: string; description?: string; width?: 'normal' | 'wide' | 'extra-wide' | 'full' }>(), { width: 'normal' })
const emit = defineEmits<{ close: [] }>()
</script>

<template>
  <BaseDialog :show="show" :title="title" :description="description" presentation="drawer" :width="width" close-on-click-outside @close="emit('close')">
    <slot />
    <template v-if="$slots.footer" #footer><slot name="footer" /></template>
  </BaseDialog>
</template>
