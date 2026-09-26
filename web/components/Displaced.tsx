import { Notice } from './Brand'
import { Button } from './ui/button'

export function Displaced({ onReclaim }: { onReclaim: () => void }) {
  return (
    <main className="mx-auto flex min-h-dvh max-w-[560px] flex-col justify-center gap-6 px-5 py-10">
      <Notice tone="default">mesa aberta em outra aba</Notice>
      <p className="text-[17px] leading-relaxed text-muted-foreground">
        Seu assento foi para a outra aba. Continue por lá, ou traga a mesa de volta para cá.
      </p>
      <Button variant="default" size="lg" className="self-start" onClick={onReclaim}>
        jogar nesta aba
      </Button>
    </main>
  )
}
