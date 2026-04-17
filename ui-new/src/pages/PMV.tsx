import { useState, useMemo, useRef, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Disc3, Radio, Wand2, Sparkles, Play, ListMusic, FileAudio,
  CheckCircle2, AlertTriangle, Loader2, ExternalLink,
  ChevronRight, FileText, Image as ImageIcon,
  Building2, Type as TypeIcon, Link2, MoreHorizontal, Activity,
  CircleAlert, X,
} from 'lucide-react'
import {
  startPMVMatchTask, importPMVVideo, importPMVList,
  type PMVImportResult, type PMVImportBatchResult, type PMVImportBatchItem,
  getStorage, type Volume,
} from '../api/client'

type ConsoleMode = 'match' | 'video' | 'collection'

interface ActivityEntry {
  id: number
  ts: Date
  kind: 'match' | 'video' | 'collection'
  status: 'success' | 'warning' | 'error' | 'running'
  title: string
  detail?: string
  meta?: { label: string; value: string | number }[]
}

function Equalizer({ active = false, bars = 14 }: { active?: boolean; bars?: number }) {
  const heights = useMemo(
    () => Array.from({ length: bars }, () => 30 + Math.random() * 70),
    [bars]
  )
  return (
    <div className="flex items-end gap-[3px] h-full">
      {heights.map((h, i) => (
        <span
          key={i}
          className="w-[3px] rounded-sm bg-gradient-to-t from-cyber-pink/30 via-accent/60 to-cyber-amber"
          style={{
            height: `${h}%`,
            animation: active
              ? `eqBar ${0.6 + (i % 5) * 0.18}s ease-in-out ${i * 0.04}s infinite alternate`
              : 'none',
            opacity: active ? 1 : 0.35,
          }}
        />
      ))}
    </div>
  )
}

function StatusDot({ status }: { status: 'idle' | 'running' | 'success' | 'error' }) {
  const tones = {
    idle: 'bg-surface-500',
    running: 'bg-cyber-amber animate-pulse',
    success: 'bg-cyber-teal',
    error: 'bg-cyber-red',
  } as const
  return (
    <span className="relative flex h-2 w-2">
      <span className={`absolute inline-flex h-full w-full rounded-full opacity-60 ${tones[status]}`} />
      <span className={`relative inline-flex rounded-full h-2 w-2 ${tones[status]}`} />
    </span>
  )
}

function ToggleField({
  label, hint, value, onChange,
}: { label: string; hint?: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <button
      onClick={() => onChange(!value)}
      className={`group flex items-center justify-between w-full text-left rounded-lg
        px-3 py-2.5 border transition-all
        ${value
          ? 'border-cyber-pink/30 bg-cyber-pink/5 hover:bg-cyber-pink/10'
          : 'border-surface-700/40 bg-surface-800/40 hover:border-surface-600/50'
        }`}
    >
      <div>
        <div className="text-sm text-surface-100 font-medium">{label}</div>
        {hint && <div className="text-xs text-surface-500 mt-0.5">{hint}</div>}
      </div>
      <div className={`relative w-9 h-5 rounded-full transition-colors flex-shrink-0
        ${value ? 'bg-cyber-pink' : 'bg-surface-700'}`}>
        <span className={`absolute top-0.5 w-4 h-4 rounded-full bg-white shadow-sm transition-transform
          ${value ? 'translate-x-[18px]' : 'translate-x-0.5'}`} />
      </div>
    </button>
  )
}

function NumberField({
  label, value, onChange, min, max, suffix, hint,
}: {
  label: string; value: number | null; onChange: (n: number | null) => void
  min?: number; max?: number; suffix?: string; hint?: string
}) {
  return (
    <label className="block">
      <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
        {label}
      </div>
      <div className="relative">
        <input
          type="number"
          min={min}
          max={max}
          value={value ?? ''}
          onChange={(e) => {
            const v = e.target.value.trim()
            if (v === '') return onChange(null)
            const n = Number(v)
            onChange(Number.isFinite(n) ? n : null)
          }}
          className="input-dark w-full pr-10 font-mono text-sm tabular-nums"
        />
        {suffix && (
          <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-surface-500 font-mono">
            {suffix}
          </span>
        )}
      </div>
      {hint && <div className="text-[11px] text-surface-500 mt-1">{hint}</div>}
    </label>
  )
}

