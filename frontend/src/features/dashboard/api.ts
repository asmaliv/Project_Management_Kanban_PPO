import { apiClient } from '../../lib/api'
import type { DashboardResponseData } from './types'

export async function fetchDashboard(projectId?: string): Promise<DashboardResponseData> {
  const query = projectId ? `?project_id=${projectId}` : ''
  const res = await apiClient<DashboardResponseData>(`/api/v1/dashboard${query}`)
  if (!res.data) {
    throw new Error('Gagal memuat data dashboard')
  }
  return res.data
}
