import * as React from 'react'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const badgeVariants = cva(
  'inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium transition-colors',
  {
    variants: {
      variant: {
        default:   'border-[#3B82F6]/20 bg-[#3B82F6]/10 text-[#60A5FA]',
        secondary: 'border-[#2D3748] bg-[#252A34] text-[#D1D5DB]',
        destructive: 'border-[#EF4444]/20 bg-[#EF4444]/10 text-[#F87171]',
        outline:   'border-[#2D3748] text-[#9CA3AF]',
        success:   'border-[#22C55E]/20 bg-[#22C55E]/10 text-[#34D399]',
        warning:   'border-[#F59E0B]/20 bg-[#F59E0B]/10 text-[#FBBF24]',
        info:      'border-[#3B82F6]/20 bg-[#3B82F6]/10 text-[#60A5FA]',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <div className={cn(badgeVariants({ variant }), className)} {...props} />
}

export { Badge, badgeVariants }
