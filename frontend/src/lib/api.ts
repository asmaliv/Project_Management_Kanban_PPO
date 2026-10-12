import type {
  ApiResponse,
  AuthData,
  CreateUserPayload,
  Pagination,
  UpdateUserPayload,
  UserDetail,
  UserSummary,
} from '../types'

const CSRF_HEADER = 'X-CSRF-Token'

/** In-memory CSRF token, updated on login and /auth/me. */
let csrfToken = ''

export function setCsrfToken(token: string): void {
  csrfToken = token
}

export function getCsrfToken(): string {
  return csrfToken
}

/**
 * Generic API client that wraps fetch with:
 *  - JSON Content-Type
 *  - same-origin credentials (cookies)
 *  - automatic CSRF header on write methods
 *  - response envelope unwrapping
 */
export async function apiClient<T>(
  endpoint: string,
  options: RequestInit = {},
): Promise<ApiResponse<T>> {
  const headers = new Headers(options.headers || {})

  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  // Attach CSRF token on write methods
  const method = (options.method || 'GET').toUpperCase()
  if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method) && csrfToken) {
    headers.set(CSRF_HEADER, csrfToken)
  }

  const response = await fetch(endpoint, {
    ...options,
    headers,
    credentials: 'same-origin',
  })

  const json = (await response.json()) as ApiResponse<T>

  if (!response.ok || !json.success) {
    const err = new ApiClientError(
      json.error?.message || `HTTP error ${response.status}`,
      response.status,
      json.error?.code || 'UNKNOWN',
      json.error?.fields,
    )
    throw err
  }

  return json
}

export class ApiClientError extends Error {
  status: number
  code: string
  fields?: Record<string, string>

  constructor(
    message: string,
    status: number,
    code: string,
    fields?: Record<string, string>,
  ) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

// ─── Auth API ────────────────────────────────────────────────────────

export async function apiLogin(
  email: string,
  password: string,
): Promise<AuthData> {
  const res = await apiClient<AuthData>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
  if (res.data) {
    setCsrfToken(res.data.csrf_token)
  }
  return res.data as AuthData
}

export async function apiLogout(): Promise<void> {
  await apiClient('/api/v1/auth/logout', { method: 'POST' })
  setCsrfToken('')
}

export async function apiGetMe(): Promise<AuthData> {
  const res = await apiClient<AuthData>('/api/v1/auth/me')
  if (res.data) {
    setCsrfToken(res.data.csrf_token)
  }
  return res.data as AuthData
}

export async function apiChangePassword(
  oldPassword: string,
  newPassword: string,
  confirmPassword: string,
): Promise<void> {
  await apiClient('/api/v1/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({
      old_password: oldPassword,
      new_password: newPassword,
      confirm_password: confirmPassword,
    }),
  })
}

// ─── Users API ────────────────────────────────────────────────────────


export async function apiListUsers(
  page = 1,
  pageSize = 20,
): Promise<{ users: UserDetail[]; meta?: Pagination }> {
  const res = await apiClient<UserDetail[]>(`/api/v1/users?page=${page}&page_size=${pageSize}`)
  return {
    users: res.data || [],
    meta: res.meta,
  }
}

export async function apiListUsersSummary(): Promise<UserSummary[]> {
  const res = await apiClient<UserSummary[]>('/api/v1/users')
  return res.data || []
}

export async function apiCreateUser(
  payload: CreateUserPayload,
): Promise<UserDetail> {
  const res = await apiClient<UserDetail>('/api/v1/users', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return res.data as UserDetail
}

export async function apiGetUser(id: string): Promise<UserDetail> {
  const res = await apiClient<UserDetail>(`/api/v1/users/${id}`)
  return res.data as UserDetail
}

export async function apiUpdateUser(
  id: string,
  payload: UpdateUserPayload,
): Promise<UserDetail> {
  const res = await apiClient<UserDetail>(`/api/v1/users/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  return res.data as UserDetail
}

export async function apiDeactivateUser(id: string): Promise<UserDetail> {
  const res = await apiClient<UserDetail>(`/api/v1/users/${id}/deactivate`, {
    method: 'POST',
  })
  return res.data as UserDetail
}

export async function apiActivateUser(id: string): Promise<UserDetail> {
  const res = await apiClient<UserDetail>(`/api/v1/users/${id}/activate`, {
    method: 'POST',
  })
  return res.data as UserDetail
}

export async function apiSetUserActiveStatus(
  id: string,
  isActive: boolean,
): Promise<UserDetail> {
  if (isActive) {
    return apiActivateUser(id)
  }
  return apiDeactivateUser(id)
}

export async function apiResetPassword(
  id: string,
  temporaryPassword: string,
): Promise<void> {
  await apiClient(`/api/v1/users/${id}/reset-password`, {
    method: 'POST',
    body: JSON.stringify({ temporary_password: temporaryPassword }),
  })
}
