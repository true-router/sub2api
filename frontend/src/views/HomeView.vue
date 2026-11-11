<template>
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe v-if="isHomeContentUrl" :src="homeContent.trim()" :title="siteName" class="h-screen w-full border-0" allowfullscreen />
    <!-- Custom HTML is an administrator-owned setting. -->
    <div v-else v-html="homeContent" />
  </div>

  <div v-else :data-testid="compactHomeEnabled ? 'compact-home' : 'full-home'" class="home-page flex min-h-screen flex-col bg-background-full text-text-primary">
    <header class="border-b border-separator-border bg-background-primary-default">
      <nav :aria-label="t('home.redesign.navigation')" class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-5 py-4 sm:px-8">
        <a href="#" :aria-label="siteName" class="flex min-w-0 items-center gap-2.5 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring">
          <img :src="siteLogo || '/logo.svg'" alt="" class="h-8 w-8 shrink-0 rounded-lg object-contain" />
          <span class="max-w-40 truncate text-headline-semibold sm:max-w-64">{{ siteName }}</span>
        </a>
        <div class="flex items-center gap-1 sm:gap-3">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-link hidden text-body-medium md:inline-flex">{{ t('home.docs') }}</a>
          <LocaleSwitcher />
          <Button variant="ghost" icon-only :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')" @click="toggleTheme">
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" aria-hidden="true" />
          </Button>
          <router-link :to="destination" class="home-action home-action-secondary">
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
            <Icon name="arrowRight" size="sm" aria-hidden="true" class="hidden sm:block" />
          </router-link>
        </div>
      </nav>
    </header>

    <main class="mx-auto w-full max-w-6xl flex-1 px-5 sm:px-8">
      <section class="grid items-center gap-12 pb-12 pt-12 md:grid-cols-2 md:gap-8 md:pb-16 md:pt-20" aria-labelledby="home-title">
        <div class="flex min-w-0 flex-col items-start gap-6">
          <Badge tone="accent"><span class="inline-flex items-center gap-1.5"><Icon name="link" size="xs" aria-hidden="true" />{{ t('home.redesign.eyebrow') }}</span></Badge>
          <h1 id="home-title" class="text-display-4-semibold [overflow-wrap:anywhere]">
            {{ t('home.redesign.title') }}<br /><span class="text-text-accent">{{ t('home.redesign.titleAccent') }}</span>
          </h1>
          <p class="text-headline-medium [overflow-wrap:anywhere]">{{ siteSubtitle }}</p>
          <p class="max-w-md text-headline-regular text-text-secondary">{{ t('home.redesign.description') }}</p>
          <div class="flex flex-wrap gap-3">
            <router-link :to="destination" class="home-action home-action-primary" data-testid="home-primary-action">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="md" aria-hidden="true" />
            </router-link>
            <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="home-action home-action-secondary">{{ t('nav.modelPlaza') }}<Icon name="grid" size="md" aria-hidden="true" /></router-link>
          </div>
          <div class="flex flex-wrap gap-x-5 gap-y-2 text-caption-1-medium text-text-secondary">
            <span v-for="item in ['compatible', 'usage', 'control']" :key="item" class="inline-flex items-center gap-1.5"><Icon name="check" size="sm" class="text-text-accent" aria-hidden="true" />{{ t(`home.redesign.assurances.${item}`) }}</span>
          </div>
        </div>

        <div class="route-map relative flex min-w-0 flex-col gap-6 overflow-hidden rounded-3xl border border-border-button-default bg-background-secondary-default p-5 sm:p-8" :aria-label="t('home.redesign.routingLabel')">
          <div class="relative flex items-center justify-between gap-3">
            <span class="text-caption-1-semibold text-text-secondary">{{ t('home.redesign.routingLabel') }}</span>
            <Badge>{{ t('home.redesign.illustration') }}</Badge>
          </div>
          <div class="relative mx-auto flex w-full max-w-72 items-center gap-3 rounded-xl border border-border-button-default bg-background-primary-default p-4 shadow-sm">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-background-secondary-default"><Icon name="terminal" size="lg" aria-hidden="true" /></div>
            <div class="min-w-0"><p class="text-body-medium">{{ t('home.redesign.yourApp') }}</p><p class="text-caption-1-medium text-text-secondary">{{ t('home.redesign.yourAppHint') }}</p></div>
          </div>
          <div class="relative flex justify-center" aria-hidden="true"><span class="h-6 w-px bg-border-button-hover" /></div>
          <div class="relative mx-auto flex w-full max-w-72 items-center gap-3 rounded-2xl border border-border-focus-ring bg-background-primary-default p-4 shadow-sm">
            <img :src="siteLogo || '/logo.svg'" alt="" class="h-10 w-10 shrink-0 rounded-xl object-contain" />
            <div class="min-w-0"><p class="truncate text-headline-semibold">{{ siteName }}</p><p class="text-caption-1-medium text-text-accent">{{ t('home.redesign.oneEndpoint') }}</p></div>
            <Icon name="check" size="md" class="ms-auto shrink-0 text-text-accent" aria-hidden="true" />
          </div>
          <div class="relative flex items-center justify-center" aria-hidden="true"><span class="h-6 w-px bg-border-button-hover" /></div>
          <div class="relative grid grid-cols-3 gap-2 sm:gap-3">
            <div v-for="provider in providers" :key="provider.platform" class="flex flex-col items-center gap-3 rounded-xl border border-border-button-default bg-background-primary-default px-2 py-4">
              <PlatformIcon :platform="provider.platform" size="lg" class="text-foreground-icon-primary" />
              <span class="text-body-medium">{{ provider.name }}</span>
              <span class="text-caption-2-regular text-text-tertiary">{{ provider.company }}</span>
            </div>
          </div>
          <p class="relative text-center text-caption-1-regular text-text-secondary">{{ t('home.redesign.availability') }}</p>
        </div>
      </section>

      <section v-if="!compactHomeEnabled" class="grid gap-4 pb-12 md:grid-cols-3" :aria-label="t('home.redesign.featuresLabel')">
        <article v-for="feature in features" :key="feature.key" class="flex flex-col gap-4 rounded-3xl border border-border-button-default bg-background-primary-default p-6">
          <div class="flex h-10 w-10 items-center justify-center rounded-xl bg-background-secondary-default"><Icon :name="feature.icon" size="lg" class="text-foreground-icon-secondary" aria-hidden="true" /></div>
          <h2 class="text-title-3-semibold">{{ t(`home.features.${feature.key}`) }}</h2>
          <p class="text-body-regular text-text-secondary">{{ t(`home.features.${feature.key}Desc`) }}</p>
        </article>
      </section>
    </main>
    <footer class="border-t border-separator-border px-5 py-6 sm:px-8">
      <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 text-caption-1-regular text-text-secondary">
        <p class="min-w-0 break-all">&copy; {{ currentYear }} {{ siteName }}</p>
        <div class="flex flex-wrap items-center gap-5"><span>{{ t('home.redesign.footer') }}</span><a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-link">{{ t('home.docs') }}</a><router-link to="/key-usage" class="home-link">{{ t('keyUsage.title') }}</router-link></div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import Button from '@/components/boardui/Button.vue'
