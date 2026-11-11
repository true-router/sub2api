<template>
  <AppLayout>
    <div class="space-y-4">
      <div v-if="loading && !stats" class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4" aria-busy="true">
        <div v-for="index in 8" :key="index" class="card space-y-4 p-4"><BoardSkeleton height="2rem" width="50%" /><BoardSkeleton height="3rem" /><BoardSkeleton height="2rem" /></div>
      </div>
      <div v-if="error" role="alert" class="card flex items-center justify-between gap-3 p-4 text-body-regular text-text-error-primary">
        <span>{{ error }}</span><BoardButton variant="secondary" :disabled="loading" @click="refreshAll">{{ t('common.refresh') }}</BoardButton>
      </div>
      <template v-if="stats">
        <UserDashboardStats :stats="stats" :balance="user?.balance || 0" :is-simple="authStore.isSimpleMode" :platform-quotas="platformQuotas" />
        <UserDashboardCharts v-model:startDate="startDate" v-model:endDate="endDate" v-model:granularity="granularity" :loading="loadingCharts" :trend="trendData" :models="modelStats" @dateRangeChange="loadRange" @granularityChange="loadCharts" @refresh="refreshAll" />
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div class="lg:col-span-2"><UserDashboardRecentUsage :data="recentUsage" :loading="loadingUsage" /></div>
          <div class="lg:col-span-1"><UserDashboardQuickActions /></div>
        </div>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats as UserStatsType } from '@/api/usage'
import { getMyPlatformQuotas } from '@/api/user'
import AppLayout from '@/components/layout/AppLayout.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import BoardButton from '@/components/boardui/Button.vue'
import UserDashboardStats from '@/components/user/dashboard/UserDashboardStats.vue'
import UserDashboardCharts from '@/components/user/dashboard/UserDashboardCharts.vue'
import UserDashboardRecentUsage from '@/components/user/dashboard/UserDashboardRecentUsage.vue'
import UserDashboardQuickActions from '@/components/user/dashboard/UserDashboardQuickActions.vue'
import type { UsageLog, TrendDataPoint, ModelStat, PlatformQuotaItem } from '@/types'
import { formatDateLocalInput } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const user = computed(() => authStore.user)
const stats = ref<UserStatsType | null>(null)
const error = ref('')
const loading = ref(false)
const loadingUsage = ref(false)
const loadingCharts = ref(false)
const trendData = ref<TrendDataPoint[]>([])
const modelStats = ref<ModelStat[]>([])
const recentUsage = ref<UsageLog[]>([])
const platformQuotas = ref<PlatformQuotaItem[] | null>(null)
const startDate = ref(formatDateLocalInput(new Date(Date.now() - 6 * 86400000)))
const endDate = ref(formatDateLocalInput(new Date()))
const granularity = ref<'day' | 'hour'>('day')
let chartRequest = 0
let usageRequest = 0

async function loadStats() {
  loading.value = true
  try {
    await authStore.refreshUser()
    stats.value = await usageAPI.getDashboardStats()
  } catch {
    error.value = t('dashboard.failedToLoad')
  } finally {
    loading.value = false
  }
}

async function loadCharts() {
  await nextTick()
  const request = ++chartRequest
  loadingCharts.value = true
  try {
    const params = { start_date: startDate.value, end_date: endDate.value, granularity: granularity.value }
    const [trend, models] = await Promise.all([
      usageAPI.getDashboardTrend(params),
      usageAPI.getDashboardModels(params),
    ])
    if (request !== chartRequest) return
    trendData.value = trend.trend ?? []
    modelStats.value = models.models ?? []
  } catch {
    if (request === chartRequest) error.value = t('dashboard.failedToLoad')
  } finally {
    if (request === chartRequest) loadingCharts.value = false
  }
}

async function loadRecent() {
  const request = ++usageRequest
  loadingUsage.value = true
  try {
    const result = await usageAPI.getByDateRange(startDate.value, endDate.value)
    if (request === usageRequest) recentUsage.value = result.items.slice(0, 5)
  } catch {
    if (request === usageRequest) error.value = t('dashboard.failedToLoad')
  } finally {
    if (request === usageRequest) loadingUsage.value = false
  }
}

async function loadPlatformQuotas() {
  try {
    const data = await getMyPlatformQuotas()
    platformQuotas.value = data.platform_quotas ?? []
  } catch {
    error.value = t('dashboard.failedToLoad')
  }
}

async function loadRange() {
  await nextTick()
  error.value = ''
  await Promise.all([loadCharts(), loadRecent()])
}

async function refreshAll() {
  error.value = ''
  await Promise.all([loadStats(), loadCharts(), loadRecent(), loadPlatformQuotas()])
}

onMounted(refreshAll)
</script>
