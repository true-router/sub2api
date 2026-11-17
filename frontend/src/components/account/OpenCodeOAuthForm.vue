<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BoardButton from '@/components/boardui/Button.vue'
import BoardInput from '@/components/boardui/Input.vue'
import Select from '@/components/common/Select.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import { authorizeOpenCode, pollOpenCode, importOpenCode, type OpenCodeDevice, type OpenCodeOAuthCredentials } from '@/api/admin/opencode'

const props = defineProps<{ proxyId: number | null }>()
const emit = defineEmits<{ credentials: [value: OpenCodeOAuthCredentials | null] }>()
const { t } = useI18n()
const mode = ref('device')
const raw = ref('')
const device = ref<OpenCodeDevice | null>(null)
const busy = ref(false)
const error = ref('')
const organization = ref('')
let timer: ReturnType<typeof setTimeout> | undefined
let controller: AbortController | undefined
let generation = 0
const key = 'admin.accounts.opencodeGo.oauth.'

function cancel() {
  generation++
  controller?.abort()
  if (timer) clearTimeout(timer)
  timer = undefined
  busy.value = false
  device.value = null
}
function reset() {
  cancel()
  raw.value = ''
  error.value = ''
  organization.value = ''
  emit('credentials', null)
}
function fail(cause: unknown) {
  error.value = extractApiErrorMessage(cause, t(key + 'failed'))
  busy.value = false
}
function accept(credentials: OpenCodeOAuthCredentials) {
  busy.value = false
  device.value = null
  raw.value = ''
  organization.value = credentials.org_name || credentials.org_id
  emit('credentials', credentials)
}
async function start() {
  reset()
  busy.value = true
  const current = generation
  controller = new AbortController()
  const signal = controller.signal
  try {
    const authorization = await authorizeOpenCode(props.proxyId, signal)
    if (current !== generation) return
    device.value = authorization
    const deadline = Date.now() + authorization.expires_in * 1000
    let interval = authorization.interval * 1000
    const poll = async () => {
      if (current !== generation) return
      if (Date.now() >= deadline) {
        error.value = t(key + 'expired')
        busy.value = false
        device.value = null
        return
      }
      try {
        const result = await pollOpenCode(authorization.device_code, props.proxyId, signal)
        if (current !== generation) return
        if (result.status === 'authorized' && result.credentials) {
          accept(result.credentials)
          return
        }
        if (result.status === 'slow_down') interval += 5000
        timer = setTimeout(poll, interval)
      } catch (cause) {
        if (current === generation) fail(cause)
      }
    }
    timer = setTimeout(poll, interval)
  } catch (cause) {
    if (current === generation) fail(cause)
  }
}
async function importCredential() {
  cancel()
  error.value = ''
  organization.value = ''
  emit('credentials', null)
  let credential: unknown
  try { credential = JSON.parse(raw.value) } catch { error.value = t(key + 'invalidJSON'); return }
  busy.value = true
  const current = generation
  controller = new AbortController()
  try {
    const result = await importOpenCode(credential, controller.signal)
    if (current === generation && result.credentials) accept(result.credentials)
  } catch (cause) { if (current === generation) fail(cause) }
}
watch([mode, () => props.proxyId], reset)
onBeforeUnmount(cancel)
</script>

<template>
  <div class="space-y-3 rounded-3xl border border-border-button-default p-4" data-testid="opencode-oauth-form">
    <p class="text-body-regular text-text-secondary">{{ t(key + 'description') }}</p>
    <Select v-model="mode" :options="[{ value: 'device', label: t(key + 'device') }, { value: 'import', label: t(key + 'import') }]" :aria-label="t(key + 'method')" />
    <template v-if="mode === 'device'">
      <BoardButton :loading="busy" data-testid="opencode-oauth-start" @click="start">{{ t(key + 'start') }}</BoardButton>
      <div v-if="device" class="space-y-2">
        <p class="text-body-medium text-text-primary">{{ t(key + 'code') }}: <code>{{ device.user_code }}</code></p>
        <a :href="device.verification_uri_complete" target="_blank" rel="noopener noreferrer" class="text-body-medium text-text-accent">{{ t(key + 'open') }}</a>
        <p class="text-caption-1-regular text-text-secondary">{{ t(key + 'waiting') }}</p>
      </div>
    </template>
    <template v-else>
      <p class="text-caption-1-regular text-text-secondary">{{ t(key + 'importHint') }}</p>
      <BoardInput v-model="raw" type="password" :label="t(key + 'credential')" :revealable="false" autocomplete="off" data-testid="opencode-oauth-json" />
      <BoardButton :loading="busy" :disabled="!raw.trim()" @click="importCredential">{{ t(key + 'validate') }}</BoardButton>
    </template>
    <p v-if="organization" role="status" class="text-body-medium text-text-primary" data-testid="opencode-oauth-ready">{{ t(key + 'ready', { organization }) }}</p>
    <p v-if="error" role="alert" class="text-body-regular text-text-error-primary">{{ error }}</p>
  </div>
</template>
