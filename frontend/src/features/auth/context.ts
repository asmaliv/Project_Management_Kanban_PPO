import { createContext } from 'react'
import type { User } from '../../types'

export interface AuthState {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
}

export interface AuthContextValue extends AuthState {
  login: (email: string, password: string) => Promise<User>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)
