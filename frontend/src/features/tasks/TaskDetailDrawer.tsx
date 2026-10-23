import React, { useState, useEffect, useCallback } from 'react'
import type { Task, TaskComment, TaskPriority, UpdateTaskPayload } from './types'
import type { ProjectMember } from '../projects/types'
import { fetchTaskComments, createTaskComment } from './api'

interface TaskDetailDrawerProps {
  task: Task | null
  isOpen: boolean
  onClose: () => void
  isPM: boolean
  canEdit: boolean
  members: ProjectMember[]
  ownerName: string
  ownerId: string
  onUpdateTask: (taskId: string, payload: UpdateTaskPayload) => Promise<void>
  onDeleteTask?: (taskId: string) => Promise<void>
}

const PRIORITY_OPTIONS: { value: TaskPriority; label: string }[] = [
  { value: 'low', label: 'Rendah (Low)' },
  { value: 'medium', label: 'Sedang (Medium)' },
  { value: 'high', label: 'Tinggi (High)' },
  { value: 'critical', label: 'Kritis (Critical)' },
]

export default function TaskDetailDrawer({
  task,
  isOpen,
  onClose,
  isPM,
  canEdit,
  members,
  ownerName,
  ownerId,
  onUpdateTask,
  onDeleteTask,
}: TaskDetailDrawerProps) {
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState<TaskPriority>('medium')
  const [assigneeId, setAssigneeId] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [progressPercent, setProgressPercent] = useState(0)

  // Comments state
  const [comments, setComments] = useState<TaskComment[]>([])
  const [newComment, setNewComment] = useState('')
  const [isLoadingComments, setIsLoadingComments] = useState(false)
  const [isSubmittingComment, setIsSubmittingComment] = useState(false)

  // Action status
  const [isSaving, setIsSaving] = useState(false)
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // Reset local state when task changes
  useEffect(() => {
    if (task) {
      setTitle(task.title)
      setDescription(task.description || '')
      setPriority(task.priority)
      setAssigneeId(task.assignee_id || '')
      setDueDate(task.due_date || '')
      setProgressPercent(task.progress_percent || 0)
      setError(null)
      setSuccess(null)
    }
  }, [task])

  const loadComments = useCallback(async () => {
    if (!task) return
    try {
      setIsLoadingComments(true)
      const list = await fetchTaskComments(task.id)
      setComments(list)
    } catch {
      // silently ignore comments fetch error
    } finally {
      setIsLoadingComments(false)
    }
  }, [task])

  useEffect(() => {
    if (isOpen && task) {
      void loadComments()
    }
  }, [isOpen, task, loadComments])

  if (!isOpen || !task) return null

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!title.trim()) {
      setError('Judul task wajib diisi.')
      return
    }

    try {
      setIsSaving(true)
      setError(null)
      await onUpdateTask(task.id, {
        version: task.version,
        title: title.trim(),
        description: description.trim(),
        priority,
        assignee_id: assigneeId || null,
        due_date: dueDate || null,
        progress_percent: progressPercent,
      })
      setSuccess('Task berhasil diperbarui.')
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal memperbarui task.')
      }
    } finally {
      setIsSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!onDeleteTask) return
    if (!window.confirm(`Hapus task "${task.title}"? Aksi ini akan menghapus task dari papan.`)) {
      return
    }

    try {
      setIsDeleting(true)
      setError(null)
      await onDeleteTask(task.id)
      onClose()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal menghapus task.')
      }
    } finally {
      setIsDeleting(false)
    }
  }

  const handleAddComment = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newComment.trim()) return

    try {
      setIsSubmittingComment(true)
      const created = await createTaskComment(task.id, newComment.trim())
      setComments((prev) => [...prev, created])
      setNewComment('')
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal mengirim komentar.')
      }
    } finally {
      setIsSubmittingComment(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 overflow-hidden bg-slate-900/40 backdrop-blur-xs flex justify-end">
      <div className="w-full max-w-2xl bg-white h-full shadow-2xl border-l border-slate-200 flex flex-col animate-slide-left">
        {/* Drawer Header */}
        <div className="p-6 border-b border-slate-100 flex items-center justify-between gap-4 bg-slate-50/60">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono font-bold text-slate-500 bg-slate-200/60 px-2 py-0.5 rounded">
                v{task.version}
              </span>
              {task.column_name && (
                <span className="text-xs font-medium text-blue-700 bg-blue-50 border border-blue-100 px-2.5 py-0.5 rounded-md">
                  Kolom: {task.column_name}
                </span>
              )}
            </div>
            <h3 className="text-lg font-bold text-slate-900 line-clamp-1">{task.title}</h3>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 rounded-lg p-1.5 hover:bg-white transition-colors"
          >
            &times;
          </button>
        </div>

        {/* Alerts */}
        <div className="px-6 pt-4">
          {error && (
            <div className="p-3.5 rounded-xl bg-red-50 border border-red-200 text-red-700 text-xs flex items-start justify-between gap-2 mb-2">
              <span>{error}</span>
              <button type="button" onClick={() => setError(null)} className="text-red-400 hover:text-red-600">
                &times;
              </button>
            </div>
          )}
          {success && (
            <div className="p-3.5 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-xs flex items-start justify-between gap-2 mb-2">
              <span>{success}</span>
              <button type="button" onClick={() => setSuccess(null)} className="text-emerald-500 hover:text-emerald-700">
                &times;
              </button>
            </div>
          )}
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          <form id="task-edit-form" onSubmit={handleSave} className="space-y-4">
            {/* Title */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Judul Task</label>
              <input
                type="text"
                value={title}
                disabled={!canEdit}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50 disabled:text-slate-500 font-medium"
              />
            </div>

            {/* Description */}
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1">Deskripsi</label>
              <textarea
                rows={3}
                value={description}
                disabled={!canEdit}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Tambahkan detail atau instruksi pekerjaan..."
                className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50 disabled:text-slate-500 resize-none"
              />
            </div>

            {/* Priority & Assignee */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Prioritas</label>
                <select
                  value={priority}
                  disabled={!canEdit}
                  onChange={(e) => setPriority(e.target.value as TaskPriority)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50 bg-white"
                >
                  {PRIORITY_OPTIONS.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Penugasan (Assignee)</label>
                <select
                  value={assigneeId}
                  disabled={!canEdit}
                  onChange={(e) => setAssigneeId(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50 bg-white"
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

            {/* Due Date & Progress Slider */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">Tenggat Waktu</label>
                <input
                  type="date"
                  value={dueDate}
                  disabled={!canEdit}
                  onChange={(e) => setDueDate(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-50"
                />
              </div>

              <div>
                <div className="flex items-center justify-between text-xs font-semibold text-slate-700 mb-1">
                  <span>Progress</span>
                  <span className="font-mono text-blue-600">{progressPercent}%</span>
                </div>
                <input
                  type="range"
                  min="0"
                  max="100"
                  step="5"
                  disabled={!canEdit}
                  value={progressPercent}
                  onChange={(e) => setProgressPercent(Number(e.target.value))}
                  className="w-full mt-2 accent-blue-600 cursor-pointer disabled:opacity-50"
                />
              </div>
            </div>

            {canEdit && (
              <div className="flex items-center justify-between pt-2">
                {isPM && onDeleteTask ? (
                  <button
                    type="button"
                    onClick={handleDelete}
                    disabled={isDeleting || isSaving}
                    className="px-3.5 py-2 text-xs font-semibold text-red-600 hover:text-red-700 hover:bg-red-50 rounded-xl transition-colors"
                  >
                    {isDeleting ? 'Menghapus...' : 'Hapus Task'}
                  </button>
                ) : (
                  <div />
                )}

                <button
                  type="submit"
                  disabled={isSaving || isDeleting}
                  className="px-5 py-2 text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 rounded-xl transition-all shadow-xs"
                >
                  {isSaving ? 'Menyimpan...' : 'Simpan Perubahan'}
                </button>
              </div>
            )}
          </form>

          {/* Activity & Comments Thread */}
          <div className="border-t border-slate-100 pt-6 space-y-4">
            <h4 className="text-sm font-bold text-slate-900 flex items-center gap-2">
              <svg className="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
              <span>Diskusi & Komentar ({comments.length})</span>
            </h4>

            {/* Comments List */}
            {isLoadingComments ? (
              <div className="py-4 text-center text-xs text-slate-400">Memuat komentar...</div>
            ) : comments.length === 0 ? (
              <div className="py-6 text-center text-xs text-slate-400 bg-slate-50 rounded-2xl border border-dashed border-slate-200">
                Belum ada komentar untuk task ini.
              </div>
            ) : (
              <div className="space-y-3">
                {comments.map((c) => (
                  <div key={c.id} className="p-3.5 rounded-2xl bg-slate-50 border border-slate-100 space-y-1.5">
                    <div className="flex items-center justify-between text-xs">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-slate-800">{c.author_name}</span>
                        <span className="text-[10px] text-slate-400 font-mono">({c.author_role})</span>
                      </div>
                      <span className="text-[11px] text-slate-400">
                        {new Date(c.created_at).toLocaleString('id-ID', {
                          dateStyle: 'medium',
                          timeStyle: 'short',
                        })}
                      </span>
                    </div>
                    <p className="text-xs text-slate-600 leading-relaxed whitespace-pre-wrap">{c.body}</p>
                  </div>
                ))}
              </div>
            )}

            {/* Add Comment Input */}
            {canEdit && (
              <form onSubmit={handleAddComment} className="pt-2 space-y-2">
                <textarea
                  rows={2}
                  value={newComment}
                  onChange={(e) => setNewComment(e.target.value)}
                  placeholder="Tulis komentar atau pembaruan teknis..."
                  className="w-full px-3.5 py-2.5 rounded-xl border border-slate-200 text-xs focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                />
                <div className="flex justify-end">
                  <button
                    type="submit"
                    disabled={isSubmittingComment || !newComment.trim()}
                    className="px-4 py-1.5 text-xs font-semibold text-white bg-slate-900 hover:bg-slate-800 disabled:opacity-40 rounded-xl transition-all shadow-xs"
                  >
                    {isSubmittingComment ? 'Mengirim...' : 'Kirim Komentar'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
