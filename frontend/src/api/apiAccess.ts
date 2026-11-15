import { apiClient } from './client'
import type { ApiKey } from '@/types'

export interface APIAccess {
  state: string
  default_key: ApiKey | null
  billing_policy: 'subscription_then_bonus_then_paid'
}
export const prepareAPIAccess = async (): Promise<APIAccess> => {
  const { data } = await apiClient.post<APIAccess>('/user/api-access', {})
  return data
}
export const createUnifiedAPIKey = async (): Promise<ApiKey> => {
  const { data } = await apiClient.post<ApiKey>('/user/api-access/keys', {})
  return data
}

import type { UserSupportedModelPricing } from './channels'
export interface UnifiedWalletOffer {
  endpoint: string
  rate_multiplier: number
  pricing: UserSupportedModelPricing | null
  time_pricing?: { timezone: string; periods: Array<{ start_time: string; end_time: string; multiplier: number }> } | null
  long_context_basis?: string
}
export interface UnifiedModel {
  id: string
  platform: string
  subscription_available: boolean
  wallet_offers: UnifiedWalletOffer[]
}
export const getUnifiedModels = async (): Promise<UnifiedModel[]> => {
  const { data } = await apiClient.get<UnifiedModel[]>('/user/api-access/models')
  return data
}
