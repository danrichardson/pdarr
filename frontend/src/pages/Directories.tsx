import { useEffect, useState, useCallback } from 'react'
import { api, type Directory, type DirectoryInput, type FSBrowseResult } from '../lib/api'
import { formatBitrate, basename } from '../lib/utils'
import { Plus, Pencil, Trash2, X, Check, FolderOpen, ChevronLeft } from 'lucide-react'

interface DirForm extends Required<DirectoryInput> {
  path: string
}

const defaultForm: DirForm = {
  path: '',
  enabled: true,
  min_age_days: 7,
  max_bitrate: 2_222_000,
  min_size_mb: 500,
}

// ---- Directory browser modal ----

interface BrowserProps {
  onSelect: (path: string) => void
  onClose: () => void
  startPath?: string  // initial directory; defaults to server-side first root
}

function DirectoryBrowser({ onSelect, onClose, startPath }: BrowserProps) {
  const [browse, setBrowse] = useState<FSBrowseResult | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const navigate = useCallback(async (path: string) => {
    setLoading(true)
    setError('')
    try {
      const result = await api.browseFS(path)
      setBrowse(result)
    } catch (e: any) {
      setError(e.message || 'Cannot read directory')
    } finally {
      setLoading(false)
    }
  }, [])

  // Open at the first configured root (via server default) or an explicit startPath.
  useEffect(() => { navigate(startPath ?? '') }, [navigate, startPath])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 flex flex-col max-h-[70vh]">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-stone-200">
          <h3 className="text-sm font-semibold text-stone-900">Browse for directory</h3>
          <button onClick={onClose} className="text-stone-400 hover:text-stone-700">
            <X size={16} />
          </button>
        </div>

        {/* Current path */}
        <div className="px-4 py-2 bg-stone-50 border-b border-stone-100">
          <p className="text-xs font-mono text-stone-600 truncate">{browse?.current ?? '/'}</p>
        </div>

        {/* Directory list */}
        <div className="flex-1 overflow-y-auto">
          {loading ? (
            <p className="text-sm text-stone-400 text-center py-8">Loading…</p>
          ) : error ? (
            <p className="text-sm text-red-600 text-center py-8">{error}</p>
          ) : (
            <>
              {browse?.parent && (
                <button
                  onClick={() => navigate(browse.parent)}
                  className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-stone-600 hover:bg-stone-50 border-b border-stone-100"
                >
                  <ChevronLeft size={14} className="text-stone-400" />
                  Parent directory
                </button>
              )}
              {browse?.dirs.length === 0 && (
                <p className="text-sm text-stone-400 text-center py-6">No subdirectories</p>
              )}
              {browse?.dirs.map(dir => (
                <button
                  key={dir}
                  onClick={() => navigate(dir)}
                  className="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-stone-700 hover:bg-stone-50 border-b border-stone-100 last:border-0"
                >
                  <FolderOpen size={14} className="text-amber-500 shrink-0" />
                  <span className="truncate">{basename(dir)}</span>
                </button>
              ))}
            </>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between gap-2 px-4 py-3 border-t border-stone-200">
          <button
            onClick={() => browse?.parent && navigate(browse.parent)}
            disabled={!browse?.parent}
            className="flex items-center gap-1 text-sm text-stone-600 hover:text-stone-900 disabled:opacity-40 transition-colors"
          >
            <ChevronLeft size={14} /> Parent
          </button>
          <button
            onClick={() => browse && onSelect(browse.current)}
            disabled={!browse}
            className="px-3 py-1.5 text-sm bg-amber-500 hover:bg-amber-600 disabled:opacity-50 text-white rounded-md transition-colors"
          >
            Select This Folder
          </button>
        </div>
      </div>
    </div>
  )
}

// ---- Main page ----

export function Directories() {
  const [dirs, setDirs] = useState<Directory[]>([])
  const [editing, setEditing] = useState<Directory | null>(null)
  const [adding, setAdding] = useState(false)
  const [form, setForm] = useState<DirForm>(defaultForm)
  const [error, setError] = useState('')
  const [showBrowser, setShowBrowser] = useState(false)

  const load = useCallback(async () => {
    try { setDirs(await api.listDirectories() ?? []) } catch {}
  }, [])

  useEffect(() => { load() }, [load])

  const startAdd = () => {
    setForm(defaultForm)
    setEditing(null)
    setAdding(true)
    setError('')
  }

  const startEdit = (d: Directory) => {
    setForm({
      path: d.Path,
      enabled: d.Enabled,
      min_age_days: d.MinAgeDays,
      max_bitrate: d.MaxBitrate,
      min_size_mb: d.MinSizeMB,
    })
    setEditing(d)
    setAdding(false)
    setError('')
  }

  const save = async () => {
    setError('')
    if (!form.path) { setError('Path is required'); return }
    try {
      if (editing) {
        await api.updateDirectory(editing.ID, form)
      } else {
        await api.createDirectory(form)
      }
      await load()
      setAdding(false)
      setEditing(null)
    } catch (e: any) {
      setError(e.message ?? 'Save failed')
    }
  }

  const remove = async (id: number) => {
    if (!confirm('Delete this directory? Existing jobs will remain in history.')) return
    try {
      await api.deleteDirectory(id)
      await load()
    } catch (e: any) {
      setError(e.message || 'Delete failed')
    }
  }

  return (
    <div className="p-6 space-y-6">
      {showBrowser && (
        <DirectoryBrowser
          onSelect={path => {
            setForm(f => ({ ...f, path }))
            setShowBrowser(false)
          }}
          onClose={() => setShowBrowser(false)}
          startPath={dirs.find(d => d.Enabled)?.Path ?? dirs[0]?.Path}
        />
      )}

      <div className="flex items-center justify-between">
        <h1 className="text-xl font-semibold text-stone-900">Directories</h1>
        {!adding && !editing && (
          <button
            onClick={startAdd}
            className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md bg-stone-800 text-white hover:bg-stone-700 transition-colors"
          >
            <Plus size={14} />
            Add Directory
          </button>
        )}
      </div>

      {(adding || editing) && (
        <div className="bg-white border border-stone-200 rounded-lg p-5 space-y-4">
          <h2 className="text-sm font-medium text-stone-700">
            {editing ? 'Edit Directory' : 'Add Directory'}
          </h2>

          <div className="space-y-3">
            <label className="block">
              <span className="text-xs font-medium text-stone-600">Path</span>
              <div className="mt-1 flex gap-2">
                <input
                  type="text"
                  value={form.path}
                  onChange={e => setForm(f => ({ ...f, path: e.target.value }))}
                  placeholder="/media/Videos"
                  className="flex-1 min-w-0 rounded-md border border-stone-300 px-3 py-2 text-sm text-stone-900 focus:border-amber-500 focus:ring-amber-500 focus:outline-none"
                />
                <button
                  type="button"
                  onClick={() => setShowBrowser(true)}
                  className="shrink-0 flex items-center gap-1 px-3 py-2 text-sm border border-stone-300 rounded-md text-stone-600 hover:bg-stone-50 transition-colors"
                  title="Browse filesystem"
                >
                  <FolderOpen size={14} />
                </button>
              </div>
            </label>

            <div className="grid grid-cols-3 gap-3">
              <label className="block">
                <span className="text-xs font-medium text-stone-600">Min age (days)</span>
                <input
                  type="number"
                  min={0}
                  value={form.min_age_days}
                  onChange={e => setForm(f => ({ ...f, min_age_days: Number(e.target.value) }))}
                  className="mt-1 block w-full rounded-md border border-stone-300 px-3 py-2 text-sm text-stone-900 focus:border-amber-500 focus:ring-amber-500 focus:outline-none"
                />
              </label>
              <label className="block">
                <span className="text-xs font-medium text-stone-600">Max bitrate (bps)</span>
                <input
                  type="number"
                  min={0}
                  value={form.max_bitrate}
                  onChange={e => setForm(f => ({ ...f, max_bitrate: Number(e.target.value) }))}
                  className="mt-1 block w-full rounded-md border border-stone-300 px-3 py-2 text-sm text-stone-900 focus:border-amber-500 focus:ring-amber-500 focus:outline-none"
                />
              </label>
              <label className="block">
                <span className="text-xs font-medium text-stone-600">Min size (MB)</span>
                <input
                  type="number"
                  min={0}
                  value={form.min_size_mb}
                  onChange={e => setForm(f => ({ ...f, min_size_mb: Number(e.target.value) }))}
                  className="mt-1 block w-full rounded-md border border-stone-300 px-3 py-2 text-sm text-stone-900 focus:border-amber-500 focus:ring-amber-500 focus:outline-none"
                />
              </label>
            </div>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={e => setForm(f => ({ ...f, enabled: e.target.checked }))}
                className="rounded border-stone-300 text-amber-600 focus:ring-amber-500"
              />
              <span className="text-sm text-stone-700">Enabled</span>
            </label>
          </div>

          {error && <p className="text-sm text-red-600">{error}</p>}

          <div className="flex gap-2">
            <button
              onClick={save}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md bg-stone-800 text-white hover:bg-stone-700 transition-colors"
            >
              <Check size={14} />
              Save
            </button>
            <button
              onClick={() => { setAdding(false); setEditing(null) }}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md border border-stone-300 text-stone-700 hover:bg-stone-50 transition-colors"
            >
              <X size={14} />
              Cancel
            </button>
          </div>
        </div>
      )}

      {dirs.length === 0 && !adding ? (
        <p className="text-sm text-stone-400 py-8 text-center">
          No directories configured. Add one to get started.
        </p>
      ) : (
        <div className="space-y-3">
          {dirs.map(d => (
            <div
              key={d.ID}
              className={`bg-white border rounded-lg px-4 py-3 flex items-start gap-3 ${
                d.Enabled ? 'border-stone-200' : 'border-stone-100 opacity-60'
              }`}
            >
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-stone-900 truncate">{d.Path}</p>
                <p className="text-xs text-stone-400 mt-0.5">
                  Age ≥ {d.MinAgeDays}d &middot;{' '}
                  Bitrate &gt; {formatBitrate(d.MaxBitrate)} &middot;{' '}
                  Size ≥ {d.MinSizeMB} MB
                </p>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                {!d.Enabled && (
                  <span className="text-xs text-stone-400 bg-stone-100 px-2 py-0.5 rounded">Disabled</span>
                )}
                <button
                  onClick={() => startEdit(d)}
                  className="text-stone-400 hover:text-stone-700 transition-colors"
                >
                  <Pencil size={14} />
                </button>
                <button
                  onClick={() => remove(d.ID)}
                  className="text-stone-400 hover:text-red-500 transition-colors"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
