import { useEffect, useRef, useState } from 'react'
import { IconSearch, IconLogout, IconX } from '@tabler/icons-react'
import { useChatStore } from '../../../store/chatStore'
import { useAuthStore } from '../../../store/authStore'
import { useAuth } from '../../../hooks/useAuth'
import { getChats, createDirect, getMessages } from '../../../api/chats'
import { searchUsers } from '../../../api/users'
import type { Chat } from '../../../api/chats'
import type { SearchUser } from '../../../api/users'
import Avatar from '../../../components/Avatar/Avatar'
import Button from '../../../components/Button/Button'
import styles from './Sidebar.module.css'

function chatName(chat: Chat): string {
  if (chat.type === 'notes') return 'Заметки'
  if (chat.type === 'direct' && chat.other_user) return chat.other_user.username
  return chat.name ?? 'Чат'
}

function formatTime(dateStr: string): string {
  const d = new Date(dateStr)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const yesterday = new Date(today.getTime() - 86400000)
  const msgDay = new Date(d.getFullYear(), d.getMonth(), d.getDate())

  if (msgDay.getTime() === today.getTime())
    return d.toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
  if (msgDay.getTime() === yesterday.getTime()) return 'Вчера'
  return d.toLocaleDateString('ru', { day: 'numeric', month: 'short' })
}

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(t)
  }, [value, delay])
  return debounced
}

export default function Sidebar() {
  const { user } = useAuthStore()
  const { signOut } = useAuth()
  const { chats, activeChat, setChats, setActiveChat, setMessages, setHasMore } = useChatStore()

  const [query, setQuery] = useState('')
  const [searchResults, setSearchResults] = useState<SearchUser[]>([])
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchLoading, setSearchLoading] = useState(false)
  const searchRef = useRef<HTMLDivElement>(null)
  const debouncedQuery = useDebounce(query, 300)

  // Load chats on mount
  useEffect(() => {
    getChats().then(setChats).catch(console.error)
  }, [setChats])

  // Search users
  useEffect(() => {
    if (debouncedQuery.length < 2) {
      setSearchResults([])
      return
    }
    setSearchLoading(true)
    searchUsers(debouncedQuery)
      .then(setSearchResults)
      .catch(console.error)
      .finally(() => setSearchLoading(false))
  }, [debouncedQuery])

  // Close search on outside click
  useEffect(() => {
    if (!searchOpen) return
    const handler = (e: MouseEvent) => {
      if (searchRef.current && !searchRef.current.contains(e.target as Node)) {
        closeSearch()
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [searchOpen])

  const closeSearch = () => {
    setSearchOpen(false)
    setQuery('')
    setSearchResults([])
  }

  const handleSelectChat = async (chat: Chat) => {
    setActiveChat(chat)
    if (!useChatStore.getState().messages[chat.id]?.length) {
      try {
        const page = await getMessages(chat.id)
        setMessages(chat.id, [...page.messages].reverse())
        setHasMore(chat.id, page.has_more)
      } catch (err) {
        console.error(err)
      }
    }
  }

  const handleSelectUser = async (u: SearchUser) => {
    if (u.id === user?.id) return
    closeSearch()
    try {
      const chat = await createDirect(u.id)
      const existingChats = useChatStore.getState().chats
      if (!existingChats.find((c) => c.id === chat.id)) {
        setChats([chat, ...existingChats])
      }
      handleSelectChat(chat)
    } catch (err) {
      console.error(err)
    }
  }

  const showUnread = (chat: Chat) =>
    chat.last_message && chat.last_message.author_id !== user?.id

  return (
    <div className={styles.sidebar}>
      <div className={styles.header}>
        <span className={styles.title}>RichTalk</span>
        <div className={styles.headerActions}>
          {searchOpen ? (
            <Button variant="icon" onClick={closeSearch} aria-label="Закрыть поиск">
              <IconX size={16} stroke={1.8} />
            </Button>
          ) : (
            <Button variant="icon" onClick={() => setSearchOpen(true)} aria-label="Поиск">
              <IconSearch size={16} stroke={1.8} />
            </Button>
          )}
        </div>
      </div>

      {searchOpen && (
        <div className={styles.searchWrap} ref={searchRef}>
          <input
            className={styles.searchInput}
            placeholder="Поиск пользователей..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
          {(searchResults.length > 0 || searchLoading) && (
            <div className={`${styles.searchDropdown} glass-strong`}>
              {searchLoading && (
                <div className={styles.searchHint}>Поиск...</div>
              )}
              {!searchLoading && searchResults.length === 0 && query.length >= 2 && (
                <div className={styles.searchHint}>Не найдено</div>
              )}
              {searchResults.map((u) => (
                <button
                  key={u.id}
                  className={styles.searchItem}
                  onClick={() => handleSelectUser(u)}
                >
                  <Avatar username={u.username} size={32} />
                  <span className={styles.searchName}>{u.username}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      )}

      <div className={styles.chatList}>
        {chats.length === 0 ? (
          <div className={styles.emptyChats}>Нет чатов. Найдите людей через поиск!</div>
        ) : (
          chats.map((chat) => {
            const name = chatName(chat)
            const isActive = activeChat?.id === chat.id
            const isNotes = chat.type === 'notes'
            const hasUnread = showUnread(chat)

            return (
              <div
                key={chat.id}
                className={`${styles.chatItem} ${isActive ? styles.chatItemActive : ''}`}
                onClick={() => handleSelectChat(chat)}
              >
                {isNotes ? (
                  <div className={styles.notesIcon}>📝</div>
                ) : (
                  <Avatar username={name} size={40} />
                )}
                <div className={styles.chatInfo}>
                  <div className={styles.chatNameRow}>
                    <span className={styles.chatName}>{name}</span>
                    {chat.last_message && (
                      <span className={styles.chatTime}>
                        {formatTime(chat.last_message.created_at)}
                      </span>
                    )}
                  </div>
                  <div className={styles.chatSubRow}>
                    <span className={`${styles.lastMsg} ${chat.last_message?.deleted ? styles.lastMsgDeleted : ''}`}>
                      {chat.last_message
                        ? chat.last_message.deleted
                          ? 'Сообщение удалено'
                          : chat.last_message.content.slice(0, 35) +
                            (chat.last_message.content.length > 35 ? '…' : '')
                        : isNotes
                        ? 'Ваши заметки'
                        : 'Нет сообщений'}
                    </span>
                    {hasUnread && !isActive && <span className={styles.badge} />}
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      <div className={styles.footer}>
        <div className={styles.footerInner}>
          <div className={styles.footerUser}>
            <Avatar username={user?.username ?? '?'} size={30} />
            <span className={styles.username}>{user?.username}</span>
          </div>
          <Button variant="icon" title="Выйти" onClick={signOut} aria-label="Выйти">
            <IconLogout size={15} stroke={1.8} />
          </Button>
        </div>
      </div>
    </div>
  )
}
