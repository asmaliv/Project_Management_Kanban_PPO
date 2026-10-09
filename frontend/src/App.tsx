import { useEffect, useState } from 'react'
import type { HealthCheckResponse } from './types'

interface RoleInfo {
  id: string
  name: string
  description: string
  badgeColor: string
}

const ROLES: RoleInfo[] = [
  {
    id: 'admin',
    name: 'Admin',
    description: 'Mengelola akun user dan konfigurasi sistem. Tanpa akses ke data proyek langsung.',
    badgeColor: 'bg-purple-100 text-purple-800 border-purple-200',
  },
  {
    id: 'project_manager',
    name: 'Project Manager',
    description: 'Membuat proyek, menyusun papan, Maker-Checker persetujuan task selesai, kelola tim.',
    badgeColor: 'bg-blue-100 text-blue-800 border-blue-200',
  },
  {
    id: 'team_leader',
    name: 'Team Leader',
    description: 'Mengerjakan task proyek, ajukan risiko. Tidak dapat memindahkan task ke/dari Done.',
    badgeColor: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  },
  {
    id: 'audit',
    name: 'Audit',
    description: 'Pemeriksaan read-only ke semua proyek, membuat temuan audit, ekspor audit trail.',
    badgeColor: 'bg-amber-100 text-amber-800 border-amber-200',
  },
  {
    id: 'risk_management',
    name: 'Risk Management',
    description: 'Menilai risiko, analisis Heatmap 5x5, verifikasi mitigasi sebelum risiko ditutup.',
    badgeColor: 'bg-rose-100 text-rose-800 border-rose-200',
  },
]

