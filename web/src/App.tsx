import { useRef, useState, type FormEvent } from 'react'
import {
  actionLabel,
  cardLabel,
  optionLabel,
  roomAddress,
  type AvailableAction,
  type FromClient,
  type FromServer,
  type GameEvent,
  type LobbyView,
  type PlayerView,
  type View,
} from './coup'

export default function App() {
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [lobby, setLobby] = useState<LobbyView | null>(null)
  const [state, setState] = useState<View | null>(null)
  const [log, setLog] = useState<GameEvent[]>([])
  const [refusal, setRefusal] = useState<string | null>(null)
  const socket = useRef<WebSocket | null>(null)

  function connect(first: FromClient) {
    const opened = new WebSocket(roomAddress())
    opened.onopen = () => opened.send(JSON.stringify(first))
    opened.onmessage = (incoming) => {
      const message: FromServer = JSON.parse(incoming.data)
      if (message.type === 'error') {
        setRefusal(message.message)
        return
      }
      setRefusal(null)
      if (message.type === 'lobby') {
        setLobby(message.state)
        return
      }
      setState(message.state)
      setLog((previous) => [...previous, ...message.events])
    }
    socket.current = opened
  }

  function enter(submitted: FormEvent) {
    submitted.preventDefault()
    const player = name.trim()
    const room = code.trim().toUpperCase()
    connect(room ? { type: 'join', room, name: player } : { type: 'create_room', name: player })
  }

  function send(message: FromClient) {
    socket.current?.send(JSON.stringify(message))
  }

  if (!lobby && !state) {
    return (
      <main className="join-screen">
        <h1>Coup</h1>
        <form onSubmit={enter}>
          <input
            value={name}
            onChange={(typed) => setName(typed.target.value)}
            placeholder="seu nome"
            maxLength={16}
            autoFocus
          />
          <input
            value={code}
            onChange={(typed) => setCode(typed.target.value.toUpperCase())}
            placeholder="código da mesa"
            maxLength={4}
          />
          <button disabled={name.trim().length < 2 || (code !== '' && code.trim().length !== 4)}>
            {code === '' ? 'abrir mesa' : 'entrar na mesa'}
          </button>
        </form>
        <p className="hint">deixe o código vazio para abrir uma mesa nova</p>
        {refusal && <p className="refusal">{refusal}</p>}
      </main>
    )
  }

  if (!state && lobby) {
    return <Lobby lobby={lobby} send={send} refusal={refusal} />
  }

  if (!state) return null

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

function Lobby({
  lobby,
  send,
  refusal,
}: {
  lobby: LobbyView
  send: (message: FromClient) => void
  refusal: string | null
}) {
  const me = lobby.players.find((seat) => seat.name === lobby.you)
  const everyoneReady = lobby.players.every((seat) => seat.ready)
  const hosting = lobby.you === lobby.host

  return (
    <main className="join-screen">
      <h1>Coup</h1>
      <p className="room-code">
        mesa <strong>{lobby.room}</strong>
      </p>

      <ul className="lobby-seats">
        {lobby.players.map((seat) => (
          <li key={seat.name} className={seat.ready ? 'ready' : 'waiting-on'}>
            {seat.name}
            {seat.name === lobby.host && <small> host</small>}
            {seat.name === lobby.you && <small> (você)</small>}
            <span>{seat.ready ? 'pronto' : 'esperando'}</span>
          </li>
        ))}
      </ul>

      <section className="actions">
        <button onClick={() => send({ type: 'ready', ready: !me?.ready })}>
          {me?.ready ? 'ainda não estou pronto' : 'estou pronto'}
        </button>
        {hosting && (
          <button
            disabled={!everyoneReady || lobby.players.length < 2}
            onClick={() => send({ type: 'start' })}
          >
            começar a partida
          </button>
        )}
      </section>

      {!hosting && <p className="waiting">{lobby.host} começa a partida quando todos estiverem prontos…</p>}
      {refusal && <p className="refusal">{refusal}</p>}
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
        <p>você perdeu uma influência — qual carta revela?</p>
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

  if (state.window) {
    const window = state.window
    return (
      <section className="actions">
        <p>
          {window.action.by} declarou {actionLabel(window.action.name)}
          {window.action.target && ` em ${window.action.target}`} — esperando {window.waiting_on.join(', ')}
        </p>
        {window.your_options.map((option) => (
          <button
            key={`${option.answer}-${option.character ?? ''}`}
            onClick={() => send({ type: 'respond', window: window.id, ...option })}
          >
            {optionLabel(window, option)}
          </button>
        ))}
      </section>
    )
  }

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
