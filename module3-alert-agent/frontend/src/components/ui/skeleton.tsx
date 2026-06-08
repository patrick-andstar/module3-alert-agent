import { cn } from '@/lib/utils'

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('rounded skeleton-pulse bg-[#252A34]', className)} {...props} />
}

export { Skeleton }
