import { getLocale, i18n } from '@/i18n'

/** Display unit for internal accounting values. This is not a currency code. */
export const POINTS_SYMBOL = 'P'

export function formatPoints(
  amount: number | null | undefined,
  options: Intl.NumberFormatOptions = {}
): string {
  const value = typeof amount === 'number' && Number.isFinite(amount) ? amount : 0
  const digits = value !== 0 && Math.abs(value) < 0.01 ? 6 : 2
  const number = new Intl.NumberFormat(getLocale(), {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
    ...options,
    style: 'decimal'
  }).format(value)
  return `${number} ${i18n.global.t('common.pointsUnit')}`
}
