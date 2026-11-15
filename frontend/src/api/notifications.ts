import { apiClient } from './client'
import type { UserAnnouncement } from '@/types'

export interface UserNotification extends UserAnnouncement {
  type: 'announcement' | 'gift'
  gift_amount?: number
  gift_source?: 'initial' | 'credit' | 'payment_bonus'
}
export const notificationsAPI = {
  async list(): Promise<UserNotification[]> { const { data } = await apiClient.get<UserNotification[]>('/notifications'); return data },
  async markRead(type: UserNotification['type'], id: number): Promise<void> { await apiClient.post(`/notifications/${type}/${id}/read`) },
}
