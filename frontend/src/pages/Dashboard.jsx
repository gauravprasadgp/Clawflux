import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { RefreshCw } from 'lucide-react'
import { api } from '../api'
import StatusBadge from '../components/StatusBadge'

export default function Dashboard() {
  const navigate = useNavigate()

  const summary = useQuery({
    queryKey: ['summary'],
    queryFn: api.getSummary,
    refetchInterval: 15000,
  })

  const instances = useQuery({
    queryKey: ['instances'],
    queryFn: api.getInstances,
    refetchInterval: 10000,
  })

  const stats = summary.data || {}
  const items = instances.data?.items || []

  return (
    <div className="page">
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
        <div>
          <div className="page-title">Dashboard</div>
          <div className="page-subtitle">Platform overview and all deployed OpenClaw instances</div>
        </div>
        <button className="btn-ghost" style={{ display: 'flex', alignItems: 'center', gap: 6 }}
          onClick={() => { summary.refetch(); instances.refetch() }}>
          <RefreshCw size={14} /> Refresh
        </button>
      </div>

      {/* Stats */}
      <div className="grid-4" style={{ marginBottom: 28 }}>
        {[
          { label: 'Users', value: stats.users ?? '—' },
          { label: 'Apps', value: stats.apps ?? '—' },
          { label: 'Deployments', value: stats.deployments ?? '—' },
          { label: 'Failed', value: stats.failed_deployments ?? '—', danger: true },
        ].map(s => (
          <div key={s.label} className="card stat-card">
            <div className="stat-value" style={s.danger && s.value > 0 ? { color: 'var(--danger)' } : {}}>{s.value}</div>
            <div className="stat-label">{s.label}</div>
          </div>
        ))}
      </div>

      {/* Instances table */}
      <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
        <div style={{ padding: '16px 20px', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div className="section-title" style={{ marginBottom: 0 }}>OpenClaw Instances</div>
          {instances.isFetching && <div className="spinner" />}
        </div>

        {instances.error && (
          <div className="error-box" style={{ margin: 16 }}>{instances.error.message}</div>
        )}

        {items.length === 0 && !instances.isLoading ? (
          <div className="empty-state">No instances deployed yet.</div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>User</th>
                <th>App</th>
                <th>Namespace</th>
                <th>Status</th>
                <th>Version</th>
                <th>Last Deployed</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {items.map(inst => (
                <tr key={inst.app.id}>
                  <td>{inst.user_email || <span style={{ color: 'var(--muted)' }}>—</span>}</td>
                  <td>
                    <div style={{ fontWeight: 600 }}>{inst.app.name}</div>
                    <div className="mono">{inst.app.slug}</div>
                  </td>
                  <td className="mono">
                    {inst.deployment?.backend_ref?.namespace || <span style={{ color: 'var(--muted)' }}>—</span>}
                  </td>
                  <td>
                    <StatusBadge status={inst.deployment?.status} />
                  </td>
                  <td className="mono">
                    {inst.deployment ? `v${inst.deployment.version}` : '—'}
                  </td>
                  <td style={{ color: 'var(--muted)', fontSize: 12 }}>
                    {inst.deployment
                      ? new Date(inst.deployment.created_at).toLocaleString()
                      : '—'}
                  </td>
                  <td>
                    {inst.deployment && (
                      <button className="btn-ghost" style={{ padding: '5px 12px', fontSize: 12 }}
                        onClick={() => navigate(`/instances/${inst.deployment.id}`)}>
                        View
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {stats.repository_driver && (
        <div style={{ color: 'var(--muted)', fontSize: 11, marginTop: 12 }}>
          Storage: {stats.repository_driver}
        </div>
      )}
    </div>
  )
}
