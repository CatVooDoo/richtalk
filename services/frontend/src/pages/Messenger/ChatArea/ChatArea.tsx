import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type ChangeEvent,
} from 'react'
import { IconSend, IconPaperclip, IconX, IconPencil, IconTrash, IconCopy } from '@tabler/icons-react'
import { useChatStore } from '../../../store/chatStore'
import { useAuthStore } from '../../../store/authStore'
import { useWsStore } from '../../../store/wsStore'
import {
  getMessages,
  sendMessage as apiSend,
  editMessage as apiEdit,
  deleteMessage as apiDelete,
} from '../../../api/chats'
import type { Chat, Message } from '../../../api/chats'
import Avatar from '../../../components/Avatar/Avatar'
import Button from '../../../components/Button/Button'
import styles from './ChatArea.module.css'

// ── helpers ──────────────────────────────────────────────────────────────────

function chatDisplayName(chat: Chat): string {
  if (chat.type === 'notes') return 'Заметки'
  if (chat.type === 'direct' && chat.other_user) return chat.other_user.username
  return chat.name ?? 'Чат'
}

function getDayLabel(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 86400000)
  const msgDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  if (msgDay.getTime() === today.getTime()) return 'Сегодня'
  if (msgDay.getTime() === yesterday.getTime()) return 'Вчера'
  return d.toLocaleDateString('ru', { day: 'numeric', month: 'long' })
}

function formatMsgTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}

function isEdited(msg: Message): boolean {
  return new Date(msg.updated_at).getTime() - new Date(msg.created_at).getTime() > 1000
}

// ── main component ────────────────────────────────────────────────────────────

