import React, { useState, useEffect, useCallback } from 'react'
import { createRisk } from './api'
import { fetchProjects } from '../projects/api'
import { fetchProjectTasks } from '../tasks/api'
import type { RiskCategory, Risk } from './types'
import type { Project } from '../projects/types'
import type { Task } from '../tasks/types'

interface CreateRiskModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: (newRisk: Risk) => void
  initialProjectId?: string
  initialTaskId?: string
}

const CATEGORIES: { value: RiskCategory; label: string }[] = [
  { value: 'operational', label: 'Operasional' },
  { value: 'schedule', label: 'Jadwal (Schedule)' },
  { value: 'technical', label: 'Teknis' },
  { value: 'compliance', label: 'Kepatuhan (Compliance)' },
  { value: 'financial', label: 'Finansial' },
  { value: 'security', label: 'Keamanan (Security)' },
]

export const CreateRiskModal: React.FC<CreateRiskModalProps> = ({
  isOpen,
  onClose,
  onSuccess,
  initialProjectId,
  initialTaskId,
}) => {
  const [projects, setProjects] = useState<Project[]>([])
  const [tasks, setTasks] = useState<Task[]>([])
  const [selectedProjectId, setSelectedProjectId] = useState<string>(initialProjectId || '')
  const [selectedTaskId, setSelectedTaskId] = useState<string>(initialTaskId || '')
  const [title, setTitle] = useState('')
  const [category, setCategory] = useState<RiskCategory>('operational')
  const [description, setDescription] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadTasks = useCallback(async (projId: string) => {
    try {
      const taskList = await fetchProjectTasks(projId)
      setTasks(taskList)
    } catch {
      setTasks([])
    }
  }, [])

  const loadProjects = useCallback(async () => {
    try {
      const data = await fetchProjects()
      setProjects(data.filter((p) => p.status !== 'archived'))
      if (!selectedProjectId && data.length > 0) {
        setSelectedProjectId(data[0].id)
        void loadTasks(data[0].id)
      }
    } catch {
      // Ignore
    }
  }, [loadTasks, selectedProjectId])

  useEffect(() => {
    if (isOpen) {
      void loadProjects()
      if (initialProjectId) {
        setSelectedProjectId(initialProjectId)
        void loadTasks(initialProjectId)
      }
      if (initialTaskId) {
        setSelectedTaskId(initialTaskId)
      }
    } else {
      // Reset
      setTitle('')
      setDescription('')
      setError(null)
      if (!initialProjectId) setSelectedProjectId('')
      if (!initialTaskId) setSelectedTaskId('')
    }
  }, [isOpen, initialProjectId, initialTaskId, loadProjects, loadTasks])

  const handleProjectChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const pId = e.target.value
    setSelectedProjectId(pId)
    setSelectedTaskId('')
    if (pId) {
      loadTasks(pId)
    } else {
      setTasks([])
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) {
      setError('Judul risiko wajib diisi.')
      return
    }
    if (!selectedProjectId) {
      setError('Pilih proyek terlebih dahulu.')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const created = await createRisk({
        project_id: selectedProjectId,
        task_id: selectedTaskId || null,
        title: title.trim(),
        category,
        description: description.trim(),
      })
      onSuccess(created)
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Gagal mengajukan risiko.')
    } finally {
      setIsLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
      <div
        className="bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 w-full max-w-lg overflow-hidden animate-in fade-in zoom-in-95 duration-150"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-6 py-4 border-b border-slate-200 dark:border-slate-800 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-slate-100">
            Ajukan Risiko Baru
          </h2>
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

          {/* Project */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Proyek <span className="text-rose-500">*</span>
            </label>
            <select
              value={selectedProjectId}
              onChange={handleProjectChange}
              disabled={!!initialProjectId || isLoading}
              className="w-full text-sm px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:opacity-60"
              required
            >
              <option value="">Pilih Proyek...</option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.code} - {p.name}
                </option>
              ))}
            </select>
          </div>

          {/* Linked Task (Optional) */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Task Terkait (Opsional)
            </label>
            <select
              value={selectedTaskId}
              onChange={(e) => setSelectedTaskId(e.target.value)}
              disabled={!!initialTaskId || !selectedProjectId || isLoading}
              className="w-full text-sm px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:opacity-60"
            >
              <option value="">Tidak ada keterkaitan spesifik (Tingkat Proyek)</option>
              {tasks.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.title}
                </option>
              ))}
            </select>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-1">
              Jika ditautkan ke task, kartu task di Kanban board akan menampilkan indikator peringatan risiko.
            </p>
          </div>

          {/* Title */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Judul Risiko <span className="text-rose-500">*</span>
            </label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Contoh: Keterlambatan respons gateway pembayaran pihak ketiga"
              className="w-full text-sm px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              required
              disabled={isLoading}
            />
          </div>

          {/* Category */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Kategori <span className="text-rose-500">*</span>
            </label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value as RiskCategory)}
              className="w-full text-sm px-3 py-2 rounded-lg border border-slate-300 dark:border-slate-700 bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              disabled={isLoading}
            >
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>
                  {c.label}
                </option>
              ))}
            </select>
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
              Deskripsi Detail (Opsional)
            </label>
            <textarea
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Jelaskan pemicu, kemungkinan dampak, atau konteks risiko..."
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
              {isLoading ? 'Menyimpan...' : 'Ajukan Risiko'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
