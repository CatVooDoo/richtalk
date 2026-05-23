import { useEffect, useRef, useCallback } from 'react'
import { useAuthStore } from '../store/authStore'
import { useChatStore } from '../store/chatStore'
import { useWsStore, type WsClientEvent } from '../store/wsStore'
import type { Message } from '../api/chats'

const MAX_RETRIES = 5
const RECONNECT_DELAY_MS = 3000

function buildWsUrl(token: string): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/ws?token=${token}`
}

export function useWebSocket() {
  const { isAuthenticated, accessToken } = useAuthStore()
  const { addMessage, updateMessage, markDeleted, markDeletedById, bumpChat, setTyping } =
    useChatStore()
  const { _setConnected, _setSend } = useWsStore()

  const wsRef = useRef<WebSocket | null>(null)
  const retriesRef = useRef(0)

  const sendFn = useCallback((event: WsClientEvent) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(event))
    }
  }, [])

  useEffect(() => { _setSend(sendFn) }, [sendFn, _setSend])

  const connect = useCallback(() => {
    const token = useAuthStore.getState().accessToken
    if (!token) return

    const ws = new WebSocket(buildWsUrl(token))
    wsRef.current = ws

    ws.onopen = () => {
      retriesRef.current = 0
      _setConnected(true)
    }

    ws.onclose = (ev) => {
      _setConnected(false)
      if (ev.code === 4001) return
      if (retriesRef.current < MAX_RETRIES) {
        retriesRef.current += 1
        setTimeout(connect, RECONNECT_DELAY_MS)
      }
    }

    ws.onerror = () => ws.close()

    ws.onmessage = (ev: MessageEvent<string>) => {
      try {
        const msg = JSON.parse(ev.data) as { type: string; payload: unknown }

        switch (msg.type) {
          case 'message.new': {
            const m = msg.payload as Message
            addMessage(m.chat_id, m)
            bumpChat(m.chat_id, m)
            break
          }
          case 'message.edited': {
            const m = msg.payload as Message
            updateMessage(m.chat_id, m)
            break
          }
          case 'message.deleted': {
            const p = msg.payload as { id: string; chat_id?: string }
            if (p.chat_id) markDeleted(p.chat_id, p.id)
            else markDeletedById(p.id)
            break
          }
          case 'typing.start': {
            const p = msg.payload as { chat_id: string; user_id: string }
            setTyping(p.chat_id, p.user_id, true)
            break
          }
          case 'typing.stop': {
            const p = msg.payload as { chat_id: string; user_id: string }
            setTyping(p.chat_id, p.user_id, false)
            break
          }
        }
      } catch {
        // ignore malformed messages
      }
    }
  }, [_setConnected, addMessage, updateMessage, markDeleted, markDeletedById, bumpChat, setTyping])

  useEffect(() => {
    if (!isAuthenticated || !accessToken) return
    connect()
    return () => {
      retriesRef.current = MAX_RETRIES
      wsRef.current?.close()
    }
  }, [isAuthenticated, accessToken, connect])
}
