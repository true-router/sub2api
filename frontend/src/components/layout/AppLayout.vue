<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950" :class="{ 'boardui-admin-layout': isWorkspace }">
    <!-- Background Decoration -->
    <div v-if="!isWorkspace" class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="app-content relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader v-if="!isWorkspace" />

      <!-- Main Content -->
      <main id="main-content" class="app-main p-4 md:p-6 lg:p-8" tabindex="-1">
        <AdminContentHeading v-if="isWorkspace" />
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { usesBoardUIWorkspace } from '@/router/workspace'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import AdminContentHeading from './AdminContentHeading.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const route = useRoute()
const isWorkspace = computed(() => usesBoardUIWorkspace(route.meta))
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>
