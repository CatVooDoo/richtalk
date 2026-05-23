import styles from './Avatar.module.css'

interface Props {
  username: string
  size?: number
}

export default function Avatar({ username, size = 36 }: Props) {
  return (
    <div
      className={styles.avatar}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.38) }}
      title={username}
    >
      {username.slice(0, 2).toUpperCase()}
    </div>
  )
}
