import styles from './Avatar.module.css'

interface Props {
  username: string
  size?: number
}

const GRADIENTS = [
  ['#a855f7', '#6366f1'],
  ['#ec4899', '#8b5cf6'],
  ['#06b6d4', '#6366f1'],
  ['#10b981', '#06b6d4'],
  ['#f59e0b', '#ef4444'],
  ['#3b82f6', '#8b5cf6'],
]

function hashIndex(str: string): number {
  let h = 0
  for (let i = 0; i < str.length; i++) {
    h = str.charCodeAt(i) + ((h << 5) - h)
  }
  return Math.abs(h) % GRADIENTS.length
}

export default function Avatar({ username, size = 36 }: Props) {
  const [from, to] = GRADIENTS[hashIndex(username)]
  return (
    <div
      className={styles.avatar}
      style={{
        width: size,
        height: size,
        fontSize: Math.round(size * 0.38),
        background: `linear-gradient(135deg, ${from}, ${to})`,
      }}
      title={username}
    >
      {username.slice(0, 2).toUpperCase()}
    </div>
  )
}
