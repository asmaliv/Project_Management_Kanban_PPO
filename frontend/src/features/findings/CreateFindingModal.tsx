import React, { useState, useEffect, useCallback } from 'react'
import { createFinding } from './api'
import { fetchProjects } from '../projects/api'
import { fetchProjectTasks } from '../tasks/api'
import type { FindingSeverity, AuditFinding } from './types'
import type { Project } from '../projects/types'
import type { Task } from '../tasks/types'

interface CreateFindingModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: (newFinding: AuditFinding) => void
  initialProjectId?: string
  initialTaskId?: string
}

const SEVERITIES: { value: FindingSeverity; label: string; desc: string; color: string }[] = [
  { value: 'critical', label: 'Critical', desc: 'Berdampak langsung terhadap integritas sistem, data rahasia, atau kepatuhan regulasi.', color: 'border-red-500 bg-red-50 text-red-700' },
  { value: 'high', label: 'High', desc: 'Penyimpangan signifikan terhadap kebijakan kontrol internal bank.', color: 'border-orange-500 bg-orange-50 text-orange-700' },
  { value: 'medium', label: 'Medium', desc: 'Penyimpangan operasional yang memerlukan perbaikan terencana.', color: 'border-amber-500 bg-amber-50 text-amber-700' },
  { value: 'low', label: 'Low', desc: 'Observasi prosedural minor atau peluang peningkatan efisiensi.', color: 'border-blue-500 bg-blue-50 text-blue-700' },
]

export const CreateFindingModal: React.FC<CreateFindingModalProps> = ({
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
  const [severity, setSeverity] = useState<FindingSeverity>('medium')
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
      const activeList = data.filter((p) => p.status !== 'archived')
      setProjects(activeList)
      if (!selectedProjectId && activeList.length > 0) {
        setSelectedProjectId(activeList[0].id)
        void loadTasks(activeList[0].id)
      }
    } catch {
      // ignore
    }
  }, [selectedProjectId, loadTasks])

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
      setError('Judul temuan audit wajib diisi.')
      return
    }
    if (title.trim().length > 255) {
      setError('Judul temuan audit maksimal 255 karakter.')
      return
    }
    if (!selectedProjectId) {
      setError('Proyek terkait wajib dipilih.')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      const res = await createFinding({
        project_id: selectedProjectId,
        task_id: selectedTaskId || null,
        title: title.trim(),
        description: description.trim(),
        severity,
      })
      onSuccess(res)
      onClose()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal membuat temuan audit')
      }
    } finally {
      setIsLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto bg-slate-900/60 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="relative bg-surface rounded-2xl shadow-2xl max-w-lg w-full border border-border overflow-hidden animate-in fade-in zoom-in-95 duration-150">
        <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-surface-muted">
          <div>
            <h2 className="text-lg font-bold text-text flex items-center gap-2">
              <span className="p-1 rounded bg-amber-100 text-amber-700">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
              </span>
              Buat Temuan Audit Baru
            </h2>
            <p className="text-xs text-text-muted mt-0.5">
              Hanya role Audit yang memiliki wewenang menerbitkan temuan audit perbankan.
            </p>
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

        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 text-red-700 text-xs rounded-xl flex items-start gap-2">
              <svg className="w-4 h-4 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <span>{error}</span>
            </div>
          )}

          <div>
            <label className="block text-xs font-semibold text-text mb-1">
              Proyek Terkait <span className="text-red-500">*</span>
            </label>
            <select
              value={selectedProjectId}
              onChange={handleProjectChange}
              disabled={!!initialProjectId}
              className="w-full px-3 py-2 text-sm bg-surface border border-border rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:bg-surface-muted"
            >
              {projects.length === 0 ? (
                <option value="">Tidak ada proyek aktif</option>
              ) : (
                projects.map((p) => (
                  <option key={p.id} value={p.id}>
                    [{p.code}] {p.name}
                  </option>
                ))
              )}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-text mb-1">
              Task Terkait (Opsional)
            </label>
            <select
              value={selectedTaskId}
              onChange={(e) => setSelectedTaskId(e.target.value)}
              disabled={!!initialTaskId || !selectedProjectId}
              className="w-full px-3 py-2 text-sm bg-surface border border-border rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary disabled:bg-surface-muted"
            >
              <option value="">-- Tidak terkait spesifik ke task --</option>
              {tasks.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.title}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-text mb-1">
              Tingkat Keparahan (Severity) <span className="text-red-500">*</span>
            </label>
            <div className="grid grid-cols-2 gap-2">
              {SEVERITIES.map((s) => (
                <button
                  type="button"
                  key={s.value}
                  onClick={() => setSeverity(s.value)}
                  className={`p-2.5 rounded-xl border text-left transition-all ${
                    severity === s.value
                      ? `${s.color} font-bold ring-2 ring-primary/20`
                      : 'border-border bg-surface text-text hover:bg-surface-muted'
                  }`}
                >
                  <div className="text-xs font-bold capitalize">{s.label}</div>
                  <div className="text-[10px] text-text-muted mt-0.5 line-clamp-2">
                    {s.desc}
                  </div>
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-text mb-1">
              Judul Temuan <span className="text-red-500">*</span>
            </label>
            <input
              type="text"
              required
              maxLength={255}
              placeholder="Contoh: Pemisahan tugas pada deployment belum terdokumentasi"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="w-full px-3 py-2 text-sm bg-surface border border-border rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-text mb-1">
              Deskripsi Observasi & Bukti Temuan
            </label>
            <textarea
              rows={4}
              placeholder="Jelaskan temuan audit, rujukan SOP/kebijakan kontrol, serta rekomendasi awal..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 text-sm bg-surface border border-border rounded-xl focus:ring-2 focus:ring-primary/20 focus:border-primary resize-y"
            />
          </div>

          <div className="flex justify-end gap-3 pt-3 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={isLoading}
              className="px-4 py-2 text-xs font-medium text-text-muted hover:text-text bg-surface-muted hover:bg-slate-200 rounded-xl transition-colors"
            >
              Batal
            </button>
            <button
              type="submit"
              disabled={isLoading || !title.trim() || !selectedProjectId}
              className="px-4 py-2 text-xs font-bold text-white bg-primary hover:bg-primary-hover disabled:opacity-50 rounded-xl transition-colors flex items-center gap-1.5 shadow-sm"
            >
              {isLoading && (
                <div className="w-3.5 h-3.5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
              )}
              Terbitkan Temuan
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
