import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import {
  HardDrive, Film, FileText, Search, ChevronDown, ChevronUp,
  Folder, Monitor, Clock,
} from 'lucide-react'
import { getFiles, type SceneFile } from '../api/client'

type StorageFile = SceneFile
type SortField = 'filename' | 'size' | 'video_width' | 'created_time'
type SortDir = 'asc' | 'desc'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

export default function Files() {
  const [files, setFiles] = useState<StorageFile[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [sortField, setSortField] = useState<SortField>('filename')
  const [sortDir, setSortDir] = useState<SortDir>('asc')
  const [typeFilter, setTypeFilter] = useState<string>('all')

  useEffect(() => {
    setLoading(true)
    getFiles()
      .then((data) => setFiles(data || []))
      .catch(() => setFiles([]))
      .finally(() => setLoading(false))
  }, [])

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir(field === 'filename' ? 'asc' : 'desc')
    }
  }

  const SortIcon = ({ field }: { field: SortField }) => {
    if (sortField !== field) return null
    return sortDir === 'asc'
      ? <ChevronUp className="w-3 h-3 text-accent" />
      : <ChevronDown className="w-3 h-3 text-accent" />
  }

  const filtered = files
    .filter((f) => {
      if (typeFilter !== 'all' && f.type !== typeFilter) return false
      if (search && !f.filename.toLowerCase().includes(search.toLowerCase()) &&
          !f.path.toLowerCase().includes(search.toLowerCase())) return false
      return true
    })
    .sort((a, b) => {
      const dir = sortDir === 'asc' ? 1 : -1
      switch (sortField) {
        case 'filename': return a.filename.localeCompare(b.filename) * dir
        case 'size': return (a.size - b.size) * dir
        case 'video_width': return (a.video_width - b.video_width) * dir
        case 'created_time': return a.created_time.localeCompare(b.created_time) * dir
        default: return 0
      }
    })

  const totalSize = files.reduce((acc, f) => acc + f.size, 0)
  const videoCount = files.filter((f) => f.type === 'video').length
  const scriptCount = files.filter((f) => f.type === 'script').length

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold text-surface-100">Files</h1>
          <span className="text-sm text-surface-500">{files.length.toLocaleString()} files</span>
        </div>
      </div>

      {/* Stats */}
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
            <div className="text-lg font-semibold text-surface-100">
              {new Set(files.map(f => f.path.split('/').slice(0, -1).join('/'))).size}
            </div>
            <div className="text-xs text-surface-500">Directories</div>
          </div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="relative flex-1 max-w-sm">
          <Search className="w-4 h-4 text-surface-500 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search files..."
            className="input-dark pl-10 w-full text-sm"
          />
        </div>
        <div className="flex items-center border border-surface-700 rounded-lg overflow-hidden">
          {['all', 'video', 'script'].map((t) => (
            <button
              key={t}
              onClick={() => setTypeFilter(t)}
              className={`px-3 py-1.5 text-xs font-medium capitalize transition-colors
                ${typeFilter === t ? 'bg-accent text-white' : 'text-surface-400 hover:text-surface-200 hover:bg-surface-700/50'}`}
            >
              {t}
            </button>
          ))}
        </div>
        <span className="text-xs text-surface-500">{filtered.length} shown</span>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex items-center justify-center py-32">
          <div className="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin" />
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
                      Filename <SortIcon field="filename" />
                    </span>
                  </th>
                  <th className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-20">
                    Type
                  </th>
                  <th
                    onClick={() => handleSort('video_width')}
                    className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-28"
                  >
                    <span className="flex items-center gap-1">
                      Resolution <SortIcon field="video_width" />
                    </span>
                  </th>
                  <th
                    onClick={() => handleSort('size')}
                    className="text-right px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium cursor-pointer hover:text-surface-300 select-none w-28"
                  >
                    <span className="flex items-center gap-1 justify-end">
                      Size <SortIcon field="size" />
                    </span>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-700/30">
                {filtered.slice(0, 200).map((file) => (
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
                      <span className={`badge text-[11px] ${file.type === 'video' ? 'bg-cyber-blue/15 text-cyber-blue' : 'bg-cyber-teal/15 text-cyber-teal'}`}>
                        {file.type}
                      </span>
                    </td>
                    <td className="px-4 py-2.5">
                      {file.video_width > 0 && (
                        <span className="flex items-center gap-1.5 text-sm text-surface-300">
                          <Monitor className="w-3 h-3 text-surface-500" />
                          {file.video_width}x{file.video_height}
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-sm text-surface-300">{formatBytes(file.size)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {filtered.length > 200 && (
            <div className="px-4 py-3 border-t border-surface-700/50 text-center text-xs text-surface-500">
              Showing 200 of {filtered.length} files
            </div>
          )}
        </motion.div>
      )}
    </div>
  )
}
