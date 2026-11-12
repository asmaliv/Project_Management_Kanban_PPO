import { useState, useEffect } from 'react'
import { apiListUsers, apiCreateUser } from './api'
import type { User, UserRole } from '../../types'

const ROLE_CONFIG: Record<UserRole, { label: string; badge: string }> = {
  admin: { label: 'Administrator', badge: 'bg-purple-100 text-purple-800 border-purple-200' },
  project_manager: { label: 'Project Manager', badge: 'bg-blue-100 text-blue-800 border-blue-200' },
  team_leader: { label: 'Team Leader', badge: 'bg-emerald-100 text-emerald-800 border-emerald-200' },
  audit: { label: 'Team Audit', badge: 'bg-amber-100 text-amber-800 border-amber-200' },
  risk_management: { label: 'Team Risk', badge: 'bg-rose-100 text-rose-800 border-rose-200' },
}

export default function UsersManagementPage() {
  const [users, setUsers] = useState<User[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState('')
  const [isModalOpen, setIsModalOpen] = useState(false)

  // Form states
  const [fullName, setFullName] = useState('')
  const [role, setRole] = useState<UserRole>('project_manager')
  const [password, setPassword] = useState('')
  const [formError, setFormError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [successMessage, setSuccessMessage] = useState('')

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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')

    if (!fullName.trim()) {
      setFormError('Nama lengkap pengguna wajib diisi.')
      return
    }
    if (password.length < 12) {
      setFormError('Password sementara minimal 12 karakter.')
      return
    }

    try {
      setIsSubmitting(true)
      const newUser = await apiCreateUser({
        full_name: fullName.trim(),
        role,
        password,
      })
      setSuccessMessage(`Pengguna "${newUser.full_name}" berhasil ditambahkan sebagai ${ROLE_CONFIG[role].label}.`)
      setIsModalOpen(false)
      setFullName('')
      setPassword('')
      setRole('project_manager')
      void fetchUsers()
      setTimeout(() => setSuccessMessage(''), 5000)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setFormError(err.message)
      } else {
        setFormError('Terjadi kesalahan saat menambahkan pengguna.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-text tracking-tight">Manajemen Pengguna</h1>
          <p className="text-sm text-text-muted mt-1">
            Kelola akun dan hak akses: Project Manager, Team Leader, Team Risk, dan Team Audit.
          </p>
        </div>
        <button
          onClick={() => {
            setFormError('')
            setIsModalOpen(true)
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
                <th className="px-6 py-3.5">Status</th>
                <th className="px-6 py-3.5">Password Sementara</th>
                <th className="px-6 py-3.5">Terdaftar</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {isLoading ? (
                <tr>
                  <td colSpan={5} className="px-6 py-8 text-center text-text-muted">
                    Memuat data pengguna…
                  </td>
                </tr>
              ) : users.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-6 py-8 text-center text-text-muted">
                    Belum ada data pengguna.
                  </td>
                </tr>
              ) : (
                users.map((u) => {
                  const roleMeta = ROLE_CONFIG[u.role] || { label: u.role, badge: 'bg-slate-100 text-slate-800' }
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
                            <div className="font-semibold">{u.full_name}</div>
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
                        <span className="inline-flex items-center gap-1.5 text-xs text-emerald-700 font-medium">
                          <span className="w-2 h-2 rounded-full bg-emerald-500" />
                          Aktif
                        </span>
                      </td>
                      <td className="px-6 py-4 text-xs">
                        {u.must_change_password ? (
                          <span className="text-amber-700 bg-amber-50 px-2 py-0.5 rounded border border-amber-200">
                            Wajib Ganti Password
                          </span>
                        ) : (
                          <span className="text-text-muted">Sudah Diganti</span>
                        )}
                      </td>
                      <td className="px-6 py-4 text-xs text-text-muted">
                        {u.created_at ? new Date(u.created_at).toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric' }) : '-'}
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
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-xs p-4">
          <div className="bg-surface rounded-xl border border-border shadow-xl w-full max-w-md p-6 space-y-5 animate-in fade-in zoom-in duration-150">
            <div className="flex items-center justify-between pb-3 border-b border-border">
              <h2 className="text-lg font-bold text-text">Tambah Pengguna Baru</h2>
              <button
                onClick={() => setIsModalOpen(false)}
                className="text-text-muted hover:text-text cursor-pointer"
              >
                ✕
              </button>
            </div>

            {formError && (
              <div className="p-3 rounded-lg bg-red-50 border border-red-200 text-xs text-red-700">
                {formError}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Nama Lengkap <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="Contoh: Budi Prasetyo"
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-text mb-1.5">
                  Role / Peran <span className="text-red-500">*</span>
                </label>
                <select
                  value={role}
                  onChange={(e) => setRole(e.target.value as UserRole)}
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
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  className="w-full px-3.5 py-2 text-sm rounded-lg border border-border focus:outline-none focus:ring-2 focus:ring-primary focus:border-primary font-mono"
                />
                <p className="text-[11px] text-text-muted mt-1">
                  Pengguna akan dipaksa mengganti password ini saat login pertama kali.
                </p>
              </div>

              <div className="flex items-center justify-end gap-3 pt-3 border-t border-border">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-4 py-2 text-sm font-medium text-text-muted hover:text-text cursor-pointer"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="px-4 py-2 bg-primary text-white text-sm font-semibold rounded-lg hover:bg-primary-hover disabled:opacity-50 transition cursor-pointer"
                >
                  {isSubmitting ? 'Menyimpan…' : 'Simpan Pengguna'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
