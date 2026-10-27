import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'
import { fetchDashboard } from './api'
import type {
  DashboardResponseData,
  AdminDashboardData,
  PMDashboardData,
  TLDashboardData,
  AuditDashboardData,
  RiskManagementDashboardData,
} from './types'

export default function DashboardPage() {
  const { user } = useAuth()
  const [data, setData] = useState<DashboardResponseData | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [tlTab, setTlTab] = useState<'team' | 'my_tasks'>('team')

  useEffect(() => {
    async function loadData() {
      setIsLoading(true)
      setError(null)
      try {
        const res = await fetchDashboard()
        setData(res)
      } catch (err: unknown) {
        if (err instanceof Error) setError(err.message)
        else setError('Gagal memuat dashboard')
      } finally {
        setIsLoading(false)
      }
    }
    loadData()
  }, [])

  if (!user) return null

  return (
    <div className="space-y-6">
      {/* Welcome Banner */}
      <div className="bg-surface p-6 rounded-2xl border border-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-text">
            Selamat datang, {user.full_name}!
          </h1>
          <p className="text-sm text-text-muted mt-1">
            Anda masuk sebagai{' '}
            <span className="font-semibold text-primary capitalize">
              {user.role.replace('_', ' ')}
            </span>{' '}
            pada sistem manajemen proyek Kanban perbankan dengan pemisahan tugas ketat.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Link
            to="/change-password"
            className="px-3.5 py-2 text-xs font-medium text-slate-700 bg-surface-muted hover:bg-slate-200 rounded-xl border border-border transition-colors"
          >
            Ganti Password
          </Link>
        </div>
      </div>

      {isLoading ? (
        <div className="p-12 text-center text-text-muted text-sm flex items-center justify-center gap-2">
          <div className="w-4 h-4 border-2 border-primary/20 border-t-primary rounded-full animate-spin" />
          Memuat metrik dashboard...
        </div>
      ) : error ? (
        <div className="p-8 text-center text-red-600 bg-red-50 border border-red-200 rounded-xl text-sm">
          {error}
        </div>
      ) : !data ? null : (
        <>
          {/* Admin Dashboard */}
          {data.role === 'admin' && <AdminView data={data as AdminDashboardData} />}

          {/* PM Dashboard */}
          {data.role === 'project_manager' && <PMView data={data as PMDashboardData} />}

          {/* Team Leader Dashboard */}
          {data.role === 'team_leader' && (
            <TLView
              data={data as TLDashboardData}
              activeTab={tlTab}
              onTabChange={setTlTab}
            />
          )}

          {/* Audit Dashboard */}
          {data.role === 'audit' && <AuditView data={data as AuditDashboardData} />}

          {/* Risk Management Dashboard */}
          {data.role === 'risk_management' && (
            <RiskManagementView data={data as RiskManagementDashboardData} />
          )}
        </>
      )}
    </div>
  )
}

// ─── ADMIN VIEW ──────────────────────────────────────────────────────────────

