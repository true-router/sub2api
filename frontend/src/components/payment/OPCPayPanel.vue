<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import QRCode from 'qrcode'
import BoardButton from '@/components/boardui/Button.vue'
import BoardBadge from '@/components/boardui/Badge.vue'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import alipayIcon from '@/assets/icons/alipay.svg'
import { isMobileDevice } from '@/utils/device'
import { opcpayAPI, type OPCPayInfo, type OPCPayOperation } from '@/api/opcpay'
import { useAuthStore } from '@/stores/auth'

const emit = defineEmits<{ availability: [available: boolean] }>()
const auth = useAuthStore()
const info = ref<OPCPayInfo>()
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const message = ref('')
const selectedSku = ref('')
const operation = ref<OPCPayOperation>()
const link = ref('')
const qrImage = ref('')
const modalOpen = ref(false)
const isMobile = isMobileDevice()
type Action = { id: string; sku: string; phase: 'purchase' | 'consume' }
const action = ref<Action>()
const storageKey = () => `opcpay-action:${auth.user?.id}`
const cards = computed(() => info.value?.product.charging_options || [])
const selectedCard = computed(() => cards.value.find(card => card.sku_id === selectedSku.value))
const available = computed(() => info.value?.enabled && info.value.product.capability_status === 'ENABLED' && info.value.capabilities.provider_configured)
const unresolved = computed(() => !!action.value || !!info.value?.pending)
const canConfirm = computed(() => operation.value?.kind === 'purchase' && operation.value.status === 'READY')
function saveAction(value: Action) { action.value = value; sessionStorage.setItem(storageKey(), JSON.stringify(value)) }
function clearAction() { action.value = undefined; sessionStorage.removeItem(storageKey()) }
function safeURL(value?: string) {
  if (!value) return ''
  try { const url = new URL(value); return ['https:', 'alipays:'].includes(url.protocol) && !url.username && !url.password ? value : '' } catch { return '' }
}
async function refresh() {
  try {
    info.value = (await opcpayAPI.info()).data
    emit('availability', info.value.enabled && (info.value.product.charging_options.length > 0 || !!info.value.pending))
    if (info.value.pending) {
      operation.value = info.value.pending
      selectedSku.value = info.value.pending.sku_id
      if (!action.value) saveAction({ id: info.value.pending.action_id, sku: info.value.pending.sku_id, phase: info.value.pending.kind })
    }
  } catch (e) { error.value = (e as { message?: string }).message || '无法读取充值卡，请重试。'; emit('availability', false) }
  finally { loading.value = false }
}
async function accept(value: OPCPayOperation) {
  operation.value = value
  modalOpen.value = true
  if (value.kind === 'purchase' && value.status === 'READY') {
    link.value = safeURL(value.result?.purchase_url)
    const code = safeURL(value.result?.purchase_qr_code) || link.value
    qrImage.value = code ? await QRCode.toDataURL(code, { width: 224, margin: 2 }) : ''
    message.value = `购买入口已生成。完成支付宝购买后，点击“确认充值”，将核销 ${value.amount} 积分并增加 ${value.credit_usd} USD 余额。`
  } else if (value.status === 'COMPLETED') {
    message.value = `充值完成。已核销 ${value.amount} 积分，增加 ${value.credit_usd} USD 余额。`
    clearAction(); link.value = ''; qrImage.value = ''
    await auth.refreshUser()
  } else if (value.status === 'FAILED' || value.status === 'CLOSED') {
    message.value = value.kind === 'purchase' ? '购买入口已结束，可重新选择充值卡。' : '核销失败，未增加余额。'
    clearAction(); link.value = ''; qrImage.value = ''
  } else message.value = '充值结果尚未确认。请恢复原操作，不要重复购买。'
  await refresh()
}
async function run(phase: 'purchase' | 'consume') {
  if (busy.value) return
  if (!action.value && !selectedCard.value) return
  busy.value = true; error.value = ''
  try {
    const current = action.value || { id: crypto.randomUUID(), sku: selectedSku.value, phase }
    saveAction({ ...current, phase })
    await accept((await opcpayAPI.start(phase, current.id, current.sku)).data)
  } catch (e) {
    error.value = (e as { message?: string }).message || '请求未确认，请恢复原操作。'
    await refresh()
  } finally { busy.value = false }
}
async function restore() {
  if (action.value) { await run(action.value.phase); return }
  if (!operation.value || busy.value) return
  busy.value = true; error.value = ''
  try { await accept((await opcpayAPI.resume(operation.value.business_id)).data) }
  catch (e) { error.value = (e as { message?: string }).message || '恢复失败，请重试。' }
  finally { busy.value = false }
}
function closeModal() {
  if (busy.value || !modalOpen.value) return
  modalOpen.value = false
}
async function onModalClosed() {
  if (canConfirm.value && action.value?.phase === 'purchase') {
    if (!await finishPurchase()) modalOpen.value = true
  }
}
async function finishPurchase() {
  if (busy.value || action.value?.phase === 'consume') return false
  busy.value = true; error.value = ''
  try {
    if (operation.value?.kind === 'purchase') await opcpayAPI.close(operation.value.business_id)
    clearAction(); operation.value = undefined; link.value = ''; qrImage.value = ''; message.value = ''
    await refresh()
    return true
  } catch (e) {
    error.value = (e as { message?: string }).message || '无法结束购买入口。'
    return false
  }
  finally { busy.value = false }
}
onMounted(async () => {
  const saved = sessionStorage.getItem(storageKey())
  if (saved) {
    try {
      const old = JSON.parse(saved) as Action & { kind?: 'purchase' | 'consume' }
      if (old.id && old.sku) action.value = { id: old.id, sku: old.sku, phase: old.phase || old.kind || 'purchase' }
    } catch { error.value = '本地操作记录无效，请联系管理员检查业务编号。' }
  }
  await refresh()
  if (unresolved.value) await restore()
})
</script>

