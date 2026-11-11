import type { RouteMeta } from 'vue-router'

export function usesBoardUIWorkspace(meta: RouteMeta): boolean {
  return meta.requiresAdmin === true || meta.workspace === 'account'
}
