import { apiClient } from '../../lib/api'
import type { User, UserRole } from '../../types'

export interface CreateUserInput {
  full_name: string
  role: UserRole
  password: string
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
