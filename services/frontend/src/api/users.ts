import { apiClient } from './client'

export interface User {
  id: string
  username: string
  created_at: string
}

export interface SearchUser {
  id: string
  username: string
}

export async function getMe(): Promise<User> {
  const { data } = await apiClient.get<User>('/users/me')
  return data
}

export async function searchUsers(q: string): Promise<SearchUser[]> {
  const { data } = await apiClient.get<{ users: SearchUser[] }>('/users/search', { params: { q } })
  return data.users
}
