import React, { useState, useEffect } from 'react'
import { assessRisk } from './api'
import type { Risk, RiskLevel } from './types'

interface AssessRiskModalProps {
  isOpen: boolean
  risk: Risk | null
  onClose: () => void
  onSuccess: (updatedRisk: Risk) => void
}

const LIKELIHOOD_OPTIONS = [
  { value: 1, label: '1 - Sangat Jarang', desc: 'Peluang terjadi < 10%' },
  { value: 2, label: '2 - Jarang', desc: 'Peluang terjadi 10% - 30%' },
  { value: 3, label: '3 - Mungkin', desc: 'Peluang terjadi 30% - 60%' },
  { value: 4, label: '4 - Sering', desc: 'Peluang terjadi 60% - 85%' },
  { value: 5, label: '5 - Hampir Pasti', desc: 'Peluang terjadi > 85%' },
]

const IMPACT_OPTIONS = [
  { value: 1, label: '1 - Ringan', desc: 'Dampak operasional/biaya minimal' },
  { value: 2, label: '2 - Kecil', desc: 'Penundaan minor, toleransi tercukupi' },
  { value: 3, label: '3 - Sedang', desc: 'Perlu eskalasi dan penyesuaian jadwal' },
  { value: 4, label: '4 - Berat', desc: 'Keterlambatan rilis atau kerugian finansial' },
  { value: 5, label: '5 - Katastropik', desc: 'Kegagalan kepatuhan regulasi / sistem inti' },
]

