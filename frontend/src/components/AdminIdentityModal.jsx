import { ShieldCheck } from 'lucide-react'
import { useState } from 'react'

export default function AdminIdentityModal({ onSave }) {
  const [email, setEmail] = useState(localStorage.getItem('adminEmail') || '')
  const [name, setName] = useState(localStorage.getItem('adminName') || '')

  function save() {
    localStorage.setItem('adminEmail', email.trim())
    localStorage.setItem('adminName', name.trim())
    onSave?.()
  }

  return (
    <div className="modal-backdrop">
      <div className="modal-card">
        <div className="card-body">
          <div className="sidebar-brand-kicker">
            <ShieldCheck size={14} />
            Secure Setup
          </div>
          <div className="modal-title">Set your admin identity</div>
          <p className="modal-copy">
            Clawflux sends these values with every admin request. They stay in local storage so you only need to set them once on this browser.
          </p>

          <div className="form-group" style={{ marginTop: '1.5rem' }}>
            <label>Admin Email</label>
            <input value={email} onChange={e => setEmail(e.target.value)} placeholder="admin@example.com" />
          </div>
          <div className="form-group">
            <label>Admin Name</label>
            <input value={name} onChange={e => setName(e.target.value)} placeholder="Platform operator" />
          </div>

          <div className="surface" style={{ marginBottom: '1rem' }}>
            <div className="surface-title">Why this matters</div>
            <div className="surface-copy">Audit logs, provisioning actions, and deployment operations are all tagged with this identity.</div>
          </div>

          <button className="btn-primary" style={{ width: '100%' }} onClick={save} disabled={!email.trim()}>
            Save and open console
          </button>
        </div>
      </div>
    </div>
  )
}
