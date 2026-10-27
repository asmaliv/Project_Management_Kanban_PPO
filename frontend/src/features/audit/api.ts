import { apiClient, getCsrfToken } from '../../lib/api'
import type { ListAuditLogsParams, ListAuditLogsResponse } from './types'

export async function fetchAuditLogs(params?: ListAuditLogsParams): Promise<ListAuditLogsResponse> {
  const query = new URLSearchParams()
  if (params?.project_id) query.set('project_id', params.project_id)
  if (params?.actor_id) query.set('actor_id', params.actor_id)
  if (params?.entity_type) query.set('entity_type', params.entity_type)
  if (params?.action) query.set('action', params.action)
  if (params?.from_date) query.set('from_date', params.from_date)
  if (params?.to_date) query.set('to_date', params.to_date)
  if (params?.limit) query.set('limit', params.limit.toString())
  if (params?.offset) query.set('offset', params.offset.toString())

  const queryString = query.toString() ? `?${query.toString()}` : ''
  const res = await apiClient<ListAuditLogsResponse>(`/api/v1/audit-logs${queryString}`)
  return res.data || { logs: [], total: 0 }
}

export async function exportAuditLogsCSV(params?: ListAuditLogsParams): Promise<void> {
  const query = new URLSearchParams()
  if (params?.project_id) query.set('project_id', params.project_id)
  if (params?.actor_id) query.set('actor_id', params.actor_id)
  if (params?.entity_type) query.set('entity_type', params.entity_type)
  if (params?.action) query.set('action', params.action)
  if (params?.from_date) query.set('from_date', params.from_date)
  if (params?.to_date) query.set('to_date', params.to_date)

  const queryString = query.toString() ? `?${query.toString()}` : ''
  const headers = new Headers()
  const token = getCsrfToken()
  if (token) {
    headers.set('X-CSRF-Token', token)
  }

  const response = await fetch(`/api/v1/audit-logs/export${queryString}`, {
    method: 'GET',
    headers,
    credentials: 'same-origin',
  })

  if (!response.ok) {
    if (response.status === 429) {
      throw new Error('Batas ekspor tercapai (maksimal 5 kali per menit). Silakan coba sesaat lagi.')
    }
    if (response.status === 403) {
      throw new Error('Anda tidak memiliki izin untuk mengekspor audit trail (khusus role Audit).')
    }
    throw new Error(`Gagal mengekspor data audit trail (Status: ${response.status})`)
  }

  const blob = await response.blob()
  const url = window.URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  const dateStr = new Date().toISOString().split('T')[0]
  a.download = `audit-trail-${dateStr}.csv`
  document.body.appendChild(a)
  a.click()
  window.URL.revokeObjectURL(url)
  document.body.removeChild(a)
}
