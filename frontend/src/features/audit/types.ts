export interface AuditLog {
  id: string
  created_at: string
  actor_id: string
  actor_role?: string
  action: string
  entity_type: string
  entity_id: string
  project_id?: string | null
  before?: Record<string, unknown> | null
  after?: Record<string, unknown> | null
  ip?: string
  request_id?: string
}

export interface ListAuditLogsParams {
  project_id?: string
  actor_id?: string
  entity_type?: string
  action?: string
  from_date?: string
  to_date?: string
  limit?: number
  offset?: number
}

export interface ListAuditLogsResponse {
  logs: AuditLog[]
  total: number
}
