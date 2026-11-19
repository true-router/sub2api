<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SlotUsage } from '@/types'
defineProps<{ usage: SlotUsage }>()
const { t, locale } = useI18n()
function percent(value: number | null) {
  return value == null ? t('vmUser.noData') : new Intl.NumberFormat(locale.value, { style: 'percent', maximumFractionDigits: 1 }).format(value)
}
</script>
<template>
  <div class="space-y-2 text-body-regular text-text-primary" data-testid="slot-usage">
    <p v-if="usage.status === 'unavailable'">{{ t('vmUser.unavailable') }}</p>
    <p v-else-if="!usage.slots?.length">{{ t('vmUser.empty') }}</p>
    <div v-for="slot in usage.slots" v-else :key="slot.slot_id" class="flex flex-wrap items-center gap-3">
      <bdi class="text-body-medium">{{ slot.slot_id }}</bdi>
      <span>{{ t('vmUser.five') }} {{ percent(slot.utilization_5h) }}</span>
      <span>{{ t('vmUser.seven') }} {{ percent(slot.utilization_7d) }}</span>
    </div>
  </div>
</template>
