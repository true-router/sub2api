<script setup lang="ts">
import { cx } from '@/utils/cx'

defineProps<{ title: string; description?: string; class?: string }>()
</script>

<template>
  <section
    data-boardui="auth-card"
    :class="cx('flex w-full max-w-[400px] flex-col rounded-3xl border border-border-button-default bg-background-primary-default p-6 shadow-sm sm:p-8', $props.class)"
  >
    <div v-if="$slots.logo" class="mb-5 flex justify-start"><slot name="logo" /></div>
    <header class="flex flex-col gap-1.5">
      <h1 class="text-title-2-medium text-text-primary">{{ title }}</h1>
      <p v-if="description" class="text-body-regular text-text-secondary">{{ description }}</p>
    </header>
    <div class="mt-6"><slot /></div>
    <div v-if="$slots.footer" class="mt-6 text-center text-body-regular text-text-secondary"><slot name="footer" /></div>
  </section>
</template>
