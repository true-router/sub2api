<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import BaseDialog from '@/components/common/BaseDialog.vue'
import BoardButton from '@/components/boardui/Button.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAppStore } from '@/stores/app'
import { formatRelativeWithDateTime } from '@/utils/format'
import type { Announcement, UserAnnouncement } from '@/types'
import '@/styles/announcement-markdown.css'
type PreviewAnnouncement = Pick<Announcement | UserAnnouncement,'title'|'content'|'created_at'>
const props=withDefaults(defineProps<{announcement?: PreviewAnnouncement|null;preview?:boolean}>(),{announcement:null,preview:false})
const emit=defineEmits<{close:[]}>()
const { t }=useI18n()
const inbox=useAnnouncementStore()
const app=useAppStore()
const busy=ref(false)
const displayedAnnouncement=computed(()=>props.preview?props.announcement:inbox.currentPopup)
const content=computed(()=>DOMPurify.sanitize(marked.parse(displayedAnnouncement.value?.content||'',{async:false,breaks:true,gfm:true}) as string))
async function dismiss(){
 if(props.preview){emit('close');return}
 if(busy.value)return
 const original=inbox.currentPopup
 busy.value=true
 try{await inbox.dismissPopup();if(inbox.currentPopup===original&&inbox.currentPopup?.type==='announcement')app.showError(t('common.unknownError'))}finally{busy.value=false}
}
</script>
<template>
  <BaseDialog :show="!!displayedAnnouncement" :title="displayedAnnouncement?.title || ''" width="wide" :z-index="120" :close-on-escape="preview" :close-on-click-outside="preview" :show-close-button="preview" @close="dismiss">
    <div v-if="displayedAnnouncement" class="space-y-4" data-testid="announcement-popup"><p class="text-caption-1-regular text-text-secondary">{{ formatRelativeWithDateTime(displayedAnnouncement.created_at) }}</p><div class="markdown-body break-words" v-html="content" /></div>
    <template #footer><BoardButton :loading="busy" data-testid="dismiss-announcement" @click="dismiss">{{ t(preview ? 'common.close' : 'announcements.markRead') }}</BoardButton></template>
  </BaseDialog>
</template>
