const BASE = ''

function adminHeaders() {
  const email = localStorage.getItem('adminEmail') || ''
  const name = localStorage.getItem('adminName') || ''
  const headers = {
    'Content-Type': 'application/json',
    'X-Platform-Admin': 'true',
  }
  if (email) headers['X-User-Email'] = email
  if (name) headers['X-User-Name'] = name
  return headers
}

async function request(method, path, body) {
  const res = await fetch(BASE + path, {
    method,
    headers: adminHeaders(),
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let data = text
  try {
    data = JSON.parse(text)
  } catch {
    data = text
  }
  if (!res.ok) {
    const msg = (data && data.message) || (data && data.error) || text || res.statusText
    throw new Error(msg)
  }
  return data
}

export const api = {
  getSummary: () => request('GET', '/v1/admin/summary'),
  getInstances: () => request('GET', '/v1/admin/instances'),
  getPreflight: () => request('GET', '/v1/admin/preflight'),
  getAuditLogs: (limit = 50) => request('GET', `/v1/admin/audit-logs?limit=${limit}`),

  provisionUser: (email, displayName) =>
    request('POST', '/v1/admin/users', { email, display_name: displayName }),

  deployOpenClaw: (payload) =>
    request('POST', '/v1/admin/openclaw/deploy', payload),

  getDeployment: (id) => request('GET', `/v1/deployments/${id}`),
  getDeploymentEvents: (id) => request('GET', `/v1/deployments/${id}/events`),
  retryDeployment: (id) => request('POST', `/v1/deployments/${id}/retry`),
  cancelDeployment: (id) => request('POST', `/v1/deployments/${id}/cancel`),
  deleteDeployment: (id) => request('POST', `/v1/deployments/${id}/delete`),

  getApp: (id) => request('GET', `/v1/apps/${id}`),
  listDeployments: (appId) => request('GET', `/v1/apps/${appId}/deployments`),
}
