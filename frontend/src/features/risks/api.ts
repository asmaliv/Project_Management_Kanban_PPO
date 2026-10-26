import { apiClient } from '../../lib/api'
import type {
  Risk,
  RiskSummary,
  CreateRiskPayload,
  AssessRiskPayload,
  UpdateRiskPayload,
  ListRisksParams,
  ListRisksResponse,
  TaskRiskCount,
} from './types'

export async function fetchRisks(params?: ListRisksParams): Promise<ListRisksResponse> {
  const query = new URLSearchParams()
  if (params?.project_id) query.set('project_id', params.project_id)
  if (params?.task_id) query.set('task_id', params.task_id)
  if (params?.level) query.set('level', params.level)
  if (params?.status) query.set('status', params.status)
  if (params?.category) query.set('category', params.category)
  if (params?.search) query.set('search', params.search)
  if (params?.page) query.set('page', params.page.toString())
  if (params?.per_page) query.set('per_page', params.per_page.toString())

  const queryString = query.toString() ? `?${query.toString()}` : ''
  const res = await apiClient<ListRisksResponse>(`/api/v1/risks${queryString}`)
  return res.data || { risks: [], total: 0, page: 1, per_page: 20 }
}

export async function fetchRisk(id: string): Promise<Risk> {
  const res = await apiClient<Risk>(`/api/v1/risks/${id}`)
  if (!res.data) {
    throw new Error('Risiko tidak ditemukan')
  }
  return res.data
}

export async function createRisk(payload: CreateRiskPayload): Promise<Risk> {
  const res = await apiClient<Risk>('/api/v1/risks', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal mengajukan risiko')
  }
  return res.data
}

export async function updateRisk(id: string, payload: UpdateRiskPayload): Promise<Risk> {
  const res = await apiClient<Risk>(`/api/v1/risks/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal memperbarui risiko')
  }
  return res.data
}

export async function assessRisk(id: string, payload: AssessRiskPayload): Promise<Risk> {
  const res = await apiClient<Risk>(`/api/v1/risks/${id}/assess`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal menilai risiko')
  }
  return res.data
}

export async function closeRisk(id: string): Promise<Risk> {
  const res = await apiClient<Risk>(`/api/v1/risks/${id}/close`, {
    method: 'POST',
  })
  if (!res.data) {
    throw new Error('Gagal menutup risiko')
  }
  return res.data
}

export async function fetchRiskSummary(projectId?: string): Promise<RiskSummary> {
  const query = projectId ? `?project_id=${projectId}` : ''
  const res = await apiClient<RiskSummary>(`/api/v1/risks/summary${query}`)
  if (!res.data) {
    throw new Error('Gagal memuat ringkasan risiko')
  }
  return res.data
}

export async function fetchTaskRiskCounts(taskIds: string[]): Promise<Record<string, TaskRiskCount>> {
  if (taskIds.length === 0) return {}
  const res = await apiClient<{ counts: Record<string, TaskRiskCount> }>(
    `/api/v1/risks/task-counts?task_ids=${taskIds.join(',')}`
  )
  return res.data?.counts || {}
}
