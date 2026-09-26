import * as React from 'react'
import { cn } from '@/lib/utils'

function Input({ className, type, ...props }: React.ComponentProps<'input'>) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        'h-[60px] w-full min-w-0 rounded-2xl border-3 border-input bg-card px-5 text-xl font-semibold outline-none selection:bg-gold placeholder:text-faint disabled:cursor-not-allowed disabled:opacity-50',
        'focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-ring',
        'aria-invalid:border-destructive',
        className,
      )}
      {...props}
    />
  )
}

export { Input }
