import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { ClipboardList, ShieldPlus, Users as UsersIcon } from 'lucide-react'
import { api } from '../api'

export default function Users() {
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [result, setResult] = useState(null)

  const auditLogs = useQuery({
    queryKey: ['audit-logs'],
    queryFn: () => api.getAuditLogs(100),
    refetchInterval: 30000,
  })

  const provision = useMutation({
    mutationFn: () => api.provisionUser(email.trim(), name.trim()),
    onSuccess: (data) => { setResult({ ok: true, data }); setEmail(''); setName('') },
    onError: (e) => setResult({ ok: false, msg: e.message }),
  })

  const logs = auditLogs.data?.items || []

  return (
    <div className="page">
      <div className="page-inner">
        <div className="page-header">
          <div>
            <div className="eyebrow">
              <UsersIcon size={14} />
              Identity and audit
            </div>
            <div className="page-title">Provision users and inspect platform activity</div>
            <div className="page-subtitle">Create operator identities quickly, then review the audit stream to understand what changed and when.</div>
          </div>
        </div>

        <div className="hero-grid">
          <div className="card">
            <div className="card-body">
              <div className="section-title">Access lane</div>
              <div className="section-copy">Provisioning here is lightweight on purpose: just enough information to get a user into the system and audited correctly.</div>
              <div className="hero-stats">
                <div className="mini-stat">
                  <strong>{logs.length}</strong>
                  <span>recent audit entries loaded</span>
                </div>
                <div className="mini-stat">
                  <strong>{provision.isPending ? 'Busy' : 'Ready'}</strong>
                  <span>current state of the user provisioning action</span>
                </div>
              </div>
            </div>
          </div>

          <div className="card">
            <div className="card-body">
              <div className="build-pill">
                <ClipboardList size={14} />
                Audit stream
              </div>
              <div className="section-copy" style={{ marginTop: '1rem' }}>
                Audit log entries refresh every 30 seconds so operational changes stay visible without manual refresh.
              </div>
            </div>
          </div>
        </div>

        <div className="grid-2" style={{ alignItems: 'start' }}>
          <div className="card">
            <div className="card-header">
              <div>
                <div className="section-title">Provision user</div>
                <div className="section-copy">Create a user record and stamp it with the current platform admin identity.</div>
              </div>
              <ShieldPlus size={18} color="var(--accent)" />
            </div>
            <div className="card-body">
              <div className="form-group">
                <label>Email</label>
                <input value={email} onChange={e => setEmail(e.target.value)} placeholder="user@example.com" />
              </div>
              <div className="form-group">
                <label>Display Name</label>
                <input value={name} onChange={e => setName(e.target.value)} placeholder="Alice" />
              </div>
              <button
                className="btn-primary"
                style={{ width: '100%' }}
                onClick={() => provision.mutate()}
                disabled={provision.isPending || !email.trim()}
              >
                {provision.isPending ? 'Provisioning…' : 'Create user'}
              </button>
              {result ? (
                <div style={{ marginTop: '1rem' }}>
                  {result.ok
                    ? <div className="success-box">User provisioned: {result.data.email}</div>
                    : <div className="error-box">{result.msg}</div>}
                </div>
              ) : null}
            </div>
          </div>

          <div className="card">
            <div className="table-toolbar">
              <div>
                <div className="section-title">Audit log</div>
                <div className="toolbar-copy">Recent platform actions across users, resources, and deployment workflows.</div>
              </div>
              {auditLogs.isFetching ? <div className="spinner" /> : null}
            </div>

            {auditLogs.error ? (
              <div className="card-body" style={{ paddingTop: 0 }}>
                <div className="error-box">{auditLogs.error.message}</div>
              </div>
            ) : null}

            {logs.length === 0 && !auditLogs.isLoading ? (
              <div className="empty-state">No audit logs yet.</div>
            ) : (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Time</th>
                      <th>Action</th>
                      <th>Resource</th>
                      <th>Message</th>
                    </tr>
                  </thead>
                  <tbody>
                    {logs.map(log => (
                      <tr key={log.id}>
                        <td className="mono">{new Date(log.created_at).toLocaleString()}</td>
                        <td><span className="mono" style={{ color: 'var(--info)' }}>{log.action}</span></td>
                        <td className="mono">{log.resource_type}/{log.resource_id?.slice(0, 8) || '—'}</td>
                        <td style={{ color: 'var(--muted-strong)' }}>{log.message}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
