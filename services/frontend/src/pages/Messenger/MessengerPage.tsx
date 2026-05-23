import { useState } from 'react'
import { useWebSocket } from '../../hooks/useWebSocket'
import Sidebar from './Sidebar/Sidebar'
import ChatArea from './ChatArea/ChatArea'
import styles from './Messenger.module.css'

export default function MessengerPage() {
  useWebSocket()
  const [mobilePanel, setMobilePanel] = useState<'sidebar' | 'chat'>('sidebar')

  return (
    <div className={styles.page}>
      <div className={styles.orb1} />
      <div className={styles.orb2} />
      <div className={styles.content}>
        <Sidebar
          mobileHidden={mobilePanel === 'chat'}
          onChatSelect={() => setMobilePanel('chat')}
        />
        <ChatArea
          mobileHidden={mobilePanel === 'sidebar'}
          onBack={() => setMobilePanel('sidebar')}
        />
      </div>
    </div>
  )
}
