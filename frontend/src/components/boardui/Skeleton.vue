<script setup lang="ts">
import { computed } from 'vue'
import { cx } from '@/utils/cx'
const props = withDefaults(defineProps<{ variant?: 'rect' | 'circle' | 'text'; width?: string | number; height?: string | number; class?: string }>(), { variant: 'rect', width: '100%' })
const style = computed(() => ({ width: typeof props.width === 'number' ? `${props.width}px` : props.width, height: props.height ? typeof props.height === 'number' ? `${props.height}px` : props.height : props.variant === 'text' ? '1em' : '1rem' }))
</script>

<template>
  <div aria-hidden="true" data-boardui="skeleton" :style="style" :class="cx('boardui-skeleton relative overflow-hidden bg-background-tertiary-default', variant === 'circle' ? 'rounded-full' : 'rounded-lg', props.class)" />
</template>

<style scoped>
.boardui-skeleton::after { content: ''; position: absolute; inset: 0; background: linear-gradient(90deg, transparent, rgb(var(--boardui-foreground-full) / .45), transparent); transform: translateX(-100%); animation: boardui-skeleton 1.8s ease-in-out infinite; }
@keyframes boardui-skeleton { 60%, 100% { transform: translateX(100%); } }
@media (prefers-reduced-motion: reduce) { .boardui-skeleton::after { animation: none; display: none; } }
</style>
