<template>
  <BaseDialog :show="show" :title="t('admin.users.replaceGroupTitle')" width="narrow" @close="$emit('close')">
    <div v-if="oldGroup" class="space-y-4">
      <!-- 提示信息 -->
      <p class="text-body-regular text-text-secondary ">
        {{ t('admin.users.replaceGroupHint', { old: oldGroup.name }) }}
      </p>

      <!-- 当前分组 -->
      <div class="rounded-lg border border-border-button-default bg-background-secondary-default p-3 ">
        <div class="flex items-center gap-2">
          <Icon name="shield" size="sm" class="text-text-feature" />
          <span class="font-medium text-text-primary ">{{ oldGroup.name }}</span>
          <Icon name="arrowRight" size="sm" class="ml-auto text-text-tertiary" />
          <span v-if="selectedGroupId" class="font-medium text-text-accent ">
            {{ availableGroups.find(g => g.id === selectedGroupId)?.name }}
          </span>
          <span v-else class="text-body-regular text-text-tertiary">?</span>
        </div>
      </div>

      <!-- 可选分组列表 -->
      <div v-if="availableGroups.length > 0" class="max-h-64 space-y-2 overflow-y-auto">
        <label
          v-for="group in availableGroups"
          :key="group.id"
          class="flex cursor-pointer items-center gap-3 rounded-lg border-2 p-3 transition-all"
          :class="selectedGroupId === group.id
            ? 'border-border-focus-ring bg-background-accent-subtle/50 '
            : 'border-border-button-default hover:border-border-button-hover '"
        >
          <input
            type="radio"
            :value="group.id"
            v-model="selectedGroupId"
            class="sr-only"
          />
          <div
            class="flex h-5 w-5 items-center justify-center rounded-full border-2 transition-all"
            :class="selectedGroupId === group.id
              ? 'border-border-focus-ring bg-background-accent-default'
              : 'border-border-button-hover '"
          >
            <div v-if="selectedGroupId === group.id" class="h-2 w-2 rounded-full bg-background-primary-default"></div>
          </div>
          <div class="flex-1">
            <span class="font-medium text-text-primary ">{{ group.name }}</span>
            <span class="ml-2 text-caption-1-regular text-text-tertiary">{{ group.platform }}</span>
          </div>
        </label>
      </div>

      <!-- 无可选分组 -->
      <div v-else class="py-6 text-center text-body-regular text-text-tertiary">
        {{ t('admin.users.noOtherGroups') }}
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <BoardButton type="submit" @click="$emit('close')" class="px-5" variant="secondary" size="medium">{{ t('common.cancel') }}</BoardButton>
        <BoardButton type="submit"
          @click="handleReplace"
          :disabled="!selectedGroupId || submitting"
          class="px-6" variant="primary" size="medium"
        >
          <svg v-if="submitting" class="-ml-1 mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ submitting ? t('common.saving') : t('admin.users.replaceGroupConfirm') }}
        </BoardButton>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { AdminUser, AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

interface Props {
  show: boolean
  user: AdminUser | null
  oldGroup: { id: number; name: string } | null
  allGroups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits(['close', 'success'])
const { t } = useI18n()
const appStore = useAppStore()

const selectedGroupId = ref<number | null>(null)
const submitting = ref(false)

// 可选的专属标准分组（排除当前 oldGroup）
const availableGroups = computed(() => {
  if (!props.oldGroup) return []
  return props.allGroups.filter(
    g => g.status === 'active' && g.is_exclusive && g.subscription_type === 'standard' && g.id !== props.oldGroup!.id
  )
})

watch(() => props.show, (v) => {
  if (v) {
    selectedGroupId.value = null
  }
})

const handleReplace = async () => {
  if (!props.user || !props.oldGroup || !selectedGroupId.value) return
  submitting.value = true

  try {
    const result = await adminAPI.users.replaceGroup(props.user.id, props.oldGroup.id, selectedGroupId.value)
    appStore.showSuccess(t('admin.users.replaceGroupSuccess', { count: result.migrated_keys }))
    emit('success')
    emit('close')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
    console.error('Failed to replace group:', error)
  } finally {
    submitting.value = false
  }
}
</script>
