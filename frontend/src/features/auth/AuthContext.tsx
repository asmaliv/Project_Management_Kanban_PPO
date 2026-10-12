import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react'
import type { ReactNode } from 'react'
import type { User } from '../../types'
import { apiGetMe, apiLogin, apiLogout, ApiClientError } from '../../lib/api'
import { AuthContext } from './context'
import type { AuthContextValue } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // On mount: try to recover session via GET /auth/me
  useEffect(() => {
    let active = true
    const init = async () => {
      try {
        const data = await apiGetMe()
        if (active) setUser(data.user)
      } catch {
        // No valid session — stay logged out
        if (active) setUser(null)
      } finally {
        if (active) setIsLoading(false)
      }
    }
    void init()
    return () => {
      active = false
    }
  }, [])

  const login = useCallback(async (email: string, password: string): Promise<User> => {
    const data = await apiLogin(email, password)
    setUser(data.user)
    return data.user
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiLogout()
    } catch (err) {
      // Even if the API call fails, clear local state
      if (err instanceof ApiClientError && err.status === 401) {
        // Session was already expired
      } else {
        throw err
      }
    } finally {
      setUser(null)
    }
  }, [])

  const refreshUser = useCallback(async () => {
    try {
      const data = await apiGetMe()
      setUser(data.user)
    } catch {
      setUser(null)
    }
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isLoading,
      isAuthenticated: user !== null,
      login,
      logout,
      refreshUser,
    }),
    [user, isLoading, login, logout, refreshUser],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
