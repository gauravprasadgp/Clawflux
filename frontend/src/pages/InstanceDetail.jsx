import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, RefreshCw, RotateCcw, XCircle, Trash2 } from 'lucide-react'
import { api } from '../api'
import StatusBadge from '../components/StatusBadge'
import { useState } from 'react'

export default function InstanceDetail() {
  const { deploymentId } = useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [actionError, setActionError] = useState('')
  const [actionSuccess, setActionSuccess] = useState('')

  const dep = useQuery({
    queryKey: ['deployment', deploymentId],
    queryFn: () => api.getDeployment(deploymentId),
    refetchInterval: 8000,
  })

  const events = useQuery({
    queryKey: ['events', deploymentId],
    queryFn: () => api.getDeploymentEvents(deploymentId),
    refetchInterval: 8000,
  })

  function mutate(fn, successMsg) {
    return useMutation({
      mutationFn: fn,
      onSuccess: () => {
        setActionError('')
        setActionSuccess(successMsg)
        qc.invalidateQueries(['deployment', deploymentId])
        qc.invalidateQueries(['instances'])
      },
      onError: (e) => { setActionError(e.message); setActionSuccess('') },
    })
  }

  const retry = mutate(() => api.retryDeployment(deploymentId), 'Retry queued.')
  const cancel = mutate(() => api.cancelDeployment(deploymentId), 'Deployment cancelled.')
  const del = mutate(() => api.deleteDeployment(deploymentId), 'Deletion queued.')

  const d = dep.data
  const evts = events.data?.items || []

  return (
    <div className="page">
      <button className="btn-ghost" style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 20 }}
        onClick={() => navigate('/')}>
        <ArrowLeft size={14} /> Back
      </button>

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 20 }}>
        <div>
          <div className="page-title">Deployment Detail</div>
          <div className="mono" style={{ marginTop: 4 }}>{deploymentId}</div>
        </div>
        <button className="btn-ghost" style={{ display: 'flex', alignItems: 'center', gap: 6 }}
          onClick={() => { dep.refetch(); events.refetch() }}>
          <RefreshCw size={14} /> Refresh
        </button>
      </div>

      {actionError && <div className="error-box">{actionError}</div>}
      {actionSuccess && <div className="success-box">{actionSuccess}</div>}
      {dep.error && <div className="error-box">{dep.error.message}</div>}

      {d && (
        <>
          {/* Info + actions */}
          <div className="grid-2" style={{ marginBottom: 20 }}>
            <div className="card">
              <div className="section-title">Deployment Info</div>
              <InfoRow label="Status"><StatusBadge status={d.status} /></InfoRow>
              <InfoRow label="Version"><span className="mono">v{d.version}</span></InfoRow>
              <InfoRow label="Image"><span className="mono" style={{ wordBreak: 'break-all' }}>{d.image_ref}</span></InfoRow>
              <InfoRow label="Backend">{d.backend || '—'}</InfoRow>
              <InfoRow label="Namespace"><span className="mono">{d.backend_ref?.namespace || '—'}</span></InfoRow>
              <InfoRow label="K8s Deployment"><span className="mono">{d.backend_ref?.deployment || '—'}</span></InfoRow>
              {d.status_reason && <InfoRow label="Reason"><span style={{ color: 'var(--muted)', fontSize: 12 }}>{d.status_reason}</span></InfoRow>}
              <InfoRow label="Created">{new Date(d.created_at).toLocaleString()}</InfoRow>
            </div>

            <div className="card">
              <div className="section-title">Actions</div>
              <p style={{ color: 'var(--muted)', fontSize: 13, marginBottom: 16 }}>
                Manage this deployment lifecycle.
              </p>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                <button className="btn-primary" style={{ display: 'flex', alignItems: 'center', gap: 8 }}
                  onClick={() => retry.mutate()}
                  disabled={retry.isPending}>
                  <RotateCcw size={14} /> {retry.isPending ? 'Retrying…' : 'Retry Deployment'}
                </button>
                <button className="btn-warn" style={{ display: 'flex', alignItems: 'center', gap: 8 }}
                  onClick={() => cancel.mutate()}
                  disabled={cancel.isPending}>
                  <XCircle size={14} /> {cancel.isPending ? 'Cancelling…' : 'Cancel'}
                </button>
                <button className="btn-danger" style={{ display: 'flex', alignItems: 'center', gap: 8 }}
                  onClick={() => {
                    if (confirm('Queue this deployment for deletion?')) del.mutate()
                  }}
                  disabled={del.isPending}>
                  <Trash2 size={14} /> {del.isPending ? 'Queuing delete…' : 'Delete'}
                </button>
              </div>
            </div>
          </div>

          {/* Events */}
          <div className="card" style={{ padding: 0, overflow: 'hidden' }}>
            <div style={{ padding: '14px 20px', borderBottom: '1px solid var(--border)' }}>
              <div className="section-title" style={{ marginBottom: 0 }}>Deployment Events</div>
            </div>
            {evts.length === 0 ? (
              <div className="empty-state">No events yet.</div>
            ) : (
              <table>
                <thead>
                  <tr>
                    <th>Time</th>
                    <th>Type</th>
                    <th>Message</th>
                  </tr>
                </thead>
                <tbody>
                  {[...evts].reverse().map(e => (
                    <tr key={e.id}>
                      <td className="mono" style={{ whiteSpace: 'nowrap' }}>{new Date(e.created_at).toLocaleString()}</td>
                      <td><span className="mono" style={{ color: 'var(--info)' }}>{e.type}</span></td>
                      <td style={{ color: 'var(--muted)' }}>{e.message}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>

          {/* Config snapshot */}
          <div className="card" style={{ marginTop: 20 }}>
            <div className="section-title">Config Snapshot</div>
            <pre>{JSON.stringify(d.config_snapshot, null, 2)}</pre>
          </div>
        </>
      )}
    </div>
  )
}

function InfoRow({ label, children }) {
  return (
    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', padding: '7px 0', borderBottom: '1px solid var(--border)', fontSize: 13 }}>
      <span style={{ color: 'var(--muted)', flexShrink: 0, marginRight: 12 }}>{label}</span>
      <span style={{ textAlign: 'right' }}>{children}</span>
    </div>
  )
}
