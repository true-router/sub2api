<template>
  <div class="flex flex-wrap items-center gap-3">
    <slot name="before"></slot>
    <BoardButton type="submit" @click="$emit('refresh')" :disabled="loading" variant="secondary" size="medium">
      <Icon name="refresh" size="md" :class="[loading ? 'animate-spin' : '']" />
    </BoardButton>
    <slot name="after"></slot>
    <slot name="beforeCreate"></slot>
    <BoardButton type="submit" @click="$emit('create')" variant="primary" size="medium">{{ t('admin.accounts.createAccount') }}</BoardButton>
    <slot name="afterCreate"></slot>
  </div>
</template>

<script setup lang="ts">
import BoardButton from '@/components/boardui/Button.vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'

defineProps(['loading'])
defineEmits(['refresh', 'create'])

const { t } = useI18n()
</script>
