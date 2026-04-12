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
    <div style={{
      position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)',
      display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100,
    }}>
      <div className="card" style={{ width: 420 }}>
        <div className="section-title">Set Admin Identity</div>
        <p style={{ color: 'var(--muted)', fontSize: 13, marginBottom: 16 }}>
          Enter your admin email to authenticate requests. This is saved in localStorage.
        </p>
        <div className="form-group">
          <label>Admin Email</label>
          <input value={email} onChange={e => setEmail(e.target.value)} placeholder="admin@example.com" />
        </div>
        <div className="form-group">
          <label>Admin Name (optional)</label>
          <input value={name} onChange={e => setName(e.target.value)} placeholder="Admin" />
        </div>
        <button className="btn-primary" style={{ width: '100%' }} onClick={save}>Save & Continue</button>
      </div>
    </div>
  )
}
