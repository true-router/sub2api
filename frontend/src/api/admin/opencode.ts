import { apiClient } from '../client'

export type OpenCodeOAuthCredentials = Record<string, unknown> & {
  access_token: string
  refresh_token: string
  expires_at: string
  org_id: string
  org_name?: string
}
export interface OpenCodeDevice {
  device_code: string
  user_code: string
  verification_uri_complete: string
  expires_in: number
  interval: number
}
export interface OpenCodeAuthorization {
  status: 'authorized' | 'authorization_pending' | 'slow_down'
  credentials: OpenCodeOAuthCredentials | null
}
export async function authorizeOpenCode(proxyID: number | null, signal?: AbortSignal) {
  const { data } = await apiClient.post<OpenCodeDevice>('/admin/opencode/oauth/device', { proxy_id: proxyID }, { signal })
  return data
}
export async function pollOpenCode(deviceCode: string, proxyID: number | null, signal?: AbortSignal) {
  const { data } = await apiClient.post<OpenCodeAuthorization>('/admin/opencode/oauth/poll', { device_code: deviceCode, proxy_id: proxyID }, { signal })
  return data
}
export async function importOpenCode(credential: unknown, signal?: AbortSignal) {
  const { data } = await apiClient.post<OpenCodeAuthorization>('/admin/opencode/oauth/import', { credential }, { signal })
  return data
}
