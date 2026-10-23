import { useState, useEffect, useCallback } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import {
  fetchProject,
  fetchBoardColumns,
  fetchProjectMembers,
  archiveProject,
  deleteBoardColumn,
  removeProjectMember,
} from './api'
import type { Project, BoardColumn, ProjectMember, ProjectStatus } from './types'
import { useAuth } from '../auth/useAuth'
import EditProjectModal from './EditProjectModal'
import TransferOwnerModal from './TransferOwnerModal'
import AddMemberModal from './AddMemberModal'
import ColumnModal from './ColumnModal'
import KanbanBoard from '../tasks/KanbanBoard'

const STATUS_CONFIG: Record<
  ProjectStatus,
  { label: string; badge: string; dot: string }
> = {
  active: {
    label: 'Aktif',
    badge: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    dot: 'bg-emerald-500',
  },
  on_hold: {
    label: 'Ditangguhkan',
    badge: 'bg-amber-50 text-amber-700 border-amber-200',
    dot: 'bg-amber-500',
  },
  completed: {
    label: 'Selesai',
    badge: 'bg-blue-50 text-blue-700 border-blue-200',
    dot: 'bg-blue-500',
  },
  archived: {
    label: 'Diarsipkan',
    badge: 'bg-slate-100 text-slate-600 border-slate-300',
    dot: 'bg-slate-400',
  },
}

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()

  const [project, setProject] = useState<Project | null>(null)
  const [columns, setColumns] = useState<BoardColumn[]>([])
  const [members, setMembers] = useState<ProjectMember[]>([])
  const [activeTab, setActiveTab] = useState<'board' | 'columns' | 'members' | 'info'>('board')

  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [actionSuccess, setActionSuccess] = useState<string | null>(null)

  // Modals state
  const [isEditModalOpen, setIsEditModalOpen] = useState(false)
  const [isTransferModalOpen, setIsTransferModalOpen] = useState(false)
  const [isAddMemberModalOpen, setIsAddMemberModalOpen] = useState(false)
  const [isColumnModalOpen, setIsColumnModalOpen] = useState(false)
  const [columnToEdit, setColumnToEdit] = useState<BoardColumn | null>(null)

  const isOwner = user?.id === project?.owner_id
  const isPM = user?.role === 'project_manager'
  const isTL = user?.role === 'team_leader'
  const isMember = members.some((m) => m.id === user?.id)
  const isAdmin = user?.role === 'admin'
  const isArchived = project?.status === 'archived'

  const canManageProject = isPM && isOwner && !isArchived
  const canWriteTasks = !isArchived && ((isPM && isOwner) || (isTL && isMember))

  const loadData = useCallback(async () => {
    if (!id) return
    try {
      setIsLoading(true)
      setError(null)
      const [projData, colData, memData] = await Promise.all([
        fetchProject(id),
        fetchBoardColumns(id),
        fetchProjectMembers(id),
      ])
      setProject(projData)
      setColumns(colData.sort((a, b) => a.position - b.position))
      setMembers(memData)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Gagal memuat data proyek.')
      }
    } finally {
      setIsLoading(false)
    }
  }, [id])

  useEffect(() => {
    void loadData()
  }, [loadData])

  const handleArchive = async () => {
    if (!project) return
    if (
      !window.confirm(
        'Apakah Anda yakin ingin mengarsipkan proyek ini? Proyek yang diarsipkan akan menjadi read-only untuk seluruh pengguna.',
      )
    ) {
      return
    }

    try {
      setActionError(null)
      const updated = await archiveProject(project.id)
      setProject(updated)
      setActionSuccess('Proyek berhasil diarsipkan.')
    } catch (err: unknown) {
      if (err instanceof Error) {
        setActionError(err.message)
      } else {
        setActionError('Gagal mengarsipkan proyek.')
      }
    }
  }

  const handleDeleteColumn = async (col: BoardColumn) => {
    if (col.is_done_column) {
      alert('Kolom status Done dilindungi dan tidak dapat dihapus.')
      return
    }
    if (!window.confirm(`Hapus kolom "${col.name}"? Pastikan tidak ada task pada kolom ini.`)) {
      return
    }

    try {
      setActionError(null)
      await deleteBoardColumn(col.id)
      setColumns((prev) => prev.filter((c) => c.id !== col.id))
      setActionSuccess(`Kolom "${col.name}" berhasil dihapus.`)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setActionError(err.message)
      } else {
        setActionError('Gagal menghapus kolom.')
      }
    }
  }

  const handleRemoveMember = async (mem: ProjectMember) => {
    if (!project) return
    if (!window.confirm(`Hapus anggota "${mem.full_name}" dari proyek ini?`)) {
      return
    }

    try {
      setActionError(null)
      await removeProjectMember(project.id, mem.id)
      setMembers((prev) => prev.filter((m) => m.id !== mem.id))
      setActionSuccess(`Anggota "${mem.full_name}" berhasil dihapus.`)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setActionError(err.message)
      } else {
        setActionError('Gagal menghapus anggota tim.')
      }
    }
  }

  if (isLoading) {
    return (
      <div className="p-8 max-w-5xl mx-auto space-y-6 animate-pulse">
        <div className="h-4 w-32 bg-slate-200 rounded" />
        <div className="h-10 w-2/3 bg-slate-200 rounded-xl" />
        <div className="h-64 bg-slate-100 rounded-2xl" />
      </div>
    )
  }

  if (error || !project) {
    return (
      <div className="max-w-md mx-auto my-12 p-8 text-center bg-white rounded-2xl border border-red-100 shadow-sm space-y-4">
        <div className="w-12 h-12 rounded-full bg-red-50 text-red-500 flex items-center justify-center mx-auto">
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h2 className="text-lg font-bold text-slate-900">Proyek Tidak Ditemukan</h2>
        <p className="text-sm text-slate-500">
          {error || 'Proyek tidak ditemukan atau Anda tidak memiliki izin untuk melihatnya.'}
        </p>
        <button
          type="button"
          onClick={() => navigate('/projects')}
          className="px-4 py-2 text-sm font-semibold text-white bg-slate-900 hover:bg-slate-800 rounded-xl transition-all"
        >
          Kembali ke Daftar Proyek
        </button>
      </div>
    )
  }

  const statusInfo = STATUS_CONFIG[project.status] || STATUS_CONFIG.active

  return (
    <div className="space-y-6 max-w-6xl mx-auto animate-fade-in">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-2 text-xs font-medium text-slate-500">
        <Link to="/projects" className="hover:text-blue-600 transition-colors">
          Daftar Proyek
        </Link>
        <span>/</span>
        <span className="font-mono text-slate-800 font-semibold">{project.code}</span>
      </nav>

      {/* Notifications */}
      {actionError && (
        <div className="p-4 rounded-xl bg-red-50 border border-red-200 text-red-700 text-sm flex items-start justify-between gap-3">
          <div className="flex items-start gap-2.5">
            <svg className="w-5 h-5 text-red-500 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <span>{actionError}</span>
          </div>
          <button type="button" onClick={() => setActionError(null)} className="text-red-400 hover:text-red-600">
            &times;
          </button>
        </div>
      )}

      {actionSuccess && (
        <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 text-sm flex items-start justify-between gap-3">
          <div className="flex items-start gap-2.5">
            <svg className="w-5 h-5 text-emerald-600 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            <span>{actionSuccess}</span>
          </div>
          <button type="button" onClick={() => setActionSuccess(null)} className="text-emerald-500 hover:text-emerald-700">
            &times;
          </button>
        </div>
      )}

      {/* Top Banner Card */}
      <div className="bg-white rounded-3xl p-6 sm:p-8 shadow-sm border border-slate-200/80 flex flex-col md:flex-row md:items-start justify-between gap-6">
        <div className="space-y-3 max-w-2xl">
          <div className="flex flex-wrap items-center gap-2.5">
            <span className="font-mono text-xs font-bold px-3 py-1 rounded-lg bg-indigo-50 text-indigo-700 border border-indigo-100">
              {project.code}
            </span>
            <span
              className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold border ${statusInfo.badge}`}
            >
              <span className={`w-1.5 h-1.5 rounded-full ${statusInfo.dot}`} />
              {statusInfo.label}
            </span>
            {isArchived && (
              <span className="px-2.5 py-0.5 rounded-md bg-rose-50 text-rose-700 border border-rose-200 text-xs font-semibold">
                Read-Only
              </span>
            )}
          </div>

          <h1 className="text-2xl sm:text-3xl font-extrabold text-slate-900 tracking-tight">
            {project.name}
          </h1>

          <p className="text-sm text-slate-600 leading-relaxed">
            {project.description || 'Tidak ada catatan deskripsi.'}
          </p>

          <div className="flex flex-wrap items-center gap-4 text-xs text-slate-500 pt-1">
            <div className="flex items-center gap-1.5">
              <svg className="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span>Jadwal: {project.start_date || 'TBA'} &rarr; {project.end_date || 'TBA'}</span>
            </div>
            <div className="flex items-center gap-1.5">
              <svg className="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
              <span>Pemilik: {isOwner ? 'Anda (PM)' : project.owner_id.slice(0, 8) + '...'}</span>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-wrap items-center gap-2 shrink-0">
          {canManageProject && (
            <>
              <button
                type="button"
                onClick={() => setIsEditModalOpen(true)}
                className="px-4 py-2 text-xs font-semibold text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-xl transition-colors"
              >
                Edit Metadata
              </button>
              <button
                type="button"
                onClick={handleArchive}
                className="px-4 py-2 text-xs font-semibold text-amber-700 bg-amber-50 hover:bg-amber-100 border border-amber-200 rounded-xl transition-colors"
              >
                Arsipkan
              </button>
            </>
          )}

          {isAdmin && (
            <button
              type="button"
              onClick={() => setIsTransferModalOpen(true)}
              className="px-4 py-2 text-xs font-semibold text-purple-700 bg-purple-50 hover:bg-purple-100 border border-purple-200 rounded-xl transition-colors"
            >
              Transfer Pemilik
            </button>
          )}
        </div>
      </div>

      {/* Tabs Navigation */}
      <div className="border-b border-slate-200 flex items-center gap-6 overflow-x-auto">
        {[
          { id: 'board', label: 'Papan Kanban' },
          { id: 'columns', label: 'Pengaturan Kolom', count: columns.length },
          { id: 'members', label: 'Anggota Tim (Members)', count: members.length },
          { id: 'info', label: 'Informasi Lengkap' },
        ].map((tab) => (
          <button
            key={tab.id}
            type="button"
            onClick={() => setActiveTab(tab.id as 'board' | 'columns' | 'members' | 'info')}
            className={`pb-3 text-sm font-semibold transition-all relative flex items-center gap-2 whitespace-nowrap ${
              activeTab === tab.id
                ? 'text-blue-600 border-b-2 border-blue-600'
                : 'text-slate-500 hover:text-slate-800'
            }`}
          >
            <span>{tab.label}</span>
            {tab.count !== undefined && (
              <span
                className={`px-2 py-0.5 text-xs rounded-full ${
                  activeTab === tab.id
                    ? 'bg-blue-100 text-blue-700'
                    : 'bg-slate-100 text-slate-600'
                }`}
              >
                {tab.count}
              </span>
            )}
          </button>
        ))}
      </div>

      {/* Tab: Papan Kanban */}
      {activeTab === 'board' && (
        <KanbanBoard
          projectId={project.id}
          columns={columns}
          members={members}
          ownerName={isOwner ? (user?.full_name || 'Project Manager') : 'Project Manager'}
          ownerId={project.owner_id}
          isPM={isPM && isOwner && !isArchived}
          canWrite={canWriteTasks}
        />
      )}

      {/* Tab: Kolom Papan Kanban */}
      {activeTab === 'columns' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-base font-bold text-slate-900">Alur Kolom Papan</h3>
              <p className="text-xs text-slate-500 mt-0.5">
                Kolom disusun berdasarkan posisi kiri-ke-kanan dalam papan Kanban.
              </p>
            </div>

            {canManageProject && (
              <button
                type="button"
                onClick={() => {
                  setColumnToEdit(null)
                  setIsColumnModalOpen(true)
                }}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-white bg-blue-600 hover:bg-blue-700 rounded-xl transition-all shadow-sm"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Tambah Kolom
              </button>
            )}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {columns.map((col) => (
              <div
                key={col.id}
                className="bg-white rounded-2xl p-5 border border-slate-200/90 shadow-sm flex flex-col justify-between"
              >
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <span className="font-mono text-xs font-bold w-6 h-6 rounded-lg bg-slate-100 text-slate-700 flex items-center justify-center">
                      {col.position}
                    </span>

                    {col.is_done_column ? (
                      <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                        <svg className="w-3 h-3 text-emerald-600" fill="currentColor" viewBox="0 0 20 20">
                          <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                        </svg>
                        Kolom Done
                      </span>
                    ) : (
                      <span className="text-xs text-slate-500 font-medium">Standard</span>
                    )}
                  </div>

                  <h4 className="text-base font-bold text-slate-900">{col.name}</h4>

                  <div className="mt-3 flex items-center gap-2 text-xs">
                    <span className="text-slate-500">WIP Limit:</span>
                    {col.wip_limit ? (
                      <span className="font-semibold text-blue-700 bg-blue-50 px-2 py-0.5 rounded-md border border-blue-100">
                        Maks. {col.wip_limit} task
                      </span>
                    ) : (
                      <span className="text-slate-400 italic">Tanpa batas</span>
                    )}
                  </div>
                </div>

                {canManageProject && (
                  <div className="pt-4 mt-4 border-t border-slate-100 flex items-center justify-end gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        setColumnToEdit(col)
                        setIsColumnModalOpen(true)
                      }}
                      className="px-3 py-1.5 text-xs font-semibold text-slate-600 hover:text-slate-900 hover:bg-slate-100 rounded-lg transition-colors"
                    >
                      Edit
                    </button>

                    {!col.is_done_column && (
                      <button
                        type="button"
                        onClick={() => handleDeleteColumn(col)}
                        className="px-3 py-1.5 text-xs font-semibold text-red-600 hover:text-red-700 hover:bg-red-50 rounded-lg transition-colors"
                      >
                        Hapus
                      </button>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tab 2: Anggota Tim (Members) */}
      {activeTab === 'members' && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-base font-bold text-slate-900">Daftar Anggota Proyek</h3>
              <p className="text-xs text-slate-500 mt-0.5">
                Hanya user ber-role Team Leader yang dapat ditambahkan sebagai anggota tim.
              </p>
            </div>

            {canManageProject && (
              <button
                type="button"
                onClick={() => setIsAddMemberModalOpen(true)}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-white bg-emerald-600 hover:bg-emerald-700 rounded-xl transition-all shadow-sm"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Tambah Member
              </button>
            )}
          </div>

          {members.length === 0 ? (
            <div className="p-8 text-center bg-white rounded-2xl border border-slate-100 text-slate-500 text-sm">
              Belum ada anggota tim yang ditambahkan ke proyek ini.
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {members.map((m) => (
                <div
                  key={m.id}
                  className="bg-white rounded-2xl p-5 border border-slate-200/90 shadow-sm flex items-center justify-between gap-3"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-emerald-500 to-teal-400 text-white font-bold flex items-center justify-center text-sm shadow-sm">
                      {m.full_name
                        .split(' ')
                        .map((n) => n[0])
                        .slice(0, 2)
                        .join('')
                        .toUpperCase()}
                    </div>
                    <div>
                      <h4 className="text-sm font-bold text-slate-900">{m.full_name}</h4>
                      <p className="text-xs text-slate-500">{m.email}</p>
                      <span className="inline-block mt-1 px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                        Team Leader
                      </span>
                    </div>
                  </div>

                  {canManageProject && (
                    <button
                      type="button"
                      onClick={() => handleRemoveMember(m)}
                      title="Hapus member"
                      className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                      </svg>
                    </button>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Tab 3: Informasi Proyek */}
      {activeTab === 'info' && (
        <div className="bg-white rounded-2xl p-6 border border-slate-200/80 shadow-sm space-y-4">
          <h3 className="text-base font-bold text-slate-900">Audit & Metadata Sistem</h3>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-xs">
            <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
              <span className="text-slate-500 block mb-1">ID Proyek (UUID):</span>
              <span className="font-mono font-semibold text-slate-800 break-all">{project.id}</span>
            </div>
            <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
              <span className="text-slate-500 block mb-1">Owner ID (Project Manager):</span>
              <span className="font-mono font-semibold text-slate-800 break-all">{project.owner_id}</span>
            </div>
            <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
              <span className="text-slate-500 block mb-1">Created By:</span>
              <span className="font-mono font-semibold text-slate-800 break-all">{project.created_by}</span>
            </div>
            <div className="p-3.5 rounded-xl bg-slate-50 border border-slate-100">
              <span className="text-slate-500 block mb-1">Waktu Dibuat:</span>
              <span className="font-semibold text-slate-800">
                {new Date(project.created_at).toLocaleString('id-ID')}
              </span>
            </div>
          </div>
        </div>
      )}

      {/* Modals */}
      <EditProjectModal
        isOpen={isEditModalOpen}
        project={project}
        onClose={() => setIsEditModalOpen(false)}
        onSuccess={(updated) => {
          setProject(updated)
          setActionSuccess('Metadata proyek berhasil diperbarui.')
        }}
      />

      <TransferOwnerModal
        isOpen={isTransferModalOpen}
        project={project}
        onClose={() => setIsTransferModalOpen(false)}
        onSuccess={(updated) => {
          setProject(updated)
          setActionSuccess('Pemilik proyek berhasil dialihkan.')
        }}
      />

      <AddMemberModal
        isOpen={isAddMemberModalOpen}
        projectId={project.id}
        onClose={() => setIsAddMemberModalOpen(false)}
        onSuccess={() => {
          loadData()
          setActionSuccess('Anggota tim berhasil ditambahkan.')
        }}
      />

      <ColumnModal
        isOpen={isColumnModalOpen}
        projectId={project.id}
        columnToEdit={columnToEdit}
        nextPosition={columns.length + 1}
        onClose={() => setIsColumnModalOpen(false)}
        onSuccess={() => {
          loadData()
          setActionSuccess('Kolom papan berhasil diperbarui.')
        }}
      />
    </div>
  )
}