function TextField({
  label, value, onChange, placeholder, mono = false, hint, leftIcon,
}: {
  label: string; value: string; onChange: (v: string) => void
  placeholder?: string; mono?: boolean; hint?: string
  leftIcon?: React.ReactNode
}) {
  return (
    <label className="block">
      <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
        {label}
      </div>
      <div className="relative">
        {leftIcon && (
          <span className="absolute left-3 top-1/2 -translate-y-1/2 text-surface-500">
            {leftIcon}
          </span>
        )}
        <input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className={`input-dark w-full ${leftIcon ? 'pl-9' : ''} ${mono ? 'font-mono text-sm' : ''}`}
        />
      </div>
      {hint && <div className="text-[11px] text-surface-500 mt-1">{hint}</div>}
    </label>
  )
}

const modes: { id: ConsoleMode; label: string; sub: string; icon: typeof Disc3; tone: string }[] = [
  { id: 'match', label: 'Match Library', sub: 'PMVHaven · unmatched files', icon: Wand2, tone: 'cyber-pink' },
  { id: 'video', label: 'Import Video', sub: 'Single PMVHaven URL', icon: Play, tone: 'cyber-amber' },
  { id: 'collection', label: 'Import Collection', sub: 'Profile · ranked list', icon: ListMusic, tone: 'cyber-teal' },
]

let _entryId = 0
const nextId = () => ++_entryId

