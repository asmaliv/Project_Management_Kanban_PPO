export type UserRole =
  | 'admin'
  | 'project_manager'
  | 'team_leader'
  | 'audit'
  | 'risk_management'

export interface User {
  id: string
  email: string
  fullName: string
  role: UserRole
  isActive: boolean
  mustChangePassword: boolean
  createdAt: string
  updatedAt: string
}

export interface ApiResponse<T> {
  data?: T
  error?: string
}

export interface HealthCheckResponse {
  status: string
}
