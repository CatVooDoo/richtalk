import { useEffect } from 'react'
import { useChatStore } from '../../store/chatStore'
import { getChats } from '../../api/chats'
import { useWebSocket } from '../../hooks/useWebSocket'
import Sidebar from './Sidebar/Sidebar'
import ChatArea from './ChatArea/ChatArea'
import styles from './Messenger.module.css'

export default function MessengerPage() {
  useWebSocket()

  const setChats = useChatStore((s) => s.setChats)

  useEffect(() => {
    getChats().then(setChats).catch(console.error)
  }, [setChats])

  return (
    <div className={styles.page}>
      <div className={styles.orb1} />
      <div className={styles.orb2} />
      <div className={styles.content}>
        <Sidebar />
        <ChatArea />
      </div>
    </div>
  )
}
