import React from 'react'
import type { Risk, RiskLevel, RiskStatus, RiskCategory } from './types'

interface RiskTableProps {
  risks: Risk[]
  total: number
  page: number
  perPage: number
  isLoading: boolean
  userRole: string
  onPageChange: (newPage: number) => void
  onSelectRisk: (risk: Risk) => void
  onAssessRisk: (risk: Risk) => void
  onUpdateMitigation: (risk: Risk) => void
  onCloseRisk: (risk: Risk) => void
}

const CATEGORY_NAMES: Record<RiskCategory, string> = {
  operational: 'Operasional',
  schedule: 'Jadwal',
  technical: 'Teknis',
  compliance: 'Kepatuhan',
  financial: 'Finansial',
  security: 'Keamanan',
}

const STATUS_NAMES: Record<RiskStatus, string> = {
  identified: 'Teridentifikasi',
  assessed: 'Dinilai',
  mitigating: 'Mitigasi',
  accepted: 'Diterima',
  closed: 'Ditutup',
}

export const RiskTable: React.FC<RiskTableProps> = ({
  risks,
  total,
  page,
  perPage,
  isLoading,
  userRole,
  onPageChange,
  onSelectRisk,
  onAssessRisk,
  onUpdateMitigation,
  onCloseRisk,
}) => {
  const totalPages = Math.ceil(total / perPage) || 1

  const getLevelBadge = (level?: RiskLevel | null, score?: number | null) => {
    if (!level) {
      return (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400 border border-slate-200 dark:border-slate-700">
          Belum dinilai
        </span>
      )
    }

    const scoreText = score ? `(${score})` : ''

    switch (level) {
      case 'low':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-400 dark:border-emerald-800">
            Low {scoreText}
          </span>
        )
      case 'medium':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold bg-amber-50 text-amber-700 border border-amber-200 dark:bg-amber-950/40 dark:text-amber-400 dark:border-amber-800">
            Medium {scoreText}
          </span>
        )
      case 'high':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold bg-orange-50 text-orange-700 border border-orange-200 dark:bg-orange-950/40 dark:text-orange-400 dark:border-orange-800">
            High {scoreText}
          </span>
        )
      case 'critical':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold bg-rose-50 text-rose-700 border border-rose-200 dark:bg-rose-950/40 dark:text-rose-400 dark:border-rose-800 animate-pulse">
            Critical {scoreText}
          </span>
        )
    }
  }

  const getStatusBadge = (status: RiskStatus) => {
    switch (status) {
      case 'identified':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300">
            {STATUS_NAMES[status]}
          </span>
        )
      case 'assessed':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700 dark:bg-blue-950/50 dark:text-blue-300">
            {STATUS_NAMES[status]}
          </span>
        )
      case 'mitigating':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-purple-50 text-purple-700 dark:bg-purple-950/50 dark:text-purple-300">
            {STATUS_NAMES[status]}
          </span>
        )
      case 'accepted':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700 dark:bg-amber-950/50 dark:text-amber-300">
            {STATUS_NAMES[status]}
          </span>
        )
      case 'closed':
        return (
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300">
            {STATUS_NAMES[status]}
          </span>
        )
    }
  }

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 shadow-sm overflow-hidden">
      {/* Table Content */}
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse text-sm">
          <thead>
            <tr className="border-b border-slate-200 dark:border-slate-800 bg-slate-50/75 dark:bg-slate-800/40 text-slate-500 dark:text-slate-400 text-xs font-medium uppercase tracking-wider">
              <th className="py-3 px-4">Kode</th>
              <th className="py-3 px-4">Judul & Proyek</th>
              <th className="py-3 px-4">Kategori</th>
              <th className="py-3 px-4">L × I</th>
              <th className="py-3 px-4">Level</th>
              <th className="py-3 px-4">Status</th>
              <th className="py-3 px-4">Mitigasi</th>
              <th className="py-3 px-4 text-right">Aksi</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-800 text-slate-700 dark:text-slate-300">
            {isLoading ? (
              <tr>
                <td colSpan={8} className="py-12 text-center text-slate-400 dark:text-slate-500">
                  <div className="inline-block animate-spin w-6 h-6 border-2 border-indigo-600 border-t-transparent rounded-full mb-2"></div>
                  <p>Memuat daftar risiko...</p>
                </td>
              </tr>
            ) : risks.length === 0 ? (
              <tr>
                <td colSpan={8} className="py-12 text-center text-slate-400 dark:text-slate-500">
                  <p className="text-base font-medium text-slate-600 dark:text-slate-400">Tidak ada risiko yang ditemukan.</p>
                  <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">Coba sesuaikan filter pencarian atau ajukan risiko baru.</p>
                </td>
              </tr>
            ) : (
              risks.map((risk) => (
                <tr
                  key={risk.id}
                  className="hover:bg-slate-50/80 dark:hover:bg-slate-800/50 transition-colors cursor-pointer group"
                  onClick={() => onSelectRisk(risk)}
                >
                  {/* Code */}
                  <td className="py-3 px-4 whitespace-nowrap font-mono font-medium text-xs text-indigo-600 dark:text-indigo-400">
                    {risk.code}
                  </td>

                  {/* Title & Project */}
                  <td className="py-3 px-4">
                    <div className="font-semibold text-slate-900 dark:text-slate-100 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors line-clamp-1">
                      {risk.title}
                    </div>
                    <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                      <span>{risk.project_code || 'PRJ'} • {risk.project_name || 'Proyek'}</span>
                      {risk.task_title && (
                        <>
                          <span>•</span>
                          <span className="text-slate-600 dark:text-slate-300 truncate max-w-[180px]">
                            Task: {risk.task_title}
                          </span>
                        </>
                      )}
                    </div>
                  </td>

                  {/* Category */}
                  <td className="py-3 px-4 whitespace-nowrap">
                    <span className="text-xs text-slate-600 dark:text-slate-400 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                      {CATEGORY_NAMES[risk.category] || risk.category}
                    </span>
                  </td>

                  {/* Likelihood x Impact */}
                  <td className="py-3 px-4 whitespace-nowrap text-xs font-mono">
                    {risk.likelihood && risk.impact ? `${risk.likelihood} × ${risk.impact}` : '—'}
                  </td>

                  {/* Level */}
                  <td className="py-3 px-4 whitespace-nowrap">
                    {getLevelBadge(risk.level, risk.score)}
                  </td>

                  {/* Status */}
                  <td className="py-3 px-4 whitespace-nowrap">
                    {getStatusBadge(risk.status)}
                  </td>

                  {/* Mitigation Plan Preview */}
                  <td className="py-3 px-4 text-xs text-slate-500 dark:text-slate-400 max-w-[200px] truncate">
                    {risk.mitigation_plan ? (
                      <span className="text-slate-700 dark:text-slate-300">{risk.mitigation_plan}</span>
                    ) : (
                      <span className="text-slate-400 dark:text-slate-600 italic">Belum ada</span>
                    )}
                  </td>

                  {/* Actions */}
                  <td
                    className="py-3 px-4 text-right whitespace-nowrap"
                    onClick={(e) => e.stopPropagation()}
                  >
                    <div className="flex items-center justify-end gap-1.5">
                      {userRole === 'risk_management' && risk.status !== 'closed' && (
                        <button
                          type="button"
                          onClick={() => onAssessRisk(risk)}
                          className="px-2 py-1 text-xs font-medium rounded text-indigo-700 bg-indigo-50 hover:bg-indigo-100 dark:bg-indigo-950/40 dark:text-indigo-300 dark:hover:bg-indigo-900/50 transition-colors"
                          title="Nilai Likelihood & Impact"
                        >
                          Nilai
                        </button>
                      )}

                      {(userRole === 'risk_management' || userRole === 'project_manager') && risk.status !== 'closed' && (
                        <button
                          type="button"
                          onClick={() => onUpdateMitigation(risk)}
                          className="px-2 py-1 text-xs font-medium rounded text-slate-700 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:hover:bg-slate-700 transition-colors"
                          title="Rencana Mitigasi"
                        >
                          Mitigasi
                        </button>
                      )}

                      {userRole === 'risk_management' && risk.status !== 'closed' && (
                        <button
                          type="button"
                          onClick={() => onCloseRisk(risk)}
                          className="px-2 py-1 text-xs font-medium rounded text-emerald-700 bg-emerald-50 hover:bg-emerald-100 dark:bg-emerald-950/40 dark:text-emerald-300 dark:hover:bg-emerald-900/50 transition-colors"
                          title="Tutup Risiko"
                        >
                          Tutup
                        </button>
                      )}

                      <button
                        type="button"
                        onClick={() => onSelectRisk(risk)}
                        className="px-2 py-1 text-xs font-medium rounded text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200"
                      >
                        Detail
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination Footer */}
      <div className="flex flex-col sm:flex-row items-center justify-between px-4 py-3 border-t border-slate-200 dark:border-slate-800 bg-slate-50/50 dark:bg-slate-850 text-xs text-slate-500 dark:text-slate-400 gap-3">
        <div>
          Menampilkan <span className="font-semibold text-slate-700 dark:text-slate-200">{risks.length}</span> dari{' '}
          <span className="font-semibold text-slate-700 dark:text-slate-200">{total}</span> risiko
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            disabled={page <= 1 || isLoading}
            onClick={() => onPageChange(page - 1)}
            className="px-2.5 py-1.5 rounded border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/60 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Sebelumnya
          </button>
          <span>
            Halaman {page} dari {totalPages}
          </span>
          <button
            type="button"
            disabled={page >= totalPages || isLoading}
            onClick={() => onPageChange(page + 1)}
            className="px-2.5 py-1.5 rounded border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700/60 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
          >
            Berikutnya
          </button>
        </div>
      </div>
    </div>
  )
}
