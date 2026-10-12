export type UserRole =
  | 'admin'
  | 'project_manager'
  | 'team_leader'
  | 'audit'
  | 'risk_management'

export interface User {
  id: string
  email: string
  full_name: string
  role: UserRole
  must_change_password: boolean
  created_at: string
  updated_at: string
}

/** Standard backend response envelope (matches httpapi.ApiResponse). */
export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: ApiError
  meta?: Pagination
}

export interface ApiError {
  code: string
  message: string
  fields?: Record<string, string>
}

export interface Pagination {
  page: number
  page_size: number
  total: number
}

export interface AuthData {
  user: User
  csrf_token: string
}

export interface HealthCheckResponse {
  status: string
}
