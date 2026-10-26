import React from 'react'
import type { RiskSummary, HeatmapCell, RiskLevel } from './types'

interface RiskHeatmapProps {
  summary: RiskSummary | null
  isLoading: boolean
  selectedCell: { likelihood: number; impact: number } | null
  onSelectCell: (cell: { likelihood: number; impact: number } | null) => void
}

const LIKELIHOOD_LABELS = [
  { value: 5, label: '5 - Hampir Pasti' },
  { value: 4, label: '4 - Sering' },
  { value: 3, label: '3 - Mungkin' },
  { value: 2, label: '2 - Jarang' },
  { value: 1, label: '1 - Sangat Jarang' },
]

const IMPACT_LABELS = [
  { value: 1, label: '1 - Ringan' },
  { value: 2, label: '2 - Kecil' },
  { value: 3, label: '3 - Sedang' },
  { value: 4, label: '4 - Berat' },
  { value: 5, label: '5 - Katastropik' },
]

export const RiskHeatmap: React.FC<RiskHeatmapProps> = ({
  summary,
  isLoading,
  selectedCell,
  onSelectCell,
}) => {
  if (isLoading || !summary) {
    return (
      <div className="bg-white dark:bg-slate-900 rounded-xl p-8 border border-slate-200 dark:border-slate-800 shadow-sm animate-pulse">
        <div className="h-6 bg-slate-200 dark:bg-slate-700 rounded w-1/4 mb-4"></div>
        <div className="h-64 bg-slate-100 dark:bg-slate-800 rounded-lg"></div>
      </div>
    )
  }

  // Create lookup for heatmap cells
  const cellMap = new Map<string, HeatmapCell>()
  summary.heatmap.forEach((cell) => {
    cellMap.set(`${cell.likelihood}_${cell.impact}`, cell)
  })

  const getCellBgClass = (level: RiskLevel, _count: number, isSelected: boolean) => {
    const activeRing = isSelected ? 'ring-2 ring-indigo-600 dark:ring-indigo-400 z-10 scale-[1.03] shadow-md' : ''
    
    switch (level) {
      case 'low':
        return `${activeRing} bg-emerald-50 dark:bg-emerald-950/40 text-emerald-800 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800/60 hover:bg-emerald-100 dark:hover:bg-emerald-900/50`
      case 'medium':
        return `${activeRing} bg-amber-50 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 border-amber-200 dark:border-amber-800/60 hover:bg-amber-100 dark:hover:bg-amber-900/50`
      case 'high':
        return `${activeRing} bg-orange-50 dark:bg-orange-950/40 text-orange-800 dark:text-orange-300 border-orange-200 dark:border-orange-800/60 hover:bg-orange-100 dark:hover:bg-orange-900/50`
      case 'critical':
        return `${activeRing} bg-rose-50 dark:bg-rose-950/40 text-rose-800 dark:text-rose-300 border-rose-200 dark:border-rose-800/60 hover:bg-rose-100 dark:hover:bg-rose-900/50`
    }
  }

  return (
    <div className="bg-white dark:bg-slate-900 rounded-xl p-6 border border-slate-200 dark:border-slate-800 shadow-sm space-y-6">
      {/* Metric Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-slate-50 dark:bg-slate-800/50 p-4 rounded-lg border border-slate-200 dark:border-slate-700/60">
          <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">Total Risiko</p>
          <p className="text-2xl font-bold text-slate-900 dark:text-slate-100 mt-1">{summary.total_risks}</p>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">{summary.open_risks} masih terbuka</p>
        </div>

        <div className="bg-rose-50/70 dark:bg-rose-950/20 p-4 rounded-lg border border-rose-200 dark:border-rose-900/40">
          <p className="text-xs font-medium text-rose-600 dark:text-rose-400 uppercase tracking-wider">High & Critical</p>
          <p className="text-2xl font-bold text-rose-700 dark:text-rose-300 mt-1">{summary.high_critical_risks}</p>
          <p className="text-xs text-rose-600/80 dark:text-rose-400/80 mt-1">Perlu mitigasi mendesak</p>
        </div>

        <div className="bg-amber-50/70 dark:bg-amber-950/20 p-4 rounded-lg border border-amber-200 dark:border-amber-900/40">
          <p className="text-xs font-medium text-amber-600 dark:text-amber-400 uppercase tracking-wider">Tanpa Mitigasi</p>
          <p className="text-2xl font-bold text-amber-700 dark:text-amber-300 mt-1">{summary.unmitigated_risks}</p>
          <p className="text-xs text-amber-600/80 dark:text-amber-400/80 mt-1">Belum ada rencana tindak</p>
        </div>

        <div className="bg-emerald-50/70 dark:bg-emerald-950/20 p-4 rounded-lg border border-emerald-200 dark:border-emerald-900/40">
          <p className="text-xs font-medium text-emerald-600 dark:text-emerald-400 uppercase tracking-wider">Low & Medium</p>
          <p className="text-2xl font-bold text-emerald-700 dark:text-emerald-300 mt-1">
            {(summary.by_level['low'] || 0) + (summary.by_level['medium'] || 0)}
          </p>
          <p className="text-xs text-emerald-600/80 dark:text-emerald-400/80 mt-1">Terkendali</p>
        </div>
      </div>

      {/* Heatmap Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pt-2">
        <div>
          <h3 className="text-base font-semibold text-slate-900 dark:text-slate-100">
            Heatmap Matriks Risiko (5×5)
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            Klik sel pada matriks untuk menyaring daftar risiko berdasarkan Likelihood × Impact.
          </p>
        </div>
        {selectedCell && (
          <button
            onClick={() => onSelectCell(null)}
            className="self-start sm:self-auto text-xs font-medium text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 bg-indigo-50 dark:bg-indigo-950/40 px-2.5 py-1 rounded-md border border-indigo-200 dark:border-indigo-800"
          >
            Reset Filter Sel ({selectedCell.likelihood} × {selectedCell.impact})
          </button>
        )}
      </div>

      {/* 5x5 Grid */}
      <div className="overflow-x-auto pb-2">
        <div className="min-w-[620px]">
          <div className="grid grid-cols-[130px_repeat(5,1fr)] gap-2">
            {/* Header Column Labels */}
            <div className="h-10 flex items-center justify-center font-semibold text-xs text-slate-500 dark:text-slate-400">
              Likelihood \ Impact
            </div>
            {IMPACT_LABELS.map((imp) => (
              <div
                key={imp.value}
                className="h-10 flex items-center justify-center text-center font-medium text-xs text-slate-700 dark:text-slate-300 px-1 bg-slate-50 dark:bg-slate-800/40 rounded border border-slate-200/80 dark:border-slate-800"
              >
                {imp.label}
              </div>
            ))}

            {/* Matrix Rows (5 down to 1) */}
            {LIKELIHOOD_LABELS.map((lh) => (
              <React.Fragment key={lh.value}>
                {/* Row Label */}
                <div className="flex items-center text-xs font-medium text-slate-700 dark:text-slate-300 px-2 bg-slate-50 dark:bg-slate-800/40 rounded border border-slate-200/80 dark:border-slate-800">
                  {lh.label}
                </div>

                {/* 5 Cells in this row */}
                {IMPACT_LABELS.map((imp) => {
                  const cell = cellMap.get(`${lh.value}_${imp.value}`)
                  const isSelected =
                    selectedCell?.likelihood === lh.value && selectedCell?.impact === imp.value
                  const count = cell?.count || 0
                  const score = lh.value * imp.value
                  const level = cell?.level || 'low'

                  return (
                    <button
                      key={imp.value}
                      type="button"
                      onClick={() => onSelectCell(isSelected ? null : { likelihood: lh.value, impact: imp.value })}
                      className={`h-20 p-2.5 rounded-lg border transition-all flex flex-col justify-between text-left cursor-pointer focus:outline-none focus:ring-2 focus:ring-offset-1 ${getCellBgClass(
                        level,
                        count,
                        isSelected
                      )}`}
                      title={`Likelihood: ${lh.label}, Impact: ${imp.label} | Skor: ${score} (${level.toUpperCase()}) | ${count} risiko`}
                      aria-label={`Likelihood ${lh.value}, Impact ${imp.value}, Skor ${score}, ${count} risiko`}
                    >
                      <div className="flex items-center justify-between w-full">
                        <span className="text-[10px] uppercase font-bold tracking-wider opacity-75">
                          {level}
                        </span>
                        <span className="text-[11px] font-mono opacity-80">
                          {score}
                        </span>
                      </div>
                      <div className="flex items-baseline justify-between mt-auto">
                        <span className="text-xl font-bold leading-none">
                          {count}
                        </span>
                        <span className="text-[10px] opacity-75">risiko</span>
                      </div>
                    </button>
                  )
                })}
              </React.Fragment>
            ))}
          </div>
        </div>
      </div>

      {/* Legend */}
      <div className="flex flex-wrap items-center justify-between pt-2 border-t border-slate-200 dark:border-slate-800 text-xs text-slate-600 dark:text-slate-400 gap-3">
        <span className="font-medium text-slate-700 dark:text-slate-300">Level Risiko:</span>
        <div className="flex flex-wrap gap-4">
          <div className="flex items-center gap-1.5">
            <span className="w-3.5 h-3.5 rounded bg-emerald-200 dark:bg-emerald-900 border border-emerald-400 dark:border-emerald-700"></span>
            <span>Low (1–4)</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3.5 h-3.5 rounded bg-amber-200 dark:bg-amber-900 border border-amber-400 dark:border-amber-700"></span>
            <span>Medium (5–9)</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3.5 h-3.5 rounded bg-orange-200 dark:bg-orange-900 border border-orange-400 dark:border-orange-700"></span>
            <span>High (10–15)</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="w-3.5 h-3.5 rounded bg-rose-200 dark:bg-rose-900 border border-rose-400 dark:border-rose-700"></span>
            <span>Critical (16–25)</span>
          </div>
        </div>
      </div>
    </div>
  )
}
