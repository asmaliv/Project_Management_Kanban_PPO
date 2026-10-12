import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from './useAuth'
import type { UserRole } from '../../types'

interface ProtectedRouteProps {
  allowedRoles?: UserRole[]
}

/**
 * Ensures user is authenticated.
 * Enforces must_change_password redirect.
 * Supports optional role restriction.
 */
export function ProtectedRoute({ allowedRoles }: ProtectedRouteProps) {
  const { user, isLoading, isAuthenticated } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return <LoadingScreen />
  }

  if (!isAuthenticated || !user) {
    return <Navigate to="/login" state={{ from: location.pathname + location.search }} replace />
  }

  // Force password change if required
  if (user.must_change_password && location.pathname !== '/change-password') {
    return <Navigate to="/change-password" replace />
  }

  // If password does not need to be changed, don't allow staying on /change-password unless accessed directly
  if (!user.must_change_password && location.pathname === '/change-password') {
    return <Navigate to="/" replace />
  }

  // Role check if specified
  if (allowedRoles && allowedRoles.length > 0 && !allowedRoles.includes(user.role)) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center p-6 text-center">
        <div className="w-12 h-12 rounded-full bg-red-100 text-red-600 flex items-center justify-center mb-3">
          <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m0 0v2m0-2h2m-2 0H10m4-11a4 4 0 00-8 0v4h8V6z" />
          </svg>
        </div>
        <h2 className="text-lg font-bold text-text">Akses Ditolak (403)</h2>
        <p className="text-sm text-text-muted mt-1 max-w-md">
          Peran Anda ({user.role}) tidak memiliki izin untuk mengakses halaman ini.
        </p>
      </div>
    )
  }

  return <Outlet />
}

/**
 * Route only accessible when NOT logged in (e.g. /login).
 */
export function GuestRoute() {
  const { user, isLoading, isAuthenticated } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return <LoadingScreen />
  }

  if (isAuthenticated && user) {
    if (user.must_change_password) {
      return <Navigate to="/change-password" replace />
    }
    const from = (location.state as { from?: string })?.from || '/'
    return <Navigate to={from} replace />
  }

  return <Outlet />
}

function LoadingScreen() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background">
      <div className="flex flex-col items-center gap-3">
        <div className="w-8 h-8 border-3 border-primary border-t-transparent rounded-full animate-spin" />
        <p className="text-xs text-text-muted font-medium">Memeriksa sesi pengguna…</p>
      </div>
    </div>
  )
}
