export type PlayerView = {
  name: string
  coins: number
  hidden: number
  revealed: string[]
  eliminated: boolean
  my_cards?: string[]
}

export type AvailableAction = {
  name: string
  cost?: number
  targets?: string[]
}

export type Option = { answer: 'challenge' | 'block' | 'pass'; character?: string }

export type WindowView = {
  id: number
  action: { name: string; by: string; target?: string; claims?: string }
  block: { by: string; character: string } | null
  your_options: Option[]
  waiting_on: string[]
}

export type GameState = {
  room: string
  phase: string
  you: string
  turn_of: string
  losing?: string
  winner?: string
  deck_remaining: number
  players: PlayerView[]
  window: WindowView | null
  your_actions: AvailableAction[]
  your_returns?: string[][]
  closes_in_ms?: number
  paused: { waiting_for: string[]; resumes_in_ms: number } | null
  disconnected: string[]
}

export type GameEvent = { n: number; type: string; text: string }

export type SeatView = {
  name: string
  ready: boolean
}

export type LobbyView = {
  room: string
  you: string
  host: string
  players: SeatView[]
}

export type FromServer =
  | { type: 'welcome'; room: string; token: string }
  | { type: 'lobby'; state: LobbyView }
  | { type: 'update'; state: GameState; events: GameEvent[] }
  | { type: 'error'; code: string; message: string; received?: unknown; expected?: unknown }

export type FromClient =
  | { type: 'create_room'; name: string }
  | { type: 'join'; room: string; name: string }
  | { type: 'reconnect'; room: string; token: string }
  | { type: 'ready'; ready: boolean }
  | { type: 'start' }
  | { type: 'play'; action: string; target?: string }
  | { type: 'lose_influence'; card: string }
  | { type: 'respond'; window: number; answer: Option['answer']; character?: string }
  | { type: 'return_cards'; cards: string[] }

export type Send = (message: FromClient) => void

const actionLabels: Record<string, string> = {
  income: 'renda',
  foreign_aid: 'ajuda externa',
  tax: 'taxas',
  assassinate: 'assassinar',
  steal: 'extorquir',
  exchange: 'trocar',
  coup: 'golpe',
}

const cardLabels: Record<string, string> = {
  duke: 'duque',
  assassin: 'assassino',
  captain: 'capitão',
  ambassador: 'embaixador',
  contessa: 'condessa',
}

export function actionLabel(name: string): string {
  return actionLabels[name] ?? name
}

export function cardLabel(name: string): string {
  return cardLabels[name] ?? name
}

export function optionLabel(window: WindowView, option: Option): string {
  if (option.answer === 'challenge' && window.block) {
    return `contestar o ${cardLabel(window.block.character)} de ${window.block.by}`
  }
  if (option.answer === 'challenge') return `contestar o ${cardLabel(window.action.claims ?? '')} de ${window.action.by}`
  if (option.answer === 'block') return `bloquear com ${cardLabel(option.character ?? '')}`
  return 'deixar passar'
}

export function roomAddress(): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${scheme}://${location.host}/ws`
}

export type Session = { room: string; token: string }

const sessionKey = 'coup-session'

export function loadSession(): Session | null {
  try {
    const saved = localStorage.getItem(sessionKey)
    return saved ? (JSON.parse(saved) as Session) : null
  } catch {
    return null
  }
}

export function saveSession(session: Session) {
  try {
    localStorage.setItem(sessionKey, JSON.stringify(session))
  } catch {
    return
  }
}

export function forgetSession() {
  try {
    localStorage.removeItem(sessionKey)
  } catch {
    return
  }
}
