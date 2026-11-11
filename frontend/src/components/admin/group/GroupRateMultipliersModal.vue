<template>
  <BaseDialog :show="show" :title="t('admin.groups.rateMultipliersTitle')" width="wide" @close="handleClose">
    <div v-if="group" class="space-y-4">
      <!-- 分组信息 -->
      <div class="flex flex-wrap items-center gap-3 rounded-lg bg-background-secondary-default px-4 py-2.5 text-body-regular ">
        <span class="inline-flex items-center gap-1.5" :class="platformColorClass">
          <PlatformIcon :platform="group.platform" size="sm" />
          {{ t('admin.groups.platforms.' + group.platform, platformLabel(group.platform)) }}
        </span>
        <span class="text-text-tertiary">|</span>
        <span class="font-medium text-text-primary ">{{ group.name }}</span>
        <span class="text-text-tertiary">|</span>
        <span class="text-text-secondary ">
          {{ t('admin.groups.columns.rateMultiplier') }}: {{ group.rate_multiplier }}x
        </span>
      </div>

      <!-- 操作区 -->
      <div class="rounded-lg border border-border-button-default p-3 ">
        <!-- 添加用户 -->
        <h4 class="mb-2 text-body-medium text-text-primary ">
          {{ t('admin.groups.addUserRate') }}
        </h4>
        <div class="flex items-end gap-2">
          <div class="relative flex-1">
            <input
              v-model="searchQuery"
              type="text"
              autocomplete="off"
              class="input w-full"
              :placeholder="t('admin.groups.searchUserPlaceholder')"
              @input="handleSearchUsers"
              @focus="showDropdown = true"
            />
            <div
              v-if="showDropdown && searchResults.length > 0"
              class="absolute left-0 right-0 top-full z-10 mt-1 max-h-48 overflow-y-auto rounded-lg border border-border-button-default bg-background-primary-default shadow-lg "
            >
              <button
                v-for="user in searchResults"
                :key="user.id"
                type="button"
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-body-regular hover:bg-background-secondary-default "
                @click="selectUser(user)"
              >
                <span class="text-text-tertiary">#{{ user.id }}</span>
                <span class="text-text-primary ">{{ user.username || user.email }}</span>
                <span v-if="user.username" class="text-caption-1-regular text-text-tertiary">{{ user.email }}</span>
              </button>
            </div>
          </div>
          <div class="w-24">
            <input
              v-model.number="newRate"
              type="number"
              step="0.001"
              min="0"
              autocomplete="off"
              class="hide-spinner input w-full"
              placeholder="1.0"
            />
          </div>
          <BoardButton
            type="button"
            class="shrink-0" variant="primary" size="medium"
            :disabled="!selectedUser || newRate == null || newRate <= 0"
            @click="handleAddLocal"
          >
            {{ t('common.add') }}
          </BoardButton>
        </div>

        <!-- 批量调整 + 全部清空 -->
        <div v-if="localEntries.length > 0" class="mt-3 flex items-center gap-3 border-t border-border-button-default pt-3 ">
          <span class="text-caption-1-medium text-text-secondary ">{{ t('admin.groups.batchAdjust') }}</span>
          <div class="flex items-center gap-1.5">
            <span class="text-caption-1-regular text-text-tertiary">×</span>
            <input
              v-model.number="batchFactor"
              type="number"
              step="0.1"
              min="0"
              autocomplete="off"
              class="hide-spinner w-20 rounded border border-border-button-default bg-background-primary-default px-2 py-1 text-center text-body-regular transition-colors focus:border-border-focus-ring focus:outline-none focus:ring-1 focus:ring-border-focus-ring/20 "
              placeholder="0.5"
            />
            <BoardButton
              type="button"
              class="shrink-0 px-2.5 py-1 text-caption-1-regular" variant="primary" size="small"
              :disabled="!batchFactor || batchFactor <= 0"
              @click="applyBatchFactor"
            >
              {{ t('admin.groups.applyMultiplier') }}
            </BoardButton>
          </div>
          <div class="ml-auto">
            <button
              type="button"
              class="rounded-lg border border-border-error-default bg-background-error-subtle px-3 py-1.5 text-body-medium text-text-error-primary transition-colors hover:bg-background-error-subtle "
              @click="clearAllLocal"
            >
              {{ t('admin.groups.clearAll') }}
            </button>
          </div>
        </div>
      </div>

      <!-- 加载状态 -->
      <div v-if="loading" class="flex justify-center py-6">
        <svg class="h-6 w-6 animate-spin text-text-accent" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <!-- 已设置的用户列表 -->
      <div v-else>
        <h4 class="mb-2 text-body-medium text-text-primary ">
          {{ t('admin.groups.rateMultipliers') }} ({{ localEntries.length }})
        </h4>

        <div v-if="localEntries.length === 0" class="py-6 text-center text-body-regular text-text-tertiary ">
          {{ t('admin.groups.noRateMultipliers') }}
        </div>

        <div v-else>
          <!-- 表格 -->
          <div class="overflow-hidden rounded-lg border border-border-button-default ">
            <div class="max-h-[420px] overflow-auto">
              <BoardTable size="sm" class="w-full min-w-max text-body-regular">
                <thead class="sticky top-0 z-[1]">
                  <tr class="border-b border-border-button-default bg-background-secondary-default ">
                    <th class="px-3 py-2 text-left text-caption-1-medium text-text-secondary ">{{ t('admin.groups.columns.userEmail') }}</th>
                    <th class="px-3 py-2 text-left text-caption-1-medium text-text-secondary ">ID</th>
                    <th class="px-3 py-2 text-left text-caption-1-medium text-text-secondary ">{{ t('admin.groups.columns.userName') }}</th>
                    <th class="px-3 py-2 text-left text-caption-1-medium text-text-secondary ">{{ t('admin.groups.columns.userNotes') }}</th>
                    <th class="px-3 py-2 text-left text-caption-1-medium text-text-secondary ">{{ t('admin.groups.columns.userStatus') }}</th>
                    <th class="px-3 py-2 text-left text-caption-1-medium text-text-secondary ">{{ t('admin.groups.columns.rateMultiplier') }}</th>
                    <th v-if="showFinalRate" class="px-3 py-2 text-left text-caption-1-medium text-text-accent ">{{ t('admin.groups.finalRate') }}</th>
                    <th class="w-10 px-2 py-2"></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-border-button-default ">
                  <tr
                    v-for="entry in paginatedLocalEntries"
                    :key="entry.user_id"
                    class="hover:bg-background-secondary-default "
                  >
                    <td class="px-3 py-2 text-text-secondary ">{{ entry.user_email }}</td>
                    <td class="whitespace-nowrap px-3 py-2 text-text-tertiary ">{{ entry.user_id }}</td>
                    <td class="whitespace-nowrap px-3 py-2 text-text-primary ">{{ entry.user_name || '-' }}</td>
                    <td class="max-w-[160px] truncate px-3 py-2 text-text-secondary " :title="entry.user_notes">{{ entry.user_notes || '-' }}</td>
                    <td class="whitespace-nowrap px-3 py-2">
                      <span
                        :class="[
                          'inline-flex rounded-full px-2 py-0.5 text-caption-1-medium',
                          entry.user_status === 'active'
                            ? 'bg-background-success-subtle text-text-success '
                            : 'bg-background-secondary-default text-text-secondary '
                        ]"
                      >
                        {{ entry.user_status }}
                      </span>
                    </td>
                    <td class="whitespace-nowrap px-3 py-2">
                      <input
                        type="number"
                        step="0.001"
                        min="0.001"
                        autocomplete="off"
                        :value="entry.rate_multiplier ?? ''"
                        :placeholder="String(props.group?.rate_multiplier ?? 1)"
                        class="hide-spinner w-20 rounded border border-border-button-default bg-background-primary-default px-2 py-1 text-center text-body-medium transition-colors focus:border-border-focus-ring focus:outline-none focus:ring-1 focus:ring-border-focus-ring/20 "
                        @change="updateLocalRate(entry.user_id, ($event.target as HTMLInputElement).value)"
                      />
                    </td>
                    <td v-if="showFinalRate" class="whitespace-nowrap px-3 py-2 font-medium text-text-accent ">
                      {{ computeFinalRate(entry.rate_multiplier) }}
                    </td>
                    <td class="px-2 py-2">
                      <button
                        type="button"
                        class="rounded p-1 text-text-tertiary transition-colors hover:bg-background-error-subtle hover:text-text-error-primary "
                        @click="removeLocal(entry.user_id)"
                      >
                        <Icon name="trash" size="sm" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </BoardTable>
            </div>
          </div>

          <!-- 分页 -->
          <Pagination
            :total="localEntries.length"
            :page="currentPage"
            :page-size="pageSize"
            @update:page="currentPage = $event"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </div>

      <!-- 底部操作栏 -->
      <div class="flex items-center gap-3 border-t border-border-button-default pt-4 ">
        <!-- 左侧：未保存提示 + 撤销 -->
        <template v-if="isDirty">
          <span class="text-caption-1-regular text-text-warning ">{{ t('admin.groups.unsavedChanges') }}</span>
          <button
            type="button"
            class="text-caption-1-medium text-text-accent hover:text-text-accent "
            @click="handleCancel"
          >
            {{ t('admin.groups.revertChanges') }}
          </button>
        </template>
        <!-- 右侧：关闭 / 保存 -->
        <div class="ml-auto flex items-center gap-3">
          <button type="button" class="btn btn-sm px-4 py-1.5" @click="handleClose">
            {{ t('common.close') }}
          </button>
          <BoardButton
            v-if="isDirty"
            type="button"
            class="px-4 py-1.5" variant="primary" size="small"
            :disabled="saving"
            @click="handleSave"
          >
            <Icon v-if="saving" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ t('common.save') }}
          </BoardButton>
        </div>
      </div>
    </div>
  </BaseDialog>

