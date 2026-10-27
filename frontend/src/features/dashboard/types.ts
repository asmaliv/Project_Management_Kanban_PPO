import type { Project } from '../projects/types'
import type { Task } from '../tasks/types'
import type { Risk, RiskSummary } from '../risks/types'
import type { AuditFinding, FindingSummary } from '../findings/types'
import type { AuditLog } from '../audit/types'

export interface AdminUserStats {
  total_users: number
  admin_count: number
  pm_count: number
  tl_count: number
  audit_count: number
  risk_count: number
  locked_count: number
  inactive_count: number
}

export interface AdminDashboardData {
  role: 'admin'
  user_stats?: AdminUserStats
  recent_activities?: AuditLog[]
}

export interface PMDashboardData {
  role: 'project_manager'
  projects: Project[]
  totals: {
    active_projects: number
    overdue_tasks: number
    pending_approval: number
    open_findings: number
  }
  overdue_tasks: Task[]
  pending_tasks: Task[]
  finding_stats?: FindingSummary
}

export interface TLColumnStat {
  column_name: string
  count: number
}

export interface TLMemberWorkload {
  user_id: string
  full_name: string
  active_tasks: number
}

export interface TLDashboardData {
  role: 'team_leader'
  projects: Project[]
  totals: {
    total_tasks: number
    overdue_tasks: number
    pending_approval: number
  }
  by_column: TLColumnStat[]
  workload: TLMemberWorkload[]
  overdue: Task[]
  my_tasks: Task[]
}

export interface AuditDashboardData {
  role: 'audit'
  findings_summary?: FindingSummary
  open_findings: AuditFinding[]
  recent_activities?: AuditLog[]
}

export interface RiskManagementDashboardData {
  role: 'risk_management'
  risk_summary?: RiskSummary
  high_critical_risks: Risk[]
}

export type DashboardResponseData =
  | AdminDashboardData
  | PMDashboardData
  | TLDashboardData
  | AuditDashboardData
  | RiskManagementDashboardData
