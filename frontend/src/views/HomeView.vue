<template>
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe v-if="isHomeContentUrl" :src="homeContent.trim()" :title="siteName" class="h-screen w-full border-0" allowfullscreen />
    <!-- Custom HTML is an administrator-owned setting. -->
    <div v-else v-html="homeContent" />
  </div>

  <div v-else ref="root" class="landing" data-testid="landing-home">
    <div class="aurora"><div class="blob b1"></div><div class="blob b2"></div><div class="blob b3"></div></div>

    <!-- PROMO -->
    <router-link v-if="!isAuthenticated" :to="claimTo" class="promo" data-testid="home-promo">
      <span class="promo-in"><b>{{ t('lp.newUser') }}</b><span>{{ t('lp.pill') }}</span><svg class="arrow" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6" /></svg></span>
    </router-link>

    <!-- NAV -->
    <div class="navbar">
      <div class="nav">
        <a class="logo" href="#" :aria-label="siteName">
          <span class="logo-mark" :class="{ 'has-img': siteLogo }"><img v-if="siteLogo" :src="siteLogo" alt="" /><svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>{{ siteName }}
        </a>
        <nav class="nav-links" :aria-label="t('lp.navLabel')">
          <a href="#fidelity">{{ t('lp.fidelity') }}</a><a href="#pricing">{{ t('lp.pricing') }}</a><a href="#how">{{ t('lp.how') }}</a>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('lp.docs') }}</a>
          <router-link v-if="showModelPlazaEntry" to="/model-plaza">{{ t('lp.plaza') }}</router-link>
        </nav>
        <div class="nav-r">
          <button type="button" class="ib ib-lang" data-testid="home-locale-toggle" role="switch" :aria-checked="locale === 'en'" :aria-label="t('lp.switchLanguage')" :title="t('lp.switchLanguage')" :disabled="switchingLocale" @click="toggleLocale"><Icon name="globe" size="sm" aria-hidden="true" /><span>{{ locale === 'en' ? 'EN' : '中' }}</span></button>
          <button type="button" class="ib ib-theme" data-testid="home-theme-toggle" role="switch" :aria-checked="isDark" :aria-label="t(isDark ? 'nav.lightMode' : 'nav.darkMode')" :title="t(isDark ? 'nav.lightMode' : 'nav.darkMode')" @click="toggleTheme"><Icon :name="isDark ? 'moon' : 'sun'" size="sm" aria-hidden="true" /></button>
          <router-link v-if="!isAuthenticated" to="/login" class="btn btn-t btn-sm nav-login">{{ t('lp.login') }}</router-link>
          <router-link :to="claimTo" class="btn btn-p btn-sm" data-testid="home-primary-action"><span class="shine"></span>{{ claimText }}</router-link>
          <button type="button" class="ib ib-menu" data-testid="home-menu-toggle" :aria-expanded="menuOpen" aria-controls="home-menu" :aria-label="t('common.toggleMenu')" @click.stop="menuOpen = !menuOpen"><Icon name="menu" size="sm" aria-hidden="true" /></button>
        </div>
      </div>
      <nav v-if="menuOpen" id="home-menu" class="menu" data-testid="home-menu" :aria-label="t('lp.navLabel')" @click="menuOpen = false" @keydown.esc="menuOpen = false">
        <a href="#fidelity">{{ t('lp.fidelity') }}</a><a href="#pricing">{{ t('lp.pricing') }}</a><a href="#how">{{ t('lp.how') }}</a>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('lp.docs') }}</a>
        <router-link v-if="showModelPlazaEntry" to="/model-plaza">{{ t('lp.plaza') }}</router-link>
        <router-link v-if="!isAuthenticated" to="/login" data-testid="home-menu-login">{{ t('lp.login') }}</router-link>
        <div class="menu-foot" @click.stop>
          <button type="button" class="ib ib-lang" data-testid="home-menu-locale" role="switch" :aria-checked="locale === 'en'" :aria-label="t('lp.switchLanguage')" :disabled="switchingLocale" @click="toggleLocale"><Icon name="globe" size="sm" aria-hidden="true" /><span>{{ locale === 'en' ? 'EN' : '中' }}</span></button>
          <button type="button" class="ib ib-theme" data-testid="home-menu-theme" role="switch" :aria-checked="isDark" :aria-label="t(isDark ? 'nav.lightMode' : 'nav.darkMode')" @click="toggleTheme"><Icon :name="isDark ? 'moon' : 'sun'" size="sm" aria-hidden="true" /></button>
          <router-link :to="claimTo" class="btn btn-p btn-sm" data-testid="home-menu-primary" @click="menuOpen = false"><span class="shine"></span>{{ claimText }}</router-link>
        </div>
      </nav>
    </div>

    <!-- HERO -->
    <header class="hero wrap">
      <div class="hero-g">
        <div>
          <div class="rv"><span class="pill"><b style="display:inline-flex;gap:5px;align-items:center"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linejoin="round"><rect x="3" y="5" width="18" height="14" rx="2.5" /><path d="M3.5 6.5l8.5 6.5 8.5-6.5" /></svg>{{ t('lp.newUser') }}</b>{{ t('lp.pill') }}</span></div>
          <h1 class="rv d1">{{ t('lp.heroTitle') }}<br /><span class="grad">{{ t('lp.heroAccent') }}</span></h1>
          <p class="lead rv d2"><i18n-t keypath="lp.lead1" tag="span"><template #strong><strong>{{ t('lp.leadStrong') }}</strong></template></i18n-t><br />{{ t('lp.lead2') }}</p>
          <div class="ctas rv d3">
            <router-link :to="claimTo" class="btn btn-p"><span class="shine"></span>{{ claimText }}<svg class="arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M5 12h14M13 6l6 6-6 6" /></svg></router-link>
            <a class="btn btn-g" href="#pricing">{{ t('lp.viewPrices') }}</a>
          </div>
          <div class="trust rv d3">
            <span v-for="item in trust" :key="item"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="M5 12l5 5L20 7" /></svg>{{ item }}</span>
          </div>
          <div class="models rv d3"><div class="stack"><span v-for="k in ['claude', 'openai', 'gemini']" :key="k"><i class="li" style="width:18px;height:18px" v-html="icon(k)"></i></span></div>{{ t('lp.modelsNote') }}</div>
        </div>
        <div class="pstage rv d2">
          <div class="orbit"></div>
          <div class="ftile ft1"><i class="li" v-html="icon('claude')"></i></div>
          <div class="ftile ft2"><i class="li" v-html="icon('gemini')"></i></div>
          <div class="ftile ft3"><i class="li" v-html="icon('openai')"></i></div>
          <div class="stamp"><b>{{ t('lp.stampBig') }}</b><small>{{ t('lp.stampSmall') }}</small></div>
          <div class="pc"><div class="pc-in">
            <div class="pc-h">
              <div><b>{{ t('lp.priceCompare') }}</b><small>{{ t('lp.perMillion') }}</small></div>
              <div class="seg"><span :class="{ on: priceKind === 'in' }" @click="priceKind = 'in'">{{ t('lp.input') }}</span><span :class="{ on: priceKind === 'out' }" @click="priceKind = 'out'">{{ t('lp.output') }}</span></div>
            </div>
            <div>
              <div v-for="(m, k) in priceModels" :key="m.n" class="pr" :class="{ hot: k === hot }">
                <div class="m"><span class="mi"><i class="li" v-html="icon(m.i)"></i></span><div><b>{{ m.n }}</b><small>{{ m.c }}</small></div></div>
                <s class="num">{{ fmt(priceKind === 'in' ? m.a : m.b) }}</s>
                <div class="o num"><strong>{{ fmt((priceKind === 'in' ? m.a : m.b) / 10) }}</strong><small>{{ siteName }}</small></div>
              </div>
            </div>
            <div class="pc-f"><span>{{ t('lp.priceFoot') }}</span><b>-90%</b></div>
          </div></div>
        </div>
      </div>
    </header>

    <!-- LOGOS -->
    <div class="logos wrap rv">
      <p>{{ t('lp.toolsNote') }}</p>
      <div class="marq"><div class="marq-t"><span v-for="(t, k) in marquee" :key="k"><i class="li" style="width:20px;height:20px" v-html="icon(t[0])"></i>{{ t[1] }}</span></div></div>
    </div>

    <!-- FEATURES -->
    <section id="fidelity" class="sec">
      <div class="wrap">
        <div class="head rv"><span class="eyebrow">{{ t('lp.whyEyebrow', { name: siteName }) }}</span><h2>{{ t('lp.whyTitle') }}</h2><p class="sub">{{ t('lp.whySub') }}</p></div>
        <div class="bento">
          <div ref="fidCard" class="card span7 rv" :class="{ in: fidIn }" @pointermove="spot">
            <div class="glow"></div>
            <div class="ic"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3l7 3v6c0 4.5-3 7.5-7 9-4-1.5-7-4.5-7-9V6z" /><path d="M9 12l2 2 4-4" /></svg></div>
            <h3>{{ t('lp.fidTitle') }}</h3>
            <p class="d">{{ t('lp.fidDesc') }}</p>
            <div class="fid">
              <div class="gauge">
                <svg width="190" height="190" viewBox="0 0 190 190">
                  <defs><linearGradient id="gg" x1="0" x2="1"><stop offset="0" stop-color="var(--green)" /><stop offset="1" stop-color="var(--cyan)" /></linearGradient></defs>
                  <circle cx="95" cy="95" r="80" fill="none" stroke="var(--line2)" stroke-width="14" />
                  <circle class="p" cx="95" cy="95" r="80" fill="none" stroke="url(#gg)" stroke-width="14" stroke-linecap="round" stroke-dasharray="502.65" :stroke-dashoffset="fidIn ? 0 : 502.65" />
                </svg>
                <div class="val"><b class="num">{{ score }}%</b><small>{{ t('lp.fidScore') }}</small></div>
              </div>
              <div class="labs">
                <dl class="fidelity-metrics flex flex-col gap-2 text-body-regular" data-testid="fidelity-metrics">
                  <div v-for="metric in fidelityMetrics" :key="metric" class="flex items-center justify-between gap-4 rounded-xl border p-3">
                    <dt>{{ t(`lp.${metric}`) }}</dt><dd class="text-body-medium">{{ t('lp.fidPassed') }}</dd>
                  </div>
                  <div class="rounded-xl border p-3">
                    <dt>{{ t('lp.fidFamily') }}</dt><dd class="text-body-medium">{{ t('lp.fidFamilyResult') }}</dd>
                  </div>
                </dl>
                <a href="https://www.hvoyai.com/report/CJ26erY31J4" target="_blank" rel="noopener noreferrer" class="text-body-medium text-text-primary underline outline-none focus-visible:ring-2 focus-visible:ring-border-focus-ring" data-testid="fidelity-report">{{ t('lp.fidReport') }} ↗</a>
              </div>
            </div>
          </div>

          <div id="pricing" class="card span5 rv d1" @pointermove="spot">
            <div class="glow"></div>
            <div class="ic"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.6 13.4l-7.2 7.2a2 2 0 01-2.8 0L3 13V3h10l7.6 7.6a2 2 0 010 2.8z" /><circle cx="7.5" cy="7.5" r="1.5" /></svg></div>
            <h3>{{ t('lp.priceCardTitle') }}</h3>
            <p class="d">{{ t('lp.priceCardDesc') }}</p>
            <div class="calc">
              <div class="calc-l"><span>{{ t('lp.monthlyUsage') }}</span><b class="num">{{ tokens }}M tokens</b></div>
              <input v-model.number="tokens" type="range" min="1" max="500" :aria-label="t('lp.monthlyUsage')" :style="{ '--p': ((tokens - 1) / 499) * 100 + '%' }" />
              <div class="cmp">
                <div class="cmp-r off"><span>{{ t('lp.official') }}</span><div class="track"><div class="fill" style="width:100%"></div></div><span class="v num">{{ money(officialCost) }}</span></div>
                <div class="cmp-r us"><span>{{ siteName }}</span><div class="track"><div class="fill" style="width:10%"></div></div><span class="v num">{{ money(ourCost) }}</span></div>
              </div>
              <div class="save"><span>{{ t('lp.monthlySave') }}</span><b class="num">{{ money(officialCost - ourCost) }}</b></div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- HOW -->
    <section id="how" class="sec" style="padding-top:40px">
      <div class="wrap">
        <div class="head rv"><span class="eyebrow">{{ t('lp.howEyebrow') }}</span><h2>{{ t('lp.howTitle') }}</h2><p class="sub">{{ t('lp.howSub') }}</p></div>
        <div class="hw rv">
          <div class="hw-card">
            <div class="hw-vis"><div class="v-env">
              <div class="env-scale"><div class="env">
                <span class="env-shadow"></span>
                <span class="env-back"></span>
                <div class="env-card">
                  <div class="ec-top"><span>{{ t('lp.giftLabel', { name: siteName }) }}</span></div>
                  <div class="ec-amt num">$<span class="ec-v">?.??</span></div>
                  <div class="ec-bot"><span>{{ t('lp.giftCardSub') }}</span><i class="ec-chip"></i></div>
                  <span class="ec-stamp">{{ t('lp.claimedStamp') }}</span>
                </div>
                <span class="env-front"></span>
                <span class="env-flap"><i></i></span>
                <span class="seal"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.6" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>
              </div></div>
              <span class="env-tag"><svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linejoin="round"><rect x="3" y="5" width="18" height="14" rx="2.5" /><path d="M3.5 6.5l8.5 6.5 8.5-6.5" /></svg>{{ t('lp.openAfterSignup') }}</span>
            </div></div>
            <div class="hw-body"><div class="hw-k">STEP 01</div><h3>{{ t('lp.step1Title') }}</h3><p><i18n-t keypath="lp.step1Body" tag="span"><template #strong><b>{{ t('lp.step1Strong') }}</b></template></i18n-t></p></div>
          </div>
          <div class="hw-card">
            <div class="hw-vis"><div class="v-hub">
              <span class="lbl" style="top:10px">{{ t('lp.yourTools') }}</span><span class="lbl" style="bottom:10px">{{ t('lp.originalModels') }}</span>
              <svg><line x1="18%" y1="27%" x2="50%" y2="50%" /><line x1="50%" y1="27%" x2="50%" y2="50%" /><line x1="82%" y1="27%" x2="50%" y2="50%" /><line x1="50%" y1="50%" x2="18%" y2="73%" /><line x1="50%" y1="50%" x2="50%" y2="73%" /><line x1="50%" y1="50%" x2="82%" y2="73%" /></svg>
              <span v-for="n in nodes" :key="n[1] + n[2]" class="node" :style="{ left: n[2], top: n[3] }"><i class="li" v-html="icon(n[0])"></i>{{ n[1] }}</span>
              <div class="hub"><span class="logo-mark" :class="{ 'has-img': siteLogo }"><img v-if="siteLogo" :src="siteLogo" alt="" /><svg v-else width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.6" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>{{ siteName }}</div>
            </div></div>
            <div class="hw-body"><div class="hw-k">STEP 02</div><h3>{{ t('lp.step2Title') }}</h3><p><i18n-t keypath="lp.step2Body" tag="span"><template #strong><b>{{ t('lp.step2Strong') }}</b></template></i18n-t></p></div>
          </div>
          <div class="hw-card">
            <div class="hw-vis"><div class="v-use">
              <div class="t"><div><small>{{ t('lp.weekSpend') }}</small><strong class="num">$3.42</strong></div><s class="num">{{ t('lp.weekOfficial', { amount: '$34.20' }) }}</s></div>
              <div class="bars"><i v-for="(h, k) in bars" :key="k" :style="{ height: h + '%', transitionDelay: (k * 0.06).toFixed(2) + 's' }"></i></div>
              <div class="days"><span v-for="d in days" :key="d">{{ d }}</span></div>
            </div></div>
            <div class="hw-body"><div class="hw-k">STEP 03</div><h3>{{ t('lp.step3Title') }}</h3><p><i18n-t keypath="lp.step3Body" tag="span"><template #strong><b>{{ t('lp.step3Strong') }}</b></template></i18n-t></p></div>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA: gift -->
    <section class="wrap" style="padding-top:40px">
      <div class="gift-sec rv" :data-state="giftState">
        <div class="gift-stage">
          <span class="g-rays"></span>
          <span class="g-spark"><i v-for="(s, k) in sparks" :key="k" :style="s"></i></span>
          <button class="env-btn" :aria-label="t('lp.openGift')" @click="openGift">
            <div class="env">
              <span class="env-shadow"></span>
              <span class="env-back"></span>
              <div class="env-card">
                <div class="ec-top"><span>{{ t('lp.giftLabel', { name: siteName }) }}</span></div>
                <div class="ec-amt num">$<span class="ec-v">{{ giftAmount }}</span></div>
                <div class="ec-bot"><span>{{ t('lp.giftCardSub') }}</span><i class="ec-chip"></i></div>
                <span class="ec-stamp">{{ t('lp.claimedStamp') }}</span>
              </div>
              <span class="env-front"></span>
              <span class="env-flap"><i></i></span>
              <span class="seal"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.6" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>
            </div>
          </button>
        </div>
        <div class="g-copy">
          <h2>{{ gift.t }}</h2>
          <p class="sub">{{ gift.s }}</p>
          <div class="ctas" style="justify-content:center;margin-top:30px">
            <button class="btn btn-p" @click="giftAction"><span class="shine"></span>{{ gift.b }}<svg class="arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><path d="M5 12h14M13 6l6 6-6 6" /></svg></button>
          </div>
          <p class="g-note">{{ gift.n }}</p>
          <p class="g-sign">{{ t('lp.giftSign', { name: siteName }) }}</p>
        </div>
      </div>
    </section>

    <!-- FOOTER -->
    <footer>
      <div class="wrap foot">
        <div><h4>{{ t('lp.product') }}</h4><ul><li><a href="#pricing">{{ t('lp.pricing') }}</a></li><li><router-link v-if="showModelPlazaEntry" to="/model-plaza">{{ t('lp.plaza') }}</router-link></li><li><router-link to="/key-usage">{{ t('lp.usageQuery') }}</router-link></li></ul></div>
        <div><h4>{{ t('lp.developers') }}</h4><ul><li><a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('lp.docs') }}</a></li><li><a href="#how">{{ t('lp.quickStart') }}</a></li><li><a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">{{ t('lp.apiRef') }}</a></li></ul></div>
        <div><h4>{{ t('lp.company') }}</h4><ul><li><a>{{ t('lp.contact') }}</a></li><li><a>{{ t('lp.terms') }}</a></li><li><a>{{ t('lp.privacy') }}</a></li></ul></div>
        <div class="foot-brand">
          <a class="logo" href="#" :aria-label="siteName"><span class="logo-mark" :class="{ 'has-img': siteLogo }"><img v-if="siteLogo" :src="siteLogo" alt="" /><svg v-else width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#fff" stroke-width="2.4" stroke-linecap="round"><path d="M4 12h6l2-6 2 12 2-6h4" /></svg></span>{{ siteName }}</a>
          <p>{{ t('lp.footDesc') }}</p>
        </div>
      </div>
      <div class="foot-giant" aria-hidden="true">{{ siteName }}</div>
      <div class="wrap foot-b"><span>© {{ currentYear }} {{ siteName }}</span><span>{{ t('lp.madeFor') }}</span></div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import Icon from '@/components/icons/Icon.vue'
