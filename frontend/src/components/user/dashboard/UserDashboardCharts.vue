<template>
  <div class="space-y-4">
    <div class="card flex flex-wrap items-center justify-between gap-3 p-4">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <span class="text-body-medium text-text-secondary">{{ t('dashboard.timeRange') }}</span>
        <DateRangePicker
          :start-date="startDate"
          :end-date="endDate"
          @update:start-date="$emit('update:startDate', $event)"
          @update:end-date="$emit('update:endDate', $event)"
          @change="$emit('dateRangeChange')"
        />
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <BoardSegmentedControl
          :model-value="granularity"
          :label="t('dashboard.granularity')"
          :items="[{ value: 'day', label: t('dashboard.day') }, { value: 'hour', label: t('dashboard.hour') }]"
          @update:model-value="changeGranularity"
        />
        <BoardButton
          data-testid="user-dashboard-refresh"
          variant="secondary"
          :disabled="loading"
          @click="$emit('refresh')"
        ><Icon name="refresh" size="sm" />{{ t('common.refresh') }}</BoardButton>
      </div>
    </div>

    <div class="grid min-w-0 grid-cols-1 gap-4 xl:grid-cols-2">
      <ModelDistributionChart
        v-model:metric="metric"
        :model-stats="models"
        :loading="loading"
        :show-source-toggle="false"
        :show-metric-toggle="true"
        :show-account-cost="false"
        :enable-breakdown="false"
      />
      <TokenUsageTrend :trend-data="trend" :loading="loading" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardButton from '@/components/boardui/Button.vue'
import BoardSegmentedControl from '@/components/boardui/SegmentedControl.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelDistributionChart from '@/components/charts/ModelDistributionChart.vue'
import TokenUsageTrend from '@/components/charts/TokenUsageTrend.vue'
import type { TrendDataPoint, ModelStat } from '@/types'

defineProps<{ loading: boolean; startDate: string; endDate: string; granularity: 'day' | 'hour'; trend: TrendDataPoint[]; models: ModelStat[] }>()
const emit = defineEmits<{
  'update:startDate': [value: string]
  'update:endDate': [value: string]
  'update:granularity': [value: 'day' | 'hour']
  dateRangeChange: []
  granularityChange: []
  refresh: []
}>()
const { t } = useI18n()
const metric = ref<'tokens' | 'actual_cost'>('tokens')

function changeGranularity(value: string) {
  if (value !== 'day' && value !== 'hour') return
  emit('update:granularity', value)
  emit('granularityChange')
}
</script>
