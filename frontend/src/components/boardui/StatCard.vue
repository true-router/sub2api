<template>
  <article class="flex min-w-0 flex-col rounded-2xl bg-background-secondary-default p-2" data-boardui="stat-card">
    <div class="flex items-start justify-between gap-3 p-2 lg:p-1">
      <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-gradient-to-b from-background-accent-default to-background-accent-hover text-text-white shadow-sm lg:h-9 lg:w-9"><Icon :name="icon" size="md" /></span>
    </div>
    <div class="flex flex-col gap-1 px-2 pb-4 pt-2.5 lg:pb-3 lg:pt-1">
      <p class="truncate text-body-medium text-text-secondary">{{ label }}</p>
      <p class="min-w-0 truncate text-display-4-medium tabular-nums text-text-primary" :title="String(value)">{{ value }}</p>
    </div>
    <div class="mt-auto flex min-h-10 flex-wrap items-center gap-x-2 gap-y-1 rounded-[10px] border border-border-button-default bg-background-primary-default px-3 py-2 text-caption-1-regular text-text-secondary shadow-sm lg:min-h-8 lg:px-2.5 lg:py-1.5"><slot /></div>
  </article>
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
type IconName = InstanceType<typeof Icon>['$props']['name']

defineProps<{ label: string; value: string | number; icon: IconName }>()
</script>
