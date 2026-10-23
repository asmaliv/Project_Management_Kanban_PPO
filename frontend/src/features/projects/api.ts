import { apiClient } from '../../lib/api'
import type {
  Project,
  BoardColumn,
  ProjectMember,
  CreateProjectPayload,
  UpdateProjectPayload,
  CreateColumnPayload,
  UpdateColumnPayload,
} from './types'

// ─── Projects ────────────────────────────────────────────────────────────────

export async function fetchProjects(): Promise<Project[]> {
  const res = await apiClient<Project[]>('/api/v1/projects')
  return res.data || []
}

export async function fetchProject(id: string): Promise<Project> {
  const res = await apiClient<Project>(`/api/v1/projects/${id}`)
  if (!res.data) {
    throw new Error('Proyek tidak ditemukan')
  }
  return res.data
}

export async function createProject(payload: CreateProjectPayload): Promise<Project> {
  const res = await apiClient<Project>('/api/v1/projects', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal membuat proyek')
  }
  return res.data
}

export async function updateProject(id: string, payload: UpdateProjectPayload): Promise<Project> {
  const res = await apiClient<Project>(`/api/v1/projects/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal memperbarui proyek')
  }
  return res.data
}

export async function archiveProject(id: string): Promise<Project> {
  const res = await apiClient<Project>(`/api/v1/projects/${id}/archive`, {
    method: 'POST',
  })
  if (!res.data) {
    throw new Error('Gagal mengarsipkan proyek')
  }
  return res.data
}

export async function transferProjectOwner(id: string, newOwnerId: string): Promise<Project> {
  const res = await apiClient<Project>(`/api/v1/projects/${id}/transfer-owner`, {
    method: 'POST',
    body: JSON.stringify({ new_owner_id: newOwnerId }),
  })
  if (!res.data) {
    throw new Error('Gagal mengalihkan kepemilikan proyek')
  }
  return res.data
}

// ─── Members ─────────────────────────────────────────────────────────────────

export async function fetchProjectMembers(projectId: string): Promise<ProjectMember[]> {
  const res = await apiClient<ProjectMember[]>(`/api/v1/projects/${projectId}/members`)
  return res.data || []
}

export async function addProjectMember(projectId: string, userId: string): Promise<void> {
  await apiClient(`/api/v1/projects/${projectId}/members`, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId }),
  })
}

export async function removeProjectMember(projectId: string, userId: string): Promise<void> {
  await apiClient(`/api/v1/projects/${projectId}/members/${userId}`, {
    method: 'DELETE',
  })
}

// ─── Board Columns ───────────────────────────────────────────────────────────

export async function fetchBoardColumns(projectId: string): Promise<BoardColumn[]> {
  const res = await apiClient<BoardColumn[]>(`/api/v1/projects/${projectId}/columns`)
  return res.data || []
}

export async function createBoardColumn(
  projectId: string,
  payload: CreateColumnPayload,
): Promise<BoardColumn> {
  const res = await apiClient<BoardColumn>(`/api/v1/projects/${projectId}/columns`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal menambahkan kolom')
  }
  return res.data
}

export async function updateBoardColumn(
  columnId: string,
  payload: UpdateColumnPayload,
): Promise<BoardColumn> {
  const res = await apiClient<BoardColumn>(`/api/v1/columns/${columnId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  if (!res.data) {
    throw new Error('Gagal memperbarui kolom')
  }
  return res.data
}

export async function deleteBoardColumn(columnId: string): Promise<void> {
  await apiClient(`/api/v1/columns/${columnId}`, {
    method: 'DELETE',
  })
}
