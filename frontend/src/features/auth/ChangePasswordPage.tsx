import { useState, useMemo } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from './useAuth'
import { apiChangePassword, ApiClientError } from '../../lib/api'

interface PasswordRule {
  id: string
  label: string
  valid: boolean
}

export default function ChangePasswordPage() {
  const { user, refreshUser, logout } = useAuth()
  const navigate = useNavigate()

  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [successMessage, setSuccessMessage] = useState('')

  // Password rules validation
  const rules = useMemo<PasswordRule[]>(() => {
    return [
      {
        id: 'length',
        label: 'Minimal 12 karakter',
        valid: newPassword.length >= 12,
      },
      {
        id: 'uppercase',
        label: 'Mengandung huruf besar (A-Z)',
        valid: /[A-Z]/.test(newPassword),
      },
      {
        id: 'lowercase',
        label: 'Mengandung huruf kecil (a-z)',
        valid: /[a-z]/.test(newPassword),
      },
      {
        id: 'number',
        label: 'Mengandung angka (0-9)',
        valid: /[0-9]/.test(newPassword),
      },
      {
        id: 'symbol',
        label: 'Mengandung simbol khusus (@$!%*?&#^()_+-=~)',
        valid: /[^A-Za-z0-9]/.test(newPassword),
      },
      {
        id: 'different',
        label: 'Berbeda dari password lama',
        valid: newPassword.length > 0 && oldPassword.length > 0 && newPassword !== oldPassword,
      },
      {
        id: 'match',
        label: 'Konfirmasi password cocok',
        valid: confirmPassword.length > 0 && newPassword === confirmPassword,
      },
    ]
  }, [oldPassword, newPassword, confirmPassword])

  const allRulesPassed = useMemo(() => {
    return rules.every((r) => r.valid)
  }, [rules])

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setFieldErrors({})

    if (!allRulesPassed) {
      setError('Harap penuhi semua kriteria keamanan password di bawah.')
      return
    }

    setIsSubmitting(true)

    try {
      await apiChangePassword(oldPassword, newPassword, confirmPassword)
      setSuccessMessage('Password berhasil diubah. Memperbarui sesi…')
      await refreshUser()
      setTimeout(() => {
        navigate('/', { replace: true })
      }, 1000)
    } catch (err: unknown) {
      if (err instanceof ApiClientError) {
        setError(err.message)
        if (err.fields) setFieldErrors(err.fields)
      } else {
        setError('Terjadi kesalahan jaringan. Silakan coba lagi.')
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 py-8">
      <div className="w-full max-w-lg">
        {/* Header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-primary text-white font-bold text-2xl shadow-lg mb-4">
            P
          </div>
          <h1 className="text-2xl font-bold text-text tracking-tight">
            Ganti Password
          </h1>
          <p className="text-sm text-text-muted mt-1">
            {user?.must_change_password
              ? 'Anda wajib mengganti password sementara sebelum dapat melanjutkan.'
              : 'Perbarui password akun Anda demi menjaga keamanan sistem.'}
          </p>
        </div>

        {/* Card */}
        <div className="bg-surface rounded-xl border border-border shadow-sm p-6 sm:p-8">
          {successMessage && (
            <div
              role="status"
              className="mb-6 px-4 py-3 rounded-lg bg-emerald-50 border border-emerald-200 text-sm text-emerald-800 flex items-start gap-2"
            >
              <svg
                className="w-5 h-5 text-emerald-600 shrink-0 mt-0.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
              <span>{successMessage}</span>
            </div>
          )}

          {error && (
            <div
              role="alert"
              className="mb-6 px-4 py-3 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700 flex items-start gap-2"
            >
              <svg
                className="w-5 h-5 text-red-500 shrink-0 mt-0.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                aria-hidden="true"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
              <span>{error}</span>
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-5" noValidate>
            {/* Password Lama */}
            <div>
              <label
                htmlFor="old-password"
                className="block text-sm font-medium text-text mb-1.5"
              >
                Password Lama / Sementara
              </label>
              <input
                id="old-password"
                type="password"
                autoComplete="current-password"
                required
                value={oldPassword}
                onChange={(e) => setOldPassword(e.target.value)}
                disabled={isSubmitting}
                aria-describedby={fieldErrors.old_password ? 'old-password-error' : undefined}
                aria-invalid={!!fieldErrors.old_password}
                className={`w-full px-3.5 py-2.5 rounded-lg border text-sm transition-colors
                  focus:outline-none focus:ring-2 focus:ring-focus-ring focus:border-primary
                  disabled:opacity-50 disabled:cursor-not-allowed
                  ${fieldErrors.old_password ? 'border-danger' : 'border-border hover:border-slate-400'}`}
                placeholder="••••••••"
              />
              {fieldErrors.old_password && (
                <p id="old-password-error" className="mt-1 text-xs text-danger">
                  {fieldErrors.old_password}
                </p>
              )}
            </div>

            {/* Password Baru */}
            <div>
              <label
                htmlFor="new-password"
                className="block text-sm font-medium text-text mb-1.5"
              >
                Password Baru
              </label>
              <input
                id="new-password"
                type="password"
                autoComplete="new-password"
                required
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                disabled={isSubmitting}
                aria-describedby={fieldErrors.new_password ? 'new-password-error' : undefined}
                aria-invalid={!!fieldErrors.new_password}
                className={`w-full px-3.5 py-2.5 rounded-lg border text-sm transition-colors
                  focus:outline-none focus:ring-2 focus:ring-focus-ring focus:border-primary
                  disabled:opacity-50 disabled:cursor-not-allowed
                  ${fieldErrors.new_password ? 'border-danger' : 'border-border hover:border-slate-400'}`}
                placeholder="••••••••••••"
              />
              {fieldErrors.new_password && (
                <p id="new-password-error" className="mt-1 text-xs text-danger">
                  {fieldErrors.new_password}
                </p>
              )}
            </div>

            {/* Konfirmasi Password Baru */}
            <div>
              <label
                htmlFor="confirm-password"
                className="block text-sm font-medium text-text mb-1.5"
              >
                Konfirmasi Password Baru
              </label>
              <input
                id="confirm-password"
                type="password"
                autoComplete="new-password"
                required
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                disabled={isSubmitting}
                aria-describedby={fieldErrors.confirm_password ? 'confirm-password-error' : undefined}
                aria-invalid={!!fieldErrors.confirm_password}
                className={`w-full px-3.5 py-2.5 rounded-lg border text-sm transition-colors
                  focus:outline-none focus:ring-2 focus:ring-focus-ring focus:border-primary
                  disabled:opacity-50 disabled:cursor-not-allowed
                  ${fieldErrors.confirm_password ? 'border-danger' : 'border-border hover:border-slate-400'}`}
                placeholder="••••••••••••"
              />
              {fieldErrors.confirm_password && (
                <p id="confirm-password-error" className="mt-1 text-xs text-danger">
                  {fieldErrors.confirm_password}
                </p>
              )}
            </div>

            {/* Password Policy Checklist */}
            <div className="bg-surface-muted rounded-lg p-3.5 border border-border">
              <p className="text-xs font-semibold text-text mb-2.5">
                Kriteria Keamanan Password (Standar Perbankan):
              </p>
              <ul className="space-y-1.5 text-xs">
                {rules.map((r) => (
                  <li
                    key={r.id}
                    className={`flex items-center gap-2 transition-colors ${
                      r.valid ? 'text-success font-medium' : 'text-text-muted'
                    }`}
                  >
                    {r.valid ? (
                      <svg
                        className="w-4 h-4 text-success shrink-0"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                        aria-hidden="true"
                      >
                        <path
                          fillRule="evenodd"
                          d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                          clipRule="evenodd"
                        />
                      </svg>
                    ) : (
                      <span className="w-1.5 h-1.5 rounded-full bg-slate-300 ml-1.5 mr-1 shrink-0" />
                    )}
                    <span>{r.label}</span>
                  </li>
                ))}
              </ul>
            </div>

            {/* Actions */}
            <div className="flex flex-col gap-3 pt-2">
              <button
                id="change-password-submit"
                type="submit"
                disabled={isSubmitting || !allRulesPassed}
                className="w-full flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg
                  bg-primary text-white text-sm font-semibold shadow-sm
                  hover:bg-primary-hover active:scale-[0.98]
                  focus:outline-none focus:ring-2 focus:ring-focus-ring focus:ring-offset-2
                  disabled:opacity-50 disabled:cursor-not-allowed
                  transition-all duration-150 cursor-pointer"
              >
                {isSubmitting && (
                  <svg
                    className="animate-spin h-4 w-4"
                    viewBox="0 0 24 24"
                    fill="none"
                    aria-hidden="true"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
                    />
                  </svg>
                )}
                {isSubmitting ? 'Memproses…' : 'Simpan Password Baru'}
              </button>

              <button
                type="button"
                onClick={() => void logout()}
                disabled={isSubmitting}
                className="w-full text-center text-xs text-text-muted hover:text-danger py-1 transition-colors cursor-pointer"
              >
                Keluar dari Akun (Logout)
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
