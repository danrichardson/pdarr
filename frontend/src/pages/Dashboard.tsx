import { useEffect, useState, useCallback } from 'react'
import { api, type Status, type Job } from '../lib/api'
import { useSSE } from '../hooks/useSSE'
import { Card, CardTitle } from '../components/Card'
import { ProgressBar } from '../components/ProgressBar'
import { StatusBadge } from '../components/StatusBadge'
import { formatBytes, formatDuration, basename } from '../lib/utils'
import { Play, Pause, RefreshCw, HardDrive, Cpu, CheckCircle, AlertCircle } from 'lucide-react'

export function Dashboard() {
  const [status, setStatus] = useState<Status | null>(null)
  const [runningJob, setRunningJob] = useState<Job | null>(null)
  const [scanning, setScanning] = useState(false)

  const load = useCallback(async () => {
    try {
      const [s, jobs] = await Promise.all([
        api.getStatus(),
        api.listJobs('running', 1, 0),
      ])
      setStatus(s)
      setRunningJob(jobs?.[0] ?? null)
    } catch {}
  }, [])

  useEffect(() => { load() }, [load])

  useSSE((e) => {
    if (e.Type === 'progress' && runningJob?.ID === e.JobID) {
      setRunningJob(j => j ? { ...j, Progress: e.Progress } : j)
    }
    if (e.Type === 'done' || e.Type === 'failed') {
      load()
    }
  })

  const handleScanNow = async () => {
    setScanning(true)
    try { await api.triggerScan() } finally {
      setTimeout(() => setScanning(false), 2000)
    }
  }

  const handleTogglePause = async () => {
    if (!status) return
    if (status.paused) await api.resumeQueue()
    else await api.pauseQueue()
    await load()
  }

  const diskPct = status
    ? Math.min(100, (status.disk_free_gb / Math.max(1, status.disk_free_gb + status.pause_threshold)) * 100)
    : 0

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-stone-900">Dashboard</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={handleTogglePause}
            disabled={!status}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md border border-stone-300 text-stone-700 hover:bg-stone-50 disabled:opacity-40 transition-colors"
          >
            {status?.paused ? <Play size={14} /> : <Pause size={14} />}
            {status?.paused ? 'Resume' : 'Pause'}
          </button>
          <button
            onClick={handleScanNow}
            disabled={scanning}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md bg-stone-800 text-white hover:bg-stone-700 disabled:opacity-40 transition-colors"
          >
            <RefreshCw size={14} className={scanning ? 'animate-spin' : ''} />
            Scan Now
          </button>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <CardTitle>Space Saved</CardTitle>
          <p className="text-2xl font-semibold text-stone-900">
            {status ? formatBytes(status.total_saved_gb * 1024 * 1024 * 1024) : '—'}
          </p>
        </Card>
        <Card>
          <CardTitle>Jobs Done</CardTitle>
          <p className="text-2xl font-semibold text-stone-900">{status?.jobs_done ?? '—'}</p>
        </Card>
        <Card>
          <CardTitle>Failed</CardTitle>
          <p className="text-2xl font-semibold text-red-600">{status?.jobs_failed ?? '—'}</p>
        </Card>
        <Card>
          <CardTitle>Status</CardTitle>
          <div className="flex items-center gap-2 mt-1">
            <span className={`w-2 h-2 rounded-full ${status?.paused ? 'bg-stone-400' : 'bg-green-500'}`} />
            <span className="text-sm font-medium text-stone-700">
              {status?.paused ? 'Paused' : 'Running'}
            </span>
          </div>
        </Card>
      </div>

      {/* Active job */}
      {runningJob && (
        <Card>
          <div className="flex items-start justify-between mb-3">
            <div className="min-w-0">
              <p className="text-xs text-stone-500 mb-1 flex items-center gap-1">
                <Cpu size={12} /> Transcoding
              </p>
              <p className="text-sm font-medium text-stone-900 truncate">
                {basename(runningJob.SourcePath)}
              </p>
              <p className="text-xs text-stone-500 mt-0.5 truncate">{runningJob.SourcePath}</p>
            </div>
            <StatusBadge status={runningJob.Status} />
          </div>
          <ProgressBar value={runningJob.Progress} />
          <p className="text-xs text-stone-500 mt-1.5">
            {Math.round(runningJob.Progress * 100)}% &middot;{' '}
            {formatDuration(runningJob.SourceDuration)} &middot;{' '}
            {formatBytes(runningJob.SourceSize)}
          </p>
        </Card>
      )}

      {/* Disk space */}
      <Card>
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2 text-sm font-medium text-stone-700">
            <HardDrive size={15} />
            Disk Space
          </div>
          <span className="text-sm text-stone-500">
            {status ? `${status.disk_free_gb.toFixed(1)} GB free` : '—'}
          </span>
        </div>
        <ProgressBar value={diskPct / 100} className="mb-1" />
        {status && status.disk_free_gb < status.pause_threshold && (
          <p className="text-xs text-amber-600 mt-1.5 flex items-center gap-1">
            <AlertCircle size={12} />
            Free space below {status.pause_threshold} GB threshold — queue paused
          </p>
        )}
        {status && status.disk_free_gb >= status.pause_threshold && (
          <p className="text-xs text-stone-400 mt-1.5 flex items-center gap-1">
            <CheckCircle size={12} />
            Pauses when below {status.pause_threshold} GB
          </p>
        )}
      </Card>

      {/* Encoder info */}
      {status && (
        <p className="text-xs text-stone-400 flex items-center gap-1">
          <Cpu size={12} />
          {status.encoder}
        </p>
      )}
    </div>
  )
}
