import React, { useState } from 'react'
import { respondFinding, reopenFinding, closeFinding } from './api'
import { useAuth } from '../auth/useAuth'
import type { AuditFinding } from './types'

interface FindingDetailDrawerProps {
  finding: AuditFinding | null
  isOpen: boolean
  onClose: () => void
  onUpdated: (updated: AuditFinding) => void
}

const SEVERITY_BADGES = {
  critical: 'bg-red-100 text-red-800 border-red-300',
  high: 'bg-orange-100 text-orange-800 border-orange-300',
  medium: 'bg-amber-100 text-amber-800 border-amber-300',
  low: 'bg-blue-100 text-blue-800 border-blue-300',
}

const STATUS_BADGES = {
  open: 'bg-rose-100 text-rose-800 border-rose-300',
  responded: 'bg-amber-100 text-amber-800 border-amber-300',
  closed: 'bg-emerald-100 text-emerald-800 border-emerald-300',
}

export const FindingDetailDrawer: React.FC<FindingDetailDrawerProps> = ({
  finding,
  isOpen,
  onClose,
  onUpdated,
}) => {
  const { user } = useAuth()
  const [responseText, setResponseText] = useState('')
  const [reopenComment, setReopenComment] = useState('')
  const [showReopenInput, setShowReopenInput] = useState(false)
  const [showConfirmClose, setShowConfirmClose] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isOpen || !finding) return null

  const isAudit = user?.role === 'audit'
  const isPM = user?.role === 'project_manager'

  const handleSendResponse = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!responseText.trim()) {
      setError('Respons / rencana tindak lanjut wajib diisi.')
      return
    }

    setIsLoading(true)
    setError(null)
    try {
      const res = await respondFinding(finding.id, { response: responseText.trim() })
      onUpdated(res)
      setResponseText('')
    } catch (err: unknown) {
      if (err instanceof Error) setError(err.message)
      else setError('Gagal mengirim respons temuan')
    } finally {
      setIsLoading(false)
    }
  }

  const handleCloseFinding = async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await closeFinding(finding.id)
      onUpdated(res)
      setShowConfirmClose(false)
    } catch (err: unknown) {
      if (err instanceof Error) setError(err.message)
      else setError('Gagal menutup temuan audit')
    } finally {
      setIsLoading(false)
    }
  }

  const handleReopenFinding = async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await reopenFinding(finding.id, { comment: reopenComment.trim() || undefined })
      onUpdated(res)
      setShowReopenInput(false)
      setReopenComment('')
    } catch (err: unknown) {
      if (err instanceof Error) setError(err.message)
      else setError('Gagal membuka kembali temuan audit')
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-slate-900/40 backdrop-blur-sm flex justify-end animate-in fade-in duration-150">
      <div className="w-full max-w-xl bg-surface h-full shadow-2xl border-l border-border flex flex-col animate-in slide-in-from-right duration-200 overflow-hidden">
        {/* Header */}
        <div className="p-6 border-b border-border bg-surface-muted flex items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 mb-1.5 flex-wrap">
              <span className="font-mono text-xs font-bold px-2 py-0.5 rounded bg-surface border border-border text-text">
                {finding.code}
              </span>
              <span
                className={`text-xs px-2.5 py-0.5 rounded-full font-bold border capitalize ${
                  SEVERITY_BADGES[finding.severity] || 'bg-slate-100 text-slate-800'
                }`}
              >
                {finding.severity}
              </span>
              <span
                className={`text-xs px-2.5 py-0.5 rounded-full font-bold border capitalize ${
                  STATUS_BADGES[finding.status] || 'bg-slate-100 text-slate-800'
                }`}
              >
                {finding.status === 'open' && 'Terbuka (Open)'}
                {finding.status === 'responded' && 'Ditanggapi (Responded)'}
                {finding.status === 'closed' && 'Selesai / Ditutup (Closed)'}
              </span>
            </div>
            <h2 className="text-lg font-bold text-text leading-snug">{finding.title}</h2>
          </div>
          <button
            onClick={onClose}
            className="text-text-muted hover:text-text rounded-lg p-1.5 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-xs rounded-xl flex items-start gap-2">
              <svg className="w-4 h-4 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <span>{error}</span>
            </div>
          )}

          {/* Meta Info Grid */}
          <div className="grid grid-cols-2 gap-4 bg-surface-muted p-4 rounded-xl border border-border text-xs">
            <div>
              <span className="text-text-muted block font-medium">Proyek:</span>
              <span className="font-semibold text-text mt-0.5 block">
                [{finding.project_code || 'PRJ'}] {finding.project_name || 'Proyek'}
              </span>
            </div>
            <div>
              <span className="text-text-muted block font-medium">Task Terkait:</span>
              <span className="font-semibold text-text mt-0.5 block truncate">
                {finding.task_title || '-'}
              </span>
            </div>
            <div>
              <span className="text-text-muted block font-medium">Diterbitkan Oleh (Audit):</span>
              <span className="font-semibold text-text mt-0.5 block">
                {finding.raised_by_name || 'Auditor'}
              </span>
            </div>
            <div>
              <span className="text-text-muted block font-medium">Tanggal Temuan:</span>
              <span className="font-semibold text-text mt-0.5 block">
                {new Date(finding.created_at).toLocaleDateString('id-ID', {
                  day: 'numeric',
                  month: 'short',
                  year: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
            </div>
          </div>

          {/* Lifecycle Progress Bar */}
          <div className="border border-border p-4 rounded-xl bg-surface">
            <h4 className="text-xs font-bold text-text-muted uppercase tracking-wider mb-3">
              Alur Siklus Temuan Audit
            </h4>
            <div className="flex items-center justify-between text-xs font-semibold relative">
              <div className="flex flex-col items-center gap-1 z-10">
                <span className="w-6 h-6 rounded-full flex items-center justify-center bg-rose-500 text-white text-[11px] font-bold">
                  1
                </span>
                <span className="text-text">Open</span>
              </div>
              <div className="flex-1 h-0.5 mx-2 bg-border relative">
                <div
                  className={`h-full bg-amber-500 transition-all ${
                    finding.status === 'responded' || finding.status === 'closed' ? 'w-full' : 'w-0'
                  }`}
                />
              </div>
              <div className="flex flex-col items-center gap-1 z-10">
                <span
                  className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-bold ${
                    finding.status === 'responded' || finding.status === 'closed'
                      ? 'bg-amber-500 text-white'
                      : 'bg-surface-muted text-text-muted border border-border'
                  }`}
                >
                  2
                </span>
                <span className="text-text">Responded</span>
              </div>
              <div className="flex-1 h-0.5 mx-2 bg-border relative">
                <div
                  className={`h-full bg-emerald-500 transition-all ${
                    finding.status === 'closed' ? 'w-full' : 'w-0'
                  }`}
                />
              </div>
              <div className="flex flex-col items-center gap-1 z-10">
                <span
                  className={`w-6 h-6 rounded-full flex items-center justify-center text-[11px] font-bold ${
                    finding.status === 'closed'
                      ? 'bg-emerald-500 text-white'
                      : 'bg-surface-muted text-text-muted border border-border'
                  }`}
                >
                  3
                </span>
                <span className="text-text">Closed</span>
              </div>
            </div>
          </div>

          {/* Description */}
          <div>
            <h4 className="text-xs font-bold text-text mb-2">Deskripsi & Observasi Temuan:</h4>
            <div className="p-4 rounded-xl bg-surface-muted border border-border text-sm text-text whitespace-pre-wrap leading-relaxed">
              {finding.description || '(Tidak ada deskripsi detail)'}
            </div>
          </div>

          {/* Response / Resolution Section */}
          <div className="pt-2">
            <h4 className="text-xs font-bold text-text mb-2 flex items-center justify-between">
              <span>Tanggapan & Rencana Tindak Lanjut PM:</span>
              {finding.responded_at && (
                <span className="text-[11px] font-normal text-text-muted">
                  Oleh: {finding.responded_by_name || 'PM'} ({new Date(finding.responded_at).toLocaleDateString('id-ID')})
                </span>
              )}
            </h4>

            {finding.response ? (
              <div className="p-4 rounded-xl bg-amber-50/60 border border-amber-200 text-sm text-text whitespace-pre-wrap leading-relaxed">
                {finding.response}
              </div>
            ) : (
              <div className="p-4 rounded-xl bg-surface-muted border border-dashed border-border text-xs text-text-muted italic">
                Belum ada tanggapan resmi dari Project Manager owner.
              </div>
            )}

            {/* PM Response Input (Only when open and PM) */}
            {finding.status === 'open' && isPM && (
              <form onSubmit={handleSendResponse} className="mt-4 space-y-3">
                <div>
                  <label className="block text-xs font-semibold text-text mb-1">
                    Beri Tanggapan / Rencana Perbaikan:
                  </label>
                  <textarea
                    rows={3}
                    required
                    placeholder="Tuliskan komitmen tindak lanjut, waktu penyelesaian, dan langkah mitigasi operasional..."
                    value={responseText}
                    onChange={(e) => setResponseText(e.target.value)}
                    className="w-full px-3 py-2 text-sm bg-surface border border-border rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary resize-y"
                  />
                </div>
                <button
                  type="submit"
                  disabled={isLoading || !responseText.trim()}
                  className="px-4 py-2 text-xs font-bold text-white bg-primary hover:bg-primary-hover disabled:opacity-50 rounded-xl transition-colors flex items-center gap-1.5"
                >
                  {isLoading && (
                    <div className="w-3 h-3 border-2 border-white/20 border-t-white rounded-full animate-spin" />
                  )}
                  Kirim Tanggapan PM
                </button>
              </form>
            )}
          </div>

          {/* Closure Details */}
          {finding.closed_at && (
            <div className="p-3 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-800 flex items-center gap-2">
              <svg className="w-4 h-4 text-emerald-600 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
              </svg>
              <span>
                Temuan telah diverifikasi & ditutup oleh <strong>{finding.closed_by_name || 'Auditor'}</strong> pada{' '}
                {new Date(finding.closed_at).toLocaleDateString('id-ID', {
                  day: 'numeric',
                  month: 'short',
                  year: 'numeric',
                })}
                .
              </span>
            </div>
          )}

          {/* Audit Verification Actions */}
          {isAudit && (
            <div className="pt-4 border-t border-border space-y-3">
              <h4 className="text-xs font-bold text-text">Aksi Verifikasi Auditor:</h4>

              {finding.status === 'open' && (
                <div className="p-3 bg-blue-50 border border-blue-200 text-blue-800 text-xs rounded-xl">
                  Temuan ini masih berstatus <strong>Open</strong>. Sesuai kebijakan perbankan, temuan baru dapat ditutup setelah Project Manager memberikan tanggapan resmi (status: <em>Responded</em>).
                </div>
              )}

              {finding.status === 'responded' && !showConfirmClose && (
                <div className="flex items-center gap-3">
                  <button
                    onClick={() => setShowConfirmClose(true)}
                    className="px-4 py-2 text-xs font-bold text-white bg-emerald-600 hover:bg-emerald-700 rounded-xl transition-colors shadow-sm flex items-center gap-1.5"
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                    </svg>
                    Tutup Temuan (Selesai)
                  </button>
                  <button
                    onClick={() => setShowReopenInput(!showReopenInput)}
                    className="px-4 py-2 text-xs font-semibold text-rose-700 bg-rose-50 hover:bg-rose-100 border border-rose-200 rounded-xl transition-colors"
                  >
                    Buka Kembali Temuan
                  </button>
                </div>
              )}

              {finding.status === 'closed' && !showReopenInput && (
                <button
                  onClick={() => setShowReopenInput(true)}
                  className="px-4 py-2 text-xs font-semibold text-amber-700 bg-amber-50 hover:bg-amber-100 border border-amber-200 rounded-xl transition-colors"
                >
                  Buka Kembali Temuan (Reopen)
                </button>
              )}

              {/* Confirmation Dialog for Closing Finding */}
              {showConfirmClose && (
                <div className="p-4 bg-emerald-50 border border-emerald-300 rounded-xl space-y-3 animate-in fade-in">
                  <div className="text-xs text-emerald-900 font-medium">
                    Apakah Anda yakin telah memverifikasi bukti perbaikan dan menutup temuan ini? Aksi ini akan dicatat dalam Audit Trail.
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={handleCloseFinding}
                      disabled={isLoading}
                      className="px-3 py-1.5 text-xs font-bold text-white bg-emerald-600 hover:bg-emerald-700 rounded-lg transition-colors"
                    >
                      {isLoading ? 'Menutup...' : 'Ya, Tutup Temuan'}
                    </button>
                    <button
                      onClick={() => setShowConfirmClose(false)}
                      className="px-3 py-1.5 text-xs font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-lg transition-colors"
                    >
                      Batal
                    </button>
                  </div>
                </div>
              )}

              {/* Reopen Input */}
              {showReopenInput && (
                <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl space-y-3 animate-in fade-in">
                  <label className="block text-xs font-semibold text-rose-900">
                    Alasan Pembukaan Kembali (Audit Note):
                  </label>
                  <textarea
                    rows={2}
                    placeholder="Contoh: Bukti implementasi kontrol belum memadai..."
                    value={reopenComment}
                    onChange={(e) => setReopenComment(e.target.value)}
                    className="w-full px-3 py-2 text-xs bg-white border border-rose-300 rounded-lg focus:ring-2 focus:ring-rose-500/20 focus:border-rose-500 resize-y"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={handleReopenFinding}
                      disabled={isLoading}
                      className="px-3 py-1.5 text-xs font-bold text-white bg-rose-600 hover:bg-rose-700 rounded-lg transition-colors"
                    >
                      {isLoading ? 'Memproses...' : 'Konfirmasi Buka Kembali'}
                    </button>
                    <button
                      onClick={() => setShowReopenInput(false)}
                      className="px-3 py-1.5 text-xs font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50 rounded-lg transition-colors"
                    >
                      Batal
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
