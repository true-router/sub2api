<template>
  <div data-testid="balance-breakdown" class="space-y-2 text-caption-1-regular text-text-secondary">
    <dl class="flex flex-wrap gap-x-4 gap-y-1">
      <div class="flex gap-1">
        <dt>{{ t('wallet.paidBalance') }}</dt>
        <dd class="tabular-nums text-text-primary">{{ formatCurrency(user?.paid_balance ?? 0) }}</dd>
      </div>
      <div class="flex gap-1">
        <dt>{{ t('wallet.bonusBalance') }}</dt>
        <dd class="tabular-nums text-text-primary">{{ formatCurrency(user?.bonus_balance ?? 0) }}</dd>
      </div>
      <div v-if="(user?.unverified_balance ?? 0) > 0" class="flex gap-1">
        <dt>{{ t('wallet.unverifiedBalance') }}</dt>
        <dd class="tabular-nums text-text-primary">{{ formatCurrency(user?.unverified_balance ?? 0) }}</dd>
      </div>
    </dl>
    <p>{{ t('wallet.spendRule') }}</p>
    <p v-if="(user?.unverified_balance ?? 0) > 0">{{ t('wallet.unverifiedRule') }}</p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { User } from '@/types'
import { formatCurrency } from '@/utils/format'

defineProps<{ user?: Pick<User, 'paid_balance' | 'bonus_balance' | 'unverified_balance'> | null }>()
const { t } = useI18n()
</script>
