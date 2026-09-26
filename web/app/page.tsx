'use client'

import dynamic from 'next/dynamic'

const Coup = dynamic(() => import('@/components/Coup').then((loaded) => loaded.Coup), { ssr: false })

export default function Page() {
  return <Coup />
}
