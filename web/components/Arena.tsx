import { actionLabel, capitalized, cardLabel, optionLabel, type GameEvent, type GameState, type Send, type WindowView } from '@/lib/coup'
import { tint } from '@/lib/palette'
import { Card, Narration } from './cards'

type ArenaProps = { game: GameState; send: Send; secondsLeft: number; pauseLeft: number; log: GameEvent[] }

export function Arena({ game, send, secondsLeft, pauseLeft, log }: ArenaProps) {
  if (game.paused) return <Pause waitingFor={game.paused.waiting_for} secondsLeft={pauseLeft} />
  if (game.window) return <Window game={game} window={game.window} send={send} secondsLeft={secondsLeft} />
  return <Story game={game} latest={log.at(-1)} />
}

function Window({ game, window, send, secondsLeft }: { game: GameState; window: WindowView; send: Send; secondsLeft: number }) {
  const claim = window.block ? window.block.character : window.action.claims
  const share = game.decision_ms ? Math.min(100, (secondsLeft * 1000 * 100) / game.decision_ms) : 0
  const aimedAtYou = window.action.target === game.you

  return (
    <section className="overflow-hidden rounded-3xl border-3 border-blood bg-card shadow-blood">
      <div className="flex flex-wrap items-center gap-3.5 bg-blood px-6 py-2.5 text-white">
        <span className="text-[11px] font-bold tracking-[0.2em] uppercase">
          {window.block ? 'bloqueio em aberto' : 'janela aberta'}
        </span>
        <span className="h-1.5 max-w-[420px] grow overflow-hidden rounded-full bg-white/30" aria-hidden="true">
          <span className="block h-full rounded-full bg-gold transition-[width] duration-300 ease-linear" style={{ width: `${share}%` }} />
        </span>
        <span className="numeric text-[17px]">{secondsLeft}s</span>
        <span className="grow" />
        <span className="text-[13px] font-medium max-md:hidden">uma reação por pessoa · quem falar primeiro resolve</span>
      </div>

      <div className="flex flex-col gap-5 px-7 py-6 max-md:px-4">
        <div className="flex items-center gap-6">
          {claim && (
            <span className="max-md:hidden">
              <Card card={claim} size="medium" lift="shadow-[4px_4px_0_var(--color-blood)]" />
            </span>
          )}
          <div className="flex min-w-0 flex-col gap-2">
            <span className="label">{window.block ? `${window.block.by} bloqueou` : `${window.action.by} declarou`}</span>
            <h1 className="text-[clamp(28px,4vw,44px)] leading-[1.06] font-extrabold text-pretty">
              {window.block ? (
                <>com {capitalized(cardLabel(window.block.character))}</>
              ) : (
                <>
                  {capitalized(actionLabel(window.action.name))}
                  {window.action.target && (
                    <>
                      {' '}
                      {aimedAtYou ? <span className="rounded-full bg-blood px-3.5 text-white">você</span> : window.action.target}
                    </>
                  )}
                </>
              )}
            </h1>
            {claim && (
              <p className="text-base text-muted">
                alegando ter o <span className="font-bold text-ink">{capitalized(cardLabel(claim))}</span> na mão.
              </p>
            )}
          </div>
        </div>

        {window.your_options.length > 0 ? (
          <div className="flex flex-wrap items-center gap-3 max-md:flex-col max-md:items-stretch">
            {window.your_options.map((option) => (
              <button
                key={`${option.answer}-${option.character ?? ''}`}
                className={`btn min-h-[54px] text-[17px] ${
                  option.answer === 'challenge'
                    ? 'btn-primary shadow-[3px_3px_0_var(--color-blood)]'
                    : option.answer === 'block'
                      ? `text-white ${tint[option.character ?? '']}`
                      : ''
                }`}
                onClick={() => send({ type: 'respond', window: window.id, ...option })}
              >
                {optionLabel(window, option)}
              </button>
            ))}
          </div>
        ) : (
          <p className="text-sm font-medium text-muted">
            esperando {window.waiting_on.join(', ')} reagir{window.waiting_on.length > 1 ? 'em' : ''}…
          </p>
        )}
      </div>
    </section>
  )
}

function Pause({ waitingFor, secondsLeft }: { waitingFor: string[]; secondsLeft: number }) {
  return (
    <section className="flex items-center gap-6 rounded-[22px] border-3 border-ink bg-gold-wash px-7 py-5 shadow-gold max-md:flex-col max-md:items-start">
      <span className="rounded-2xl bg-ink px-4 py-1.5 font-mono text-[44px] font-bold text-gold tabular-nums">
        00:{String(secondsLeft).padStart(2, '0')}
      </span>
      <div className="flex flex-col gap-2">
        <span className="label">mesa pausada</span>
        <p className="text-2xl leading-tight font-bold">{waitingFor.join(', ')} caiu no meio da jogada.</p>
        <p className="text-[15px] text-muted">
          A mesa só para porque a jogada depende de quem caiu. Voltando a tempo, o relógio recomeça cheio; se não, a jogada
          resolve pelo padrão seguro e a partida segue.
        </p>
      </div>
    </section>
  )
}

function Story({ game, latest }: { game: GameState; latest?: GameEvent }) {
  const headline = storyHeadline(game)
  return (
    <section className="flex min-h-[200px] items-center gap-7 rounded-[22px] border-3 border-dashed border-edge bg-table px-8 py-6 max-md:min-h-0 max-md:px-4 max-md:py-4">
      <div className="flex min-w-0 flex-col gap-2.5">
        <span className="label">{headline ? 'agora' : 'a última jogada'}</span>
        <p className="text-[clamp(22px,3vw,36px)] leading-[1.15] font-bold text-pretty">
          {headline ?? (latest ? <Narration text={latest.text} /> : 'A partida começou.')}
        </p>
      </div>
    </section>
  )
}

function storyHeadline(game: GameState): string | null {
  if (game.losing === game.you) return 'Você perdeu uma influência. Qual carta revela?'
  if (game.losing) return `${game.losing} está escolhendo qual carta perder…`
  if (game.phase === 'awaiting_exchange' && game.turn_of !== game.you) return `${game.turn_of} está trocando cartas com o baralho…`
  if (game.phase === 'awaiting_exchange') return 'Você comprou duas. Fique com o que quiser e devolva duas.'
  return null
}
