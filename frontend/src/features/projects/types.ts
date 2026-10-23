export type ProjectStatus = 'active' | 'on_hold' | 'completed' | 'archived'

export interface Project {
  id: string
  code: string
  name: string
  description: string
  status: ProjectStatus
  owner_id: string
  start_date: string | null
  end_date: string | null
  created_by: string
  created_at: string
  updated_at: string
}

export interface BoardColumn {
  id: string
  project_id: string
  name: string
  position: number
  is_done_column: boolean
  wip_limit: number | null
  created_at: string
}

export interface ProjectMember {
  id: string
  full_name: string
  email: string
  role: string
  added_at: string
}

export interface CreateProjectPayload {
  name: string
  description?: string
  start_date?: string
  end_date?: string
}

export interface UpdateProjectPayload {
  name: string
  description?: string
  status: ProjectStatus
  start_date?: string
  end_date?: string
}

export interface CreateColumnPayload {
  name: string
  position: number
  is_done_column?: boolean
  wip_limit?: number | null
}

export interface UpdateColumnPayload {
  name: string
  position: number
  wip_limit?: number | null
}
