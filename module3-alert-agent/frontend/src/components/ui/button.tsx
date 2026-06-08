import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#3B82F6]/30 focus-visible:ring-offset-1 focus-visible:ring-offset-[#0F1116] disabled:pointer-events-none disabled:opacity-40 active:scale-[0.98] [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default: 'bg-[#3B82F6] text-[#F9FAFB] hover:bg-[#2563EB]',
        destructive: 'bg-[#EF4444] text-[#F9FAFB] hover:bg-[#DC2626]',
        outline: 'border border-[#2D3748] bg-[#1A1D23] text-[#D1D5DB] hover:bg-[#252A34] hover:border-[#3B4A5E]',
        secondary: 'bg-[#252A34] text-[#D1D5DB] hover:bg-[#2D3748]',
        ghost: 'text-[#D1D5DB] hover:bg-[#252A34]',
        link: 'text-[#60A5FA] underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-9 px-4 py-2',
        sm: 'h-8 rounded-md px-3 text-xs',
        lg: 'h-10 rounded-md px-6',
        icon: 'h-9 w-9',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />
  }
)
Button.displayName = 'Button'

export { Button, buttonVariants }
