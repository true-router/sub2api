<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
defineProps<{ title: string; description?: string }>()
</script>
<template>
  <AppLayout hide-heading>
    <div class="mx-auto w-full max-w-4xl space-y-6" data-testid="console-page">
      <header class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0 space-y-1">
          <h1 class="text-title-1-semibold text-text-primary">{{ title }}</h1>
          <p v-if="description" class="text-body-regular text-text-secondary">{{ description }}</p>
        </div>
        <slot name="actions" />
      </header>
      <slot />
    </div>
  </AppLayout>
</template>
