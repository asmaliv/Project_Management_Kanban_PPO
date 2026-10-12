import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './features/auth/AuthContext'
import { ProtectedRoute, GuestRoute } from './features/auth/RouteGuards'
import LoginPage from './features/auth/LoginPage'
import ChangePasswordPage from './features/auth/ChangePasswordPage'
import AppLayout from './components/layout/AppLayout'
import DashboardPage from './features/dashboard/DashboardPage'
import PlaceholderPage from './features/placeholder/PlaceholderPage'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          {/* Public / Guest Routes */}
          <Route element={<GuestRoute />}>
            <Route path="/login" element={<LoginPage />} />
          </Route>

          {/* Force Change Password (protected, but without sidebar layout) */}
          <Route element={<ProtectedRoute />}>
            <Route path="/change-password" element={<ChangePasswordPage />} />
          </Route>

          {/* Authenticated Application Shell */}
          <Route element={<ProtectedRoute />}>
            <Route element={<AppLayout />}>
              <Route path="/" element={<DashboardPage />} />
              <Route path="/users" element={<PlaceholderPage />} />
              <Route path="/projects" element={<PlaceholderPage />} />
              <Route path="/team-tasks" element={<PlaceholderPage />} />
              <Route path="/my-tasks" element={<PlaceholderPage />} />
              <Route path="/risks" element={<PlaceholderPage />} />
              <Route path="/findings" element={<PlaceholderPage />} />
              <Route path="/audit-trail" element={<PlaceholderPage />} />
            </Route>
          </Route>

          {/* 404 / Catch-all */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
