import type { LobbyView, Send } from './coup'

export function Lobby({ lobby, send, refusal }: { lobby: LobbyView; send: Send; refusal: string | null }) {
  const me = lobby.players.find((seat) => seat.name === lobby.you)
  const everyoneReady = lobby.players.every((seat) => seat.ready)
  const hosting = lobby.you === lobby.host
  const invite = `${location.origin}/?room=${lobby.room}`

  return (
    <main className="join-screen">
      <h1>Coup</h1>
      <p className="room-code">
        mesa <strong>{lobby.room}</strong>
      </p>
      <p className="hint">convide pelo link {invite}</p>
      {lobby.last_winner && <p className="winner">{lobby.last_winner} venceu a última partida e começa a próxima</p>}
      {lobby.options.independent_reactions && <p className="hint">regra da casa: reações independentes</p>}

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
          <button disabled={!everyoneReady || lobby.players.length < 2} onClick={() => send({ type: 'start' })}>
            começar a partida
          </button>
        )}
      </section>

      {!hosting && <p className="waiting">{lobby.host} começa a partida quando todos estiverem prontos…</p>}
      {refusal && <p className="refusal">{refusal}</p>}
    </main>
  )
}
