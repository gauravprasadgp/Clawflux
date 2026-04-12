package http

import "net/http"

const adminUIHTML = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Clawflux Admin</title>
    <style>
      :root {
        --bg: #0c111d;
        --panel: #151c2d;
        --text: #e6edf7;
        --muted: #93a1bf;
        --accent: #34d399;
        --danger: #f87171;
        --border: #2a3550;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial, sans-serif;
        background: radial-gradient(1200px 600px at 100% -10%, #1e2a46 0%, var(--bg) 50%);
        color: var(--text);
      }
      .container {
        max-width: 980px;
        margin: 24px auto;
        padding: 0 16px 24px;
      }
      h1 { margin: 0 0 8px; }
      .subtitle { color: var(--muted); margin-bottom: 20px; }
      .grid {
        display: grid;
        gap: 16px;
        grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      }
      .card {
        background: linear-gradient(180deg, #1a243a 0%, var(--panel) 100%);
        border: 1px solid var(--border);
        border-radius: 14px;
        padding: 16px;
      }
      .card h2 { margin-top: 0; font-size: 18px; }
      label {
        display: block;
        margin-bottom: 8px;
        font-size: 13px;
        color: var(--muted);
      }
      input, button {
        width: 100%;
        border-radius: 10px;
        border: 1px solid var(--border);
        background: #0f1627;
        color: var(--text);
        padding: 10px 12px;
        margin-top: 6px;
      }
      input[type="checkbox"] {
        width: auto;
        margin-right: 8px;
      }
      .row {
        display: grid;
        gap: 12px;
        grid-template-columns: 1fr 1fr;
      }
      .checkbox-row {
        display: flex;
        align-items: center;
        margin: 8px 0;
        color: var(--muted);
        font-size: 13px;
      }
      button {
        background: linear-gradient(135deg, #1f9d6d, var(--accent));
        border: 0;
        color: #032414;
        font-weight: 700;
        cursor: pointer;
      }
      button:hover { filter: brightness(1.05); }
      pre {
        background: #0a1020;
        border: 1px solid var(--border);
        border-radius: 10px;
        padding: 10px;
        overflow: auto;
        max-height: 320px;
        font-size: 12px;
      }
      .hint {
        color: var(--muted);
        font-size: 12px;
      }
      .ok { color: var(--accent); }
      .err { color: var(--danger); }
    </style>
  </head>
  <body>
    <div class="container">
      <h1>Clawflux Admin</h1>
      <div class="subtitle">Provision users and deploy OpenClaw containers from one place.</div>

      <div class="card" style="margin-bottom: 16px;">
        <h2>Admin Identity</h2>
        <div class="row">
          <label>
            Admin Email
            <input id="adminEmail" placeholder="admin@example.com" />
          </label>
          <label>
            Admin Name
            <input id="adminName" placeholder="Admin" />
          </label>
        </div>
        <div class="hint">Requests use <code>X-Platform-Admin: true</code>. Save once and forms below reuse it.</div>
      </div>

      <div class="grid">
        <div class="card">
          <h2>Add User</h2>
          <label>
            User Email
            <input id="userEmail" placeholder="user@example.com" />
          </label>
          <label>
            Display Name
            <input id="userName" placeholder="OpenClaw User" />
          </label>
          <button id="createUserBtn">Create / Provision User</button>
          <pre id="userOut">Ready.</pre>
        </div>

        <div class="card">
          <h2>Deploy OpenClaw</h2>
          <label>
            Target User Email
            <input id="deployUserEmail" placeholder="user@example.com" />
          </label>
          <label>
            Target User Name
            <input id="deployUserName" placeholder="OpenClaw User" />
          </label>
          <div class="row">
            <label>
              App Name
              <input id="appName" value="openclaw" />
            </label>
            <label>
              App Slug
              <input id="appSlug" value="openclaw" />
            </label>
          </div>
          <label>
            Image
            <input id="image" value="ghcr.io/openclaw/openclaw:latest" />
          </label>
          <div class="row">
            <label>
              Replicas
              <input id="replicas" type="number" min="1" value="1" />
            </label>
            <label>
              Gateway Port
              <input id="gatewayPort" type="number" min="1" value="18789" />
            </label>
          </div>
          <div class="row">
            <label>
              Gateway Bind Address
              <input id="gatewayBindAddress" value="0.0.0.0" />
            </label>
            <label>
              Domain (optional)
              <input id="domain" placeholder="openclaw.example.com" />
            </label>
          </div>
          <label>
            Gateway Token (optional)
            <input id="gatewayToken" placeholder="super-secret-token" />
          </label>
          <label>
            Existing Secret Name (optional)
            <input id="existingSecretName" placeholder="openclaw-secrets" />
          </label>
          <label>
            Workspace Storage
            <input id="workspaceStorage" value="10Gi" />
          </label>
          <label>
            Provider API Keys (JSON object)
            <input id="providerKeys" value='{"OPENAI_API_KEY":""}' />
          </label>
          <label>
            Extra Env (JSON object)
            <input id="extraEnv" value='{}' />
          </label>
          <label>
            AGENTS.md (optional)
            <input id="agentsMarkdown" placeholder="- You are OpenClaw..." />
          </label>
          <label>
            settings.json (optional)
            <input id="settingsJson" placeholder='{"default_model":"gpt-5.4-mini"}' />
          </label>
          <div class="checkbox-row">
            <input id="isPublic" type="checkbox" checked />
            <label for="isPublic" style="margin: 0;">Expose via public ingress</label>
          </div>
          <button id="deployBtn">Deploy OpenClaw</button>
          <pre id="deployOut">Ready.</pre>
        </div>
      </div>
    </div>

    <script>
      const adminEmailEl = document.getElementById('adminEmail');
      const adminNameEl = document.getElementById('adminName');
      const userOutEl = document.getElementById('userOut');
      const deployOutEl = document.getElementById('deployOut');

      adminEmailEl.value = localStorage.getItem('adminEmail') || '';
      adminNameEl.value = localStorage.getItem('adminName') || '';

      function adminHeaders() {
        const email = adminEmailEl.value.trim();
        const name = adminNameEl.value.trim();
        if (!email) {
          throw new Error('Admin Email is required.');
        }
        localStorage.setItem('adminEmail', email);
        localStorage.setItem('adminName', name);

        const headers = {
          'Content-Type': 'application/json',
          'X-User-Email': email,
          'X-Platform-Admin': 'true',
        };
        if (name) {
          headers['X-User-Name'] = name;
        }
        return headers;
      }

      function print(target, value, isError) {
        target.textContent = typeof value === 'string' ? value : JSON.stringify(value, null, 2);
        target.className = isError ? 'err' : 'ok';
      }

      function parseObjectInput(inputId) {
        const raw = document.getElementById(inputId).value.trim();
        if (!raw) {
          return {};
        }
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          return parsed;
        }
        throw new Error(inputId + ' must be a JSON object');
      }

      async function callJSON(url, opts) {
        const res = await fetch(url, opts);
        const text = await res.text();
        let body = text;
        try { body = JSON.parse(text); } catch (_) {}
        if (!res.ok) {
          const message = (body && body.message) || (body && body.error) || text || res.statusText;
          throw new Error(message);
        }
        return body;
      }

      document.getElementById('createUserBtn').addEventListener('click', async () => {
        try {
          const payload = {
            email: document.getElementById('userEmail').value.trim(),
            display_name: document.getElementById('userName').value.trim(),
          };
          const out = await callJSON('/v1/admin/users', {
            method: 'POST',
            headers: adminHeaders(),
            body: JSON.stringify(payload),
          });
          print(userOutEl, out, false);
        } catch (err) {
          print(userOutEl, err.message, true);
        }
      });

      document.getElementById('deployBtn').addEventListener('click', async () => {
        try {
          const payload = {
            user_email: document.getElementById('deployUserEmail').value.trim(),
            user_name: document.getElementById('deployUserName').value.trim(),
            app_name: document.getElementById('appName').value.trim(),
            app_slug: document.getElementById('appSlug').value.trim(),
            image: document.getElementById('image').value.trim(),
            replicas: Number(document.getElementById('replicas').value) || 1,
            gateway_port: Number(document.getElementById('gatewayPort').value) || 18789,
            gateway_bind_address: document.getElementById('gatewayBindAddress').value.trim(),
            gateway_token: document.getElementById('gatewayToken').value.trim(),
            existing_secret_name: document.getElementById('existingSecretName').value.trim(),
            workspace_storage: document.getElementById('workspaceStorage').value.trim(),
            provider_api_keys: parseObjectInput('providerKeys'),
            extra_env: parseObjectInput('extraEnv'),
            agents_markdown: document.getElementById('agentsMarkdown').value,
            settings_json: document.getElementById('settingsJson').value,
            public: document.getElementById('isPublic').checked,
            domain: document.getElementById('domain').value.trim(),
          };
          const out = await callJSON('/v1/admin/openclaw/deploy', {
            method: 'POST',
            headers: adminHeaders(),
            body: JSON.stringify(payload),
          });
          print(deployOutEl, out, false);
        } catch (err) {
          print(deployOutEl, err.message, true);
        }
      });
    </script>
  </body>
</html>
`

func (r *Router) handleAdminUI(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/admin" && req.URL.Path != "/admin/" {
		http.NotFound(w, req)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminUIHTML))
}
