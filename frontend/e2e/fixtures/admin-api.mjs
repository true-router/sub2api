import { createServer } from 'node:http'

export const PREVIEW_USER = { id: 1, email: 'admin@example.test', username: 'Admin', role: 'admin', status: 'active', balance: 100, concurrency: 10, allowed_groups: [], created_at: '2026-01-01T00:00:00Z' }

/** Shared, in-memory API contracts for development preview and browser E2E. */
export function createAdminFixtureServer({ requests = [], accountPreview = false, dashboardPreview = false } = {}) {
const user = PREVIEW_USER
const users = Array.from({ length: 28 }, (_, i) => ({ ...user, id: i + 2, role: 'user', username: `Member ${i + 1}`, email: `member${i + 1}@example.test`, subscriptions: [] }))
const stats = Object.fromEntries(['total_users', 'today_new_users', 'active_users', 'hourly_active_users', 'total_api_keys', 'active_api_keys', 'total_accounts', 'normal_accounts', 'error_accounts', 'ratelimit_accounts', 'overload_accounts', 'total_requests', 'total_input_tokens', 'total_output_tokens', 'total_cache_creation_tokens', 'total_cache_read_tokens', 'total_tokens', 'total_cost', 'total_actual_cost', 'total_account_cost', 'today_requests', 'today_input_tokens', 'today_output_tokens', 'today_cache_creation_tokens', 'today_cache_read_tokens', 'today_tokens', 'today_cost', 'today_actual_cost', 'today_account_cost', 'average_duration_ms', 'uptime', 'rpm', 'tpm'].map((key, i) => [key, (i + 1) * 12]))
Object.assign(stats, { total_users: 1284, total_accounts: 48, normal_accounts: 46, error_accounts: 2, today_requests: 24680, today_tokens: 1850000, total_tokens: 86000000, stats_stale: false, stats_updated_at: '2026-11-11T04:00:00Z' })
const settings = { site_name: 'TrueRouter', site_logo: '', site_version: '1.0.0', registration_enabled: true, subscription_enabled: false, payment_enabled: false, ops_monitoring_enabled: true, custom_menu_items: [], feature_flags: {}, batch_image_enabled: false, backend_mode_enabled: false }
const member = users[0]
if (accountPreview) Object.assign(settings, {
  subscription_enabled: true, payment_enabled: true, available_channels_enabled: true,
  channel_monitor_enabled: true, affiliate_enabled: true, mcp_enabled: true,
  allow_user_view_error_requests: true,
  custom_menu_items: [{ id: 'preview', label: 'Preview resource', url: '', icon_svg: '', visibility: 'user', sort_order: 1 }]
})
const monitorTime = '2026-11-11T04:00:00Z'
const coverage = { requested_start: monitorTime, coverage_start: monitorTime, data_through: monitorTime, computed_at: monitorTime, aggregation_lag_seconds: 0, coverage_complete: true, bucket_seconds: 300 }
const latency = { sample_count: 30, p50_ms: 200, p95_ms: 320, avg_ms: 250 }
const metrics = { success_requests: 30, error_requests: 0, request_count: 30, token_count: 24000, rpm: 2, tpm: 1600, error_rate: 0, cache_rate: 0.6, cache_rate_numerator: 1200, cache_rate_denominator: 2000, ttft: latency, duration: latency }
const health = { overall: 'healthy', error_rate: 'healthy', ttft: 'healthy', cache: 'healthy', minimum_sample: 10 }
const backupS3 = { endpoint: '', region: 'auto', bucket: '', access_key_id: '', prefix: 'backups/', force_path_style: false }
const imageStorage = { ...backupS3, enabled: false, reuse_backup_s3: true, prefix: 'images/', public_base_url: '', presign_expiry_hours: 24, max_download_bytes: 33554432 }
return createServer(async (req, res) => {
  const url = new URL(req.url, 'http://127.0.0.1')
  const path = url.pathname.replace(/^\/api\/v1/, '')
  let body = ''
  for await (const chunk of req) body += chunk
  requests.push({ method: req.method, path, search: url.search, ...(body ? { body: JSON.parse(body) } : {}) })
  const currentUser = accountPreview && req.headers.authorization === 'Bearer local-member-preview' ? member : user
  let data = { items: [], total: 0, pages: 1, page: 1, page_size: 20 }
  if (path === '/setup/status') data = { needs_setup: false }
  else if (path === '/auth/me' || path === '/user/profile') data = currentUser
  else if (path === '/auth/login' && req.method === 'POST') {
    const credentials = JSON.parse(body || '{}')
    const adminLogin = accountPreview ? 'admin' : user.email
    const loginUser = credentials.email === adminLogin ? user : accountPreview && credentials.email === member.email ? member : null
    const expectedPassword = accountPreview && loginUser === user ? 'admin' : 'BoardUI-Preview-2026'
    if (!loginUser || credentials.password !== expectedPassword) {
      res.writeHead(401, { 'Content-Type': 'application/json' })
      res.end(JSON.stringify({ code: 401, message: 'Use the local preview credentials.' }))
      return
    }
    data = { access_token: loginUser === member ? 'local-member-preview' : 'local-preview-only', expires_in: 86400, token_type: 'Bearer', user: loginUser }
  }
  else if (path === '/settings/public' || path === '/admin/settings') data = settings
  else if (path === '/admin/compliance') data = { required: false, version: '1' }
  else if (path === '/admin/payment/config') data = { enabled: false }
  else if (req.method === 'GET' && ['/admin/payment/providers', '/admin/payment/plans', '/admin/payment/channels'].includes(path)) data = []
  else if (req.method === 'GET' && path === '/admin/settings/email-templates') data = { events: [], locales: ['zh', 'en'], templates: [], placeholders: [] }
  else if (req.method === 'GET' && path === '/admin/settings/beta-policy') data = { rules: [] }
  else if (req.method === 'GET' && path === '/admin/settings/web-search-emulation') data = { enabled: false, providers: [] }
  else if (req.method === 'GET' && path === '/admin/backups/s3-config') data = backupS3
  else if (req.method === 'GET' && path === '/admin/backups/image-storage') data = { config: imageStorage, secret_configured: false }
  else if (req.method === 'GET' && path === '/admin/backups/schedule') data = { enabled: false, cron_expr: '0 2 * * *', retain_days: 14, retain_count: 10 }
  else if (path === '/admin/dashboard/snapshot-v2') data = { stats, trend: Array.from({ length: 7 }, (_, i) => ({ date: `2026-11-${String(i + 1).padStart(2, '0')}`, requests: 100 + i * 30, input_tokens: 2000 + i * 500, output_tokens: 800 + i * 100, total_tokens: 2800 + i * 600, cost: i + 1, actual_cost: i + 1 })), models: [{ model: 'claude-sonnet-4-5', requests: 12400, total_tokens: 900000, actual_cost: 42 }] }
  else if (path === '/admin/users' && req.method === 'POST') {
    const created = { ...user, ...JSON.parse(body), id: users.length + 2, subscriptions: [] }
    users.unshift(created)
    data = created
  } else if (path === '/admin/users') {
    const filtered = users.filter(item => `${item.email} ${item.username}`.includes(url.searchParams.get('search') || ''))
    const page = Number(url.searchParams.get('page') || 1)
    const size = Number(url.searchParams.get('page_size') || 20)
    data = { items: filtered.slice((page - 1) * size, page * size), total: filtered.length, pages: Math.ceil(filtered.length / size) || 1, page, page_size: size }
  } else if (['/announcements', '/subscriptions/active', '/admin/user-attributes', '/admin/groups/all', '/admin/proxies/all', '/admin/groups/usage-summary', '/admin/groups/capacity-summary', '/admin/plugins'].includes(path)) data = []
  else if (req.method === 'GET' && /^\/admin\/users\/\d+\/attributes$/.test(path)) data = []
  else if (req.method === 'POST' && path === '/admin/user-attributes/batch') data = { attributes: {} }
  else if (path === '/admin/ops/runtime/logging' || path === '/admin/ops/runtime/logging/reset') data = { level: 'info', persist_access_logs: false, enable_sampling: false, sampling_initial: 100, sampling_thereafter: 100, caller: true, stacktrace_level: 'error', retention_days: 30, request_retention_days: 90, ...(req.method === 'PUT' ? JSON.parse(body) : {}) }
  else if (path === '/admin/ops/system-logs/health') data = { queue_depth: 0, queue_capacity: 1000, dropped_count: 0, write_failed_count: 0, written_count: 0, avg_write_delay_ms: 0 }
  else if (path === '/admin/ops/advanced-settings') data = { data_retention: { cleanup_enabled: false, cleanup_schedule: '0 2 * * *', error_log_retention_days: 30, minute_metrics_retention_days: 7, hourly_metrics_retention_days: 90 }, aggregation: { aggregation_enabled: false }, openai_account_quota_auto_pause: { default_threshold_5h: 0, default_threshold_7d: 0 }, display_openai_token_stats: false, display_alert_events: false, auto_refresh_enabled: false, auto_refresh_interval_seconds: 30 }
  else if (path === '/admin/ops/dashboard/snapshot-v2') data = { generated_at: new Date().toISOString(), overview: { start_time: new Date().toISOString(), end_time: new Date().toISOString(), platform: '', success_count: 0, error_count_total: 0, business_limited_count: 0, error_count_sla: 0, request_count_total: 0, request_count_sla: 0, token_consumed: 0, sla: 100, error_rate: 0, upstream_error_rate: 0, upstream_error_count_excl_429_529: 0, upstream_429_count: 0, upstream_529_count: 0, qps: { current: 0, peak: 0, avg: 0 }, tps: { current: 0, peak: 0, avg: 0 }, duration: {}, ttft: {} }, throughput_trend: { points: [], by_platform: [], top_groups: [] }, error_trend: { points: [] } }
  else if (path === '/admin/ops/dashboard/latency-histogram') data = { total_requests: 0, buckets: [] }
  else if (path === '/admin/ops/dashboard/error-distribution') data = { total: 0, items: [] }
  else if (['/admin/ops/concurrency', '/admin/ops/account-availability'].includes(path)) data = { enabled: true, platform: {}, group: {}, account: {} }
  else if (path === '/admin/ops/user-concurrency') data = { enabled: true, user: {} }
  else if (path === '/admin/ops/realtime-traffic') data = { enabled: false, summary: null }
  else if (path === '/admin/ops/settings/metric-thresholds') data = {}
  else if (path === '/admin/ops/dashboard/throughput-trend' || path === '/admin/ops/dashboard/error-trend') data = { points: [], by_platform: [], top_groups: [] }
  else if (path.endsWith('/channel-monitor-v2/dimensions')) data = { platforms: [], groups: [], models: [] }
  else if (path.endsWith('/channel-monitor-v2/snapshot')) data = {
    config: { enabled: true, version: 1, platforms: [], group_ids: [], refresh_interval_seconds: 300 },
    coverage, metrics, health, trend: [{ bucket_start: monitorTime, metrics, health }]
  }
  else if (path.includes('/channel-monitor-v2/')) data = { coverage, items: [] }
  else if (['/groups/available', '/channels/available', '/subscriptions', '/user/mcp/grants'].includes(path)) data = []
  else if (path === '/groups/rates') data = {}
  else if (path === '/usage/dashboard/stats') data = { ...stats, by_platform: [{ platform: 'anthropic', total_requests: 120, total_tokens: 25000, total_actual_cost: stats.total_actual_cost - 1, today_actual_cost: stats.today_actual_cost - 1 }] }
  else if (path === '/usage/dashboard/trend') data = { trend: Array.from({ length: 7 }, (_, i) => ({ date: `2026-11-${String(i + 1).padStart(2, '0')}`, requests: 20 + i, input_tokens: 2000 + i * 500, output_tokens: 800 + i * 100, cache_creation_tokens: 100, cache_read_tokens: 50, total_tokens: 2950 + i * 600, cost: i + 1, actual_cost: i + 1 })) }
  else if (path === '/usage/dashboard/models') data = { models: [{ model: 'claude-sonnet-4-5', requests: 120, total_tokens: 25000, cost: 12, actual_cost: 8 }, { model: 'gpt-5', requests: 60, total_tokens: 15000, cost: 4, actual_cost: 3 }] }
  else if (path === '/user/platform-quotas') data = { platform_quotas: dashboardPreview ? [{ platform: 'anthropic', daily_limit_usd: 20, daily_usage_usd: 16, weekly_limit_usd: 0, monthly_limit_usd: null }] : [] }
  else if (dashboardPreview && path === '/usage') data = { items: Array.from({ length: 6 }, (_, i) => ({ id: i + 1, model: 'claude-sonnet-4-5', input_tokens: 1000, output_tokens: 500, total_cost: 0.08, actual_cost: 0.04, created_at: '2026-11-11T04:00:00Z' })), total: 6, pages: 1, page: 1, page_size: 20 }
  else if (path === '/usage/stats') data = stats
  else if (path === '/user/totp/status') data = { enabled: false }
  else if (path === '/user/mcp/config') data = { resource: 'https://example.test/mcp' }
  else if (path === '/user/aff') data = { user_id: currentUser.id, aff_code: 'PREVIEW', aff_count: 0, aff_quota: 0, aff_frozen_quota: 0, aff_history_quota: 0, effective_rebate_rate_percent: 10, invitees: [] }
  else if (path === '/payment/checkout-info') data = { methods: {}, plans: [], balance_disabled: false, balance_recharge_multiplier: 1, subscription_usd_to_cny_rate: 0, global_min: 1, global_max: 100, recharge_fee_rate: 0, help_text: '', help_image_url: '', stripe_publishable_key: '' }
  else if (path === '/payment/orders/refund-eligible-providers') data = { provider_instance_ids: [] }
  else if (req.method === 'POST' && path === '/admin/dashboard/users-usage') data = { stats: {} }
  else if (path.includes('/dashboard/')) data = { trend: [], ranking: [], total_actual_cost: 0, total_requests: 0, total_tokens: 0 }
  else if (path.includes('batch') || path.includes('usage-stats')) data = { stats: {}, values: {} }
  res.setHeader('Content-Type', 'application/json')
  res.end(JSON.stringify({ code: 0, message: 'OK', data }))
})
}
