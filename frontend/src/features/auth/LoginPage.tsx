import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from './useAuth'
import { ApiClientError } from '../../lib/api'

export default function LoginPage() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Where to redirect after successful login
  const from = (location.state as { from?: string })?.from || '/'

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    setFieldErrors({})
    setIsSubmitting(true)

    try {
      const user = await login(email, password)
      if (user.must_change_password) {
        navigate('/change-password', { replace: true })
      } else {
        navigate(from, { replace: true })
      }
    } catch (err) {
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
    <div className="min-h-screen flex items-center justify-center bg-background px-4">
      <div className="w-full max-w-md">
        {/* Logo / Brand */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-primary text-white font-bold text-2xl shadow-lg mb-4">
            P
          </div>
          <h1 className="text-2xl font-bold text-text tracking-tight">
            PPO Kanban Board
          </h1>
          <p className="text-sm text-text-muted mt-1">
            Sistem Manajemen Proyek — Lingkungan Perbankan
          </p>
        </div>

        {/* Login Card */}
        <div className="bg-surface rounded-xl border border-border shadow-sm p-6 sm:p-8">
          <h2 className="text-lg font-semibold text-text mb-6">Masuk</h2>

          {error && (
            <div
              role="alert"
              className="mb-4 px-4 py-3 rounded-lg bg-red-50 border border-red-200 text-sm text-red-700 flex items-start gap-2"
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
            {/* Email */}
            <div>
              <label
                htmlFor="login-email"
                className="block text-sm font-medium text-text mb-1.5"
              >
                Email
              </label>
              <input
                id="login-email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isSubmitting}
                aria-describedby={fieldErrors.email ? 'login-email-error' : undefined}
                aria-invalid={!!fieldErrors.email}
                className={`w-full px-3.5 py-2.5 rounded-lg border text-sm transition-colors
                  focus:outline-none focus:ring-2 focus:ring-focus-ring focus:border-primary
                  disabled:opacity-50 disabled:cursor-not-allowed
                  ${fieldErrors.email ? 'border-danger' : 'border-border hover:border-slate-400'}`}
                placeholder="nama@bank.co.id"
              />
              {fieldErrors.email && (
                <p id="login-email-error" className="mt-1 text-xs text-danger">
                  {fieldErrors.email}
                </p>
              )}
            </div>

            {/* Password */}
            <div>
              <label
                htmlFor="login-password"
                className="block text-sm font-medium text-text mb-1.5"
              >
                Password
              </label>
              <input
                id="login-password"
                type="password"
                autoComplete="current-password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                disabled={isSubmitting}
                aria-describedby={fieldErrors.password ? 'login-password-error' : undefined}
                aria-invalid={!!fieldErrors.password}
                className={`w-full px-3.5 py-2.5 rounded-lg border text-sm transition-colors
                  focus:outline-none focus:ring-2 focus:ring-focus-ring focus:border-primary
                  disabled:opacity-50 disabled:cursor-not-allowed
                  ${fieldErrors.password ? 'border-danger' : 'border-border hover:border-slate-400'}`}
                placeholder="••••••••"
              />
              {fieldErrors.password && (
                <p id="login-password-error" className="mt-1 text-xs text-danger">
                  {fieldErrors.password}
                </p>
              )}
            </div>

            {/* Submit */}
            <button
              id="login-submit"
              type="submit"
              disabled={isSubmitting || !email || !password}
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
              {isSubmitting ? 'Memproses…' : 'Masuk'}
            </button>
          </form>

          <p className="mt-5 text-center text-xs text-text-muted">
            Lupa password? Hubungi Administrator untuk reset password.
          </p>
        </div>

        {/* Footer */}
        <p className="text-center text-xs text-text-muted mt-6">
          PPO Kanban Board • Internal Banking Workflow
        </p>
      </div>
    </div>
  )
}
