<template>
  <section class="card min-w-0 overflow-hidden" data-testid="dashboard-recent-usage">
    <header class="flex flex-wrap items-center justify-between gap-3 p-4">
      <h2 class="text-title-3-semibold text-text-primary">{{ t('dashboard.recentUsage') }}</h2>
      <router-link to="/usage" class="inline-flex items-center gap-1 rounded text-body-medium text-text-accent outline-none hover:underline focus-visible:ring-2 focus-visible:ring-border-focus-ring">
        {{ t('dashboard.viewAllUsage') }}<Icon name="arrowRight" size="sm" />
      </router-link>
    </header>
    <div v-if="loading" class="space-y-4 p-4">
      <BoardSkeleton v-for="index in 3" :key="index" height="1.5rem" />
    </div>
    <div v-else-if="data.length === 0" class="p-4">
      <EmptyState :title="t('dashboard.noUsageRecords')" :description="t('dashboard.startUsingApi')" />
    </div>
    <div v-else class="overflow-x-auto">
      <BoardTable size="sm" class="min-w-[520px]">
        <thead>
          <tr>
            <th scope="col">{{ t('dashboard.model') }}</th>
            <th scope="col">{{ t('usage.time') }}</th>
            <th scope="col" class="text-end">{{ t('dashboard.tokens') }}</th>
            <th scope="col" class="text-end">{{ t('dashboard.actual') }} / {{ t('dashboard.standard') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in data" :key="log.id">
            <td class="max-w-[200px] truncate" :title="log.model">{{ log.model }}</td>
            <td class="whitespace-nowrap text-text-secondary">{{ formatDateTime(log.created_at) }}</td>
            <td class="text-end tabular-nums">{{ (log.input_tokens + log.output_tokens).toLocaleString() }}</td>
            <td class="whitespace-nowrap text-end tabular-nums">
              <span class="text-text-primary">${{ formatCost(log.actual_cost) }}</span>
              <span class="text-text-tertiary"> / ${{ formatCost(log.total_cost) }}</span>
            </td>
          </tr>
        </tbody>
      </BoardTable>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BoardTable from '@/components/boardui/Table.vue'
import BoardSkeleton from '@/components/boardui/Skeleton.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDateTime } from '@/utils/format'
import type { UsageLog } from '@/types'

defineProps<{ data: UsageLog[]; loading: boolean }>()
const { t } = useI18n()
const formatCost = (value: number) => value.toFixed(4)
</script>
