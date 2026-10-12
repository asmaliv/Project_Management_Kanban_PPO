import { useState, useEffect, useMemo, useCallback } from 'react'
import {
  apiListUsers,
  apiCreateUser,
  apiUpdateUser,
  apiSetUserActiveStatus,
  apiResetPassword,
  ApiClientError,
} from '../../lib/api'
import type { UserDetail, UserRole, CreateUserPayload } from '../../types'
import { useAuth } from '../auth/useAuth'

const ROLES: { id: UserRole; name: string; badge: string }[] = [
  { id: 'admin', name: 'Administrator', badge: 'bg-purple-100 text-purple-800 border-purple-200' },
  { id: 'project_manager', name: 'Project Manager', badge: 'bg-blue-100 text-blue-800 border-blue-200' },
  { id: 'team_leader', name: 'Team Leader', badge: 'bg-emerald-100 text-emerald-800 border-emerald-200' },
  { id: 'audit', name: 'Audit', badge: 'bg-amber-100 text-amber-800 border-amber-200' },
  { id: 'risk_management', name: 'Risk Management', badge: 'bg-rose-100 text-rose-800 border-rose-200' },
]

export default function UsersPage() {
  const { user: currentUser } = useAuth()
  const [users, setUsers] = useState<UserDetail[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [toastMessage, setToastMessage] = useState('')

  // Search & Filters
  const [searchQuery, setSearchQuery] = useState('')
  const [roleFilter, setRoleFilter] = useState<string>('all')
  const [statusFilter, setStatusFilter] = useState<string>('all')

  // Modals state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<UserDetail | null>(null)
  const [resettingUser, setResettingUser] = useState<UserDetail | null>(null)
  const [confirmToggleUser, setConfirmToggleUser] = useState<UserDetail | null>(null)

  const [now] = useState(() => Date.now())

  const showToast = (msg: string) => {
    setToastMessage(msg)
    setTimeout(() => setToastMessage(''), 4000)
  }

  const [reloadKey, setReloadKey] = useState(0)

  const reloadUsers = useCallback(() => {
    setIsLoading(true)
    setReloadKey((prev) => prev + 1)
  }, [])

  useEffect(() => {
    let active = true
    const run = async () => {
      try {
        const res = await apiListUsers(1, 100)
        if (active) {
          setUsers(res.users)
          setError('')
        }
      } catch (err: unknown) {
        if (active) {
          setError(err instanceof ApiClientError ? err.message : 'Gagal memuat daftar pengguna.')
        }
      } finally {
        if (active) {
          setIsLoading(false)
        }
      }
    }
    void run()
    return () => {
      active = false
    }
  }, [reloadKey])

  const filteredUsers = useMemo(() => {
    return users.filter((u) => {
      const matchesSearch =
        u.full_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        u.email.toLowerCase().includes(searchQuery.toLowerCase())
      const matchesRole = roleFilter === 'all' || u.role === roleFilter
      const matchesStatus =
        statusFilter === 'all' ||
        (statusFilter === 'active' && u.is_active) ||
        (statusFilter === 'inactive' && !u.is_active)
      return matchesSearch && matchesRole && matchesStatus
    })
  }, [users, searchQuery, roleFilter, statusFilter])

  const stats = useMemo(() => {
    const total = users.length
    const active = users.filter((u) => u.is_active).length
    const mustChange = users.filter((u) => u.must_change_password).length
    const locked = users.filter((u) => u.locked_until && Date.parse(u.locked_until) > now).length
    return { total, active, mustChange, locked }
  }, [users, now])

  const getRoleBadge = (role: UserRole) => {
    const r = ROLES.find((x) => x.id === role)
    return r ? r.badge : 'bg-slate-100 text-slate-800 border-slate-200'
  }

  const getRoleName = (role: UserRole) => {
    const r = ROLES.find((x) => x.id === role)
    return r ? r.name : role
  }

  return (
    <div className="space-y-6">
      {/* Toast Notification */}
      {toastMessage && (
        <div
          role="status"
          className="fixed bottom-5 right-5 z-50 bg-slate-900 text-white px-4 py-3 rounded-lg shadow-xl text-xs flex items-center gap-2 border border-slate-700 animate-fade-in"
        >
          <svg className="w-4 h-4 text-emerald-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
          </svg>
          <span>{toastMessage}</span>
        </div>
      )}

      {/* Header Banner */}
      <div className="bg-surface p-6 rounded-xl border border-border shadow-sm flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-text">Manajemen Pengguna & RBAC</h1>
          <p className="text-sm text-text-muted mt-1">
            Kelola akun internal, pembagian peran hak akses (5 role perbankan), serta status keamanan akun.
          </p>
        </div>
        <button
          id="btn-add-user"
          onClick={() => setIsCreateModalOpen(true)}
          className="flex items-center justify-center gap-2 px-4 py-2 bg-primary text-white text-xs font-semibold rounded-lg hover:bg-primary-hover transition-colors shadow-sm cursor-pointer"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          <span>Tambah Pengguna</span>
        </button>
      </div>

      {/* Metric Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <p className="text-xs text-text-muted font-medium">Total Akun</p>
          <p className="text-2xl font-bold text-text tabular-nums mt-1">{stats.total}</p>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <p className="text-xs text-text-muted font-medium">Akun Aktif</p>
          <p className="text-2xl font-bold text-success tabular-nums mt-1">{stats.active}</p>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <p className="text-xs text-text-muted font-medium">Perlu Ganti Password</p>
          <p className="text-2xl font-bold text-warning tabular-nums mt-1">{stats.mustChange}</p>
        </div>
        <div className="bg-surface p-4 rounded-xl border border-border shadow-sm">
          <p className="text-xs text-text-muted font-medium">Akun Terkunci</p>
          <p className="text-2xl font-bold text-danger tabular-nums mt-1">{stats.locked}</p>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="bg-surface p-4 rounded-xl border border-border shadow-sm flex flex-col md:flex-row items-center justify-between gap-3">
        <div className="relative w-full md:w-80">
          <svg
            className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            id="user-search-input"
            type="text"
            placeholder="Cari nama atau email…"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full pl-9 pr-3.5 py-1.5 text-xs rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-focus-ring bg-background text-text"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2.5 w-full md:w-auto">
          <select
            id="user-role-filter"
            value={roleFilter}
            onChange={(e) => setRoleFilter(e.target.value)}
            className="text-xs px-3 py-1.5 rounded-lg border border-border bg-background text-text focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="all">Semua Peran</option>
            {ROLES.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>

          <select
            id="user-status-filter"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="text-xs px-3 py-1.5 rounded-lg border border-border bg-background text-text focus:outline-none focus:ring-2 focus:ring-focus-ring cursor-pointer"
          >
            <option value="all">Semua Status</option>
            <option value="active">Aktif Saja</option>
            <option value="inactive">Nonaktif Saja</option>
          </select>

          <button
            onClick={() => void reloadUsers()}
            title="Muat ulang data"
            className="p-1.5 text-text-muted hover:text-text rounded-lg border border-border hover:bg-slate-50 transition-colors cursor-pointer"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Error state */}
      {error && (
        <div className="p-4 rounded-xl bg-red-50 border border-red-200 text-xs text-red-700 flex items-center justify-between">
          <span>{error}</span>
          <button onClick={() => void reloadUsers()} className="underline font-semibold cursor-pointer">
            Coba lagi
          </button>
        </div>
      )}

      {/* Users Table */}
      <div className="bg-surface rounded-xl border border-border shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead>
              <tr className="bg-slate-50/75 border-b border-border text-text-muted font-semibold">
                <th className="py-3 px-4">Pengguna</th>
                <th className="py-3 px-4">Peran (Role)</th>
                <th className="py-3 px-4">Status Akun</th>
                <th className="py-3 px-4">Keamanan</th>
                <th className="py-3 px-4">Terdaftar</th>
                <th className="py-3 px-4 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="py-12 text-center text-text-muted">
                    <div className="inline-block w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin mb-2" />
                    <p>Memuat data pengguna…</p>
                  </td>
                </tr>
              ) : filteredUsers.length === 0 ? (
                <tr>
                  <td colSpan={6} className="py-12 text-center text-text-muted">
                    Tidak ada pengguna yang sesuai dengan kriteria filter.
                  </td>
                </tr>
              ) : (
                filteredUsers.map((u) => {
                  const isCurrent = currentUser?.id === u.id
                  const isLocked = Boolean(u.locked_until && Date.parse(u.locked_until) > now)
                  return (
                    <tr key={u.id} className="hover:bg-slate-50/50 transition-colors">
                      {/* Name & Email */}
                      <td className="py-3.5 px-4">
                        <div className="flex items-center gap-3">
                          <div className="w-8 h-8 rounded-full bg-slate-100 border border-slate-300 font-bold text-[11px] text-slate-700 flex items-center justify-center shrink-0">
                            {u.full_name
                              .split(' ')
                              .filter(Boolean)
                              .slice(0, 2)
                              .map((n) => n[0].toUpperCase())
                              .join('')}
                          </div>
                          <div>
                            <div className="font-semibold text-text flex items-center gap-1.5">
                              <span>{u.full_name}</span>
                              {isCurrent && (
                                <span className="text-[10px] bg-slate-100 text-slate-600 px-1.5 py-0.2 rounded border border-slate-200">
                                  Anda
                                </span>
                              )}
                            </div>
                            <div className="text-[11px] text-text-muted font-mono">{u.email}</div>
                          </div>
                        </div>
                      </td>

                      {/* Role */}
                      <td className="py-3.5 px-4">
                        <span
                          className={`inline-block px-2.5 py-0.5 rounded-full text-[11px] font-medium border ${getRoleBadge(
                            u.role,
                          )}`}
                        >
                          {getRoleName(u.role)}
                        </span>
                      </td>

                      {/* Status */}
                      <td className="py-3.5 px-4">
                        {u.is_active ? (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
                            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" />
                            Aktif
                          </span>
                        ) : (
                          <span className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-rose-50 text-rose-700 border border-rose-200">
                            <span className="w-1.5 h-1.5 rounded-full bg-rose-500" />
                            Nonaktif
                          </span>
                        )}
                      </td>

                      {/* Security State */}
                      <td className="py-3.5 px-4">
                        <div className="space-y-1">
                          {isLocked && (
                            <span className="inline-block text-[10px] text-red-700 bg-red-100 border border-red-200 px-1.5 py-0.2 rounded font-medium">
                              Terkunci sementara
                            </span>
                          )}
                          {u.must_change_password ? (
                            <span className="inline-block text-[10px] text-amber-700 bg-amber-100 border border-amber-200 px-1.5 py-0.2 rounded font-medium">
                              Wajib ganti password
                            </span>
                          ) : (
                            <span className="text-[11px] text-text-muted">Password aktif</span>
                          )}
                        </div>
                      </td>

                      {/* Created At */}
                      <td className="py-3.5 px-4 text-text-muted text-[11px]">
                        {new Date(u.created_at).toLocaleDateString('id-ID', {
                          day: 'numeric',
                          month: 'short',
                          year: 'numeric',
                        })}
                      </td>

                      {/* Actions */}
                      <td className="py-3.5 px-4 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          <button
                            onClick={() => setEditingUser(u)}
                            className="px-2.5 py-1 text-slate-700 hover:text-primary hover:bg-slate-100 rounded border border-border text-[11px] font-medium transition-colors cursor-pointer"
                          >
                            Edit
                          </button>
                          <button
                            onClick={() => setResettingUser(u)}
                            className="px-2.5 py-1 text-slate-700 hover:text-amber-700 hover:bg-amber-50 rounded border border-border text-[11px] font-medium transition-colors cursor-pointer"
                          >
                            Reset
                          </button>
                          <button
                            onClick={() => setConfirmToggleUser(u)}
                            disabled={isCurrent}
                            title={isCurrent ? 'Anda tidak dapat menonaktifkan akun sendiri' : undefined}
                            className={`px-2.5 py-1 rounded border text-[11px] font-medium transition-colors ${
                              isCurrent
                                ? 'opacity-40 cursor-not-allowed border-slate-200 text-slate-400'
                                : u.is_active
                                ? 'text-danger hover:bg-red-50 border-border hover:border-red-200 cursor-pointer'
                                : 'text-emerald-700 hover:bg-emerald-50 border-border hover:border-emerald-200 cursor-pointer'
                            }`}
                          >
                            {u.is_active ? 'Nonaktifkan' : 'Aktifkan'}
                          </button>
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

      {/* Modal Tambah Pengguna */}
      {isCreateModalOpen && (
        <CreateUserModal
          onClose={() => setIsCreateModalOpen(false)}
          onSuccess={() => {
            setIsCreateModalOpen(false)
            showToast('Pengguna baru berhasil ditambahkan.')
            reloadUsers()
          }}
        />
      )}

      {/* Modal Edit Pengguna */}
      {editingUser && (
        <EditUserModal
          user={editingUser}
          isSelf={currentUser?.id === editingUser.id}
          onClose={() => setEditingUser(null)}
          onSuccess={() => {
            setEditingUser(null)
            showToast('Data pengguna berhasil diperbarui.')
            reloadUsers()
          }}
        />
      )}

      {/* Modal Reset Password */}
      {resettingUser && (
        <ResetPasswordModal
          user={resettingUser}
          onClose={() => setResettingUser(null)}
          onSuccess={() => {
            setResettingUser(null)
            showToast('Password pengguna berhasil di-reset.')
            reloadUsers()
          }}
        />
      )}

      {/* Modal Konfirmasi Aktif/Nonaktif */}
      {confirmToggleUser && (
        <ToggleStatusModal
          user={confirmToggleUser}
          onClose={() => setConfirmToggleUser(null)}
          onSuccess={() => {
            setConfirmToggleUser(null)
            showToast(
              confirmToggleUser.is_active
                ? 'Pengguna telah dinonaktifkan dan seluruh sesinya dicabut.'
                : 'Pengguna telah diaktifkan kembali.',
            )
            reloadUsers()
          }}
        />
      )}
    </div>
  )
}

// ─── Modal Tambah Pengguna ───────────────────────────────────────────

function CreateUserModal({
  onClose,
  onSuccess,
}: {
  onClose: () => void
  onSuccess: () => void
}) {
  const [email, setEmail] = useState('')
  const [fullName, setFullName] = useState('')
  const [role, setRole] = useState<UserRole>('team_leader')
  const [tempPassword, setTempPassword] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const generatePassword = () => {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*()'
    let pass = 'PPO#'
    for (let i = 0; i < 12; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    setTempPassword(pass)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      const payload: CreateUserPayload = {
        email,
        full_name: fullName,
        role,
        temporary_password: tempPassword,
      }
      await apiCreateUser(payload)
      onSuccess()
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setError(err.message)
      } else {
        setError('Gagal membuat pengguna.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div className="bg-surface w-full max-w-md rounded-xl border border-border shadow-2xl p-6">
        <div className="flex items-center justify-between mb-4 border-b border-border pb-3">
          <h2 className="text-base font-bold text-text">Tambah Pengguna Baru</h2>
          <button onClick={onClose} className="text-text-muted hover:text-text cursor-pointer">
            ✕
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-xs text-red-700 rounded-lg">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4 text-xs">
          <div>
            <label className="block font-medium text-text mb-1">Email Perusahaan</label>
            <input
              id="create-user-email"
              type="email"
              required
              placeholder="nama@bank.co.id"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none"
            />
          </div>

          <div>
            <label className="block font-medium text-text mb-1">Nama Lengkap</label>
            <input
              id="create-user-fullname"
              type="text"
              required
              placeholder="e.g. Budi Santoso"
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none"
            />
          </div>

          <div>
            <label className="block font-medium text-text mb-1">Peran (Role RBAC)</label>
            <select
              id="create-user-role"
              value={role}
              onChange={(e) => setRole(e.target.value as UserRole)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none cursor-pointer"
            >
              {ROLES.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="font-medium text-text">Password Sementara</label>
              <button
                type="button"
                onClick={generatePassword}
                className="text-[11px] text-primary hover:underline cursor-pointer"
              >
                Buat Password Acak
              </button>
            </div>
            <input
              id="create-user-temp-password"
              type="text"
              required
              placeholder="Minimal 12 karakter kombinasi"
              value={tempPassword}
              onChange={(e) => setTempPassword(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none font-mono"
            />
            <p className="text-[10px] text-text-muted mt-1">
              Pengguna akan diwajibkan mengganti password ini pada login pertama.
            </p>
          </div>

          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-3.5 py-2 rounded-lg border border-border text-slate-700 hover:bg-slate-50 cursor-pointer"
            >
              Batal
            </button>
            <button
              id="create-user-submit"
              type="submit"
              disabled={isSubmitting || !email || !fullName || !tempPassword}
              className="px-4 py-2 bg-primary text-white font-semibold rounded-lg hover:bg-primary-hover disabled:opacity-50 cursor-pointer"
            >
              {isSubmitting ? 'Menyimpan…' : 'Simpan Pengguna'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ─── Modal Edit Pengguna ─────────────────────────────────────────────

function EditUserModal({
  user,
  isSelf,
  onClose,
  onSuccess,
}: {
  user: UserDetail
  isSelf: boolean
  onClose: () => void
  onSuccess: () => void
}) {
  const [fullName, setFullName] = useState(user.full_name)
  const [role, setRole] = useState<UserRole>(user.role)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await apiUpdateUser(user.id, { full_name: fullName, role })
      onSuccess()
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setError(err.message)
      } else {
        setError('Gagal memperbarui pengguna.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div className="bg-surface w-full max-w-md rounded-xl border border-border shadow-2xl p-6">
        <div className="flex items-center justify-between mb-4 border-b border-border pb-3">
          <h2 className="text-base font-bold text-text">Edit Pengguna</h2>
          <button onClick={onClose} className="text-text-muted hover:text-text cursor-pointer">
            ✕
          </button>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-xs text-red-700 rounded-lg">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4 text-xs">
          <div>
            <label className="block font-medium text-text-muted mb-1">Email</label>
            <input
              type="email"
              disabled
              value={user.email}
              className="w-full px-3 py-2 rounded-lg border border-border bg-slate-100 text-slate-500 cursor-not-allowed"
            />
          </div>

          <div>
            <label className="block font-medium text-text mb-1">Nama Lengkap</label>
            <input
              id="edit-user-fullname"
              type="text"
              required
              value={fullName}
              onChange={(e) => setFullName(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none"
            />
          </div>

          <div>
            <label className="block font-medium text-text mb-1">Peran (Role RBAC)</label>
            <select
              id="edit-user-role"
              value={role}
              disabled={isSelf}
              onChange={(e) => setRole(e.target.value as UserRole)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none cursor-pointer disabled:bg-slate-100 disabled:cursor-not-allowed"
            >
              {ROLES.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name}
                </option>
              ))}
            </select>
            {isSelf && (
              <p className="text-[10px] text-text-muted mt-1">
                Anda tidak dapat menurunkan atau mengubah peran akun Anda sendiri.
              </p>
            )}
          </div>

          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-3.5 py-2 rounded-lg border border-border text-slate-700 hover:bg-slate-50 cursor-pointer"
            >
              Batal
            </button>
            <button
              id="edit-user-submit"
              type="submit"
              disabled={isSubmitting || !fullName}
              className="px-4 py-2 bg-primary text-white font-semibold rounded-lg hover:bg-primary-hover disabled:opacity-50 cursor-pointer"
            >
              {isSubmitting ? 'Menyimpan…' : 'Simpan Perubahan'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ─── Modal Reset Password ────────────────────────────────────────────

function ResetPasswordModal({
  user,
  onClose,
  onSuccess,
}: {
  user: UserDetail
  onClose: () => void
  onSuccess: () => void
}) {
  const [tempPassword, setTempPassword] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const generatePassword = () => {
    const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*()'
    let pass = 'PPO#'
    for (let i = 0; i < 12; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    setTempPassword(pass)
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await apiResetPassword(user.id, tempPassword)
      onSuccess()
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setError(err.message)
      } else {
        setError('Gagal me-reset password pengguna.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div className="bg-surface w-full max-w-md rounded-xl border border-border shadow-2xl p-6">
        <div className="flex items-center justify-between mb-4 border-b border-border pb-3">
          <h2 className="text-base font-bold text-text">Reset Password Pengguna</h2>
          <button onClick={onClose} className="text-text-muted hover:text-text cursor-pointer">
            ✕
          </button>
        </div>

        <p className="text-xs text-text-muted mb-4">
          Reset password untuk <span className="font-semibold text-text">{user.full_name}</span> (
          {user.email}). Seluruh sesi aktif pengguna ini akan langsung dicabut demi keamanan.
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-xs text-red-700 rounded-lg">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4 text-xs">
          <div>
            <div className="flex items-center justify-between mb-1">
              <label className="font-medium text-text">Password Sementara Baru</label>
              <button
                type="button"
                onClick={generatePassword}
                className="text-[11px] text-primary hover:underline cursor-pointer"
              >
                Buat Password Acak
              </button>
            </div>
            <input
              id="reset-temp-password-input"
              type="text"
              required
              placeholder="Minimal 12 karakter kombinasi"
              value={tempPassword}
              onChange={(e) => setTempPassword(e.target.value)}
              className="w-full px-3 py-2 rounded-lg border border-border focus:ring-2 focus:ring-focus-ring focus:outline-none font-mono"
            />
          </div>

          <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-border">
            <button
              type="button"
              onClick={onClose}
              disabled={isSubmitting}
              className="px-3.5 py-2 rounded-lg border border-border text-slate-700 hover:bg-slate-50 cursor-pointer"
            >
              Batal
            </button>
            <button
              id="reset-password-submit"
              type="submit"
              disabled={isSubmitting || !tempPassword}
              className="px-4 py-2 bg-amber-600 text-white font-semibold rounded-lg hover:bg-amber-700 disabled:opacity-50 cursor-pointer"
            >
              {isSubmitting ? 'Memproses…' : 'Reset Password'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ─── Modal Konfirmasi Status Aktif/Nonaktif ──────────────────────────

function ToggleStatusModal({
  user,
  onClose,
  onSuccess,
}: {
  user: UserDetail
  onClose: () => void
  onSuccess: () => void
}) {
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  const handleToggle = async () => {
    setError('')
    setIsSubmitting(true)
    try {
      if (user.is_active) {
        await apiSetUserActiveStatus(user.id, false)
      } else {
        await apiSetUserActiveStatus(user.id, true)
      }
      onSuccess()
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setError(err.message)
      } else {
        setError('Gagal mengubah status pengguna.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/50 backdrop-blur-xs p-4">
      <div className="bg-surface w-full max-w-sm rounded-xl border border-border shadow-2xl p-6">
        <h2 className="text-base font-bold text-text mb-2">
          {user.is_active ? 'Nonaktifkan Akun?' : 'Aktifkan Akun Kembali?'}
        </h2>

        <p className="text-xs text-text-muted mb-4 leading-relaxed">
          {user.is_active ? (
            <>
              Apakah Anda yakin ingin menonaktifkan akun{' '}
              <span className="font-semibold text-text">{user.full_name}</span>? Pengguna tidak akan
              dapat masuk ke dalam sistem, dan seluruh sesi aktifnya akan dicabut seketika.
            </>
          ) : (
            <>
              Apakah Anda yakin ingin mengaktifkan kembali akun{' '}
              <span className="font-semibold text-text">{user.full_name}</span>? Pengguna akan dapat
              kembali masuk ke dalam sistem.
            </>
          )}
        </p>

        {error && (
          <div className="mb-4 p-3 bg-red-50 border border-red-200 text-xs text-red-700 rounded-lg">
            {error}
          </div>
        )}

        <div className="flex items-center justify-end gap-2.5 pt-2">
          <button
            type="button"
            onClick={onClose}
            disabled={isSubmitting}
            className="px-3.5 py-2 text-xs rounded-lg border border-border text-slate-700 hover:bg-slate-50 cursor-pointer"
          >
            Batal
          </button>
          <button
            id="toggle-status-confirm"
            type="button"
            onClick={() => void handleToggle()}
            disabled={isSubmitting}
            className={`px-4 py-2 text-xs font-semibold rounded-lg text-white cursor-pointer ${
              user.is_active
                ? 'bg-danger hover:bg-red-700'
                : 'bg-emerald-600 hover:bg-emerald-700'
            }`}
          >
            {isSubmitting ? 'Memproses…' : user.is_active ? 'Ya, Nonaktifkan' : 'Ya, Aktifkan'}
          </button>
        </div>
      </div>
    </div>
  )
}
