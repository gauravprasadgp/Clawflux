import { useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
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
      <div className="page-title">Users</div>
      <div className="page-subtitle">Provision users and review platform audit logs</div>

      <div style={{ display: 'grid', gridTemplateColumns: '380px 1fr', gap: 20, alignItems: 'start' }}>
        {/* Provision form */}
        <div className="card">
          <div className="section-title">Provision User</div>
          <div className="form-group">
            <label>Email</label>
            <input value={email} onChange={e => setEmail(e.target.value)} placeholder="user@example.com" />
          </div>
          <div className="form-group">
            <label>Display Name (optional)</label>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="Alice" />
          </div>
          <button className="btn-primary" style={{ width: '100%' }}
            onClick={() => provision.mutate()}
            disabled={provision.isPending || !email.trim()}>
            {provision.isPending ? 'Provisioning…' : 'Create / Provision User'}
          </button>
          {result && (
            <div style={{ marginTop: 14 }}>
              {result.ok
                ? <div className="success-box">User provisioned: {result.data.email}</div>
                : <div className="error-box">{result.msg}</div>}
            </div>
          )}
        </div>

        {/* Audit log */}
        <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
          <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--border)', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div className="section-title" style={{ marginBottom: 0 }}>Audit Log</div>
            {auditLogs.isFetching && <div className="spinner" />}
          </div>
          {auditLogs.error && <div className="error-box" style={{ margin: 12 }}>{auditLogs.error.message}</div>}
          {logs.length === 0 && !auditLogs.isLoading
            ? <div className="empty-state">No audit logs yet.</div>
            : (
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
                      <td className="mono" style={{ whiteSpace: 'nowrap', fontSize: 11 }}>
                        {new Date(log.created_at).toLocaleString()}
                      </td>
                      <td><span className="mono" style={{ color: 'var(--info)' }}>{log.action}</span></td>
                      <td className="mono" style={{ fontSize: 11 }}>{log.resource_type}/{log.resource_id?.slice(0, 8)}…</td>
                      <td style={{ color: 'var(--muted)', fontSize: 12 }}>{log.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
        </div>
      </div>
    </div>
  )
}
