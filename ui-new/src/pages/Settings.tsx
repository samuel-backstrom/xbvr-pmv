import { useEffect, useState, useCallback } from 'react'
import { motion } from 'framer-motion'
import {
  Settings as SettingsIcon, FolderOpen, RefreshCw, Globe,
  Database, Palette, Monitor, Shield, Trash2, Check,
  Cloud, FolderPlus, X, AlertTriangle, HardDrive,
} from 'lucide-react'
import {
  getStorage, addStorage, removeStorage, saveStorageOptions,
  rescanAll, rescanVolume,
  type Volume, type StorageResponse,
} from '../api/client'
import { formatDistanceToNow, parseISO } from 'date-fns'
import { useAppStore } from '../store'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

function StorageTab() {
  const storageRevision = useAppStore((s) => s.storageRevision)
  const lockRescan = useAppStore((s) => s.lockRescan)
  const lastRescanMessage = useAppStore((s) => s.lastRescanMessage)
  const [volumes, setVolumes] = useState<Volume[]>([])
  const [loading, setLoading] = useState(true)
  const [newPath, setNewPath] = useState('')
  const [addingFolder, setAddingFolder] = useState(false)
  const [cloudService, setCloudService] = useState('putio')
  const [cloudToken, setCloudToken] = useState('')
  const [addingCloud, setAddingCloud] = useState(false)
  const [matchOhash, setMatchOhash] = useState(false)
  const [videoExt, setVideoExt] = useState<string[]>([])
  const [defaultVideoExt, setDefaultVideoExt] = useState<string[]>([])
  const [forbiddenVideoExt, setForbiddenVideoExt] = useState<string[]>([])
  const [newExt, setNewExt] = useState('')
  const [extError, setExtError] = useState('')
  const [confirmRemove, setConfirmRemove] = useState<number | null>(null)
  const [rescanning, setRescanning] = useState<number | null>(null)
  const [rescanningAll, setRescanningAll] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const loadStorage = useCallback(async () => {
    try {
      const data = await getStorage()
      setVolumes(data.volumes || [])
      setMatchOhash(data.match_ohash)
      setVideoExt(data.video_ext || [])
      setDefaultVideoExt(data.default_video_ext || [])
      setForbiddenVideoExt(data.forbidden_video_ext || [])
    } catch {}
    setLoading(false)
  }, [])

  useEffect(() => {
    loadStorage()
  }, [loadStorage, storageRevision])

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 4000)
  }

  const handleAddFolder = async () => {
    if (!newPath.trim()) return
    setAddingFolder(true)
    try {
      await addStorage({ path: newPath.trim(), type: 'local' })
      setNewPath('')
      showMessage('success', 'Folder added successfully')
      await loadStorage()
    } catch (e: any) {
      showMessage('error', 'Failed to add folder. Make sure the path exists and is accessible.')
    }
    setAddingFolder(false)
  }

  const handleAddCloud = async () => {
    if (!cloudToken.trim()) return
    setAddingCloud(true)
    try {
      await addStorage({ token: cloudToken.trim(), type: cloudService })
      setCloudToken('')
      showMessage('success', 'Cloud storage added successfully')
      await loadStorage()
    } catch {
      showMessage('error', 'Failed to add cloud storage. Check your token.')
    }
    setAddingCloud(false)
  }

  const handleRemove = async (id: number) => {
    try {
      await removeStorage(id)
      setConfirmRemove(null)
      showMessage('success', 'Folder removed')
      await loadStorage()
    } catch {
      showMessage('error', 'Failed to remove folder')
    }
  }

  const handleRescan = async (id: number) => {
    setRescanning(id)
    try {
      await rescanVolume(id)
      showMessage('success', 'Rescan started')
    } catch {
      showMessage('error', 'Failed to start rescan')
    }
    setTimeout(() => setRescanning(null), 2000)
  }

  const handleRescanAll = async () => {
    setRescanningAll(true)
    try {
      await saveStorageOptions({ match_ohash: matchOhash, video_ext: videoExt })
      await rescanAll()
      showMessage('success', 'Rescan started for all folders')
    } catch {
      showMessage('error', 'Failed to start rescan')
    }
    setTimeout(() => setRescanningAll(false), 2000)
  }

  const handleSaveOptions = async () => {
    try {
      await saveStorageOptions({ match_ohash: matchOhash, video_ext: videoExt })
      showMessage('success', 'Options saved')
    } catch {
      showMessage('error', 'Failed to save options')
    }
  }

  const handleAddExt = () => {
    setExtError('')
    let ext = newExt.trim().toLowerCase()
    if (!ext || ext === '.') { setNewExt(''); return }
    if (/\s/.test(ext)) { setExtError('Extensions cannot contain whitespace'); return }
    if (!ext.startsWith('.')) ext = '.' + ext
    if (!/^\.[a-z0-9]+$/.test(ext)) { setExtError('Only letters and numbers after the dot'); return }
    if (forbiddenVideoExt.includes(ext)) { setExtError(`${ext} is a reserved extension`); return }
    if (videoExt.includes(ext)) { setExtError(`${ext} already added`); return }
    const next = [...videoExt, ext]
    setVideoExt(next)
    setNewExt('')
    saveStorageOptions({ match_ohash: matchOhash, video_ext: next })
  }

  const handleRemoveExt = (ext: string) => {
    const next = videoExt.filter(e => e !== ext)
    setVideoExt(next)
    saveStorageOptions({ match_ohash: matchOhash, video_ext: next })
  }

  const handleResetExt = () => {
    setVideoExt([...defaultVideoExt])
    saveStorageOptions({ match_ohash: matchOhash, video_ext: [...defaultVideoExt] })
  }

  const totalFiles = volumes.reduce((a, v) => a + v.file_count, 0)
  const totalUnmatched = volumes.reduce((a, v) => a + v.unmatched_count, 0)
  const totalSize = volumes.reduce((a, v) => a + v.total_size, 0)

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16">
        <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Message toast */}
      {message && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          className={`flex items-center gap-2 px-4 py-3 rounded-xl text-sm ${
            message.type === 'success'
              ? 'bg-cyber-teal/15 text-cyber-teal border border-cyber-teal/20'
              : 'bg-cyber-red/15 text-cyber-red border border-cyber-red/20'
          }`}
        >
          {message.type === 'success' ? <Check className="w-4 h-4" /> : <AlertTriangle className="w-4 h-4" />}
          {message.text}
        </motion.div>
      )}

      {lastRescanMessage && (
        <div className={`flex items-center gap-2 px-4 py-3 rounded-xl text-sm border ${
          lockRescan
            ? 'bg-cyber-amber/10 text-cyber-amber border-cyber-amber/20'
            : 'bg-surface-800/50 text-surface-300 border-surface-700/30'
        }`}>
          <RefreshCw className={`w-4 h-4 ${lockRescan ? 'animate-spin' : ''}`} />
          <span>{lastRescanMessage.message}</span>
        </div>
      )}

      {/* Header with rescan all */}
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-surface-100">Storage</h3>
        <button
          onClick={handleRescanAll}
          disabled={rescanningAll || lockRescan}
          className="btn-primary text-sm flex items-center gap-2"
        >
          <RefreshCw className={`w-4 h-4 ${(rescanningAll || lockRescan) ? 'animate-spin' : ''}`} />
          {lockRescan ? 'Rescan running...' : rescanningAll ? 'Rescanning...' : 'Rescan all folders'}
        </button>
      </div>

      {/* Volumes table */}
      {volumes.length > 0 ? (
        <div className="rounded-xl overflow-hidden border border-surface-700/30">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="bg-surface-800/50">
                  <th className="text-left px-4 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium">Path</th>
                  <th className="text-center px-3 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-16">Type</th>
                  <th className="text-center px-3 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-16">Avail</th>
                  <th className="text-right px-3 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-20">Files</th>
                  <th className="text-right px-3 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-28">Unmatched</th>
                  <th className="text-right px-3 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-24">Size</th>
                  <th className="text-right px-3 py-3 text-xs text-surface-500 uppercase tracking-wider font-medium w-28">Last scan</th>
                  <th className="px-3 py-3 w-24"></th>
                </tr>
              </thead>
              <tbody className="divide-y divide-surface-700/30">
                {volumes.map((vol) => (
                  <tr key={vol.id} className="hover:bg-surface-700/20 transition-colors">
                    <td className="px-4 py-3">
                      <span className="text-sm text-surface-200 font-mono">{vol.path}</span>
                    </td>
                    <td className="px-3 py-3 text-center">
                      {vol.type === 'local'
                        ? <FolderOpen className="w-4 h-4 text-cyber-amber mx-auto" />
                        : <Cloud className="w-4 h-4 text-cyber-blue mx-auto" />
                      }
                    </td>
                    <td className="px-3 py-3 text-center">
                      {vol.is_available && <Check className="w-4 h-4 text-cyber-teal mx-auto" />}
                    </td>
                    <td className="px-3 py-3 text-right text-sm text-surface-300">{vol.file_count}</td>
                    <td className="px-3 py-3 text-right text-sm text-surface-300">{vol.unmatched_count}</td>
                    <td className="px-3 py-3 text-right text-sm text-surface-300">{formatBytes(vol.total_size)}</td>
                    <td className="px-3 py-3 text-right text-xs text-surface-500">
                      {vol.last_scan && vol.last_scan !== '0001-01-01T00:00:00Z'
                        ? `${formatDistanceToNow(parseISO(vol.last_scan))} ago`
                        : 'never'
                      }
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => handleRescan(vol.id)}
                          disabled={rescanning === vol.id || lockRescan}
                          className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-400 hover:text-surface-200 transition-colors"
                          title="Rescan folder"
                        >
                          <RefreshCw className={`w-3.5 h-3.5 ${(rescanning === vol.id || lockRescan) ? 'animate-spin' : ''}`} />
                        </button>
                        {confirmRemove === vol.id ? (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleRemove(vol.id)}
                              className="p-1.5 rounded-lg bg-cyber-red/15 text-cyber-red hover:bg-cyber-red/25 transition-colors"
                              title="Confirm remove"
                            >
                              <Check className="w-3.5 h-3.5" />
                            </button>
                            <button
                              onClick={() => setConfirmRemove(null)}
                              className="p-1.5 rounded-lg hover:bg-surface-700/50 text-surface-400 hover:text-surface-200 transition-colors"
                              title="Cancel"
                            >
                              <X className="w-3.5 h-3.5" />
                            </button>
                          </div>
                        ) : (
                          <button
                            onClick={() => setConfirmRemove(vol.id)}
                            className="p-1.5 rounded-lg hover:bg-cyber-red/10 text-surface-400 hover:text-cyber-red transition-colors"
                            title="Remove folder"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr className="bg-surface-800/30 border-t border-surface-700/30">
                  <td className="px-4 py-2.5 text-xs text-surface-500 font-medium">
                    {volumes.length} folder{volumes.length !== 1 ? 's' : ''}
                  </td>
                  <td></td>
                  <td></td>
                  <td className="px-3 py-2.5 text-right text-xs text-surface-400 font-medium">{totalFiles}</td>
                  <td className="px-3 py-2.5 text-right text-xs text-surface-400 font-medium">{totalUnmatched}</td>
                  <td className="px-3 py-2.5 text-right text-xs text-surface-400 font-medium">{formatBytes(totalSize)}</td>
                  <td></td>
                  <td></td>
                </tr>
              </tfoot>
            </table>
          </div>
        </div>
      ) : (
        <div className="p-8 rounded-xl border border-dashed border-surface-600 text-center">
          <FolderOpen className="w-10 h-10 text-surface-500 mx-auto mb-3" />
          <p className="text-surface-300 font-medium">No folders added yet</p>
          <p className="text-sm text-surface-500 mt-1">Add folders with VR videos below</p>
        </div>
      )}

      {/* Add folder / cloud storage */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Add local folder */}
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-surface-200 flex items-center gap-2">
            <FolderPlus className="w-4 h-4 text-cyber-amber" />
            Add local folder
          </h4>
          <div>
            <label className="text-xs text-surface-500 mb-1 block">Path to folder with content</label>
            <div className="flex gap-2">
              <input
                type="text"
                value={newPath}
                onChange={(e) => setNewPath(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddFolder() }}
                placeholder="/path/to/videos"
                className="input-dark flex-1 text-sm font-mono"
              />
              <button
                onClick={handleAddFolder}
                disabled={!newPath.trim() || addingFolder}
                className="btn-primary text-sm whitespace-nowrap disabled:opacity-40 disabled:cursor-not-allowed"
              >
                {addingFolder ? 'Adding...' : 'Add folder'}
              </button>
            </div>
          </div>
        </div>

        {/* Add cloud storage */}
        <div className="space-y-3">
          <h4 className="text-sm font-semibold text-surface-200 flex items-center gap-2">
            <Cloud className="w-4 h-4 text-cyber-blue" />
            Add cloud storage
          </h4>
          <div className="flex gap-2">
            <select
              value={cloudService}
              onChange={(e) => setCloudService(e.target.value)}
              className="input-dark text-sm w-28"
            >
              <option value="putio">Put.io</option>
            </select>
            <input
              type="text"
              value={cloudToken}
              onChange={(e) => setCloudToken(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') handleAddCloud() }}
              placeholder="API token"
              className="input-dark flex-1 text-sm font-mono"
            />
            <button
              onClick={handleAddCloud}
              disabled={!cloudToken.trim() || addingCloud}
              className="btn-primary text-sm whitespace-nowrap disabled:opacity-40 disabled:cursor-not-allowed"
            >
              {addingCloud ? 'Adding...' : 'Add service'}
            </button>
          </div>
        </div>
      </div>

      {/* Options section */}
      <div className="border-t border-surface-700/30 pt-6 space-y-5">
        <h4 className="text-sm font-semibold text-surface-200">Options</h4>

        {/* Match OHash toggle */}
        <div className="flex items-center justify-between">
          <div>
            <div className="text-sm text-surface-200">Match StashDB Hashes</div>
            <div className="text-xs text-surface-500 mt-0.5">Use OsHash to match files with StashDB</div>
          </div>
          <button
            onClick={() => {
              const next = !matchOhash
              setMatchOhash(next)
              saveStorageOptions({ match_ohash: next, video_ext: videoExt })
            }}
            className={`relative w-11 h-6 rounded-full transition-colors ${
              matchOhash ? 'bg-accent' : 'bg-surface-600'
            }`}
          >
            <div className={`absolute top-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform ${
              matchOhash ? 'translate-x-[22px]' : 'translate-x-0.5'
            }`} />
          </button>
        </div>

        {/* Video extensions */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <div className="text-sm text-surface-200">Video file extensions</div>
              <div className="text-xs text-surface-500 mt-0.5">File types to scan for in your library folders</div>
            </div>
            <button
              onClick={handleResetExt}
              className="btn-ghost text-xs text-cyber-amber hover:bg-cyber-amber/10"
            >
              Reset to defaults
            </button>
          </div>

          <div className="flex flex-wrap gap-1.5">
            {videoExt.map((ext) => (
              <span
                key={ext}
                className="badge bg-surface-700/50 text-surface-300 pr-1"
              >
                {ext}
                <button
                  onClick={() => handleRemoveExt(ext)}
                  className="ml-1 p-0.5 rounded hover:bg-surface-600 text-surface-500 hover:text-surface-200 transition-colors"
                >
                  <X className="w-3 h-3" />
                </button>
              </span>
            ))}
          </div>

          <div className="flex gap-2 items-start">
            <div className="flex-1">
              <input
                type="text"
                value={newExt}
                onChange={(e) => { setNewExt(e.target.value); setExtError('') }}
                onKeyDown={(e) => { if (e.key === 'Enter') handleAddExt() }}
                placeholder="e.g. .mp4"
                className="input-dark text-sm w-full font-mono"
              />
              {extError && (
                <p className="text-xs text-cyber-red mt-1">{extError}</p>
              )}
            </div>
            <button
              onClick={handleAddExt}
              disabled={!newExt.trim()}
              className="btn-ghost text-sm disabled:opacity-40 disabled:cursor-not-allowed"
            >
              Add
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function CacheTab() {
  const [loading, setLoading] = useState(true)
  const [cacheSize, setCacheSize] = useState({ images: 0, previews: 0, searchIndex: 0 })
  const [indexSceneCount, setIndexSceneCount] = useState(0)
  const [indexInProgress, setIndexInProgress] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 4000)
  }

  const loadState = useCallback(async () => {
    setLoading(true)
    try {
      const [stateRes, searchRes] = await Promise.all([
        fetch('/api/options/state'),
        fetch('/api/options/state/search'),
      ])
      const state = await stateRes.json()
      const search = await searchRes.json()
      setCacheSize(state.currentState?.cacheSize || { images: 0, previews: 0, searchIndex: 0 })
      setIndexSceneCount(search.documentCount || 0)
      setIndexInProgress(Boolean(search.inProgress))
    } catch {}
    setLoading(false)
  }, [])

  useEffect(() => {
    loadState()
  }, [loadState])

  const resetCache = async (kind: 'images' | 'previews' | 'searchIndex', label: string) => {
    try {
      await fetch(`/api/options/cache/reset/${kind}`, { method: 'DELETE' })
      showMessage('success', `${label} cache cleared`)
      await loadState()
    } catch {
      showMessage('error', `Failed to clear ${label.toLowerCase()} cache`)
    }
  }

  const rebuildIndex = async () => {
    try {
      await fetch('/api/task/index')
      setIndexInProgress(true)
      showMessage('success', 'Search index rebuild started')
    } catch {
      showMessage('error', 'Failed to start search index rebuild')
    }
  }

  const refreshScenes = async () => {
    try {
      await fetch('/api/task/scene-refresh')
      showMessage('success', 'Scene refresh started')
    } catch {
      showMessage('error', 'Failed to start scene refresh')
    }
  }

  return (
    <div className="space-y-6">
      {message && (
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0 }}
          className={`flex items-center gap-2 px-4 py-3 rounded-xl text-sm ${
            message.type === 'success'
              ? 'bg-cyber-teal/15 text-cyber-teal border border-cyber-teal/20'
              : 'bg-cyber-red/15 text-cyber-red border border-cyber-red/20'
          }`}
        >
          {message.type === 'success' ? <Check className="w-4 h-4" /> : <AlertTriangle className="w-4 h-4" />}
          {message.text}
        </motion.div>
      )}

      <div>
        <h3 className="text-lg font-semibold text-surface-100">Cache management</h3>
        <p className="text-sm text-surface-400 mt-1">
          Reset cached assets and rebuild the search index when cache state gets stale.
        </p>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <div className="w-6 h-6 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
            <div>
              <div className="text-sm font-medium text-surface-200">Image cache</div>
              <div className="text-xs text-surface-500">Cached remote scene and actor artwork</div>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-xs text-surface-400 font-mono">{formatBytes(cacheSize.images)}</span>
              <button
                onClick={() => resetCache('images', 'Image')}
                className="btn-ghost text-xs text-cyber-red hover:bg-cyber-red/10"
              >
                Reset
              </button>
            </div>
          </div>

          <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
            <div>
              <div className="text-sm font-medium text-surface-200">Preview cache</div>
              <div className="text-xs text-surface-500">Generated local previews and thumbnails</div>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-xs text-surface-400 font-mono">{formatBytes(cacheSize.previews)}</span>
              <button
                onClick={() => resetCache('previews', 'Preview')}
                className="btn-ghost text-xs text-cyber-red hover:bg-cyber-red/10"
              >
                Reset
              </button>
            </div>
          </div>

          <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
            <div>
              <div className="text-sm font-medium text-surface-200">Search index</div>
              <div className="text-xs text-surface-500">
                {indexInProgress ? 'Indexing in progress' : `${indexSceneCount.toLocaleString()} scenes indexed`}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-surface-400 font-mono">{formatBytes(cacheSize.searchIndex)}</span>
              <button
                onClick={() => resetCache('searchIndex', 'Search index')}
                className="btn-ghost text-xs text-cyber-red hover:bg-cyber-red/10"
              >
                Reset
              </button>
              <button
                onClick={rebuildIndex}
                className="btn-ghost text-xs"
              >
                Rebuild
              </button>
            </div>
          </div>

          <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
            <div>
              <div className="text-sm font-medium text-surface-200">Scene status</div>
              <div className="text-xs text-surface-500">Refresh availability and scripted status from assigned files</div>
            </div>
            <button
              onClick={refreshScenes}
              className="btn-ghost text-xs"
            >
              Refresh scenes
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default function Settings() {
  const lockRescan = useAppStore((s) => s.lockRescan)
  const lockScrape = useAppStore((s) => s.lockScrape)
  const lastScrapeMessage = useAppStore((s) => s.lastScrapeMessage)
  const runningScrapers = useAppStore((s) => s.runningScrapers)
  const [version, setVersion] = useState('')
  const [activeTab, setActiveTab] = useState('storage')

  useEffect(() => {
    fetch('/api/options/state')
      .then(r => r.json())
      .then(d => setVersion(d.currentState?.server?.version || ''))
      .catch(() => {})
  }, [])

  const tabs = [
    { id: 'storage', label: 'Storage', icon: HardDrive },
    { id: 'sites', label: 'Sites', icon: Globe },
    { id: 'cache', label: 'Cache', icon: Database },
    { id: 'interface', label: 'Interface', icon: Monitor },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h1 className="text-xl font-bold text-surface-100">Settings</h1>
        {version && (
          <span className="badge bg-surface-700/50 text-surface-400 text-xs">v{version}</span>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Sidebar tabs */}
        <div className="lg:col-span-1">
          <nav className="glass rounded-xl p-2 space-y-0.5">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors
                    ${activeTab === tab.id
                      ? 'bg-accent/15 text-accent'
                      : 'text-surface-400 hover:text-surface-200 hover:bg-surface-700/30'
                    }`}
                >
                  <Icon className="w-4 h-4" />
                  {tab.label}
                </button>
              )
            })}
          </nav>

          {/* Quick actions */}
          <div className="glass rounded-xl p-4 mt-4 space-y-3">
            <h3 className="text-xs text-surface-500 uppercase tracking-wider font-medium">Quick actions</h3>
            <button
              onClick={() => fetch('/api/task/rescan')}
              disabled={lockRescan}
              className="btn-primary w-full text-sm flex items-center justify-center gap-2"
            >
              <RefreshCw className={`w-4 h-4 ${lockRescan ? 'animate-spin' : ''}`} />
              {lockRescan ? 'Rescan running...' : 'Rescan library'}
            </button>
            <button
              onClick={() => fetch('/api/task/scrape')}
              disabled={lockScrape}
              className="btn-ghost w-full text-sm flex items-center justify-center gap-2"
            >
              <Globe className="w-4 h-4" />
              {lockScrape ? 'Scrape running...' : 'Scrape sites'}
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="lg:col-span-3">
          <motion.div
            key={activeTab}
            initial={{ opacity: 0, x: 10 }}
            animate={{ opacity: 1, x: 0 }}
            className="glass rounded-xl p-6"
          >
            {activeTab === 'storage' && <StorageTab />}

            {activeTab === 'sites' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-surface-100">Site scrapers</h3>
                <p className="text-sm text-surface-400">
                  Configure which sites to scrape for scene metadata. Access the full scraper
                  configuration through the main XBVR options panel.
                </p>
                {lastScrapeMessage && (
                  <div className={`flex items-center gap-2 px-4 py-3 rounded-xl text-sm border ${
                    lockScrape
                      ? 'bg-cyber-teal/10 text-cyber-teal border-cyber-teal/20'
                      : 'bg-surface-800/50 text-surface-300 border-surface-700/30'
                  }`}>
                    <RefreshCw className={`w-4 h-4 ${lockScrape ? 'animate-spin' : ''}`} />
                    <span>{lastScrapeMessage.message}</span>
                    {runningScrapers.length > 0 && (
                      <span className="ml-auto text-xs text-surface-500">
                        {runningScrapers.length} running
                      </span>
                    )}
                  </div>
                )}
                <button
                  onClick={() => fetch('/api/task/scrape')}
                  disabled={lockScrape}
                  className="btn-primary text-sm flex items-center gap-2"
                >
                  <RefreshCw className={`w-4 h-4 ${lockScrape ? 'animate-spin' : ''}`} />
                  {lockScrape ? 'Scrape running...' : 'Run all scrapers'}
                </button>
              </div>
            )}

            {activeTab === 'cache' && <CacheTab />}

            {activeTab === 'interface' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-surface-100">Interface</h3>
                <p className="text-sm text-surface-400">
                  This new UI is a modern alternative to the classic XBVR interface. Both UIs
                  connect to the same backend and share the same data.
                </p>
                <div className="p-4 rounded-xl bg-accent/5 border border-accent/20">
                  <div className="flex items-center gap-2 mb-2">
                    <Palette className="w-4 h-4 text-accent" />
                    <span className="text-sm font-medium text-accent">New UI</span>
                  </div>
                  <p className="text-xs text-surface-400 leading-relaxed">
                    Built with React, TypeScript, Tailwind CSS, and Framer Motion.
                    Features a modern glass-morphism design with a cinematic dark theme.
                  </p>
                </div>
              </div>
            )}
          </motion.div>
        </div>
      </div>
    </div>
  )
}
