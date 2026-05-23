import { create } from 'zustand'

export interface WsClientEvent {
  type: 'typing.start' | 'typing.stop'
  payload: { chat_id: string }
}

interface WsState {
  connected: boolean
  send: (event: WsClientEvent) => void
  _setConnected: (v: boolean) => void
  _setSend: (fn: (event: WsClientEvent) => void) => void
}

const noop = () => {}

export const useWsStore = create<WsState>((set) => ({
  connected: false,
  send: noop,
  _setConnected: (connected) => set({ connected }),
  _setSend: (send) => set({ send }),
}))