</template>

<script setup lang="ts">
import BoardTable from '@/components/boardui/Table.vue'
import BoardButton from '@/components/boardui/Button.vue'
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { GroupRateMultiplierEntry } from '@/api/admin/groups'
import type { AdminGroup, AdminUser } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformLabel } from '@/utils/platformColors'

interface LocalEntry extends GroupRateMultiplierEntry {}

const props = defineProps<{
  show: boolean
  group: AdminGroup | null
}>()

const emit = defineEmits<{
  close: []
  success: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const saving = ref(false)
const serverEntries = ref<GroupRateMultiplierEntry[]>([])
const localEntries = ref<LocalEntry[]>([])
const searchQuery = ref('')
const searchResults = ref<AdminUser[]>([])
const showDropdown = ref(false)
const selectedUser = ref<AdminUser | null>(null)
const newRate = ref<number | null>(null)
const currentPage = ref(1)
const pageSize = ref(10)
const batchFactor = ref<number | null>(null)

let searchTimeout: ReturnType<typeof setTimeout>
let loadVersion = 0

const platformColorClass = computed(() => {
  switch (props.group?.platform) {
    case 'anthropic': return 'text-text-warning '
    case 'openai': return 'text-text-success '
    case 'antigravity': return 'text-text-feature '
    default: return 'text-text-info '
  }
})

// 是否显示"最终倍率"预览列
const showFinalRate = computed(() => {
  return batchFactor.value != null && batchFactor.value > 0 && batchFactor.value !== 1
})

// 计算最终倍率预览
const computeFinalRate = (rate: number | null | undefined) => {
  const base = rate ?? props.group?.rate_multiplier ?? 1
  if (!batchFactor.value) return base
  return parseFloat((base * batchFactor.value).toFixed(6))
}

// 检测是否有未保存的修改
const isDirty = computed(() => {
  if (localEntries.value.length !== serverEntries.value.length) return true
  const serverMap = new Map(serverEntries.value.map(e => [e.user_id, e.rate_multiplier ?? null]))
  return localEntries.value.some(e => serverMap.get(e.user_id) !== (e.rate_multiplier ?? null))
})

const paginatedLocalEntries = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return localEntries.value.slice(start, start + pageSize.value)
})