<template>
  <div>
  <section v-if="loading || info?.enabled || error" class="space-y-6 rounded-3xl border border-border-button-default bg-background-primary-default p-6" aria-labelledby="opcpay-title" data-testid="opcpay-panel">
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="flex items-start gap-3">
        <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-gradient-to-b from-background-accent-default to-background-accent-hover text-text-white shadow-sm"><Icon name="creditCard" size="md" /></span>
        <div class="space-y-1">
          <h2 id="opcpay-title" class="text-title-3-semibold text-text-primary">一人付充值卡</h2>
          <p class="text-body-regular text-text-secondary">选择充值卡，通过支付宝购买。每 1 积分可兑换 1 USD 余额。</p>
        </div>
      </div>
      <BoardButton variant="secondary" size="small" :loading="loading" :disabled="busy" @click="refresh">
        <template #leading><Icon name="refresh" size="sm" /></template>
        刷新充值卡和权益
      </BoardButton>
    </div>
    <p v-if="loading" role="status" class="text-body-regular text-text-secondary">正在读取充值卡…</p>
    <p v-if="error" role="alert" class="rounded-2xl bg-background-error-subtle px-4 py-3 text-body-regular text-text-error-primary">{{ error }}</p>
    <template v-if="info?.enabled">
      <p v-if="!available" class="text-body-regular text-text-secondary">一人付尚未开放充值卡购买。</p>
      <template v-else>
        <div class="flex flex-wrap items-center justify-between gap-3 rounded-2xl bg-background-secondary-default px-4 py-3">
          <div class="flex items-center gap-2">
            <span class="text-body-regular text-text-secondary">账号</span>
            <BoardBadge tone="neutral">{{ info.provider_name }}</BoardBadge>
          </div>
          <p class="text-body-regular text-text-secondary">可用积分 <span class="ml-1 text-title-3-semibold tabular-nums text-text-primary">{{ info.access?.remaining ?? '—' }}</span></p>
        </div>
        <p v-if="!cards.length" class="text-body-regular text-text-secondary">当前没有可用的积分充值卡。请管理员检查一人付套餐配置。</p>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3" role="group" aria-label="可选充值卡">
          <button
            v-for="card in cards"
            :key="card.sku_id"
            type="button"
            :aria-pressed="selectedSku === card.sku_id"
            :disabled="busy || unresolved"
            :data-selected="selectedSku === card.sku_id"
            data-testid="opcpay-card"
            :class="['group relative flex flex-col gap-4 rounded-3xl border p-5 text-left outline-none transition-colors duration-150 focus-visible:ring-2 focus-visible:ring-border-focus-ring disabled:cursor-not-allowed disabled:opacity-60',
              selectedSku === card.sku_id
                ? 'border-background-accent-default bg-background-accent-subtle ring-1 ring-background-accent-default'
                : 'border-border-button-default bg-background-primary-default hover:border-border-button-hover hover:bg-background-primary-hover']"
            @click="selectedSku = card.sku_id"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="truncate text-body-medium text-text-secondary">{{ card.display_name }}</span>
              <span :class="['flex h-5 w-5 shrink-0 items-center justify-center rounded-full border', selectedSku === card.sku_id ? 'border-transparent bg-background-accent-default text-text-white' : 'border-border-button-default text-transparent']"><Icon name="check" size="xs" /></span>
            </div>
            <p class="text-display-4-medium tabular-nums text-text-primary"><span class="mr-1 text-body-medium text-text-tertiary">{{ card.currency }}</span>{{ card.price }}</p>
            <div class="mt-auto flex flex-wrap gap-2">
              <BoardBadge tone="accent">{{ card.quota_amount }} 积分</BoardBadge>
              <BoardBadge tone="success">可充值 {{ card.quota_amount }} USD</BoardBadge>
            </div>
          </button>
        </div>
        <BoardButton v-if="!unresolved && selectedCard" class="h-12 w-full !rounded-xl text-body-semibold" :disabled="busy || !info.capabilities.capabilities.purchase_entry" :loading="busy" @click="run('purchase')">
          <template #leading><img :src="alipayIcon" alt="" aria-hidden="true" class="h-5 w-5" /></template>
          支付宝购买 · {{ selectedCard.currency }} {{ selectedCard.price }}
        </BoardButton>
      </template>
    </template>
  </section>
  <BaseDialog :show="modalOpen" title="支付宝购买" :description="selectedCard ? `${selectedCard.display_name} · ${selectedCard.currency} ${selectedCard.price}` : undefined" width="narrow" @close="closeModal" @after-leave="onModalClosed">
    <div class="flex flex-col items-center gap-4 text-center" data-testid="opcpay-modal">
      <template v-if="canConfirm">
        <template v-if="isMobile">
          <span class="flex h-16 w-16 items-center justify-center rounded-2xl bg-background-secondary-default"><img :src="alipayIcon" alt="" aria-hidden="true" class="h-10 w-10" /></span>
          <p class="text-body-regular text-text-secondary">是否打开支付宝完成支付？支付完成后返回本页，点击“我已购买，确认充值”。</p>
          <a v-if="link" :href="link" target="_blank" rel="noopener noreferrer" class="inline-flex h-12 w-full items-center justify-center gap-2 rounded-xl bg-gradient-to-b from-background-accent-default to-background-accent-hover text-body-semibold text-text-white shadow-sm"><img :src="alipayIcon" alt="" aria-hidden="true" class="h-5 w-5" />打开支付宝支付</a>
        </template>
        <template v-else>
          <div class="rounded-2xl border border-border-button-default bg-white p-2">
            <img v-if="qrImage" :src="qrImage" alt="支付宝购买入口二维码" width="224" height="224" class="h-56 w-56" />
          </div>
          <p class="text-body-regular text-text-secondary">请使用手机支付宝扫码完成购买。支付完成后，点击“我已购买，确认充值”。</p>
        </template>
        <p class="text-caption-1-regular text-text-tertiary">将核销 {{ operation?.amount }} 积分，增加 {{ operation?.credit_usd }} USD 余额。</p>
      </template>
      <p v-else-if="message" role="status" class="text-body-regular text-text-secondary">{{ message }}</p>
      <p v-if="error" role="alert" class="w-full rounded-2xl bg-background-error-subtle px-4 py-3 text-body-regular text-text-error-primary">{{ error }}</p>
      <p v-if="operation" class="break-all text-caption-1-medium text-text-tertiary">业务编号：{{ operation.business_id }}</p>
    </div>
    <template #footer>
      <BoardButton v-if="canConfirm" :loading="busy" :disabled="!modalOpen" @click="run('consume')">我已购买，确认充值</BoardButton>
      <BoardButton v-else variant="secondary" @click="closeModal">关闭</BoardButton>
    </template>
  </BaseDialog>
  </div>
</template>
