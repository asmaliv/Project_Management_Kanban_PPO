import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { apiGetMe, getCsrfToken } from '../../lib/api'

export default function DashboardPage() {
  const { user } = useAuth()
  const [sessionCheck, setSessionCheck] = useState<string | null>(null)
  const [isChecking, setIsChecking] = useState(false)

  if (!user) return null

  const handleVerifySession = async () => {
    setIsChecking(true)
    try {
      const data = await apiGetMe()
      setSessionCheck(`Sesi valid. User ID: ${data.user.id}, Role: ${data.user.role}`)
    } catch {
      setSessionCheck('Gagal memverifikasi sesi.')
    } finally {
      setIsChecking(false)
    }
  }

  const csrfToken = getCsrfToken()

  return (
    <div className="space-y-6">
      {/* Top Banner */}
      <div className="bg-surface p-6 rounded-xl border border-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-text">
            Selamat datang, {user.full_name}!
          </h1>
          <p className="text-sm text-text-muted mt-1">
            Anda masuk sebagai{' '}
            <span className="font-semibold text-primary capitalize">{user.role.replace('_', ' ')}</span>{' '}
            pada sistem manajemen proyek Kanban internal perbankan.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Link
            to="/change-password"
            className="px-3.5 py-2 text-xs font-medium text-slate-700 bg-surface-muted hover:bg-slate-200 rounded-lg border border-border transition-colors text-center"
          >
            Ganti Password
          </Link>
        </div>
      </div>

      {/* Security Status Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <div className="flex items-center gap-2 text-xs text-text-muted font-medium mb-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500" />
            Status Sesi
          </div>
          <div className="text-lg font-bold text-text">Aktif (HTTP-Only)</div>
          <p className="text-[11px] text-text-muted mt-1">
            Cookie SameSite=Lax terproteksi
          </p>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <div className="flex items-center gap-2 text-xs text-text-muted font-medium mb-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500" />
            Perlindungan CSRF
          </div>
          <div className="text-lg font-bold text-text truncate">
            {csrfToken ? 'Token Terpasang' : 'Tersinkron'}
          </div>
          <p className="text-[11px] text-text-muted mt-1">
            Header X-CSRF-Token otomatis
          </p>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <div className="flex items-center gap-2 text-xs text-text-muted font-medium mb-1">
            <span className="w-2 h-2 rounded-full bg-blue-500" />
            Batas Waktu Sesi
          </div>
          <div className="text-lg font-bold text-text">15 mnt / 12 jam</div>
          <p className="text-[11px] text-text-muted mt-1">
            Idle 15 mnt, absolute 12 jam
          </p>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <div className="flex items-center gap-2 text-xs text-text-muted font-medium mb-1">
            <span className="w-2 h-2 rounded-full bg-purple-500" />
            Role Pengguna
          </div>
          <div className="text-lg font-bold text-text capitalize">
            {user.role.replace('_', ' ')}
          </div>
          <p className="text-[11px] text-text-muted mt-1">
            Otorisasi RBAC terverifikasi server
          </p>
        </div>
      </div>

      {/* Phase 1 Verification Widget */}
      <div className="bg-surface p-6 rounded-xl border border-border shadow-sm">
        <h2 className="text-base font-semibold text-text mb-2">
          Verifikasi Sesi & Keamanan (Phase 1)
        </h2>
        <p className="text-xs text-text-muted mb-4 max-w-2xl">
          Widget ini memvalidasi sesi aktif Anda langsung ke backend via <code className="bg-slate-100 px-1 py-0.5 rounded text-slate-800">GET /api/v1/auth/me</code> dan memeriksa keberadaan token CSRF.
        </p>

        <div className="flex flex-wrap items-center gap-3">
          <button
            onClick={() => void handleVerifySession()}
            disabled={isChecking}
            className="px-4 py-2 bg-primary text-white text-xs font-semibold rounded-lg hover:bg-primary-hover disabled:opacity-50 transition-colors cursor-pointer"
          >
            {isChecking ? 'Memeriksa…' : 'Cek Status Sesi Backend'}
          </button>
        </div>

        {sessionCheck && (
          <div className="mt-3 p-3 bg-slate-50 border border-slate-200 rounded-lg text-xs font-mono text-slate-700">
            {sessionCheck}
          </div>
        )}
      </div>

      {/* Roadmap Summary */}
      <div className="bg-surface p-6 rounded-xl border border-border shadow-sm">
        <h2 className="text-base font-semibold text-text mb-4">
          Status Milestone Implementasi
        </h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between p-3 rounded-lg bg-emerald-50/60 border border-emerald-200">
            <div className="flex items-center gap-3">
              <span className="w-6 h-6 rounded-full bg-emerald-600 text-white flex items-center justify-center text-xs font-bold">
                ✓
              </span>
              <div>
                <p className="text-xs font-semibold text-emerald-950">Phase 0: Setup & Pondasi</p>
                <p className="text-[11px] text-emerald-700">Database, Chi backend, Vite frontend, CI checklist</p>
              </div>
            </div>
            <span className="text-xs font-bold text-emerald-700 bg-emerald-100 px-2 py-0.5 rounded-full">Selesai</span>
          </div>

          <div className="flex items-center justify-between p-3 rounded-lg bg-blue-50/60 border border-blue-200">
            <div className="flex items-center gap-3">
              <span className="w-6 h-6 rounded-full bg-blue-600 text-white flex items-center justify-center text-xs font-bold">
                1
              </span>
              <div>
                <p className="text-xs font-semibold text-blue-950">Phase 1: Autentikasi & Sesi</p>
                <p className="text-[11px] text-blue-700">Argon2id, cookie HttpOnly, CSRF token, login lockout, ganti password</p>
              </div>
            </div>
            <span className="text-xs font-bold text-blue-700 bg-blue-100 px-2 py-0.5 rounded-full">Aktif</span>
          </div>

          <div className="flex items-center justify-between p-3 rounded-lg bg-slate-50 border border-slate-200 opacity-75">
            <div className="flex items-center gap-3">
              <span className="w-6 h-6 rounded-full bg-slate-400 text-white flex items-center justify-center text-xs font-bold">
                2
              </span>
              <div>
                <p className="text-xs font-semibold text-slate-800">Phase 2: Manajemen User & RBAC</p>
                <p className="text-[11px] text-slate-500">Kelola pengguna, matriks izin 5 role, form tambah user</p>
              </div>
            </div>
            <span className="text-xs font-medium text-slate-600 bg-slate-200 px-2 py-0.5 rounded-full">Berikutnya</span>
          </div>
        </div>
      </div>
    </div>
  )
}
