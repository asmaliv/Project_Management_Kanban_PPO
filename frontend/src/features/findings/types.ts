export type FindingSeverity = 'low' | 'medium' | 'high' | 'critical'

export type FindingStatus = 'open' | 'responded' | 'closed'

export interface AuditFinding {
  id: string
  code: string
  project_id: string
  project_name?: string
  project_code?: string
  task_id?: string | null
  task_title?: string | null
  title: string
  description: string
  severity: FindingSeverity
  status: FindingStatus
  raised_by: string
  raised_by_name?: string
  response?: string
  responded_by?: string | null
  responded_by_name?: string | null
  responded_at?: string | null
  closed_by?: string | null
  closed_by_name?: string | null
  closed_at?: string | null
  created_at: string
  updated_at: string
}

export interface CreateFindingPayload {
  project_id: string
  task_id?: string | null
  title: string
  description?: string
  severity: FindingSeverity
}

export interface RespondFindingPayload {
  response: string
}

export interface ReopenFindingPayload {
  comment?: string
}

export interface ListFindingsParams {
  project_id?: string
  task_id?: string
  severity?: FindingSeverity
  status?: FindingStatus
  search?: string
  page?: number
  per_page?: number
}

export interface ListFindingsResponse {
  findings: AuditFinding[]
  total: number
  page: number
  per_page: number
}

export interface FindingSummary {
  total_findings: number
  open_findings: number
  critical_count: number
  high_count: number
  medium_count: number
  low_count: number
  responded_count: number
}
