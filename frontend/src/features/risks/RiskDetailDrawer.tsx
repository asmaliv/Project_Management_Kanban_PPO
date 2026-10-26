import React from 'react'
import type { Risk, RiskLevel, RiskStatus, RiskCategory } from './types'

interface RiskDetailDrawerProps {
  isOpen: boolean
  risk: Risk | null
  userRole: string
  onClose: () => void
  onAssess: (risk: Risk) => void
  onMitigate: (risk: Risk) => void
  onCloseRisk: (risk: Risk) => void
}

const CATEGORY_NAMES: Record<RiskCategory, string> = {
  operational: 'Operasional',
  schedule: 'Jadwal (Schedule)',
  technical: 'Teknis',
  compliance: 'Kepatuhan (Compliance)',
  financial: 'Finansial',
  security: 'Keamanan (Security)',
}

const STATUS_NAMES: Record<RiskStatus, string> = {
  identified: 'Teridentifikasi',
  assessed: 'Dinilai',
  mitigating: 'Dalam Mitigasi',
  accepted: 'Diterima',
  closed: 'Ditutup',
}

export const RiskDetailDrawer: React.FC<RiskDetailDrawerProps> = ({
  isOpen,
  risk,
  userRole,
  onClose,
  onAssess,
  onMitigate,
  onCloseRisk,
}) => {
  if (!isOpen || !risk) return null

  const getLevelBadge = (level?: RiskLevel | null, score?: number | null) => {
    if (!level) {
      return (
        <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
          Belum dinilai
        </span>
      )
    }

    const scoreText = score ? `(${score})` : ''

    switch (level) {
      case 'low':
        return (
          <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300 border border-emerald-300 dark:border-emerald-800">
            Low {scoreText}
          </span>
        )
      case 'medium':
        return (
          <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300 border border-amber-300 dark:border-amber-800">
            Medium {scoreText}
          </span>
        )
      case 'high':
        return (
          <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider bg-orange-100 text-orange-800 dark:bg-orange-950/60 dark:text-orange-300 border border-orange-300 dark:border-orange-800">
            High {scoreText}
          </span>
        )
      case 'critical':
        return (
          <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-bold uppercase tracking-wider bg-rose-100 text-rose-800 dark:bg-rose-950/60 dark:text-rose-300 border border-rose-300 dark:border-rose-800 animate-pulse">
            Critical {scoreText}
          </span>
        )
    }
  }

  const isClosed = risk.status === 'closed'
  const isHighOrCritical = risk.level === 'high' || risk.level === 'critical'
  const canClose = userRole === 'risk_management' && !isClosed

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-slate-900/40 backdrop-blur-xs flex justify-end">
      <div
        className="w-full max-w-lg bg-white dark:bg-slate-900 h-full shadow-2xl border-l border-slate-200 dark:border-slate-800 flex flex-col animate-in slide-in-from-right duration-200"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-start justify-between gap-4 bg-slate-50/50 dark:bg-slate-850">
          <div>
            <div className="flex items-center gap-2">
              <span className="font-mono text-xs font-bold px-2 py-0.5 rounded bg-indigo-50 text-indigo-700 dark:bg-indigo-950/60 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800">
                {risk.code}
              </span>
              <span className="text-xs text-slate-500 dark:text-slate-400">
                {risk.project_code} • {risk.project_name}
              </span>
            </div>
            <h2 className="text-lg font-bold text-slate-900 dark:text-slate-100 mt-1.5 leading-snug">
              {risk.title}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          >
            ✕
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          {/* Key Metrics */}
          <div className="grid grid-cols-2 gap-3 p-4 bg-slate-50 dark:bg-slate-800/40 rounded-xl border border-slate-200/80 dark:border-slate-800">
            <div>
              <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Level Risiko</p>
              <div className="mt-1">{getLevelBadge(risk.level, risk.score)}</div>
            </div>
            <div>
              <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Status</p>
              <div className="mt-1 font-semibold text-xs text-slate-800 dark:text-slate-200">
                {STATUS_NAMES[risk.status] || risk.status}
              </div>
            </div>
            <div>
              <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Kategori</p>
              <div className="mt-1 text-xs text-slate-700 dark:text-slate-300 font-medium">
                {CATEGORY_NAMES[risk.category] || risk.category}
              </div>
            </div>
            <div>
              <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Likelihood × Impact</p>
              <div className="mt-1 text-xs font-mono font-medium text-slate-700 dark:text-slate-300">
                {risk.likelihood && risk.impact ? `${risk.likelihood} × ${risk.impact} = ${risk.score}` : 'Belum dihitung'}
              </div>
            </div>
          </div>

          {/* Description */}
          <div>
            <h4 className="text-xs font-semibold text-slate-900 dark:text-slate-100 uppercase tracking-wider mb-2">
              Deskripsi Risiko
            </h4>
            <div className="text-sm text-slate-700 dark:text-slate-300 bg-slate-50/50 dark:bg-slate-800/30 p-3 rounded-lg border border-slate-200/60 dark:border-slate-800">
              {risk.description || <span className="text-slate-400 italic">Tidak ada deskripsi detail tambahan.</span>}
            </div>
          </div>

          {/* Mitigation Plan */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-xs font-semibold text-slate-900 dark:text-slate-100 uppercase tracking-wider">
                Rencana Mitigasi
              </h4>
              {(userRole === 'project_manager' || userRole === 'risk_management') && !isClosed && (
                <button
                  type="button"
                  onClick={() => onMitigate(risk)}
                  className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
                >
                  {risk.mitigation_plan ? 'Ubah Mitigasi' : '+ Tambah Mitigasi'}
                </button>
              )}
            </div>
            <div className="text-sm text-slate-700 dark:text-slate-300 bg-slate-50/50 dark:bg-slate-800/30 p-3 rounded-lg border border-slate-200/60 dark:border-slate-800">
              {risk.mitigation_plan ? (
                <p className="whitespace-pre-wrap">{risk.mitigation_plan}</p>
              ) : (
                <p className="text-slate-400 italic">
                  Belum ada rencana mitigasi terdaftar.
                  {isHighOrCritical && (
                    <span className="block text-amber-600 dark:text-amber-400 not-italic font-medium text-xs mt-1">
                      Wajib diisi sebelum risiko ini dapat ditutup.
                    </span>
                  )}
                </p>
              )}
            </div>
          </div>

          {/* Linked Task */}
          {risk.task_title && (
            <div>
              <h4 className="text-xs font-semibold text-slate-900 dark:text-slate-100 uppercase tracking-wider mb-2">
                Task Terkait
              </h4>
              <div className="flex items-center gap-2 p-3 bg-indigo-50/50 dark:bg-indigo-950/20 rounded-lg border border-indigo-200/60 dark:border-indigo-900/40 text-xs">
                <span className="w-2 h-2 rounded-full bg-indigo-500"></span>
                <span className="font-semibold text-slate-900 dark:text-slate-100">{risk.task_title}</span>
              </div>
            </div>
          )}

          {/* Meta Information */}
          <div className="pt-3 border-t border-slate-200 dark:border-slate-800 text-xs text-slate-500 dark:text-slate-400 space-y-2">
            <div className="flex justify-between">
              <span>Diajukan oleh:</span>
              <span className="font-medium text-slate-700 dark:text-slate-300">{risk.raised_by_name || '—'}</span>
            </div>
            <div className="flex justify-between">
              <span>Waktu Pengajuan:</span>
              <span>{new Date(risk.created_at).toLocaleString('id-ID')}</span>
            </div>
            <div className="flex justify-between">
              <span>Terakhir Diperbarui:</span>
              <span>{new Date(risk.updated_at).toLocaleString('id-ID')}</span>
            </div>
            {risk.closed_at && (
              <div className="flex justify-between text-emerald-600 dark:text-emerald-400 font-medium">
                <span>Ditutup pada:</span>
                <span>{new Date(risk.closed_at).toLocaleString('id-ID')}</span>
              </div>
            )}
          </div>
        </div>

        {/* Footer Actions */}
        <div className="p-4 border-t border-slate-200 dark:border-slate-800 bg-slate-50/75 dark:bg-slate-850 flex items-center justify-between gap-3">
          <div className="flex gap-2">
            {userRole === 'risk_management' && !isClosed && (
              <button
                type="button"
                onClick={() => onAssess(risk)}
                className="px-3 py-2 text-xs font-semibold rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm transition-colors"
              >
                {risk.score ? 'Nilai Ulang' : 'Beri Penilaian'}
              </button>
            )}

            {(userRole === 'project_manager' || userRole === 'risk_management') && !isClosed && (
              <button
                type="button"
                onClick={() => onMitigate(risk)}
                className="px-3 py-2 text-xs font-semibold rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-750 transition-colors"
              >
                Rencana Mitigasi
              </button>
            )}
          </div>

          {canClose && (
            <button
              type="button"
              onClick={() => onCloseRisk(risk)}
              className="px-3 py-2 text-xs font-semibold rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white shadow-sm transition-colors"
            >
              Tutup Risiko
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
