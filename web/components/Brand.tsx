export function Brand({ hero = false }: { hero?: boolean }) {
  if (hero) {
    return (
      <h1 className="inline-block rounded-[26px] bg-ink px-8 py-3.5 text-[clamp(52px,12vw,76px)] leading-none font-extrabold text-paper shadow-gold">
        Coup
      </h1>
    )
  }
  return (
    <span className="inline-block rounded-[14px] bg-ink px-5 py-2 text-[30px] leading-none font-extrabold text-paper max-md:rounded-[11px] max-md:px-3 max-md:text-xl">
      Coup
    </span>
  )
}

export function Banner({ children, tone = 'blood' }: { children: React.ReactNode; tone?: 'blood' | 'gold' }) {
  const colors = tone === 'blood' ? 'border-blood bg-blood-wash text-blood' : 'border-ink bg-gold-wash text-ink'
  return <p className={`rounded-[14px] border-2 px-4 py-2.5 font-semibold ${colors}`}>{children}</p>
}
