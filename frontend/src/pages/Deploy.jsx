import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api } from '../api'

const defaults = {
  user_email: '',
  user_name: '',
  app_name: 'openclaw',
  app_slug: 'openclaw',
  image: 'ghcr.io/openclaw/openclaw:latest',
  replicas: 1,
  gateway_port: 18789,
  gateway_bind_address: '0.0.0.0',
  domain: '',
  gateway_token: '',
  existing_secret_name: '',
  workspace_storage: '10Gi',
  provider_api_keys: '{"OPENAI_API_KEY":""}',
  extra_env: '{}',
  agents_markdown: '',
  settings_json: '',
  public: true,
}

export default function Deploy() {
  const navigate = useNavigate()
  const [form, setForm] = useState(defaults)
  const [result, setResult] = useState(null)

  const deploy = useMutation({
    mutationFn: () => {
      const payload = {
        ...form,
        replicas: Number(form.replicas) || 1,
        gateway_port: Number(form.gateway_port) || 18789,
        provider_api_keys: JSON.parse(form.provider_api_keys || '{}'),
        extra_env: JSON.parse(form.extra_env || '{}'),
      }
      return api.deployOpenClaw(payload)
    },
    onSuccess: (data) => setResult({ ok: true, data }),
    onError: (e) => setResult({ ok: false, msg: e.message }),
  })

  function field(key, label, opts = {}) {
    return (
      <div className="form-group">
        <label>{label}</label>
        <input
          type={opts.type || 'text'}
          value={form[key]}
          placeholder={opts.placeholder}
          onChange={e => setForm(f => ({ ...f, [key]: opts.type === 'number' ? e.target.value : e.target.value }))}
        />
      </div>
    )
  }

  return (
    <div className="page">
      <div className="page-title">Deploy OpenClaw</div>
      <div className="page-subtitle">Provision a new OpenClaw instance for a user on Kubernetes</div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 20 }}>
        {/* Left column */}
        <div>
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="section-title">Target User</div>
            {field('user_email', 'User Email', { placeholder: 'user@example.com' })}
            {field('user_name', 'Display Name', { placeholder: 'Alice' })}
          </div>

          <div className="card" style={{ marginBottom: 16 }}>
            <div className="section-title">App</div>
            <div className="grid-2">
              {field('app_name', 'App Name')}
              {field('app_slug', 'App Slug')}
            </div>
            {field('image', 'Container Image')}
            <div className="grid-2">
              {field('replicas', 'Replicas', { type: 'number' })}
              {field('gateway_port', 'Gateway Port', { type: 'number' })}
            </div>
            <div className="grid-2">
              {field('gateway_bind_address', 'Gateway Bind Address')}
              {field('domain', 'Domain (optional)', { placeholder: 'openclaw.example.com' })}
            </div>
            {field('workspace_storage', 'Workspace Storage')}
          </div>

          <div className="card">
            <div className="section-title">Secrets & Config</div>
            {field('gateway_token', 'Gateway Token (optional)', { placeholder: 'super-secret-token' })}
            {field('existing_secret_name', 'Existing K8s Secret Name (optional)', { placeholder: 'openclaw-secrets' })}
            <div className="form-group">
              <label>Provider API Keys (JSON)</label>
              <textarea rows={3} value={form.provider_api_keys}
                onChange={e => setForm(f => ({ ...f, provider_api_keys: e.target.value }))}
                style={{ fontFamily: 'monospace', fontSize: 12 }} />
            </div>
            <div className="form-group">
              <label>Extra Env (JSON)</label>
              <textarea rows={2} value={form.extra_env}
                onChange={e => setForm(f => ({ ...f, extra_env: e.target.value }))}
                style={{ fontFamily: 'monospace', fontSize: 12 }} />
            </div>
          </div>
        </div>

        {/* Right column */}
        <div>
          <div className="card" style={{ marginBottom: 16 }}>
            <div className="section-title">OpenClaw Config (optional)</div>
            <div className="form-group">
              <label>AGENTS.md content</label>
              <textarea rows={5} value={form.agents_markdown}
                placeholder="# AGENTS.md..."
                onChange={e => setForm(f => ({ ...f, agents_markdown: e.target.value }))} />
            </div>
            <div className="form-group">
              <label>settings.json content</label>
              <textarea rows={4} value={form.settings_json}
                placeholder='{"default_model":"gpt-5.4-mini"}'
                onChange={e => setForm(f => ({ ...f, settings_json: e.target.value }))}
                style={{ fontFamily: 'monospace', fontSize: 12 }} />
            </div>
            <div className="form-group">
              <label style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
                <input type="checkbox" style={{ width: 'auto' }}
                  checked={form.public}
                  onChange={e => setForm(f => ({ ...f, public: e.target.checked }))} />
                Expose via public ingress
              </label>
            </div>
          </div>

          <div className="card">
            <div className="section-title">Admin Identity</div>
            <div className="form-group">
              <label>Admin Email</label>
              <input value={localStorage.getItem('adminEmail') || ''}
                onChange={e => { localStorage.setItem('adminEmail', e.target.value); }}
                placeholder="admin@example.com" />
            </div>
            <div className="form-group">
              <label>Admin Name</label>
              <input value={localStorage.getItem('adminName') || ''}
                onChange={e => { localStorage.setItem('adminName', e.target.value); }}
                placeholder="Admin" />
            </div>
          </div>

          <button className="btn-primary"
            style={{ width: '100%', padding: '12px', fontSize: 14, marginTop: 16 }}
            onClick={() => deploy.mutate()}
            disabled={deploy.isPending}>
            {deploy.isPending ? 'Deploying…' : '🚀 Deploy OpenClaw'}
          </button>

          {result && (
            <div style={{ marginTop: 16 }}>
              {result.ok ? (
                <>
                  <div className="success-box">
                    Deployment created!{' '}
                    <button className="btn-ghost" style={{ padding: '2px 8px', fontSize: 12, marginLeft: 8 }}
                      onClick={() => navigate(`/instances/${result.data.deployment?.id}`)}>
                      View →
                    </button>
                  </div>
                  <pre>{JSON.stringify(result.data, null, 2)}</pre>
                </>
              ) : (
                <div className="error-box">{result.msg}</div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
