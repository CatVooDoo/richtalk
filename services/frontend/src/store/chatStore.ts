import { create } from 'zustand'
import type { Chat, Message } from '../api/chats'

interface ChatState {
  chats: Chat[]
  activeChat: Chat | null
  messages: Record<string, Message[]>   // chatId → messages (oldest first)
  hasMore: Record<string, boolean>
  typingUsers: Record<string, string[]> // chatId → userId[]

  setChats(chats: Chat[]): void
  addOrUpdateChat(chat: Chat): void
  bumpChat(chatId: string, msg: Message): void
  setActiveChat(chat: Chat | null): void
  setMessages(chatId: string, msgs: Message[], prepend?: boolean): void
  addMessage(chatId: string, msg: Message): void
  updateMessage(chatId: string, msg: Message): void
  markDeleted(chatId: string, msgId: string): void
  markDeletedById(msgId: string): void
  setHasMore(chatId: string, v: boolean): void
  setTyping(chatId: string, userId: string, isTyping: boolean): void
}

export const useChatStore = create<ChatState>((set) => ({
  chats: [],
  activeChat: null,
  messages: {},
  hasMore: {},
  typingUsers: {},

  setChats: (chats) => set({ chats }),

  addOrUpdateChat: (chat) =>
    set((s) => {
      const idx = s.chats.findIndex((c) => c.id === chat.id)
      if (idx >= 0) {
        const updated = [...s.chats]
        updated[idx] = chat
        return { chats: updated }
      }
      return { chats: [chat, ...s.chats] }
    }),

  bumpChat: (chatId, msg) =>
    set((s) => ({
      chats: s.chats.map((c) =>
        c.id !== chatId
          ? c
          : {
              ...c,
              last_message: {
                id: msg.id,
                content: msg.deleted ? '' : msg.content,
                author_id: msg.author.id,
                created_at: msg.created_at,
                deleted: msg.deleted,
              },
            }
      ),
    })),

  setActiveChat: (activeChat) => set({ activeChat }),

  setMessages: (chatId, msgs, prepend) =>
    set((s) => {
      const existing = s.messages[chatId] ?? []
      const updated = prepend ? [...msgs, ...existing] : msgs
      return { messages: { ...s.messages, [chatId]: updated } }
    }),

  addMessage: (chatId, msg) =>
    set((s) => {
      const existing = s.messages[chatId] ?? []
      if (existing.some((m) => m.id === msg.id)) return s // dedup from WS + HTTP
      return { messages: { ...s.messages, [chatId]: [...existing, msg] } }
    }),

  updateMessage: (chatId, msg) =>
    set((s) => ({
      messages: {
        ...s.messages,
        [chatId]: (s.messages[chatId] ?? []).map((m) => (m.id === msg.id ? msg : m)),
      },
    })),

  markDeleted: (chatId, msgId) =>
    set((s) => ({
      messages: {
        ...s.messages,
        [chatId]: (s.messages[chatId] ?? []).map((m) =>
          m.id === msgId ? { ...m, deleted: true, content: '' } : m
        ),
      },
    })),

  markDeletedById: (msgId) =>
    set((s) => {
      const newMessages = { ...s.messages }
      for (const [chatId, msgs] of Object.entries(newMessages)) {
        if (msgs.some((m) => m.id === msgId)) {
          newMessages[chatId] = msgs.map((m) =>
            m.id === msgId ? { ...m, deleted: true, content: '' } : m
          )
          break
        }
      }
      return { messages: newMessages }
    }),

  setHasMore: (chatId, v) =>
    set((s) => ({ hasMore: { ...s.hasMore, [chatId]: v } })),

  setTyping: (chatId, userId, isTyping) =>
    set((s) => {
      const current = s.typingUsers[chatId] ?? []
      const next = isTyping
        ? current.includes(userId) ? current : [...current, userId]
        : current.filter((id) => id !== userId)
      return { typingUsers: { ...s.typingUsers, [chatId]: next } }
    }),
}))
