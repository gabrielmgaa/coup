import type { GameState } from '@/lib/coup'
import { Card, CardChip, Coins } from './cards'

export function Ending({ game, onBack }: { game: GameState; onBack: () => void }) {
  const standings = [...game.players].sort((first, second) => Number(first.eliminated) - Number(second.eliminated))
  const mine = game.players.find((player) => player.name === game.you)
  const championHand = game.winner === game.you ? (mine?.my_cards ?? []) : []

  return (
    <main className="mx-auto flex min-h-dvh max-w-[900px] flex-col justify-center gap-7 px-5 py-10">
      <span className="tag self-start border-ink bg-ink text-paper">a corte tem dono</span>
      <div className="flex flex-wrap items-center gap-7">
        <h1 className="text-[clamp(40px,8vw,60px)] leading-[1.05] font-extrabold">
          <span className="rounded-full bg-ink px-4 text-paper">{game.winner}</span> venceu a partida.
        </h1>
        {championHand.length > 0 && (
          <div className="flex gap-3">
            {championHand.map((card, position) => (
              <Card key={position} card={card} size="medium" />
            ))}
          </div>
        )}
      </div>

      <section className="flex flex-col gap-3">
        <span className="label">como a mesa terminou</span>
        <ol className="flex flex-col gap-2.5">
          {standings.map((player) => {
            const champion = player.name === game.winner
            return (
              <li
                key={player.name}
                className={`flex flex-wrap items-center gap-3.5 rounded-2xl border-3 border-ink px-[18px] py-3 text-lg font-bold ${
                  champion ? 'bg-ink text-white shadow-gold' : 'bg-card'
                }`}
              >
                {player.name}
                <span className="grow" />
                {player.revealed.map((card, position) => (
                  <CardChip key={position} card={card} />
                ))}
                {champion && <Coins count={player.coins} ring="border-white" />}
              </li>
            )
          })}
        </ol>
      </section>

      <p className="text-sm font-medium text-muted">
        A mesa continua de pé com o mesmo código. Quem venceu começa a próxima — é regra do livreto.
      </p>
      <button className="btn btn-primary self-start" onClick={onBack}>
        voltar pra sala
      </button>
    </main>
  )
}
