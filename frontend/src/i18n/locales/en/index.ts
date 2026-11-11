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
