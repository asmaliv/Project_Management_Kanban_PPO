import { useState, useEffect, useCallback } from 'react'
import { fetchAuditLogs, exportAuditLogsCSV } from './api'
import { fetchProjects } from '../projects/api'
import { useAuth } from '../auth/useAuth'
import type { AuditLog } from './types'
import type { Project } from '../projects/types'

const ENTITY_OPTIONS = [
  { value: '', label: 'Semua Entitas' },
  { value: 'project', label: 'Proyek (Project)' },
  { value: 'task', label: 'Tugas (Task)' },
  { value: 'task_comment', label: 'Komentar Tugas' },
  { value: 'risk', label: 'Risiko (Risk)' },
  { value: 'audit_finding', label: 'Temuan Audit' },
  { value: 'auth', label: 'Autentikasi (Auth)' },
  { value: 'session', label: 'Sesi Pengguna' },
  { value: 'user', label: 'Pengguna (User)' },
]

export default function AuditTrailPage() {
  const { user } = useAuth()
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [projects, setProjects] = useState<Project[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isExporting, setIsExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [exportMessage, setExportMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  // Filters
  const [selectedProject, setSelectedProject] = useState('')
  const [selectedEntity, setSelectedEntity] = useState('')
  const [actionQuery, setActionQuery] = useState('')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')
  const [offset, setOffset] = useState(0)
  const limit = 25

  // Inspect Modal
  const [inspectLog, setInspectLog] = useState<AuditLog | null>(null)

  const isAudit = user?.role === 'audit'

  const loadLogs = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await fetchAuditLogs({
        project_id: selectedProject || undefined,
        entity_type: selectedEntity || undefined,
        action: actionQuery.trim() || undefined,
        from_date: fromDate ? `${fromDate}T00:00:00Z` : undefined,
        to_date: toDate ? `${toDate}T23:59:59Z` : undefined,
        limit,
        offset,
      })
      setLogs(res.logs)
      setTotalCount(res.total)
    } catch (err: unknown) {
      if (err instanceof Error) setError(err.message)
      else setError('Gagal memuat log audit trail')
    } finally {
      setIsLoading(false)
    }
  }, [selectedProject, selectedEntity, actionQuery, fromDate, toDate, offset])

  useEffect(() => {
    loadLogs()
  }, [loadLogs])

  useEffect(() => {
    async function getProjects() {
      try {
        const projs = await fetchProjects()
        setProjects(projs)
      } catch {
        // ignore
      }
    }
    getProjects()
  }, [])

  const handleExportCSV = async () => {
    setIsExporting(true)
    setExportMessage(null)
    try {
      await exportAuditLogsCSV({
        project_id: selectedProject || undefined,
        entity_type: selectedEntity || undefined,
        action: actionQuery.trim() || undefined,
        from_date: fromDate ? `${fromDate}T00:00:00Z` : undefined,
        to_date: toDate ? `${toDate}T23:59:59Z` : undefined,
      })
      setExportMessage({
        type: 'success',
        text: 'File CSV berhasil diunduh. Kolom aman telah diekspor.',
      })
    } catch (err: unknown) {
      if (err instanceof Error) {
        setExportMessage({ type: 'error', text: err.message })
      } else {
        setExportMessage({ type: 'error', text: 'Gagal mengekspor CSV' })
      }
    } finally {
      setIsExporting(false)
    }
  }

  const resetFilters = () => {
    setSelectedProject('')
    setSelectedEntity('')
    setActionQuery('')
    setFromDate('')
    setToDate('')
    setOffset(0)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-surface p-6 rounded-2xl border border-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <span className="p-2 rounded-xl bg-purple-500/10 text-purple-600">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </span>
            <h1 className="text-xl font-bold text-text">Audit Trail Perbankan</h1>
          </div>
          <p className="text-sm text-text-muted mt-1">
            Log jejak audit transaksi sistem yang bersifat <em>append-only</em> dan tidak dapat diubah (immutable).
          </p>
        </div>

        {isAudit && (
          <div className="flex flex-col items-end gap-1">
            <button
              onClick={handleExportCSV}
              disabled={isExporting}
              className="px-4 py-2.5 text-xs font-bold text-slate-800 bg-surface-muted hover:bg-slate-200 border border-border rounded-xl shadow-sm transition-colors flex items-center gap-2 self-start md:self-auto disabled:opacity-50"
            >
              {isExporting ? (
                <div className="w-3.5 h-3.5 border-2 border-slate-600 border-t-transparent rounded-full animate-spin" />
              ) : (
                <svg className="w-4 h-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                </svg>
              )}
              {isExporting ? 'Mengekspor...' : 'Ekspor CSV'}
            </button>
            <span className="text-[10px] text-text-muted">Rate limit: 5 ekspor / menit</span>
          </div>
        )}
      </div>

      {exportMessage && (
        <div
          className={`p-3 rounded-xl border text-xs flex items-center justify-between ${
            exportMessage.type === 'success'
              ? 'bg-emerald-50 border-emerald-200 text-emerald-800'
              : 'bg-red-50 border-red-200 text-red-800'
          }`}
        >
          <span>{exportMessage.text}</span>
          <button
            onClick={() => setExportMessage(null)}
            className="text-text-muted hover:text-text text-sm font-bold ml-2"
          >
            &times;
          </button>
        </div>
      )}

      {/* Filter Toolbar */}
      <div className="bg-surface p-4 rounded-xl border border-border shadow-sm space-y-3">
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-5 gap-3">
          {/* Project Filter */}
          <div>
            <label className="block text-[11px] font-semibold text-text-muted mb-1">Proyek</label>
            <select
              value={selectedProject}
              onChange={(e) => {
                setSelectedProject(e.target.value)
                setOffset(0)
              }}
              className="w-full px-2.5 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
            >
              <option value="">Semua Proyek</option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  [{p.code}] {p.name}
                </option>
              ))}
            </select>
          </div>

          {/* Entity Filter */}
          <div>
            <label className="block text-[11px] font-semibold text-text-muted mb-1">Tipe Entitas</label>
            <select
              value={selectedEntity}
              onChange={(e) => {
                setSelectedEntity(e.target.value)
                setOffset(0)
              }}
              className="w-full px-2.5 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
            >
              {ENTITY_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>

          {/* Action Search */}
          <div>
            <label className="block text-[11px] font-semibold text-text-muted mb-1">Nama Aksi</label>
            <input
              type="text"
              placeholder="Contoh: task.move, risk.close"
              value={actionQuery}
              onChange={(e) => {
                setActionQuery(e.target.value)
                setOffset(0)
              }}
              className="w-full px-2.5 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
            />
          </div>

          {/* Date From */}
          <div>
            <label className="block text-[11px] font-semibold text-text-muted mb-1">Dari Tanggal</label>
            <input
              type="date"
              value={fromDate}
              onChange={(e) => {
                setFromDate(e.target.value)
                setOffset(0)
              }}
              className="w-full px-2.5 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
            />
          </div>

          {/* Date To */}
          <div>
            <label className="block text-[11px] font-semibold text-text-muted mb-1">Sampai Tanggal</label>
            <input
              type="date"
              value={toDate}
              onChange={(e) => {
                setToDate(e.target.value)
                setOffset(0)
              }}
              className="w-full px-2.5 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
            />
          </div>
        </div>

        {(selectedProject || selectedEntity || actionQuery || fromDate || toDate) && (
          <div className="flex justify-end pt-2 border-t border-border">
            <button
              onClick={resetFilters}
              className="px-3 py-1 text-xs text-text-muted hover:text-text bg-surface-muted hover:bg-slate-200 rounded-lg transition-colors"
            >
              Reset Semua Filter
            </button>
          </div>
        )}
      </div>

      {/* Audit Log Table */}
      <div className="bg-surface rounded-xl border border-border shadow-sm overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-text-muted text-sm flex items-center justify-center gap-2">
            <div className="w-4 h-4 border-2 border-primary/20 border-t-primary rounded-full animate-spin" />
            Memuat audit trail...
          </div>
        ) : error ? (
          <div className="p-8 text-center text-red-600 text-sm">{error}</div>
        ) : logs.length === 0 ? (
          <div className="p-12 text-center">
            <svg className="w-12 h-12 text-text-muted/40 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <h3 className="text-sm font-bold text-text">Tidak ada aktivitas audit trail</h3>
            <p className="text-xs text-text-muted mt-1">
              Tidak ada catatan audit yang cocok dengan filter yang ditentukan.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-border bg-surface-muted/60 text-text-muted font-semibold">
                  <th className="py-3 px-4">Waktu (WIB / Lokal)</th>
                  <th className="py-3 px-4">Aksi</th>
                  <th className="py-3 px-4">Entitas</th>
                  <th className="py-3 px-4">Aktor / Role</th>
                  <th className="py-3 px-4">IP & Request ID</th>
                  <th className="py-3 px-4 text-right">Detail Data</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {logs.map((log) => (
                  <tr
                    key={log.id}
                    onClick={() => setInspectLog(log)}
                    className="hover:bg-surface-muted/40 cursor-pointer transition-colors"
                  >
                    <td className="py-3 px-4 whitespace-nowrap text-text-muted">
                      <div className="font-mono text-text font-medium">
                        {new Date(log.created_at).toLocaleTimeString('id-ID')}
                      </div>
                      <div className="text-[10px]">
                        {new Date(log.created_at).toLocaleDateString('id-ID', {
                          day: 'numeric',
                          month: 'short',
                          year: 'numeric',
                        })}
                      </div>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <span className="px-2 py-0.5 rounded font-mono font-semibold text-[11px] bg-slate-100 text-slate-800 border border-slate-200">
                        {log.action}
                      </span>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      <div className="font-semibold text-text capitalize">{log.entity_type}</div>
                      <div className="text-[10px] font-mono text-text-muted truncate max-w-[120px]">
                        {log.entity_id}
                      </div>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap">
                      {log.actor_role && (
                        <span className="px-1.5 py-0.2 rounded text-[10px] font-bold uppercase bg-primary/10 text-primary">
                          {log.actor_role}
                        </span>
                      )}
                      <div className="text-[10px] font-mono text-text-muted truncate max-w-[120px] mt-0.5">
                        {log.actor_id}
                      </div>
                    </td>
                    <td className="py-3 px-4 whitespace-nowrap text-text-muted text-[10px] font-mono">
                      <div>{log.ip || '-'}</div>
                      <div className="truncate max-w-[110px]" title={log.request_id || ''}>
                        {log.request_id ? `req:${log.request_id.slice(0, 8)}...` : '-'}
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right whitespace-nowrap">
                      <button
                        onClick={(e) => {
                          e.stopPropagation()
                          setInspectLog(log)
                        }}
                        className="px-2.5 py-1 text-xs font-medium text-primary hover:text-primary-hover hover:underline"
                      >
                        Lihat Diff &rarr;
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination Bar */}
        <div className="py-3 px-4 border-t border-border bg-surface-muted/30 text-xs text-text-muted flex items-center justify-between">
          <span>
            Menampilkan {logs.length} dari {totalCount} riwayat log
          </span>
          <div className="flex gap-2">
            <button
              onClick={() => setOffset(Math.max(0, offset - limit))}
              disabled={offset === 0}
              className="px-3 py-1 bg-surface border border-border rounded-lg disabled:opacity-40 hover:bg-surface-muted transition-colors"
            >
              &larr; Sebelumnya
            </button>
            <button
              onClick={() => setOffset(offset + limit)}
              disabled={offset + limit >= totalCount}
              className="px-3 py-1 bg-surface border border-border rounded-lg disabled:opacity-40 hover:bg-surface-muted transition-colors"
            >
              Selanjutnya &rarr;
            </button>
          </div>
        </div>
      </div>

      {/* Inspect Modal: Before & After Diff */}
      {inspectLog && (
        <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="relative bg-surface rounded-2xl shadow-2xl max-w-2xl w-full border border-border overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-surface-muted">
              <div>
                <h3 className="text-base font-bold text-text flex items-center gap-2">
                  <span className="font-mono text-xs px-2 py-0.5 rounded bg-primary/10 text-primary">
                    {inspectLog.action}
                  </span>
                  Detail Data Jejak Audit
                </h3>
                <p className="text-xs text-text-muted mt-0.5">
                  ID: <span className="font-mono">{inspectLog.id}</span> | Waktu:{' '}
                  {new Date(inspectLog.created_at).toLocaleString('id-ID')}
                </p>
              </div>
              <button
                onClick={() => setInspectLog(null)}
                className="text-text-muted hover:text-text rounded-lg p-1.5 transition-colors"
              >
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="p-6 space-y-4 max-h-[70vh] overflow-y-auto">
              <div className="grid grid-cols-2 gap-3 text-xs bg-surface-muted p-3 rounded-xl border border-border">
                <div>
                  <span className="text-text-muted block">Entitas:</span>
                  <span className="font-semibold text-text capitalize">
                    {inspectLog.entity_type} ({inspectLog.entity_id})
                  </span>
                </div>
                <div>
                  <span className="text-text-muted block">Aktor:</span>
                  <span className="font-semibold text-text">
                    Role: {inspectLog.actor_role || '-'} (ID: {inspectLog.actor_id})
                  </span>
                </div>
                <div>
                  <span className="text-text-muted block">IP Address:</span>
                  <span className="font-mono text-text">{inspectLog.ip || '-'}</span>
                </div>
                <div>
                  <span className="text-text-muted block">Request ID:</span>
                  <span className="font-mono text-text truncate block">{inspectLog.request_id || '-'}</span>
                </div>
              </div>

              {/* Before vs After comparison */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div>
                  <h4 className="text-xs font-bold text-text-muted uppercase tracking-wider mb-1.5 flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-rose-500" /> Data Sebelum (Before):
                  </h4>
                  <pre className="p-3 bg-surface-muted border border-border rounded-xl text-xs font-mono text-text overflow-x-auto max-h-60 leading-relaxed">
                    {inspectLog.before && Object.keys(inspectLog.before).length > 0
                      ? JSON.stringify(inspectLog.before, null, 2)
                      : '(Kosong / Entitas Baru Dibuat)'}
                  </pre>
                </div>

                <div>
                  <h4 className="text-xs font-bold text-text-muted uppercase tracking-wider mb-1.5 flex items-center gap-1.5">
                    <span className="w-2 h-2 rounded-full bg-emerald-500" /> Data Sesudah (After):
                  </h4>
                  <pre className="p-3 bg-surface-muted border border-border rounded-xl text-xs font-mono text-text overflow-x-auto max-h-60 leading-relaxed">
                    {inspectLog.after && Object.keys(inspectLog.after).length > 0
                      ? JSON.stringify(inspectLog.after, null, 2)
                      : '(Kosong / Entitas Dihapus)'}
                  </pre>
                </div>
              </div>
            </div>

            <div className="flex justify-end p-4 border-t border-border bg-surface-muted">
              <button
                onClick={() => setInspectLog(null)}
                className="px-4 py-2 text-xs font-medium text-text bg-surface hover:bg-slate-200 border border-border rounded-xl transition-colors"
              >
                Tutup
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
