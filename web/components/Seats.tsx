import type { GameState, PlayerView } from '@/lib/coup'
import { CardBack, CardChip, Coins } from './cards'

export function Seats({ game }: { game: GameState }) {
  const others = game.players.filter((player) => player.name !== game.you)
  return (
    <section className="grid grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-4 max-md:grid-cols-1 max-md:gap-2.5">
      {others.map((player) => (
        <Seat key={player.name} player={player} game={game} />
      ))}
    </section>
  )
}

function statusOf(player: PlayerView, game: GameState): string {
  if (player.eliminated) return 'fora do jogo'
  if (game.disconnected.includes(player.name)) return 'caiu · esperando voltar'
  const influence = player.hidden === 1 ? ' · 1 influência' : ''
  if (game.window?.block?.by === player.name) return `bloqueou${influence}`
  if (game.window?.action.by === player.name) return `declarou${influence}`
  if (game.window?.waiting_on.includes(player.name)) return `pensando…${influence}`
  if (game.losing === player.name) return `escolhendo carta${influence}`
  if (game.turn_of === player.name) return `na vez${influence}`
  return `esperando${influence}`
}

function Seat({ player, game }: { player: PlayerView; game: GameState }) {
  const onTurn = player.name === game.turn_of && !player.eliminated
  const dropped = game.disconnected.includes(player.name)
  const frame = player.eliminated
    ? 'border-faint bg-table'
    : onTurn
      ? 'border-ink bg-ink text-white shadow-gold'
      : 'border-ink bg-card shadow-lift'
  const strip = player.eliminated
    ? 'border-faint bg-blood-wash text-blood'
    : onTurn
      ? 'border-ink-soft bg-ink text-gold'
      : dropped
        ? 'border-ink bg-gold-wash text-muted'
        : 'border-ink bg-table text-muted'

  return (
    <article className={`flex flex-col overflow-hidden rounded-[18px] border-3 max-md:flex-row max-md:items-center max-md:rounded-2xl ${frame}`}>
      <div className={`border-b-3 px-4 py-2 max-md:hidden ${strip}`}>
        <span className="text-[11px] leading-[13px] font-bold tracking-[0.16em] uppercase">{statusOf(player, game)}</span>
      </div>
      <div className="flex grow flex-col gap-2.5 px-4 pt-3 pb-3.5 max-md:flex-row max-md:items-center max-md:py-2.5">
        <h2
          className={`text-[22px] leading-none font-bold max-md:grow max-md:text-[19px] ${
            player.eliminated ? 'text-muted line-through decoration-blood' : ''
          }`}
        >
          {player.name}
        </h2>
        {!player.eliminated && <Coins count={player.coins} ring={onTurn ? 'border-white' : 'border-ink'} />}
        <div className="mt-auto flex flex-wrap items-center gap-2 max-md:mt-0">
          {Array.from({ length: player.hidden }, (_, position) => (
            <CardBack key={position} />
          ))}
          {player.revealed.map((card, position) => (
            <CardChip key={position} card={card} />
          ))}
        </div>
      </div>
    </article>
  )
}
