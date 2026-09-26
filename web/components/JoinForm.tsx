import { useState, type FormEvent } from 'react'
import type { Table } from '@/lib/connection'
import { Banner, Brand } from './Brand'
import { cardNames, tint } from '@/lib/palette'
import { Glyph } from './cards'

export function JoinForm({ table }: { table: Table }) {
  const [name, setName] = useState('')
  const [code, setCode] = useState(() => new URLSearchParams(location.search).get('room') ?? '')
  const [independent, setIndependent] = useState(false)
  const opening = code.trim() === ''

  function enter(submitted: FormEvent) {
    submitted.preventDefault()
    const player = name.trim()
    table.enter(
      opening
        ? { type: 'create_room', name: player, options: { independent_reactions: independent } }
        : { type: 'join', room: code.trim().toUpperCase(), name: player },
    )
  }

  return (
    <main className="mx-auto flex min-h-dvh max-w-[820px] flex-col justify-center gap-10 px-5 py-12">
      <header className="flex flex-col items-center gap-6 text-center">
        <Brand hero />
        <p className="max-w-[560px] text-[17px] leading-relaxed text-muted text-pretty">
          Duas influências, uma corte podre e ninguém pra checar se você está mentindo.
          <br />
          De 2 a 6 pessoas. A última de pé fica com o que sobrou.
        </p>
      </header>

      <form onSubmit={enter} className="flex flex-col gap-[18px]">
        <div className="flex gap-4 max-md:flex-col">
          <label className="flex grow flex-col gap-2">
            <span className="label">seu nome</span>
            <input
              value={name}
              onChange={(typed) => setName(typed.target.value)}
              maxLength={16}
              autoFocus
              className="h-[60px] rounded-2xl border-3 border-ink bg-card px-5 text-xl font-semibold"
            />
          </label>
          <label className="flex w-[236px] flex-col gap-2 max-md:w-full">
            <span className="label">código da mesa</span>
            <input
              value={code}
              onChange={(typed) => setCode(typed.target.value.toUpperCase())}
              maxLength={4}
              placeholder="····"
              className="h-[60px] rounded-2xl border-3 border-ink bg-gold px-4 text-center font-mono text-2xl font-bold tracking-[0.3em] uppercase"
            />
          </label>
        </div>
        <button
          className="btn btn-primary min-h-[62px] w-full text-[19px]"
          disabled={name.trim().length < 2 || (!opening && code.trim().length !== 4)}
        >
          {opening ? 'abrir mesa nova' : 'entrar na mesa'}
        </button>
        <p className="text-center text-sm font-medium text-muted">deixe o código vazio para abrir uma mesa nova</p>
        {opening && (
          <label className="flex items-center justify-center gap-2.5 text-sm font-medium text-muted">
            <input
              type="checkbox"
              checked={independent}
              onChange={(ticked) => setIndependent(ticked.target.checked)}
              className="size-5 accent-ink"
            />
            regra da casa: quem contesta ainda pode bloquear
          </label>
        )}
        {table.reconnecting && <Banner tone="gold">reconectando à mesa…</Banner>}
        {table.refusal && <Banner>{table.refusal}</Banner>}
      </form>

      <footer className="flex flex-col items-center gap-6">
        <div className="flex gap-3.5 text-white" aria-hidden="true">
          {cardNames.map((card) => (
            <span key={card} className={`flex size-[54px] items-center justify-center rounded-2xl border-3 border-ink ${tint[card]}`}>
              <Glyph card={card} size={30} />
            </span>
          ))}
        </div>
        <p className="text-center text-xs leading-[1.7] text-faint">
          Coup é de Rikki Tahta, publicado por La Mame Games e Indie Boards &amp; Cards; no Brasil pela Mandala Jogos.
          <br />
          Implementação independente das regras, sem arte da caixa. Código sob licença MIT.
        </p>
      </footer>
    </main>
  )
}
