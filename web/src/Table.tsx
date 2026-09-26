import { useSecondsLeft, type Table as Connection } from './connection'
import {
  actionLabel,
  cardLabel,
  optionLabel,
  type AvailableAction,
  type GameState,
  type PlayerView,
  type Send,
  type WindowView,
} from './coup'

export function Table({ table, game }: { table: Connection; game: GameState }) {
  const secondsLeft = useSecondsLeft(table.deadline)

  return (
    <main className="table">
      <header>
        <h1>Coup</h1>
        <span className="deck">
          mesa {game.room} · {game.deck_remaining} cartas no baralho
        </span>
      </header>

      {game.winner && <p className="winner">{game.winner} venceu a partida</p>}
      {game.paused && (
        <p className="refusal">
          mesa pausada esperando {game.paused.waiting_for.join(', ')} voltar (
          {Math.ceil(game.paused.resumes_in_ms / 1000)}s)
        </p>
      )}
      {table.reconnecting && <p className="refusal">conexão perdida, reconectando…</p>}

      <section className="seats">
        {game.players.map((player) => (
          <Seat key={player.name} player={player} game={game} />
        ))}
      </section>

      {secondsLeft > 0 && <p className="waiting">{secondsLeft}s para decidir</p>}
      {table.refusal && <p className="refusal">{table.refusal}</p>}

      <Decision game={game} send={table.send} />

      <ol className="log">
        {table.log.map((event) => (
          <li key={event.n}>{event.text}</li>
        ))}
      </ol>
    </main>
  )
}

function Seat({ player, game }: { player: PlayerView; game: GameState }) {
  const classes = ['seat']
  if (player.name === game.turn_of) classes.push('on-turn')
  if (player.eliminated) classes.push('eliminated')

  return (
    <article className={classes.join(' ')}>
      <h2>
        {player.name}
        {player.name === game.you && <small> (você)</small>}
        {game.disconnected.includes(player.name) && <small> (caiu)</small>}
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

function Decision({ game, send }: { game: GameState; send: Send }) {
  const me = game.players.find((player) => player.name === game.you)

  if (game.phase === 'finished') return null
  if (game.losing === game.you) {
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
  if (game.losing) return <p className="waiting">{game.losing} está escolhendo qual carta perder…</p>
  if (game.your_returns) return <Returns pairs={game.your_returns} send={send} />
  if (game.phase === 'awaiting_exchange') return <p className="waiting">{game.turn_of} está escolhendo cartas…</p>
  if (game.window) return <Reactions window={game.window} send={send} />
  if (game.turn_of !== game.you) return <p className="waiting">é a vez de {game.turn_of}…</p>
  return <section className="actions">{game.your_actions.flatMap((action) => actionButtons(action, send))}</section>
}

function Returns({ pairs, send }: { pairs: string[][]; send: Send }) {
  return (
    <section className="actions">
      <p>escolha as 2 cartas que voltam para o baralho</p>
      {pairs.map((pair) => (
        <button key={pair.join('-')} onClick={() => send({ type: 'return_cards', cards: pair })}>
          devolver {pair.map(cardLabel).join(' e ')}
        </button>
      ))}
    </section>
  )
}

function Reactions({ window, send }: { window: WindowView; send: Send }) {
  return (
    <section className="actions">
      <p>
        {window.action.by} declarou {actionLabel(window.action.name)}
        {window.action.target && ` em ${window.action.target}`}
        {window.block && `; ${window.block.by} bloqueou com ${cardLabel(window.block.character)}`} — esperando{' '}
        {window.waiting_on.join(', ')}
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

function actionButtons(action: AvailableAction, send: Send) {
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
    <button key={`${action.name}-${target}`} onClick={() => send({ type: 'play', action: action.name, target })}>
      {label} em {target}
      {price}
    </button>
  ))
}
