import { useAuthStore } from '../store/authStore'
import { logout as apiLogout } from '../api/auth'

export function useAuth() {
  const { user, accessToken, isAuthenticated, logout } = useAuthStore()

  const signOut = async () => {
    await apiLogout()
    logout()
  }

  return { user, accessToken, isAuthenticated, signOut }
}
