import { useRef, useState, type FormEvent } from 'react'
import {
  actionLabel,
  cardLabel,
  roomAddress,
  type AvailableAction,
  type FromClient,
  type FromServer,
  type GameEvent,
  type PlayerView,
  type View,
} from './coup'

export default function App() {
  const [name, setName] = useState('')
  const [connected, setConnected] = useState(false)
  const [state, setState] = useState<View | null>(null)
  const [log, setLog] = useState<GameEvent[]>([])
  const [refusal, setRefusal] = useState<string | null>(null)
  const socket = useRef<WebSocket | null>(null)

  function join(submitted: FormEvent) {
    submitted.preventDefault()
    const opened = new WebSocket(roomAddress())
    opened.onopen = () => {
      setConnected(true)
      opened.send(JSON.stringify({ type: 'join', name } satisfies FromClient))
    }
    opened.onmessage = (incoming) => {
      const message: FromServer = JSON.parse(incoming.data)
      if (message.type === 'error') {
        setRefusal(message.message)
        return
      }
      setRefusal(null)
      setState(message.state)
      setLog((previous) => [...previous, ...message.events])
    }
    opened.onclose = () => setConnected(false)
    socket.current = opened
  }

  function send(message: FromClient) {
    socket.current?.send(JSON.stringify(message))
  }

  if (!connected && !state) {
    return (
      <main className="join-screen">
        <h1>Coup</h1>
        <form onSubmit={join}>
          <input
            value={name}
            onChange={(typed) => setName(typed.target.value)}
            placeholder="seu nome"
            autoFocus
          />
          <button disabled={name.trim() === ''}>entrar na mesa</button>
        </form>
        {refusal && <p className="refusal">{refusal}</p>}
      </main>
    )
  }

  if (!state) {
    return (
      <main className="join-screen">
        <h1>Coup</h1>
        <p className="waiting">esperando alguém entrar na mesa…</p>
        {refusal && <p className="refusal">{refusal}</p>}
      </main>
    )
  }

  return (
    <main className="table">
      <header>
        <h1>Coup</h1>
        <span className="deck">{state.deck_remaining} cartas no baralho</span>
      </header>

      {state.winner && <p className="winner">{state.winner} venceu a partida</p>}

      <section className="seats">
        {state.players.map((player) => (
          <Seat key={player.name} player={player} state={state} />
        ))}
      </section>

      {refusal && <p className="refusal">{refusal}</p>}

      <YourTurn state={state} send={send} />

      <ol className="log">
        {log.map((event) => (
          <li key={event.n}>{event.text}</li>
        ))}
      </ol>
    </main>
  )
}

function Seat({ player, state }: { player: PlayerView; state: View }) {
  const classes = ['seat']
  if (player.name === state.turn_of) classes.push('on-turn')
  if (player.eliminated) classes.push('eliminated')

  return (
    <article className={classes.join(' ')}>
      <h2>
        {player.name}
        {player.name === state.you && <small> (você)</small>}
      </h2>
      <p className="coins">{player.coins} moedas</p>
      <p className="cards">
        {player.my_cards?.map((card, position) => (
          <span className="mine" key={`mine-${position}`}>
            {cardLabel(card)}
          </span>
        ))}
        {!player.my_cards &&
          Array.from({ length: player.hidden }, (_, position) => (
            <span className="hidden" key={`hidden-${position}`}>
              ?
            </span>
          ))}
        {player.revealed.map((card, position) => (
          <span className="revealed" key={`revealed-${position}`}>
            {cardLabel(card)}
          </span>
        ))}
      </p>
    </article>
  )
}

function YourTurn({ state, send }: { state: View; send: (message: FromClient) => void }) {
  const me = state.players.find((player) => player.name === state.you)

  if (state.losing === state.you) {
    return (
      <section className="actions">
        <p>você levou um golpe — qual carta revela?</p>
        {me?.my_cards?.map((card, position) => (
          <button key={`${card}-${position}`} onClick={() => send({ type: 'lose_influence', card })}>
            revelar {cardLabel(card)}
          </button>
        ))}
      </section>
    )
  }

  if (state.losing) {
    return <p className="waiting">{state.losing} está escolhendo qual carta perder…</p>
  }

  if (state.phase === 'finished') return null

  if (state.turn_of !== state.you) {
    return <p className="waiting">é a vez de {state.turn_of}…</p>
  }

  return (
    <section className="actions">{state.your_actions.flatMap((action) => buttonsFor(action, send))}</section>
  )
}

function buttonsFor(action: AvailableAction, send: (message: FromClient) => void) {
  const price = action.cost ? ` (${action.cost})` : ''
  const label = actionLabel(action.name)
  if (!action.targets) {
    return [
      <button key={action.name} onClick={() => send({ type: 'play', action: action.name })}>
        {label}
        {price}
      </button>,
    ]
  }
  return action.targets.map((target) => (
    <button
      key={`${action.name}-${target}`}
      onClick={() => send({ type: 'play', action: action.name, target })}
    >
      {label} em {target}
      {price}
    </button>
  ))
}