export default function PMV() {
  const [mode, setMode] = useState<ConsoleMode>('match')

  // Match state
  const [dryRun, setDryRun] = useState(true)
  const [refreshExisting, setRefreshExisting] = useState(false)
  const [matchLimit, setMatchLimit] = useState<number | null>(20)
  const [concurrency, setConcurrency] = useState<number | null>(10)
  const [pathPrefix, setPathPrefix] = useState('/mnt/g/Videos')
  const [volumeId, setVolumeId] = useState<number | null>(0)
  const [updateTitle, setUpdateTitle] = useState(true)
  const [updateStudio, setUpdateStudio] = useState(true)
  const [updateSceneURL, setUpdateSceneURL] = useState(true)
  const [updateThumbnail, setUpdateThumbnail] = useState(true)
  const [updateDescription, setUpdateDescription] = useState(true)

  // Video import state
  const [importURL, setImportURL] = useState('')

  // Collection import state
  const [listURL, setListURL] = useState('')
  const [listLimit, setListLimit] = useState<number | null>(null)
  const [listConcurrency, setListConcurrency] = useState<number | null>(3)

  // System state
  const [busy, setBusy] = useState<ConsoleMode | null>(null)
  const [activity, setActivity] = useState<ActivityEntry[]>([])
  const [lastBatch, setLastBatch] = useState<PMVImportBatchResult | null>(null)
  const [lastSingle, setLastSingle] = useState<PMVImportResult | null>(null)
  const [volumes, setVolumes] = useState<Volume[]>([])

  const feedRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    getStorage()
      .then((d) => setVolumes(d.volumes || []))
      .catch(() => {})
  }, [])

  const pushActivity = (e: Omit<ActivityEntry, 'id' | 'ts'>) => {
    setActivity((prev) => [{ id: nextId(), ts: new Date(), ...e }, ...prev].slice(0, 25))
    requestAnimationFrame(() => feedRef.current?.scrollTo({ top: 0, behavior: 'smooth' }))
  }

  const runMatch = async () => {
    setBusy('match')
    pushActivity({
      kind: 'match',
      status: 'running',
      title: refreshExisting ? 'Refreshing PMV metadata' : 'Matching unmatched files',
      detail: dryRun ? 'Dry run — no DB writes' : 'Live mode',
    })
    try {
      await startPMVMatchTask({
        dryRun,
        refreshExisting,
        limit: matchLimit ?? 20,
        concurrency: concurrency ?? 10,
        pathPrefix: pathPrefix.trim(),
        volumeId: volumeId ?? 0,
        updateTitle, updateStudio, updateSceneURL, updateThumbnail, updateDescription,
      })
      pushActivity({
        kind: 'match',
        status: 'success',
        title: refreshExisting ? 'Metadata refresh task started' : 'PMV match task started',
        detail: 'Streaming progress is visible in the Logs page.',
      })
    } catch {
      pushActivity({
        kind: 'match',
        status: 'error',
        title: 'Failed to start task',
        detail: 'Backend rejected the request.',
      })
    } finally {
      setBusy(null)
    }
  }

  const runImportVideo = async () => {
    if (!importURL.trim()) return
    setBusy('video')
    setLastSingle(null)
    pushActivity({
      kind: 'video',
      status: 'running',
      title: 'Importing PMVHaven video',
      detail: importURL,
    })
    try {
      const result = await importPMVVideo({
        url: importURL.trim(),
        path_prefix: pathPrefix.trim() || undefined,
      })
      setLastSingle(result)
      pushActivity({
        kind: 'video',
        status: result.skipped ? 'warning' : 'success',
        title: result.skipped ? 'Skipped (already imported)' : 'PMV imported',
        detail: result.message || result.scene_url || result.url,
        meta: [
          result.scene_id ? { label: 'scene', value: result.scene_id } : null,
          result.file_id ? { label: 'file', value: String(result.file_id) } : null,
          result.funscript_generated ? { label: 'funscript', value: 'generated' }
            : result.funscript_downloaded ? { label: 'funscript', value: 'downloaded' } : null,
        ].filter(Boolean) as { label: string; value: string | number }[],
      })
    } catch {
      pushActivity({
        kind: 'video',
        status: 'error',
        title: 'Failed to import video',
        detail: importURL,
      })
    } finally {
      setBusy(null)
    }
  }

  const runImportList = async () => {
    if (!listURL.trim()) return
    setBusy('collection')
    setLastBatch(null)
    pushActivity({
      kind: 'collection',
      status: 'running',
      title: 'Crawling PMVHaven list',
      detail: listURL,
    })
    try {
      const result = await importPMVList({
        list_url: listURL.trim(),
        path_prefix: pathPrefix.trim() || undefined,
        limit: listLimit ?? 0,
        concurrency: listConcurrency ?? 3,
      })
      setLastBatch(result)
      pushActivity({
        kind: 'collection',
        status: result.errors > 0 ? 'warning' : 'success',
        title: 'Collection import complete',
        meta: [
          { label: 'imported', value: result.imported },
          { label: 'skipped', value: result.skipped_existing },
          { label: 'funscripts', value: result.funscripts_generated },
          { label: 'errors', value: result.errors },
        ],
      })
    } catch {
      pushActivity({
        kind: 'collection',
        status: 'error',
        title: 'Collection import failed',
        detail: listURL,
      })
    } finally {
      setBusy(null)
    }
  }

  const activeMode = modes.find((m) => m.id === mode)!
  const ActiveIcon = activeMode.icon

  return (
    <div className="space-y-6">
      {/* Console header */}
      <div className="relative overflow-hidden rounded-2xl border border-surface-700/50
        bg-gradient-to-br from-surface-900 via-surface-800/80 to-surface-900">
        {/* Background equalizer */}
        <div className="pointer-events-none absolute inset-y-0 right-0 w-1/2 opacity-20">
          <div className="absolute inset-0 bg-gradient-to-l from-cyber-pink/30 via-accent/10 to-transparent" />
          <div className="absolute inset-y-6 right-8 left-1/4">
            <Equalizer active={busy !== null} bars={48} />
          </div>
        </div>
        {/* Grain texture */}
        <div className="pointer-events-none absolute inset-0 opacity-[0.04]"
          style={{ backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='200' height='200'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2'/%3E%3C/filter%3E%3Crect width='200' height='200' filter='url(%23n)'/%3E%3C/svg%3E")` }} />

        <div className="relative px-6 py-7 flex items-center gap-5">
          <div className="relative">
            <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-cyber-pink via-accent to-cyber-amber p-[2px]">
              <div className="w-full h-full rounded-[14px] bg-surface-900 flex items-center justify-center">
                <Disc3 className="w-7 h-7 text-cyber-pink" style={{
                  animation: busy ? 'spin 3s linear infinite' : 'none',
                }} />
              </div>
            </div>
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 text-[11px] uppercase tracking-[0.18em] text-cyber-pink font-medium">
              <span>PMV Console</span>
              <span className="text-surface-600">·</span>
              <StatusDot status={busy ? 'running' : 'idle'} />
              <span className="text-surface-500 normal-case tracking-normal">
                {busy ? 'task in flight' : 'ready'}
              </span>
            </div>
            <h1 className="mt-1 text-2xl font-bold text-surface-50 tracking-tight">
              <span className="text-cyber-pink">P</span>MV
              <span className="text-cyber-amber">/</span>
              Haven
              <span className="text-surface-500 font-light"> matching · scraping · ingest</span>
            </h1>
            <p className="text-sm text-surface-400 mt-0.5">
              Match unmatched files against PMVHaven, pull individual videos, or crawl ranked lists.
            </p>
          </div>
        </div>

        {/* Mode transport */}
        <div className="relative border-t border-surface-700/40 px-2 py-2 flex gap-1">
          {modes.map((m) => {
            const Icon = m.icon
            const active = mode === m.id
            return (
              <button
                key={m.id}
                onClick={() => setMode(m.id)}
                className={`group flex-1 flex items-center gap-3 rounded-xl px-4 py-3 transition-all
                  ${active
                    ? 'bg-surface-800/80 border border-surface-700/60 shadow-inner'
                    : 'border border-transparent hover:bg-surface-800/40'
                  }`}
              >
                <div className={`relative w-9 h-9 rounded-lg flex items-center justify-center transition-all
                  ${active
                    ? `bg-${m.tone}/15 text-${m.tone} ring-1 ring-${m.tone}/30`
                    : 'bg-surface-700/40 text-surface-400 group-hover:text-surface-200'
                  }`}>
                  <Icon className="w-4 h-4" />
                  {active && (
                    <span className={`absolute -bottom-[3px] left-1/2 -translate-x-1/2 w-1 h-1 rounded-full bg-${m.tone}`} />
                  )}
                </div>
                <div className="text-left min-w-0">
                  <div className={`text-sm font-semibold truncate ${active ? 'text-surface-50' : 'text-surface-300'}`}>
                    {m.label}
                  </div>
                  <div className="text-[11px] text-surface-500 truncate">{m.sub}</div>
                </div>
              </button>
            )
          })}
        </div>
      </div>

      {/* Main grid: form + activity */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Form column */}
        <div className="lg:col-span-2 space-y-6">
          <AnimatePresence mode="wait">
            <motion.div
              key={mode}
              initial={{ opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -4 }}
              transition={{ duration: 0.18 }}
              className="glass rounded-2xl overflow-hidden"
            >
              {/* Form header */}
              <div className="flex items-center gap-3 px-6 py-4 border-b border-surface-700/40
                bg-gradient-to-r from-surface-800/40 to-transparent">
                <ActiveIcon className={`w-5 h-5 text-${activeMode.tone}`} />
                <div className="flex-1">
                  <div className="text-sm font-semibold text-surface-100">{activeMode.label}</div>
                  <div className="text-xs text-surface-500">{activeMode.sub}</div>
                </div>
                {busy === mode && (
                  <Loader2 className="w-4 h-4 text-cyber-amber animate-spin" />
                )}
              </div>

              {mode === 'match' && (
                <div className="p-6 space-y-5">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <ToggleField
                      label="Dry run"
                      hint="Preview matches without writing to the DB."
                      value={dryRun}
                      onChange={setDryRun}
                    />
                    <ToggleField
                      label="Refresh existing PMV scenes"
                      hint="Re-pull metadata for scenes that already have a PMVHaven URL."
                      value={refreshExisting}
                      onChange={setRefreshExisting}
                    />
                  </div>

                  <AnimatePresence initial={false}>
                    {refreshExisting ? (
                      <motion.div
                        key="refresh"
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: 'auto' }}
                        exit={{ opacity: 0, height: 0 }}
                        className="overflow-hidden"
                      >
                        <div className="rounded-xl border border-surface-700/40 bg-surface-800/30 p-4">
                          <div className="text-xs uppercase tracking-wider text-surface-500 mb-3 flex items-center gap-2">
                            <Sparkles className="w-3.5 h-3.5 text-cyber-pink" />
                            Fields to refresh
                          </div>
                          <div className="grid grid-cols-2 md:grid-cols-5 gap-2">
                            {[
                              { label: 'Title', val: updateTitle, set: setUpdateTitle, icon: TypeIcon },
                              { label: 'Studio', val: updateStudio, set: setUpdateStudio, icon: Building2 },
                              { label: 'URL', val: updateSceneURL, set: setUpdateSceneURL, icon: Link2 },
                              { label: 'Thumbnail', val: updateThumbnail, set: setUpdateThumbnail, icon: ImageIcon },
                              { label: 'Description', val: updateDescription, set: setUpdateDescription, icon: FileText },
                            ].map(({ label, val, set, icon: I }) => (
                              <button
                                key={label}
                                onClick={() => set(!val)}
                                className={`group flex flex-col items-center gap-1.5 py-3 rounded-lg border transition-all
                                  ${val
                                    ? 'border-cyber-pink/40 bg-cyber-pink/10 text-cyber-pink'
                                    : 'border-surface-700/50 bg-surface-800/40 text-surface-500 hover:text-surface-300'
                                  }`}
                              >
                                <I className="w-4 h-4" />
                                <span className="text-xs font-medium">{label}</span>
                              </button>
                            ))}
                          </div>
                        </div>
                      </motion.div>
                    ) : (
                      <motion.div
                        key="match-only"
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: 'auto' }}
                        exit={{ opacity: 0, height: 0 }}
                        className="overflow-hidden"
                      >
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                          <NumberField
                            label="Limit"
                            value={matchLimit}
                            onChange={setMatchLimit}
                            min={1}
                            max={500}
                            suffix="files"
                          />
                          <TextField
                            label="Path prefix"
                            value={pathPrefix}
                            onChange={setPathPrefix}
                            placeholder="/mnt/g/Videos"
                            mono
                          />
                          <div>
                            <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
                              Volume
                            </div>
                            <select
                              value={volumeId ?? 0}
                              onChange={(e) => setVolumeId(Number(e.target.value))}
                              className="input-dark w-full text-sm"
                            >
                              <option value={0}>Any volume</option>
                              {volumes.map((v) => (
                                <option key={v.id} value={v.id}>
                                  #{v.id} · {v.path}
                                </option>
                              ))}
                            </select>
                          </div>
                        </div>
                      </motion.div>
                    )}
                  </AnimatePresence>

                  <NumberField
                    label="Concurrency"
                    value={concurrency}
                    onChange={setConcurrency}
                    min={1}
                    max={50}
                    suffix="parallel"
                    hint="Number of files matched simultaneously."
                  />

                  <div className="flex items-center justify-between pt-2 border-t border-surface-700/40">
                    <div className="text-xs text-surface-500 flex items-center gap-1.5">
                      <CircleAlert className="w-3.5 h-3.5" />
                      {dryRun ? 'Safe — no database writes.' : 'Live mode — writes to database.'}
                    </div>
                    <button
                      onClick={runMatch}
                      disabled={busy !== null}
                      className="group relative flex items-center gap-2 px-5 py-2.5 rounded-xl
                        bg-gradient-to-r from-cyber-pink to-accent text-white font-medium text-sm
                        shadow-lg shadow-cyber-pink/20
                        hover:shadow-cyber-pink/40 hover:-translate-y-px
                        active:translate-y-0
                        disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0
                        transition-all"
                    >
                      {busy === 'match'
                        ? <Loader2 className="w-4 h-4 animate-spin" />
                        : <Wand2 className="w-4 h-4" />
                      }
                      {refreshExisting ? 'Run metadata refresh' : 'Run match task'}
                      <ChevronRight className="w-3.5 h-3.5 opacity-60 group-hover:translate-x-0.5 transition-transform" />
                    </button>
                  </div>
                </div>
              )}

              {mode === 'video' && (
                <div className="p-6 space-y-5">
                  <TextField
                    label="PMVHaven video URL"
                    value={importURL}
                    onChange={setImportURL}
                    placeholder="https://pmvhaven.com/video/..."
                    leftIcon={<Link2 className="w-4 h-4" />}
                    hint="Downloads the video, creates a scene, and imports a funscript when one is published."
                  />
                  <TextField
                    label="Destination path prefix"
                    value={pathPrefix}
                    onChange={setPathPrefix}
                    placeholder="/mnt/g/Videos"
                    mono
                    hint="Files are written under this directory inside the matching volume."
                  />

                  <div className="flex items-center justify-end pt-2 border-t border-surface-700/40">
                    <button
                      onClick={runImportVideo}
                      disabled={busy !== null || !importURL.trim()}
                      className="group flex items-center gap-2 px-5 py-2.5 rounded-xl
                        bg-gradient-to-r from-cyber-amber to-cyber-pink text-surface-950 font-semibold text-sm
                        shadow-lg shadow-cyber-amber/20
                        hover:shadow-cyber-amber/40 hover:-translate-y-px
                        active:translate-y-0
                        disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0
                        transition-all"
                    >
                      {busy === 'video'
                        ? <Loader2 className="w-4 h-4 animate-spin" />
                        : <Play className="w-4 h-4 fill-current" />
                      }
                      Download &amp; ingest
                      <ChevronRight className="w-3.5 h-3.5 opacity-70 group-hover:translate-x-0.5 transition-transform" />
                    </button>
                  </div>

                  {lastSingle && <SingleResultCard result={lastSingle} />}
                </div>
              )}

              {mode === 'collection' && (
                <div className="p-6 space-y-5">
                  <TextField
                    label="PMVHaven list or profile URL"
                    value={listURL}
                    onChange={setListURL}
                    placeholder="https://pmvhaven.com/popular?period=month"
                    leftIcon={<Radio className="w-4 h-4" />}
                    hint="Crawls available results, downloading any video that doesn't already exist on disk."
                  />

                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <NumberField
                      label="Top N"
                      value={listLimit}
                      onChange={setListLimit}
                      min={1}
                      max={100}
                      suffix="all"
                      hint="Leave empty to pull every result the list returns."
                    />
                    <NumberField
                      label="Parallel imports"
                      value={listConcurrency}
                      onChange={setListConcurrency}
                      min={1}
                      max={10}
                      suffix="streams"
                    />
                    <TextField
                      label="Path prefix"
                      value={pathPrefix}
                      onChange={setPathPrefix}
                      placeholder="/mnt/g/Videos"
                      mono
                    />
                  </div>

                  <div className="flex items-center justify-end pt-2 border-t border-surface-700/40">
                    <button
                      onClick={runImportList}
                      disabled={busy !== null || !listURL.trim()}
                      className="group flex items-center gap-2 px-5 py-2.5 rounded-xl
                        bg-gradient-to-r from-cyber-teal to-cyber-blue text-surface-950 font-semibold text-sm
                        shadow-lg shadow-cyber-teal/20
                        hover:shadow-cyber-teal/40 hover:-translate-y-px
                        active:translate-y-0
                        disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:translate-y-0
                        transition-all"
                    >
                      {busy === 'collection'
                        ? <Loader2 className="w-4 h-4 animate-spin" />
                        : <ListMusic className="w-4 h-4" />
                      }
                      Crawl &amp; ingest list
                      <ChevronRight className="w-3.5 h-3.5 opacity-70 group-hover:translate-x-0.5 transition-transform" />
                    </button>
                  </div>

                  {lastBatch && <BatchResultCard result={lastBatch} />}
                </div>
              )}
            </motion.div>
          </AnimatePresence>
        </div>

        {/* Activity sidebar */}
        <div className="lg:col-span-1">
          <div className="glass rounded-2xl overflow-hidden h-full flex flex-col max-h-[680px]">
            <div className="flex items-center justify-between px-5 py-4 border-b border-surface-700/40">
              <div className="flex items-center gap-2">
                <Activity className="w-4 h-4 text-cyber-pink" />
                <span className="text-sm font-semibold text-surface-100">Live activity</span>
              </div>
              {activity.length > 0 && (
                <button
                  onClick={() => { setActivity([]); setLastSingle(null); setLastBatch(null) }}
                  className="text-[11px] text-surface-500 hover:text-surface-300 transition-colors"
                >
                  Clear
                </button>
              )}
            </div>

            <div ref={feedRef} className="flex-1 overflow-y-auto p-4 space-y-2.5">
              {activity.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-14 text-center">
                  <div className="w-12 h-12 rounded-full bg-surface-800/60 flex items-center justify-center mb-3">
                    <FileAudio className="w-5 h-5 text-surface-500" />
                  </div>
                  <div className="text-sm text-surface-400 font-medium">No activity yet</div>
                  <div className="text-xs text-surface-500 mt-1 max-w-[200px]">
                    Run a task to see it stream in here.
                  </div>
                </div>
              ) : (
                activity.map((e) => <ActivityCard key={e.id} entry={e} />)
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Inline keyframes for equalizer */}
      <style>{`
        @keyframes eqBar {
          from { transform: scaleY(0.35); }
          to { transform: scaleY(1); }
        }
      `}</style>
    </div>
  )
}

