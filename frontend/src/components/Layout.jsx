import { NavLink, Outlet } from 'react-router-dom'
import { Activity, LayoutDashboard, Rocket, ShieldCheck, Users } from 'lucide-react'

const navItems = [
  {
    to: '/',
    icon: LayoutDashboard,
    label: 'Dashboard',
    description: 'Fleet health and live deployments',
  },
  {
    to: '/deploy',
    icon: Rocket,
    label: 'Deploy',
    description: 'Launch new OpenClaw workspaces',
  },
  {
    to: '/users',
    icon: Users,
    label: 'Users',
    description: 'Provision users and trace activity',
  },
]

export default function Layout() {
  return (
    <div className="app-shell">
      <aside className="app-sidebar">
        <div className="sidebar-brand">
          <div className="sidebar-brand-kicker">
            <Activity size={14} />
            Platform Ops
          </div>
          <h1>Clawflux</h1>
          <p>Admin console for managing OpenClaw rollouts, operators, and deployment recovery.</p>
        </div>

        <nav className="sidebar-nav">
          {navItems.map(({ to, icon: Icon, label, description }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              className={({ isActive }) => `sidebar-link${isActive ? ' active' : ''}`}
            >
              <Icon size={16} />
              <div className="sidebar-link-copy">
                <strong>{label}</strong>
                <span>{description}</span>
              </div>
            </NavLink>
          ))}
        </nav>

        <div className="identity-card">
          <div className="sidebar-brand-kicker">
            <ShieldCheck size={14} />
            Admin Identity
          </div>
          <AdminIdentityBar />
        </div>
      </aside>

      <main className="app-main">
        <Outlet />
      </main>
    </div>
  )
}

function AdminIdentityBar() {
  const email = localStorage.getItem('adminEmail') || ''
  const name = localStorage.getItem('adminName') || ''

  return (
    <div>
      <div className="identity-label">Authenticated as</div>
      {email ? (
        <>
          <div className="identity-email">{email}</div>
          <div className="identity-meta">
            {name || 'Platform Admin'}
          </div>
        </>
      ) : (
        <div className="identity-meta identity-warning">Admin email not set yet.</div>
      )}
    </div>
  )
}
