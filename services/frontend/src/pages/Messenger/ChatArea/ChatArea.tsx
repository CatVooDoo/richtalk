import { IconSend } from '@tabler/icons-react'
import { useChatStore } from '../../../store/chatStore'
import type { Chat } from '../../../api/chats'
import Avatar from '../../../components/Avatar/Avatar'
import Button from '../../../components/Button/Button'
import styles from './ChatArea.module.css'

function chatDisplayName(chat: Chat): string {
  if (chat.type === 'direct' && chat.other_user) return chat.other_user.username
  return chat.name ?? 'Чат'
}

export default function ChatArea() {
  const { activeChat } = useChatStore()

  if (!activeChat) {
    return (
      <div className={styles.area}>
        <div className={styles.empty}>Выберите чат</div>
      </div>
    )
  }

  const name = chatDisplayName(activeChat)

  return (
    <div className={styles.area}>
      <div className={styles.header}>
        <Avatar username={name} size={36} />
        <span className={styles.chatName}>{name}</span>
        <div className={styles.statusDot} title="Онлайн" />
      </div>

      <div className={styles.messages}>
        <div className={styles.empty}>Нет сообщений. Напишите первым!</div>
      </div>

      <div className={styles.inputRow}>
        <div className={styles.inputWrap}>
          <input
            className={styles.messageInput}
            placeholder="Сообщение..."
            disabled
          />
        </div>
        <Button variant="ghost" disabled aria-label="Отправить">
          <IconSend size={18} stroke={1.8} />
        </Button>
      </div>
    </div>
  )
}
