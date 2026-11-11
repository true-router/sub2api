<script setup lang="ts">
import { cx } from '@/utils/cx'

defineOptions({ inheritAttrs: false })
withDefaults(defineProps<{ size?: 'sm' | 'md' }>(), { size: 'md' })
</script>

<template>
  <table
    v-bind="$attrs"
    :class="cx('bui-table w-full border-separate border-spacing-0 text-start', $attrs.class as string)"
    data-boardui="table"
    :data-size="size"
  >
    <slot />
  </table>
</template>

<style>
/* Vue port of the BoardUI Table registry styles. Separate borders keep sticky
   cells painted during scroll. The page owns the single scroll container. */
:where(.bui-table) > thead > tr > th {
  @apply whitespace-nowrap border-y border-separator-border bg-background-secondary-default px-3 py-2.5 text-start align-middle text-body-medium text-text-tertiary;
}

:where(.bui-table) > tbody > tr > td {
  @apply px-3 py-2.5 align-middle text-body-medium text-text-primary;
}

:where(.bui-table[data-size='sm']) > thead > tr > th,
:where(.bui-table[data-size='sm']) > tbody > tr > td {
  @apply px-2.5 py-1.5 text-body-2-medium;
}

:where(.bui-table) > tbody > tr:not(:last-child):not([aria-hidden='true']) > td {
  @apply border-b border-separator-border;
}

:where(.bui-table) > tbody > tr[data-row-id] {
  @apply h-16 transition-colors duration-150 motion-reduce:transition-none;
}

:where(.bui-table[data-size='sm']) > tbody > tr[data-row-id] {
  @apply h-12;
}

:where(.bui-table) > tbody > tr[data-row-id]:hover {
  @apply bg-background-primary-hover;
}

:where(.bui-table) > tbody > tr[data-selected] {
  @apply bg-background-secondary-default;
}
</style>
