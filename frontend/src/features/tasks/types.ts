export type TaskPriority = 'low' | 'medium' | 'high' | 'critical'

export interface Task {
  id: string
  project_id: string
  column_id: string
  title: string
  description: string
  priority: TaskPriority
  assignee_id: string | null
  assignee_name?: string
  reporter_id: string
  reporter_name?: string
  due_date: string | null
  progress_percent: number
  position: number
  labels: string[]
  version: number
  completed_at: string | null
  created_at: string
  updated_at: string
  column_name?: string
  project_code?: string
  project_name?: string
}

export interface TaskComment {
  id: string
  task_id: string
  author_id: string
  author_name: string
  author_role: string
  body: string
  created_at: string
}

export interface CreateTaskPayload {
  column_id: string
  title: string
  description?: string
  priority?: TaskPriority
  assignee_id?: string | null
  due_date?: string | null
  progress_percent?: number
  labels?: string[]
}

export interface UpdateTaskPayload {
  version: number
  title: string
  description?: string
  priority?: TaskPriority
  assignee_id?: string | null
  due_date?: string | null
  progress_percent?: number
  labels?: string[]
}

export interface MoveTaskPayload {
  to_column_id: string
  position: number
  version: number
}

export interface TasksFilter {
  project_id?: string
  assignee_id?: string
  column_id?: string
  overdue?: boolean
  mine?: boolean
  limit?: number
  offset?: number
}
