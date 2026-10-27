import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { fetchFindings, fetchFindingSummary } from './api'
import { fetchProjects } from '../projects/api'
import { CreateFindingModal } from './CreateFindingModal'
import { FindingDetailDrawer } from './FindingDetailDrawer'
import { useAuth } from '../auth/useAuth'
import type { AuditFinding, FindingSummary, FindingSeverity, FindingStatus } from './types'
import type { Project } from '../projects/types'

const SEVERITY_BADGES: Record<FindingSeverity, string> = {
  critical: 'bg-red-100 text-red-800 border-red-300',
  high: 'bg-orange-100 text-orange-800 border-orange-300',
  medium: 'bg-amber-100 text-amber-800 border-amber-300',
  low: 'bg-blue-100 text-blue-800 border-blue-300',
}

const STATUS_BADGES: Record<FindingStatus, string> = {
  open: 'bg-rose-100 text-rose-800 border-rose-300',
  responded: 'bg-amber-100 text-amber-800 border-amber-300',
  closed: 'bg-emerald-100 text-emerald-800 border-emerald-300',
}

export default function FindingListPage() {
  const { user } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()

  const [findings, setFindings] = useState<AuditFinding[]>([])
  const [summary, setSummary] = useState<FindingSummary | null>(null)
  const [projects, setProjects] = useState<Project[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Filters from URL query params or state
  const selectedProject = searchParams.get('project_id') || ''
  const selectedSeverity = (searchParams.get('severity') as FindingSeverity) || ''
  const selectedStatus = (searchParams.get('status') as FindingStatus) || ''
  const searchQuery = searchParams.get('search') || ''
  const currentPage = parseInt(searchParams.get('page') || '1', 10)

  // Modal / Drawer state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [selectedFinding, setSelectedFinding] = useState<AuditFinding | null>(null)
  const [isDetailOpen, setIsDetailOpen] = useState(false)

  const isAudit = user?.role === 'audit'

  const loadData = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const [findingsRes, summaryRes] = await Promise.all([
        fetchFindings({
          project_id: selectedProject || undefined,
          severity: selectedSeverity || undefined,
          status: selectedStatus || undefined,
          search: searchQuery || undefined,
          page: currentPage,
          per_page: 20,
        }),
        fetchFindingSummary(selectedProject || undefined),
      ])

      setFindings(findingsRes.findings)
      setTotalCount(findingsRes.total)
      setSummary(summaryRes)
    } catch (err: unknown) {
      if (err instanceof Error) setError(err.message)
      else setError('Gagal memuat data temuan audit')
    } finally {
      setIsLoading(false)
    }
  }, [selectedProject, selectedSeverity, selectedStatus, searchQuery, currentPage])

  useEffect(() => {
    loadData()
  }, [loadData])

  useEffect(() => {
    async function loadProjectsList() {
      try {
        const projs = await fetchProjects()
        setProjects(projs)
      } catch {
        // ignore
      }
    }
    loadProjectsList()
  }, [])

  const updateFilter = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value) {
      next.set(key, value)
    } else {
      next.delete(key)
    }
    next.set('page', '1')
    setSearchParams(next)
  }

  const handleFindingCreated = (newFinding: AuditFinding) => {
    setFindings((prev) => [newFinding, ...prev])
    loadData()
  }

  const handleFindingUpdated = (updated: AuditFinding) => {
    setFindings((prev) => prev.map((f) => (f.id === updated.id ? updated : f)))
    setSelectedFinding(updated)
    loadData()
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-surface p-6 rounded-2xl border border-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <span className="p-2 rounded-xl bg-amber-500/10 text-amber-600">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </span>
            <h1 className="text-xl font-bold text-text">Temuan Audit</h1>
          </div>
          <p className="text-sm text-text-muted mt-1">
            Pengelolaan observasi audit kepatuhan, rencana tanggapan manajemen, dan verifikasi penyelesaian perbankan.
          </p>
        </div>

        {isAudit && (
          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="px-4 py-2.5 text-xs font-bold text-white bg-primary hover:bg-primary-hover rounded-xl shadow-sm transition-colors flex items-center justify-center gap-2 self-start md:self-auto"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Buat Temuan Baru
          </button>
        )}
      </div>

      {/* Summary KPI Cards */}
      {summary && (
        <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-5 gap-4">
          <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
            <span className="text-xs text-text-muted font-medium">Total Temuan</span>
            <div className="text-2xl font-bold text-text mt-1">{summary.total_findings}</div>
          </div>
          <div className="bg-surface p-4 rounded-xl border border-rose-200 shadow-sm bg-rose-50/20">
            <span className="text-xs text-rose-600 font-medium">Temuan Terbuka (Open)</span>
            <div className="text-2xl font-bold text-rose-600 mt-1">{summary.open_findings}</div>
          </div>
          <div className="bg-surface p-4 rounded-xl border border-red-200 shadow-sm bg-red-50/20">
            <span className="text-xs text-red-600 font-medium">Critical & High</span>
            <div className="text-2xl font-bold text-red-700 mt-1">
              {summary.critical_count + summary.high_count}
            </div>
            <div className="text-[10px] text-text-muted mt-0.5">
              Crit: {summary.critical_count} | High: {summary.high_count}
            </div>
          </div>
          <div className="bg-surface p-4 rounded-xl border border-amber-200 shadow-sm bg-amber-50/20">
            <span className="text-xs text-amber-700 font-medium">Menunggu Verifikasi</span>
            <div className="text-2xl font-bold text-amber-700 mt-1">{summary.responded_count}</div>
            <div className="text-[10px] text-text-muted mt-0.5">Status Responded</div>
          </div>
          <div className="bg-surface p-4 rounded-xl border border-emerald-200 shadow-sm bg-emerald-50/20 col-span-2 sm:col-span-1">
            <span className="text-xs text-emerald-700 font-medium">Selesai (Closed)</span>
            <div className="text-2xl font-bold text-emerald-700 mt-1">
              {summary.total_findings - (summary.open_findings + summary.responded_count)}
            </div>
          </div>
        </div>
      )}

      {/* Filter Toolbar */}
      <div className="bg-surface p-4 rounded-xl border border-border shadow-sm flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2 flex-1">
          {/* Project Filter */}
          <select
            value={selectedProject}
            onChange={(e) => updateFilter('project_id', e.target.value)}
            className="px-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          >
            <option value="">Semua Proyek</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                [{p.code}] {p.name}
              </option>
            ))}
          </select>

          {/* Severity Filter */}
          <select
            value={selectedSeverity}
            onChange={(e) => updateFilter('severity', e.target.value)}
            className="px-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          >
            <option value="">Semua Keparahan</option>
            <option value="critical">Critical</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>

          {/* Status Filter */}
          <select
            value={selectedStatus}
            onChange={(e) => updateFilter('status', e.target.value)}
            className="px-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          >
            <option value="">Semua Status</option>
            <option value="open">Open (Terbuka)</option>
            <option value="responded">Responded (Ditanggapi)</option>
            <option value="closed">Closed (Ditutup)</option>
          </select>

          {(selectedProject || selectedSeverity || selectedStatus || searchQuery) && (
            <button
              onClick={() => setSearchParams(new URLSearchParams())}
              className="px-2.5 py-1.5 text-xs text-text-muted hover:text-text bg-surface-muted hover:bg-slate-200 rounded-lg transition-colors"
            >
              Reset Filter
            </button>
          )}
        </div>

        {/* Search Input */}
        <div className="relative min-w-[240px]">
          <input
            type="text"
            placeholder="Cari kode atau judul temuan..."
            value={searchQuery}
            onChange={(e) => updateFilter('search', e.target.value)}
            className="w-full pl-8 pr-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          />
          <svg className="w-3.5 h-3.5 text-text-muted absolute left-2.5 top-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      {/* Findings Table */}
      <div className="bg-surface rounded-xl border border-border shadow-sm overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-text-muted text-sm flex items-center justify-center gap-2">
            <div className="w-4 h-4 border-2 border-primary/20 border-t-primary rounded-full animate-spin" />
            Memuat temuan audit...
          </div>
        ) : error ? (
          <div className="p-8 text-center text-red-600 text-sm">{error}</div>
        ) : findings.length === 0 ? (
          <div className="p-12 text-center">
            <svg className="w-12 h-12 text-text-muted/40 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
            <h3 className="text-sm font-bold text-text">Tidak ada temuan audit</h3>
            <p className="text-xs text-text-muted mt-1">
              Tidak ada data yang cocok dengan kriteria pencarian atau filter yang dipilih.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-border bg-surface-muted/60 text-text-muted font-semibold">
                  <th className="py-3 px-4">Kode</th>
                  <th className="py-3 px-4">Judul & Proyek</th>
                  <th className="py-3 px-4">Severity</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4">Auditor & Tanggal</th>
                  <th className="py-3 px-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {findings.map((f) => (
                  <tr
                    key={f.id}
                    onClick={() => {
                      setSelectedFinding(f)
                      setIsDetailOpen(true)
                    }}
                    className="hover:bg-surface-muted/40 cursor-pointer transition-colors"
                  >
                    <td className="py-3 px-4 font-mono font-bold text-text whitespace-nowrap">
                      {f.code}
                    </td>
                    <td className="py-3 px-4 max-w-md">
                      <div className="font-semibold text-text truncate">{f.title}</div>
                      <div className="text-[11px] text-text-muted flex items-center gap-1.5 mt-0.5">
                        <span className="font-medium text-primary">[{f.project_code || 'PRJ'}]</span>
                        <span>{f.project_name || 'Proyek'}</span>
                        {f.task_title && (
                          <span className="px-1.5 py-0.2 rounded bg-surface border border-border text-[10px] text-text-muted truncate max-w-[150px]">
                            Task: {f.task_title}
                          </span>
                        )}
                      </div>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[11px] font-bold border capitalize ${
                          SEVERITY_BADGES[f.severity] || 'bg-slate-100 text-slate-800'
                        }`}
                      >
                        {f.severity}
                      </span>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <span
                        className={`px-2 py-0.5 rounded-full text-[11px] font-bold border capitalize ${
                          STATUS_BADGES[f.status] || 'bg-slate-100 text-slate-800'
                        }`}
                      >
                        {f.status === 'open' && 'Open'}
                        {f.status === 'responded' && 'Responded'}
                        {f.status === 'closed' && 'Closed'}
                      </span>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap text-text-muted">
                      <div>{f.raised_by_name || 'Auditor'}</div>
                      <div className="text-[10px]">
                        {new Date(f.created_at).toLocaleDateString('id-ID')}
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right whitespace-nowrap">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setSelectedFinding(f)
                          setIsDetailOpen(true)
                        }}
                        className="px-2.5 py-1 text-xs font-medium text-primary hover:text-primary-hover hover:underline"
                      >
                        Detail &rarr;
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Footer / Total count */}
        {!isLoading && findings.length > 0 && (
          <div className="py-3 px-4 border-t border-border bg-surface-muted/30 text-xs text-text-muted flex items-center justify-between">
            <span>Menampilkan {findings.length} dari {totalCount} temuan audit</span>
          </div>
        )}
      </div>

      {/* Modals & Drawers */}
      <CreateFindingModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        onSuccess={handleFindingCreated}
      />

      <FindingDetailDrawer
        finding={selectedFinding}
        isOpen={isDetailOpen}
        onClose={() => {
          setIsDetailOpen(false)
          setSelectedFinding(null)
        }}
        onUpdated={handleFindingUpdated}
      />
    </div>
  )
}
