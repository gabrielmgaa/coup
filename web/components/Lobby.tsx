import { useState } from 'react'
import type { LobbyView, Send } from '@/lib/coup'
import { Banner, Brand } from './Brand'

const seatsAtTheTable = 6

export function Lobby({ lobby, send, refusal }: { lobby: LobbyView; send: Send; refusal: string | null }) {
  const [copied, setCopied] = useState(false)
  const me = lobby.players.find((seat) => seat.name === lobby.you)
  const everyoneReady = lobby.players.every((seat) => seat.ready)
  const canStart = everyoneReady && lobby.players.length >= 2
  const hosting = lobby.you === lobby.host
  const freeSeats = Math.max(0, seatsAtTheTable - lobby.players.length)

  function copyInvite() {
    navigator.clipboard?.writeText(`${location.origin}/?room=${lobby.room}`).then(() => setCopied(true))
  }

  return (
    <main className="mx-auto flex min-h-dvh max-w-[720px] flex-col gap-6 px-5 py-8">
      <header className="flex items-center gap-3">
        <Brand />
        <span className="grow" />
        <span className="tag">sala de espera</span>
      </header>

      <section className="flex flex-wrap items-end gap-4">
        <div className="flex flex-col gap-2">
          <span className="label">código da mesa</span>
          <span className="rounded-full border-3 border-ink bg-gold py-1.5 pr-4 pl-6 font-mono text-[clamp(36px,10vw,56px)] font-bold tracking-[0.2em] shadow-lift">
            {lobby.room}
          </span>
        </div>
        <button className="btn" onClick={copyInvite}>
          {copied ? 'link copiado' : 'copiar o link'}
        </button>
      </section>

      {lobby.last_winner && <Banner tone="gold">{lobby.last_winner} venceu a última partida e começa a próxima.</Banner>}
      {lobby.options.independent_reactions && (
        <p className="text-sm font-medium text-muted">regra da casa: quem contesta ainda pode bloquear</p>
      )}

      <ul className="flex flex-col gap-2.5">
        {lobby.players.map((seat) => (
          <li
            key={seat.name}
            className="flex items-center gap-3 rounded-2xl border-3 border-ink bg-card px-[18px] py-3 text-xl font-bold shadow-lift"
          >
            {seat.name}
            {(seat.name === lobby.you || seat.name === lobby.host) && (
              <span className="label">
                {[seat.name === lobby.you && 'você', seat.name === lobby.host && 'host'].filter(Boolean).join(' · ')}
              </span>
            )}
            <span className="grow" />
            <span className={`tag ${seat.ready ? 'bg-ambassador text-white' : ''}`}>{seat.ready ? 'pronto' : 'esperando'}</span>
          </li>
        ))}
        {Array.from({ length: freeSeats }, (_, position) => (
          <li key={position} className="rounded-2xl border-3 border-dashed border-edge px-[18px] py-3 text-xl text-faint">
            assento livre
          </li>
        ))}
      </ul>

      <div className="flex flex-wrap items-center gap-3">
        <span className="numeric text-xl">
          {lobby.players.length} / {seatsAtTheTable}
        </span>
        <span className="grow" />
        <button className="btn" onClick={() => send({ type: 'ready', ready: !me?.ready })}>
          {me?.ready ? 'ainda não estou pronto' : 'estou pronto'}
        </button>
        {hosting && (
          <button className="btn btn-primary" disabled={!canStart} onClick={() => send({ type: 'start' })}>
            começar a partida
          </button>
        )}
      </div>

      <p className="text-sm font-medium text-muted">
        {hosting
          ? canStart
            ? 'todo mundo pronto — você é o host, você começa'
            : 'a partida começa quando todos marcarem pronto'
          : `${lobby.host} começa a partida quando todos estiverem prontos…`}
      </p>
      {refusal && <Banner>{refusal}</Banner>}
    </main>
  )
}
