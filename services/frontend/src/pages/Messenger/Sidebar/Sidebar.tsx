import { IconSearch, IconLogout } from '@tabler/icons-react'
import { useChatStore } from '../../../store/chatStore'
import { useAuth } from '../../../hooks/useAuth'
import type { Chat } from '../../../api/chats'
import Avatar from '../../../components/Avatar/Avatar'
import Button from '../../../components/Button/Button'
import styles from './Sidebar.module.css'

function chatDisplayName(chat: Chat): string {
  if (chat.type === 'direct' && chat.other_user) return chat.other_user.username
  return chat.name ?? 'Чат'
}

export default function Sidebar() {
  const { user, signOut } = useAuth()
  const { chats, activeChat, setActiveChat } = useChatStore()

  return (
    <div className={styles.sidebar}>
      <div className={styles.header}>
        <span className={styles.title}>RichTalk</span>
        <Button variant="ghost" title="Поиск" aria-label="Поиск пользователей">
          <IconSearch size={18} stroke={1.8} />
        </Button>
      </div>

      <div className={styles.chatList}>
        {chats.length === 0 ? (
          <div className={styles.emptyChats}>Нет чатов. Начните разговор!</div>
        ) : (
          chats.map((chat) => {
            const name = chatDisplayName(chat)
            const isActive = activeChat?.id === chat.id
            return (
              <div
                key={chat.id}
                className={`${styles.chatItem} ${isActive ? styles.chatItemActive : ''}`}
                onClick={() => setActiveChat(chat)}
              >
                <Avatar username={name} size={38} />
                <div className={styles.chatInfo}>
                  <div className={styles.chatName}>{name}</div>
                  {chat.last_message && (
                    <div className={styles.lastMsg}>
                      {chat.last_message.deleted
                        ? 'Сообщение удалено'
                        : chat.last_message.content}
                    </div>
                  )}
                </div>
              </div>
            )
          })
        )}
      </div>

      <div className={styles.footer}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <span className={styles.username}>{user?.username}</span>
          <Button variant="ghost" title="Выйти" onClick={signOut} aria-label="Выйти">
            <IconLogout size={16} stroke={1.8} />
          </Button>
        </div>
      </div>
    </div>
  )
}
