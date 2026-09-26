import { useState } from 'react'
import type { LobbyView, Send } from '@/lib/coup'
import { Brand, Notice } from './Brand'
import { Badge } from './ui/badge'
import { Button } from './ui/button'

const seatsAtTheTable = 6

export function Lobby({ lobby, send, refusal }: { lobby: LobbyView; send: Send; refusal: string | null }) {
  const [copied, setCopied] = useState(false)
  const me = lobby.players.find((seat) => seat.name === lobby.you)
  const canStart = lobby.players.every((seat) => seat.ready) && lobby.players.length >= 2
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
        <Badge>sala de espera</Badge>
      </header>

      <section className="flex flex-wrap items-end gap-4">
        <div className="flex flex-col gap-2">
          <span className="eyebrow">código da mesa</span>
          <span className="rounded-full border-3 border-border bg-gold py-1.5 pr-4 pl-6 font-mono text-[clamp(36px,10vw,56px)] font-bold tracking-[0.2em] shadow-lift">
            {lobby.room}
          </span>
        </div>
        <Button onClick={copyInvite}>{copied ? 'link copiado' : 'copiar o link'}</Button>
      </section>

      {lobby.last_winner && <Notice tone="default">{lobby.last_winner} venceu a última partida e começa a próxima.</Notice>}
      {lobby.options.independent_reactions && (
        <p className="text-sm font-medium text-muted-foreground">regra da casa: quem contesta ainda pode bloquear</p>
      )}

      <ul className="flex flex-col gap-2.5">
        {lobby.players.map((seat) => (
          <li
            key={seat.name}
            className="flex items-center gap-3 rounded-2xl border-3 border-border bg-card px-[18px] py-3 text-xl font-bold shadow-lift"
          >
            {seat.name}
            {(seat.name === lobby.you || seat.name === lobby.host) && (
              <span className="eyebrow">
                {[seat.name === lobby.you && 'você', seat.name === lobby.host && 'host'].filter(Boolean).join(' · ')}
              </span>
            )}
            <span className="grow" />
            <Badge variant={seat.ready ? 'ready' : 'default'}>{seat.ready ? 'pronto' : 'esperando'}</Badge>
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
        <Button onClick={() => send({ type: 'ready', ready: !me?.ready })}>
          {me?.ready ? 'ainda não estou pronto' : 'estou pronto'}
        </Button>
        {hosting && (
          <Button variant="default" disabled={!canStart} onClick={() => send({ type: 'start' })}>
            começar a partida
          </Button>
        )}
      </div>

      <p className="text-sm font-medium text-muted-foreground">
        {hosting
          ? canStart
            ? 'todo mundo pronto — você é o host, você começa'
            : 'a partida começa quando todos marcarem pronto'
          : `${lobby.host} começa a partida quando todos estiverem prontos…`}
      </p>
      {refusal && <Notice>{refusal}</Notice>}
    </main>
  )
}
