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
