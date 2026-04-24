import { useMemo, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { KeyRound, Rocket, ServerCog, ShieldCheck, Sparkles } from 'lucide-react'
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

const profiles = {
  local: {
    label: 'Local',
    description: 'Small demo instance with private routing.',
    values: {
      replicas: 1,
      workspace_storage: '5Gi',
      public: false,
      domain: '',
      image: 'ghcr.io/openclaw/openclaw:latest',
    },
  },
  team: {
    label: 'Team',
    description: 'Default shared workspace for a real user.',
    values: {
      replicas: 1,
      workspace_storage: '10Gi',
      public: true,
      image: 'ghcr.io/openclaw/openclaw:latest',
    },
  },
  production: {
    label: 'Production',
    description: 'Larger persistent workspace with two replicas.',
    values: {
      replicas: 2,
      workspace_storage: '25Gi',
      public: true,
      image: 'ghcr.io/openclaw/openclaw:latest',
    },
  },
}

export default function Deploy() {
  const navigate = useNavigate()
  const [form, setForm] = useState(defaults)
  const [result, setResult] = useState(null)
  const [profile, setProfile] = useState('team')

  const providerKeys = useMemo(() => parseJSONBlock(form.provider_api_keys, 'Provider API Keys'), [form.provider_api_keys])
  const extraEnv = useMemo(() => parseJSONBlock(form.extra_env, 'Extra Env'), [form.extra_env])
  const settings = useMemo(() => parseOptionalJSONBlock(form.settings_json, 'settings.json content'), [form.settings_json])
  const validationErrors = [
    !form.user_email.trim() ? 'User email is required.' : '',
    providerKeys.error,
    extraEnv.error,
    settings.error,
  ].filter(Boolean)

  const deploy = useMutation({
    mutationFn: () => {
      const payload = {
        ...form,
        replicas: Number(form.replicas) || 1,
        gateway_port: Number(form.gateway_port) || 18789,
        provider_api_keys: providerKeys.value,
        extra_env: extraEnv.value,
      }
      return api.deployOpenClaw(payload)
    },
    onSuccess: (data) => setResult({ ok: true, data }),
    onError: (e) => setResult({ ok: false, msg: e.message }),
  })
  const canDeploy = validationErrors.length === 0 && !deploy.isPending

  function applyProfile(key) {
    const next = profiles[key]
    setProfile(key)
    setForm(f => ({ ...f, ...next.values }))
  }

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
        {opts.hint ? <div className="field-hint">{opts.hint}</div> : null}
      </div>
    )
  }

  return (
    <div className="page">
      <div className="page-inner">
        <div className="page-header">
          <div>
            <div className="eyebrow">
              <Rocket size={14} />
              Guided rollout
            </div>
            <div className="page-title">Deploy a new OpenClaw instance</div>
            <div className="page-subtitle">Prepare the target user, workload configuration, secrets, and runtime options in one controlled flow.</div>
          </div>
        </div>

        <div className="hero-grid" style={{ marginBottom: '1rem' }}>
          <div className="card">
            <div className="card-body">
              <div className="section-title">What this creates</div>
              <div className="section-copy">
                Clawflux provisions the app record, creates or reuses secrets, and queues deployment work for the configured Kubernetes backend.
              </div>

              <div className="hero-stats">
                <div className="mini-stat">
                  <strong>{form.app_slug}</strong>
                  <span>application slug to track the deployment</span>
                </div>
                <div className="mini-stat">
                  <strong>{form.replicas || 1}</strong>
                  <span>replica target requested for rollout</span>
                </div>
                <div className="mini-stat">
                  <strong>{form.gateway_port || 18789}</strong>
                  <span>gateway port exposed by the container</span>
                </div>
                <div className="mini-stat">
                  <strong>{form.public ? 'Public' : 'Private'}</strong>
                  <span>ingress exposure mode for this instance</span>
                </div>
              </div>
            </div>
          </div>

          <div className="stack">
            <div className="card">
              <div className="card-body">
                <div className="build-pill">
                  <Sparkles size={14} />
                  Launch profile
                </div>
                <div className="profile-selector">
                  {Object.entries(profiles).map(([key, item]) => (
                    <button
                      key={key}
                      className={`profile-option${profile === key ? ' active' : ''}`}
                      onClick={() => applyProfile(key)}
                    >
                      <strong>{item.label}</strong>
                      <span>{item.description}</span>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <div className="card">
              <div className="card-body">
                <div className="section-title">Admin identity</div>
                <div className="section-copy">Deployment requests inherit the browser’s stored admin headers.</div>
                <div className="surface" style={{ marginTop: '1rem' }}>
                  <div className="surface-title">{localStorage.getItem('adminEmail') || 'No admin email set'}</div>
                  <div className="surface-copy">{localStorage.getItem('adminName') || 'Platform operator'}</div>
                </div>
                {validationErrors.length > 0 ? (
                  <div className="validation-list">
                    {validationErrors.map(error => <div key={error}>{error}</div>)}
                  </div>
                ) : (
                  <div className="success-box" style={{ marginTop: '1rem' }}>Deployment payload is ready to submit.</div>
                )}
              </div>
            </div>
          </div>
        </div>

        <div className="grid-2" style={{ alignItems: 'start' }}>
          <div className="stack">
            <div className="card">
              <div className="card-header">
                <div>
                  <div className="section-title">Target user</div>
                  <div className="section-copy">Define who this workspace belongs to and how it should be labeled in the console.</div>
                </div>
                <UsersBadge />
              </div>
              <div className="card-body">
                {field('user_email', 'User Email', { placeholder: 'user@example.com', hint: 'This user will own the deployment and related app record.' })}
                {field('user_name', 'Display Name', { placeholder: 'Alice', hint: 'Optional friendly name shown alongside the email.' })}
              </div>
            </div>

            <div className="card">
              <div className="card-header">
                <div>
                  <div className="section-title">Application workload</div>
                  <div className="section-copy">Core settings for image, replicas, network, and storage sizing.</div>
                </div>
                <ServerCog size={18} color="var(--info)" />
              </div>
              <div className="card-body">
                <div className="grid-2">
                  {field('app_name', 'App Name')}
                  {field('app_slug', 'App Slug')}
                </div>
                {field('image', 'Container Image', { hint: 'Prefer a pinned tag instead of `latest` for repeatable deployments.' })}
                <div className="grid-2">
                  {field('replicas', 'Replicas', { type: 'number' })}
                  {field('gateway_port', 'Gateway Port', { type: 'number' })}
                </div>
                <div className="grid-2">
                  {field('gateway_bind_address', 'Gateway Bind Address')}
                  {field('domain', 'Domain (optional)', { placeholder: 'openclaw.example.com', hint: 'Leave blank if the platform will assign or skip ingress domain routing.' })}
                </div>
                {field('workspace_storage', 'Workspace Storage', { hint: 'Persistent volume request for the workspace storage claim.' })}
              </div>
            </div>

            <div className="card">
              <div className="card-header">
                <div>
                  <div className="section-title">Secrets and runtime config</div>
                  <div className="section-copy">Choose whether to inline credentials or reference an existing secret.</div>
                </div>
                <KeyRound size={18} color="var(--warn)" />
              </div>
              <div className="card-body">
                {field('gateway_token', 'Gateway Token (optional)', { placeholder: 'super-secret-token' })}
                {field('existing_secret_name', 'Existing K8s Secret Name (optional)', { placeholder: 'openclaw-secrets' })}
                <div className="form-group">
                  <label>Provider API Keys (JSON)</label>
                  <textarea
                    rows={4}
                    className="mono"
                    value={form.provider_api_keys}
                    onChange={e => setForm(f => ({ ...f, provider_api_keys: e.target.value }))}
                  />
                  <div className="field-hint">Example: {`{"OPENAI_API_KEY":"..."}`}</div>
                  {providerKeys.error ? <div className="field-error">{providerKeys.error}</div> : null}
                </div>
                <div className="form-group">
                  <label>Extra Env (JSON)</label>
                  <textarea
                    rows={3}
                    className="mono"
                    value={form.extra_env}
                    onChange={e => setForm(f => ({ ...f, extra_env: e.target.value }))}
                  />
                  {extraEnv.error ? <div className="field-error">{extraEnv.error}</div> : null}
                </div>
              </div>
            </div>
          </div>

          <div className="stack">
            <div className="card">
              <div className="card-header">
                <div>
                  <div className="section-title">OpenClaw config</div>
                  <div className="section-copy">Optional content injected into the deployed workspace for agent behavior and app settings.</div>
                </div>
                <ShieldCheck size={18} color="var(--accent)" />
              </div>
              <div className="card-body">
                <div className="form-group">
                  <label>AGENTS.md content</label>
                  <textarea
                    rows={8}
                    value={form.agents_markdown}
                    placeholder="# AGENTS.md..."
                    onChange={e => setForm(f => ({ ...f, agents_markdown: e.target.value }))}
                  />
                </div>
                <div className="form-group">
                  <label>settings.json content</label>
                  <textarea
                    rows={6}
                    className="mono"
                    value={form.settings_json}
                    placeholder='{"default_model":"gpt-5.4-mini"}'
                    onChange={e => setForm(f => ({ ...f, settings_json: e.target.value }))}
                  />
                  {settings.error ? <div className="field-error">{settings.error}</div> : null}
                </div>
                <label className="checkbox-row">
                  <input
                    type="checkbox"
                    checked={form.public}
                    onChange={e => setForm(f => ({ ...f, public: e.target.checked }))}
                  />
                  <span>
                    <strong>Expose via public ingress</strong>
                    <div className="field-hint">Disable this for internal-only workspace access.</div>
                  </span>
                </label>
              </div>
            </div>

            <div className="card">
              <div className="card-body">
                <div className="section-title">Request summary</div>
                <div className="section-copy">This is the exact shape of the deployment payload after numeric fields and JSON blocks are normalized.</div>
                <pre style={{ marginTop: '1rem' }}>
{JSON.stringify({
  ...form,
  replicas: Number(form.replicas) || 1,
  gateway_port: Number(form.gateway_port) || 18789,
  provider_api_keys: providerKeys.value,
  extra_env: extraEnv.value,
}, null, 2)}
                </pre>
              </div>
            </div>

            <button
              className="btn-primary"
              style={{ width: '100%' }}
              onClick={() => deploy.mutate()}
              disabled={!canDeploy}
            >
              <Rocket size={16} />
              {deploy.isPending ? 'Deploying…' : 'Deploy OpenClaw'}
            </button>

            {result && (
              <div>
                {result.ok ? (
                  <div className="stack">
                    <div className="success-box">
                      Deployment created successfully.
                    </div>
                    <button className="btn-ghost" onClick={() => navigate(`/instances/${result.data.deployment?.id}`)}>
                      View deployment detail
                    </button>
                    <pre>{JSON.stringify(result.data, null, 2)}</pre>
                  </div>
                ) : (
                  <div className="error-box">{result.msg}</div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function UsersBadge() {
  return <div className="build-pill">User target</div>
}

function parseJSONBlock(raw, label) {
  try {
    const value = JSON.parse(raw || '{}')
    if (!value || Array.isArray(value) || typeof value !== 'object') {
      return { value: {}, error: `${label} must be a JSON object.` }
    }
    return { value, error: '' }
  } catch (error) {
    return { value: {}, error: `${label} is invalid JSON: ${error.message}` }
  }
}

function parseOptionalJSONBlock(raw, label) {
  if (!raw.trim()) return { value: null, error: '' }
  return parseJSONBlock(raw, label)
}