export default function ChatArea() {
  const { activeChat, messages, hasMore, typingUsers, setMessages, setHasMore, addMessage,
          updateMessage, markDeleted, bumpChat } = useChatStore()
  const { user } = useAuthStore()
  const { send, connected } = useWsStore()

  const chatMessages = activeChat ? (messages[activeChat.id] ?? []) : []
  const typingList = activeChat ? (typingUsers[activeChat.id] ?? []) : []
  const isNotes = activeChat?.type === 'notes'

  // refs
  const scrollRef = useRef<HTMLDivElement>(null)
  const endRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const savedScrollHeightRef = useRef(0)
  const shouldRestoreScrollRef = useRef(false)
  const isTypingRef = useRef(false)
  const typingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // local state
  const [text, setText] = useState('')
  const [editing, setEditing] = useState<Message | null>(null)
  const [contextMenu, setContextMenu] = useState<{ msg: Message; x: number; y: number } | null>(null)
  const [isAtBottom, setIsAtBottom] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [sending, setSending] = useState(false)
  const [toastMsg, setToastMsg] = useState<string | null>(null)

  // Load messages when active chat changes
  useEffect(() => {
    if (!activeChat) return
    if (messages[activeChat.id]?.length) return // already loaded
    getMessages(activeChat.id)
      .then((page) => {
        setMessages(activeChat.id, [...page.messages].reverse())
        setHasMore(activeChat.id, page.has_more)
        setTimeout(() => {
          if (scrollRef.current) scrollRef.current.scrollTop = scrollRef.current.scrollHeight
        }, 30)
      })
      .catch(console.error)
  }, [activeChat?.id]) // eslint-disable-line react-hooks/exhaustive-deps

  // Scroll to bottom on new message if already at bottom
  useEffect(() => {
    if (isAtBottom && chatMessages.length > 0) {
      endRef.current?.scrollIntoView({ behavior: 'smooth' })
    }
  }, [chatMessages.length]) // eslint-disable-line react-hooks/exhaustive-deps

  // Restore scroll position after prepend
  useLayoutEffect(() => {
    if (shouldRestoreScrollRef.current && scrollRef.current) {
      scrollRef.current.scrollTop =
        scrollRef.current.scrollHeight - savedScrollHeightRef.current
      shouldRestoreScrollRef.current = false
    }
  })

  // Close context menu on outside click
  useEffect(() => {
    if (!contextMenu) return
    const handler = () => setContextMenu(null)
    document.addEventListener('click', handler)
    return () => document.removeEventListener('click', handler)
  }, [contextMenu])

  // Toast auto-dismiss
  useEffect(() => {
    if (!toastMsg) return
    const t = setTimeout(() => setToastMsg(null), 2500)
    return () => clearTimeout(t)
  }, [toastMsg])

  // ── handlers ──────────────────────────────────────────────────────────────

  const handleScroll = () => {
    if (!scrollRef.current) return
    const { scrollTop, scrollHeight, clientHeight } = scrollRef.current
    setIsAtBottom(scrollHeight - scrollTop - clientHeight < 80)
    if (scrollTop < 120 && !loadingMore && activeChat && hasMore[activeChat.id]) {
      loadOlder()
    }
  }

  const loadOlder = async () => {
    if (!activeChat || loadingMore) return
    const oldest = chatMessages[0]
    if (!oldest) return

    setLoadingMore(true)
    savedScrollHeightRef.current = scrollRef.current?.scrollHeight ?? 0
    shouldRestoreScrollRef.current = true

    try {
      const page = await getMessages(activeChat.id, oldest.created_at)
      const older = [...page.messages].reverse()
      setMessages(activeChat.id, older, true)
      setHasMore(activeChat.id, page.has_more)
    } catch (err) {
      console.error(err)
    } finally {
      setLoadingMore(false)
    }
  }

  const handleTextChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    setText(e.target.value)
    const ta = e.target
    ta.style.height = 'auto'
    ta.style.height = Math.min(ta.scrollHeight, 120) + 'px'

    if (activeChat && connected) {
      if (!isTypingRef.current) {
        isTypingRef.current = true
        send({ type: 'typing.start', payload: { chat_id: activeChat.id } })
      }
      if (typingTimerRef.current) clearTimeout(typingTimerRef.current)
      typingTimerRef.current = setTimeout(() => {
        isTypingRef.current = false
        send({ type: 'typing.stop', payload: { chat_id: activeChat.id } })
      }, 3000)
    }
  }

  const stopTyping = () => {
    if (isTypingRef.current && activeChat && connected) {
      isTypingRef.current = false
      if (typingTimerRef.current) clearTimeout(typingTimerRef.current)
      send({ type: 'typing.stop', payload: { chat_id: activeChat.id } })
    }
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
    if (e.key === 'Escape' && editing) {
      cancelEditing()
    }
  }

  const handleSend = async () => {
    const content = text.trim()
    if (!content || !activeChat || sending) return
    stopTyping()
    setSending(true)

    try {
      if (editing) {
        const updated = await apiEdit(editing.id, content)
        updateMessage(activeChat.id, updated)
        cancelEditing()
      } else {
        const msg = await apiSend(activeChat.id, content)
        addMessage(activeChat.id, msg)
        bumpChat(activeChat.id, msg)
        setText('')
        if (textareaRef.current) {
          textareaRef.current.style.height = 'auto'
        }
        endRef.current?.scrollIntoView({ behavior: 'smooth' })
      }
    } catch (err) {
      console.error(err)
    } finally {
      setSending(false)
    }
  }

  const cancelEditing = () => {
    setEditing(null)
    setText('')
    if (textareaRef.current) textareaRef.current.style.height = 'auto'
    textareaRef.current?.focus()
  }

  const handleDelete = async (msg: Message) => {
    if (!activeChat) return
    try {
      await apiDelete(msg.id)
      markDeleted(activeChat.id, msg.id)
    } catch (err) {
      console.error(err)
    }
  }

  const handleCopy = (msg: Message) => {
    navigator.clipboard.writeText(msg.content).then(() => setToastMsg('Скопировано'))
  }

  const handleContextMenu = (e: React.MouseEvent, msg: Message) => {
    e.preventDefault()
    if (msg.deleted) return
    setContextMenu({ msg, x: e.clientX, y: e.clientY })
  }

  // ── render ────────────────────────────────────────────────────────────────

  if (!activeChat) {
    return (
      <div className={styles.area}>
        <div className={styles.empty}>Выберите чат для начала общения</div>
      </div>
    )
  }

  const name = chatDisplayName(activeChat)

  // Group messages by day
  const items: Array<{ type: 'divider'; label: string } | { type: 'msg'; msg: Message }> = []
  let lastLabel = ''
  for (const msg of chatMessages) {
    const label = getDayLabel(msg.created_at)
    if (label !== lastLabel) {
      items.push({ type: 'divider', label })
      lastLabel = label
    }
    items.push({ type: 'msg', msg })
  }

  return (
    <div className={styles.area}>
      {/* Header */}
      <div className={styles.header}>
        {isNotes ? (
          <div className={styles.notesIconHeader}>📝</div>
        ) : (
          <Avatar username={name} size={36} />
        )}
        <div className={styles.headerInfo}>
          <span className={styles.chatName}>{name}</span>
          {isNotes ? (
            <span className={styles.chatSub}>Только для вас</span>
          ) : typingList.length > 0 ? (
            <span className={styles.typingLabel}>печатает...</span>
          ) : null}
        </div>
      </div>

      {/* Messages */}
      <div className={styles.messages} ref={scrollRef} onScroll={handleScroll}>
        {loadingMore && <div className={styles.loadingMore}>Загрузка...</div>}

        {items.map((item, i) => {
          if (item.type === 'divider') {
            return (
              <div key={`div-${i}`} className={styles.dayDivider}>
                <span>{item.label}</span>
              </div>
            )
          }

          const msg = item.msg
          const isMine = isNotes || msg.author.id === user?.id

          return (
            <div
              key={msg.id}
              className={`${styles.msgRow} ${isMine ? styles.msgRowRight : styles.msgRowLeft}`}
              onContextMenu={(e) => handleContextMenu(e, msg)}
            >
              {!isMine && <Avatar username={msg.author.username} size={28} />}
              <div
                className={`${styles.bubble} ${isMine ? styles.bubbleOut : styles.bubbleIn} ${msg.deleted ? styles.bubbleDeleted : ''}`}
              >
                {msg.deleted ? (
                  <span className={styles.deletedText}>Сообщение удалено</span>
                ) : (
                  <>
                    <span className={styles.msgContent}>{msg.content}</span>
                    {msg.attachment_type && (
                      <span className={styles.attachStub}>📎 Файлы скоро</span>
                    )}
                    <span className={styles.msgMeta}>
                      {formatMsgTime(msg.created_at)}
                      {isEdited(msg) && <span className={styles.editedTag}> изм.</span>}
                    </span>
                  </>
                )}
              </div>
            </div>
          )
        })}

        {/* Typing indicator */}
        {typingList.length > 0 && !isNotes && (
          <div className={styles.typingRow}>
            <div className={styles.typingBubble}>
              <span className={styles.dot} />
              <span className={styles.dot} />
              <span className={styles.dot} />
            </div>
          </div>
        )}

        <div ref={endRef} />
      </div>

      {/* Edit bar */}
      {editing && (
        <div className={`${styles.editBar} glass`}>
          <div className={styles.editBarInfo}>
            <IconPencil size={14} className={styles.editBarIcon} />
            <span className={styles.editBarText}>{editing.content}</span>
          </div>
          <button className={styles.editBarClose} onClick={cancelEditing} aria-label="Отменить">
            <IconX size={14} />
          </button>
        </div>
      )}

      {/* Input */}
      <div className={styles.inputRow}>
        <Button
          variant="ghost"
          className={styles.attachBtn}
          onClick={() => setToastMsg('Загрузка файлов — скоро!')}
          aria-label="Прикрепить файл"
        >
          <IconPaperclip size={18} stroke={1.8} />
        </Button>

        <div className={styles.inputWrap}>
          <textarea
            ref={textareaRef}
            className={styles.messageInput}
            placeholder={editing ? 'Редактирование...' : 'Сообщение...'}
            value={text}
            rows={1}
            onChange={handleTextChange}
            onKeyDown={handleKeyDown}
          />
        </div>

        <Button
          variant="ghost"
          className={`${styles.sendBtn} ${text.trim() ? styles.sendBtnActive : ''}`}
          disabled={!text.trim() || sending}
          onClick={handleSend}
          aria-label="Отправить"
        >
          <IconSend size={18} stroke={1.8} />
        </Button>
      </div>

      {/* Context menu */}
      {contextMenu && (
        <div
          className={`${styles.contextMenu} glass-strong`}
          style={{
            left: Math.min(contextMenu.x, window.innerWidth - 170),
            top: Math.min(contextMenu.y, window.innerHeight - 130),
          }}
          onClick={(e) => e.stopPropagation()}
        >
          <button
            className={styles.ctxItem}
            onClick={() => { handleCopy(contextMenu.msg); setContextMenu(null) }}
          >
            <IconCopy size={14} /> Копировать
          </button>
          {contextMenu.msg.author.id === user?.id && (
            <>
              <button
                className={styles.ctxItem}
                onClick={() => {
                  setEditing(contextMenu.msg)
                  setText(contextMenu.msg.content)
                  textareaRef.current?.focus()
                  setContextMenu(null)
                }}
              >
                <IconPencil size={14} /> Редактировать
              </button>
              <button
                className={`${styles.ctxItem} ${styles.ctxItemDanger}`}
                onClick={() => { handleDelete(contextMenu.msg); setContextMenu(null) }}
              >
                <IconTrash size={14} /> Удалить
              </button>
            </>
          )}
        </div>
      )}

      {/* Toast */}
      {toastMsg && <div className={styles.toast}>{toastMsg}</div>}
    </div>
  )
}
