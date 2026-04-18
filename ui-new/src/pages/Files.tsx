import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from 'react'
import { motion } from 'framer-motion'
import {
  ArrowUpDown,
  CalendarRange,
  ChevronDown,
  ChevronUp,
  Clock,
  Film,
  FileText,
  Folder,
  HardDrive,
  Monitor,
  Search,
  SlidersHorizontal,
  Wand2,
  SquarePen,
  Trash2,
  X,
} from 'lucide-react'
import { format, parseISO, subDays } from 'date-fns'
import {
  deleteFile,
  getFiles,
  matchFile,
  unmatchFile,
  type Scene,
  type SceneFile,
} from '../api/client'
import { useAppStore } from '../store'
import FileMatchModal from '../components/FileMatchModal'
import FileCreateSceneModal from '../components/FileCreateSceneModal'

type StorageFile = SceneFile
type FileState = 'all' | 'matched' | 'unmatched'
type SortField =
  | 'filename'
  | 'created_time'
  | 'size'
  | 'video_width'
  | 'video_height'
  | 'video_bitrate'
  | 'duration'
  | 'video_avgfps_val'
type SortDir = 'asc' | 'desc'

const STATE_OPTIONS: { label: string; value: FileState }[] = [
  { label: 'All', value: 'all' },
  { label: 'Matched', value: 'matched' },
  { label: 'Unmatched', value: 'unmatched' },
]

const RESOLUTION_OPTIONS = [
  { label: 'Below 4K', value: 'below4k' },
  { label: '4K', value: '4k' },
  { label: '5K', value: '5k' },
  { label: '6K', value: '6k' },
  { label: 'Above 6K', value: 'above6k' },
]

const BITRATE_OPTIONS = [
  { label: 'Low', hint: '< 15 Mbps', value: 'low' },
  { label: 'Medium', hint: '15-24 Mbps', value: 'medium' },
  { label: 'High', hint: '25-35 Mbps', value: 'high' },
  { label: 'Ultra', hint: '> 35 Mbps', value: 'ultra' },
]

const FRAMERATE_OPTIONS = [
  { label: '30 fps', value: '30fps' },
  { label: '60 fps', value: '60fps' },
  { label: 'Other', value: 'other' },
]

const SORT_OPTIONS: { label: string; value: SortField }[] = [
  { label: 'Filename', value: 'filename' },
  { label: 'Created time', value: 'created_time' },
  { label: 'Size', value: 'size' },
  { label: 'Width', value: 'video_width' },
  { label: 'Height', value: 'video_height' },
  { label: 'Bitrate', value: 'video_bitrate' },
  { label: 'Duration', value: 'duration' },
  { label: 'FPS', value: 'video_avgfps_val' },
]

const DEFAULT_SORT_DIR: Record<SortField, SortDir> = {
  filename: 'asc',
  created_time: 'desc',
  size: 'desc',
  video_width: 'desc',
  video_height: 'desc',
  video_bitrate: 'desc',
  duration: 'desc',
  video_avgfps_val: 'desc',
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

function formatBitsPerSecond(bitsPerSecond: number): string {
  if (bitsPerSecond === 0) return '0 b/s'
  const k = 1000
  const sizes = ['b/s', 'Kb/s', 'Mb/s', 'Gb/s', 'Tb/s']
  const i = Math.floor(Math.log(bitsPerSecond) / Math.log(k))
  return `${(bitsPerSecond / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

function formatDuration(seconds: number): string {
  if (!seconds) return '-'
  const totalSeconds = Math.max(0, Math.floor(seconds))
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60

  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, '0')}:${String(secs).padStart(2, '0')}`
  }

  return `${minutes}:${String(secs).padStart(2, '0')}`
}

function formatDateInput(date: Date): string {
  return format(date, 'yyyy-MM-dd')
}

function rangeToRfc3339(value: string, endOfDay: boolean): string {
  const [year, month, day] = value.split('-').map((part) => Number(part))
  const date = new Date(
    year,
    month - 1,
    day,
    endOfDay ? 23 : 0,
    endOfDay ? 59 : 0,
    endOfDay ? 59 : 0,
    endOfDay ? 999 : 0,
  )
  return date.toISOString()
}

