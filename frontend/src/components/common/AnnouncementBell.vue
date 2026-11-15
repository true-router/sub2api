<script setup lang="ts">
import { computed, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import BaseDialog from '@/components/common/BaseDialog.vue'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardSegmentedControl from '@/components/boardui/SegmentedControl.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAppStore } from '@/stores/app'
import { formatRelativeWithDateTime } from '@/utils/format'
import type { UserNotification } from '@/api/notifications'
const { t, locale } = useI18n()
const inbox = useAnnouncementStore()
const app = useAppStore()
const {announcements,loading}=storeToRefs(inbox)
const show = ref(false)
const filter = ref('all')
const selected = ref<UserNotification | null>(null)
const filtered = computed(() => announcements.value.filter(n => filter.value === 'all' || n.type === filter.value))
const unreadAnnouncements = computed(() => announcements.value.some(n => n.type === 'announcement' && !n.read_at))
const content = computed(() => DOMPurify.sanitize(marked.parse(selected.value?.content || '',{async:false}) as string))
function open(){ show.value=true; void inbox.fetchAnnouncements(true) }
async function detail(item: UserNotification){
 if(item.type==='gift'){show.value=false;inbox.openGift(item);return}
 selected.value=item
 if(!item.read_at && !await inbox.markAsRead(item.id,item.type)) app.showError(t('common.unknownError'))
}
async function markAll(){try{await inbox.markAllAsRead()}catch{app.showError(t('common.unknownError'))}}
</script>
<template>
  <div>
    <BoardButton variant="ghost" size="small" icon-only class="relative" :aria-label="t('giftNotice.inbox')" data-testid="notification-bell" @click="open"><Icon name="bell" size="md" /><span v-if="inbox.unreadCount" class="absolute end-1 top-1 h-2 w-2 rounded-full bg-background-error-default" /></BoardButton>
    <BaseDialog :show="show" :title="t('giftNotice.inbox')" :z-index="100" @close="show=false">
      <div class="space-y-4" data-testid="notification-inbox">
        <BoardSegmentedControl v-model="filter" :label="t('giftNotice.inbox')" :items="[{value:'all',label:t('giftNotice.all')},{value:'announcement',label:t('giftNotice.announcements')},{value:'gift',label:t('giftNotice.gifts')}]" />
        <BoardButton v-if="unreadAnnouncements" variant="ghost" :disabled="loading" data-testid="mark-announcements-read" @click="markAll">{{ t('giftNotice.markAnnouncements') }}</BoardButton>
        <p v-if="loading" class="text-body-regular text-text-secondary">{{ t('common.loading') }}</p>
        <p v-else-if="!filtered.length" class="text-body-regular text-text-secondary">{{ t('giftNotice.empty') }}</p>
        <div class="space-y-2"><BoardButton v-for="item in filtered" :key="`${item.type}:${item.id}`" variant="secondary" class="h-auto w-full justify-start whitespace-normal p-4 text-start" :data-notification="`${item.type}:${item.id}`" @click="detail(item)">
          <Icon :name="item.type==='gift' ? 'gift' : 'bell'" size="md" class="shrink-0 text-foreground-icon-secondary" />
          <span class="min-w-0 flex-1 space-y-1"><span class="block break-words text-body-medium text-text-primary">{{ item.type==='gift' ? t('giftNotice.amount',{amount: new Intl.NumberFormat(locale,{style:'currency',currency:'USD',maximumFractionDigits:8}).format(item.gift_amount ?? 0)}) : item.title }}</span><span class="block text-caption-1-regular text-text-secondary">{{ formatRelativeWithDateTime(item.created_at) }}</span></span>
          <BoardBadge :tone="item.read_at ? 'neutral' : 'accent'">{{ t(item.type==='gift' ? item.read_at ? 'giftNotice.claimed' : 'giftNotice.pending' : item.read_at ? 'announcements.read' : 'announcements.unread') }}</BoardBadge>
        </BoardButton></div>
      </div>
    </BaseDialog>
    <BaseDialog :show="!!selected" :title="selected?.title || ''" :z-index="120" @close="selected=null"><div class="markdown-body break-words" v-html="content" /></BaseDialog>
  </div>
</template>
