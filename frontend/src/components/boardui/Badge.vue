<script setup lang="ts">
import { cx } from '@/utils/cx'
withDefaults(defineProps<{ tone?: 'neutral' | 'accent' | 'success' | 'warning' | 'danger'; dot?: boolean; removable?: boolean; removeLabel?: string }>(), { tone: 'neutral', removeLabel: 'Remove' })
const emit = defineEmits<{ remove: [] }>()
const tones = { neutral: 'bg-background-secondary-default text-text-secondary', accent: 'bg-background-accent-subtle text-text-accent', success: 'bg-background-success-subtle text-text-success', warning: 'bg-background-warning-subtle text-text-warning', danger: 'bg-background-error-subtle text-text-error-primary' }
</script>

<template>
  <span data-boardui="badge" :class="cx('inline-flex max-w-full items-center gap-1.5 rounded-md border border-border-button-default px-2 py-0.5 text-caption-1-medium', tones[tone])"><span v-if="dot" aria-hidden="true" class="h-1.5 w-1.5 shrink-0 rounded-full bg-current" /><span class="truncate"><slot /></span><button v-if="removable" type="button" :aria-label="removeLabel" class="rounded-sm px-0.5 outline-none hover:bg-background-primary-active focus-visible:ring-2 focus-visible:ring-border-focus-ring" @click="emit('remove')">×</button></span>
</template>