import Badge from '@/components/boardui/Badge.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || t('home.heroSubtitle'))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const isHomeContentUrl = computed(() => /^https?:\/\//.test(homeContent.value.trim()))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
const destination = computed(() => isAuthenticated.value ? dashboardPath.value : '/login')
const showModelPlazaEntry = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (isAuthenticated.value || appStore.cachedPublicSettings?.model_plaza_require_auth !== true))
const currentYear = new Date().getFullYear()
const isDark = ref(document.documentElement.classList.contains('dark'))
const providers = [
  { name: 'Claude', company: 'Anthropic', platform: 'anthropic' as const },
  { name: 'GPT', company: 'OpenAI', platform: 'openai' as const },
  { name: 'Gemini', company: 'Google', platform: 'gemini' as const },
]
const features = [{ key: 'unifiedGateway', icon: 'link' as const }, { key: 'multiAccount', icon: 'sync' as const }, { key: 'balanceQuota', icon: 'chart' as const }]
function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
onMounted(() => {
  const savedTheme = localStorage.getItem('theme')
  isDark.value = savedTheme === 'dark' || (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  document.documentElement.classList.toggle('dark', isDark.value)
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
})
</script>

<style scoped>
.home-link {
  @apply rounded text-text-secondary outline-none transition-colors duration-150 hover:text-text-primary focus-visible:ring-2 focus-visible:ring-border-focus-ring;
}
.home-action {
  @apply inline-flex min-h-10 items-center justify-center gap-2 rounded-xl border px-4 py-2 text-body-medium outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-border-focus-ring focus-visible:ring-offset-2;
}
.home-action-primary {
  @apply border-transparent bg-gradient-to-b from-background-accent-default to-background-accent-hover text-text-white shadow-sm hover:brightness-105 active:brightness-95;
}
.home-action-secondary {
  @apply border-border-button-default bg-background-primary-default text-text-primary shadow-sm hover:bg-background-primary-hover;
}
.route-map {
  background-image: radial-gradient(rgb(var(--boardui-border-button-hover)) 1px, transparent 1px);
  background-size: 16px 16px;
}
</style>