function ActivityCard({ entry }: { entry: ActivityEntry }) {
  const tone = {
    success: { ring: 'ring-cyber-teal/20', bar: 'bg-cyber-teal', icon: <CheckCircle2 className="w-3.5 h-3.5 text-cyber-teal" /> },
    warning: { ring: 'ring-cyber-amber/20', bar: 'bg-cyber-amber', icon: <AlertTriangle className="w-3.5 h-3.5 text-cyber-amber" /> },
    error: { ring: 'ring-cyber-red/20', bar: 'bg-cyber-red', icon: <X className="w-3.5 h-3.5 text-cyber-red" /> },
    running: { ring: 'ring-cyber-pink/30', bar: 'bg-cyber-pink', icon: <Loader2 className="w-3.5 h-3.5 text-cyber-pink animate-spin" /> },
  }[entry.status]

  const time = entry.ts.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })

  return (
    <motion.div
      initial={{ opacity: 0, x: 8 }}
      animate={{ opacity: 1, x: 0 }}
      className={`relative pl-3 rounded-lg ring-1 ${tone.ring} bg-surface-800/40 hover:bg-surface-800/60 transition-colors`}
    >
      <span className={`absolute left-0 top-2 bottom-2 w-[2px] rounded-full ${tone.bar}`} />
      <div className="px-3 py-2.5">
        <div className="flex items-center gap-2 mb-1">
          {tone.icon}
          <span className="text-xs font-medium text-surface-100 truncate">{entry.title}</span>
          <span className="ml-auto text-[10px] text-surface-500 font-mono tabular-nums">{time}</span>
        </div>
        {entry.detail && (
          <div className="text-[11px] text-surface-400 break-all line-clamp-2 font-mono">
            {entry.detail}
          </div>
        )}
        {entry.meta && entry.meta.length > 0 && (
          <div className="flex flex-wrap gap-1 mt-2">
            {entry.meta.map((m, i) => (
              <span key={i} className="badge bg-surface-900/60 border border-surface-700/40 text-[10px] text-surface-300">
                <span className="text-surface-500">{m.label}</span>
                <span className="text-surface-100 font-mono tabular-nums ml-1">{m.value}</span>
              </span>
            ))}
          </div>
        )}
      </div>
    </motion.div>
  )
}

