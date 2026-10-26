import React, { useState, useEffect } from 'react'
import { updateRisk } from './api'
import type { Risk } from './types'

interface MitigationModalProps {
  isOpen: boolean
  risk: Risk | null
  onClose: () => void
  onSuccess: (updatedRisk: Risk) => void
}

export const MitigationModal: React.FC<MitigationModalProps> = ({
  isOpen,
  risk,
  onClose,
  onSuccess,
}) => {
  const [mitigationPlan, setMitigationPlan] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (risk) {
      setMitigationPlan(risk.mitigation_plan || '')
      setError(null)
    }
  }, [risk])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!risk) return

    if (!mitigationPlan.trim()) {
      setError('Rencana mitigasi tidak boleh kosong.')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const updated = await updateRisk(risk.id, {
        mitigation_plan: mitigationPlan.trim(),
      })
      onSuccess(updated)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal memperbarui rencana mitigasi.')
    } finally {
      setIsLoading(false)
    }
  }

  if (!isOpen || !risk) return null

  const isHighOrCritical = risk.level === 'high' || risk.level === 'critical'

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
      <div
        className="bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 w-full max-w-lg overflow-hidden animate-in fade-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <div>
            <span className="text-xs font-mono font-bold text-indigo-600 dark:text-indigo-400">
              {risk.code}
            </span>
            <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
              Perbarui Rencana Mitigasi
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

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3 text-xs rounded-lg bg-rose-50 text-rose-700 dark:bg-rose-950/50 dark:text-rose-300 border border-rose-200 dark:border-rose-900">
              {error}
            </div>
          )}

          {isHighOrCritical && (
            <div className="p-3 text-xs rounded-lg bg-amber-50 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300 border border-amber-200 dark:border-amber-900">
              <span className="font-semibold">Perhatian:</span> Risiko ini berlevel{' '}
              <span className="uppercase font-bold">{risk.level}</span>. Sistem perbankan mewajibkan rencana mitigasi lengkap sebelum risiko ini dapat ditutup oleh Risk Management.
            </div>
          )}

          <div className="bg-slate-50 dark:bg-slate-800/50 p-3 rounded-lg border border-slate-200 dark:border-slate-700/60">
            <p className="text-xs text-slate-500 dark:text-slate-400">{risk.project_name || 'Proyek'}</p>
            <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{risk.title}</p>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Rencana Mitigasi & Tindakan Pencegahan <span className="text-rose-500">*</span>
            </label>
            <textarea
              rows={4}
              value={mitigationPlan}
              onChange={(e) => setMitigationPlan(e.target.value)}
              placeholder="Rincikan langkah mitigasi, penanggung jawab, atau prosedur alternatif..."
              className="w-full text-sm px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              required
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
              {isLoading ? 'Menyimpan...' : 'Simpan Mitigasi'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