function AdminView({ data }: { data: AdminDashboardData }) {
  const stats = data.user_stats
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Total Akun Pengguna</span>
          <div className="text-2xl font-bold text-text mt-1">{stats?.total_users ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Administrator</span>
          <div className="text-2xl font-bold text-purple-600 mt-1">{stats?.admin_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Project Manager</span>
          <div className="text-2xl font-bold text-blue-600 mt-1">{stats?.pm_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Team Leader</span>
          <div className="text-2xl font-bold text-emerald-600 mt-1">{stats?.tl_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Auditor</span>
          <div className="text-2xl font-bold text-amber-600 mt-1">{stats?.audit_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Risk Management</span>
          <div className="text-2xl font-bold text-rose-600 mt-1">{stats?.risk_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-rose-200 bg-rose-50/20 shadow-sm">
          <span className="text-xs text-rose-600 font-medium">Akun Terkunci (Locked)</span>
          <div className="text-2xl font-bold text-rose-700 mt-1">{stats?.locked_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Akun Nonaktif</span>
          <div className="text-2xl font-bold text-slate-500 mt-1">{stats?.inactive_count ?? 0}</div>
        </div>
      </div>

      {/* Recent activities */}
      <div className="bg-surface rounded-xl border border-border shadow-sm p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-base font-bold text-text">Aktivitas Sistem Terbaru (Audit Trail)</h2>
          <Link to="/audit-trail" className="text-xs text-primary font-medium hover:underline">
            Lihat Semua Log &rarr;
          </Link>
        </div>
        <div className="divide-y divide-border">
          {(!data.recent_activities || data.recent_activities.length === 0) ? (
            <p className="text-xs text-text-muted py-4">Belum ada aktivitas tercatat.</p>
          ) : (
            data.recent_activities.map((act) => (
              <div key={act.id} className="py-2.5 flex items-center justify-between text-xs">
                <div className="flex items-center gap-2">
                  <span className="px-2 py-0.5 rounded font-mono text-[10px] bg-slate-100 text-slate-800">
                    {act.action}
                  </span>
                  <span className="text-text font-medium capitalize">{act.entity_type}</span>
                  <span className="text-text-muted font-mono text-[10px]">({act.entity_id.slice(0, 8)})</span>
                </div>
                <div className="text-text-muted text-[11px]">
                  {new Date(act.created_at).toLocaleString('id-ID')}
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}

// ─── PM VIEW ─────────────────────────────────────────────────────────────────

function PMView({ data }: { data: PMDashboardData }) {
  return (
    <div className="space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Proyek Aktif</span>
          <div className="text-2xl font-bold text-text mt-1">{data.totals.active_projects}</div>
          <Link to="/projects" className="text-[11px] text-primary hover:underline mt-1 block">
            Kelola Proyek &rarr;
          </Link>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-rose-200 bg-rose-50/20 shadow-sm">
          <span className="text-xs text-rose-600 font-medium">Task Terlambat</span>
          <div className="text-2xl font-bold text-rose-600 mt-1">{data.totals.overdue_tasks}</div>
          <Link to="/team-tasks?overdue=true" className="text-[11px] text-rose-700 hover:underline mt-1 block">
            Lihat Task Terlambat &rarr;
          </Link>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-amber-200 bg-amber-50/20 shadow-sm">
          <span className="text-xs text-amber-700 font-medium">Menunggu Persetujuan (In Review)</span>
          <div className="text-2xl font-bold text-amber-700 mt-1">{data.totals.pending_approval}</div>
          <span className="text-[11px] text-text-muted mt-1 block">Siap diverifikasi PM</span>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-red-200 bg-red-50/20 shadow-sm">
          <span className="text-xs text-red-600 font-medium">Temuan Audit Terbuka</span>
          <div className="text-2xl font-bold text-red-700 mt-1">{data.totals.open_findings}</div>
          <Link to="/findings?status=open" className="text-[11px] text-red-700 hover:underline mt-1 block">
            Beri Tanggapan &rarr;
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Overdue tasks list */}
        <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
          <h2 className="text-sm font-bold text-text mb-3 flex items-center justify-between">
            <span>Task Terlambat yang Perlu Perhatian</span>
            <Link to="/team-tasks?overdue=true" className="text-xs text-primary font-normal hover:underline">
              Semua &rarr;
            </Link>
          </h2>
          <div className="divide-y divide-border">
            {data.overdue_tasks.length === 0 ? (
              <p className="text-xs text-text-muted py-4">Bagus! Tidak ada task yang terlambat.</p>
            ) : (
              data.overdue_tasks.slice(0, 5).map((t) => (
                <div key={t.id} className="py-2.5 flex items-center justify-between text-xs">
                  <div>
                    <span className="font-semibold text-text">{t.title}</span>
                    <span className="text-[11px] text-text-muted block mt-0.5">
                      PIC: {t.assignee_name || 'Belum ditugaskan'}
                    </span>
                  </div>
                  <span className="text-rose-600 font-bold whitespace-nowrap">
                    {t.due_date ? new Date(t.due_date).toLocaleDateString('id-ID') : '-'}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Tasks in review */}
        <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
          <h2 className="text-sm font-bold text-text mb-3">Task Menunggu Review / Approval</h2>
          <div className="divide-y divide-border">
            {data.pending_tasks.length === 0 ? (
              <p className="text-xs text-text-muted py-4">Tidak ada task yang sedang menunggu persetujuan.</p>
            ) : (
              data.pending_tasks.slice(0, 5).map((t) => (
                <div key={t.id} className="py-2.5 flex items-center justify-between text-xs">
                  <div>
                    <span className="font-semibold text-text">{t.title}</span>
                    <span className="text-[11px] text-text-muted block mt-0.5">
                      Kolom: {t.column_name} | PIC: {t.assignee_name || '-'}
                    </span>
                  </div>
                  <Link
                    to={`/projects/${t.project_id}`}
                    className="px-2 py-1 text-[11px] font-medium text-primary bg-primary/10 rounded-lg hover:bg-primary/20"
                  >
                    Tinjau di Board &rarr;
                  </Link>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ─── TL VIEW ─────────────────────────────────────────────────────────────────

interface TLViewProps {
  data: TLDashboardData
  activeTab: 'team' | 'my_tasks'
  onTabChange: (tab: 'team' | 'my_tasks') => void
}

function TLView({ data, activeTab, onTabChange }: TLViewProps) {
  return (
    <div className="space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Total Task Tim</span>
          <div className="text-2xl font-bold text-text mt-1">{data.totals.total_tasks}</div>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-rose-200 bg-rose-50/20 shadow-sm">
          <span className="text-xs text-rose-600 font-medium">Task Terlambat</span>
          <div className="text-2xl font-bold text-rose-600 mt-1">{data.totals.overdue_tasks}</div>
        </div>

        <div className="bg-surface p-4 rounded-xl border border-amber-200 bg-amber-50/20 shadow-sm">
          <span className="text-xs text-amber-700 font-medium">Menunggu Persetujuan PM</span>
          <div className="text-2xl font-bold text-amber-700 mt-1">{data.totals.pending_approval}</div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-border gap-4">
        <button
          onClick={() => onTabChange('team')}
          className={`pb-3 text-xs font-bold transition-colors border-b-2 ${
            activeTab === 'team'
              ? 'border-primary text-primary'
              : 'border-transparent text-text-muted hover:text-text'
          }`}
        >
          Ringkasan Tim & Beban Kerja
        </button>
        <button
          onClick={() => onTabChange('my_tasks')}
          className={`pb-3 text-xs font-bold transition-colors border-b-2 flex items-center gap-1.5 ${
            activeTab === 'my_tasks'
              ? 'border-primary text-primary'
              : 'border-transparent text-text-muted hover:text-text'
          }`}
        >
          <span>Tugas Saya</span>
          <span className="px-1.5 py-0.2 rounded-full text-[10px] bg-primary/10 text-primary font-bold">
            {data.my_tasks.length}
          </span>
        </button>
      </div>

      {activeTab === 'team' ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Status by Board Column */}
          <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
            <h2 className="text-sm font-bold text-text mb-4">Distribusi Status Task (Kolom Board)</h2>
            <div className="space-y-3">
              {data.by_column.map((col) => (
                <div key={col.column_name}>
                  <div className="flex justify-between text-xs mb-1">
                    <span className="font-semibold text-text">{col.column_name}</span>
                    <span className="text-text-muted font-bold">{col.count} task</span>
                  </div>
                  <div className="w-full bg-border rounded-full h-2">
                    <div
                      className="bg-primary h-full rounded-full"
                      style={{
                        width: `${
                          data.totals.total_tasks > 0
                            ? (col.count / data.totals.total_tasks) * 100
                            : 0
                        }%`,
                      }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Member Workload */}
          <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
            <h2 className="text-sm font-bold text-text mb-4">Beban Kerja Anggota Tim (Aktif)</h2>
            <div className="divide-y divide-border">
              {data.workload.length === 0 ? (
                <p className="text-xs text-text-muted py-4">Belum ada penugasan task aktif.</p>
              ) : (
                data.workload.map((m) => (
                  <div key={m.user_id} className="py-2.5 flex items-center justify-between text-xs">
                    <span className="font-semibold text-text">{m.full_name}</span>
                    <span className="px-2 py-0.5 rounded-full bg-primary/10 text-primary font-bold">
                      {m.active_tasks} task aktif
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      ) : (
        /* My Tasks View */
        <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
          <h2 className="text-sm font-bold text-text mb-3">Tugas Ditugaskan kepada Anda</h2>
          <div className="divide-y divide-border">
            {data.my_tasks.length === 0 ? (
              <p className="text-xs text-text-muted py-4">Tidak ada task yang ditugaskan kepada Anda saat ini.</p>
            ) : (
              data.my_tasks.map((t) => (
                <div key={t.id} className="py-3 flex items-center justify-between text-xs">
                  <div>
                    <span className="font-semibold text-text">{t.title}</span>
                    <span className="text-[11px] text-text-muted block mt-0.5">
                      Kolom: {t.column_name} | Progres: {t.progress_percent}%
                    </span>
                  </div>
                  <div className="text-right">
                    <span className="text-text-muted block text-[11px]">
                      Tenggat: {t.due_date ? new Date(t.due_date).toLocaleDateString('id-ID') : '-'}
                    </span>
                    <Link
                      to={`/projects/${t.project_id}`}
                      className="text-primary hover:underline text-[11px] font-medium"
                    >
                      Buka di Papan &rarr;
                    </Link>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}

// ─── AUDIT VIEW ──────────────────────────────────────────────────────────────

function AuditView({ data }: { data: AuditDashboardData }) {
  const sum = data.findings_summary
  return (
    <div className="space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-red-200 bg-red-50/20 shadow-sm">
          <span className="text-xs text-red-600 font-medium">Critical Findings</span>
          <div className="text-2xl font-bold text-red-700 mt-1">{sum?.critical_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-orange-200 bg-orange-50/20 shadow-sm">
          <span className="text-xs text-orange-600 font-medium">High Findings</span>
          <div className="text-2xl font-bold text-orange-700 mt-1">{sum?.high_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-amber-200 bg-amber-50/20 shadow-sm">
          <span className="text-xs text-amber-600 font-medium">Medium Findings</span>
          <div className="text-2xl font-bold text-amber-700 mt-1">{sum?.medium_count ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-blue-200 bg-blue-50/20 shadow-sm">
          <span className="text-xs text-blue-600 font-medium">Low Findings</span>
          <div className="text-2xl font-bold text-blue-700 mt-1">{sum?.low_count ?? 0}</div>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Open findings needing response or verification */}
        <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-bold text-text">Temuan Audit Aktif</h2>
            <Link to="/findings" className="text-xs text-primary font-medium hover:underline">
              Kelola Temuan &rarr;
            </Link>
          </div>
          <div className="divide-y divide-border">
            {data.open_findings.length === 0 ? (
              <p className="text-xs text-text-muted py-4">Tidak ada temuan audit terbuka.</p>
            ) : (
              data.open_findings.slice(0, 6).map((f) => (
                <div key={f.id} className="py-2.5 flex items-center justify-between text-xs">
                  <div>
                    <span className="font-mono font-bold text-text mr-2">{f.code}</span>
                    <span className="font-semibold text-text">{f.title}</span>
                    <span className="text-[11px] text-text-muted block mt-0.5">
                      Proyek: {f.project_name || '-'}
                    </span>
                  </div>
                  <span className="px-2 py-0.5 rounded font-bold uppercase text-[10px] bg-rose-100 text-rose-800">
                    {f.severity}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Audit trail quick logs */}
        <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-bold text-text">Log Audit Terbaru</h2>
            <Link to="/audit-trail" className="text-xs text-primary font-medium hover:underline">
              Ekspor / Audit Trail &rarr;
            </Link>
          </div>
          <div className="divide-y divide-border">
            {(!data.recent_activities || data.recent_activities.length === 0) ? (
              <p className="text-xs text-text-muted py-4">Belum ada aktivitas tercatat.</p>
            ) : (
              data.recent_activities.slice(0, 6).map((act) => (
                <div key={act.id} className="py-2.5 flex items-center justify-between text-xs">
                  <div className="flex items-center gap-2">
                    <span className="px-1.5 py-0.5 rounded font-mono text-[10px] bg-slate-100 text-slate-800">
                      {act.action}
                    </span>
                    <span className="text-text capitalize">{act.entity_type}</span>
                  </div>
                  <span className="text-text-muted text-[11px]">
                    {new Date(act.created_at).toLocaleTimeString('id-ID')}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

// ─── RISK MANAGEMENT VIEW ────────────────────────────────────────────────────

function RiskManagementView({ data }: { data: RiskManagementDashboardData }) {
  const sum = data.risk_summary
  return (
    <div className="space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Total Risiko</span>
          <div className="text-2xl font-bold text-text mt-1">{sum?.total_risks ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <span className="text-xs text-text-muted font-medium">Risiko Terbuka (Open)</span>
          <div className="text-2xl font-bold text-amber-600 mt-1">{sum?.open_risks ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-red-200 bg-red-50/20 shadow-sm">
          <span className="text-xs text-red-600 font-medium">High & Critical</span>
          <div className="text-2xl font-bold text-red-700 mt-1">{sum?.high_critical_risks ?? 0}</div>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-rose-200 bg-rose-50/20 shadow-sm">
          <span className="text-xs text-rose-600 font-medium">Belum Dimitigasi</span>
          <div className="text-2xl font-bold text-rose-700 mt-1">{sum?.unmitigated_risks ?? 0}</div>
        </div>
      </div>

      {/* Critical & High Risks list */}
      <div className="bg-surface rounded-xl border border-border shadow-sm p-5">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-bold text-text">Daftar Risiko Kritis & Tinggi Memerlukan Tindakan</h2>
          <Link to="/risks" className="text-xs text-primary font-medium hover:underline">
            Buka Risk Register &rarr;
          </Link>
        </div>
        <div className="divide-y divide-border">
          {data.high_critical_risks.length === 0 ? (
            <p className="text-xs text-text-muted py-4">Tidak ada risiko berstatus critical saat ini.</p>
          ) : (
            data.high_critical_risks.map((r) => (
              <div key={r.id} className="py-2.5 flex items-center justify-between text-xs">
                <div>
                  <span className="font-mono font-bold text-text mr-2">{r.code}</span>
                  <span className="font-semibold text-text">{r.title}</span>
                  <span className="text-[11px] text-text-muted block mt-0.5">
                    Proyek: {r.project_name || '-'} | Kategori: {r.category}
                  </span>
                </div>
                <div className="text-right">
                  <span className="px-2 py-0.5 rounded font-bold uppercase text-[10px] bg-red-100 text-red-800">
                    {r.level || 'Critical'}
                  </span>
                  <span className="text-[10px] text-text-muted block mt-0.5">
                    Skor: {r.score ?? '-'}
                  </span>
                </div>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