const cloneEntries = (entries: GroupRateMultiplierEntry[]): LocalEntry[] => {
  return entries.map(e => ({ ...e }))
}

const loadEntries = async () => {
  if (!props.group) return
  const version = loadVersion
  loading.value = true
  serverEntries.value = []
  localEntries.value = []
  try {
    const raw = await adminAPI.groups.getGroupRateMultipliers(props.group.id)
    if (version !== loadVersion) return
    // 仅显示已设置 rate_multiplier 的条目；rpm_override 在另一个弹窗管理，保留不动
    serverEntries.value = raw.filter(e => e.rate_multiplier != null)
    localEntries.value = cloneEntries(serverEntries.value)
    adjustPage()
  } catch (error) {
    if (version !== loadVersion) return
    appStore.showError(t('admin.groups.failedToLoad'))
    console.error('Error loading group rate multipliers:', error)
  } finally {
    if (version === loadVersion) loading.value = false
  }
}

const adjustPage = () => {
  const totalPages = Math.max(1, Math.ceil(localEntries.value.length / pageSize.value))
  if (currentPage.value > totalPages) {
    currentPage.value = totalPages
  }
}

watch([() => props.show, () => props.group?.id], ([show]) => {
  loadVersion++
  if (show && props.group) {
    currentPage.value = 1
    batchFactor.value = null
    searchQuery.value = ''
    searchResults.value = []
    selectedUser.value = null
    newRate.value = null
    loadEntries()
  }
})

