export type RiskCategory =
  | 'operational'
  | 'schedule'
  | 'technical'
  | 'compliance'
  | 'financial'
  | 'security'

export type RiskLevel = 'low' | 'medium' | 'high' | 'critical'

export type RiskStatus =
  | 'identified'
  | 'assessed'
  | 'mitigating'
  | 'accepted'
  | 'closed'

export interface Risk {
  id: string
  code: string
  project_id: string
  project_name?: string
  project_code?: string
  task_id?: string | null
  task_title?: string | null
  title: string
  description: string
  category: RiskCategory
  likelihood?: number | null
  impact?: number | null
  score?: number | null
  level?: RiskLevel | null
  status: RiskStatus
  mitigation_plan: string
  owner_id?: string | null
  owner_name?: string | null
  raised_by: string
  raised_by_name?: string
  created_at: string
  updated_at: string
  closed_at?: string | null
}

export interface HeatmapCell {
  likelihood: number
  impact: number
  score: number
  level: RiskLevel
  count: number
}

export interface RiskSummary {
  total_risks: number
  open_risks: number
  high_critical_risks: number
  unmitigated_risks: number
  by_level: Record<string, number>
  heatmap: HeatmapCell[]
}

export interface CreateRiskPayload {
  project_id: string
  task_id?: string | null
  title: string
  description?: string
  category: RiskCategory
  owner_id?: string | null
}

export interface AssessRiskPayload {
  likelihood: number
  impact: number
  mitigation_plan?: string
}

export interface UpdateRiskPayload {
  title?: string
  description?: string
  category?: RiskCategory
  task_id?: string | null
  owner_id?: string | null
  status?: RiskStatus
  mitigation_plan?: string
}

export interface ListRisksParams {
  project_id?: string
  task_id?: string
  level?: string
  status?: string
  category?: string
  search?: string
  page?: number
  per_page?: number
}

export interface ListRisksResponse {
  risks: Risk[]
  total: number
  page: number
  per_page: number
}

export interface TaskRiskCount {
  task_id: string
  total_risks: number
  high_critical_risks: number
}
