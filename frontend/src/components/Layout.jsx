import { NavLink, Outlet } from 'react-router-dom'
import { LayoutDashboard, Rocket, Users, Settings } from 'lucide-react'

const navItems = [
  { to: '/', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/deploy', icon: Rocket, label: 'Deploy' },
  { to: '/users', icon: Users, label: 'Users' },
]

export default function Layout() {
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar */}
      <aside style={{
        width: 'var(--sidebar-w)',
        background: 'var(--panel)',
        borderRight: '1px solid var(--border)',
        display: 'flex',
        flexDirection: 'column',
        flexShrink: 0,
        position: 'fixed',
        top: 0,
        left: 0,
        bottom: 0,
      }}>
        <div style={{ padding: '20px 16px 12px', borderBottom: '1px solid var(--border)' }}>
          <div style={{ fontWeight: 800, fontSize: 17, color: 'var(--accent)', letterSpacing: '-0.02em' }}>
            Clawflux
          </div>
          <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>Admin Console</div>
        </div>
        <nav style={{ padding: '12px 8px', flex: 1 }}>
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              end={to === '/'}
              style={({ isActive }) => ({
                display: 'flex',
                alignItems: 'center',
                gap: 10,
                padding: '9px 12px',
                borderRadius: 8,
                fontSize: 13,
                fontWeight: 500,
                marginBottom: 2,
                color: isActive ? 'var(--accent)' : 'var(--muted)',
                background: isActive ? 'rgba(52,211,153,0.08)' : 'transparent',
                transition: 'all 0.15s',
              })}
            >
              <Icon size={16} />
              {label}
            </NavLink>
          ))}
        </nav>
        <div style={{ padding: '12px 16px', borderTop: '1px solid var(--border)' }}>
          <AdminIdentityBar />
        </div>
      </aside>

      {/* Main */}
      <main style={{ marginLeft: 'var(--sidebar-w)', flex: 1, minHeight: '100vh' }}>
        <Outlet />
      </main>
    </div>
  )
}

function AdminIdentityBar() {
  const email = localStorage.getItem('adminEmail') || ''
  return (
    <div style={{ fontSize: 11, color: 'var(--muted)' }}>
      {email ? (
        <>
          <div style={{ color: 'var(--text)', fontWeight: 600, fontSize: 12, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{email}</div>
          <div style={{ marginTop: 2 }}>Platform Admin</div>
        </>
      ) : (
        <span style={{ color: 'var(--warn)' }}>⚠ Admin email not set</span>
      )}
    </div>
  )
}