import { setLocale } from '@/i18n'
import { LOBE } from '@/assets/lobe-icons'

const router = useRouter()
const { t, locale } = useI18n()
const switchingLocale = ref(false)
const menuOpen = ref(false)
const closeMenu = () => { menuOpen.value = false }
const isDark = ref(document.documentElement.classList.contains('dark'))
async function toggleLocale() {
  if (switchingLocale.value) return
  switchingLocale.value = true
  try {
    await setLocale(locale.value === 'zh' ? 'en' : 'zh')
  } finally {
    switchingLocale.value = false
  }
}
function toggleTheme() {
  isDark.value = !document.documentElement.classList.contains('dark')
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
const authStore = useAuthStore()
const appStore = useAppStore()
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const isHomeContentUrl = computed(() => /^https?:\/\//.test(homeContent.value.trim()))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const dashboardPath = computed(() => (authStore.isAdmin ? '/admin/dashboard' : '/dashboard'))
const registerPath = computed(() => (appStore.cachedPublicSettings?.registration_enabled === false ? '/login' : '/register'))
const claimTo = computed(() => (isAuthenticated.value ? dashboardPath.value : registerPath.value))
const claimText = computed(() => t(isAuthenticated.value ? 'lp.dashboard' : 'lp.claim'))
const showModelPlazaEntry = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza) && (isAuthenticated.value || appStore.cachedPublicSettings?.model_plaza_require_auth !== true))
const currentYear = new Date().getFullYear()
const icon = (key: string) => LOBE[key] ?? ''
const reduce = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

