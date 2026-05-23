import { type ReactNode, useEffect, useState } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuthStore } from '../../store/authStore'
import { getMe } from '../../api/users'
import { RT_KEY } from '../../api/client'
import styles from './PrivateRoute.module.css'

interface Props {
  children: ReactNode
}

export default function PrivateRoute({ children }: Props) {
  const { user, isAuthenticated, setUser, logout } = useAuthStore()
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    if (user) {
      setChecking(false)
      return
    }

    const hasRT = Boolean(localStorage.getItem(RT_KEY))
    if (!hasRT) {
      setChecking(false)
      return
    }

    getMe()
      .then((u) => setUser(u))
      .catch(() => logout())
      .finally(() => setChecking(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  if (checking) {
    return (
      <div className={styles.spinner}>
        <div className={styles.ring} />
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}
