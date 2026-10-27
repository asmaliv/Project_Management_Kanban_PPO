import { apiClient } from '../../lib/api'
import type {
  AuditFinding,
  FindingSummary,
  CreateFindingPayload,
  RespondFindingPayload,
  ReopenFindingPayload,
  ListFindingsParams,
  ListFindingsResponse,
} from './types'

export async function fetchFindings(params?: ListFindingsParams): Promise<ListFindingsResponse> {
  const query = new URLSearchParams()
  if (params?.project_id) query.set('project_id', params.project_id)
  if (params?.task_id) query.set('task_id', params.task_id)
  if (params?.severity) query.set('severity', params.severity)
  if (params?.status) query.set('status', params.status)
  if (params?.search) query.set('search', params.search)
  if (params?.page) query.set('page', params.page.toString())
  if (params?.per_page) query.set('per_page', params.per_page.toString())

  const queryString = query.toString() ? `?${query.toString()}` : ''
  const res = await apiClient<ListFindingsResponse>(`/api/v1/findings${queryString}`)
  return res.data || { findings: [], total: 0, page: 1, per_page: 20 }
}

export async function fetchFinding(id: string): Promise<AuditFinding> {
  const res = await apiClient<AuditFinding>(`/api/v1/findings/${id}`)
  if (!res.data) {
    throw new Error('Temuan audit tidak ditemukan')
  }
  return res.data
}

export async function createFinding(payload: CreateFindingPayload): Promise<AuditFinding> {
  const res = await apiClient<AuditFinding>('/api/v1/findings', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal membuat temuan audit')
  }
  return res.data
}

export async function respondFinding(id: string, payload: RespondFindingPayload): Promise<AuditFinding> {
  const res = await apiClient<AuditFinding>(`/api/v1/findings/${id}/respond`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal merespons temuan audit')
  }
  return res.data
}

export async function reopenFinding(id: string, payload?: ReopenFindingPayload): Promise<AuditFinding> {
  const res = await apiClient<AuditFinding>(`/api/v1/findings/${id}/reopen`, {
    method: 'POST',
    body: JSON.stringify(payload || {}),
  })
  if (!res.data) {
    throw new Error('Gagal membuka kembali temuan audit')
  }
  return res.data
}

export async function closeFinding(id: string): Promise<AuditFinding> {
  const res = await apiClient<AuditFinding>(`/api/v1/findings/${id}/close`, {
    method: 'POST',
  })
  if (!res.data) {
    throw new Error('Gagal menutup temuan audit')
  }
  return res.data
}

export async function fetchFindingSummary(projectId?: string): Promise<FindingSummary> {
  const query = projectId ? `?project_id=${projectId}` : ''
  const res = await apiClient<FindingSummary>(`/api/v1/findings/summary${query}`)
  if (!res.data) {
    throw new Error('Gagal memuat ringkasan temuan audit')
  }
  return res.data
}
