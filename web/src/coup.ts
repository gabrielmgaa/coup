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

export type View = {
  phase: string
  you: string
  turn_of: string
  losing?: string
  winner?: string
  deck_remaining: number
  players: PlayerView[]
  your_actions: AvailableAction[]
}

export type GameEvent = { n: number; type: string; text: string }

export type FromServer =
  | { type: 'update'; state: View; events: GameEvent[] }
  | { type: 'error'; code: string; message: string; received?: unknown; expected?: unknown }

export type FromClient =
  | { type: 'join'; name: string }
  | { type: 'play'; action: string; target?: string }
  | { type: 'lose_influence'; card: string }

const actionLabels: Record<string, string> = {
  income: 'renda',
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

export function roomAddress(): string {
  const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
  return `${scheme}://${location.host}/ws`
}
