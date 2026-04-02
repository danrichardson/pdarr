import { useEffect, useState, useCallback } from 'react'
import { api, type Job } from '../lib/api'
import { StatusBadge } from '../components/StatusBadge'
import { formatBytes, timeAgo, basename } from '../lib/utils'
import { ChevronDown, ChevronUp, RotateCcw } from 'lucide-react'

const PAGE_SIZE = 50

export function History() {
  const [jobs, setJobs] = useState<Job[]>([])
  const [offset, setOffset] = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [expanded, setExpanded] = useState<Set<number>>(new Set())

  const load = useCallback(async (reset = false) => {
    const off = reset ? 0 : offset
    try {
      const results = await api.listJobs(undefined, PAGE_SIZE, off)
      const list = results ?? []
      if (reset) {
        setJobs(list)
        setOffset(list.length)
      } else {
        setJobs(j => [...j, ...list])
        setOffset(o => o + list.length)
      }
      setHasMore(list.length === PAGE_SIZE)
    } catch {}
  }, [offset])

  useEffect(() => { load(true) }, [])

  const retry = async (id: number) => {
    try {
      await api.retryJob(id)
      load(true)
    } catch {}
  }

  const toggle = (id: number) =>
    setExpanded(s => {
      const next = new Set(s)
      if (next.has(id)) next.delete(id); else next.add(id)
      return next
    })

  return (
    <div className="p-6">
      <h1 className="text-xl font-semibold text-stone-900 mb-6">History</h1>

      {jobs.length === 0 ? (
        <p className="text-sm text-stone-400 py-8 text-center">No jobs yet</p>
      ) : (
        <div className="space-y-2">
          {jobs.map(job => {
            const isExpanded = expanded.has(job.ID)
            const saved = job.BytesSaved?.Valid ? job.BytesSaved.Int64 : null
            const errMsg = job.ErrorMessage?.Valid ? job.ErrorMessage.String : null

            return (
              <div
                key={job.ID}
                className="bg-white border border-stone-200 rounded-lg overflow-hidden"
              >
                <div
                  className="flex items-center gap-3 px-4 py-3 cursor-pointer hover:bg-stone-50"
                  onClick={() => toggle(job.ID)}
                >
                  <div className="flex-1 min-w-0">
                    <p className="text-sm text-stone-900 truncate">{basename(job.SourcePath)}</p>
                    <p className="text-xs text-stone-400">
                      {timeAgo(job.CreatedAt)}
                      {saved !== null && saved > 0 && (
                        <> &middot; <span className="text-green-600">saved {formatBytes(saved)}</span></>
                      )}
                    </p>
                  </div>
                  <StatusBadge status={job.Status} />
                  {(job.Status === 'failed' || job.Status === 'cancelled') && (
                    <button
                      onClick={e => { e.stopPropagation(); retry(job.ID) }}
                      className="text-stone-400 hover:text-amber-600 transition-colors"
                      title="Retry"
                    >
                      <RotateCcw size={14} />
                    </button>
                  )}
                  {isExpanded ? <ChevronUp size={14} className="text-stone-400" /> : <ChevronDown size={14} className="text-stone-400" />}
                </div>

                {isExpanded && (
                  <div className="border-t border-stone-100 px-4 py-3 bg-stone-50 text-xs text-stone-600 space-y-1">
                    <p><span className="text-stone-400">Path:</span> {job.SourcePath}</p>
                    <p><span className="text-stone-400">Codec:</span> {job.SourceCodec}</p>
                    <p><span className="text-stone-400">Size:</span> {formatBytes(job.SourceSize)}</p>
                    {job.EncoderUsed?.Valid && (
                      <p><span className="text-stone-400">Encoder:</span> {job.EncoderUsed.String}</p>
                    )}
                    {errMsg && (
                      <p className="text-red-600"><span className="text-stone-400">Error:</span> {errMsg}</p>
                    )}
                  </div>
                )}
              </div>
            )
          })}

          {hasMore && (
            <button
              onClick={() => load()}
              className="w-full py-2 text-sm text-stone-500 hover:text-stone-700 transition-colors"
            >
              Load more
            </button>
          )}
        </div>
      )}
    </div>
  )
}
