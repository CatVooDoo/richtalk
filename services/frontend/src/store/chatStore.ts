import { create } from 'zustand'
import type { Chat, Message } from '../api/chats'

interface ChatState {
  chats: Chat[]
  activeChat: Chat | null
  messages: Message[]
  typingUsers: Set<string>

  setChats: (chats: Chat[]) => void
  addOrUpdateChat: (chat: Chat) => void
  setActiveChat: (chat: Chat | null) => void
  setMessages: (messages: Message[]) => void
  prependMessages: (older: Message[]) => void
  addMessage: (msg: Message) => void
  updateMessage: (msg: Message) => void
  markMessageDeleted: (msgId: string) => void
  setTyping: (userId: string, typing: boolean) => void
}

export const useChatStore = create<ChatState>((set) => ({
  chats: [],
  activeChat: null,
  messages: [],
  typingUsers: new Set(),

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

  setActiveChat: (activeChat) =>
    set({ activeChat, messages: [], typingUsers: new Set() }),

  setMessages: (messages) => set({ messages }),

  prependMessages: (older) =>
    set((s) => ({ messages: [...older, ...s.messages] })),

  addMessage: (msg) =>
    set((s) => ({ messages: [...s.messages, msg] })),

  updateMessage: (msg) =>
    set((s) => ({
      messages: s.messages.map((m) => (m.id === msg.id ? msg : m)),
    })),

  markMessageDeleted: (msgId) =>
    set((s) => ({
      messages: s.messages.map((m) =>
        m.id === msgId ? { ...m, deleted: true, content: '' } : m
      ),
    })),

  setTyping: (userId, typing) =>
    set((s) => {
      const next = new Set(s.typingUsers)
      typing ? next.add(userId) : next.delete(userId)
      return { typingUsers: next }
    }),
}))
