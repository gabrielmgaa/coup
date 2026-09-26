import { useEffect, useRef } from 'react'
import type { GameEvent } from '@/lib/coup'
import { cn } from '@/lib/utils'
import { Narration } from './cards'
import { Badge } from './ui/badge'
import { Card, CardAction, CardContent, CardHeader } from './ui/card'

export function Log({ log }: { log: GameEvent[] }) {
  const bottom = useRef<HTMLLIElement | null>(null)
  useEffect(() => {
    bottom.current?.scrollIntoView({ block: 'nearest' })
  }, [log.length])

  return (
    <Card className="sticky top-4 max-h-[calc(100dvh-120px)] self-start max-lg:static max-lg:max-h-80">
      <CardHeader>
        <Badge>registro</Badge>
        <CardAction className="font-mono text-[11px] text-faint">{log.length} jogadas</CardAction>
      </CardHeader>
      <CardContent className="min-h-0 overflow-y-auto">
        <ol className="flex flex-col gap-3 text-[15px] leading-normal text-muted-foreground">
          {log.map((event, position) => {
            const latest = position === log.length - 1
            return (
              <li
                key={event.n}
                ref={latest ? bottom : undefined}
                className={cn(latest && 'rounded-[14px] border-2 border-border bg-accent px-3.5 py-2.5 text-foreground')}
              >
                <Narration text={event.text} />
              </li>
            )
          })}
        </ol>
      </CardContent>
    </Card>
  )
}