const fidelityMetrics = ['fidKnowledge', 'fidPdf', 'fidStructured'] as const
const trust = computed(() => [t('lp.trust1'), t('lp.trust2'), t('lp.trust3')])
const tools: [string, string][] = [['claude', 'Claude Code'], ['openai', 'OpenAI SDK'], ['anthropic', 'Anthropic SDK'], ['cursor', 'Cursor'], ['openai', 'Codex CLI'], ['cline', 'Cline'], ['gemini', 'Gemini CLI'], ['claude', 'Claude Desktop']]
const marquee = [...tools, ...tools]
const nodes = [['claude', 'Claude Code', '18%', '27%'], ['cursor', 'Cursor', '50%', '27%'], ['openai', 'Codex CLI', '82%', '27%'], ['anthropic', 'Anthropic', '18%', '73%'], ['openai', 'OpenAI', '50%', '73%'], ['google', 'Google', '82%', '73%']]
const bars = [38, 62, 45, 80, 58, 92, 70]
const days = computed(() => [1, 2, 3, 4, 5, 6, 7].map((n) => t(`lp.d${n}`)))
// Example prices per million tokens. Replace with live data when a public pricing API exists.
const priceModels = [
  { n: 'Claude Opus', c: 'Anthropic', i: 'claude', a: 15, b: 75 },
  { n: 'Claude Sonnet', c: 'Anthropic', i: 'claude', a: 3, b: 15 },
  { n: 'GPT', c: 'OpenAI', i: 'openai', a: 1.25, b: 10 },
  { n: 'Gemini Pro', c: 'Google', i: 'gemini', a: 1.25, b: 10 },
]
const fmt = (v: number) => '$' + (+v.toFixed(3)).toString()
const money = (v: number) => '$' + Math.round(v).toLocaleString()

/* hero price rows */
const priceKind = ref<'in' | 'out'>('in')
const hot = ref(0)
let hotTimer: number | undefined

/* calculator: blended $9 per million tokens (example) */
const tokens = ref(50)
const officialCost = computed(() => tokens.value * 9)
const ourCost = computed(() => officialCost.value / 10)

