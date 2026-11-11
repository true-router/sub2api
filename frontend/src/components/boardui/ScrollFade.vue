<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ scrollTop: number }>()
const opacity = computed(() => Math.min(1, Math.max(0, props.scrollTop) / 24))
</script>

<template>
  <div data-boardui="scroll-fade" aria-hidden="true" class="pointer-events-none absolute inset-x-0 top-0 z-10 h-6">
    <div class="absolute inset-0 backdrop-blur-[1px] [mask-image:linear-gradient(to_bottom,black,transparent)]" :style="{ opacity }" />
    <div class="absolute inset-x-0 top-0 h-4 backdrop-blur-[4px] [mask-image:linear-gradient(to_bottom,black,transparent)]" :style="{ opacity }" />
    <div class="absolute inset-0 bg-gradient-to-b from-background-full to-transparent" :style="{ opacity }" />
  </div>
</template>
