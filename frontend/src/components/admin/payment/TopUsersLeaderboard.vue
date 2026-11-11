<template>
  <div class="card p-4">
    <h3 class="mb-4 text-body-semibold text-text-primary ">
      {{ t('payment.admin.topUsers') }}
    </h3>
    <div
      v-if="!hasUsers(props.users)"
      class="flex h-32 items-center justify-center text-body-regular text-text-secondary "
    >
      {{ t('payment.admin.noData') }}
    </div>
    <div v-else class="space-y-2">
      <div v-for="[currency, currencyUsers] in sortedUsers(props.users)" :key="currency" class="space-y-2">
        <p class="text-caption-1-semibold text-text-secondary ">{{ currency }}</p>
        <div
          v-for="(user, idx) in currencyUsers"
          :key="user.user_id"
          class="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-background-secondary-default "
        >
          <div class="flex items-center gap-3">
            <span
              :class="[
                'flex h-6 w-6 items-center justify-center rounded-full text-caption-1-bold',
                rankClass(idx),
              ]"
            >
              {{ idx + 1 }}
            </span>
            <span class="text-body-regular text-text-primary ">{{ user.email }}</span>
          </div>
          <span class="text-body-medium text-text-primary ">
            {{ formatMoney(currency, user.amount) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { TopUserPaymentStats } from '@/types/payment'

const { t } = useI18n()

const props = defineProps<{
  users: Record<string, TopUserPaymentStats[]>
}>()

function rankClass(idx: number): string {
  if (idx === 0) return 'bg-background-warning-subtle text-text-warning '
  if (idx === 1) return 'bg-background-tertiary-default text-text-secondary '
  if (idx === 2) return 'bg-background-warning-subtle text-text-warning '
  return 'bg-background-secondary-default text-text-secondary '
}

function hasUsers(usersByCurrency: Record<string, TopUserPaymentStats[]>): boolean {
  return Object.values(usersByCurrency).some(users => users.length > 0)
}

function sortedUsers(usersByCurrency: Record<string, TopUserPaymentStats[]>): [string, TopUserPaymentStats[]][] {
  return Object.entries(usersByCurrency).sort(([left], [right]) => left.localeCompare(right))
}

function formatMoney(currency: string, amount: number): string {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(amount)
}
</script>
