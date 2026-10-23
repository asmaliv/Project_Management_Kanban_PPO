import React, { useState, useEffect, useCallback } from 'react'
import type { Task, CreateTaskPayload, UpdateTaskPayload } from './types'
import type { BoardColumn, ProjectMember } from '../projects/types'
import {
  fetchProjectTasks,
  createTask,
  updateTask,
  moveTask,
  deleteTask,
} from './api'
import TaskCard from './TaskCard'
import CreateTaskModal from './CreateTaskModal'
import TaskDetailDrawer from './TaskDetailDrawer'

interface KanbanBoardProps {
  projectId: string
  columns: BoardColumn[]
  members: ProjectMember[]
  ownerName: string
  ownerId: string
  isPM: boolean
  canWrite: boolean
}

export default function KanbanBoard({
  projectId,
  columns,
  members,
  ownerName,
  ownerId,
  isPM,
  canWrite,
}: KanbanBoardProps) {
  const [tasks, setTasks] = useState<Task[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [boardError, setBoardError] = useState<string | null>(null)
  const [boardSuccess, setBoardSuccess] = useState<string | null>(null)

  // Drag-and-drop state
  const [dragOverColId, setDragOverColId] = useState<string | null>(null)

  // Modals state
  const [selectedTask, setSelectedTask] = useState<Task | null>(null)
  const [isDetailDrawerOpen, setIsDetailDrawerOpen] = useState(false)
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [createModalColumnId, setCreateModalColumnId] = useState<string>('')

  const loadTasks = useCallback(async () => {
    try {
      setIsLoading(true)
      setBoardError(null)
      const data = await fetchProjectTasks(projectId)
      setTasks(data)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setBoardError(err.message)
      } else {
        setBoardError('Gagal memuat task papan Kanban.')
      }
    } finally {
      setIsLoading(false)
    }
  }, [projectId])

  useEffect(() => {
    void loadTasks()
  }, [loadTasks])

  // Move handler
  const handleMoveTask = async (
    taskId: string,
    toColumnId: string,
    position: number,
    version: number,
  ) => {
    try {
      setBoardError(null)
      const updated = await moveTask(taskId, {
        to_column_id: toColumnId,
        position,
        version,
      })

      // Update state with updated task
      setTasks((prev) =>
        prev.map((t) => (t.id === updated.id ? updated : t)),
      )
      // Reload tasks to synchronize all reordered positions
      void loadTasks()
      setBoardSuccess(`Task berhasil dipindahkan.`)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setBoardError(err.message)
      } else {
        setBoardError('Gagal memindahkan task.')
      }
      // Reload to ensure UI reflects database state on conflict
      void loadTasks()
    }
  }

  // Drag and drop handlers
  const handleDragOver = (e: React.DragEvent, colId: string) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
    if (dragOverColId !== colId) {
      setDragOverColId(colId)
    }
  }

  const handleDragLeave = (colId: string) => {
    if (dragOverColId === colId) {
      setDragOverColId(null)
    }
  }

  const handleDrop = async (e: React.DragEvent, targetCol: BoardColumn) => {
    e.preventDefault()
    setDragOverColId(null)

    const rawData = e.dataTransfer.getData('text/plain')
    if (!rawData) return

    try {
      const { taskId, version } = JSON.parse(rawData) as { taskId: string; version: number }
      const movingTask = tasks.find((t) => t.id === taskId)
      if (!movingTask) return

      // Maker-checker check: moving to Done column requires PM
      if (targetCol.is_done_column && !isPM) {
        setBoardError('Hanya Project Manager yang dapat memindahkan task ke kolom Done.')
        return
      }

      // Check WIP limit on client for immediate feedback
      if (
        movingTask.column_id !== targetCol.id &&
        targetCol.wip_limit &&
        targetCol.wip_limit > 0
      ) {
        const countInCol = tasks.filter((t) => t.column_id === targetCol.id).length
        if (countInCol >= targetCol.wip_limit) {
          setBoardError(
            `WIP Limit tercapai! Kolom "${targetCol.name}" dibatasi maksimal ${targetCol.wip_limit} task aktif.`,
          )
          return
        }
      }

      // Append to the end of target column
      const targetColTasks = tasks.filter((t) => t.column_id === targetCol.id)
      const targetPos = targetColTasks.length + 1

      await handleMoveTask(taskId, targetCol.id, targetPos, version)
    } catch {
      // ignore JSON parse error
    }
  }

  // Create Task handler
  const handleCreateTask = async (payload: CreateTaskPayload) => {
    const created = await createTask(projectId, payload)
    setTasks((prev) => [...prev, created])
    setBoardSuccess(`Task "${created.title}" berhasil dibuat.`)
    void loadTasks()
  }

  // Update Task handler
  const handleUpdateTask = async (taskId: string, payload: UpdateTaskPayload) => {
    const updated = await updateTask(taskId, payload)
    setTasks((prev) => prev.map((t) => (t.id === updated.id ? updated : t)))
    if (selectedTask?.id === updated.id) {
      setSelectedTask(updated)
    }
    setBoardSuccess(`Perubahan task berhasil disimpan.`)
  }

  // Delete Task handler
  const handleDeleteTask = async (taskId: string) => {
    await deleteTask(taskId)
    setTasks((prev) => prev.filter((t) => t.id !== taskId))
    setSelectedTask(null)
    setIsDetailDrawerOpen(false)
    setBoardSuccess(`Task berhasil dihapus.`)
  }

  const openCreateModalForColumn = (colId: string) => {
    setCreateModalColumnId(colId)
    setIsCreateModalOpen(true)
  }

  const handleCardClick = (t: Task) => {
    setSelectedTask(t)
    setIsDetailDrawerOpen(true)
  }

  if (isLoading && tasks.length === 0) {
    return (
      <div className="flex gap-6 overflow-x-auto pb-6 animate-pulse">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="w-80 shrink-0 h-96 bg-slate-100 rounded-3xl" />
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Notifications */}
      {boardError && (
        <div className="p-4 rounded-2xl bg-red-50 border border-red-200 text-red-700 text-sm flex items-start justify-between gap-3 animate-fade-in">
          <div className="flex items-start gap-2.5">
            <svg className="w-5 h-5 text-red-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{boardError}</span>
          </div>
          <button type="button" onClick={() => setBoardError(null)} className="text-red-400 hover:text-red-600">
            &times;
          </button>
        </div>
      )}

      {boardSuccess && (
        <div className="p-4 rounded-2xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-sm flex items-start justify-between gap-3 animate-fade-in">
          <div className="flex items-start gap-2.5">
            <svg className="w-5 h-5 text-emerald-600 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            <span>{boardSuccess}</span>
          </div>
          <button type="button" onClick={() => setBoardSuccess(null)} className="text-emerald-500 hover:text-emerald-700">
            &times;
          </button>
        </div>
      )}

      {/* Board Controls Bar */}
      <div className="flex items-center justify-between gap-4">
        <div className="text-xs text-slate-500 font-medium">
          Total: <span className="font-bold text-slate-900">{tasks.length}</span> task di{' '}
          <span className="font-bold text-slate-900">{columns.length}</span> kolom
        </div>

        {canWrite && (
          <button
            type="button"
            onClick={() => openCreateModalForColumn(columns[0]?.id || '')}
            className="inline-flex items-center gap-2 px-4 py-2 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-xl transition-all shadow-xs"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            <span>Tambah Task</span>
          </button>
        )}
      </div>

      {/* Kanban Board Columns Horizontal Scroll */}
      <div className="flex gap-6 overflow-x-auto pb-8 pt-1 items-start min-h-[500px]">
        {columns.map((column) => {
          const colTasks = tasks
            .filter((t) => t.column_id === column.id)
            .sort((a, b) => a.position - b.position)

          const isOverLimit =
            Boolean(column.wip_limit && column.wip_limit > 0 && colTasks.length >= column.wip_limit)
          const isDraggingOver = dragOverColId === column.id

          return (
            <div
              key={column.id}
              onDragOver={(e) => handleDragOver(e, column.id)}
              onDragLeave={() => handleDragLeave(column.id)}
              onDrop={(e) => void handleDrop(e, column)}
              className={`w-80 shrink-0 bg-slate-50/80 rounded-3xl p-4 border transition-all flex flex-col max-h-[780px] ${
                isDraggingOver
                  ? 'border-blue-400 bg-blue-50/40 ring-2 ring-blue-400/20 shadow-md'
                  : 'border-slate-200/90 shadow-2xs'
              }`}
            >
              {/* Column Header */}
              <div className="flex items-center justify-between gap-2 pb-3 border-b border-slate-200/80 mb-3">
                <div className="flex items-center gap-2 min-w-0">
                  <h3 className="text-sm font-bold text-slate-800 truncate" title={column.name}>
                    {column.name}
                  </h3>
                  <span className="shrink-0 text-[11px] font-bold text-slate-600 bg-white border border-slate-200 px-2 py-0.5 rounded-full shadow-2xs">
                    {colTasks.length}
                  </span>
                </div>

                <div className="flex items-center gap-1.5 shrink-0">
                  {/* WIP Limit Pill */}
                  {column.wip_limit && column.wip_limit > 0 ? (
                    <span
                      title={`Batas WIP: ${column.wip_limit}`}
                      className={`text-[10px] font-semibold px-2 py-0.5 rounded-md border ${
                        isOverLimit
                          ? 'bg-rose-50 text-rose-700 border-rose-200'
                          : 'bg-slate-100 text-slate-600 border-slate-200'
                      }`}
                    >
                      WIP {colTasks.length}/{column.wip_limit}
                    </span>
                  ) : null}

                  {/* Maker-checker Done Column Indicator */}
                  {column.is_done_column && (
                    <span
                      title="Kolom Done dilindungi oleh aturan Maker-Checker (Hanya PM yang dapat memindahkan task masuk/keluar)"
                      className="text-[10px] font-semibold text-emerald-800 bg-emerald-50 border border-emerald-200 px-2 py-0.5 rounded-md flex items-center gap-1"
                    >
                      <svg className="w-3 h-3 text-emerald-600" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                      </svg>
                      Done
                    </span>
                  )}

                  {/* Quick Add Button */}
                  {canWrite && (
                    <button
                      type="button"
                      onClick={() => openCreateModalForColumn(column.id)}
                      title={`Tambah task di kolom ${column.name}`}
                      className="p-1 text-slate-400 hover:text-blue-600 hover:bg-white rounded-lg transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                      </svg>
                    </button>
                  )}
                </div>
              </div>

              {/* Tasks List */}
              <div className="flex-1 overflow-y-auto space-y-3 pr-1 min-h-[120px]">
                {colTasks.length === 0 ? (
                  <div
                    className={`h-28 rounded-2xl border-2 border-dashed flex items-center justify-center text-xs transition-colors ${
                      isDraggingOver
                        ? 'border-blue-400 bg-blue-50/50 text-blue-600 font-medium'
                        : 'border-slate-200 text-slate-400'
                    }`}
                  >
                    {isDraggingOver ? 'Lepaskan di sini' : 'Belum ada task'}
                  </div>
                ) : (
                  colTasks.map((t) => (
                    <TaskCard
                      key={t.id}
                      task={t}
                      columns={columns}
                      currentColumn={column}
                      canMove={canWrite}
                      isPM={isPM}
                      onTaskClick={handleCardClick}
                      onMoveTask={handleMoveTask}
                    />
                  ))
                )}
              </div>
            </div>
          )
        })}
      </div>

      {/* Create Task Modal */}
      {isCreateModalOpen && (
        <CreateTaskModal
          isOpen={isCreateModalOpen}
          onClose={() => setIsCreateModalOpen(false)}
          columns={columns}
          members={members}
          ownerName={ownerName}
          ownerId={ownerId}
          defaultColumnId={createModalColumnId}
          onSubmit={handleCreateTask}
        />
      )}

      {/* Detail Drawer */}
      {isDetailDrawerOpen && selectedTask && (
        <TaskDetailDrawer
          task={selectedTask}
          isOpen={isDetailDrawerOpen}
          onClose={() => {
            setIsDetailDrawerOpen(false)
            setSelectedTask(null)
          }}
          isPM={isPM}
          canEdit={canWrite}
          members={members}
          ownerName={ownerName}
          ownerId={ownerId}
          onUpdateTask={handleUpdateTask}
          onDeleteTask={isPM ? handleDeleteTask : undefined}
        />
      )}
    </div>
  )
}
