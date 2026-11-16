<script setup lang="ts">
// Shared shell for the login and register pages.
// The shell fills exactly one viewport. It never shows a page scrollbar.
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import Icon from '@/components/icons/Icon.vue'

defineProps<{
  title: string
  subtitle?: string
  /** Gift headline. Pass it to show the gift card with this text. */
  giftTitle?: string
}>()

const { t } = useI18n()
const appStore = useAppStore()
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
// Theme state is shared with the landing page through the `dark` class and the `theme` storage key.
const isDark = ref(document.documentElement.classList.contains('dark'))
function toggleTheme() {
  isDark.value = !document.documentElement.classList.contains('dark')
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const points = computed(() => [t('lp.trust1'), t('lp.trust2'), t('lp.auth.point3')])
</script>

<template>
  <div class="auth" data-testid="auth-shell">
    <!-- Brand side (desktop only) -->
    <aside class="brand">
      <div class="aurora"><i class="b1"></i><i class="b2"></i></div>
      <router-link to="/" class="logo" :aria-label="siteName">
        <span class="logo-mark" :class="{ 'has-img': siteLogo }"><img v-if="siteLogo" :src="siteLogo" alt="" /><svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>{{ siteName }}
      </router-link>
      <div class="pitch">
        <h2>{{ t('lp.auth.pitch') }}<br /><span class="grad">{{ t('lp.auth.pitchAccent') }}</span></h2>
        <ul>
          <li v-for="p in points" :key="p"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="M5 12l5 5L20 7" /></svg>{{ p }}</li>
        </ul>
      </div>
      <div class="gift-card" data-testid="auth-gift">
        <div class="card"><small>{{ t('lp.giftLabel', { name: siteName }) }}</small><b class="num">P 5.00</b></div>
        <div>
          <h3>{{ giftTitle || t('lp.auth.giftNew') }}</h3>
          <p>{{ t('lp.auth.giftSub') }}</p>
        </div>
      </div>
    </aside>

    <!-- Form side -->
    <main class="pane">
      <div class="topbar">
        <router-link to="/" class="logo m-logo" :aria-label="siteName">
          <span class="logo-mark" :class="{ 'has-img': siteLogo }"><img v-if="siteLogo" :src="siteLogo" alt="" /><svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>{{ siteName }}
        </router-link>
        <router-link to="/" class="home-link">{{ t('lp.auth.backHome') }}</router-link>
        <button type="button" class="theme-btn" data-testid="auth-theme-toggle" role="switch" :aria-checked="isDark" :aria-label="t(isDark ? 'nav.lightMode' : 'nav.darkMode')" :title="t(isDark ? 'nav.lightMode' : 'nav.darkMode')" @click="toggleTheme"><Icon :name="isDark ? 'moon' : 'sun'" size="sm" aria-hidden="true" /></button>
      </div>
      <div class="center">
        <div class="form-box">
          <div v-if="giftTitle" class="m-gift">
            <span class="m-card num">P 5.00</span><span>{{ giftTitle }}</span>
          </div>
          <h1>{{ title }}</h1>
          <p v-if="subtitle" class="sub">{{ subtitle }}</p>
          <div class="slot"><slot /></div>
          <div v-if="$slots.footer" class="foot"><slot name="footer" /></div>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.auth{--ink:rgb(var(--boardui-text-primary));--ink2:rgb(var(--boardui-text-primary));--ink3:rgb(var(--boardui-text-secondary));--line:rgb(var(--boardui-border-button-default));--blue:rgb(var(--boardui-text-accent));--bg:rgb(var(--boardui-background-full));--soft:rgb(var(--boardui-background-secondary-default));color-scheme:light dark;
  position:relative;display:grid;grid-template-columns:minmax(0,5fr) minmax(0,6fr);height:100dvh;overflow:hidden;color:var(--ink);background:var(--bg);-webkit-font-smoothing:antialiased;
  font-family:-apple-system,BlinkMacSystemFont,"SF Pro Display","PingFang SC","Hiragino Sans GB","Segoe UI",sans-serif}
.num{font-variant-numeric:tabular-nums}
.logo{display:flex;gap:10px;align-items:center;font-weight:650;font-size:17px;letter-spacing:-.01em;width:max-content}
.logo-mark img{width:100%;height:100%;object-fit:contain;border-radius:inherit}
.logo-mark.has-img{background:none;box-shadow:none}
.logo-mark{width:30px;height:30px;border-radius:9px;background:linear-gradient(135deg,#3b82f6,#06b6d4);box-shadow:inset 0 1px 0 rgba(255,255,255,.35),0 4px 12px -2px rgba(59,130,246,.5);display:flex;align-items:center;justify-content:center}

/* brand side */
.brand{position:relative;display:flex;flex-direction:column;justify-content:space-between;gap:24px;padding:40px 48px;overflow:hidden;background:var(--soft)}
.brand:before{content:"";position:absolute;inset:0;background-image:radial-gradient(rgb(var(--boardui-text-primary) / .09) 1px,transparent 1px);background-size:22px 22px;-webkit-mask-image:radial-gradient(ellipse 80% 70% at 30% 40%,#000 20%,transparent 75%);mask-image:radial-gradient(ellipse 80% 70% at 30% 40%,#000 20%,transparent 75%)}
.aurora{position:absolute;inset:0;pointer-events:none}
.aurora i{position:absolute;border-radius:50%;filter:blur(80px)}
.b1{width:420px;height:360px;left:-100px;top:-60px;background:rgb(var(--boardui-background-accent-subtle));opacity:.7}
.b2{width:380px;height:340px;right:-120px;bottom:-60px;background:rgb(var(--boardui-background-warning-subtle));opacity:.6}
.brand>*:not(.aurora){position:relative}
.pitch h2{font-size:clamp(30px,3.2vw,44px);line-height:1.1;letter-spacing:-.04em;font-weight:750}
.grad{background:linear-gradient(100deg,#1d4ed8 10%,#06b6d4 55%,#06b6d4 95%);-webkit-background-clip:text;background-clip:text;color:transparent}
.pitch ul{list-style:none;display:flex;flex-direction:column;gap:10px;margin-top:22px;font-size:14.5px;color:var(--ink3)}
.pitch li{display:flex;gap:8px;align-items:center}
.pitch li svg{color:#10b981;flex-shrink:0}
.gift-card{display:flex;gap:16px;align-items:center;padding:16px 18px;border-radius:20px;border:1px solid var(--line);background:rgb(var(--boardui-background-warning-subtle));box-shadow:0 24px 40px -28px rgba(180,120,40,.45)}
.card{position:relative;width:96px;height:62px;flex-shrink:0;border-radius:10px;color:#fff;padding:8px 10px;overflow:hidden;transform:rotate(-4deg);
  background:radial-gradient(80px 40px at 90% 0,rgba(255,255,255,.3),transparent 60%),linear-gradient(135deg,#3b82f6,#0891b2 70%);box-shadow:inset 0 0 0 1px rgba(253,230,138,.55),0 10px 18px -10px rgba(30,58,138,.6)}
.card:before{content:"";position:absolute;top:0;bottom:0;left:78%;width:7px;background:linear-gradient(90deg,#f3c969,#fde68a,#f3c969)}
.card small{position:relative;font-size:8px;opacity:.85}
.card b{position:relative;display:block;font-size:20px;font-weight:700;letter-spacing:-.04em;margin-top:6px}
.gift-card h3{font-size:15px;font-weight:650}
.gift-card p{font-size:13px;color:var(--ink3);margin-top:3px}

/* form side */
.pane{position:relative;display:flex;flex-direction:column;min-width:0;height:100dvh;overflow-y:auto;scrollbar-width:none;background:var(--bg)}
.pane::-webkit-scrollbar{display:none}
.topbar{display:flex;justify-content:space-between;align-items:center;padding:24px 32px 0;flex-shrink:0}
.m-logo{display:none}
.theme-btn{display:inline-flex;align-items:center;justify-content:center;width:36px;height:36px;margin-left:8px;border-radius:11px;color:var(--ink3);cursor:pointer;transition:background .15s,color .15s}
.theme-btn:hover{background:var(--soft);color:var(--ink)}
.theme-btn:focus-visible{outline:2px solid var(--blue);outline-offset:2px}
.home-link{margin-left:auto;font-size:13.5px;color:var(--ink3);transition:color .15s}
.home-link:hover{color:var(--ink)}
.center{flex:1;display:flex;align-items:center;justify-content:center;padding:16px 32px 32px}
.form-box{width:100%;max-width:400px}
.form-box h1{font-size:28px;font-weight:700;letter-spacing:-.03em;line-height:1.15}
.sub{margin-top:6px;font-size:14.5px;color:var(--ink3)}
.slot{margin-top:22px}
.slot :deep(form){gap:14px}
.slot :deep([data-boardui=input]){gap:6px}
.slot :deep([data-boardui=input] > label){font-size:13px;font-weight:550;color:var(--ink2)}
.slot :deep([data-boardui=input] > div){min-height:44px;padding:0 14px;border-radius:12px;border:1px solid var(--line);box-shadow:none}
.slot :deep([data-boardui=input] > div:focus-within){border-color:var(--blue);box-shadow:0 0 0 4px rgb(var(--boardui-text-accent) / .15)}
.slot :deep([data-boardui=input] > div:has(input[aria-invalid=true])){border-color:#ef4444}
.slot :deep([data-boardui=input] input){font-size:15px}
.slot :deep(button[type=submit]){margin-top:4px;height:48px;border:0;border-radius:14px;font-size:15.5px;font-weight:600;color:#fff;
  background:linear-gradient(180deg,#3b82f6,#2563eb);box-shadow:inset 0 1px 0 rgba(255,255,255,.25),0 0 0 1px #1d4ed8,0 8px 24px -6px rgba(37,99,235,.55)}
.foot{margin-top:18px;text-align:center;font-size:13.5px;color:var(--ink3)}
.foot :deep(a){color:var(--blue);font-weight:550}
.m-gift{display:none}

/* short viewports: tighten vertical rhythm so nothing scrolls */
@media (max-height:720px){
  .center{padding-top:4px;padding-bottom:16px}.topbar{padding-top:16px}
  .form-box h1{font-size:24px}.slot{margin-top:14px}.slot :deep(form){gap:10px}
  .slot :deep([data-boardui=input] > div){min-height:40px}.slot :deep(button[type=submit]){height:44px}.foot{margin-top:12px}
}
@media (max-width:900px){
  .auth{grid-template-columns:1fr}
  .brand{display:none}
  .m-logo{display:flex}
  .topbar{padding:18px 20px 0}.center{padding:12px 20px 24px}
  .m-gift{display:flex;align-items:center;gap:10px;margin-bottom:16px;padding:8px 12px;border-radius:12px;border:1px solid var(--line);background:rgb(var(--boardui-background-warning-subtle));font-size:13px;color:var(--ink3)}
  .m-card{padding:3px 8px;border-radius:7px;background:linear-gradient(135deg,#3b82f6,#0891b2);color:#fff;font-weight:700;font-size:13px}
}
</style>
