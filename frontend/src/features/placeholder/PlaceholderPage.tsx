import { useLocation, Link } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'

export default function PlaceholderPage() {
  const location = useLocation()
  const { user } = useAuth()

  const pathName = location.pathname.replace('/', '')
  const title = pathName
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')

  return (
    <div className="bg-surface rounded-xl border border-border shadow-sm p-8 text-center max-w-xl mx-auto my-12">
      <div className="w-12 h-12 rounded-2xl bg-blue-50 text-primary flex items-center justify-center mx-auto mb-4 border border-blue-100">
        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      </div>

      <h2 className="text-lg font-bold text-text">Modul {title}</h2>
      <p className="text-sm text-text-muted mt-2">
        Modul ini dijadwalkan pada phase implementasi berikutnya sesuai Roadmap (IMPLEMENTATION_PLAN.md).
      </p>

      <div className="mt-4 p-3 bg-slate-50 rounded-lg border border-slate-200 text-xs text-text-muted inline-block">
        Role aktif: <span className="font-semibold text-text capitalize">{user?.role}</span> • Route: <code className="text-primary font-mono">{location.pathname}</code>
      </div>

      <div className="mt-6">
        <Link
          to="/"
          className="inline-flex items-center gap-2 px-4 py-2 text-xs font-semibold text-primary hover:text-primary-hover bg-primary-soft rounded-lg transition-colors"
        >
          Kembali ke Dashboard
        </Link>
      </div>
    </div>
  )
}
