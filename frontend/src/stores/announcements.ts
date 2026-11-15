import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { notificationsAPI, type UserNotification } from '@/api/notifications'

const THROTTLE_MS = 30 * 1000 // Bound inbox refresh while the user is active.

export const useAnnouncementStore = defineStore('announcements', () => {
  // State
  const announcements = ref<UserNotification[]>([])
  const loading = ref(false)
  const lastFetchTime = ref(0)
  const popupQueue = ref<UserNotification[]>([])
  const currentPopup = ref<UserNotification | null>(null)

  // Session-scoped dedup set — not reactive, used as plain lookup only
  let shownPopupIds = new Set<string>()
  let fetchGeneration = 0

  // Getters
  const unreadCount = computed(() =>
    announcements.value.filter((a) => !a.read_at).length
  )

  // Actions
  async function fetchAnnouncements(force = false) {
    const now = Date.now()
    if (!force && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) {
      return
    }

    // Set immediately to prevent concurrent duplicate requests
    lastFetchTime.value = now
    const generation = ++fetchGeneration

    try {
      loading.value = true
      const all = await notificationsAPI.list()
      if (generation !== fetchGeneration) return
      announcements.value = all
      enqueueNewPopups()
    } catch (err: any) {
      if (generation !== fetchGeneration) return
      // Revert throttle timestamp on failure so retry is allowed
      lastFetchTime.value = 0
      console.error('Failed to fetch announcements:', err)
    } finally {
      if (generation === fetchGeneration) loading.value = false
    }
  }

  function enqueueNewPopups() {
    const newPopups = announcements.value.filter(
      (a) => a.notify_mode === 'popup' && !a.read_at && !shownPopupIds.has(`${a.type}:${a.id}`)
    )
    if (newPopups.length === 0) return

    for (const p of newPopups) {
      if (!popupQueue.value.some((q) => q.id === p.id && q.type === p.type)) {
        popupQueue.value.push(p)
      }
    }

    if (!currentPopup.value) {
      showNextPopup()
    }
  }

  function showNextPopup() {
    currentPopup.value = null
    while (popupQueue.value.length) {
      const queued = popupQueue.value.shift()!
      const latest = announcements.value.find(a => a.type === queued.type && a.id === queued.id)
      if (!latest || latest.read_at) continue
      currentPopup.value = latest
      shownPopupIds.add(`${latest.type}:${latest.id}`)
      return
    }
  }

  async function dismissPopup() {
    if (!currentPopup.value) return
    const { id, type } = currentPopup.value
    if (type !== 'gift' && !await markAsRead(id, type)) return
    currentPopup.value = null
    showNextPopup()
  }

  async function markAsRead(id: number, type: UserNotification['type'] = 'announcement') {
    try {
      await notificationsAPI.markRead(type, id)
      fetchGeneration++
      loading.value = false
      const ann = announcements.value.find((a) => a.id === id && a.type === type)
      const readAt = new Date().toISOString()
      if (ann) ann.read_at = readAt
      if (currentPopup.value?.id === id && currentPopup.value.type === type) currentPopup.value.read_at = readAt
      return true
    } catch (err: any) {
      console.error('Failed to mark announcement as read:', err)
      return false
    }
  }

  async function markAllAsRead() {
    const unread = announcements.value.filter((a) => !a.read_at && a.type === 'announcement')
    if (unread.length === 0) return

    try {
      loading.value = true
      fetchGeneration++
      const results = await Promise.allSettled(unread.map(async (a) => {
        await notificationsAPI.markRead(a.type, a.id)
        a.read_at = new Date().toISOString()
      }))
      const failure = results.find((result) => result.status === 'rejected')
      if (failure) throw failure.reason
    } catch (err: any) {
      console.error('Failed to mark all as read:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  function openGift(item: UserNotification) {
    if (currentPopup.value && (currentPopup.value.type !== item.type || currentPopup.value.id !== item.id)) popupQueue.value.unshift(currentPopup.value)
    popupQueue.value = popupQueue.value.filter(a => !(a.type === item.type && a.id === item.id))
    currentPopup.value = item
    shownPopupIds.add(`${item.type}:${item.id}`)
  }

  function reset() {
    fetchGeneration++
    announcements.value = []
    lastFetchTime.value = 0
    shownPopupIds = new Set()
    popupQueue.value = []
    currentPopup.value = null
    loading.value = false
  }

  return {
    // State
    announcements,
    loading,
    currentPopup,
    // Getters
    unreadCount,
    // Actions
    fetchAnnouncements,
    dismissPopup,
    openGift,
    markAsRead,
    markAllAsRead,
    reset,
  }
})
