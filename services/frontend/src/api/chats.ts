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
  type: 'direct' | 'group' | 'channel' | 'notes'
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
  attachment_type?: 'image' | 'audio' | null
  attachment_url?: string | null
  attachment_name?: string | null
  attachment_size?: number | null
  created_at: string
  updated_at: string
}

export interface MessagesPage {
  messages: Message[]
  has_more: boolean
  next_cursor: string | null
}

export interface MessageAttachment {
  url: string
  type: 'image' | 'audio'
  name: string
  size: number
}

export interface UploadResult {
  url: string
  type: 'image' | 'audio'
  name: string
  size: number
}

export async function getChats(): Promise<Chat[]> {
  const { data } = await apiClient.get<{ chats: Chat[] }>('/chats')
  return data.chats
}

export async function getNotesChat(): Promise<Chat> {
  const { data } = await apiClient.get<Chat>('/chats/notes')
  return data
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

export async function sendMessage(
  chatId: string,
  content: string,
  attachment?: MessageAttachment
): Promise<Message> {
  const body: Record<string, unknown> = { content }
  if (attachment) {
    body.attachment_url  = attachment.url
    body.attachment_type = attachment.type
    body.attachment_name = attachment.name
    body.attachment_size = attachment.size
  }
  const { data } = await apiClient.post<Message>(`/chats/${chatId}/messages`, body)
  return data
}

export async function editMessage(messageId: string, content: string): Promise<Message> {
  const { data } = await apiClient.patch<Message>(`/messages/${messageId}`, { content })
  return data
}

export async function deleteMessage(messageId: string): Promise<void> {
  await apiClient.delete(`/messages/${messageId}`)
}

export async function uploadFile(file: File | Blob, filename?: string): Promise<UploadResult> {
  const form = new FormData()
  // For Blob (MediaRecorder output), pass explicit filename with extension
  form.append('file', file, filename ?? (file instanceof File ? file.name : 'file'))
  const { data } = await apiClient.post<UploadResult>('/uploads', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return data
}
