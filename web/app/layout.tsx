import type { Metadata, Viewport } from 'next'
import type { ReactNode } from 'react'
import './globals.css'

export const metadata: Metadata = {
  title: 'Coup',
  description: 'Duas influências, uma corte podre e ninguém pra checar se você está mentindo.',
  icons: { icon: '/favicon.svg' },
}

export const viewport: Viewport = { themeColor: '#FBF6EA', width: 'device-width', initialScale: 1 }

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="pt-BR">
      <body>{children}</body>
    </html>
  )
}
