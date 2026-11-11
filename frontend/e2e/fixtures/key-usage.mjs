/** Synthetic public usage responses. Never forwards keys to a real API. */
export function keyUsageFixture(scenario) {
  const today = { requests: 128, input_tokens: 48000, output_tokens: 12000, total_tokens: 60000, cache_creation_tokens: 4000, cache_read_tokens: 18000, actual_cost: 2.45 }
  const data = {
    mode: 'balance', balance: 128.5, remaining: 128.5,
    usage: { today, total: { ...today, requests: 2048, total_tokens: 960000, actual_cost: 39.2 }, rpm: 8, tpm: 3200, average_duration_ms: 845 },
    daily_usage: [{ date: '2026-11-11', requests: 128, input_tokens: 48000, output_tokens: 12000, cache_read_tokens: 18000, cache_write_tokens: 4000, cost: 9, actual_cost: 2.45 }, { date: '2026-11-10', requests: 96, input_tokens: 36000, output_tokens: 9000, cache_read_tokens: 12000, cache_write_tokens: 3000, cost: 1.85 }],
    model_stats: [{ model: 'claude-sonnet-4-5', requests: 80, input_tokens: 30000, output_tokens: 8000, cache_creation_tokens: 3000, cache_read_tokens: 12000, total_tokens: 38000, cost: 9, actual_cost: 1.75 }, { model: 'gpt-5', requests: 48, input_tokens: 18000, output_tokens: 4000, total_tokens: 22000, cost: 0.7 }],
  }
  data.daily_usage.push(...[0.72, 1.24, 0.93, 2.1, 1.35].map((cost, index) => ({ date: '2026-11-' + String(index + 5).padStart(2, '0'), requests: [42, 68, 51, 110, 74][index], input_tokens: (index + 1) * 4000, output_tokens: (index + 1) * 1000, cache_read_tokens: (index + 1) * 2000, cache_write_tokens: 1000, cost, actual_cost: cost })))
  if (scenario === 'subscription') Object.assign(data, { mode: 'subscription', planName: 'Team Plan', subscription: { daily_usage_usd: 6, daily_limit_usd: 20, weekly_usage_usd: 24, weekly_limit_usd: 100, monthly_usage_usd: 80, monthly_limit_usd: 300, expires_at: '2026-12-11T00:00:00Z' } })
  if (['quota', 'exhausted', 'expired'].includes(scenario)) Object.assign(data, { mode: 'quota_limited', status: scenario === 'quota' ? 'active' : scenario === 'exhausted' ? 'quota_exhausted' : 'expired', isValid: scenario === 'quota', quota: { limit: 100, used: scenario === 'exhausted' ? 100 : 24.5, remaining: scenario === 'exhausted' ? 0 : 75.5 }, rate_limits: [{ window: '5h', limit: 20, used: 12, reset_at: new Date(Date.now() + 7200000).toISOString() }], expires_at: '2026-12-11T00:00:00Z', days_until_expiry: 30 })
  if (scenario === 'zero') {
    data.daily_usage = [{ date: '2026-11-11', requests: 0, input_tokens: 0, output_tokens: 0, cost: 9, actual_cost: 0 }]
    data.model_stats = [{ model: 'zero-cost-model', requests: 0, cost: 9, actual_cost: 0 }]
    data.usage = { today: { ...today, input_tokens: 0, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0 }, total: {} }
  }
  if (scenario === 'many') data.model_stats = Array.from({ length: 10 }, (_, index) => ({ model: 'model-' + index, requests: index + 1, actual_cost: index + 1 }))
  if (scenario === 'minimal') return { balance: 0, remaining: 0 }
  if (scenario === 'empty') data.daily_usage = []
  return data
}

export async function serveKeyUsageFixture(req, res, requests = []) {
  const url = new URL(req.url, 'http://localhost')
  if (url.pathname !== '/v1/usage') return false
  const scenario = String(req.headers.authorization || '').replace(/^Bearer sk-preview-/, '')
  const valid = ['wallet', 'slow', 'subscription', 'quota', 'exhausted', 'expired', 'minimal', 'empty', 'zero', 'many', 'period-error'].includes(scenario)
  requests.push({ path: '/v1/usage', search: url.search, scenario: valid ? scenario : 'invalid', authorizationPresent: Boolean(req.headers.authorization) })
  if (scenario === 'slow') await new Promise(resolve => setTimeout(resolve, 1500))
  if (scenario === 'period-error' && url.searchParams.get('days') === '90') {
    await new Promise(resolve => setTimeout(resolve, 400))
    res.writeHead(503, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ message: 'Preview daily refresh failed' }))
    return true
  }
  res.writeHead(valid ? 200 : 401, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' })
  res.end(JSON.stringify(valid ? keyUsageFixture(scenario) : { error: { message: '预览密钥无效。请使用 sk-preview-wallet。' } }))
  return true
}
