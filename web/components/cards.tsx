import type { ReactNode } from 'react'
import { capitalized, cardLabel } from '@/lib/coup'
import { cardNames, inkOf, tint } from '@/lib/palette'

type Face = { caption: string; blocks?: string; glyph: ReactNode }

const faces: Record<string, Face> = {
  duke: {
    caption: 'taxas · +3',
    blocks: 'bloqueia ajuda externa',
    glyph: (
      <>
        <path d="M7 35 L11 13 L18 22 L24 9 L30 22 L37 13 L41 35 Z" />
        <path d="M7 39 H41" />
      </>
    ),
  },
  assassin: {
    caption: 'assassinar · −3',
    glyph: (
      <>
        <path d="M24 5 L30 21 L24 43 L18 21 Z" />
        <path d="M11 22 H37" />
      </>
    ),
  },
  captain: {
    caption: 'extorquir · +2',
    blocks: 'bloqueia extorsão',
    glyph: (
      <>
        <circle cx="24" cy="10" r="4" />
        <path d="M24 14 V40 M13 25 H35 M10 28 A14 14 0 0 0 38 28" />
      </>
    ),
  },
  ambassador: {
    caption: 'trocar',
    blocks: 'bloqueia extorsão',
    glyph: (
      <>
        <circle cx="24" cy="24" r="14" />
        <path d="M24 10 L28 20 L38 24 L28 28 L24 38 L20 28 L10 24 L20 20 Z" />
      </>
    ),
  },
  contessa: {
    caption: 'sem ação própria',
    blocks: 'bloqueia assassinato',
    glyph: (
      <>
        <path d="M24 39 L8 17 M24 39 L14 12 M24 39 L24 10 M24 39 L34 12 M24 39 L40 17" />
        <path d="M9 19 A17 17 0 0 1 39 19" />
      </>
    ),
  },
}

export function Glyph({ card, size }: { card: string; size: number }) {
  return (
    <svg
      viewBox="0 0 48 48"
      width={size}
      height={size}
      fill="none"
      stroke="currentColor"
      strokeWidth="2.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {faces[card]?.glyph}
    </svg>
  )
}

const cardSizes = {
  large: {
    box: 'h-[224px] w-[158px] rounded-[18px] p-[15px] max-md:h-[170px] max-md:w-[120px] max-md:p-3',
    name: 'text-[26px] max-md:text-[19px]',
    longName: 'text-[21px] max-md:text-[16px]',
    glyph: 64,
  },
  medium: { box: 'h-[164px] w-[116px] rounded-2xl p-3', name: 'text-[19px]', longName: 'text-[16px]', glyph: 48 },
  small: { box: 'h-[128px] w-[92px] rounded-xl p-2.5', name: 'text-[15px]', longName: 'text-[12px]', glyph: 36 },
}

const longNameFrom = 9

export function Card({ card, size = 'large', lift = 'shadow-lift' }: { card: string; size?: keyof typeof cardSizes; lift?: string }) {
  const face = faces[card]
  const measure = cardSizes[size]
  return (
    <div
      className={`flex shrink-0 flex-col justify-between border-3 border-ink text-left text-white ${lift} ${tint[card]} ${measure.box}`}
      role="img"
      aria-label={capitalized(cardLabel(card))}
    >
      <span className="text-[9.5px] leading-[1.6] font-bold tracking-[0.12em] uppercase max-md:text-[8px]">
        {face?.caption}
        {face?.blocks && size === 'large' && (
          <span className="max-md:hidden">
            <br />
            {face.blocks}
          </span>
        )}
      </span>
      <span className="self-center max-md:[&_svg]:size-12">
        <Glyph card={card} size={measure.glyph} />
      </span>
      <span className={`leading-none font-extrabold ${cardLabel(card).length >= longNameFrom ? measure.longName : measure.name}`}>
        {capitalized(cardLabel(card))}
      </span>
    </div>
  )
}

export function CardBack({ size = 'small' }: { size?: 'small' | 'tiny' }) {
  const measure = size === 'small' ? 'h-14 w-10 rounded-[7px] max-md:h-[30px] max-md:w-[22px] max-md:rounded-[5px]' : 'h-[30px] w-[22px] rounded-[5px]'
  return (
    <span
      className={`inline-block border-2 border-ink bg-paper bg-[repeating-linear-gradient(45deg,var(--color-line)_0_2px,transparent_2px_8px)] ${measure}`}
      aria-label="carta virada"
    />
  )
}

export function CardChip({ card }: { card: string }) {
  return (
    <span
      className={`inline-block rounded-full border-2 border-ink px-2.5 py-1 text-[10px] font-bold tracking-[0.06em] text-white uppercase line-through ${tint[card]}`}
    >
      {cardLabel(card)}
    </span>
  )
}

export function Coins({ count, ring = 'border-ink' }: { count: number; ring?: string }) {
  return (
    <span className="inline-flex items-center gap-2" aria-label={`${count} moedas`}>
      <span className="flex gap-1 max-md:hidden" aria-hidden="true">
        {Array.from({ length: Math.min(count, 10) }, (_, position) => (
          <span key={position} className={`size-[13px] rounded-full border-2 bg-gold ${ring}`} />
        ))}
      </span>
      <span className="numeric">{count}</span>
    </span>
  )
}

const cardPattern = new RegExp(`(${cardNames.map((name) => capitalized(cardLabel(name))).join('|')})`, 'g')

export function Narration({ text }: { text: string }) {
  return (
    <>
      {text.split(cardPattern).map((piece, position) => {
        const card = cardNames.find((name) => capitalized(cardLabel(name)) === piece)
        return card ? (
          <span key={position} className={`font-bold ${inkOf[card]}`}>
            {piece}
          </span>
        ) : (
          piece
        )
      })}
    </>
  )
}