function SingleResultCard({ result }: { result: PMVImportResult }) {
  const isSkipped = result.skipped
  const tone = isSkipped ? 'cyber-amber' : 'cyber-teal'
  const Icon = isSkipped ? AlertTriangle : CheckCircle2

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className={`rounded-xl border border-${tone}/30 bg-${tone}/5 p-4`}
    >
      <div className="flex items-start gap-3">
        <Icon className={`w-5 h-5 text-${tone} flex-shrink-0 mt-0.5`} />
        <div className="flex-1 min-w-0">
          <div className={`text-sm font-semibold text-${tone}`}>
            {isSkipped ? 'Skipped (already imported)' : 'PMV imported successfully'}
          </div>
          {result.message && (
            <div className="text-xs text-surface-300 mt-1">{result.message}</div>
          )}
          <div className="grid grid-cols-2 md:grid-cols-3 gap-2 mt-3">
            {result.scene_id && <Stat label="Scene ID" value={result.scene_id} mono />}
            {result.file_id ? <Stat label="File ID" value={String(result.file_id)} mono /> : null}
            {result.funscript_generated && <Stat label="Funscript" value="generated" />}
            {!result.funscript_generated && result.funscript_downloaded && <Stat label="Funscript" value="downloaded" />}
          </div>
          {result.downloaded_path && (
            <div className="mt-3 text-[11px] text-surface-500 font-mono break-all">
              → {result.downloaded_path}
            </div>
          )}
          {result.scene_url && (
            <a
              href={result.scene_url}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 mt-3 text-xs text-cyber-blue hover:text-cyber-teal transition-colors"
            >
              <ExternalLink className="w-3 h-3" />
              {result.scene_url}
            </a>
          )}
        </div>
      </div>
    </motion.div>
  )
}

