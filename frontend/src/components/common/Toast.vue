<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores/app'
import BoardToast from '@/components/boardui/Toast.vue'
const appStore = useAppStore()
const toasts = computed(() => appStore.toasts)
</script>

<template>
  <Teleport to="body">
    <div class="pointer-events-none fixed right-4 top-4 z-[9999] flex w-[calc(100vw-2rem)] max-w-sm flex-col gap-3" aria-live="polite">
      <TransitionGroup enter-active-class="transition duration-150 ease-out motion-reduce:transition-none" enter-from-class="scale-95 opacity-0 blur-sm" leave-active-class="transition duration-100 ease-in motion-reduce:transition-none" leave-to-class="scale-95 opacity-0">
        <BoardToast v-for="toast in toasts" :key="toast.id" :type="toast.type" :title="toast.title" :message="toast.message" :duration="toast.duration" @dismiss="appStore.hideToast(toast.id)" />
      </TransitionGroup>
    </div>
  </Teleport>
</template>
