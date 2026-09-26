import { useState } from 'react'
import { actionLabel, capitalized, cardLabel, type AvailableAction, type GameState, type Send } from '@/lib/coup'
import { cn } from '@/lib/utils'
import { CardChip, Coins, PlayingCard } from './cards'
import { Badge } from './ui/badge'
import { Button } from './ui/button'
import { Card, CardContent } from './ui/card'

type HandProps = { game: GameState; send: Send; secondsLeft: number }

export function Hand({ game, send, secondsLeft }: HandProps) {
  const me = game.players.find((player) => player.name === game.you)
  if (!me) return null
  const cards = me.my_cards ?? []
  const onTurn = game.turn_of === game.you && game.phase === 'awaiting_action'

  return (
    <Card>
      <CardContent className="flex gap-6 p-6 max-md:flex-col max-md:p-4">
        {game.your_returns ? (
          <Exchange cards={cards} pairs={game.your_returns} send={send} />
        ) : (
          <>
            <HandCards cards={cards} revealing={game.losing === game.you} send={send} />
            <div className="flex min-w-0 grow flex-col gap-4">
              <div className="flex flex-wrap items-center gap-3">
                {onTurn && <Badge variant="solid">sua vez</Badge>}
                <h2 className="text-2xl font-bold">{me.name}</h2>
                {secondsLeft > 0 && onTurn && <span className="numeric text-muted-foreground">{secondsLeft}s</span>}
                <span className="grow" />
                <Coins count={me.coins} />
              </div>
              {me.revealed.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {me.revealed.map((card, position) => (
                    <CardChip key={position} card={card} />
                  ))}
                </div>
              )}
              {onTurn ? (
                <Actions actions={game.your_actions} send={send} />
              ) : (
                <p className="text-sm font-medium text-muted-foreground">{waitingLine(game)}</p>
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

function waitingLine(game: GameState): string {
  if (game.players.find((player) => player.name === game.you)?.eliminated) return 'Você está fora desta partida — mas pode assistir até o fim.'
  if (game.losing === game.you) return 'Toque na carta que você vai revelar.'
  if (game.window?.your_options.length) return 'Responda na janela acima.'
  return `É a vez de ${game.turn_of}.`
}

function HandCards({ cards, revealing, send }: { cards: string[]; revealing: boolean; send: Send }) {
  return (
    <div className="flex shrink-0 gap-3.5">
      {cards.map((card, position) =>
        revealing ? (
          <Button
            key={position}
            variant="bare"
            size="bare"
            className="flex-col gap-2.5"
            onClick={() => send({ type: 'lose_influence', card })}
          >
            <PlayingCard card={card} />
            revelar o {capitalized(cardLabel(card))}
          </Button>
        ) : (
          <PlayingCard key={position} card={card} />
        ),
      )}
    </div>
  )
}

function Actions({ actions, send }: { actions: AvailableAction[]; send: Send }) {
  const [aiming, setAiming] = useState<AvailableAction | null>(null)

  if (aiming) {
    return (
      <div className="flex flex-col gap-3">
        <span className="eyebrow">{actionLabel(aiming.name)} — escolha o alvo</span>
        <div className="flex flex-wrap gap-2.5 max-md:grid max-md:grid-cols-2">
          {aiming.targets?.map((target) => (
            <Button
              key={target}
              variant="default"
              onClick={() => {
                send({ type: 'play', action: aiming.name, target })
                setAiming(null)
              }}
            >
              {target}
            </Button>
          ))}
          <Button onClick={() => setAiming(null)}>cancelar</Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-wrap gap-2.5 max-md:grid max-md:grid-cols-2">
      {actions.map((action) => (
        <Button
          key={action.name}
          variant={action.name === 'income' ? 'default' : 'outline'}
          className="max-md:gap-1.5 max-md:px-3 max-md:text-[15px] max-md:whitespace-normal"
          onClick={() => (action.targets ? setAiming(action) : send({ type: 'play', action: action.name }))}
        >
          {actionLabel(action.name)}
          {action.cost ? <span className="font-mono text-[13px]">−{action.cost}</span> : null}
          {action.targets && <span className="text-[11px] tracking-[0.12em] text-muted-foreground uppercase">alvo ▸</span>}
        </Button>
      ))}
    </div>
  )
}

function Exchange({ cards, pairs, send }: { cards: string[]; pairs: string[][]; send: Send }) {
  const [returning, setReturning] = useState<number[]>([])
  const chosen = returning.map((position) => cards[position]).sort()
  const valid = pairs.some((pair) => [...pair].sort().join() === chosen.join())

  function toggle(position: number) {
    setReturning((previous) =>
      previous.includes(position) ? previous.filter((kept) => kept !== position) : [...previous, position].slice(-2),
    )
  }

  return (
    <div className="flex w-full flex-col gap-5">
      <div className="flex flex-wrap items-center gap-3">
        <Badge variant="solid">troca</Badge>
        <h2 className="text-2xl font-bold">Escolha as duas que voltam pro baralho</h2>
        <span className="grow" />
        <span className="numeric text-muted-foreground">{returning.length} de 2</span>
      </div>
      <div className="flex flex-wrap gap-3.5">
        {cards.map((card, position) => {
          const goesBack = returning.includes(position)
          return (
            <Button
              key={position}
              variant="bare"
              size="bare"
              aria-pressed={goesBack}
              className="flex-col gap-2.5"
              onClick={() => toggle(position)}
            >
              <span className={cn('transition', goesBack && 'translate-y-2 opacity-35')}>
                <PlayingCard card={card} lift={goesBack ? 'shadow-none' : 'shadow-lift'} />
              </span>
              <span className={cn('text-[13px] tracking-[0.1em] uppercase', goesBack ? 'text-blood' : 'text-ambassador')}>
                {goesBack ? 'devolve' : '✓ fica'}
              </span>
            </Button>
          )
        })}
      </div>
      <Button
        variant="default"
        className="self-start"
        disabled={!valid}
        onClick={() => send({ type: 'return_cards', cards: returning.map((position) => cards[position]) })}
      >
        devolver as duas marcadas
      </Button>
    </div>
  )
}
