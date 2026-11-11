import { computed, ref } from 'vue'
import { useMutationObserver } from '@vueuse/core'

/** Resolve CSS colors for canvas charts and update them when the root theme changes. */
export function useBoardUIChartTheme() {
  const revision = ref(0)
  useMutationObserver(document.documentElement, () => { revision.value++ }, {
    attributes: true,
    attributeFilter: ['class', 'data-admin-ui']
  })
  return computed(() => {
    void revision.value
    const style = getComputedStyle(document.documentElement)
    const color = (name: string) => {
      const channels = style.getPropertyValue(`--boardui-${name}`).trim().split(/\s+/).map(Number)
      return `#${channels.map(channel => channel.toString(16).padStart(2, '0')).join('')}`
    }
    return {
      text: color('text-secondary'),
      grid: color('separator-border'),
      neutral: color('chart-track'),
      series: Array.from({ length: 12 }, (_, index) => color(`chart-${index + 1}`))
    }
  })
}
