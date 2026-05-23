import type { ReactNode, CSSProperties } from 'react'
import styles from './GlassPanel.module.css'

interface Props {
  children: ReactNode
  strong?: boolean
  className?: string
  style?: CSSProperties
}

export default function GlassPanel({ children, strong, className = '', style }: Props) {
  return (
    <div className={`${strong ? styles.strong : styles.panel} ${className}`} style={style}>
      {children}
    </div>
  )
}
