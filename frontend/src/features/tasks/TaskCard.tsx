import React, { useState, useMemo } from 'react'
import type { Task, TaskPriority } from './types'
import type { BoardColumn } from '../projects/types'
import type { Risk, RiskLevel } from '../risks/types'

interface TaskCardProps {
  task: Task
  columns: BoardColumn[]
  currentColumn: BoardColumn
  canMove: boolean
  isPM: boolean
  linkedRisks?: Risk[]
  onTaskClick: (task: Task) => void
  onMoveTask: (taskId: string, toColumnId: string, position: number, version: number) => Promise<void>
}

const PRIORITY_CONFIG: Record<
  TaskPriority,
  { label: string; badge: string; dot: string }
> = {
  low: {
    label: 'Rendah',
    badge: 'bg-slate-100 text-slate-700 border-slate-200',
    dot: 'bg-slate-400',
  },
  medium: {
    label: 'Sedang',
    badge: 'bg-blue-50 text-blue-700 border-blue-200',
    dot: 'bg-blue-500',
  },
  high: {
    label: 'Tinggi',
    badge: 'bg-amber-50 text-amber-700 border-amber-200',
    dot: 'bg-amber-500',
  },
  critical: {
    label: 'Kritis',
    badge: 'bg-rose-50 text-rose-700 border-rose-200',
    dot: 'bg-rose-500',
  },
}

