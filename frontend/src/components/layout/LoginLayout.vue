<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import BoardButton from '@/components/boardui/Button.vue'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()
const { t } = useI18n()
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }) || '/logo.svg')
const currentYear = new Date().getFullYear()
const isDark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
onMounted(() => { void appStore.fetchPublicSettings() })
</script>

<template>
  <div data-boardui="login-page" class="flex min-h-dvh flex-col bg-background-full text-text-primary">
    <header class="flex shrink-0 items-center justify-between gap-4 px-5 py-4 sm:px-8 sm:py-6">
      <div class="flex min-w-0 items-center gap-2.5">
        <img :src="siteLogo" alt="" class="h-8 w-8 shrink-0 rounded-lg object-contain" />
        <span class="truncate text-headline-semibold">{{ siteName }}</span>
      </div>
      <BoardButton
        data-testid="login-theme-toggle"
        variant="ghost"
        size="small"
        icon-only
        :aria-label="t(isDark ? 'nav.lightMode' : 'nav.darkMode')"
        :title="t(isDark ? 'nav.lightMode' : 'nav.darkMode')"
        @click="toggleTheme"
      ><Icon :name="isDark ? 'sun' : 'moon'" size="sm" /></BoardButton>
    </header>

    <main id="main-content" class="flex flex-1 items-center justify-center px-4 py-8 sm:py-12">
      <slot />
    </main>

    <footer class="px-5 py-5 text-center text-caption-1-regular text-text-tertiary">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>
</template>
