import type { ReactNode } from 'react'
import { Alert, AlertDescription } from './ui/alert'

export function Brand({ hero = false }: { hero?: boolean }) {
  if (hero) {
    return (
      <h1 className="inline-block rounded-[26px] bg-primary px-8 py-3.5 text-[clamp(52px,12vw,76px)] leading-none font-extrabold text-primary-foreground shadow-gold">
        Coup
      </h1>
    )
  }
  return (
    <span className="inline-block rounded-[14px] bg-primary px-5 py-2 text-[30px] leading-none font-extrabold text-primary-foreground max-md:rounded-[11px] max-md:px-3 max-md:text-xl">
      Coup
    </span>
  )
}

export function Notice({ children, tone = 'destructive' }: { children: ReactNode; tone?: 'destructive' | 'default' }) {
  return (
    <Alert variant={tone}>
      <AlertDescription className="text-[15px] font-semibold">{children}</AlertDescription>
    </Alert>
  )
}
