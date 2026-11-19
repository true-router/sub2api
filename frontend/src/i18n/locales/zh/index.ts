import adminAccessKeys from './adminAccessKeys'
import mcp from './mcp'
import landing from './landing'
import landingPage from './landingPage'
import common from './common'
import dashboard from './dashboard'
import channelMonitorV2 from './channelMonitorV2'
import batchImage from './batchImage'
import admin from './admin'
import misc from './misc'

export default {
  vmUser: {
    method: '分配方式',
    dedicated: '创建专属分组',
    existing: '选择已有分组',
    active: '已启用',
    inactive: '已停用',
    bound: '已按固定用户 ID 绑定。',
    checkIdentity: '请核对两端身份。确认后将绑定并激活此 vm2api 用户。',
    createIdentity: '确认后将使用此邮箱创建 vm2api 用户。',
    loading: '正在核对用户身份',
    noData: '暂无数据',
    unavailable: '用量暂不可用',
    empty: '暂无槽位',
    five: '5 小时已用',
    seven: '7 天已用',
  },
  wallet: {
    convertToBonus: '转为赠送积分',
    conversionAmount: '划转金额',
    conversionReason: '划转原因（必填）',
    conversionRule: '划转保持总积分不变。划转后的金额不可退款。只能划转未消费、未冻结的充值积分。',
    paidAfterConversion: '划转后充值积分',
    bonusAfterConversion: '划转后赠送积分',
    conversionReasonRequired: '请填写划转原因。',
    conversionInsufficient: '可用充值积分不足，无法划转。',

    creditSource: '资金来源',
    paidCreditEvidence: '计入充值积分前，请在备注中填写收款凭证。',

    paidBalance: '充值积分',
    bonusBalance: '赠送积分',
    unverifiedBalance: '待核对积分',
    spendRule: '消费先扣赠送积分。赠送积分不可退款。',
    unverifiedRule: '待核对积分可用于消费。核对资金来源后才能退款。',
    refundRule: '退款扣除充值积分，并收回该订单尚未消费的赠送积分。',
    refundPrincipal: '订单充值本金',
    refundPaidAvailable: '可用充值积分',
    refundPrincipalAmount: '退还充值本金（积分）',
    refundBonus: '原赠送金额（不可退款）',
    adminCreditRule: '选择赠送积分时，增加的金额不可退款。',
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
  ...landingPage,
  ...common,
  ...dashboard,
  ...channelMonitorV2,
  ...batchImage,
  admin,
  ...misc,
}
