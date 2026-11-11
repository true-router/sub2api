<template>
  <BoardBadge :tone="tone" dot>{{ label }}</BoardBadge>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import BoardBadge from '@/components/boardui/Badge.vue'

const props = defineProps<{
  status: string
  label: string
}>()

const tone = computed(() => {
  if (['active', 'success'].includes(props.status)) return 'success'
  if (['disabled', 'inactive', 'warning'].includes(props.status)) return 'warning'
  if (['error', 'danger'].includes(props.status)) return 'danger'
  return 'neutral'
})
</script>
