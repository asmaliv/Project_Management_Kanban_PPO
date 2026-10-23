import { useState } from 'react'
import { createBoardColumn, updateBoardColumn } from './api'
import type { BoardColumn } from './types'

interface ColumnModalProps {
  isOpen: boolean
  projectId: string
  columnToEdit?: BoardColumn | null
  nextPosition: number
  onClose: () => void
  onSuccess: () => void
}

export default function ColumnModal({
  isOpen,
  projectId,
  columnToEdit,
  nextPosition,
  onClose,
  onSuccess,
}: ColumnModalProps) {
  if (!isOpen) return null

  return (
    <ColumnModalForm
      key={columnToEdit?.id || 'new'}
      projectId={projectId}
      columnToEdit={columnToEdit}
      nextPosition={nextPosition}
      onClose={onClose}
      onSuccess={onSuccess}
    />
  )
}

function ColumnModalForm({
  projectId,
  columnToEdit,
  nextPosition,
  onClose,
  onSuccess,
}: {
  projectId: string
  columnToEdit?: BoardColumn | null
  nextPosition: number
  onClose: () => void
  onSuccess: () => void
}) {
  const isEditing = Boolean(columnToEdit)
  const [name, setName] = useState(columnToEdit ? columnToEdit.name : '')
  const [position, setPosition] = useState<number>(
    columnToEdit ? columnToEdit.position : nextPosition,
  )
  const [wipLimit, setWipLimit] = useState<string>(
    columnToEdit && columnToEdit.wip_limit !== null ? String(columnToEdit.wip_limit) : '',
  )
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) {
      setError('Nama kolom wajib diisi.')
      return
    }

    const parsedWip = wipLimit.trim() !== '' ? parseInt(wipLimit.trim(), 10) : null
    if (parsedWip !== null && (isNaN(parsedWip) || parsedWip < 1)) {
      setError('WIP limit harus berupa angka positif.')
      return
    }

    try {
      setIsLoading(true)
      setError(null)

      if (isEditing && columnToEdit) {
        await updateBoardColumn(columnToEdit.id, {
          name: name.trim(),
          position,
          wip_limit: parsedWip,
        })
      } else {
        await createBoardColumn(projectId, {
          name: name.trim(),
          position,
          wip_limit: parsedWip,
        })
      }

      onSuccess()
      onClose()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal menyimpan kolom papan.')
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm animate-fade-in">
      <div className="w-full max-w-md bg-white rounded-2xl shadow-2xl border border-slate-100 overflow-hidden">
        {/* Header */}
        <div className="px-6 py-5 bg-gradient-to-r from-blue-600 to-indigo-700 text-white flex items-center justify-between">
          <div>
            <h2 className="text-xl font-bold tracking-tight">
              {isEditing ? 'Edit Kolom Papan' : 'Tambah Kolom Baru'}
            </h2>
            <p className="text-xs text-blue-100 mt-0.5">
              {isEditing ? `Mengubah kolom "${columnToEdit?.name}"` : 'Kolom alur kerja papan Kanban proyek.'}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-blue-100 hover:text-white hover:bg-white/10 transition-colors"
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

          <div>
            <label className="block text-sm font-semibold text-slate-700 mb-1">
              Nama Kolom <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              required
              placeholder="Contoh: QA Testing / UAT"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3.5 py-2.5 text-sm rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-semibold text-slate-700 mb-1">
                Urutan Posisi <span className="text-red-500">*</span>
              </label>
              <input
                type="number"
                required
                min={1}
                value={position}
                onChange={(e) => setPosition(parseInt(e.target.value, 10) || 1)}
                className="w-full px-3.5 py-2.5 text-sm rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800"
              />
            </div>

            <div>
              <label className="block text-sm font-semibold text-slate-700 mb-1">WIP Limit</label>
              <input
                type="number"
                min={1}
                placeholder="Tak terbatas"
                value={wipLimit}
                onChange={(e) => setWipLimit(e.target.value)}
                className="w-full px-3.5 py-2.5 text-sm rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all text-slate-800 placeholder:text-slate-400"
              />
            </div>
          </div>

          {columnToEdit?.is_done_column && (
            <div className="p-3 bg-amber-50 rounded-xl border border-amber-200 text-xs text-amber-800 flex items-center gap-2">
              <svg className="w-4 h-4 text-amber-600 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>Ini adalah kolom status akhir (Done) yang dilindungi oleh sistem maker-checker.</span>
            </div>
          )}

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
              className="px-5 py-2.5 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-xl shadow-sm hover:shadow transition-all disabled:opacity-60 flex items-center gap-2"
            >
              {isLoading ? 'Menyimpan...' : isEditing ? 'Simpan Perubahan' : 'Tambah Kolom'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
