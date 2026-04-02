import { cn } from '../lib/utils'

interface ProgressBarProps {
  value: number // 0–1
  className?: string
  size?: 'sm' | 'md'
}

export function ProgressBar({ value, className, size = 'md' }: ProgressBarProps) {
  const pct = Math.min(100, Math.max(0, value * 100))
  return (
    <div className={cn('w-full bg-stone-200 rounded-full overflow-hidden',
      size === 'sm' ? 'h-1' : 'h-2',
      className,
    )}>
      <div
        className="h-full bg-amber-500 rounded-full transition-all duration-300"
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}
