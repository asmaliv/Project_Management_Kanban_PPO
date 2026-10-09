import type { ApiResponse } from '../types'

export async function apiClient<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
  const headers = new Headers(options.headers || {})
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }

  const response = await fetch(endpoint, {
    ...options,
    headers,
    credentials: 'same-origin',
  })

  const json = (await response.json()) as ApiResponse<T>
  if (!response.ok) {
    throw new Error(json.error || `HTTP error ${response.status}`)
  }

  return json
}