export default function App() {
  const [healthStatus, setHealthStatus] = useState<'idle' | 'checking' | 'connected' | 'error'>('checking')
  const [healthData, setHealthData] = useState<HealthCheckResponse | null>(null)
  const [latency, setLatency] = useState<number | null>(null)
  const [errorMessage, setErrorMessage] = useState<string>('')

  const checkHealth = async () => {
    setHealthStatus('checking')
    setErrorMessage('')
    const start = performance.now()
    try {
      const res = await fetch('/healthz')
      const end = performance.now()
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}`)
      }
      const data = await res.json()
      setHealthData(data.data || data)
      setLatency(Math.round(end - start))
      setHealthStatus('connected')
    } catch (err) {
      setHealthStatus('error')
      setErrorMessage(err instanceof Error ? err.message : 'Koneksi gagal')
    }
  }

  useEffect(() => {
    let active = true
    const run = async () => {
      const start = performance.now()
      try {
        const res = await fetch('/healthz')
        const end = performance.now()
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`)
        }
        const data = await res.json()
        if (active) {
          setHealthData(data.data || data)
          setLatency(Math.round(end - start))
          setHealthStatus('connected')
        }
      } catch (err) {
        if (active) {
          setHealthStatus('error')
          setErrorMessage(err instanceof Error ? err.message : 'Koneksi gagal')
        }
      }
    }
    void run()
    return () => {
      active = false
    }
  }, [])

  return (
    <div className="min-h-screen bg-slate-50 flex flex-col font-sans text-slate-900">
      {/* Topbar */}
      <header className="bg-white border-b border-slate-200 sticky top-0 z-30 shadow-xs">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <div className="w-9 h-9 rounded-lg bg-blue-600 text-white flex items-center justify-center font-bold text-lg shadow-sm">
              P
            </div>
            <div>
              <span className="text-lg font-bold text-slate-900 tracking-tight">PPO Kanban Board</span>
              <span className="hidden sm:inline-block ml-2.5 px-2 py-0.5 text-xs font-medium bg-slate-100 text-slate-600 rounded border border-slate-200">
                Lingkungan Perbankan
              </span>
            </div>
          </div>
          <div className="flex items-center space-x-3">
            <span className="inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium bg-blue-50 text-blue-700 border border-blue-200">
              Phase 0: Project Setup
            </span>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 space-y-8">
        {/* Hero Section */}
        <div className="bg-white rounded-xl border border-slate-200 p-6 sm:p-8 shadow-xs">
          <div className="max-w-3xl space-y-3">
            <h1 className="text-2xl sm:text-3xl font-bold text-slate-900 tracking-tight">
              Sistem Manajemen Proyek, Risiko & Jejak Audit
            </h1>
            <p className="text-slate-600 text-base leading-relaxed">
              Platform Kanban terintegrasi dengan pemisahan tugas (Segregation of Duties), matriks RBAC ketat 5 role, Maker-Checker selesai task, Risk Register 5×5, dan Audit Trail yang tidak dapat dimutasi (append-only).
            </p>
          </div>

          {/* Backend Connectivity Status Bar */}
          <div className="mt-6 pt-6 border-t border-slate-100 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div className="flex items-center space-x-3">
              <div className="flex items-center">
                <span className={`w-3 h-3 rounded-full mr-2 ${
                  healthStatus === 'connected' ? 'bg-emerald-500 animate-pulse' :
                  healthStatus === 'checking' ? 'bg-amber-500 animate-spin' :
                  healthStatus === 'error' ? 'bg-rose-500' : 'bg-slate-400'
                }`} />
                <span className="text-sm font-medium text-slate-700">
                  Backend API (/healthz):
                </span>
              </div>

              {healthStatus === 'connected' && (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold bg-emerald-100 text-emerald-800">
                  Online (Status: {healthData?.status || 'ok'}) • {latency}ms
                </span>
              )}
              {healthStatus === 'checking' && (
                <span className="text-xs text-slate-500">Memeriksa koneksi...</span>
              )}
              {healthStatus === 'error' && (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded text-xs font-semibold bg-rose-100 text-rose-800">
                  Offline / Belum Jalan ({errorMessage})
                </span>
              )}
            </div>

            <button
              type="button"
              onClick={checkHealth}
              disabled={healthStatus === 'checking'}
              className="inline-flex items-center justify-center px-4 py-2 border border-slate-300 rounded-md text-xs font-medium text-slate-700 bg-white hover:bg-slate-50 active:bg-slate-100 focus:outline-hidden focus:ring-2 focus:ring-blue-500 disabled:opacity-50 transition-colors shadow-2xs cursor-pointer"
            >
              Uji Sambungan API
            </button>
          </div>
        </div>

        {/* 5 Roles Matrix Overview */}
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-bold text-slate-900 tracking-tight">
              5 Role Pengguna Terdaftar (RBAC)
            </h2>
            <span className="text-xs text-slate-500 font-medium">Sesuai SECURITY.md</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {ROLES.map((role) => (
              <div
                key={role.id}
                className="bg-white rounded-lg border border-slate-200 p-5 hover:border-slate-300 transition-shadow shadow-xs flex flex-col justify-between space-y-3"
              >
                <div>
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-semibold text-slate-900">{role.name}</span>
                    <span className={`px-2 py-0.5 rounded text-xs font-medium border ${role.badgeColor}`}>
                      {role.id}
                    </span>
                  </div>
                  <p className="text-xs text-slate-600 leading-relaxed">
                    {role.description}
                  </p>
                </div>
              </div>
            ))}

            {/* Architecture Card */}
            <div className="bg-slate-900 text-white rounded-lg p-5 flex flex-col justify-between shadow-xs">
              <div>
                <span className="text-xs uppercase tracking-wider text-blue-400 font-bold block mb-1">
                  Arsitektur Sistem
                </span>
                <span className="text-sm font-semibold text-white block mb-2">
                  Go Chi + PostgreSQL + React SPA
                </span>
                <p className="text-xs text-slate-300 leading-relaxed">
                  Cookie Sesi HttpOnly, CSRF Token Header, sqlc query terparameter, append-only trigger audit.
                </p>
              </div>
              <div className="pt-2 text-[11px] text-slate-400 font-mono">
                /backend (chi) ↔ /frontend (vite)
              </div>
            </div>
          </div>
        </div>

        {/* Implementation Roadmap */}
        <div className="bg-white rounded-xl border border-slate-200 p-6 shadow-xs space-y-4">
          <h2 className="text-lg font-bold text-slate-900 tracking-tight">
            Status Fase Pengerjaan (IMPLEMENTATION_PLAN.md)
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 text-xs">
            <div className="border border-blue-300 bg-blue-50/50 p-3 rounded-lg">
              <span className="font-bold text-blue-800 block">Phase 0: Project Setup</span>
              <span className="text-blue-600">Aktif — Monorepo & Pondasi</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 1: Autentikasi & Sesi</span>
              <span>Argon2id, Cookie, Lockout</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 2: User & RBAC</span>
              <span>Admin kelola 5 role</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 3: Audit Trail</span>
              <span>Append-only DB trigger</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 4: Proyek & Kolom</span>
              <span>Ownership & Member TL</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 5: Kanban & Task</span>
              <span>Maker-checker & WIP Limit</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 6: Risk Management</span>
              <span>Heatmap 5×5 & Mitigasi</span>
            </div>
            <div className="border border-slate-200 p-3 rounded-lg bg-slate-50/50 text-slate-600">
              <span className="font-semibold block text-slate-800">Phase 7-8: Temuan & Prod</span>
              <span>Dashboard Role & Hardening</span>
            </div>
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-200 bg-white py-4 mt-auto">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 gap-2">
          <span>PPO Kanban Board • Internal Banking Workflow</span>
          <span>Prinsip Keamanan: Default Deny, Maker-Checker, Zero Data Leakage</span>
        </div>
      </footer>
    </div>
  )
}
