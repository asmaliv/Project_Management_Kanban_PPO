import { useState, type ReactNode } from 'react'
import { Link, useLocation, Outlet } from 'react-router-dom'
import { useAuth } from '../../features/auth/useAuth'
import type { UserRole } from '../../types'

interface NavItem {
  label: string
  to: string
  icon: (className: string) => ReactNode
  allowedRoles: UserRole[]
}

const NAV_ITEMS: NavItem[] = [
  {
    label: 'Dashboard',
    to: '/',
    allowedRoles: ['admin', 'project_manager', 'team_leader', 'audit', 'risk_management'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
      </svg>
    ),
  },
  {
    label: 'Pengguna',
    to: '/users',
    allowedRoles: ['admin'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
      </svg>
    ),
  },
  {
    label: 'Proyek',
    to: '/projects',
    allowedRoles: ['admin', 'project_manager', 'team_leader', 'audit', 'risk_management'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" />
      </svg>
    ),
  },
  {
    label: 'Tugas Tim',
    to: '/team-tasks',
    allowedRoles: ['team_leader'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
      </svg>
    ),
  },
  {
    label: 'Tugas Saya',
    to: '/my-tasks',
    allowedRoles: ['team_leader'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
    ),
  },
  {
    label: 'Risk Register / Risiko',
    to: '/risks',
    allowedRoles: ['risk_management', 'project_manager', 'team_leader'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
  },
  {
    label: 'Temuan Audit',
    to: '/findings',
    allowedRoles: ['admin', 'project_manager', 'team_leader', 'audit', 'risk_management'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
    ),
  },
  {
    label: 'Audit Trail',
    to: '/audit-trail',
    allowedRoles: ['audit', 'admin'],
    icon: (className) => (
      <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
]

const ROLE_LABELS: Record<UserRole, { label: string; badge: string }> = {
  admin: { label: 'Administrator', badge: 'bg-purple-100 text-purple-800 border-purple-200' },
  project_manager: { label: 'Project Manager', badge: 'bg-blue-100 text-blue-800 border-blue-200' },
  team_leader: { label: 'Team Leader', badge: 'bg-emerald-100 text-emerald-800 border-emerald-200' },
  audit: { label: 'Audit', badge: 'bg-amber-100 text-amber-800 border-amber-200' },
  risk_management: { label: 'Risk Management', badge: 'bg-rose-100 text-rose-800 border-rose-200' },
}

export default function AppLayout() {
  const { user, logout } = useAuth()
  const location = useLocation()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  if (!user) return null

  const allowedNav = NAV_ITEMS.filter((item) => item.allowedRoles.includes(user.role))
  const roleInfo = ROLE_LABELS[user.role]

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .filter(Boolean)
      .slice(0, 2)
      .map((n) => n[0].toUpperCase())
      .join('')
  }

  return (
    <div className="min-h-screen bg-background flex flex-col md:flex-row font-sans">
      {/* Mobile Header */}
      <header className="md:hidden bg-surface border-b border-border px-4 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-primary text-white font-bold flex items-center justify-center text-sm shadow">
            P
          </div>
          <span className="font-semibold text-text text-sm">PPO Kanban</span>
        </div>
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="p-2 text-text-muted hover:text-text rounded-md border border-border"
          aria-label="Toggle Navigation"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
      </header>

      {/* Sidebar Navigation */}
      <aside
        className={`${
          mobileMenuOpen ? 'block' : 'hidden'
        } md:block md:w-64 bg-surface border-r border-border shrink-0 md:min-h-screen flex flex-col justify-between`}
      >
        <div>
          {/* Logo / Header */}
          <div className="p-5 border-b border-border hidden md:flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-primary text-white font-bold flex items-center justify-center text-base shadow-sm">
              P
            </div>
            <div>
              <h1 className="font-bold text-text text-sm leading-tight">PPO Kanban Board</h1>
              <p className="text-[11px] text-text-muted">Perbankan</p>
            </div>
          </div>

          {/* Nav Items */}
          <nav className="p-3 space-y-1">
            {allowedNav.map((item) => {
              const isActive = location.pathname === item.to
              return (
                <Link
                  key={item.to}
                  to={item.to}
                  onClick={() => setMobileMenuOpen(false)}
                  className={`flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                    isActive
                      ? 'bg-primary-soft text-primary font-semibold'
                      : 'text-text-muted hover:text-text hover:bg-surface-muted'
                  }`}
                >
                  {item.icon(`w-5 h-5 shrink-0 ${isActive ? 'text-primary' : 'text-slate-400'}`)}
                  <span>{item.label}</span>
                </Link>
              )
            })}
          </nav>
        </div>

        {/* User Card at bottom of sidebar */}
        <div className="p-4 border-t border-border bg-slate-50/50">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-9 h-9 rounded-full bg-slate-200 text-slate-700 font-bold text-xs flex items-center justify-center shrink-0 border border-slate-300">
              {getInitials(user.full_name)}
            </div>
            <div className="min-w-0 flex-1">
              <p className="text-xs font-semibold text-text truncate">{user.full_name}</p>
              <p className="text-[11px] text-text-muted truncate">{user.email}</p>
            </div>
          </div>
          <div className="flex items-center justify-between pt-1">
            <span
              className={`inline-block px-2 py-0.5 text-[10px] font-medium rounded-full border ${roleInfo.badge}`}
            >
              {roleInfo.label}
            </span>
            <button
              onClick={() => void logout()}
              className="text-xs text-danger hover:underline cursor-pointer font-medium"
            >
              Keluar
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col min-w-0">
        {/* Topbar */}
        <header className="bg-surface border-b border-border px-6 py-3.5 hidden md:flex items-center justify-between">
          <div className="flex items-center gap-2 text-xs text-text-muted">
            <span>Sistem Manajemen Proyek</span>
            <span>/</span>
            <span className="text-text font-semibold capitalize">
              {location.pathname === '/' ? 'Dashboard' : location.pathname.replace('/', '')}
            </span>
          </div>

          <div className="flex items-center gap-4">
            <span
              className={`px-2.5 py-1 text-xs font-medium rounded-full border ${roleInfo.badge}`}
            >
              {roleInfo.label}
            </span>
            <button
              id="topbar-logout-btn"
              onClick={() => void logout()}
              className="px-3 py-1.5 text-xs text-slate-600 hover:text-danger hover:bg-slate-100 rounded-md border border-border transition-colors cursor-pointer"
            >
              Keluar
            </button>
          </div>
        </header>

        {/* Content Outlet */}
        <div className="flex-1 p-6 max-w-7xl w-full mx-auto">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
