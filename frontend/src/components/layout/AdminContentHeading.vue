<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore, useAuthStore, useAdminSettingsStore } from '@/stores'
import { resolveRouteMetaKeys } from '@/router/title'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import BoardBreadcrumb from '@/components/boardui/Breadcrumb.vue'
import BoardButton from '@/components/boardui/Button.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import Icon from '@/components/icons/Icon.vue'

const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()
const { t } = useI18n()
const keys = computed(() => resolveRouteMetaKeys(route, { billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings) }))
const title = computed(() => {
  if (route.name === 'CustomPage') {
    const id = route.params.id
    const item = appStore.cachedPublicSettings?.custom_menu_items?.find(item => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find(item => item.id === id) : undefined)
    if (item?.label) return item.label
  }
  return keys.value.titleKey ? t(keys.value.titleKey) : String(route.meta.title || '')
})
const breadcrumbs = computed(() => [
  { label: appStore.siteName, to: authStore.isAdmin ? '/admin/dashboard' : '/dashboard' },
  ...(route.meta.workspace === 'account' ? [{ label: t('nav.myAccount') }] : []),
  { label: title.value }
])
const description = computed(() => keys.value.descriptionKey ? t(keys.value.descriptionKey) : String(route.meta.description || ''))
</script>

<template>
  <section class="admin-content-heading mb-6 flex flex-col gap-5 lg:mb-4 lg:gap-3" data-boardui="content-heading">
    <div class="flex min-w-0 items-center gap-3">
      <BoardButton data-testid="mobile-menu-toggle" variant="ghost" size="small" icon-only class="lg:hidden" :aria-label="t('common.toggleMenu')" @click="appStore.toggleMobileSidebar()"><Icon name="menu" size="md" /></BoardButton>
      <BoardBreadcrumb :label="t('boardui.breadcrumb')" :items="breadcrumbs" />
    </div>
    <div class="flex min-w-0 items-start justify-between gap-4">
      <div class="min-w-0">
        <h1 class="text-title-1-semibold text-text-primary">{{ title }}</h1>
        <p v-if="description" class="mt-1 text-body-regular text-text-secondary">{{ description }}</p>
      </div>
      <div class="flex shrink-0 items-center gap-2"><AnnouncementBell /></div>
    </div>
  </section>
</template>
