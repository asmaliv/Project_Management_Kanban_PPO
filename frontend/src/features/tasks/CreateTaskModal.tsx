import React, { useState } from 'react'
import type { TaskPriority, CreateTaskPayload } from './types'
import type { BoardColumn, ProjectMember } from '../projects/types'

interface CreateTaskModalProps {
  isOpen: boolean
  onClose: () => void
  columns: BoardColumn[]
  members: ProjectMember[]
  ownerName: string
  ownerId: string
  defaultColumnId?: string
  onSubmit: (payload: CreateTaskPayload) => Promise<void>
}

export default function CreateTaskModal({
  isOpen,
  onClose,
  columns,
  members,
  ownerName,
  ownerId,
  defaultColumnId,
  onSubmit,
}: CreateTaskModalProps) {
  const [columnId, setColumnId] = useState(defaultColumnId || columns[0]?.id || '')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<TaskPriority>('medium')
  const [assigneeId, setAssigneeId] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [progressPercent, setProgressPercent] = useState(0)

  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isOpen) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) {
      setError('Judul task wajib diisi.')
      return
    }
    if (!columnId) {
      setError('Kolom tujuan wajib dipilih.')
      return
    }

    try {
      setIsSubmitting(true)
      setError(null)
      await onSubmit({
        column_id: columnId,
        title: title.trim(),
        description: description.trim(),
        priority,
        assignee_id: assigneeId || null,
        due_date: dueDate || null,
        progress_percent: progressPercent,
      })
      onClose()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal membuat task.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/40 backdrop-blur-xs flex items-center justify-center p-4">
      <div className="bg-white rounded-3xl max-w-lg w-full p-6 sm:p-8 shadow-2xl border border-slate-100 animate-fade-in space-y-6">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <h3 className="text-lg font-bold text-slate-900">Buat Task Baru</h3>
            <p className="text-xs text-slate-500">Tambahkan pekerjaan ke dalam papan Kanban</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 rounded-lg p-1.5 transition-colors"
          >
            &times;
          </button>
        </div>

        {error && (
          <div className="p-3.5 rounded-xl bg-red-50 border border-red-200 text-red-700 text-xs flex items-start gap-2">
            <svg className="w-4 h-4 text-red-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Kolom Tujuan */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Kolom Tujuan <span className="text-red-500">*</span>
            </label>
            <select
              value={columnId}
              onChange={(e) => setColumnId(e.target.value)}
              className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
            >
              {columns.map((col) => (
                <option key={col.id} value={col.id}>
                  {col.name} {col.wip_limit ? `(WIP: ${col.wip_limit})` : ''} {col.is_done_column ? '— Done' : ''}
                </option>
              ))}
            </select>
          </div>

          {/* Judul Task */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Judul Task <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Contoh: Implementasi modul validasi KYC"
              className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              autoFocus
            />
          </div>

          {/* Deskripsi */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 mb-1">
              Deskripsi Pekerjaan
            </label>
            <textarea
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Rincian persyaratan, kriteria penyelesaian, atau catatan teknis..."
              className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
            />
          </div>

          {/* Prioritas & Assignee */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Prioritas</label>
              <select
                value={priority}
                onChange={(e) => setPriority(e.target.value as TaskPriority)}
                className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
              >
                <option value="low">Rendah (Low)</option>
                <option value="medium">Sedang (Medium)</option>
                <option value="high">Tinggi (High)</option>
                <option value="critical">Kritis (Critical)</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Penugasan (Assignee)</label>
              <select
                value={assigneeId}
                onChange={(e) => setAssigneeId(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white"
              >
                <option value="">Belum Ditugaskan</option>
                <option value={ownerId}>{ownerName} (Project Manager)</option>
                {members.map((mem) => (
                  <option key={mem.id} value={mem.id}>
                    {mem.full_name} ({mem.role})
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Tenggat Waktu & Progress */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Tenggat Waktu</label>
              <input
                type="date"
                value={dueDate}
                onChange={(e) => setDueDate(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">
                Progress Awal ({progressPercent}%)
              </label>
              <input
                type="range"
                min="0"
                max="100"
                step="5"
                value={progressPercent}
                onChange={(e) => setProgressPercent(Number(e.target.value))}
                className="w-full mt-2 accent-blue-600 cursor-pointer"
              />
            </div>
          </div>

          <div className="flex items-center justify-end gap-3 pt-4 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-4 py-2 text-sm font-semibold text-slate-600 hover:text-slate-900 rounded-xl hover:bg-slate-50 transition-colors"
            >
              Batal
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="px-5 py-2 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 rounded-xl transition-all shadow-xs"
            >
              {isSubmitting ? 'Menyimpan...' : 'Buat Task'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
