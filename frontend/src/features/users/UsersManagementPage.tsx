import { useState, useEffect } from 'react'
import { apiListUsers, apiCreateUser, apiUpdateUser, apiDeleteUser } from './api'
import { useAuth } from '../auth/useAuth'
import type { User, UserRole } from '../../types'

const ROLE_CONFIG: Record<UserRole, { label: string; badge: string }> = {
  admin: { label: 'Administrator', badge: 'bg-purple-100 text-purple-800 border-purple-200' },
  project_manager: { label: 'Project Manager', badge: 'bg-blue-100 text-blue-800 border-blue-200' },
  team_leader: { label: 'Team Leader', badge: 'bg-emerald-100 text-emerald-800 border-emerald-200' },
  audit: { label: 'Team Audit', badge: 'bg-amber-100 text-amber-800 border-amber-200' },
  risk_management: { label: 'Team Risk', badge: 'bg-rose-100 text-rose-800 border-rose-200' },
}

export default function UsersManagementPage() {
  const { user: currentUser } = useAuth()
  const [users, setUsers] = useState<User[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  // Create Modal state
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [createFullName, setCreateFullName] = useState('')
  const [createRole, setCreateRole] = useState<UserRole>('project_manager')
  const [createPassword, setCreatePassword] = useState('')
  const [createFormError, setCreateFormError] = useState('')
  const [isSubmittingCreate, setIsSubmittingCreate] = useState(false)

  // Edit Modal state
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [editFullName, setEditFullName] = useState('')
  const [editRole, setEditRole] = useState<UserRole>('project_manager')
  const [editIsActive, setEditIsActive] = useState(true)
  const [editFormError, setEditFormError] = useState('')
  const [isSubmittingEdit, setIsSubmittingEdit] = useState(false)

  // Delete Confirm state
  const [deletingUser, setDeletingUser] = useState<User | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')

  const fetchUsers = async () => {
    try {
      setIsLoading(true)
      const data = await apiListUsers()
      setUsers(data)
    } catch {
      setError('Gagal memuat daftar pengguna.')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    void fetchUsers()
  }, [])

  const showNotification = (msg: string) => {
    setSuccessMessage(msg)
    setTimeout(() => setSuccessMessage(''), 5000)
  }

  // Create handler
  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateFormError('')

    if (!createFullName.trim()) {
      setCreateFormError('Nama lengkap pengguna wajib diisi.')
      return
    }
    if (createPassword.length < 12) {
      setCreateFormError('Password sementara minimal 12 karakter.')
      return
    }

    try {
      setIsSubmittingCreate(true)
      const newUser = await apiCreateUser({
        full_name: createFullName.trim(),
        role: createRole,
        password: createPassword,
      })
      showNotification(`Pengguna "${newUser.full_name}" berhasil ditambahkan sebagai ${ROLE_CONFIG[createRole].label}.`)
      setIsCreateOpen(false)
      setCreateFullName('')
      setCreatePassword('')
      setCreateRole('project_manager')
      void fetchUsers()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setCreateFormError(err.message)
      } else {
        setCreateFormError('Terjadi kesalahan saat menambahkan pengguna.')
      }
    } finally {
      setIsSubmittingCreate(false)
    }
  }

  // Open Edit Modal
  const openEditModal = (u: User) => {
    setEditingUser(u)
    setEditFullName(u.full_name)
    setEditRole(u.role)
    setEditIsActive(u.is_active ?? true)
    setEditFormError('')
  }

  // Edit handler
  const handleEditSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingUser) return
    setEditFormError('')

    if (!editFullName.trim()) {
      setEditFormError('Nama lengkap pengguna tidak boleh kosong.')
      return
    }

    try {
      setIsSubmittingEdit(true)
      const updated = await apiUpdateUser(editingUser.id, {
        full_name: editFullName.trim(),
        role: editRole,
        is_active: editIsActive,
      })
      showNotification(`Data pengguna "${updated.full_name}" berhasil diperbarui.`)
      setEditingUser(null)
      void fetchUsers()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setEditFormError(err.message)
      } else {
        setEditFormError('Gagal memperbarui data pengguna.')
      }
    } finally {
      setIsSubmittingEdit(false)
    }
  }

  // Delete handler
  const handleDeleteConfirm = async () => {
    if (!deletingUser) return
    setDeleteError('')

    try {
      setIsDeleting(true)
      await apiDeleteUser(deletingUser.id)
      showNotification(`Pengguna "${deletingUser.full_name}" berhasil dihapus.`)
      setDeletingUser(null)
      void fetchUsers()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setDeleteError(err.message)
      } else {
        setDeleteError('Gagal menghapus pengguna.')
      }
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text tracking-tight">Manajemen Pengguna</h1>
          <p className="text-sm text-text-muted mt-1">
            Kelola akun, peran (Role), serta status aktif pengguna pada sistem.
          </p>
        </div>
        <button
          onClick={() => {
            setCreateFormError('')
            setIsCreateOpen(true)
          }}
          className="inline-flex items-center gap-2 px-4 py-2.5 bg-primary text-white text-sm font-semibold rounded-lg shadow-sm hover:bg-primary-hover active:scale-[0.98] transition cursor-pointer self-start sm:self-auto"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Tambah Pengguna
        </button>
      </div>

      {successMessage && (
        <div className="p-4 rounded-lg bg-emerald-50 border border-emerald-200 text-sm text-emerald-800 flex items-center gap-2">
          <svg className="w-5 h-5 text-emerald-600 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          <span>{successMessage}</span>
        </div>
      )}

      {error && (
        <div className="p-4 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700">
          {error}
        </div>
      )}

      {/* Users Table */}
      <div className="bg-surface rounded-xl border border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-text">
            <thead className="bg-slate-50 border-b border-border text-xs uppercase font-semibold text-text-muted">
              <tr>
                <th className="px-6 py-3.5">Nama Pengguna</th>
                <th className="px-6 py-3.5">Peran (Role)</th>
                <th className="px-6 py-3.5">Status Akun</th>
                <th className="px-6 py-3.5">Password Sementara</th>
                <th className="px-6 py-3.5">Terdaftar</th>
                <th className="px-6 py-3.5 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-text-muted">
                    Memuat data pengguna…
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-6 py-8 text-center text-text-muted">
                    Belum ada data pengguna.
                  </td>
                </tr>
              ) : (
                users.map((u) => {
                  const roleMeta = ROLE_CONFIG[u.role] || { label: u.role, badge: 'bg-slate-100 text-slate-800' }
                  const isCurrent = currentUser?.id === u.id
                  return (
                    <tr key={u.id} className="hover:bg-slate-50/50 transition">
                      <td className="px-6 py-4 font-medium text-text">
                        <div className="flex items-center gap-3">
                          <div className="w-9 h-9 rounded-full bg-primary-soft text-primary font-bold flex items-center justify-center text-xs">
                            {u.full_name
                              .split(' ')
                              .filter(Boolean)
                              .slice(0, 2)
                              .map((n) => n[0].toUpperCase())
                              .join('')}
                          </div>
                          <div>
                            <div className="font-semibold flex items-center gap-2">
                              {u.full_name}
                              {isCurrent && (
                                <span className="text-[10px] bg-slate-100 text-slate-600 px-1.5 py-0.5 rounded font-normal">
                                  Anda
                                </span>
                              )}
                            </div>
                            <div className="text-xs text-text-muted">{u.email}</div>
                          </div>
                        </div>
                      </td>
                      <td className="px-6 py-4">
                        <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-semibold border ${roleMeta.badge}`}>
                          {roleMeta.label}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        {u.is_active !== false ? (
                          <span className="inline-flex items-center gap-1.5 text-xs text-emerald-700 font-medium">
                            <span className="w-2 h-2 rounded-full bg-emerald-500" />
                            Aktif
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 text-xs text-red-600 font-medium">
                            <span className="w-2 h-2 rounded-full bg-red-400" />
                            Nonaktif
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-xs">
                        {u.must_change_password ? (
                          <span className="text-amber-700 bg-amber-50 px-2 py-0.5 rounded border border-amber-200">
                            Wajib Ganti
                          </span>
                        ) : (
                          <span className="text-text-muted">Aman</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-xs text-text-muted">
                        {u.created_at ? new Date(u.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }) : '-'}
                      </td>
                      <td className="px-6 py-4 text-right">
                        <div className="flex items-center justify-end gap-2">
                          <button
                            onClick={() => openEditModal(u)}
                            className="px-2.5 py-1 text-xs font-medium text-primary hover:bg-primary-soft rounded transition cursor-pointer"
                          >
                            Edit
                          </button>
                          {!isCurrent && (
                            <button
                              onClick={() => {
                                setDeleteError('')
                                setDeletingUser(u)
                              }}
                              className="px-2.5 py-1 text-xs font-medium text-red-600 hover:bg-red-50 rounded transition cursor-pointer"
                            >
                              Hapus
                            </button>
                          )}
                        </div>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Modal Tambah User */}
      {isCreateOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-surface rounded-xl border border-border shadow-xl w-full max-w-md p-6 space-y-5 animate-in fade-in zoom-in duration-150">
            <div className="flex items-center justify-between pb-3 border-b border-border">
              <h2 className="text-lg font-bold text-text">Tambah Pengguna Baru</h2>
              <button
                onClick={() => setIsCreateOpen(false)}
                className="text-text-muted hover:text-text cursor-pointer"
              >
                ✕
              </button>
            </div>

            {createFormError && (
              <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-xs text-red-700">
                {createFormError}
              </div>
            )}

            <form onSubmit={handleCreateSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Nama Lengkap <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="Contoh: Budi Prasetyo"
                  value={createFullName}
                  onChange={(e) => setCreateFullName(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Role / Peran <span className="text-red-500">*</span>
                </label>
                <select
                  value={createRole}
                  onChange={(e) => setCreateRole(e.target.value as UserRole)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary bg-surface"
                >
                  <option value="project_manager">Project Manager</option>
                  <option value="team_leader">Team Leader</option>
                  <option value="risk_management">Team Risk</option>
                  <option value="audit">Team Audit</option>
                </select>
                <p className="text-[11px] text-text-muted mt-1">
                  Pilih hak akses operasional pengguna pada sistem perbankan.
                </p>
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Password Sementara (Min 12 Karakter) <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="Contoh: BankPPO#2026Secure"
                  value={createPassword}
                  onChange={(e) => setCreatePassword(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary font-mono"
                />
                <p className="text-[11px] text-text-muted mt-1">
                  Pengguna akan dipaksa mengganti password ini saat login pertama kali.
                </p>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-border">
                <button
                  type="button"
                  onClick={() => setIsCreateOpen(false)}
                  className="px-4 py-2 text-sm font-medium text-text-muted hover:text-text cursor-pointer"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={isSubmittingCreate}
                  className="px-4 py-2 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary-hover disabled:opacity-50 transition cursor-pointer"
                >
                  {isSubmittingCreate ? 'Menyimpan…' : 'Simpan Pengguna'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal Edit User */}
      {editingUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-surface rounded-xl border border-border shadow-xl w-full max-w-md p-6 space-y-5 animate-in fade-in zoom-in duration-150">
            <div className="flex items-center justify-between pb-3 border-b border-border">
              <h2 className="text-lg font-bold text-text">Edit Data Pengguna</h2>
              <button
                onClick={() => setEditingUser(null)}
                className="text-text-muted hover:text-text cursor-pointer"
              >
                ✕
              </button>
            </div>

            {editFormError && (
              <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-xs text-red-700">
                {editFormError}
              </div>
            )}

            <form onSubmit={handleEditSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Email Internal (ID)
                </label>
                <input
                  type="text"
                  disabled
                  value={editingUser.email}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border bg-slate-50 text-text-muted cursor-not-allowed"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Nama Lengkap <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  value={editFullName}
                  onChange={(e) => setEditFullName(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Role / Peran <span className="text-red-500">*</span>
                </label>
                <select
                  value={editRole}
                  onChange={(e) => setEditRole(e.target.value as UserRole)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary bg-surface"
                >
                  <option value="project_manager">Project Manager</option>
                  <option value="team_leader">Team Leader</option>
                  <option value="risk_management">Team Risk</option>
                  <option value="audit">Team Audit</option>
                  {editingUser.role === 'admin' && (
                    <option value="admin">Administrator</option>
                  )}
                </select>
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Status Pengguna
                </label>
                <div className="flex items-center gap-3 pt-1">
                  <label className="inline-flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="radio"
                      name="status"
                      checked={editIsActive}
                      onChange={() => setEditIsActive(true)}
                      className="text-primary focus:ring-primary"
                    />
                    <span className="text-text font-medium">Aktif</span>
                  </label>
                  <label className="inline-flex items-center gap-2 text-sm cursor-pointer">
                    <input
                      type="radio"
                      name="status"
                      checked={!editIsActive}
                      onChange={() => setEditIsActive(false)}
                      className="text-primary focus:ring-primary"
                    />
                    <span className="text-text font-medium">Nonaktif (Suspend)</span>
                  </label>
                </div>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-border">
                <button
                  type="button"
                  onClick={() => setEditingUser(null)}
                  className="px-4 py-2 text-sm font-medium text-text-muted hover:text-text cursor-pointer"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={isSubmittingEdit}
                  className="px-4 py-2 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary-hover disabled:opacity-50 transition cursor-pointer"
                >
                  {isSubmittingEdit ? 'Menyimpan…' : 'Perbarui Pengguna'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal Konfirmasi Hapus */}
      {deletingUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-surface rounded-xl border border-border shadow-xl w-full max-w-sm p-6 space-y-4 animate-in fade-in zoom-in duration-150">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-red-100 text-red-600 flex items-center justify-center shrink-0">
                <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </div>
              <div>
                <h3 className="text-base font-bold text-text">Hapus Pengguna</h3>
                <p className="text-xs text-text-muted mt-0.5">Tindakan ini tidak dapat dibatalkan.</p>
              </div>
            </div>

            {deleteError && (
              <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-xs text-red-700">
                {deleteError}
              </div>
            )}

            <p className="text-sm text-text">
              Apakah Anda yakin ingin menghapus pengguna <span className="font-semibold text-text">{deletingUser.full_name}</span>?
            </p>

            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                onClick={() => setDeletingUser(null)}
                className="px-4 py-2 text-sm font-medium text-text-muted hover:text-text cursor-pointer"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={isDeleting}
                onClick={handleDeleteConfirm}
                className="px-4 py-2 bg-red-600 text-white text-sm font-semibold rounded-lg hover:bg-red-700 disabled:opacity-50 transition cursor-pointer"
              >
                {isDeleting ? 'Menghapus…' : 'Ya, Hapus'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
