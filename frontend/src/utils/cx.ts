import { extendTailwindMerge } from 'tailwind-merge'

const families = ['display-4', 'title-1', 'title-2', 'title-3', 'headline', 'body', 'body-2', 'caption-1', 'caption-2']
const weights = ['regular', 'medium', 'semibold', 'bold']

/** Tailwind CSS 3 class merge with the port's composite typography groups. */
export const cx = extendTailwindMerge({
  extend: { classGroups: { 'font-size': [{ text: families.flatMap(family => weights.map(weight => `${family}-${weight}`)) }] } }
})