export const AssessRiskModal: React.FC<AssessRiskModalProps> = ({
  isOpen,
  risk,
  onClose,
  onSuccess,
}) => {
  const [likelihood, setLikelihood] = useState<number>(3)
  const [impact, setImpact] = useState<number>(3)
  const [mitigationPlan, setMitigationPlan] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (risk) {
      setLikelihood(risk.likelihood || 3)
      setImpact(risk.impact || 3)
      setMitigationPlan(risk.mitigation_plan || '')
      setError(null)
    }
  }, [risk])

  const calculatedScore = likelihood * impact

  let calculatedLevel: RiskLevel = 'low'
  if (calculatedScore <= 4) calculatedLevel = 'low'
  else if (calculatedScore <= 9) calculatedLevel = 'medium'
  else if (calculatedScore <= 15) calculatedLevel = 'high'
  else calculatedLevel = 'critical'

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!risk) return

    setIsLoading(true)
    setError(null)

    try {
      const updated = await assessRisk(risk.id, {
        likelihood,
        impact,
        mitigation_plan: mitigationPlan.trim() || undefined,
      })
      onSuccess(updated)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal menilai risiko.')
    } finally {
      setIsLoading(false)
    }
  }

  if (!isOpen || !risk) return null

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
      <div
        className="bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 w-full max-w-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <div>
            <span className="text-xs font-mono font-bold text-indigo-600 dark:text-indigo-400">
              {risk.code}
            </span>
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
              Penilaian Risiko (Likelihood & Impact)
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
          >
            ✕
          </button>
        </div>

        <form onSubmit={handleSubmit} className="p-6 space-y-5">
          {error && (
            <div className="p-3 text-xs rounded-lg bg-rose-50 text-rose-700 dark:bg-rose-950/50 dark:text-rose-300 border border-rose-200 dark:border-rose-900">
              {error}
            </div>
          )}

          {/* Risk Overview */}
          <div className="bg-slate-50 dark:bg-slate-800/50 p-3.5 rounded-lg border border-slate-200 dark:border-slate-700/60 space-y-1">
            <p className="text-xs text-slate-500 dark:text-slate-400">{risk.project_name || 'Proyek'}</p>
            <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{risk.title}</p>
            {risk.description && (
              <p className="text-xs text-slate-600 dark:text-slate-300 mt-1 line-clamp-2">
                {risk.description}
              </p>
            )}
          </div>

          {/* Likelihood Buttons (1-5) */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Likelihood / Kemungkinan Terjadi (1–5) <span className="text-rose-500">*</span>
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-5 gap-2">
              {LIKELIHOOD_OPTIONS.map((opt) => {
                const isSelected = likelihood === opt.value
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => setLikelihood(opt.value)}
                    className={`p-2 rounded-lg border text-left transition-all ${
                      isSelected
                        ? 'border-indigo-600 bg-indigo-50/70 text-indigo-900 dark:border-indigo-500 dark:bg-indigo-950/50 dark:text-indigo-200 ring-2 ring-indigo-500/20'
                        : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-750'
                    }`}
                  >
                    <div className="font-bold text-xs">{opt.label}</div>
                    <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-2">
                      {opt.desc}
                    </div>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Impact Buttons (1-5) */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Impact / Dampak Kerugian (1–5) <span className="text-rose-500">*</span>
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-5 gap-2">
              {IMPACT_OPTIONS.map((opt) => {
                const isSelected = impact === opt.value
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => setImpact(opt.value)}
                    className={`p-2 rounded-lg border text-left transition-all ${
                      isSelected
                        ? 'border-indigo-600 bg-indigo-50/70 text-indigo-900 dark:border-indigo-500 dark:bg-indigo-950/50 dark:text-indigo-200 ring-2 ring-indigo-500/20'
                        : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-750'
                    }`}
                  >
                    <div className="font-bold text-xs">{opt.label}</div>
                    <div className="text-[10px] text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-2">
                      {opt.desc}
                    </div>
                  </button>
                )
              })}
            </div>
          </div>

          {/* Live Preview Box */}
          <div className="p-4 rounded-lg bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 flex flex-col sm:flex-row items-center justify-between gap-3">
            <div>
              <span className="text-xs font-medium text-slate-500 dark:text-slate-400">Hasil Perhitungan Server:</span>
              <div className="text-sm font-semibold text-slate-900 dark:text-slate-100 mt-0.5">
                Likelihood ({likelihood}) × Impact ({impact}) = <span className="text-indigo-600 dark:text-indigo-400 font-mono text-base">{calculatedScore}</span>
              </div>
            </div>

            <div>
              <span
                className={`inline-flex items-center px-3 py-1 rounded-md text-xs font-bold uppercase tracking-wider ${
                  calculatedLevel === 'low'
                    ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/60 dark:text-emerald-300 border border-emerald-300 dark:border-emerald-800'
                    : calculatedLevel === 'medium'
                    ? 'bg-amber-100 text-amber-800 dark:bg-amber-950/60 dark:text-amber-300 border border-amber-300 dark:border-amber-800'
                    : calculatedLevel === 'high'
                    ? 'bg-orange-100 text-orange-800 dark:bg-orange-950/60 dark:text-orange-300 border border-orange-300 dark:border-orange-800'
                    : 'bg-rose-100 text-rose-800 dark:bg-rose-950/60 dark:text-rose-300 border border-rose-300 dark:border-rose-800 animate-pulse'
                }`}
              >
                Level: {calculatedLevel}
              </span>
            </div>
          </div>

          {/* Mitigation Plan (Optional but required for close if high/critical) */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Rencana Mitigasi (Mitigation Plan)
              {(calculatedLevel === 'high' || calculatedLevel === 'critical') && (
                <span className="text-amber-600 dark:text-amber-400 font-normal ml-1">
                  (Wajib diisi sebelum risiko dapat ditutup)
                </span>
              )}
            </label>
            <textarea
              rows={3}
              value={mitigationPlan}
              onChange={(e) => setMitigationPlan(e.target.value)}
              placeholder="Contoh: Implementasi failover cluster database dan prosedur backup berkala..."
              className="w-full text-sm px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              disabled={isLoading}
            />
          </div>

          <div className="pt-2 flex items-center justify-end gap-3 border-t border-slate-200 dark:border-slate-800">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="px-4 py-2 text-xs font-semibold rounded-lg border border-slate-300 dark:border-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800 transition-colors"
            >
              Batal
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-4 py-2 text-xs font-semibold rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white shadow-sm transition-colors disabled:opacity-50"
            >
              {isLoading ? 'Menyimpan...' : 'Simpan Penilaian'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
