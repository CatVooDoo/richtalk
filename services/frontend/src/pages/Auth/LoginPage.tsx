import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { login } from '../../api/auth'
import { useAuthStore } from '../../store/authStore'
import { RT_KEY } from '../../api/client'
import Input from '../../components/Input/Input'
import Button from '../../components/Button/Button'
import styles from './Auth.module.css'

export default function LoginPage() {
  const navigate = useNavigate()
  const setAuth = useAuthStore((s) => s.setAuth)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const data = await login(username.trim(), password)
      localStorage.setItem(RT_KEY, data.refresh_token)
      setAuth(data.user, data.access_token)
      navigate('/')
    } catch (err: unknown) {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data?.message ??
        'Ошибка входа. Попробуйте снова.'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.orb1} />
      <div className={styles.orb2} />
      <div className={styles.orb3} />

      <div className={`${styles.card} glass-strong`}>
        <div className={styles.cardHeader}>
          <div className={styles.logo}>RichTalk</div>
          <p className={styles.subtitle}>Войдите в аккаунт</p>
        </div>

        <form className={styles.form} onSubmit={handleSubmit} noValidate>
          <Input
            id="username"
            label="Имя пользователя"
            placeholder="username"
            autoComplete="username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            required
          />
          <Input
            id="password"
            label="Пароль"
            type="password"
            placeholder="••••••••"
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
          />
          <Button type="submit" disabled={loading} style={{ width: '100%', marginTop: 4 }}>
            {loading ? 'Входим...' : 'Войти'}
          </Button>
        </form>

        <p className={styles.switchText}>
          Нет аккаунта?{' '}
          <Link to="/register" className={styles.switchLink}>
            Зарегистрироваться
          </Link>
        </p>
      </div>

      {error && <div className={styles.toast}>{error}</div>}
    </div>
  )
}
