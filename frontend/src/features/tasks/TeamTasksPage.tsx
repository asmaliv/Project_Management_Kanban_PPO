import { useState, useEffect, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import { fetchTasks, updateTask, deleteTask } from './api'
import { fetchProjects } from '../projects/api'
import TaskDetailDrawer from './TaskDetailDrawer'
import { useAuth } from '../auth/useAuth'
import type { Task, TaskPriority, UpdateTaskPayload } from './types'
import type { Project } from '../projects/types'

const PRIORITY_BADGES: Record<TaskPriority, string> = {
  critical: 'bg-red-100 text-red-800 border-red-300',
  high: 'bg-orange-100 text-orange-800 border-orange-300',
  medium: 'bg-amber-100 text-amber-800 border-amber-300',
  low: 'bg-blue-100 text-blue-800 border-blue-300',
}

interface TeamTasksPageProps {
  initialMineOnly?: boolean
}

export default function TeamTasksPage({ initialMineOnly = false }: TeamTasksPageProps) {
  const { user } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()

  const [tasks, setTasks] = useState<Task[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [projects, setProjects] = useState<Project[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Drawer state
  const [selectedTask, setSelectedTask] = useState<Task | null>(null)
  const [isDrawerOpen, setIsDrawerOpen] = useState(false)

  // Filters
  const selectedProject = searchParams.get('project_id') || ''
  const isOverdue = searchParams.get('overdue') === 'true'
  const isMine = initialMineOnly || searchParams.get('mine') === 'true'
  const searchQuery = searchParams.get('search') || ''
  const selectedPriority = searchParams.get('priority') || ''

  const loadTasksList = useCallback(async () => {
    setIsLoading(true)
    setError(null)
    try {
      const res = await fetchTasks({
        project_id: selectedProject || undefined,
        overdue: isOverdue ? true : undefined,
        mine: isMine ? true : undefined,
        limit: 100,
      })
      let list = res.tasks
      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase()
        list = list.filter((t) => t.title.toLowerCase().includes(q) || t.description?.toLowerCase().includes(q))
      }
      if (selectedPriority) {
        list = list.filter((t) => t.priority === selectedPriority)
      }
      setTasks(list)
      setTotalCount(res.total)
    } catch (err: unknown) {
      if (err instanceof Error) setError(err.message)
      else setError('Gagal memuat daftar tugas')
    } finally {
      setIsLoading(false)
    }
  }, [selectedProject, isOverdue, isMine, searchQuery, selectedPriority])

  useEffect(() => {
    loadTasksList()
  }, [loadTasksList])

  useEffect(() => {
    async function loadProj() {
      try {
        const p = await fetchProjects()
        setProjects(p)
      } catch {
        // ignore
      }
    }
    loadProj()
  }, [])

  const updateParam = (key: string, value: string) => {
    const next = new URLSearchParams(searchParams)
    if (value) next.set(key, value)
    else next.delete(key)
    setSearchParams(next)
  }

  const handleUpdateTask = async (taskId: string, payload: UpdateTaskPayload) => {
    await updateTask(taskId, payload)
    loadTasksList()
    setIsDrawerOpen(false)
  }

  const handleDeleteTask = async (taskId: string) => {
    await deleteTask(taskId)
    loadTasksList()
    setIsDrawerOpen(false)
  }

  const isTL = user?.role === 'team_leader'
  const isPM = user?.role === 'project_manager'

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="bg-surface p-6 rounded-2xl border border-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <span className="p-2 rounded-xl bg-emerald-500/10 text-emerald-600">
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
              </svg>
            </span>
            <h1 className="text-xl font-bold text-text">
              {isMine ? 'Tugas Saya' : 'Daftar Tugas Tim'}
            </h1>
          </div>
          <p className="text-sm text-text-muted mt-1">
            {isMine
              ? 'Tugas-tugas yang ditugaskan kepada akun Anda di semua proyek aktif.'
              : 'Pantau status, tenggat waktu, dan progres pengerjaan task di seluruh tim proyek.'}
          </p>
        </div>

        {/* Quick toggle for TL */}
        {!initialMineOnly && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => updateParam('mine', isMine ? '' : 'true')}
              className={`px-3.5 py-2 text-xs font-semibold rounded-xl border transition-colors flex items-center gap-1.5 ${
                isMine
                  ? 'bg-primary text-white border-primary'
                  : 'bg-surface border-border text-text hover:bg-surface-muted'
              }`}
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              {isMine ? 'Semua Tugas Tim' : 'Hanya Tugas Saya'}
            </button>
          </div>
        )}
      </div>

      {/* Filter Bar */}
      <div className="bg-surface p-4 rounded-xl border border-border shadow-sm flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2 flex-1">
          {/* Project filter */}
          <select
            value={selectedProject}
            onChange={(e) => updateParam('project_id', e.target.value)}
            className="px-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          >
            <option value="">Semua Proyek</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                [{p.code}] {p.name}
              </option>
            ))}
          </select>

          {/* Priority filter */}
          <select
            value={selectedPriority}
            onChange={(e) => updateParam('priority', e.target.value)}
            className="px-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          >
            <option value="">Semua Prioritas</option>
            <option value="critical">Critical</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>

          {/* Overdue filter */}
          <label className="flex items-center gap-1.5 px-3 py-1.5 text-xs border border-border rounded-lg bg-surface text-text cursor-pointer hover:bg-surface-muted">
            <input
              type="checkbox"
              checked={isOverdue}
              onChange={(e) => updateParam('overdue', e.target.checked ? 'true' : '')}
              className="rounded text-rose-600 focus:ring-rose-500"
            />
            <span className={isOverdue ? 'font-bold text-rose-600' : ''}>Hanya Terlambat</span>
          </label>

          {(selectedProject || isOverdue || selectedPriority || searchQuery) && (
            <button
              onClick={() => setSearchParams(new URLSearchParams())}
              className="px-2.5 py-1.5 text-xs text-text-muted hover:text-text bg-surface-muted hover:bg-slate-200 rounded-lg transition-colors"
            >
              Reset Filter
            </button>
          )}
        </div>

        {/* Search */}
        <div className="relative min-w-[240px]">
          <input
            type="text"
            placeholder="Cari judul task..."
            value={searchQuery}
            onChange={(e) => updateParam('search', e.target.value)}
            className="w-full pl-8 pr-3 py-1.5 text-xs bg-surface border border-border rounded-lg text-text focus:ring-1 focus:ring-primary"
          />
          <svg className="w-3.5 h-3.5 text-text-muted absolute left-2.5 top-2.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </div>
      </div>

      {/* Task List Table */}
      <div className="bg-surface rounded-xl border border-border shadow-sm overflow-hidden">
        {isLoading ? (
          <div className="p-8 text-center text-text-muted text-sm flex items-center justify-center gap-2">
            <div className="w-4 h-4 border-2 border-primary/20 border-t-primary rounded-full animate-spin" />
            Memuat daftar task...
          </div>
        ) : error ? (
          <div className="p-8 text-center text-red-600 text-sm">{error}</div>
        ) : tasks.length === 0 ? (
          <div className="p-12 text-center">
            <svg className="w-12 h-12 text-text-muted/40 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
            </svg>
            <h3 className="text-sm font-bold text-text">Tidak ada task ditemukan</h3>
            <p className="text-xs text-text-muted mt-1">
              Tidak ada tugas yang cocok dengan filter atau kriteria pencarian saat ini.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-border bg-surface-muted/60 text-text-muted font-semibold">
                  <th className="py-3 px-4">Judul Task</th>
                  <th className="py-3 px-4">Kolom / Status</th>
                  <th className="py-3 px-4">Prioritas</th>
                  <th className="py-3 px-4">Penerima Tugas</th>
                  <th className="py-3 px-4">Tenggat Waktu</th>
                  <th className="py-3 px-4">Progres</th>
                  <th className="py-3 px-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {tasks.map((t) => {
                  const isTaskOverdue =
                    t.due_date &&
                    new Date(t.due_date) < new Date() &&
                    !t.column_name?.toLowerCase().includes('done') &&
                    !t.column_name?.toLowerCase().includes('selesai')

                  return (
                    <tr
                      key={t.id}
                      onClick={() => {
                        setSelectedTask(t)
                        setIsDrawerOpen(true)
                      }}
                      className="hover:bg-surface-muted/40 cursor-pointer transition-colors"
                    >
                      <td className="py-3 px-4 max-w-sm">
                        <div className="font-semibold text-text truncate">{t.title}</div>
                        {t.labels && t.labels.length > 0 && (
                          <div className="flex gap-1 mt-1 flex-wrap">
                            {t.labels.map((lbl, idx) => (
                              <span
                                key={idx}
                                className="px-1.5 py-0.2 rounded bg-slate-100 text-[10px] text-slate-700 border border-slate-200"
                              >
                                {lbl}
                              </span>
                            ))}
                          </div>
                        )}
                      </td>
                      <td className="py-3 px-4 whitespace-nowrap">
                        <span className="px-2 py-0.5 rounded font-medium bg-slate-100 text-slate-800 border border-slate-200">
                          {t.column_name || 'Board Column'}
                        </span>
                      </td>
                      <td className="py-3 px-4 whitespace-nowrap">
                        <span
                          className={`px-2 py-0.5 rounded-full font-bold border capitalize ${
                            PRIORITY_BADGES[t.priority] || 'bg-slate-100 text-slate-800'
                          }`}
                        >
                          {t.priority}
                        </span>
                      </td>
                      <td className="py-3 px-4 whitespace-nowrap text-text">
                        {t.assignee_name || (
                          <span className="text-text-muted italic">Belum ditugaskan</span>
                        )}
                      </td>
                      <td className="py-3 px-4 whitespace-nowrap">
                        {t.due_date ? (
                          <span className={isTaskOverdue ? 'font-bold text-rose-600' : 'text-text-muted'}>
                            {new Date(t.due_date).toLocaleDateString('id-ID', {
                              day: 'numeric',
                              month: 'short',
                              year: 'numeric',
                            })}
                            {isTaskOverdue && ' (Terlambat)'}
                          </span>
                        ) : (
                          <span className="text-text-muted">-</span>
                        )}
                      </td>
                      <td className="py-3 px-4 whitespace-nowrap">
                        <div className="w-24 bg-border rounded-full h-2 overflow-hidden">
                          <div
                            className="bg-primary h-full rounded-full transition-all"
                            style={{ width: `${t.progress_percent}%` }}
                          />
                        </div>
                        <span className="text-[10px] text-text-muted mt-0.5 block">
                          {t.progress_percent}%
                        </span>
                      </td>
                      <td className="py-3 px-4 text-right whitespace-nowrap">
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            setSelectedTask(t)
                            setIsDrawerOpen(true)
                          }}
                          className="px-2.5 py-1 text-xs font-medium text-primary hover:text-primary-hover hover:underline"
                        >
                          Detail &rarr;
                        </button>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}

        {!isLoading && tasks.length > 0 && (
          <div className="py-3 px-4 border-t border-border bg-surface-muted/30 text-xs text-text-muted flex items-center justify-between">
            <span>Menampilkan {tasks.length} dari {totalCount} task</span>
          </div>
        )}
      </div>

      {/* Task Detail Drawer */}
      <TaskDetailDrawer
        task={selectedTask}
        isOpen={isDrawerOpen}
        onClose={() => {
          setIsDrawerOpen(false)
          setSelectedTask(null)
        }}
        isPM={isPM}
        canEdit={isPM || isTL}
        members={[]}
        ownerName="PM Owner"
        ownerId=""
        onUpdateTask={handleUpdateTask}
        onDeleteTask={isPM ? handleDeleteTask : undefined}
      />
    </div>
  )
}
