import { cn } from '../lib/utils'

interface CardProps {
  className?: string
  children: React.ReactNode
}

export function Card({ className, children }: CardProps) {
  return (
    <div className={cn('bg-white rounded-lg border border-stone-200 p-5', className)}>
      {children}
    </div>
  )
}

export function CardTitle({ children }: { children: React.ReactNode }) {
  return <h2 className="text-sm font-medium text-stone-500 mb-1">{children}</h2>
}
