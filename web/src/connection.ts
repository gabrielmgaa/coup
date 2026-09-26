import { useEffect, useRef, useState } from 'react'
import {
  forgetSession,
  loadSession,
  roomAddress,
  saveSession,
  type FromClient,
  type FromServer,
  type GameEvent,
  type GameState,
  type LobbyView,
} from './coup'

const retryDelayMs = 1000
const maxRetries = 30
const lostSessionCodes = ['invalid_token', 'room_not_found']

export type Table = {
  lobby: LobbyView | null
  game: GameState | null
  log: GameEvent[]
  refusal: string | null
  reconnecting: boolean
  deadline: number
  enter: (first: FromClient) => void
  send: (message: FromClient) => void
}

export function useTable(): Table {
  const [lobby, setLobby] = useState<LobbyView | null>(null)
  const [game, setGame] = useState<GameState | null>(null)
  const [log, setLog] = useState<GameEvent[]>([])
  const [refusal, setRefusal] = useState<string | null>(null)
  const [reconnecting, setReconnecting] = useState(false)
  const [deadline, setDeadline] = useState(0)
  const socket = useRef<WebSocket | null>(null)
  const retries = useRef(0)

  function receive(message: FromServer) {
    if (message.type === 'welcome') {
      saveSession({ room: message.room, token: message.token })
      retries.current = 0
      setReconnecting(false)
      return
    }
    if (message.type === 'error') {
      if (lostSessionCodes.includes(message.code)) forgetSession()
      setRefusal(message.message)
      return
    }
    setRefusal(null)
    if (message.type === 'lobby') {
      setLobby(message.state)
      setGame(null)
      setLog([])
      return
    }
    setGame(message.state)
    setDeadline(message.state.closes_in_ms ? Date.now() + message.state.closes_in_ms : 0)
    setLog((previous) => [...previous, ...message.events])
  }

  function retry() {
    const saved = loadSession()
    if (!saved || retries.current >= maxRetries) {
      setReconnecting(false)
      return
    }
    retries.current += 1
    setReconnecting(true)
    setTimeout(() => enter({ type: 'reconnect', ...saved }), retryDelayMs)
  }

  function enter(first: FromClient) {
    const opened = new WebSocket(roomAddress())
    opened.onopen = () => opened.send(JSON.stringify(first))
    opened.onmessage = (incoming) => receive(JSON.parse(incoming.data) as FromServer)
    opened.onclose = () => {
      if (socket.current === opened) retry()
    }
    socket.current = opened
  }

  function send(message: FromClient) {
    socket.current?.send(JSON.stringify(message))
  }

  useEffect(() => {
    const saved = loadSession()
    if (saved) enter({ type: 'reconnect', ...saved })
  }, [])

  return { lobby, game, log, refusal, reconnecting, deadline, enter, send }
}

export function useSecondsLeft(deadline: number): number {
  const [now, setNow] = useState(() => Date.now())
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 250)
    return () => clearInterval(timer)
  }, [])
  return Math.max(0, Math.ceil((deadline - now) / 1000))
}
