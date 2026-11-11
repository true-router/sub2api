<template>
  <div>
    <!-- Loading state -->
    <div v-if="props.loading && !props.stats" class="space-y-0.5">
      <div class="h-3 w-12 animate-pulse rounded bg-background-tertiary-default "></div>
      <div class="h-3 w-16 animate-pulse rounded bg-background-tertiary-default "></div>
      <div class="h-3 w-10 animate-pulse rounded bg-background-tertiary-default "></div>
    </div>

    <!-- Error state -->
    <div v-else-if="props.error && !props.stats" class="text-caption-1-regular text-text-error-primary">
      {{ props.error }}
    </div>

    <!-- Stats data -->
    <div v-else-if="props.stats" class="space-y-0.5 text-caption-1-regular">
      <!-- Requests -->
      <div class="flex items-center gap-1">
        <span class="text-text-secondary "
          >{{ t('admin.accounts.stats.requests') }}:</span
        >
        <span class="font-medium text-text-primary ">{{
          formatNumber(props.stats.requests)
        }}</span>
      </div>
      <!-- Tokens -->
      <div class="flex items-center gap-1">
        <span class="text-text-secondary "
          >{{ t('admin.accounts.stats.tokens') }}:</span
        >
        <span class="font-medium text-text-primary ">{{
          formatTokens(props.stats.tokens)
        }}</span>
      </div>
      <!-- Cost (Account) -->
      <div class="flex items-center gap-1">
        <span class="text-text-secondary ">{{ t('usage.accountBilled') }}:</span>
        <span class="font-medium text-text-success ">{{
          formatCurrency(props.stats.cost)
        }}</span>
      </div>
      <!-- Cost (User/API Key) -->
      <div v-if="props.stats.user_cost != null" class="flex items-center gap-1">
        <span class="text-text-secondary ">{{ t('usage.userBilled') }}:</span>
        <span class="font-medium text-text-primary ">{{
          formatCurrency(props.stats.user_cost)
        }}</span>
      </div>
    </div>

    <!-- No data -->
    <div v-else class="text-caption-1-regular text-text-tertiary">-</div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { WindowStats } from '@/types'
import { formatNumber, formatCurrency } from '@/utils/format'

const props = withDefaults(
  defineProps<{
    stats?: WindowStats | null
    loading?: boolean
    error?: string | null
  }>(),
  {
    stats: null,
    loading: false,
    error: null
  }
)

const { t } = useI18n()

// Format large token numbers (e.g., 1234567 -> 1.23M)
const formatTokens = (tokens: number): string => {
  if (tokens >= 1000000) {
    return `${(tokens / 1000000).toFixed(2)}M`
  } else if (tokens >= 1000) {
    return `${(tokens / 1000).toFixed(1)}K`
  }
  return tokens.toString()
}
</script>
