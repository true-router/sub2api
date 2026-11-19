<template>
  <ConsolePage :title="t('seatTerminals.title')">
    <template #actions>
      <BoardButton variant="secondary" :loading="loading" @click="refreshSeats">
        {{ t('seatTerminals.refresh') }}
      </BoardButton>
    </template>

    <p v-if="listError" role="alert" class="text-body-regular text-text-error-primary">
      {{ t('seatTerminals.listError') }}
    </p>
    <DataTable
      v-if="!listError || seats.length"
      :columns="columns"
      :data="seats"
      :loading="loading && !seats.length"
      :sticky-first-column="false"
      :sticky-actions-column="false"
      row-key="id"
      clickable-rows
      @row-click="openTerminal($event.id)"
    >
      <template #cell-session_id="{ value }">
        <bdi v-if="value" class="break-all font-mono">{{ value }}</bdi>
        <span v-else class="text-text-tertiary">{{ t('seatTerminals.sessionMissing') }}</span>
      </template>
      <template #cell-slot="{ value }">
        <bdi dir="ltr" class="font-mono text-text-secondary">{{ value }}</bdi>
      </template>
      <template #cell-state="{ row }">
        <BoardBadge :tone="seatStatusTone(row)" dot>{{ t(`seatTerminals.${row.state}`) }}</BoardBadge>
      </template>
      <template #cell-actions="{ row }">
        <BoardButton variant="secondary" size="small" @click.stop="openTerminal(row.id)">
          {{ t('seatTerminals.open') }}
        </BoardButton>
      </template>
      <template #empty>{{ t('seatTerminals.empty') }}</template>
    </DataTable>

    <BoardDrawer
      :show="!!selected"
      :title="t('seatTerminals.terminal')"
      :description="selectedSession"
      width="full"
      @close="closeTerminal"
    >
      <div class="space-y-3">
        <div class="flex items-center justify-between gap-2">
          <span aria-live="polite">
            <BoardBadge :tone="connectionTone[connectionState]" dot>{{ t(`seatTerminals.${connectionState}`) }}</BoardBadge>
          </span>
          <BoardButton v-if="connectionState === 'disconnected'" variant="secondary" @click="openTerminal(selected)">
            {{ t('seatTerminals.reconnect') }}
          </BoardButton>
        </div>
        <div class="relative overflow-hidden rounded-xl border border-border-button-default bg-background-secondary-default">
          <BoardScrollFade :scroll-top="terminalScrollTop" />
          <div class="terminal-viewport overflow-auto p-3" @scroll="terminalScrollTop = ($event.target as HTMLElement).scrollTop">
            <div ref="terminalHost" class="w-max min-w-full" :aria-label="t('seatTerminals.terminal')" />
          </div>
        </div>
      </div>
    </BoardDrawer>
  </ConsolePage>
</template>

<style scoped>
.terminal-viewport {
  max-block-size: 70dvh;
}
</style>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import BoardDrawer from '@/components/boardui/Drawer.vue'
import BoardScrollFade from '@/components/boardui/ScrollFade.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import ConsolePage from '@/components/user/console/ConsolePage.vue'
import { apiClient, buildApiUrl } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

type Seat = { id: string; slot: string; session_id?: string; state: 'active' | 'waiting_tools' | 'idle' }

const props = defineProps<{ admin?: boolean }>()
const { t } = useI18n()
const auth = useAuthStore()
const seats = ref<Seat[]>([])
const columns = computed<Column[]>(() => [
  { key: 'session_id', label: t('seatTerminals.session'), class: '!whitespace-normal min-w-48' },
  { key: 'slot', label: t('seatTerminals.slotColumn') },
  { key: 'state', label: t('seatTerminals.status') },
  { key: 'actions', label: t('seatTerminals.action'), class: 'text-right' },
])
const statusTone = { active: 'success', waiting_tools: 'warning', idle: 'neutral' } as const
const connectionTone = { connecting: 'neutral', live: 'success', disconnected: 'warning' } as const
const endpoint = computed(() => props.admin ? '/admin/seat-terminals' : '/seat-terminals')
const selected = ref('')
const selectedSession = computed(() => seats.value.find((seat) => seat.id === selected.value)?.session_id)
const terminalScrollTop = ref(0)
const loading = ref(true)
const listError = ref(false)
const connectionState = ref<'connecting' | 'live' | 'disconnected'>('disconnected')
const terminalHost = ref<HTMLDivElement>()
let socket: WebSocket | undefined
let terminal: Terminal | undefined
let listRequest: AbortController | undefined
let selection = 0

function seatStatusTone(seat: Seat) {
  return statusTone[seat.state]
}

async function refreshSeats() {
  listRequest?.abort()
  const request = new AbortController()
  listRequest = request
  loading.value = true
  listError.value = false
  try {
    const { data } = await apiClient.get<{ seats: Seat[] }>(endpoint.value, { signal: request.signal })
    if (request.signal.aborted) return
    seats.value = data.seats
    if (selected.value && !seats.value.some((seat) => seat.id === selected.value)) closeTerminal()
  } catch {
    if (!request.signal.aborted) listError.value = true
  } finally {
    if (listRequest === request) loading.value = false
  }
}

function closeTerminal() {
  selection++
  socket?.close()
  terminal?.dispose()
  socket = undefined
  terminal = undefined
  selected.value = ''
  terminalScrollTop.value = 0
  connectionState.value = 'disconnected'
}

async function openTerminal(id: string) {
  closeTerminal()
  const current = selection
  selected.value = id
  connectionState.value = 'connecting'
  await nextTick()
  if (current !== selection || !terminalHost.value || !auth.token) return
  const pathname = new URL(buildApiUrl(`${endpoint.value}/${id}/stream`), location.origin).pathname
  const url = new URL(pathname, location.origin)
  url.protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const viewer = new WebSocket(url, [props.admin ? 'sub2api-admin' : 'sub2api-terminal', `jwt.${auth.token}`])
  viewer.binaryType = 'arraybuffer'
  socket = viewer
  let rendered = 0
  viewer.onmessage = ({ data }) => {
    if (socket !== viewer) return
    if (typeof data === 'string' && !terminal && terminalHost.value) {
      const { columns, rows } = JSON.parse(data)
      terminal = new Terminal({ cols: columns, rows, disableStdin: true, scrollback: 0 })
      terminal.open(terminalHost.value)
      connectionState.value = 'live'
      return
    }
    if (!(data instanceof ArrayBuffer) || !terminal) return
    const chunk = new Uint8Array(data)
    terminal.write(chunk, () => {
      if (socket !== viewer || viewer.readyState !== WebSocket.OPEN) return
      rendered += chunk.byteLength
      viewer.send(String(rendered))
    })
  }
  viewer.onclose = () => {
    if (socket !== viewer) return
    socket = undefined
    connectionState.value = 'disconnected'
    void refreshSeats()
  }
}

watch(endpoint, () => {
  closeTerminal()
  seats.value = []
  void refreshSeats()
})
onMounted(refreshSeats)
onUnmounted(() => {
  listRequest?.abort()
  closeTerminal()
})
</script>
