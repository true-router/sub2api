import type { ModelPlazaGroup, PlazaExchangeRate, PlazaModel } from '@/api/modelPlaza'
import type { UserPricingInterval } from '@/api/channels'
import { resolveIntervalPrices } from '@/utils/pricing'

export type DisplayCurrency = 'CNY' | 'USD'
export type PriceKey = 'input_price' | 'output_price' | 'cache_write_price' | 'cache_write_1h_price' | 'cache_write_12h_price' | 'cache_read_price'
export interface PriceRow {
  key: string
  label: string
  credits: number | null
  official: number | null
  tier?: string
  scale: number
}
const fields: [PriceKey, string][] = [
  ['input_price', 'input'], ['output_price', 'output'],
  ['cache_write_price', 'cacheWrite'], ['cache_write_1h_price', 'cacheWrite1h'],
  ['cache_write_12h_price', 'cacheWrite12h'], ['cache_read_price', 'cacheRead']
]
export function validQuote(quote?: PlazaExchangeRate | null): quote is PlazaExchangeRate {
  if (!quote || !Number.isFinite(quote.usd_to_cny) || quote.usd_to_cny <= 0) return false
  const date = Date.parse(quote.date)
  return Number.isFinite(date) && date <= Date.now() + 86400000 && Date.now() - date <= 7 * 86400000
}
export function creditCost(value?: number): number {
  return value != null && Number.isFinite(value) && value > 0 ? value : 1
}
export function effectiveRate(group: ModelPlazaGroup, model: PlazaModel): number {
  const mode = model.pricing?.billing_mode
  if (mode === 'image' && group.image_rate_independent) return group.image_rate_multiplier
  if (mode === 'video' && group.video_rate_independent) return group.video_rate_multiplier
  return group.user_rate_multiplier ?? group.rate_multiplier
}
export function money(value: number | null, currency: DisplayCurrency): string {
  if (value == null || !Number.isFinite(value) || value < 0) return '—'
  const prefix = currency === 'CNY' ? '¥' : 'US$'
  if (value > 0 && value < 1e-12) return prefix + value.toExponential(2)
  // Never turn a positive price into a zero charge.
  const digits = value > 0 && value < .01 ? Math.min(14, Math.max(4, 2 - Math.floor(Math.log10(value)))) : value < 1 ? 4 : 2
  return prefix + value.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: digits })
}
export function savings(credits: number | null, official: number | null, cnyPerCredit: number, quote?: PlazaExchangeRate | null): number | null {
  if (credits == null || !Number.isFinite(credits) || credits < 0 || official == null || !Number.isFinite(official) || official <= 0 || !validQuote(quote)) return null
  return (1 - credits * creditCost(cnyPerCredit) / (official * quote.usd_to_cny)) * 100
}
export function savingsPercent(value: number): string {
  if (value > 0 && value < .1) return '<0.1'
  return (value >= 100 ? 100 : Math.min(99.9, Math.max(0, value))).toFixed(1)
}
function tokenCount(value: number) { return value >= 1000000 ? `${value / 1000000}M` : value >= 1000 ? `${value / 1000}K` : String(value) }
function band(min: number, max: number | null) { return max == null ? `>${tokenCount(min)}` : min === 0 ? `≤${tokenCount(max)}` : `${tokenCount(min)}–${tokenCount(max)}` }
function intervalAt(intervals: UserPricingInterval[], min: number) {
  return intervals.find(iv => iv.min_tokens <= min && (iv.max_tokens == null || iv.max_tokens > min))
}
export function priceRows(model: PlazaModel): PriceRow[] {
  const p = model.pricing
  if (p && p.billing_mode !== 'token') {
    const rows = (p.intervals ?? []).filter(iv => iv.per_request_price != null)
    return rows.length ? rows.map((iv, i) => ({ key: `request-${i}`, label: p.billing_mode, tier: iv.tier_label || band(iv.min_tokens, iv.max_tokens), credits: iv.per_request_price, official: null, scale: 1 }))
      : [{ key: 'request', label: p.billing_mode, credits: p.per_request_price, official: null, scale: 1 }]
  }
  const paid = [...(p?.intervals ?? [])].sort((a, b) => a.min_tokens - b.min_tokens).map(iv => resolveIntervalPrices(iv, p!))
  const official = [...(model.official_pricing?.intervals ?? [])].sort((a, b) => a.min_tokens - b.min_tokens)
  // Split at both catalogs' boundaries so every comparison covers the same context band.
  const boundaries = [...new Set([0, ...paid.map(iv => iv.min_tokens), ...official.map(iv => iv.min_tokens), ...[...paid, ...official].flatMap(iv => iv.max_tokens == null ? [] : [iv.max_tokens])])].sort((a, b) => a - b)
  const result: PriceRow[] = []
  for (let i = 0; i < boundaries.length; i++) {
    const min = boundaries[i]
    const own = paid.length ? intervalAt(paid, min) : p
    const ref = official.length ? intervalAt(official, min) : model.official_pricing
    for (const [key, label] of fields) {
      const credits = own?.[key] ?? null
      // Marginal ladders and whole-request ladders cannot be compared as flat prices.
      const reference = model.long_context_basis === 'marginal' && boundaries.length > 1 ? null : ref?.[key] ?? null
      if (label !== 'input' && label !== 'output' && credits == null && reference == null) continue
      result.push({ key: `${key}-${min}`, label, credits, official: reference, tier: boundaries.length > 1 ? band(min, boundaries[i + 1] ?? null) : undefined, scale: 1000000 })
    }
  }
  return result
}
