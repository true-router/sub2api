<template>
  <section class="card mb-4 flex flex-wrap items-center justify-between gap-3 p-3">
    <BoardSegmentedControl
      :model-value="window"
      :items="windowOptions"
      :label="t('channelStatus.title')"
      @update:model-value="selectWindow"
    />
    <div class="flex flex-wrap items-center gap-2">
      <BoardBadge :tone="overallStatus === 'operational' ? 'success' : 'warning'" dot>
        {{ overallLabel }}
      </BoardBadge>
      <BoardButton
        type="button"
        variant="secondary"
        size="small"
        icon-only
        :disabled="loading"
        :aria-label="t('common.refresh')"
        :title="t('common.refresh')"
        @click="emit('refresh')"
      >
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
      </BoardButton>
      <AutoRefreshButton
        v-if="autoRefresh"
        :enabled="autoRefresh.enabled.value"
        :interval-seconds="autoRefresh.intervalSeconds.value"
        :countdown="autoRefresh.countdown.value"
        :intervals="autoRefresh.intervals"
        @update:enabled="autoRefresh.setEnabled"
        @update:interval="autoRefresh.setInterval"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardSegmentedControl from '@/components/boardui/SegmentedControl.vue'
import Icon from '@/components/icons/Icon.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
export type MonitorWindow = '7d' | '15d' | '30d'
export type OverallStatus = 'operational' | 'degraded'

const props = defineProps<{
  overallStatus: OverallStatus
  intervalSeconds: number
  window: MonitorWindow
  loading: boolean
  autoRefresh?: {
    enabled: { value: boolean }
    intervalSeconds: { value: number }
    countdown: { value: number }
    intervals: readonly number[]
    setEnabled: (v: boolean) => void
    setInterval: (v: number) => void
  }
}>()

const emit = defineEmits<{
  (e: 'update:window', value: MonitorWindow): void
  (e: 'refresh'): void
}>()

const { t } = useI18n()

const windowOptions = computed<{ value: MonitorWindow; label: string }[]>(() => [
  { value: '7d', label: t('channelStatus.windowTab.7d') },
  { value: '15d', label: t('channelStatus.windowTab.15d') },
  { value: '30d', label: t('channelStatus.windowTab.30d') },
])

const overallLabel = computed(() => t(`channelStatus.overall.${props.overallStatus}`))

function selectWindow(value: string) {
  const option = windowOptions.value.find(option => option.value === value)
  if (option) emit('update:window', option.value)
}
</script>
