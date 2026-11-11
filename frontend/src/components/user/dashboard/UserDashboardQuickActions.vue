<template>
  <section class="card" data-testid="dashboard-quick-actions">
    <h2 class="px-4 pt-4 text-title-3-semibold text-text-primary">{{ t('dashboard.quickActions') }}</h2>
    <div class="space-y-1 p-2">
      <router-link v-for="action in actions" :key="action.path" :to="action.path"
        class="flex items-center gap-3 rounded-xl p-3 text-start outline-none transition-colors hover:bg-background-secondary-hover focus-visible:ring-2 focus-visible:ring-border-focus-ring">
        <span class="flex h-9 w-9 shrink-0 items-center justify-center rounded-[10px] bg-background-secondary-default text-foreground-icon-primary"><Icon :name="action.icon" size="md" /></span>
        <span class="min-w-0 flex-1">
          <span class="block text-body-medium text-text-primary">{{ t(action.label) }}</span>
          <span class="block text-caption-1-regular text-text-secondary">{{ t(action.description) }}</span>
        </span>
        <Icon name="chevronRight" size="sm" class="text-foreground-icon-tertiary" />
      </router-link>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const actions = computed(() => [
  { path: '/keys', icon: 'key' as const, label: 'dashboard.createApiKey', description: 'dashboard.generateNewKey' },
  { path: '/usage', icon: 'chart' as const, label: 'dashboard.viewUsage', description: 'dashboard.checkDetailedLogs' },
  ...(canUseBatchImage.value ? [{ path: '/batch-image', icon: 'sparkles' as const, label: 'dashboard.batchImageAgent', description: 'dashboard.batchImageAgentDesc' }] : []),
  { path: '/redeem', icon: 'gift' as const, label: 'dashboard.redeemCode', description: 'dashboard.addBalanceWithCode' },
])
onMounted(() => { void refreshBatchImageAccess() })
</script>
