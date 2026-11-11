<script setup lang="ts">
import { cx } from '@/utils/cx'

defineOptions({ inheritAttrs: false })
defineProps<{ disabled?: boolean }>()
</script>

<template>
  <button
    v-bind="$attrs"
    type="button"
    :disabled="disabled"
    data-boardui="social-button"
    :class="cx(
      'inline-flex h-10 w-full cursor-pointer select-none items-center justify-center gap-2 overflow-hidden whitespace-nowrap rounded-[10px] border border-border-button-default bg-background-primary-default px-3 text-body-medium text-text-primary shadow-sm outline-none transition-colors duration-150 hover:border-border-button-hover hover:bg-background-primary-hover active:border-border-button-active active:bg-background-primary-active focus-visible:ring-2 focus-visible:ring-border-focus-ring disabled:cursor-not-allowed disabled:opacity-60',
      $attrs.class as string
    )"
  ><slot /></button>
</template>