export default function TaskCard({
  task,
  columns,
  currentColumn,
  canMove,
  isPM,
  linkedRisks = [],
  onTaskClick,
  onMoveTask,
}: TaskCardProps) {
  const [isMoveMenuOpen, setIsMoveMenuOpen] = useState(false)
  const [isMoving, setIsMoving] = useState(false)

  const priorityInfo = PRIORITY_CONFIG[task.priority] || PRIORITY_CONFIG.medium

  const maxRiskLevel = useMemo<RiskLevel | null>(() => {
    if (!linkedRisks || linkedRisks.length === 0) return null
    const levels = linkedRisks.map((r) => r.level).filter(Boolean) as RiskLevel[]
    if (levels.includes('critical')) return 'critical'
    if (levels.includes('high')) return 'high'
    if (levels.includes('medium')) return 'medium'
    if (levels.includes('low')) return 'low'
    return null
  }, [linkedRisks])

  const riskBadgeClass = useMemo(() => {
    switch (maxRiskLevel) {
      case 'critical':
        return 'bg-rose-50 text-rose-700 border-rose-200'
      case 'high':
        return 'bg-orange-50 text-orange-700 border-orange-200'
      case 'medium':
        return 'bg-amber-50 text-amber-700 border-amber-200'
      case 'low':
        return 'bg-emerald-50 text-emerald-700 border-emerald-200'
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200'
    }
  }, [maxRiskLevel])

  // Overdue check
  const todayStr = useMemo(() => new Date().toISOString().slice(0, 10), [])
  const isOverdue = Boolean(
    task.due_date &&
    !task.completed_at &&
    task.due_date < todayStr
  )

  // Maker-checker lock: if task is in Done column and user is not PM, it cannot be moved
  const isLockedByMakerChecker = currentColumn.is_done_column && !isPM

  const handleDragStart = (e: React.DragEvent) => {
    if (isLockedByMakerChecker || !canMove) {
      e.preventDefault()
      return
    }
    e.dataTransfer.setData('text/plain', JSON.stringify({ taskId: task.id, version: task.version }))
    e.dataTransfer.effectAllowed = 'move'
  }

  const handleQuickMove = async (toColumnId: string) => {
    setIsMoveMenuOpen(false)
    setIsMoving(true)
    try {
      await onMoveTask(task.id, toColumnId, 1, task.version)
    } finally {
      setIsMoving(false)
    }
  }

  return (
    <div
      draggable={canMove && !isLockedByMakerChecker}
      onDragStart={handleDragStart}
      onClick={() => onTaskClick(task)}
      className={`group relative bg-white rounded-xl p-4 border border-slate-200 shadow-xs hover:shadow-md transition-all cursor-pointer select-none ${
        isMoving ? 'opacity-50 pointer-events-none' : ''
      } ${isLockedByMakerChecker ? 'border-dashed border-slate-300' : 'hover:border-blue-400'}`}
    >
      {/* Top row: Priority & Quick Move Menu */}
      <div className="flex items-center justify-between gap-2 mb-2">
        <div className="flex items-center gap-1.5 flex-wrap">
          <span
            className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-md text-xs font-medium border ${priorityInfo.badge}`}
          >
            <span className={`w-1.5 h-1.5 rounded-full ${priorityInfo.dot}`} />
            {priorityInfo.label}
          </span>

          {/* Linked Risks Indicator Badge */}
          {linkedRisks.length > 0 && (
            <span
              title={`${linkedRisks.length} Risiko terkait (Level tertinggi: ${maxRiskLevel})`}
              className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md text-[11px] font-semibold border ${riskBadgeClass}`}
            >
              <svg className="w-3 h-3 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
              <span>{linkedRisks.length} Risiko</span>
            </span>
          )}
        </div>

        {/* Maker-checker Done Column Lock Badge */}
        {isLockedByMakerChecker && (
          <span
            title="Hanya Project Manager yang dapat memindahkan task dari status Done"
            className="inline-flex items-center gap-1 text-[11px] font-medium text-amber-700 bg-amber-50 px-2 py-0.5 rounded-md border border-amber-200"
          >
            <svg className="w-3 h-3 text-amber-600" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M5 9V7a5 5 0 0110 0v2a2 2 0 012 2v5a2 2 0 01-2 2H5a2 2 0 01-2-2v-5a2 2 0 012-2zm8-2v2H7V7a3 3 0 016 0z"
                clipRule="evenodd"
              />
            </svg>
            Terkunci (PM)
          </span>
        )}

        {/* Keyboard / Alternative Quick Move Dropdown */}
        {canMove && !isLockedByMakerChecker && (
          <div className="relative" onClick={(e) => e.stopPropagation()}>
            <button
              type="button"
              onClick={() => setIsMoveMenuOpen(!isMoveMenuOpen)}
              title="Pindahkan ke kolom lain"
              aria-label="Pindahkan task"
              className="p-1 rounded-md text-slate-400 hover:text-slate-600 hover:bg-slate-100 transition-colors"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
              </svg>
            </button>

            {isMoveMenuOpen && (
              <div className="absolute right-0 top-full mt-1 w-44 bg-white rounded-xl shadow-lg border border-slate-200 py-1.5 z-20 animate-fade-in">
                <div className="px-3 py-1 text-[10px] font-semibold tracking-wider text-slate-400 uppercase">
                  Pindahkan ke...
                </div>
                {columns
                  .filter((col) => col.id !== currentColumn.id)
                  .map((col) => {
                    const isDoneTarget = col.is_done_column
                    const disabled = isDoneTarget && !isPM
                    return (
                      <button
                        key={col.id}
                        type="button"
                        disabled={disabled}
                        onClick={() => handleQuickMove(col.id)}
                        className={`w-full text-left px-3 py-1.5 text-xs flex items-center justify-between ${
                          disabled
                            ? 'text-slate-300 cursor-not-allowed bg-slate-50'
                            : 'text-slate-700 hover:bg-slate-50 hover:text-blue-600'
                        }`}
                      >
                        <span className="truncate">{col.name}</span>
                        {isDoneTarget && !isPM && (
                          <span className="text-[10px] text-amber-600 font-medium">Khusus PM</span>
                        )}
                      </button>
                    )
                  })}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Task Title */}
      <h4 className="text-sm font-semibold text-slate-900 group-hover:text-blue-600 transition-colors line-clamp-2 mb-2">
        {task.title}
      </h4>

      {/* Description Preview (if present) */}
      {task.description && (
        <p className="text-xs text-slate-500 line-clamp-2 mb-3">
          {task.description}
        </p>
      )}

      {/* Progress Bar (if > 0) */}
      {task.progress_percent > 0 && (
        <div className="mb-3 space-y-1">
          <div className="flex items-center justify-between text-[11px] text-slate-500 font-medium">
            <span>Progress</span>
            <span>{task.progress_percent}%</span>
          </div>
          <div className="w-full bg-slate-100 rounded-full h-1.5 overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${
                task.progress_percent === 100 ? 'bg-emerald-500' : 'bg-blue-600'
              }`}
              style={{ width: `${task.progress_percent}%` }}
            />
          </div>
        </div>
      )}

      {/* Card Footer: Due Date & Assignee */}
      <div className="flex items-center justify-between gap-2 pt-2 border-t border-slate-100 text-xs text-slate-500">
        {/* Due Date */}
        {task.due_date ? (
          <div
            className={`inline-flex items-center gap-1 font-medium ${
              isOverdue ? 'text-red-600 font-semibold' : 'text-slate-500'
            }`}
            title={isOverdue ? 'Tenggat waktu telah terlewat!' : `Tenggat: ${task.due_date}`}
          >
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <span>{task.due_date}</span>
            {isOverdue && <span className="text-[10px] text-red-500">(Overdue)</span>}
          </div>
        ) : (
          <span className="text-slate-400 text-[11px]">Tanpa tenggat</span>
        )}

        {/* Assignee Avatar / Badge */}
        <div className="flex items-center gap-1.5">
          {task.assignee_name ? (
            <div
              title={`Ditugaskan kepada: ${task.assignee_name}`}
              className="w-6 h-6 rounded-full bg-indigo-100 text-indigo-700 font-bold text-[10px] flex items-center justify-center ring-1 ring-white"
            >
              {task.assignee_name.slice(0, 2).toUpperCase()}
            </div>
          ) : (
            <div
              title="Belum ditugaskan"
              className="w-6 h-6 rounded-full bg-slate-100 text-slate-400 text-[10px] flex items-center justify-center ring-1 ring-white"
            >
              ?
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