function BatchResultCard({ result }: { result: PMVImportBatchResult }) {
  const stats = [
    { label: 'Imported', value: result.imported, tone: 'cyber-teal' },
    { label: 'Skipped', value: result.skipped_existing, tone: 'cyber-amber' },
    { label: 'Funscripts', value: result.funscripts_generated, tone: 'cyber-pink' },
    { label: 'Errors', value: result.errors, tone: 'cyber-red' },
  ]

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      animate={{ opacity: 1, y: 0 }}
      className="rounded-xl border border-surface-700/40 bg-surface-800/40 overflow-hidden"
    >
      <div className="px-4 py-3 border-b border-surface-700/40 flex items-center gap-2">
        <ListMusic className="w-4 h-4 text-cyber-teal" />
        <div className="text-sm font-semibold text-surface-100">Crawl results</div>
        <span className="ml-auto text-xs text-surface-500">{result.queued} queued · {result.requested} requested</span>
      </div>

      <div className="grid grid-cols-4 divide-x divide-surface-700/40">
        {stats.map((s) => (
          <div key={s.label} className="px-4 py-3 text-center">
            <div className={`text-2xl font-bold tabular-nums text-${s.tone}`}>{s.value}</div>
            <div className="text-[10px] uppercase tracking-wider text-surface-500 mt-0.5">{s.label}</div>
          </div>
        ))}
      </div>

      {result.results.length > 0 && (
        <div className="border-t border-surface-700/40 max-h-72 overflow-y-auto">
          {result.results.map((r) => <BatchRow key={r.rank} item={r} />)}
        </div>
      )}
    </motion.div>
  )
}

