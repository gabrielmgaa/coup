import { useEffect, useRef } from 'react'
import type { GameEvent } from '@/lib/coup'
import { Narration } from './cards'

export function Log({ log }: { log: GameEvent[] }) {
  const bottom = useRef<HTMLLIElement | null>(null)
  useEffect(() => bottom.current?.scrollIntoView({ block: 'nearest' }), [log.length])

  return (
    <aside className="panel sticky top-4 flex max-h-[calc(100dvh-120px)] flex-col gap-3.5 self-start px-6 py-5 max-lg:static max-lg:max-h-80">
      <div className="flex items-center gap-3">
        <span className="tag">registro</span>
        <span className="grow" />
        <span className="font-mono text-[11px] text-faint">{log.length} jogadas</span>
      </div>
      <ol className="flex flex-col gap-3 overflow-y-auto text-[15px] leading-normal text-muted">
        {log.map((event, position) => {
          const latest = position === log.length - 1
          return (
            <li
              key={event.n}
              ref={latest ? bottom : undefined}
              className={latest ? 'rounded-[14px] border-2 border-ink bg-gold-wash px-3.5 py-2.5 text-ink' : ''}
            >
              <Narration text={event.text} />
            </li>
          )
        })}
      </ol>
    </aside>
  )
}
