import { apiClient } from '../../lib/api'
import type { User, UserRole } from '../../types'

export interface CreateUserInput {
  username: string
  full_name: string
  role: UserRole
  password: string
}

export interface UpdateUserInput {
  full_name: string
  role: UserRole
  is_active: boolean
  password?: string
}

export async function apiListUsers(): Promise<User[]> {
  const res = await apiClient<User[]>('/api/v1/users')
  return res.data ?? []
}

export async function apiCreateUser(input: CreateUserInput): Promise<User> {
  const res = await apiClient<User>('/api/v1/users', {
    method: 'POST',
    body: JSON.stringify(input),
  })
  if (!res.data) throw new Error('Gagal menambahkan pengguna')
  return res.data
}

export async function apiUpdateUser(id: string, input: UpdateUserInput): Promise<User> {
  const res = await apiClient<User>(`/api/v1/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
  if (!res.data) throw new Error('Gagal memperbarui pengguna')
  return res.data
}

export async function apiDeleteUser(id: string): Promise<void> {
  await apiClient<{ deleted: boolean }>(`/api/v1/users/${id}`, {
    method: 'DELETE',
  })
}
