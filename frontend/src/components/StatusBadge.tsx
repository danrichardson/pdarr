import { cn } from '../lib/utils'
import type { Job } from '../lib/api'

const config: Record<Job['Status'], { label: string; className: string }> = {
  pending:   { label: 'Pending',   className: 'bg-stone-100 text-stone-600' },
  running:   { label: 'Running',   className: 'bg-amber-100 text-amber-700' },
  done:      { label: 'Done',      className: 'bg-green-100 text-green-700' },
  failed:    { label: 'Failed',    className: 'bg-red-100 text-red-700' },
  cancelled: { label: 'Cancelled', className: 'bg-stone-100 text-stone-500' },
  skipped:   { label: 'Skipped',   className: 'bg-stone-100 text-stone-500' },
}

export function StatusBadge({ status }: { status: Job['Status'] }) {
  const { label, className } = config[status] ?? config.pending
  return (
    <span className={cn('inline-flex items-center px-2 py-0.5 rounded text-xs font-medium', className)}>
      {label}
    </span>
  )
}
