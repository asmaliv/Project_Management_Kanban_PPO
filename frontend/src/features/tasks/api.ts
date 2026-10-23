import { apiClient } from '../../lib/api'
import type {
  Task,
  TaskComment,
  CreateTaskPayload,
  UpdateTaskPayload,
  MoveTaskPayload,
  TasksFilter,
} from './types'

export async function fetchProjectTasks(projectId: string): Promise<Task[]> {
  const res = await apiClient<Task[]>(`/api/v1/projects/${projectId}/tasks`)
  return res.data || []
}

export async function fetchTasks(filter?: TasksFilter): Promise<{ tasks: Task[]; total: number }> {
  const params = new URLSearchParams()
  if (filter?.project_id) params.set('project_id', filter.project_id)
  if (filter?.assignee_id) params.set('assignee_id', filter.assignee_id)
  if (filter?.column_id) params.set('column_id', filter.column_id)
  if (filter?.overdue) params.set('overdue', 'true')
  if (filter?.mine) params.set('mine', 'true')
  if (filter?.limit) params.set('limit', String(filter.limit))
  if (filter?.offset) params.set('offset', String(filter.offset))

  const qs = params.toString() ? `?${params.toString()}` : ''
  const res = await apiClient<{ tasks: Task[]; total: number }>(`/api/v1/tasks${qs}`)
  return res.data || { tasks: [], total: 0 }
}

export async function fetchTask(taskId: string): Promise<Task> {
  const res = await apiClient<Task>(`/api/v1/tasks/${taskId}`)
  if (!res.data) {
    throw new Error('Task tidak ditemukan')
  }
  return res.data
}

export async function createTask(projectId: string, payload: CreateTaskPayload): Promise<Task> {
  const res = await apiClient<Task>(`/api/v1/projects/${projectId}/tasks`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal membuat task')
  }
  return res.data
}

export async function updateTask(taskId: string, payload: UpdateTaskPayload): Promise<Task> {
  const res = await apiClient<Task>(`/api/v1/tasks/${taskId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal memperbarui task')
  }
  return res.data
}

export async function moveTask(taskId: string, payload: MoveTaskPayload): Promise<Task> {
  const res = await apiClient<Task>(`/api/v1/tasks/${taskId}/move`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal memindahkan task')
  }
  return res.data
}

export async function deleteTask(taskId: string): Promise<void> {
  await apiClient(`/api/v1/tasks/${taskId}`, {
    method: 'DELETE',
  })
}

export async function fetchTaskComments(taskId: string): Promise<TaskComment[]> {
  const res = await apiClient<TaskComment[]>(`/api/v1/tasks/${taskId}/comments`)
  return res.data || []
}

export async function createTaskComment(taskId: string, body: string): Promise<TaskComment> {
  const res = await apiClient<TaskComment>(`/api/v1/tasks/${taskId}/comments`, {
    method: 'POST',
    body: JSON.stringify({ body }),
  })
  if (!res.data) {
    throw new Error('Gagal menambahkan komentar')
  }
  return res.data
}
