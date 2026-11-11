import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import landing from './landing'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'

export default {
  wallet: {
    convertToBonus: 'Convert to promotional balance',
    conversionAmount: 'Conversion amount',
    conversionReason: 'Conversion reason (required)',
    conversionRule: 'Total balance stays unchanged. Converted credit cannot be refunded. Only unspent, unfrozen paid credit can be converted.',
    paidAfterConversion: 'Paid balance after conversion',
    bonusAfterConversion: 'Promotional balance after conversion',
    conversionReasonRequired: 'Enter the conversion reason.',
    conversionInsufficient: 'Available paid balance is insufficient for conversion.',

    creditSource: 'Funding source',
    paidCreditEvidence: 'Provide payment evidence in notes before crediting paid balance.',

    paidBalance: 'Paid balance',
    bonusBalance: 'Promotional balance',
    unverifiedBalance: 'Unverified balance',
    spendRule: 'Promotional credit is spent first and cannot be refunded.',
    unverifiedRule: 'Unverified credit can be spent. Funding must be verified before a refund.',
    refundRule: 'Refunds deduct paid credit and reclaim this order’s unspent promotional credit.',
    refundPrincipal: 'Order paid credit',
    refundPaidAvailable: 'Available paid credit',
    refundPrincipalAmount: 'Refund paid credit (USD)',
    refundBonus: 'Original promotional credit (not refundable)',
    adminCreditRule: 'Promotional additions cannot be refunded.',
  },
  boardui: {
    language: 'Language',
    quickSearch: 'Quick search',
    theme: 'Theme',
    pagination: 'Pagination',
    clearSearch: 'Clear search',
    commandTitle: 'Search or run a command',
    commandPlaceholder: 'Search pages and actions…',
    commandNavigate: 'Navigate',
    commandSelect: 'Open',
    commandNavigation: 'Navigation',
    breadcrumb: 'Breadcrumb',
    searchNavigation: 'Search navigation',
    overview: 'Overview',
    resources: 'Resources',
    business: 'Business',
    system: 'System',
    workspace: 'WORKSPACE OVERVIEW',
    dashboardHeading: 'Your gateway at a glance',
    updatedData: 'Statistics updated',
    staleData: 'Statistics pending update',
  },
  adminAccessKeys,
  mcp,
  ...landing,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
}
