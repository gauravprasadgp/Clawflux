import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle, ArrowLeft, RefreshCw, RotateCcw, Trash2, XCircle } from 'lucide-react'
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
      <div className="page-inner">
        <button className="btn-ghost" style={{ marginBottom: '1rem' }} onClick={() => navigate('/')}>
          <ArrowLeft size={14} />
          Back to dashboard
        </button>

        <div className="page-header page-header-compact">
          <div>
            <div className="eyebrow">
              <AlertTriangle size={14} />
              Deployment lifecycle
            </div>
            <div className="page-title">Deployment detail</div>
            <div className="page-subtitle mono">{deploymentId}</div>
          </div>
          <div className="page-actions">
            <button className="btn-ghost" onClick={() => { dep.refetch(); events.refetch() }}>
              <RefreshCw size={14} />
              Refresh
            </button>
          </div>
        </div>

        {actionError && <div className="error-box" style={{ marginBottom: '1rem' }}>{actionError}</div>}
        {actionSuccess && <div className="success-box" style={{ marginBottom: '1rem' }}>{actionSuccess}</div>}
        {dep.error && <div className="error-box" style={{ marginBottom: '1rem' }}>{dep.error.message}</div>}

        {d && (
          <div className="stack">
            <div className="split-detail">
              <div className="card">
                <div className="card-header">
                  <div>
                    <div className="section-title">Deployment info</div>
                    <div className="section-copy">Core metadata, rollout target, and backend reference for this deployment.</div>
                  </div>
                  <StatusBadge status={d.status} />
                </div>
                <div className="card-body">
                  <div className="info-list">
                    <InfoRow label="Version"><span className="mono">v{d.version}</span></InfoRow>
                    <InfoRow label="Image"><span className="mono" style={{ wordBreak: 'break-all' }}>{d.image_ref}</span></InfoRow>
                    <InfoRow label="Backend">{d.backend || '—'}</InfoRow>
                    <InfoRow label="Namespace"><span className="mono">{d.backend_ref?.namespace || '—'}</span></InfoRow>
                    <InfoRow label="K8s Deployment"><span className="mono">{d.backend_ref?.deployment || '—'}</span></InfoRow>
                    {d.status_reason ? <InfoRow label="Reason"><span style={{ color: 'var(--muted-strong)' }}>{d.status_reason}</span></InfoRow> : null}
                    <InfoRow label="Created"><span className="mono">{new Date(d.created_at).toLocaleString()}</span></InfoRow>
                  </div>
                </div>
              </div>

              <div className="card">
                <div className="card-header">
                  <div>
                    <div className="section-title">Actions</div>
                    <div className="section-copy">Queue recovery or cleanup actions for this deployment.</div>
                  </div>
                </div>
                <div className="card-body">
                  <div className="stack">
                    <button className="btn-primary" onClick={() => retry.mutate()} disabled={retry.isPending}>
                      <RotateCcw size={14} />
                      {retry.isPending ? 'Retrying…' : 'Retry deployment'}
                    </button>
                    <button className="btn-warn" onClick={() => cancel.mutate()} disabled={cancel.isPending}>
                      <XCircle size={14} />
                      {cancel.isPending ? 'Cancelling…' : 'Cancel deployment'}
                    </button>
                    <button
                      className="btn-danger"
                      onClick={() => {
                        if (confirm('Queue this deployment for deletion?')) del.mutate()
                      }}
                      disabled={del.isPending}
                    >
                      <Trash2 size={14} />
                      {del.isPending ? 'Queuing delete…' : 'Delete deployment'}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div className="card">
              <div className="table-toolbar">
                <div>
                  <div className="section-title">Deployment events</div>
                  <div className="toolbar-copy">Chronological event trail for this deployment record.</div>
                </div>
              </div>
              {evts.length === 0 ? (
                <div className="empty-state">No events yet.</div>
              ) : (
                <div className="timeline">
                  {[...evts].reverse().map(e => (
                    <div key={e.id} className="timeline-item">
                      <div className="timeline-head">
                        <div className="timeline-title mono">{e.type}</div>
                        <div className="mono">{new Date(e.created_at).toLocaleString()}</div>
                      </div>
                      <div className="timeline-copy">{e.message}</div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="card">
              <div className="card-header">
                <div>
                  <div className="section-title">Config snapshot</div>
                  <div className="section-copy">Raw deployment configuration captured for this rollout.</div>
                </div>
              </div>
              <div className="card-body">
                <pre>{JSON.stringify(d.config_snapshot, null, 2)}</pre>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function InfoRow({ label, children }) {
  return (
    <div className="info-row">
      <span className="info-row-label">{label}</span>
      <span className="info-row-value">{children}</span>
    </div>
  )
}
