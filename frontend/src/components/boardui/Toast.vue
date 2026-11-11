<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardButton from './Button.vue'
import Icon from '@/components/icons/Icon.vue'
const props = defineProps<{ type: 'success' | 'error' | 'warning' | 'info'; title?: string; message: string; duration?: number }>()
const emit = defineEmits<{ dismiss: [] }>()
const { t } = useI18n()
const icons = { success: 'checkCircle', error: 'xCircle', warning: 'exclamationTriangle', info: 'infoCircle' } as const
const colors = { success: 'bg-background-success-subtle text-text-success', error: 'bg-background-error-subtle text-text-error-primary', warning: 'bg-background-warning-subtle text-text-warning', info: 'bg-background-info-subtle text-text-info' }
const color = computed(() => colors[props.type])
</script>

<template>
  <div data-boardui="toast" :role="type === 'error' ? 'alert' : 'status'" class="pointer-events-auto w-full overflow-hidden rounded-2xl border border-border-button-default bg-background-primary-default shadow-lg">
    <div class="flex items-start gap-3 p-4">
      <span :class="['flex h-9 w-9 shrink-0 items-center justify-center rounded-xl',color]"><Icon :name="icons[type]" size="md" /></span>
      <div class="min-w-0 flex-1"><p v-if="title" class="mb-1 text-body-medium text-text-primary">{{ title }}</p><p class="text-body-regular text-text-secondary">{{ message }}</p></div>
      <BoardButton variant="ghost" size="xs" icon-only :aria-label="t('common.close')" @click="emit('dismiss')"><Icon name="x" size="sm" /></BoardButton>
    </div>
    <div v-if="duration" class="h-0.5 bg-background-secondary-default"><div class="boardui-toast-progress h-full bg-background-accent-default" :style="{animationDuration:`${duration}ms`}" /></div>
  </div>
</template>

<style scoped>
.boardui-toast-progress { transform-origin: left; animation-name: boardui-toast-progress; animation-timing-function: linear; animation-fill-mode: forwards; }
@keyframes boardui-toast-progress { to { transform: scaleX(0); } }
@media (prefers-reduced-motion: reduce) { .boardui-toast-progress { animation: none; } }
</style>
