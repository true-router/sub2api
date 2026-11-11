<template>
  <div class="mb-4 flex items-center justify-between rounded-lg bg-background-accent-subtle p-3 ">
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="allResultsSelected" class="text-body-medium text-text-accent ">
        {{ t('admin.accounts.bulkActions.selectedAll', { count: selectedIds.length }) }}
      </span>
      <span v-else-if="selectedIds.length > 0" class="text-body-medium text-text-accent ">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-body-medium text-text-accent ">
        {{ t('admin.accounts.bulkEdit.title') }}
      </span>
      <template v-if="selectedIds.length > 0">
        <button
          @click="$emit('select-page')"
          class="text-caption-1-medium text-text-accent hover:text-text-accent "
        >
          {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
        </button>
      </template>
      <template v-if="!allResultsSelected && totalResults > selectedIds.length">
        <span v-if="selectedIds.length > 0" class="text-text-disabled ">•</span>
        <button
          :disabled="selectingAll"
          @click="$emit('select-all-results')"
          class="text-caption-1-medium text-text-accent hover:text-text-accent disabled:cursor-not-allowed disabled:opacity-60 "
        >
          {{
            selectingAll
              ? t('admin.accounts.bulkActions.selectingAll')
              : t('admin.accounts.bulkActions.selectAllResults', { count: totalResults })
          }}
        </button>
      </template>
      <template v-if="selectedIds.length > 0">
        <span class="text-text-disabled ">•</span>
        <button
          @click="$emit('clear')"
          class="text-caption-1-medium text-text-accent hover:text-text-accent "
        >
          {{ t('admin.accounts.bulkActions.clear') }}
        </button>
      </template>
    </div>
    <div class="flex gap-2">
      <template v-if="selectedIds.length > 0">
        <BoardButton type="submit" @click="$emit('delete')" variant="danger" size="small">{{ t('admin.accounts.bulkActions.delete') }}</BoardButton>
        <BoardButton type="submit" @click="$emit('reset-status')" variant="secondary" size="small">{{ t('admin.accounts.bulkActions.resetStatus') }}</BoardButton>
        <BoardButton type="submit" @click="$emit('refresh-token')" variant="secondary" size="small">{{ t('admin.accounts.bulkActions.refreshToken') }}</BoardButton>
        <BoardButton type="submit" @click="$emit('probe-upstream-billing')" variant="secondary" size="small">{{ t('admin.accounts.bulkActions.probeUpstreamBilling') }}</BoardButton>
        <BoardButton type="submit" @click="$emit('toggle-schedulable', true)" variant="success" size="small">{{ t('admin.accounts.bulkActions.enableScheduling') }}</BoardButton>
        <BoardButton type="submit" @click="$emit('toggle-schedulable', false)" variant="warning" size="small">{{ t('admin.accounts.bulkActions.disableScheduling') }}</BoardButton>
        <BoardButton type="submit" @click="$emit('edit-selected')" variant="primary" size="small">{{ t('admin.accounts.bulkActions.edit') }}</BoardButton>
      </template>
      <BoardButton type="submit" @click="$emit('edit-filtered')" variant="primary" size="small">
        {{ t('admin.accounts.bulkEdit.submit') }}
      </BoardButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { useI18n } from 'vue-i18n'

defineProps<{
  selectedIds: number[]
  totalResults: number
  selectingAll: boolean
  allResultsSelected: boolean
}>()

defineEmits([
  'delete',
  'edit-selected',
  'edit-filtered',
  'clear',
  'select-page',
  'select-all-results',
  'toggle-schedulable',
  'reset-status',
  'refresh-token',
  'probe-upstream-billing'
])

const { t } = useI18n()
</script>
