import { useState } from 'react'
import { transferProjectOwner } from './api'
import type { Project } from './types'

interface TransferOwnerModalProps {
  isOpen: boolean
  project: Project | null
  onClose: () => void
  onSuccess: (updated: Project) => void
}

export default function TransferOwnerModal({ isOpen, project, onClose, onSuccess }: TransferOwnerModalProps) {
  const [newOwnerId, setNewOwnerId] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isOpen || !project) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newOwnerId.trim()) {
      setError('ID Project Manager baru wajib diisi.')
      return
    }

    try {
      setIsLoading(true)
      setError(null)
      const updated = await transferProjectOwner(project.id, newOwnerId.trim())
      onSuccess(updated)
      onClose()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal mengalihkan kepemilikan proyek.')
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-fade-in">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-2xl border border-slate-100 overflow-hidden">
        {/* Header */}
        <div className="px-6 py-5 bg-gradient-to-r from-purple-700 to-indigo-800 text-white flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight">Transfer Pemilik Proyek</h2>
            <p className="text-xs text-purple-200 mt-0.5">Hanya dapat dialihkan ke user ber-role Project Manager.</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-purple-200 hover:text-white hover:bg-white/10 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3.5 rounded-xl bg-red-50 border border-red-200 text-red-700 text-sm flex items-start gap-2.5">
              <svg className="w-5 h-5 text-red-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>{error}</span>
            </div>
          )}

          <div className="p-3 bg-purple-50 rounded-xl border border-purple-100 text-xs text-purple-900 leading-relaxed">
            <span className="font-semibold">Proyek:</span> {project.name} ({project.code})
            <br />
            <span className="font-semibold">Pemilik Saat Ini:</span> {project.owner_id}
          </div>

          <div>
            <label className="block text-sm font-semibold text-slate-700 mb-1">
              UUID Project Manager Baru <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              required
              placeholder="Contoh: 3fa85f64-5717-4562-b3fc-2c963f66afa6"
              value={newOwnerId}
              onChange={(e) => setNewOwnerId(e.target.value)}
              className="w-full px-3.5 py-2.5 text-sm rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-purple-500/20 focus:border-purple-500 transition-all font-mono text-slate-800 placeholder:font-sans placeholder:text-slate-400"
            />
            <p className="text-xs text-slate-500 mt-1">
              Server akan memvalidasi role user target dan mencatat aksi ini di audit log.
            </p>
          </div>

          {/* Footer Actions */}
          <div className="pt-4 border-t border-slate-100 flex items-center justify-end gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="px-4 py-2.5 text-sm font-medium text-slate-600 hover:text-slate-800 hover:bg-slate-100 rounded-xl transition-colors"
            >
              Batal
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-5 py-2.5 text-sm font-semibold text-white bg-purple-700 hover:bg-purple-800 rounded-xl shadow-sm hover:shadow transition-all disabled:opacity-60 flex items-center gap-2"
            >
              {isLoading ? 'Mengalihkan...' : 'Transfer Pemilik'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