/* gauge */
const root = ref<HTMLElement | null>(null)
const fidCard = ref<HTMLElement | null>(null)
const fidIn = ref(false)
const score = ref(0)
function runGauge() {
  if (reduce) { score.value = 100; return }
  const t0 = performance.now()
  const step = (t: number) => {
    const p = Math.min(1, (t - t0) / 1800)
    score.value = Math.round(100 * (1 - Math.pow(1 - p, 3)))
    if (p < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}
watch(fidIn, (on) => { if (on) runGauge() })

/* card spotlight */
function spot(e: PointerEvent) {
  const c = e.currentTarget as HTMLElement
  const r = c.getBoundingClientRect()
  c.style.setProperty('--mx', e.clientX - r.left + 'px')
  c.style.setProperty('--my', e.clientY - r.top + 'px')
}

/* gift CTA */
type GiftState = 'sealed' | 'open' | 'claimed'
const G: Record<GiftState, string> = { sealed: 'lp.giftSealed', open: 'lp.giftOpen', claimed: 'lp.giftClaimed' }
const giftState = ref<GiftState>(isAuthenticated.value ? 'claimed' : 'sealed')
const giftAmount = ref(giftState.value === 'claimed' ? '5.00' : '?.??')
const gift = computed(() => {
  const k = G[giftState.value]
  return { t: t(`${k}.t`), s: t(`${k}.s`), b: t(`${k}.b`), n: t(`${k}.n`) }
})
const sparks = ref<Record<string, string>[]>([])
watch(isAuthenticated, (authed) => {
  giftState.value = authed ? 'claimed' : 'sealed'
  giftAmount.value = authed ? '5.00' : '?.??'
})
function countUp() {
  if (reduce) { giftAmount.value = '5.00'; return }
  const t0 = performance.now() + 900
  const step = (t: number) => {
    const p = Math.max(0, Math.min(1, (t - t0) / 900))
    giftAmount.value = (5 * (1 - Math.pow(1 - p, 3))).toFixed(2)
    if (p < 1) requestAnimationFrame(step)
  }
  requestAnimationFrame(step)
}
function sparkle() {
  if (reduce) return
  const cols = ['#fbbf24', '#fde68a', '#f59e0b', '#93c5fd', '#ffffff']
  sparks.value = Array.from({ length: 22 }, (_, i) => {
    const a = Math.random() * Math.PI * 2
    const r = 90 + Math.random() * 170
    return {
      '--x': Math.cos(a) * r + 'px', '--y': Math.sin(a) * r * 0.7 - 30 + 'px', '--s': 4 + Math.random() * 5 + 'px',
      '--c': cols[i % cols.length], '--t': 900 + Math.random() * 700 + 'ms', '--d': 0.95 + Math.random() * 0.25 + 's',
    }
  })
}
function openGift() {
  if (giftState.value !== 'sealed') return
  giftState.value = 'open'
  giftAmount.value = '0.00'
  countUp()
  sparkle()
  navigator.vibrate?.(12)
  sessionStorage.setItem('gift', 'opened')
}
function giftAction() {
  if (giftState.value === 'sealed') openGift()
  else router.push(giftState.value === 'open' ? registerPath.value : dashboardPath.value)
}

/* reveal on scroll */
let io: IntersectionObserver | undefined
let prevScrollBehavior = ''
onMounted(() => {
  document.addEventListener('click', closeMenu)
  prevScrollBehavior = document.documentElement.style.scrollBehavior
  document.documentElement.style.scrollBehavior = 'smooth'
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()
  if (!reduce) hotTimer = window.setInterval(() => { hot.value = (hot.value + 1) % priceModels.length }, 2200)
})
// The landing root exists only when no custom home content is configured.
watch(root, (el) => {
  io?.disconnect()
  if (!el) return
  if (typeof IntersectionObserver === 'undefined') {
    // No observer support: show everything at once.
    el.querySelectorAll('.rv').forEach((n) => n.classList.add('in'))
    fidIn.value = true
    return
  }
  io = new IntersectionObserver((entries) => entries.forEach((e) => {
    if (!e.isIntersecting) return
    e.target.classList.add('in')
    if (e.target === fidCard.value) fidIn.value = true
    io?.unobserve(e.target)
  }), { threshold: 0.18 })
  el.querySelectorAll('.rv').forEach((n) => io?.observe(n))
}, { flush: 'post', immediate: true })
onBeforeUnmount(() => {
  document.removeEventListener('click', closeMenu)
  io?.disconnect()
  if (hotTimer) clearInterval(hotTimer)
  document.documentElement.style.scrollBehavior = prevScrollBehavior
})
</script>

<style scoped>
.landing{position:relative;display:flow-root;min-height:100dvh;overflow-x:clip;font-family:-apple-system,BlinkMacSystemFont,"SF Pro Display","PingFang SC","Hiragino Sans GB","Segoe UI",sans-serif;color:var(--ink);background:var(--bg);-webkit-font-smoothing:antialiased;
  color-scheme:light dark;
  --ink:rgb(var(--boardui-text-primary));--ink2:rgb(var(--boardui-text-primary));--ink3:rgb(var(--boardui-text-secondary));--ink4:rgb(var(--boardui-text-tertiary));
  --line:rgb(var(--boardui-border-button-default));--line2:rgb(var(--boardui-separator-border));--bg:rgb(var(--boardui-background-full));--soft:rgb(var(--boardui-background-secondary-default));
  --surface:rgb(var(--boardui-background-primary-default));--white:rgb(var(--boardui-text-white));
  --blue:rgb(var(--boardui-text-accent));--blue2:rgb(var(--boardui-background-accent-default));--indigo:rgb(var(--boardui-text-feature));--cyan:rgb(var(--boardui-chart-4));--green:rgb(var(--boardui-text-success));
  --r:24px;
  --shadow:0 1px 2px rgba(15,23,42,.04),0 8px 24px -8px rgba(15,23,42,.08),0 32px 64px -32px rgba(15,23,42,.18);
}
:where(.landing) *{box-sizing:border-box;margin:0;padding:0}
:where(.landing) a{color:inherit;text-decoration:none;cursor:pointer}
.mono{font-family:ui-monospace,"SF Mono","JetBrains Mono",Menlo,monospace}
.num{font-variant-numeric:tabular-nums}
.wrap{max-width:1180px;margin:0 auto;padding:0 28px}
:where(.landing) svg,:where(.landing) :deep(svg){display:block}

/* ---------- background ---------- */
.aurora{position:absolute;inset:0 0 auto 0;height:1100px;overflow:hidden;pointer-events:none;z-index:0}
.aurora:before{content:"";position:absolute;inset:0;background-image:radial-gradient(rgba(15,23,42,.09) 1px,transparent 1px);background-size:22px 22px;-webkit-mask-image:radial-gradient(ellipse 70% 55% at 50% 30%,#000 30%,transparent 75%);mask-image:radial-gradient(ellipse 70% 55% at 50% 30%,#000 30%,transparent 75%)}
.blob{position:absolute;border-radius:50%;filter:blur(80px);opacity:.55}
.b1{width:620px;height:420px;left:-120px;top:-80px;background:rgb(var(--boardui-background-accent-subtle));animation:drift 18s ease-in-out infinite}
.b2{width:560px;height:420px;right:-140px;top:40px;background:rgb(var(--boardui-background-feature-subtle));animation:drift 22s ease-in-out infinite reverse}
.b3{width:520px;height:360px;left:35%;top:320px;background:rgb(var(--boardui-background-info-subtle));opacity:.35;animation:drift 26s ease-in-out infinite}
@keyframes drift{50%{transform:translate(40px,30px) scale(1.06)}}

/* ---------- nav ---------- */
.navbar{position:sticky;top:0;z-index:40;background:rgb(var(--boardui-background-full) / .82);backdrop-filter:saturate(180%) blur(14px);-webkit-backdrop-filter:saturate(180%) blur(14px)}
.nav{display:flex;align-items:center;justify-content:space-between;gap:12px;max-width:1180px;height:64px;margin:0 auto;padding:0 28px}
.logo{display:flex;gap:10px;align-items:center;font-weight:650;font-size:16.5px;letter-spacing:-.01em}
.logo-mark img{width:100%;height:100%;object-fit:contain;border-radius:inherit}
.logo-mark.has-img{background:none;box-shadow:none}
.logo-mark{width:30px;height:30px;border-radius:9px;background:linear-gradient(135deg,#3b82f6,#6366f1);box-shadow:inset 0 1px 0 rgba(255,255,255,.35),0 4px 12px -2px rgba(59,130,246,.5);display:flex;align-items:center;justify-content:center}
.nav-links{display:flex;gap:4px;font-size:14px;color:var(--ink2)}
.nav-links a{padding:8px;border-radius:10px;transition:background .15s,color .15s}
.nav-links a:hover{background:var(--soft);color:var(--ink)}
.nav-r{display:flex;gap:8px;align-items:center}
.ib{display:inline-flex;align-items:center;justify-content:center;gap:4px;min-width:36px;height:36px;padding:0 8px;border-radius:11px;color:var(--ink3);font:600 12.5px/1 inherit;cursor:pointer;transition:background .15s,color .15s}
.ib:hover:not(:disabled){background:var(--soft);color:var(--ink)}
.ib:focus-visible{outline:2px solid var(--blue);outline-offset:2px}
.ib:disabled{cursor:progress;opacity:.7}
.ib-menu{display:none}
.menu{position:absolute;top:calc(100% + 4px);left:14px;right:14px;z-index:45;display:flex;flex-direction:column;gap:2px;padding:8px;border-radius:18px;background:rgb(var(--boardui-background-primary-default));border:1px solid var(--line);box-shadow:0 16px 40px -12px rgba(15,23,42,.25);font-size:15px}
.menu a{padding:12px 14px;border-radius:12px;color:var(--ink)}
.menu a:hover{background:var(--soft)}
.menu-foot{display:flex;align-items:center;gap:6px;margin-top:6px;padding:10px 6px 4px;border-top:1px solid var(--line2)}
.menu-foot .btn{margin-left:auto;color:#fff}
.menu{display:none}
.promo{position:relative;z-index:41;display:block;color:#fff;background:linear-gradient(90deg,#2563eb,#6366f1);font-size:13.5px}
.promo-in{display:flex;align-items:center;justify-content:center;gap:10px;min-height:36px;padding:6px 28px;text-align:center}
.promo b{padding:1px 8px;border-radius:999px;background:rgba(255,255,255,.22);font-size:12px;font-weight:600}
.promo:hover .arrow{transform:translateX(3px)}

/* ---------- buttons ---------- */
.btn{position:relative;display:inline-flex;align-items:center;gap:8px;height:46px;padding:0 20px;border-radius:14px;font-weight:600;font-size:15px;transition:transform .15s,box-shadow .2s,background .2s;white-space:nowrap}
.btn-sm{height:38px;padding:0 14px;font-size:13.5px;border-radius:11px}
.btn-p{color:#fff;background:linear-gradient(180deg,#3b82f6,#2563eb);box-shadow:inset 0 1px 0 rgba(255,255,255,.25),0 0 0 1px #1d4ed8,0 8px 24px -6px rgba(37,99,235,.55)}
.btn-p:hover{transform:translateY(-1px);box-shadow:inset 0 1px 0 rgba(255,255,255,.25),0 0 0 1px #1d4ed8,0 14px 32px -8px rgba(37,99,235,.7)}
.btn-p .shine{position:absolute;inset:0;border-radius:inherit;overflow:hidden}
.btn-p .shine:after{content:"";position:absolute;top:0;left:-60%;width:40%;height:100%;background:linear-gradient(90deg,transparent,rgba(255,255,255,.35),transparent);transform:skewX(-20deg);animation:shine 3.6s ease-in-out infinite}
@keyframes shine{0%,60%{left:-60%}100%{left:130%}}
.btn-g{background:var(--surface);color:var(--ink);box-shadow:0 0 0 1px var(--line),0 2px 6px -2px rgba(15,23,42,.08)}
.btn-g:hover{box-shadow:0 0 0 1px #d5dae4,0 6px 16px -6px rgba(15,23,42,.15)}
.btn-t{color:var(--ink2)}.btn-t:hover{color:var(--ink)}
.arrow{transition:transform .2s}.btn:hover .arrow{transform:translateX(3px)}

/* ---------- hero ---------- */
.hero{position:relative;z-index:1;padding-block:76px 40px;text-align:center}
.pill{display:inline-flex;align-items:center;gap:10px;padding:5px 14px 5px 5px;border-radius:999px;background:rgb(var(--boardui-background-primary-default) / .9);border:1px solid var(--line);font-size:13.5px;color:var(--ink2);box-shadow:0 2px 10px -4px rgba(15,23,42,.1)}
.pill b{display:inline-flex;align-items:center;gap:5px;padding:3px 10px;border-radius:999px;background:linear-gradient(135deg,#2563eb,#6366f1);color:#fff;font-size:12px;font-weight:600}
h1{font-size:76px;line-height:1.04;letter-spacing:-.045em;font-weight:750;margin:26px auto 22px;max-width:900px}
.grad{background:linear-gradient(100deg,var(--blue) 10%,var(--indigo) 50%,var(--cyan) 90%);-webkit-background-clip:text;background-clip:text;color:transparent;background-size:200% auto;animation:gradmove 8s linear infinite}
@keyframes gradmove{to{background-position:200% center}}
.lead{font-size:19.5px;line-height:1.65;color:var(--ink3);max-width:620px;margin:0 auto}
.lead strong{color:var(--ink2);font-weight:600}
.ctas{display:flex;gap:12px;justify-content:center;margin-top:36px;flex-wrap:wrap}
.trust{display:flex;gap:22px;justify-content:center;margin-top:24px;font-size:13px;color:var(--ink3);flex-wrap:wrap}
.trust span{display:inline-flex;gap:6px;align-items:center}
.trust svg{color:var(--green)}

/* ---------- hero split ---------- */
.hero{text-align:left;padding-block:72px 56px}
.hero-g{display:grid;grid-template-columns:1fr 1.02fr;gap:64px;align-items:center}
.hero h1{margin:24px 0 20px;font-size:68px;max-width:none}
.hero .lead{margin:0;max-width:540px}
.hero .ctas{justify-content:flex-start;margin:34px 0 22px}
.hero .trust{justify-content:flex-start;margin:0}
.models{display:flex;align-items:center;gap:12px;margin-top:30px;font-size:13px;color:var(--ink4)}
.models .stack{display:flex}
.models .stack span{width:34px;height:34px;border-radius:11px;background:var(--surface);border:1px solid var(--line);display:flex;align-items:center;justify-content:center;margin-left:-8px;box-shadow:0 4px 10px -4px rgba(15,23,42,.15)}
.models .stack span:first-child{margin-left:0}
.li{display:inline-flex;align-items:center;justify-content:center;line-height:0}
.li :deep(svg){width:100%;height:100%}

.pstage{position:relative}
.orbit{position:absolute;inset:-60px -40px;pointer-events:none}
.orbit:before,.orbit:after{content:"";position:absolute;inset:0;border-radius:50%;border:1px dashed rgba(99,102,241,.18)}
.orbit:after{inset:60px 80px;border-color:rgba(59,130,246,.14)}
.ftile{position:absolute;z-index:3;width:58px;height:58px;border-radius:18px;background:rgb(var(--boardui-background-primary-default) / .9);backdrop-filter:blur(10px);border:1px solid var(--line);box-shadow:var(--shadow);display:flex;align-items:center;justify-content:center;animation:bob 6s ease-in-out infinite}
.ftile .li{width:30px;height:30px}
.ft1{left:-86px;top:40px}
.ft2{left:-60px;bottom:-34px;animation-delay:-2s}
.ft3{right:-40px;bottom:-40px;animation-delay:-4s}
@keyframes bob{50%{transform:translateY(-8px)}}
.pc{position:relative;z-index:2;padding:10px;border-radius:24px;background:rgb(var(--boardui-background-primary-default) / .9);border:1px solid var(--line);box-shadow:var(--shadow),0 60px 120px -40px rgba(37,99,235,.28)}
.pc-in{background:var(--surface);border-radius:18px;border:1px solid var(--line2);overflow:hidden}
.pc-h{display:flex;justify-content:space-between;align-items:center;padding:16px 20px;border-bottom:1px solid var(--line2)}
.pc-h b{font-size:14px}.pc-h small{display:block;font-size:12px;color:var(--ink4);font-weight:400;margin-top:2px}
.seg{display:flex;gap:2px;padding:3px;background:var(--soft);border-radius:9px;border:1px solid var(--line2)}
.seg span{font-size:12px;padding:4px 10px;border-radius:7px;cursor:pointer;color:var(--ink3)}.seg span.on{background:var(--surface);box-shadow:0 1px 3px rgba(15,23,42,.1);color:var(--ink);font-weight:600}
.pr{display:grid;grid-template-columns:1fr auto auto;gap:18px;align-items:center;padding:15px 20px;border-bottom:1px solid var(--line2);transition:background .4s}
.pr:last-child{border-bottom:0}
.pr.hot{background:linear-gradient(90deg,rgb(var(--boardui-background-accent-subtle)),var(--surface) 80%)}
.pr .m{display:flex;gap:12px;align-items:center}
.pr .mi{width:38px;height:38px;border-radius:12px;background:var(--surface);border:1px solid var(--line);display:flex;align-items:center;justify-content:center;flex-shrink:0;transition:box-shadow .4s,border-color .4s}
.pr.hot .mi{border-color:#c7d2fe;box-shadow:0 0 0 4px rgba(99,102,241,.08)}
.pr .mi .li{width:22px;height:22px}
.pr .m b{display:block;font-size:15px}.pr .m small{font-size:12px;color:var(--ink4)}
.pr s{color:var(--ink4);font-size:14px}
.pr .o{text-align:right;min-width:72px}.pr .o strong{display:block;font-size:20px;color:var(--blue);letter-spacing:-.02em}.pr .o small{font-size:11px;color:var(--ink4)}
.pc-f{display:flex;justify-content:space-between;align-items:center;padding:14px 20px;background:rgb(var(--boardui-background-success-subtle));font-size:13.5px;color:var(--green);border-top:1px solid var(--line)}
.pc-f b{font-size:18px}
.stamp{position:absolute;z-index:4;right:160px;top:-50px;width:92px;height:92px;border-radius:50%;background:linear-gradient(135deg,#2563eb,#6366f1);color:#fff;display:flex;flex-direction:column;align-items:center;justify-content:center;transform:rotate(12deg);box-shadow:0 14px 30px -10px rgba(79,70,229,.7),inset 0 0 0 4px rgba(255,255,255,.25);animation:wob 5s ease-in-out infinite}
.stamp b{font-size:28px;line-height:1;letter-spacing:-.03em}.stamp small{font-size:11px;opacity:.85;margin-top:2px}
@keyframes wob{50%{transform:rotate(4deg) scale(1.04)}}

/* ---------- logos ---------- */
.logos{position:relative;z-index:1;padding:72px 0 24px;text-align:center}
.logos p{font-size:13px;color:var(--ink4);margin-bottom:22px}
.marq{overflow:hidden;-webkit-mask-image:linear-gradient(90deg,transparent,#000 15%,#000 85%,transparent);mask-image:linear-gradient(90deg,transparent,#000 15%,#000 85%,transparent)}
.marq-t{display:flex;gap:56px;width:max-content;animation:marq 32s linear infinite}
.marq-t span{font-size:19px;font-weight:650;color:var(--ink4);letter-spacing:-.01em;display:inline-flex;gap:8px;align-items:center;white-space:nowrap}
@keyframes marq{to{transform:translateX(-50%)}}

/* ---------- sections ---------- */
section.sec{position:relative;padding:96px 0}
.head{text-align:center;max-width:680px;margin:0 auto 52px}
.eyebrow{display:inline-flex;align-items:center;gap:8px;font-size:13px;font-weight:600;color:var(--blue);background:rgb(var(--boardui-background-accent-subtle));padding:5px 12px;border-radius:999px}
h2{font-size:48px;letter-spacing:-.035em;line-height:1.1;margin:18px 0 16px;font-weight:720}
.sub{color:var(--ink3);font-size:18px;line-height:1.7}

.bento{display:grid;grid-template-columns:repeat(12,1fr);gap:18px}
.card{position:relative;background:var(--surface);border:1px solid var(--line);border-radius:var(--r);padding:32px;overflow:hidden;transition:box-shadow .3s,transform .3s,border-color .3s}
.card:hover{box-shadow:var(--shadow);border-color:#dde2ec}
.card .glow{position:absolute;inset:-1px;border-radius:inherit;pointer-events:none;opacity:0;transition:opacity .3s;background:radial-gradient(420px circle at var(--mx,50%) var(--my,50%),rgba(59,130,246,.09),transparent 45%)}
.card:hover .glow{opacity:1}
.span7{grid-column:span 7}.span5{grid-column:span 5}
.card h3{font-size:24px;letter-spacing:-.02em;font-weight:680;margin:18px 0 10px}
.card p.d{color:var(--ink3);font-size:15.5px;line-height:1.7;max-width:440px}
.ic{width:44px;height:44px;border-radius:13px;display:flex;align-items:center;justify-content:center;background:linear-gradient(180deg,var(--surface),rgb(var(--boardui-background-accent-subtle)));border:1px solid var(--line);color:var(--blue);box-shadow:0 4px 12px -4px rgba(37,99,235,.25)}

/* fidelity */
.fid{display:grid;grid-template-columns:190px 1fr;gap:28px;align-items:center;margin-top:28px}
.gauge{position:relative;width:190px;height:190px}
.gauge svg{transform:rotate(-90deg)}
.gauge .val{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center}
.gauge .val b{font-size:54px;font-weight:750;letter-spacing:-.04em;line-height:1}
.gauge .val small{font-size:12px;color:var(--ink4);margin-top:6px}
.gauge circle.p{transition:stroke-dashoffset 1.8s cubic-bezier(.2,.8,.2,1)}
.labs{display:flex;flex-direction:column;gap:8px}
.fidelity-metrics{color:var(--ink)}
.fidelity-metrics > div{background:var(--soft);border-color:var(--line2)}
.lab{display:grid;grid-template-columns:1fr auto;gap:6px 12px;padding:12px 14px;border-radius:14px;background:var(--soft);border:1px solid var(--line2)}
.lab span{font-size:13.5px;font-weight:550}
.lab em{font-style:normal;font-size:12px;color:var(--ink4);grid-column:1}
.lab b{grid-row:1/3;grid-column:2;align-self:center;font-size:20px;color:var(--green);font-weight:700}
.lab .bar{grid-column:1/3;height:4px;border-radius:2px;background:#e6eaf1;overflow:hidden}
.lab .bar i{display:block;height:100%;width:0;border-radius:2px;background:linear-gradient(90deg,#10b981,#34d399);transition:width 1.6s cubic-bezier(.2,.8,.2,1)}
.in .lab .bar i{width:100%}

/* pricing calc */
.calc{margin-top:26px}
.calc-l{display:flex;justify-content:space-between;font-size:13.5px;color:var(--ink3);margin-bottom:12px}
.calc-l b{color:var(--ink);font-weight:650}
input[type=range]{-webkit-appearance:none;appearance:none;width:100%;height:6px;border-radius:3px;background:linear-gradient(90deg,var(--blue) var(--p,40%),var(--line) var(--p,40%));outline:none}
input[type=range]::-webkit-slider-thumb{-webkit-appearance:none;width:22px;height:22px;border-radius:50%;background:var(--surface);border:1px solid #cfd8e8;box-shadow:0 2px 8px rgba(15,23,42,.18);cursor:grab}
.cmp{display:flex;flex-direction:column;gap:12px;margin-top:24px}
.cmp-r{display:grid;grid-template-columns:76px 1fr 90px;gap:12px;align-items:center;font-size:13.5px}
.cmp-r .track{height:34px;border-radius:10px;background:var(--soft);overflow:hidden}
.cmp-r .fill{height:100%;border-radius:10px;transition:width .35s cubic-bezier(.2,.8,.2,1)}
.cmp-r.off .fill{background:repeating-linear-gradient(-45deg,var(--line),var(--line) 6px,var(--soft) 6px,var(--soft) 12px)}
.cmp-r.us .fill{background:linear-gradient(90deg,#2563eb,#6366f1);min-width:8px}
.cmp-r .v{text-align:right;font-weight:650}
.cmp-r.off .v{color:var(--ink4);text-decoration:line-through}
.cmp-r.us .v{color:var(--blue)}
.save{margin-top:20px;display:flex;justify-content:space-between;align-items:center;padding:14px 16px;border-radius:14px;background:rgb(var(--boardui-background-success-subtle));border:1px solid var(--line);font-size:14px;color:var(--green)}
.save b{font-size:22px;letter-spacing:-.02em}

.ptable{width:100%;border-collapse:collapse;font-size:14.5px;margin-top:6px}
.ptable th{text-align:left;font-weight:550;color:var(--ink4);font-size:12.5px;padding:0 0 10px;border-bottom:1px solid var(--line)}
.ptable td{padding:15px 0;border-bottom:1px solid var(--line2)}
.ptable tr:last-child td{border-bottom:0}
.ptable .mdl{display:flex;gap:10px;align-items:center;font-weight:600}
.ptable s{color:var(--ink4)}
.ptable .ours{color:var(--blue);font-weight:650}
.tag90{font-size:11.5px;font-weight:650;color:var(--green);background:#ecfdf5;padding:3px 8px;border-radius:999px}
.mdot{width:24px;height:24px;border-radius:7px;display:flex;align-items:center;justify-content:center;font-size:11px;font-weight:700;color:#fff}

/* how it works */
.hw{display:grid;grid-template-columns:repeat(3,1fr);gap:18px}
.hw-card{display:flex;flex-direction:column;background:var(--surface);border:1px solid var(--line);border-radius:var(--r);padding:10px;transition:box-shadow .3s,border-color .3s}
.hw-card:hover{box-shadow:var(--shadow);border-color:#dde2ec}
.hw-vis{position:relative;height:232px;border-radius:16px;background:linear-gradient(180deg,var(--surface),var(--soft));border:1px solid var(--line2);overflow:hidden}
.hw-vis:before{content:"";position:absolute;inset:0;background-image:radial-gradient(rgba(15,23,42,.07) 1px,transparent 1px);background-size:16px 16px;-webkit-mask-image:radial-gradient(ellipse at center,#000 30%,transparent 80%);mask-image:radial-gradient(ellipse at center,#000 30%,transparent 80%)}
.hw-body{padding:22px 18px 16px}
.hw-k{font-size:12.5px;font-weight:650;color:var(--blue);letter-spacing:.04em}
.hw-body h3{font-size:20px;font-weight:680;letter-spacing:-.015em;margin:8px 0 8px}
.hw-body p{color:var(--ink3);font-size:15px;line-height:1.7}
.hw-body p b{color:var(--ink2);font-weight:600}

/* step 2: compatibility hub */
.v-hub{position:absolute;inset:0}
.v-hub>svg{position:absolute;inset:0;width:100%;height:100%}
.v-hub line{stroke:#c7d2fe;stroke-width:1.5;stroke-dasharray:4 5;animation:flow 1.2s linear infinite}
@keyframes flow{to{stroke-dashoffset:-18}}
.node{position:absolute;transform:translate(-50%,-50%);display:inline-flex;align-items:center;gap:6px;white-space:nowrap;font-size:11.5px;font-weight:600;color:var(--ink2);background:var(--surface);border:1px solid var(--line);border-radius:10px;padding:6px 9px;box-shadow:0 2px 6px -2px rgba(15,23,42,.1)}
.node .li{width:16px;height:16px}
.hub{position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);display:flex;align-items:center;gap:8px;padding:9px 14px;border-radius:14px;background:var(--surface);border:1px solid #c7d2fe;font-weight:650;font-size:14px;box-shadow:0 0 0 6px rgba(99,102,241,.08),0 10px 24px -10px rgba(37,99,235,.5)}
.hub .logo-mark{width:24px;height:24px;border-radius:7px}
.lbl{position:absolute;left:12px;font-size:10.5px;font-weight:600;color:var(--ink4);letter-spacing:.06em}

/* step 3: usage */
.v-use{position:absolute;inset:18px;display:flex;flex-direction:column;background:var(--surface);border:1px solid var(--line);border-radius:14px;padding:14px 16px;box-shadow:0 8px 20px -12px rgba(15,23,42,.18)}
.v-use .t{display:flex;justify-content:space-between;align-items:baseline}
.v-use .t small{font-size:11.5px;color:var(--ink4)}
.v-use .t strong{display:block;margin-top:2px;font-size:24px;letter-spacing:-.03em}
.v-use .t s{font-size:12px;color:var(--ink4)}
.bars{flex:1;display:flex;align-items:flex-end;gap:8px;margin:12px 0 8px}
.bars i{flex:1;border-radius:6px 6px 3px 3px;background:linear-gradient(180deg,#93c5fd,#3b82f6);transform-origin:bottom;transform:scaleY(.05);transition:transform 1s cubic-bezier(.2,.8,.2,1)}
.hw.in .bars i{transform:scaleY(1)}
.days{display:flex;gap:8px;font-size:10.5px;color:var(--ink4)}.days span{flex:1;text-align:center}

/* ---------- envelope (shared) ---------- */
.env{position:relative;width:360px;height:230px;perspective:1100px}
.env-shadow{position:absolute;left:8%;right:8%;bottom:-24px;height:26px;border-radius:50%;background:radial-gradient(closest-side,rgba(120,72,20,.26),transparent);filter:blur(3px)}
.env-back{position:absolute;inset:0;border-radius:16px;background:linear-gradient(180deg,#e9cd9d,#e3c18b);z-index:1}
.env-card{position:absolute;left:22px;right:22px;top:16px;height:190px;z-index:2;border-radius:16px;padding:18px 20px;color:#fff;text-align:left;overflow:hidden;
  background:radial-gradient(220px 120px at 90% 0,rgba(255,255,255,.28),transparent 60%),linear-gradient(135deg,#3b82f6,#4f46e5 70%);
  box-shadow:inset 0 0 0 1px rgba(253,230,138,.55),0 18px 30px -18px rgba(30,41,99,.55);
  transition:transform .95s cubic-bezier(.2,.9,.25,1)}
.env-card:before{content:"";position:absolute;top:0;bottom:0;left:62%;width:16px;background:linear-gradient(90deg,#f3c969,#fde68a,#f3c969);opacity:.9}
.ec-top{position:relative;display:flex;justify-content:space-between;font-size:12px;opacity:.85}
.ec-amt{position:relative;font-size:56px;font-weight:750;letter-spacing:-.045em;margin-top:22px;line-height:1;filter:blur(7px);opacity:.75;transition:filter .6s .9s,opacity .6s .9s}
.ec-bot{position:absolute;left:20px;right:20px;bottom:16px;display:flex;justify-content:space-between;align-items:flex-end;font-size:12.5px;opacity:.9}
.ec-chip{width:30px;height:22px;border-radius:5px;background:linear-gradient(135deg,#fde68a,#f2b84b);box-shadow:inset 0 0 0 1px rgba(146,98,20,.35)}
.ec-stamp{position:absolute;right:16px;top:58px;padding:4px 10px;border:2px solid #fde68a;color:#fde68a;border-radius:8px;font-weight:750;font-size:14px;letter-spacing:.1em;transform:rotate(-12deg) scale(1.6);opacity:0;transition:transform .4s cubic-bezier(.3,1.6,.5,1),opacity .2s}
.env-front{position:absolute;inset:0;z-index:3;border-radius:16px;clip-path:polygon(0 0,50% 54%,100% 0,100% 100%,0 100%);
  background:linear-gradient(to top right,transparent calc(50% - 1px),rgba(160,110,40,.18) 50%,transparent calc(50% + 1px)) left bottom/50% 72% no-repeat,
             linear-gradient(to top left,transparent calc(50% - 1px),rgba(160,110,40,.18) 50%,transparent calc(50% + 1px)) right bottom/50% 72% no-repeat,
             linear-gradient(165deg,#fff8ec,#f8e6c6)}
.env-flap{position:absolute;left:0;right:0;top:0;height:60%;z-index:4;transform-origin:top center;transform:rotateX(0);filter:drop-shadow(0 5px 6px rgba(120,72,20,.2));
  transition:transform .75s cubic-bezier(.3,.8,.3,1),z-index 0s .38s}
.env-flap i{position:absolute;inset:0;border-radius:16px 16px 0 0;clip-path:polygon(0 0,100% 0,50% 100%);background:linear-gradient(180deg,#fbe9c9,#efd3a5)}
.seal{position:absolute;left:50%;top:60%;width:56px;height:56px;margin:-28px 0 0 -28px;z-index:5;border-radius:50%;display:flex;align-items:center;justify-content:center;
  background:radial-gradient(circle at 35% 30%,#fde68a,#f2b84b 52%,#b7791f);box-shadow:0 8px 16px -6px rgba(146,64,14,.65),inset 0 0 0 4px rgba(255,255,255,.28);
  transition:transform .45s cubic-bezier(.3,1.6,.5,1),opacity .3s}

/* step 1: sealed envelope */
.v-env{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;background:radial-gradient(200px 120px at 50% 40%,rgba(253,224,171,.45),transparent 70%)}
.env-scale{transform:scale(.52);transform-origin:center;flex-shrink:0;margin-top:-6px}
.env-tag{position:absolute;right:18px;bottom:16px;display:inline-flex;gap:6px;align-items:center;font-size:12px;font-weight:650;color:#92400e;background:#fffbeb;border:1px solid #fde68a;padding:6px 10px;border-radius:999px;box-shadow:0 8px 20px -10px rgba(217,119,6,.5)}
.hw-card:hover .v-env .env-flap{transform:rotateX(32deg)}
.hw-card:hover .v-env .seal{transform:translateY(-6px) rotate(-8deg)}

/* ---------- CTA gift ---------- */
.gift-sec{position:relative;overflow:hidden;border-radius:32px;padding:36px 64px 36px 40px;text-align:left;display:grid;grid-template-columns:1.05fr 1fr;gap:24px;align-items:center;scroll-margin-top:96px;color:var(--ink);border:1px solid var(--line);
  background:linear-gradient(135deg,rgb(var(--boardui-background-warning-subtle)),var(--surface));
  box-shadow:0 50px 90px -60px rgba(180,120,40,.45)}
.gift-sec:before{content:"";position:absolute;inset:0;background-image:radial-gradient(rgba(180,120,40,.13) 1px,transparent 1px);background-size:20px 20px;-webkit-mask-image:radial-gradient(ellipse 55% 60% at 50% 35%,#000,transparent);mask-image:radial-gradient(ellipse 55% 60% at 50% 35%,#000,transparent);pointer-events:none}
.gift-sec>*{position:relative}
.gift-sec h2{font-size:46px;margin:0 0 14px;text-wrap:balance}
.gift-sec .sub{color:var(--ink3);max-width:440px;margin:0}
.g-copy .ctas{justify-content:flex-start!important}
.gift-stage{position:relative;height:420px;display:flex;align-items:flex-end;justify-content:center;padding-bottom:30px}
.env-btn{position:relative;border:0;background:none;padding:0;cursor:pointer;border-radius:18px;outline:none;animation:breathe 4.5s ease-in-out infinite}
.env-btn:focus-visible{box-shadow:0 0 0 3px #fff,0 0 0 5px var(--blue)}
@keyframes breathe{50%{transform:translateY(-7px)}}
.env-btn:hover .env-flap{transform:rotateX(34deg)}
.env-btn:hover .env-card{transform:translateY(-14px)}
.env-btn:hover .seal{transform:translateY(-8px) rotate(-10deg) scale(1.06)}
.g-rays{position:absolute;left:50%;top:120px;width:560px;height:560px;margin:-280px 0 0 -280px;border-radius:50%;opacity:0;pointer-events:none;transition:opacity .8s .5s;
  background:radial-gradient(circle,rgba(253,224,141,.75),transparent 42%),repeating-conic-gradient(rgba(59,130,246,.13) 0 8deg,transparent 8deg 18deg);
  -webkit-mask:radial-gradient(circle,#000 22%,transparent 66%);mask:radial-gradient(circle,#000 22%,transparent 66%)}
.g-spark{position:absolute;left:50%;top:120px;width:0;height:0;z-index:8;pointer-events:none}
.g-spark i{position:absolute;width:var(--s);height:var(--s);border-radius:50%;background:var(--c);box-shadow:0 0 8px var(--c);opacity:0;animation:spark var(--t) cubic-bezier(.15,.7,.35,1) forwards;animation-delay:var(--d)}
@keyframes spark{0%{opacity:0;transform:translate(-50%,-50%) scale(.2)}15%{opacity:1}100%{opacity:0;transform:translate(var(--x),var(--y)) scale(1)}}
.g-note{margin-top:14px;font-size:13px;color:var(--ink3);min-height:20px}
.g-sign{margin-top:20px;font-family:"Kaiti SC","STKaiti","KaiTi",cursive;font-size:19px;color:rgb(var(--boardui-text-warning))}

/* opened */
.gift-sec[data-state=open] .env-btn,.gift-sec[data-state=claimed] .env-btn{animation:none;cursor:default}
.gift-sec[data-state=open] .env-flap,.gift-sec[data-state=claimed] .env-flap{transform:rotateX(180deg)!important;z-index:1}
.gift-sec[data-state=open] .seal,.gift-sec[data-state=claimed] .seal{transform:scale(1.4)!important;opacity:0}
.gift-sec[data-state=open] .env-card,.gift-sec[data-state=claimed] .env-card{transform:translateY(-170px) rotate(-3deg)!important;transition-delay:.4s}
.gift-sec[data-state=open] .ec-amt,.gift-sec[data-state=claimed] .ec-amt{filter:none;opacity:1}
.gift-sec[data-state=open] .g-rays{opacity:1;animation:spin 22s linear infinite}
.gift-sec[data-state=claimed] .ec-stamp{opacity:1;transform:rotate(-12deg) scale(1);transition-delay:1.2s}
.gift-sec[data-state=claimed] .env-card{filter:saturate(.75)}
@keyframes spin{to{transform:rotate(360deg)}}
/* footer */
footer{margin-top:96px;border-top:1px solid var(--line);background:var(--soft)}
.foot{display:grid;grid-template-columns:repeat(3,1fr) 1.4fr;gap:40px;padding-block:56px 12px}
.foot-brand{order:4}
.foot p{color:var(--ink3);font-size:14px;line-height:1.7;margin-top:14px;max-width:300px}
.foot h4{font-size:13px;font-weight:650;margin-bottom:14px}
.foot ul{list-style:none;display:flex;flex-direction:column;gap:10px;font-size:14px;color:var(--ink3)}
.foot ul a:hover{color:var(--ink)}
.foot-giant{overflow:hidden;white-space:nowrap;text-align:center;margin-top:12px;font-size:clamp(56px,15vw,200px);font-weight:800;letter-spacing:-.05em;line-height:.95;color:transparent;background:linear-gradient(180deg,var(--line),transparent 90%);-webkit-background-clip:text;background-clip:text;user-select:none;pointer-events:none}
@media(max-width:760px){.foot{grid-template-columns:1fr 1fr}.foot-brand{order:-1;grid-column:1/-1}}
.foot-b{display:flex;flex-wrap:wrap;gap:6px 16px;justify-content:space-between;border-top:1px solid var(--line);padding-block:22px;font-size:13px;color:var(--ink4)}

/* reveal */
.rv{opacity:0;transform:translateY(24px);transition:opacity .8s cubic-bezier(.2,.8,.2,1),transform .8s cubic-bezier(.2,.8,.2,1)}
.rv.in{opacity:1;transform:none}
.d1{transition-delay:.08s}.d2{transition-delay:.16s}.d3{transition-delay:.24s}


@media (max-width:900px){
  h1{font-size:46px}h2{font-size:34px}
  .nav-links{display:none}
  .ib-menu{display:inline-flex}
  .nav-r>:not(.ib-menu){display:none}
  .menu{display:flex}
  .hero-g{grid-template-columns:1fr;gap:72px}.hero h1{font-size:46px}.ftile,.orbit{display:none}.stamp{right:16px}
  
  .span7,.span5{grid-column:span 12}
  .fid{grid-template-columns:1fr;justify-items:center}
  .hw{grid-template-columns:1fr}
  .foot{grid-template-columns:1fr 1fr}
  .gift-sec{grid-template-columns:minmax(0,1fr);gap:4px;padding:36px 18px 32px;text-align:center}.gift-sec h2{font-size:32px}.gift-sec .sub{margin:0 auto}.g-copy .ctas{justify-content:center!important}.gift-stage{height:330px;padding-bottom:20px}.gift-stage .env{transform:scale(.78);transform-origin:bottom center}.g-rays,.g-spark{top:100px}
}
@media (max-width:900px){
  section.sec{padding:64px 0}
  section.sec#how{padding-top:24px}
  .head{margin-bottom:32px}
  .logos{padding:40px 0 8px}
  footer{margin-top:64px}
  .hero{padding-block:36px 40px}
  .card{padding:24px}
  .fid{justify-items:stretch}.fid .gauge{justify-self:center}
  .hw-card{padding:6px}.hw-vis{border:0}
  .pc{padding:0;background:none;border:0}
  .stamp{right:10px;top:-34px;width:62px;height:62px}.stamp b{font-size:20px}.stamp small{font-size:9px}
  .gift-sec{padding-top:24px}.gift-stage{height:300px}
}
@media (max-width:600px){
  .hero .lead{font-size:17px;text-wrap:pretty}
  .hero .trust{flex-direction:column;align-items:flex-start;gap:8px}
  .pr{grid-template-columns:minmax(0,1fr) auto auto;gap:8px;padding:14px}
  .pr .m{gap:8px}.pr .mi{width:32px;height:32px;border-radius:10px}.pr .mi .li{width:18px;height:18px}
  .pr .m b{font-size:14px;white-space:nowrap}.pr .o{min-width:0}.pr .o strong{font-size:17px}
  .pc-h,.pc-f{padding-inline:14px}
  .foot{gap:32px 24px}
  .nav{padding:0 14px}
  .logo{font-size:15px}
  .promo-in{gap:8px;padding:6px 14px;font-size:12.5px}
}
@media (max-width:400px){
  .gift-sec{padding:32px 12px 28px}.gift-stage .env{transform:scale(.68)}
}
@media (prefers-reduced-motion:reduce){
  *,*:before,*:after{animation:none!important;transition:none!important}
  .rv{opacity:1;transform:none}
  .env-card,.ec-amt{transition:none!important}
}
</style>
