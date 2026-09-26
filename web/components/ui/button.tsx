import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { Slot } from 'radix-ui'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  "inline-flex shrink-0 cursor-pointer items-center justify-center gap-2.5 rounded-full border-3 border-border font-bold whitespace-nowrap transition-[translate,box-shadow] duration-75 outline-none focus-visible:outline-3 focus-visible:outline-offset-3 focus-visible:outline-ring enabled:hover:-translate-x-px enabled:hover:-translate-y-px enabled:active:translate-x-[3px] enabled:active:translate-y-[3px] enabled:active:shadow-none disabled:cursor-not-allowed disabled:border-edge disabled:bg-secondary disabled:text-faint disabled:shadow-none [&_svg]:pointer-events-none [&_svg]:shrink-0 [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground shadow-press enabled:hover:shadow-lift',
        outline: 'bg-card text-foreground shadow-press enabled:hover:shadow-lift',
        challenge: 'bg-primary text-primary-foreground shadow-[3px_3px_0_var(--blood)]',
        duke: 'bg-duke text-white shadow-press enabled:hover:shadow-lift',
        assassin: 'bg-assassin text-white shadow-press enabled:hover:shadow-lift',
        captain: 'bg-captain text-white shadow-press enabled:hover:shadow-lift',
        ambassador: 'bg-ambassador text-white shadow-press enabled:hover:shadow-lift',
        contessa: 'bg-contessa text-white shadow-press enabled:hover:shadow-lift',
        bare: 'h-auto rounded-[18px] border-0 bg-transparent p-0 shadow-none enabled:hover:translate-x-0 enabled:hover:-translate-y-1',
      },
      size: {
        default: 'min-h-[50px] px-[22px] text-base',
        md: 'min-h-[54px] px-6 text-[17px]',
        lg: 'min-h-[62px] px-7 text-[19px]',
        bare: '',
      },
    },
    defaultVariants: {
      variant: 'outline',
      size: 'default',
    },
  },
)

function Button({
  className,
  variant = 'outline',
  size = 'default',
  asChild = false,
  ...props
}: React.ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean
  }) {
  const Comp = asChild ? Slot.Root : 'button'

  return (
    <Comp
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
