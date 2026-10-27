import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './features/auth/AuthContext'
import { ProtectedRoute, GuestRoute } from './features/auth/RouteGuards'
import LoginPage from './features/auth/LoginPage'
import ChangePasswordPage from './features/auth/ChangePasswordPage'
import AppLayout from './components/layout/AppLayout'
import DashboardPage from './features/dashboard/DashboardPage'
import PlaceholderPage from './features/placeholder/PlaceholderPage'
import ProjectListPage from './features/projects/ProjectListPage'
import ProjectDetailPage from './features/projects/ProjectDetailPage'
import RiskRegisterPage from './features/risks/RiskRegisterPage'
import FindingListPage from './features/findings/FindingListPage'
import AuditTrailPage from './features/audit/AuditTrailPage'
import TeamTasksPage from './features/tasks/TeamTasksPage'

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
              <Route path="/projects" element={<ProjectListPage />} />
              <Route path="/projects/:id" element={<ProjectDetailPage />} />
              <Route path="/team-tasks" element={<TeamTasksPage />} />
              <Route path="/my-tasks" element={<TeamTasksPage initialMineOnly={true} />} />
              <Route path="/risks" element={<RiskRegisterPage />} />
              <Route path="/findings" element={<FindingListPage />} />
              <Route path="/audit-trail" element={<AuditTrailPage />} />
            </Route>
          </Route>

          {/* 404 / Catch-all */}
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
