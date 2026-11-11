import { apiClient } from './client'

export interface OPCPayOperation {
  amount: number
  credit_usd: number
  business_id: string
  action_id: string
  kind: 'purchase' | 'consume'
  sku_id: string
  status: string
  result?: { status: string; purchase_url?: string; purchase_qr_code?: string }
}
export interface OPCPayInfo {
  configured: boolean
  enabled: boolean
  provider_name: string
  provider_id: number
  capabilities: { provider_configured: boolean; capabilities: { purchase_entry: boolean; consumption: boolean } }
  product: {
    capability_status: string
    product_name: string
    billing_mode: string
    quota_unit?: string
    charging_options: { sku_id: string; display_name: string; price: string; currency: string; quota_amount?: number; quota_unit?: string }[]
  }
  access?: { decision: string; access_granted: boolean; requires_consumption: boolean; remaining: number; quota_unit?: string; valid_until?: string }
  credit_usd_per_unit: number
  pending?: OPCPayOperation
}
export const opcpayAPI = {
  info: () => apiClient.get<OPCPayInfo>('/payment/opcpay'),
  start: (kind: 'purchase' | 'consume', action_id: string, sku_id = '') =>
    apiClient.post<OPCPayOperation>('/payment/opcpay/operations', { kind, action_id, sku_id }, { timeout: 65000 }),
  close: (id: string) => apiClient.post(`/payment/opcpay/operations/${encodeURIComponent(id)}/close`),
  resume: (id: string) => apiClient.post<OPCPayOperation>(`/payment/opcpay/operations/${encodeURIComponent(id)}/resume`, {}, { timeout: 65000 })
}
