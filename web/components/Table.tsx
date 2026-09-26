import { useSecondsLeft, type Table as Connection } from '@/lib/connection'
import type { GameState } from '@/lib/coup'
import { Arena } from './Arena'
import { Banner, Brand } from './Brand'
import { Hand } from './Hand'
import { Log } from './Log'
import { Seats } from './Seats'

export function Table({ table, game }: { table: Connection; game: GameState }) {
  const secondsLeft = useSecondsLeft(table.deadline)
  const pauseLeft = useSecondsLeft(table.pauseDeadline)
  const standing = game.players.filter((player) => !player.eliminated).length

  return (
    <main className="grid min-h-dvh grid-cols-[minmax(0,1fr)_360px] grid-rows-[auto_1fr] gap-x-7 px-8 pb-8 max-lg:grid-cols-1 max-lg:grid-rows-none max-lg:gap-4 max-lg:px-4">
      <header className="col-span-2 flex min-h-[88px] flex-wrap items-center gap-3 max-lg:col-span-1 max-lg:min-h-16">
        <Brand />
        <span className="grow" />
        <span className="inline-flex items-center gap-3 rounded-full border-3 border-ink bg-card py-1.5 pr-2 pl-4">
          <span className="label max-md:hidden">mesa</span>
          <span className="rounded-full bg-gold px-2.5 py-0.5 font-mono text-lg font-bold tracking-[0.2em]">{game.room}</span>
        </span>
        <span className="inline-flex items-center gap-2.5 rounded-full border-3 border-ink px-4 py-2 max-md:hidden">
          <span className="label">baralho</span>
          <span className="numeric">{String(game.deck_remaining).padStart(2, '0')}</span>
          <span className="h-4 w-0.5 bg-line" />
          <span className="label">em jogo</span>
          <span className="numeric">
            {standing}/{game.players.length}
          </span>
        </span>
      </header>

      <div className="flex min-w-0 flex-col gap-5">
        {table.reconnecting && <Banner tone="gold">conexão perdida, reconectando…</Banner>}
        <Seats game={game} />
        <Arena game={game} send={table.send} secondsLeft={secondsLeft} pauseLeft={pauseLeft} log={table.log} />
        {table.refusal && <Banner>{table.refusal}</Banner>}
        <Hand game={game} send={table.send} secondsLeft={secondsLeft} />
      </div>

      <Log log={table.log} />
    </main>
  )
}