export default function Files() {
  const openSceneDetail = useAppStore((s) => s.openSceneDetail)
  const loadRequestId = useRef(0)
  const [files, setFiles] = useState<StorageFile[]>([])
  const [loading, setLoading] = useState(true)
  const [filename, setFilename] = useState('')
  const [appliedFilename, setAppliedFilename] = useState('')
  const [fileState, setFileState] = useState<FileState>('unmatched')
  const [createdStart, setCreatedStart] = useState('')
  const [createdEnd, setCreatedEnd] = useState('')
  const [sortField, setSortField] = useState<SortField>('created_time')
  const [sortDir, setSortDir] = useState<SortDir>('desc')
  const [typeFilter, setTypeFilter] = useState<string>('all')
  const [resolutions, setResolutions] = useState<string[]>([])
  const [bitrates, setBitrates] = useState<string[]>([])
  const [framerates, setFramerates] = useState<string[]>([])
  const [matchFileTarget, setMatchFileTarget] = useState<StorageFile | null>(null)
  const [createSceneFile, setCreateSceneFile] = useState<StorageFile | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<StorageFile | null>(null)

  const loadFiles = useCallback(() => {
    const requestId = ++loadRequestId.current
    setLoading(true)

    const params: Record<string, any> = {
      sort: `${sortField}_${sortDir}`,
    }

    if (fileState !== 'all') params.state = fileState
    if (appliedFilename) params.filename = appliedFilename
    if (resolutions.length > 0) params.resolutions = resolutions
    if (bitrates.length > 0) params.bitrates = bitrates
    if (framerates.length > 0) params.framerates = framerates
    if (createdStart && createdEnd) {
      params.createdDate = [
        rangeToRfc3339(createdStart, false),
        rangeToRfc3339(createdEnd, true),
      ]
    }

    getFiles(params)
      .then((data) => {
        if (loadRequestId.current !== requestId) return
        setFiles(data || [])
      })
      .catch(() => {
        if (loadRequestId.current !== requestId) return
        setFiles([])
      })
      .finally(() => {
        if (loadRequestId.current === requestId) setLoading(false)
      })
  }, [
    fileState,
    sortField,
    sortDir,
    resolutions,
    bitrates,
    framerates,
    createdStart,
    createdEnd,
    appliedFilename,
  ])

  useEffect(() => {
    const trimmed = filename.trim()
    if (trimmed.length > 0 && trimmed.length < 3) {
      return
    }

    const timeout = window.setTimeout(() => {
      setAppliedFilename(trimmed)
    }, 250)

    return () => window.clearTimeout(timeout)
  }, [filename])

  useEffect(() => {
    loadFiles()
  }, [loadFiles])

  useEffect(() => () => {
    loadRequestId.current += 1
  }, [])

  const visibleFiles = useMemo(
    () => files.filter((file) => typeFilter === 'all' || file.type === typeFilter),
    [files, typeFilter],
  )

  const hasActiveFilters =
    fileState !== 'unmatched' ||
    appliedFilename.length > 0 ||
    createdStart.length > 0 ||
    createdEnd.length > 0 ||
    resolutions.length > 0 ||
    bitrates.length > 0 ||
    framerates.length > 0 ||
    typeFilter !== 'all'

  const getRelativeVisibleFile = (current: StorageFile, delta: number) => {
    if (visibleFiles.length <= 1) return null
    const index = visibleFiles.findIndex((file) => file.id === current.id)
    if (index < 0) return null
    const nextIndex = (index + delta + visibleFiles.length) % visibleFiles.length
    return visibleFiles[nextIndex] || null
  }

  const refreshFiles = () => {
    loadFiles()
  }

  const openMatchForFile = (file: StorageFile) => {
    setMatchFileTarget(file)
  }

  const handleMatchAssign = async (sceneId: string) => {
    if (!matchFileTarget) return
    try {
      await matchFile(matchFileTarget.id, sceneId)
      refreshFiles()
      const next = getRelativeVisibleFile(matchFileTarget, 1)
      setMatchFileTarget(next)
    } catch {}
  }

  const handleMatchPrevious = () => {
    if (!matchFileTarget) return
    const prev = getRelativeVisibleFile(matchFileTarget, -1)
    if (prev) setMatchFileTarget(prev)
  }

  const handleMatchNext = () => {
    if (!matchFileTarget) return
    const next = getRelativeVisibleFile(matchFileTarget, 1)
    if (next) setMatchFileTarget(next)
  }

  const handleUnmatch = async (file: StorageFile) => {
    try {
      await unmatchFile(file.id)
      refreshFiles()
    } catch {}
  }

  const handleDelete = async (file: StorageFile) => {
    try {
      await deleteFile(file.id)
      refreshFiles()
      return true
    } catch {
      return false
    }
  }

  const handleCreatedScene = async (scene: Scene, openDetail: boolean) => {
    refreshFiles()
    if (openDetail) {
      openSceneDetail(scene)
    }
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir((current) => (current === 'asc' ? 'desc' : 'asc'))
      return
    }

    setSortField(field)
    setSortDir(DEFAULT_SORT_DIR[field])
  }

  const toggleSelection = (
    value: string,
    setSelected: Dispatch<SetStateAction<string[]>>,
  ) => {
    setSelected((current) => (
      current.includes(value)
        ? current.filter((item) => item !== value)
        : [...current, value]
    ))
  }

  const clearFilters = () => {
    setFilename('')
    setAppliedFilename('')
    setFileState('unmatched')
    setCreatedStart('')
    setCreatedEnd('')
    setResolutions([])
    setBitrates([])
    setFramerates([])
    setTypeFilter('all')
    setSortField('created_time')
    setSortDir('desc')
  }

  const totalSize = visibleFiles.reduce((acc, file) => acc + file.size, 0)
  const videoCount = visibleFiles.filter((file) => file.type === 'video').length
  const scriptCount = visibleFiles.filter((file) => file.type === 'script').length
  const directoryCount = new Set(
    visibleFiles.map((file) => file.path.split('/').slice(0, -1).join('/')),
  ).size

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold text-surface-100">Files</h1>
          <span className="text-sm text-surface-500">
            {visibleFiles.length.toLocaleString()} visible
          </span>
        </div>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="glass rounded-xl p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-accent/15 flex items-center justify-center">
            <HardDrive className="w-5 h-5 text-accent" />
          </div>
          <div>
            <div className="text-lg font-semibold text-surface-100">{formatBytes(totalSize)}</div>
            <div className="text-xs text-surface-500">Total size</div>
          </div>
        </div>
        <div className="glass rounded-xl p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-cyber-blue/15 flex items-center justify-center">
            <Film className="w-5 h-5 text-cyber-blue" />
          </div>
          <div>
            <div className="text-lg font-semibold text-surface-100">{videoCount}</div>
            <div className="text-xs text-surface-500">Videos</div>
          </div>
        </div>
        <div className="glass rounded-xl p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-cyber-teal/15 flex items-center justify-center">
            <FileText className="w-5 h-5 text-cyber-teal" />
          </div>
          <div>
            <div className="text-lg font-semibold text-surface-100">{scriptCount}</div>
            <div className="text-xs text-surface-500">Scripts</div>
          </div>
        </div>
        <div className="glass rounded-xl p-4 flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-cyber-amber/15 flex items-center justify-center">
            <Folder className="w-5 h-5 text-cyber-amber" />
          </div>
          <div>
            <div className="text-lg font-semibold text-surface-100">{directoryCount}</div>
            <div className="text-xs text-surface-500">Directories</div>
          </div>
        </div>
      </div>

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        className="glass rounded-xl p-4 space-y-4"
      >
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <div className="flex items-center gap-2 text-sm text-surface-200 font-medium">
            <SlidersHorizontal className="w-4 h-4 text-accent" />
            Filters
            {hasActiveFilters && <span className="w-1.5 h-1.5 rounded-full bg-accent" />}
          </div>
          {hasActiveFilters && (
            <button
              onClick={clearFilters}
              className="btn-ghost text-xs text-accent hover:text-accent-hover flex items-center gap-1"
            >
              <X className="w-3 h-3" />
              Clear all
            </button>
          )}
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                State
              </label>
              <div className="flex flex-wrap gap-2">
                {STATE_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => setFileState(option.value)}
                    className={`badge transition-colors ${
                      fileState === option.value
                        ? 'bg-accent/20 text-accent'
                        : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            <label className="block">
              <div className="text-xs text-surface-500 uppercase tracking-wider font-medium mb-1.5">
                Filename
              </div>
              <div className="relative">
                <Search className="w-4 h-4 text-surface-500 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  value={filename}
                  onChange={(e) => setFilename(e.target.value)}
                  placeholder="Search filenames"
                  className="input-dark pl-10 pr-10 w-full text-sm"
                />
                {filename.length > 0 && (
                  <button
                    type="button"
                    onClick={() => setFilename('')}
                    className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-surface-500 hover:text-surface-200"
                    aria-label="Clear filename search"
                  >
                    <X className="w-4 h-4" />
                  </button>
                )}
              </div>
              <div className="text-[11px] text-surface-500 mt-1">
                Searches apply after 3 characters, matching the legacy files page.
              </div>
            </label>

            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                Type
              </label>
              <div className="flex flex-wrap gap-2">
                {['all', 'video', 'script'].map((type) => (
                  <button
                    key={type}
                    onClick={() => setTypeFilter(type)}
                    className={`badge capitalize transition-colors ${
                      typeFilter === type
                        ? 'bg-cyber-blue/20 text-cyber-blue'
                        : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'
                    }`}
                  >
                    {type}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                Created between
              </label>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                <label className="block">
                  <span className="sr-only">Created start</span>
                  <input
                    type="date"
                    value={createdStart}
                    onChange={(e) => setCreatedStart(e.target.value)}
                    className="input-dark w-full"
                  />
                </label>
                <label className="block">
                  <span className="sr-only">Created end</span>
                  <input
                    type="date"
                    value={createdEnd}
                    onChange={(e) => setCreatedEnd(e.target.value)}
                    className="input-dark w-full"
                  />
                </label>
              </div>
              <div className="flex flex-wrap gap-2">
                {[7, 14, 30].map((days) => (
                  <button
                    key={days}
                    type="button"
                    onClick={() => {
                      setCreatedStart(formatDateInput(subDays(new Date(), days)))
                      setCreatedEnd(formatDateInput(new Date()))
                    }}
                    className="btn-ghost text-xs flex items-center gap-1"
                  >
                    <CalendarRange className="w-3 h-3" />
                    Last {days} days
                  </button>
                ))}
                {(createdStart || createdEnd) && (
                  <button
                    type="button"
                    onClick={() => {
                      setCreatedStart('')
                      setCreatedEnd('')
                    }}
                    className="btn-ghost text-xs flex items-center gap-1"
                  >
                    <X className="w-3 h-3" />
                    Clear range
                  </button>
                )}
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                Resolution
              </label>
              <div className="flex flex-wrap gap-2">
                {RESOLUTION_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => toggleSelection(option.value, setResolutions)}
                    className={`badge transition-colors ${
                      resolutions.includes(option.value)
                        ? 'bg-cyber-amber/20 text-cyber-amber'
                        : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                Bitrate
              </label>
              <div className="flex flex-wrap gap-2">
                {BITRATE_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => toggleSelection(option.value, setBitrates)}
                    className={`badge transition-colors ${
                      bitrates.includes(option.value)
                        ? 'bg-cyber-teal/20 text-cyber-teal'
                        : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'
                    }`}
                  >
                    <span>{option.label}</span>
                    <span className="opacity-70">{option.hint}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                Framerate
              </label>
              <div className="flex flex-wrap gap-2">
                {FRAMERATE_OPTIONS.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => toggleSelection(option.value, setFramerates)}
                    className={`badge transition-colors ${
                      framerates.includes(option.value)
                        ? 'bg-cyber-pink/20 text-cyber-pink'
                        : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between gap-3 flex-wrap pt-1 border-t border-surface-700/40">
          <div className="flex items-center gap-2 flex-wrap">
            <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
              Sort by
            </label>
            <select
              value={sortField}
              onChange={(e) => handleSort(e.target.value as SortField)}
              className="input-dark w-48"
            >
              {SORT_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={() => setSortDir((current) => (current === 'asc' ? 'desc' : 'asc'))}
              className="btn-ghost flex items-center gap-1.5"
            >
              <ArrowUpDown className="w-4 h-4" />
              {sortDir === 'asc' ? 'Ascending' : 'Descending'}
            </button>
          </div>
          <span className="text-xs text-surface-500">
            {loading ? 'Loading files…' : `${visibleFiles.length} shown`}
          </span>
        </div>
      </motion.div>

      {loading ? (
        <div className="flex items-center justify-center py-32">
          <div className="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : visibleFiles.length === 0 ? (
        <div className="glass rounded-xl p-12 text-center">
          <div className="mx-auto w-14 h-14 rounded-full bg-surface-700/60 flex items-center justify-center mb-4">
            <FileText className="w-7 h-7 text-surface-400" />
          </div>
          <h2 className="text-lg font-semibold text-surface-100">No files matching your selection</h2>
          <p className="text-sm text-surface-500 mt-1">
            Try clearing a filter or switching the file state back to all.
          </p>
        </div>
      ) : (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="glass rounded-xl overflow-hidden"
        >
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-surface-700/50">
                  <th
                    onClick={() => handleSort('filename')}
                    className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none"
                  >
                    <span className="flex items-center gap-1">
                      Filename
                      {sortField === 'filename' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-20">
                    Type
                  </th>
                  <th
                    onClick={() => handleSort('created_time')}
                    className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-40"
                  >
                    <span className="flex items-center gap-1">
                      Created
                      {sortField === 'created_time' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th
                    onClick={() => handleSort('video_width')}
                    className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-28"
                  >
                    <span className="flex items-center gap-1">
                      Resolution
                      {sortField === 'video_width' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th
                    onClick={() => handleSort('video_bitrate')}
                    className="text-right px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-28"
                  >
                    <span className="flex items-center gap-1 justify-end">
                      Bitrate
                      {sortField === 'video_bitrate' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th
                    onClick={() => handleSort('duration')}
                    className="text-right px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-24"
                  >
                    <span className="flex items-center gap-1 justify-end">
                      Duration
                      {sortField === 'duration' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th
                    onClick={() => handleSort('video_avgfps_val')}
                    className="text-right px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-20"
                  >
                    <span className="flex items-center gap-1 justify-end">
                      FPS
                      {sortField === 'video_avgfps_val' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th
                    onClick={() => handleSort('size')}
                    className="text-right px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-28"
                  >
                    <span className="flex items-center gap-1 justify-end">
                      Size
                      {sortField === 'size' && (
                        sortDir === 'asc'
                          ? <ChevronUp className="w-3 h-3 text-accent" />
                          : <ChevronDown className="w-3 h-3 text-accent" />
                      )}
                    </span>
                  </th>
                  <th className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-28">
                    Match
                  </th>
                  <th className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-28">
                    Create
                  </th>
                  <th className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-28">
                    Delete
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-700/30">
                {visibleFiles.slice(0, 200).map((file) => (
                  <tr key={file.id} className="hover:bg-surface-700/20 transition-colors group">
                    <td className="px-4 py-2.5">
                      <div className="flex items-center gap-2 min-w-0">
                        {file.type === 'video'
                          ? <Film className="w-4 h-4 text-cyber-blue flex-shrink-0" />
                          : <FileText className="w-4 h-4 text-cyber-teal flex-shrink-0" />
                        }
                        <span className="text-sm text-surface-200 truncate font-mono">{file.filename}</span>
                      </div>
                      <div className="text-[11px] text-surface-600 truncate ml-6 font-mono">{file.path}</div>
                    </td>
                    <td className="px-4 py-2.5">
                      <span
                        className={`badge text-[11px] ${
                          file.type === 'video'
                            ? 'bg-cyber-blue/15 text-cyber-blue'
                            : 'bg-cyber-teal/15 text-cyber-teal'
                        }`}
                      >
                        {file.type}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      <span className="flex items-center gap-1.5 text-sm text-surface-300">
                        <Clock className="w-3 h-3 text-surface-500" />
                        {format(parseISO(file.created_time), 'yyyy-MM-dd HH:mm:ss')}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      {file.video_width > 0 ? (
                        <span className="flex items-center gap-1.5 text-sm text-surface-300">
                          <Monitor className="w-3 h-3 text-surface-500" />
                          {file.video_width}x{file.video_height}
                        </span>
                      ) : (
                        <span className="text-sm text-surface-600">-</span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-sm text-surface-300">
                        {file.video_bitrate > 0 ? formatBitsPerSecond(file.video_bitrate) : '-'}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-sm text-surface-300">{formatDuration(file.duration)}</span>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-sm text-surface-300">
                        {file.video_avg_frame_rate_val > 0
                          ? file.video_avg_frame_rate_val.toFixed(file.video_avg_frame_rate_val % 1 === 0 ? 0 : 2)
                          : '-'}
                      </span>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-sm text-surface-300">{formatBytes(file.size)}</span>
                    </td>
                    <td className="px-4 py-2.5">
                      {file.scene_id > 0 ? (
                        <button
                          onClick={() => void handleUnmatch(file)}
                          className="btn-ghost text-xs flex items-center gap-1.5 text-cyber-blue"
                        >
                          <Wand2 className="w-3.5 h-3.5" />
                          Unmatch
                        </button>
                      ) : (
                        <button
                          onClick={() => openMatchForFile(file)}
                          className="btn-ghost text-xs flex items-center gap-1.5 text-cyber-pink"
                        >
                          <Wand2 className="w-3.5 h-3.5" />
                          Match
                        </button>
                      )}
                    </td>
                    <td className="px-4 py-2.5">
                      <button
                        onClick={() => setCreateSceneFile(file)}
                        className="btn-ghost text-xs flex items-center gap-1.5"
                      >
                        <SquarePen className="w-3.5 h-3.5" />
                        Create
                      </button>
                    </td>
                    <td className="px-4 py-2.5">
                      <button
                        onClick={() => setDeleteTarget(file)}
                        className="btn-ghost text-xs flex items-center gap-1.5 text-cyber-red"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {visibleFiles.length > 200 && (
            <div className="px-4 py-3 border-t border-surface-700/50 text-center text-xs text-surface-500">
              Showing 200 of {visibleFiles.length} files
            </div>
          )}
        </motion.div>
      )}

      <FileMatchModal
        file={matchFileTarget}
        onClose={() => setMatchFileTarget(null)}
        onPrevious={visibleFiles.length > 1 ? handleMatchPrevious : undefined}
        onNext={visibleFiles.length > 1 ? handleMatchNext : undefined}
        onAssign={handleMatchAssign}
      />

      <FileCreateSceneModal
        file={createSceneFile}
        onClose={() => setCreateSceneFile(null)}
        onCreated={handleCreatedScene}
      />

      {deleteTarget && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm"
            onClick={() => setDeleteTarget(null)}
          />
          <motion.div
            initial={{ opacity: 0, y: 20, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            className="fixed inset-x-4 top-16 sm:inset-x-auto sm:left-1/2 sm:-translate-x-1/2 z-50 w-full sm:max-w-lg glass-strong rounded-2xl p-5"
          >
            <div className="flex items-start gap-3">
              <div className="w-10 h-10 rounded-full bg-cyber-red/15 flex items-center justify-center flex-shrink-0">
                <Trash2 className="w-5 h-5 text-cyber-red" />
              </div>
              <div className="min-w-0 flex-1">
                <h2 className="text-base font-semibold text-surface-100">Delete file from disk?</h2>
                <p className="text-sm text-surface-400 mt-1">
                  {deleteTarget.filename}
                </p>
                <p className="text-xs text-surface-500 font-mono break-all mt-1">
                  {deleteTarget.path}
                </p>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 mt-5">
              <button
                onClick={() => setDeleteTarget(null)}
                className="btn-ghost text-sm"
              >
                Cancel
              </button>
              <button
                onClick={async () => {
                  if (deleteTarget && await handleDelete(deleteTarget)) {
                    setDeleteTarget(null)
                  }
                }}
                className="px-4 py-2 rounded-lg bg-cyber-red text-white text-sm font-medium hover:opacity-90 transition-opacity"
              >
                Delete permanently
              </button>
            </div>
          </motion.div>
        </>
      )}
    </div>
  )
}
