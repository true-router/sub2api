<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
defineProps<{ items: { label: string; to?: string }[]; label?: string }>()
</script>

<template>
  <nav :aria-label="label || 'Breadcrumb'" data-boardui="breadcrumb" class="min-w-0 overflow-x-auto">
    <ol class="flex items-center gap-2.5 whitespace-nowrap text-caption-1-medium">
      <li v-for="(item,index) in items" :key="index" class="flex min-w-0 items-center gap-2.5">
        <Icon v-if="index" name="chevronRight" size="xs" class="shrink-0 text-foreground-icon-tertiary" aria-hidden="true" />
        <RouterLink v-if="item.to && index < items.length - 1" :to="item.to" class="rounded-md px-1 py-0.5 text-text-tertiary outline-none hover:bg-background-primary-hover hover:text-text-secondary focus-visible:ring-2 focus-visible:ring-border-focus-ring">{{ item.label }}</RouterLink>
        <span v-else :aria-current="index === items.length - 1 ? 'page' : undefined" class="truncate text-text-secondary">{{ item.label }}</span>
      </li>
    </ol>
  </nav>
</template>
