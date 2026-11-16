<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UsageLog } from '@/types'
defineProps<{ record: UsageLog }>()
const { t } = useI18n()
</script>
<template>
  <div class="space-y-1 tabular-nums">
    <p class="text-body-medium text-text-primary">P {{ record.actual_cost.toFixed(6) }}</p>
    <div v-if="record.billing_source" class="text-caption-1-regular text-text-secondary" data-testid="billing-source">
      <p v-if="record.billing_source.subscription_id">{{ t('apiAccess.subscription') }} · {{ record.billing_source.subscription_name }} · P {{ record.billing_source.subscription_cost.toFixed(6) }}</p>
      <p v-if="record.billing_source.bonus_cost > 0">{{ t('apiAccess.bonus') }} · P {{ record.billing_source.bonus_cost.toFixed(6) }}</p>
      <p v-if="record.billing_source.paid_cost > 0">{{ t('apiAccess.paid') }} · P {{ record.billing_source.paid_cost.toFixed(6) }}</p>
      <p v-if="record.billing_source.unverified_cost > 0">{{ t('apiAccess.unknown') }} · P {{ record.billing_source.unverified_cost.toFixed(6) }}</p>
    </div>
    <p v-else class="text-caption-1-regular text-text-secondary">{{ t(record.subscription_id ? 'apiAccess.subscription' : 'console.wallet') }}</p>
  </div>
</template>
