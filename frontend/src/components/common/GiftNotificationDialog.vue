<script setup lang="ts">
// Gift reveal. Follows .planning/2026-11-14-console-redesign/prototype.html (dialog.gift).
// The box opens only after the server acknowledges the claim.
import { computed, nextTick, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import BoardButton from '@/components/boardui/Button.vue'
import { useAnnouncementStore } from '@/stores/announcements'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import type { UserNotification } from '@/api/notifications'

const props = defineProps<{ notification: UserNotification }>()
const emit = defineEmits<{ close: [] }>()
const { t, locale } = useI18n()
const inbox = useAnnouncementStore()
const app = useAppStore()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

type Phase = 'closed' | 'shaking' | 'opened'
const CONFETTI = ['oklch(80% .15 80)', 'oklch(62% .21 259)', 'oklch(72% .18 150)', 'oklch(70% .2 20)', 'oklch(88% .14 90)']

const dialogRef = ref<HTMLDialogElement | null>(null)
const doneRef = ref<InstanceType<typeof BoardButton> | null>(null)
const phase = ref<Phase>(props.notification.read_at ? 'opened' : 'closed')
const busy = ref(false)
const error = ref(false)
const claimedNow = ref(false)
const confetti = ref<Record<string, string>[]>([])
const target = props.notification.gift_amount ?? 0
const shown = ref(props.notification.read_at ? target : 0)

const money = (value: number, exact = false) => new Intl.NumberFormat(locale.value, { style: 'currency', currency: 'USD', minimumFractionDigits: 2, maximumFractionDigits: exact ? 8 : 2 }).format(value)
const amount = computed(() => money(target, true))
const shownAmount = computed(() => (shown.value === target ? amount.value : money(shown.value)))
const eyebrow = computed(() => t(props.notification.gift_source === 'initial' ? 'giftNotice.welcomeEyebrow' : 'giftNotice.eyebrow'))
const reduceMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches
const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

function countUp() {
  if (reduceMotion()) { shown.value = target; return }
  const t0 = performance.now()
  const step = (now: number) => {
    const p = Math.min(1, (now - t0) / 900)
    shown.value = p < 1 ? target * (1 - Math.pow(1 - p, 3)) : target
    if (p < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}

function burstConfetti() {
  if (reduceMotion()) return
  confetti.value = Array.from({ length: 36 }, (_, i) => {
    const a = (Math.PI * 2 * i) / 36 + Math.random() * 0.3
    const d = 90 + Math.random() * 120
    return { '--c': CONFETTI[i % CONFETTI.length], '--x': `${(Math.cos(a) * d).toFixed(0)}px`, '--y': `${(Math.sin(a) * d - 60).toFixed(0)}px`, '--r': `${(Math.random() * 720 - 360).toFixed(0)}deg`, '--t': `${(1.1 + Math.random() * 0.7).toFixed(2)}s` }
  })
}

async function claim() {
  if (busy.value || phase.value !== 'closed') return
  busy.value = true
  error.value = false
  if (!reduceMotion()) phase.value = 'shaking'
  try {
    const [ok] = await Promise.all([inbox.markAsRead(props.notification.id, 'gift'), reduceMotion() ? null : wait(500)])
    if (!ok) { phase.value = 'closed'; error.value = true; return }
    phase.value = 'opened'
    claimedNow.value = true
    countUp()
    burstConfetti()
    void auth.refreshUser().catch(() => {})
    await nextTick()
    ;(doneRef.value?.$el as HTMLElement | undefined)?.focus()
  } finally {
    busy.value = false
  }
}

function close(start = false) {
  if (busy.value) return
  if (claimedNow.value) app.showSuccess(t('giftNotice.landed', { amount: amount.value }))
  if (start && route.path !== '/dashboard') void router.push('/dashboard')
  emit('close')
}

function onCancel(event: Event) {
  event.preventDefault()
  close()
}

onMounted(() => {
  dialogRef.value?.showModal()
  void auth.refreshUser().catch(() => {})
})
</script>

<template>
  <dialog ref="dialogRef" class="gift" :data-phase="phase" aria-labelledby="gift-title" aria-describedby="gift-desc" data-testid="gift-notification" :data-gift-id="notification.id" @cancel="onCancel">
    <div class="gift-card border border-border-button-default bg-background-primary-default text-center">
      <div class="gift-confetti" aria-hidden="true"><i v-for="(piece, i) in confetti" :key="i" :style="piece" /></div>
      <div class="gift-stage">
        <div class="gift-rays" aria-hidden="true" />
        <button type="button" class="gift-box" :aria-label="t('giftNotice.openAria')" :disabled="busy || phase === 'opened'" :tabindex="phase === 'opened' ? -1 : 0" @click="claim">
          <span class="gift-lid"><span class="gift-bow" /></span><span class="gift-base" />
        </button>
        <div class="gift-amount text-text-primary" aria-hidden="true">{{ shownAmount }}</div>
      </div>
      <div v-if="phase !== 'opened'">
        <div class="gift-eyebrow text-caption-1-semibold text-text-accent">{{ eyebrow }}</div>
        <h2 id="gift-title" class="gift-title text-text-primary">{{ t('giftNotice.title') }}</h2>
        <p id="gift-desc" class="gift-desc text-body-regular text-text-secondary">{{ t('giftNotice.closedDesc') }}</p>
        <p v-if="error" role="alert" class="mt-3 text-body-regular text-text-error-primary">{{ t('giftNotice.failed') }}</p>
        <div class="gift-actions">
          <BoardButton class="w-full" :loading="busy" data-testid="claim-gift" autofocus @click="claim">{{ t('giftNotice.claim') }}</BoardButton>
          <BoardButton variant="ghost" class="w-full" :disabled="busy" @click="close()">{{ t('giftNotice.later') }}</BoardButton>
        </div>
      </div>
      <div v-else>
        <div class="gift-eyebrow text-caption-1-semibold text-text-accent">{{ t('giftNotice.unlockedEyebrow') }}</div>
        <h2 id="gift-title" class="gift-title text-text-primary">{{ t('giftNotice.claimedAmount', { amount }) }}</h2>
        <p id="gift-desc" class="gift-desc text-body-regular text-text-secondary">{{ t('giftNotice.openedDesc') }}</p>
        <div class="gift-actions">
          <BoardButton ref="doneRef" class="w-full" data-testid="gift-done" autofocus @click="close(true)">{{ t('giftNotice.done') }}</BoardButton>
          <BoardButton variant="ghost" class="w-full" @click="close()">{{ t('giftNotice.close') }}</BoardButton>
        </div>
      </div>
    </div>
  </dialog>
</template>

<style>
html:has(dialog.gift[open]) { overflow: hidden; }
</style>

<style scoped>
dialog.gift { margin: auto; padding: 0; border: 0; background: none; overflow: visible; width: min(420px, calc(100vw - 32px)); max-width: none; color: inherit; }
dialog.gift::backdrop { background: rgb(10 10 10 / .6); backdrop-filter: blur(6px); animation: gift-fade .3s ease-out; }
.gift-card { position: relative; overflow: hidden; padding: 36px 28px 28px; border-radius: 28px; box-shadow: 0 24px 64px rgb(0 0 0 / .3); animation: gift-pop .5s cubic-bezier(.2, 1.4, .4, 1); }
.gift-stage { position: relative; height: 190px; display: grid; place-items: center; margin-bottom: 16px; }
.gift-rays { position: absolute; left: 50%; top: 50%; width: 360px; height: 360px; margin: -180px 0 0 -180px; border-radius: 50%; background: radial-gradient(circle, oklch(85% .15 85 / .55), transparent 45%), repeating-conic-gradient(oklch(62% .21 259 / .16) 0 9deg, transparent 9deg 18deg); -webkit-mask: radial-gradient(circle, #000 25%, transparent 68%); mask: radial-gradient(circle, #000 25%, transparent 68%); opacity: 0; transition: opacity .6s .1s; pointer-events: none; }
.gift-box { position: relative; width: 128px; height: 120px; padding: 0; border: 0; background: none; cursor: pointer; border-radius: 16px; animation: gift-wobble 2.8s ease-in-out infinite; }
.gift-box:focus-visible { outline: 2px solid oklch(62% .21 259); outline-offset: 6px; }
.gift-lid, .gift-base { position: absolute; display: block; }
.gift-lid { z-index: 2; left: 0; right: 0; top: 24px; height: 30px; border-radius: 8px; background: linear-gradient(135deg, oklch(68% .19 258), oklch(56% .23 263)); box-shadow: 0 4px 0 rgb(0 0 0 / .12); }
.gift-base { left: 9px; right: 9px; top: 52px; bottom: 0; border-radius: 0 0 14px 14px; background: linear-gradient(135deg, oklch(62% .21 259), oklch(50% .24 264)); }
.gift-lid::before, .gift-base::before { content: ""; position: absolute; top: 0; bottom: 0; left: 50%; width: 20px; margin-left: -10px; background: linear-gradient(90deg, oklch(80% .15 80), oklch(88% .14 90), oklch(80% .15 80)); }
.gift-bow { position: absolute; left: 50%; top: -12px; width: 16px; height: 16px; margin-left: -8px; border-radius: 50%; background: oklch(82% .15 82); z-index: 1; }
.gift-bow::before, .gift-bow::after { content: ""; position: absolute; top: -14px; width: 24px; height: 22px; border: 7px solid oklch(84% .15 85); }
.gift-bow::before { right: 10px; border-radius: 60% 10% 60% 60%; transform: rotate(-18deg); }
.gift-bow::after { left: 10px; border-radius: 10% 60% 60% 60%; transform: rotate(18deg); }
.gift-amount { position: absolute; font-weight: 700; font-size: 48px; line-height: 1; letter-spacing: -.03em; opacity: 0; transform: translateY(24px) scale(.5); font-variant-numeric: tabular-nums; pointer-events: none; }
.gift-eyebrow { font-weight: 600; text-transform: uppercase; letter-spacing: .08em; margin-bottom: 6px; }
.gift-title { font-size: 22px; line-height: 30px; font-weight: 600; }
.gift-desc { margin: 8px auto 0; max-width: 320px; }
.gift-actions { display: flex; flex-direction: column; gap: 8px; margin-top: 24px; }
.gift-actions :deep(button) { height: 44px; border-radius: 12px; font-size: 15px; }
.gift-confetti { position: absolute; left: 50%; top: 120px; width: 0; height: 0; pointer-events: none; z-index: 3; }
.gift-confetti i { position: absolute; width: 8px; height: 12px; border-radius: 2px; background: var(--c); opacity: 0; animation: gift-confetti var(--t) cubic-bezier(.15, .7, .4, 1) forwards; }
.gift[data-phase='shaking'] .gift-box { animation: gift-shake .5s linear infinite; cursor: progress; }
.gift[data-phase='opened'] .gift-box { animation: none; pointer-events: none; cursor: default; }
.gift[data-phase='opened'] .gift-lid { transform: translate(36px, -110px) rotate(32deg); opacity: 0; transition: transform .7s cubic-bezier(.2, .8, .3, 1), opacity .4s .35s; }
.gift[data-phase='opened'] .gift-base { transform: translateY(40px) scale(.85); opacity: 0; transition: transform .5s .15s, opacity .4s .2s; }
.gift[data-phase='opened'] .gift-rays { opacity: 1; animation: gift-spin 18s linear infinite; }
.gift[data-phase='opened'] .gift-amount { opacity: 1; transform: none; transition: opacity .4s .2s, transform .6s cubic-bezier(.2, 1.5, .4, 1) .2s; }
@keyframes gift-fade { from { opacity: 0; } }
@keyframes gift-pop { from { opacity: 0; transform: translateY(24px) scale(.92); } }
@keyframes gift-wobble { 0%, 64%, 100% { transform: rotate(0); } 70% { transform: rotate(-7deg); } 76% { transform: rotate(6deg); } 82% { transform: rotate(-4deg); } 88% { transform: rotate(2deg); } }
@keyframes gift-shake { 0%, 100% { transform: none; } 20% { transform: translateX(-7px) rotate(-9deg); } 40% { transform: translateX(7px) rotate(9deg); } 60% { transform: translateX(-5px) rotate(-6deg) scale(1.06); } 80% { transform: translateX(5px) rotate(6deg) scale(1.1); } }
@keyframes gift-spin { to { transform: rotate(360deg); } }
@keyframes gift-confetti { 0% { opacity: 1; transform: translate(-50%, -50%) scale(.3); } 45% { opacity: 1; transform: translate(var(--x), var(--y)) rotate(calc(var(--r) * .5)); } 100% { opacity: 0; transform: translate(calc(var(--x) * 1.15), calc(var(--y) + 240px)) rotate(var(--r)); } }
@media (prefers-reduced-motion: reduce) {
  dialog.gift *, dialog.gift::backdrop { animation: none !important; transition: none !important; }
}
</style>
