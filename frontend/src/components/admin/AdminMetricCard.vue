<script setup lang="ts">
import BoardStatCard from '@/components/boardui/StatCard.vue'
import Icon from '@/components/icons/Icon.vue'
defineProps<{ label: string; value: string | number; icon: InstanceType<typeof Icon>['$props']['name'] }>()
</script>

<template>
  <BoardStatCard :label="label" :value="value" :icon="icon" data-testid="admin-metric-card"><slot /></BoardStatCard>
</template>
