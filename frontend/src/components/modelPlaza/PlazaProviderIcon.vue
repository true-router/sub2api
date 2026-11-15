<script setup lang="ts">
import { providerIconPaths } from '@/assets/plaza-provider-icons'
defineProps<{ icon: string }>()
</script>

<template>
  <svg class="h-5 w-5 shrink-0" viewBox="0 0 24 24" fill="currentColor" fill-rule="evenodd" aria-hidden="true">
    <template v-if="providerIconPaths[icon]">
      <path v-for="(path, index) in providerIconPaths[icon]" :key="index" :d="path" />
    </template>
    <path v-else d="M3 3h7v7H3V3zm11 0h7v7h-7V3zM3 14h7v7H3v-7zm11 0h7v7h-7v-7z" />
  </svg>
</template>
