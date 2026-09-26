import { useState, type FormEvent } from 'react'
import type { Table } from '@/lib/connection'
import { cardNames, tint } from '@/lib/palette'
import { Brand, Notice } from './Brand'
import { Glyph } from './cards'
import { Button } from './ui/button'
import { Checkbox } from './ui/checkbox'
import { Input } from './ui/input'
import { Label } from './ui/label'

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
        <p className="max-w-[560px] text-[17px] leading-relaxed text-pretty text-muted-foreground">
          Duas influências, uma corte podre e ninguém pra checar se você está mentindo.
          <br />
          De 2 a 6 pessoas. A última de pé fica com o que sobrou.
        </p>
      </header>

      <form onSubmit={enter} className="flex flex-col gap-[18px]">
        <div className="flex gap-4 max-md:flex-col">
          <div className="flex grow flex-col gap-2">
            <Label htmlFor="name" className="eyebrow">
              seu nome
            </Label>
            <Input id="name" value={name} onChange={(typed) => setName(typed.target.value)} maxLength={16} autoFocus />
          </div>
          <div className="flex w-[236px] flex-col gap-2 max-md:w-full">
            <Label htmlFor="code" className="eyebrow">
              código da mesa
            </Label>
            <Input
              id="code"
              value={code}
              onChange={(typed) => setCode(typed.target.value.toUpperCase())}
              maxLength={4}
              placeholder="····"
              className="bg-gold px-4 text-center font-mono text-2xl font-bold tracking-[0.3em] uppercase"
            />
          </div>
        </div>
        <Button
          variant="default"
          size="lg"
          className="w-full"
          disabled={name.trim().length < 2 || (!opening && code.trim().length !== 4)}
        >
          {opening ? 'abrir mesa nova' : 'entrar na mesa'}
        </Button>
        <p className="text-center text-sm font-medium text-muted-foreground">deixe o código vazio para abrir uma mesa nova</p>
        {opening && (
          <div className="flex items-center justify-center gap-2.5">
            <Checkbox id="independent" checked={independent} onCheckedChange={(ticked) => setIndependent(ticked === true)} />
            <Label htmlFor="independent" className="text-muted-foreground">
              regra da casa: quem contesta ainda pode bloquear
            </Label>
          </div>
        )}
        {table.reconnecting && <Notice tone="default">reconectando à mesa…</Notice>}
        {table.refusal && <Notice>{table.refusal}</Notice>}
      </form>

      <footer className="flex flex-col items-center gap-6">
        <div className="flex gap-3.5 text-white" aria-hidden="true">
          {cardNames.map((card) => (
            <span key={card} className={`flex size-[54px] items-center justify-center rounded-2xl border-3 border-border ${tint[card]}`}>
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
