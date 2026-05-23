import { apiClient } from './client'

export interface OtherUser {
  id: string
  username: string
}

export interface LastMessage {
  id: string
  content: string
  author_id: string
  created_at: string
  deleted: boolean
}

export interface Chat {
  id: string
  type: 'direct' | 'group' | 'channel'
  name?: string
  other_user?: OtherUser
  last_message?: LastMessage
  created_at: string
}

export interface MessageAuthor {
  id: string
  username: string
}

export interface Message {
  id: string
  chat_id: string
  author: MessageAuthor
  content: string
  deleted: boolean
  created_at: string
  updated_at: string
}

export interface MessagesPage {
  messages: Message[]
  has_more: boolean
  next_cursor: string | null
}

export async function getChats(): Promise<Chat[]> {
  const { data } = await apiClient.get<{ chats: Chat[] }>('/chats')
  return data.chats
}

export async function getChat(chatId: string): Promise<Chat> {
  const { data } = await apiClient.get<Chat>(`/chats/${chatId}`)
  return data
}

export async function createDirect(userId: string): Promise<Chat> {
  const { data } = await apiClient.post<Chat>('/chats/direct', { user_id: userId })
  return data
}

export async function getMessages(
  chatId: string,
  before?: string,
  limit = 50
): Promise<MessagesPage> {
  const params: Record<string, string | number> = { limit }
  if (before) params.before = before
  const { data } = await apiClient.get<MessagesPage>(`/chats/${chatId}/messages`, { params })
  return data
}

export async function sendMessage(chatId: string, content: string): Promise<Message> {
  const { data } = await apiClient.post<Message>(`/chats/${chatId}/messages`, { content })
  return data
}

export async function editMessage(messageId: string, content: string): Promise<Message> {
  const { data } = await apiClient.patch<Message>(`/messages/${messageId}`, { content })
  return data
}

export async function deleteMessage(messageId: string): Promise<void> {
  await apiClient.delete(`/messages/${messageId}`)
}