const handlePageSizeChange = (newSize: number) => {
  pageSize.value = newSize
  currentPage.value = 1
}

const handleSearchUsers = () => {
  clearTimeout(searchTimeout)
  selectedUser.value = null
  if (!searchQuery.value.trim()) {
    searchResults.value = []
    showDropdown.value = false
    return
  }
  searchTimeout = setTimeout(async () => {
    try {
      const res = await adminAPI.users.list(1, 10, { search: searchQuery.value.trim() })
      searchResults.value = res.items
      showDropdown.value = true
    } catch {
      searchResults.value = []
    }
  }, 300)
}

const selectUser = (user: AdminUser) => {
  selectedUser.value = user
  searchQuery.value = user.email
  showDropdown.value = false
  searchResults.value = []
}

// 本地添加（或覆盖已有用户）
const handleAddLocal = () => {
  if (!selectedUser.value || newRate.value == null || newRate.value <= 0) return
  const user = selectedUser.value
  const idx = localEntries.value.findIndex(e => e.user_id === user.id)
  const entry: LocalEntry = {
    user_id: user.id,
    user_name: user.username || '',
    user_email: user.email,
    user_notes: user.notes || '',
    user_status: user.status || 'active',
    rate_multiplier: newRate.value,
    rpm_override: null
  }
  if (idx >= 0) {
    localEntries.value[idx] = entry
  } else {
    localEntries.value.push(entry)
  }
  searchQuery.value = ''
  selectedUser.value = null
  newRate.value = null
  adjustPage()
}

// 本地修改倍率
const updateLocalRate = (userId: number, value: string) => {
  const entry = localEntries.value.find(e => e.user_id === userId)
  if (!entry) return
  if (value.trim() === '') {
    entry.rate_multiplier = null
    return
  }
  const num = parseFloat(value)
  if (!Number.isFinite(num) || num <= 0) return
  entry.rate_multiplier = num
}

// 本地删除
const removeLocal = (userId: number) => {
  localEntries.value = localEntries.value.filter(e => e.user_id !== userId)
  adjustPage()
}

// 批量乘数应用到本地
const applyBatchFactor = () => {
  if (!batchFactor.value || batchFactor.value <= 0) return
  for (const entry of localEntries.value) {
    if (entry.rate_multiplier != null) {
      entry.rate_multiplier = parseFloat((entry.rate_multiplier * batchFactor.value).toFixed(6))
    }
  }
  batchFactor.value = null
}

// 本地清空
const clearAllLocal = () => {
  localEntries.value = []
}

// 取消：恢复到服务器数据
const handleCancel = () => {
  localEntries.value = cloneEntries(serverEntries.value)
  batchFactor.value = null
  adjustPage()
}

// 保存：一次性提交所有数据（只提交 rate_multiplier；rpm_override 由独立弹窗管理）
const handleSave = async () => {
  if (!props.group) return
  saving.value = true
  try {
    const entries = localEntries.value
      .filter(e => e.rate_multiplier != null)
      .map(e => ({
        user_id: e.user_id,
        rate_multiplier: e.rate_multiplier as number
      }))
    await adminAPI.groups.batchSetGroupRateMultipliers(props.group.id, entries)
    appStore.showSuccess(t('admin.groups.rateSaved'))
    emit('success')
    emit('close')
  } catch (error) {
    appStore.showError(t('admin.groups.failedToSave'))
    console.error('Error saving rate multipliers:', error)
  } finally {
    saving.value = false
  }
}

// 关闭时如果有未保存修改，先恢复
const handleClose = () => {
  if (isDirty.value) {
    localEntries.value = cloneEntries(serverEntries.value)
  }
  emit('close')
}

// 点击外部关闭下拉
const handleClickOutside = () => {
  showDropdown.value = false
}

if (typeof document !== 'undefined') {
  document.addEventListener('click', handleClickOutside)
}
onUnmounted(() => {
  loadVersion++
  clearTimeout(searchTimeout)
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.hide-spinner::-webkit-outer-spin-button,
.hide-spinner::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.hide-spinner {
  -moz-appearance: textfield;
}
</style>