function BatchRow({ item }: { item: PMVImportBatchItem }) {
  const ok = !item.error
  const skipped = item.result?.skipped
  const tone = item.error ? 'cyber-red' : skipped ? 'cyber-amber' : 'cyber-teal'

  return (
    <div className="flex items-center gap-3 px-4 py-2 hover:bg-surface-700/20 transition-colors border-b border-surface-700/30 last:border-0">
      <span className="text-[10px] font-mono text-surface-500 w-6 tabular-nums">#{item.rank}</span>
      <span className={`w-1.5 h-1.5 rounded-full bg-${tone}`} />
      <a
        href={item.url}
        target="_blank"
        rel="noreferrer"
        className="flex-1 text-xs text-surface-300 hover:text-cyber-blue truncate font-mono"
      >
        {item.url}
      </a>
      <span className={`text-[11px] text-${tone} font-medium`}>
        {item.error ? 'failed' : skipped ? 'skipped' : ok ? 'imported' : 'pending'}
      </span>
      <MoreHorizontal className="w-3 h-3 text-surface-600" />
    </div>
  )
}

function Stat({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <div className="text-[10px] uppercase tracking-wider text-surface-500">{label}</div>
      <div className={`text-xs text-surface-100 mt-0.5 truncate ${mono ? 'font-mono' : ''}`}>{value}</div>
    </div>
  )
}
