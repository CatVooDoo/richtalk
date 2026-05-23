import axios from 'axios'
import { apiClient, RT_KEY } from './client'

export interface User {
  id: string
  username: string
  created_at: string
}

export interface AuthResponse {
  access_token: string
  refresh_token: string
  user: User
}

export async function register(username: string, password: string): Promise<AuthResponse> {
  const { data } = await axios.post<AuthResponse>('/api/auth/register', { username, password })
  return data
}

export async function login(username: string, password: string): Promise<AuthResponse> {
  const { data } = await axios.post<AuthResponse>('/api/auth/login', { username, password })
  return data
}

export async function logout(): Promise<void> {
  const rt = localStorage.getItem(RT_KEY)
  if (rt) {
    await apiClient.post('/auth/logout', { refresh_token: rt }).catch(() => {})
    localStorage.removeItem(RT_KEY)
  }
}
