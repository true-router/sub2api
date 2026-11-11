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
    convertToBonus: '转为赠送余额',
    conversionAmount: '划转金额',
    conversionReason: '划转原因（必填）',
    conversionRule: '划转保持总余额不变。划转后的金额不可退款。只能划转未消费、未冻结的充值余额。',
    paidAfterConversion: '划转后充值余额',
    bonusAfterConversion: '划转后赠送余额',
    conversionReasonRequired: '请填写划转原因。',
    conversionInsufficient: '可用充值余额不足，无法划转。',

    creditSource: '资金来源',
    paidCreditEvidence: '计入充值余额前，请在备注中填写收款凭证。',

    paidBalance: '充值余额',
    bonusBalance: '赠送余额',
    unverifiedBalance: '待核对余额',
    spendRule: '消费先扣赠送余额。赠送余额不可退款。',
    unverifiedRule: '待核对余额可用于消费。核对资金来源后才能退款。',
    refundRule: '退款扣除充值余额，并收回该订单尚未消费的赠送余额。',
    refundPrincipal: '订单充值本金',
    refundPaidAvailable: '可用充值余额',
    refundPrincipalAmount: '退还充值本金（USD）',
    refundBonus: '原赠送金额（不可退款）',
    adminCreditRule: '选择赠送余额时，增加的金额不可退款。',
  },
  boardui: {
    language: '语言',
    quickSearch: '快速搜索',
    theme: '主题',
    pagination: '分页',
    clearSearch: '清除搜索',
    commandTitle: '搜索页面或操作',
    commandPlaceholder: '搜索页面和操作…',
    commandNavigate: '选择',
    commandSelect: '打开',
    commandNavigation: '导航',
    breadcrumb: '导航路径',
    searchNavigation: '搜索导航',
    overview: '概览',
    resources: '资源管理',
    business: '业务管理',
    system: '系统管理',
    workspace: '工作空间概览',
    dashboardHeading: '网关运行概览',
    updatedData: '统计已更新',
    staleData: '统计待更新',
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
