import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { AlertTriangle, ArrowRight, Boxes, CheckCircle2, RefreshCw, Rocket, ShieldAlert, Users, XCircle } from 'lucide-react'
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

  const preflight = useQuery({
    queryKey: ['preflight'],
    queryFn: api.getPreflight,
    refetchInterval: 20000,
  })

  const stats = summary.data || {}
  const items = instances.data?.items || []
  const checks = preflight.data?.checks || []

  return (
    <div className="page">
      <div className="page-inner">
        <div className="hero-grid">
          <div className="card hero-card">
            <div className="card-body">
              <div>
                <div className="eyebrow">
                  <Boxes size={14} />
                  Platform overview
                </div>
                <div className="page-title">Run the whole OpenClaw fleet from one cockpit.</div>
                <div className="page-subtitle hero-lead">
                  Monitor deployments, catch failures quickly, and jump from fleet health to instance-level action without leaving the console.
                </div>
              </div>

              <div>
                <div className="hero-actions">
                  <button className="btn-primary" onClick={() => navigate('/deploy')}>
                    <Rocket size={16} />
                    New deployment
                  </button>
                  <button className="btn-ghost" onClick={() => navigate('/users')}>
                    <Users size={16} />
                    Manage users
                  </button>
                </div>

                <div className="hero-stats">
                  <div className="mini-stat">
                    <strong>{items.length}</strong>
                    <span>tracked instances in the admin view</span>
                  </div>
                  <div className="mini-stat">
                    <strong>{stats.repository_driver || 'memory'}</strong>
                    <span>active repository driver</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="stack">
            <div className="card">
              <div className="card-body">
                <div className={`preflight-status preflight-status-${preflight.data?.status || 'loading'}`}>
                  <PreflightIcon status={preflight.data?.status} />
                  {preflight.data?.status || 'checking'}
                </div>
                <div className="section-title" style={{ marginTop: '1rem' }}>Launch readiness</div>
                <div className="section-copy">Runtime checks highlight the setup issues that usually stop first deployments.</div>

                {preflight.error ? (
                  <div className="error-box" style={{ marginTop: '1rem' }}>{preflight.error.message}</div>
                ) : (
                  <div className="preflight-list">
                    {checks.slice(0, 5).map(check => (
                      <div key={check.id} className={`preflight-row preflight-row-${check.status}`}>
                        <PreflightIcon status={check.status} />
                        <div>
                          <strong>{check.label}</strong>
                          <span>{check.message}</span>
                        </div>
                      </div>
                    ))}
                    {checks.length === 0 ? <div className="field-hint">Checking runtime dependencies...</div> : null}
                  </div>
                )}

                <div className="runtime-strip">
                  <span>{preflight.data?.backend || 'backend...'}</span>
                  <span>{preflight.data?.repository_driver || stats.repository_driver || 'repository...'}</span>
                </div>
              </div>
            </div>

            <div className="card">
              <div className="card-body">
                <div className="section-title">Attention lane</div>
                <div className="section-copy">Failed deployments are kept visible so operators can retry, cancel, or delete them quickly.</div>
                <div className="stat-value stat-value-danger" style={{ marginTop: '1rem' }}>
                  {stats.failed_deployments ?? '—'}
                </div>
                <div className="stat-footnote">deployments currently marked failed</div>
                <div className="actions-row" style={{ marginTop: '1rem' }}>
                  <button className="btn-ghost" onClick={() => { summary.refetch(); instances.refetch(); preflight.refetch() }}>
                    <RefreshCw size={14} />
                    Refresh now
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="grid-4" style={{ marginBottom: '1rem' }}>
          {[
            { label: 'Users', value: stats.users ?? '—', icon: Users, note: 'provisioned operator accounts' },
            { label: 'Apps', value: stats.apps ?? '—', icon: Boxes, note: 'application definitions on platform' },
            { label: 'Deployments', value: stats.deployments ?? '—', icon: Rocket, note: 'deployment records created' },
            { label: 'Failed', value: stats.failed_deployments ?? '—', icon: ShieldAlert, note: 'requiring intervention', danger: true },
          ].map(({ label, value, icon: Icon, note, danger }) => (
            <div key={label} className="card stat-card">
              <div className="card-body">
                <div className="stat-kicker">
                  <Icon size={14} />
                  {label}
                </div>
                <div className={`stat-value${danger && Number(value) > 0 ? ' stat-value-danger' : ''}`}>{value}</div>
                <div className="stat-footnote">{note}</div>
              </div>
            </div>
          ))}
        </div>

        <div className="card">
          <div className="table-toolbar">
            <div>
              <div className="section-title">OpenClaw instances</div>
              <div className="toolbar-copy">Browse every known instance, jump into deployment detail, and keep namespaces visible at a glance.</div>
            </div>
            {instances.isFetching ? <div className="spinner" /> : null}
          </div>

          {instances.error && (
            <div className="card-body" style={{ paddingTop: 0 }}>
              <div className="error-box">{instances.error.message}</div>
            </div>
          )}

          {items.length === 0 && !instances.isLoading ? (
            <div className="empty-state">No instances deployed yet.</div>
          ) : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>User</th>
                    <th>App</th>
                    <th>Namespace</th>
                    <th>Status</th>
                    <th>Version</th>
                    <th>Last deployed</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {items.map(inst => (
                    <tr key={inst.app.id}>
                      <td>{inst.user_email || <span className="mono">—</span>}</td>
                      <td>
                        <div style={{ fontWeight: 700 }}>{inst.app.name}</div>
                        <div className="mono">{inst.app.slug}</div>
                      </td>
                      <td className="mono">
                        {inst.deployment?.backend_ref?.namespace || '—'}
                      </td>
                      <td>
                        <StatusBadge status={inst.deployment?.status} />
                      </td>
                      <td className="mono">
                        {inst.deployment ? `v${inst.deployment.version}` : '—'}
                      </td>
                      <td className="mono">
                        {inst.deployment ? new Date(inst.deployment.created_at).toLocaleString() : '—'}
                      </td>
                      <td>
                        {inst.deployment ? (
                          <button
                            className="btn-ghost"
                            onClick={() => navigate(`/instances/${inst.deployment.id}`)}
                          >
                            View detail
                            <ArrowRight size={14} />
                          </button>
                        ) : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function PreflightIcon({ status }) {
  if (status === 'pass' || status === 'ready') return <CheckCircle2 size={15} />
  if (status === 'fail' || status === 'blocked') return <XCircle size={15} />
  return <AlertTriangle size={15} />
}
