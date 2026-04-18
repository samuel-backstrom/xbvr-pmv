import { useEffect, useMemo, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import {
  ArrowLeft, ArrowRight, Calendar, Clock, ExternalLink, Search, X,
} from 'lucide-react'
import { format, parseISO } from 'date-fns'
import { deriveMatchQueryFromFilename } from '../pages/files-utils.js'
import { imageUrl, searchScenes, type Scene, type SceneFile } from '../api/client'

type SearchScene = Scene & { score?: number }

export default function FileMatchModal({
  file,
  onClose,
  onPrevious,
  onNext,
  onAssign,
}: {
  file: SceneFile | null
  onClose: () => void
  onPrevious?: () => void
  onNext?: () => void
  onAssign: (sceneId: string) => Promise<void> | void
}) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchScene[]>([])
  const [loading, setLoading] = useState(false)
  const [assigning, setAssigning] = useState(false)

  useEffect(() => {
    if (!file) return
    setQuery(deriveMatchQueryFromFilename(file.filename))
    setResults([])
  }, [file])

  useEffect(() => {
    if (!file) return

    const trimmed = query.trim()
    if (trimmed.length < 2) {
      setResults([])
      setLoading(false)
      return
    }

    let active = true
    setLoading(true)

    const timeout = window.setTimeout(() => {
      searchScenes(trimmed)
        .then((data) => {
          if (!active) return
          setResults(((data.scenes || []) as SearchScene[]))
        })
        .catch(() => {
          if (!active) return
          setResults([])
        })
        .finally(() => {
          if (active) setLoading(false)
        })
    }, 250)

    return () => {
      active = false
      window.clearTimeout(timeout)
    }
  }, [file, query])

  useEffect(() => {
    if (!file) return

    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }

    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [file, onClose])

  const fileMeta = useMemo(() => {
    if (!file) return null
    const created = format(parseISO(file.created_time), 'yyyy-MM-dd')
    return {
      created,
      size: `${(file.size / (1024 * 1024 * 1024)).toFixed(1)} GB`,
    }
  }, [file])

  return (
    <AnimatePresence>
      {file && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm"
            onClick={onClose}
          />
          <motion.div
            initial={{ opacity: 0, y: 28, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 28, scale: 0.98 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="fixed inset-4 md:inset-8 z-50 glass-strong rounded-2xl overflow-hidden flex flex-col"
          >
            <div className="flex items-center justify-between gap-3 px-5 py-4 border-b border-surface-700/50">
              <div className="min-w-0">
                <div className="flex items-center gap-2 text-sm text-surface-400">
                  <Search className="w-4 h-4 text-cyber-pink" />
                  Match file to scene
                </div>
                <h2 className="text-base sm:text-lg font-semibold text-surface-100 truncate">
                  {file.filename}
                </h2>
              </div>
              <div className="flex items-center gap-2">
                {onPrevious && (
                  <button
                    onClick={onPrevious}
                    className="btn-ghost flex items-center gap-1.5"
                    aria-label="Previous file"
                  >
                    <ArrowLeft className="w-4 h-4" />
                    Prev
                  </button>
                )}
                {onNext && (
                  <button
                    onClick={onNext}
                    className="btn-ghost flex items-center gap-1.5"
                    aria-label="Next file"
                  >
                    Next
                    <ArrowRight className="w-4 h-4" />
                  </button>
                )}
                <button
                  onClick={onClose}
                  className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-400 hover:text-white transition-colors"
                  aria-label="Close matcher"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)] gap-0 flex-1 min-h-0">
              <div className="border-b lg:border-b-0 lg:border-r border-surface-700/40 p-5 space-y-4 overflow-y-auto">
                <div className="space-y-2">
                  <div className="text-xs uppercase tracking-wider text-surface-500 font-medium">File</div>
                  <div className="rounded-xl bg-surface-800/50 border border-surface-700/40 p-4 space-y-2">
                    <div className="font-mono text-sm text-surface-100 break-all">{file.filename}</div>
                    <div className="font-mono text-[11px] text-surface-500 break-all">{file.path}</div>
                    <div className="flex flex-wrap gap-2 text-xs text-surface-400">
                      <span className="badge bg-surface-700/50">{fileMeta?.size}</span>
                      <span className="badge bg-surface-700/50">
                        {file.video_width > 0 ? `${file.video_width}x${file.video_height}` : 'No resolution'}
                      </span>
                      <span className="badge bg-surface-700/50">{fileMeta?.created}</span>
                    </div>
                  </div>
                </div>

                <label className="block">
                  <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
                    Search
                  </div>
                  <input
                    value={query}
                    onChange={(e) => setQuery(e.target.value)}
                    className="input-dark w-full"
                    placeholder="Search scenes"
                    autoFocus
                  />
                </label>
                <div className="text-[11px] text-surface-500">
                  The query starts from the file name, then you can narrow it with any scene keywords.
                </div>
              </div>

              <div className="overflow-y-auto min-h-0">
                <div className="px-5 py-4 border-b border-surface-700/40 flex items-center justify-between gap-3">
                  <div className="text-sm text-surface-300">
                    {loading ? 'Searching…' : `${results.length} result${results.length === 1 ? '' : 's'}`}
                  </div>
                  <div className="text-xs text-surface-500">
                    Click Assign to match the file to a scene
                  </div>
                </div>

                <div className="p-4 space-y-3">
                  {!loading && results.length === 0 ? (
                    <div className="rounded-xl border border-dashed border-surface-700/50 p-8 text-center text-surface-500">
                      No scenes found yet. Try a shorter search phrase or remove release metadata.
                    </div>
                  ) : (
                    results.map((scene) => {
                      const release = scene.release_date && scene.release_date !== '0001-01-01T00:00:00Z'
                        ? format(parseISO(scene.release_date), 'yyyy-MM-dd')
                        : 'Unknown'

                      return (
                        <div
                          key={scene.id}
                          className="rounded-xl border border-surface-700/40 bg-surface-800/50 overflow-hidden"
                        >
                          <div className="flex flex-col md:flex-row gap-4 p-4">
                            <img
                              src={imageUrl(scene.cover_url, '300x')}
                              alt={scene.title}
                              className="w-full md:w-28 h-40 md:h-36 rounded-lg object-cover bg-black/50 flex-shrink-0"
                              onError={(e) => { (e.target as HTMLImageElement).src = '/ui/images/blank.png' }}
                            />
                            <div className="flex-1 min-w-0 space-y-2">
                              <div className="flex items-start justify-between gap-3">
                                <div className="min-w-0">
                                  <h3 className="text-sm font-semibold text-surface-100 truncate">
                                    {scene.title}
                                  </h3>
                                  <div className="text-xs text-surface-500 flex items-center gap-2 mt-1">
                                    <span className="badge bg-accent/15 text-accent">{scene.site || 'Unknown site'}</span>
                                    <span className="flex items-center gap-1">
                                      <Calendar className="w-3 h-3" />
                                      {release}
                                    </span>
                                    {scene.duration > 0 && (
                                      <span className="flex items-center gap-1">
                                        <Clock className="w-3 h-3" />
                                        {scene.duration} min
                                      </span>
                                    )}
                                  </div>
                                </div>
                                <button
                                  onClick={async () => {
                                    if (assigning) return
                                    setAssigning(true)
                                    try {
                                      await onAssign(scene.scene_id)
                                    } finally {
                                      setAssigning(false)
                                    }
                                  }}
                                  disabled={assigning}
                                  className="btn-primary text-xs whitespace-nowrap disabled:opacity-60"
                                >
                                  {assigning ? 'Assigning…' : 'Assign'}
                                </button>
                              </div>

                              {scene.score !== undefined && (
                                <div className="text-[11px] text-surface-500">
                                  Score: {scene.score.toFixed(2)}
                                </div>
                              )}

                              {scene.cast && scene.cast.length > 0 && (
                                <div className="flex flex-wrap gap-1.5">
                                  {scene.cast.slice(0, 4).map((actor) => (
                                    <span key={actor.id} className="badge bg-surface-700/50 text-surface-300">
                                      {actor.name}
                                    </span>
                                  ))}
                                  {scene.cast.length > 4 && (
                                    <span className="badge bg-surface-700/50 text-surface-400">
                                      +{scene.cast.length - 4}
                                    </span>
                                  )}
                                </div>
                              )}

                              {scene.scene_url && (
                                <a
                                  href={scene.scene_url}
                                  target="_blank"
                                  rel="noreferrer"
                                  className="inline-flex items-center gap-1 text-xs text-accent hover:text-accent-hover"
                                >
                                  Open scene <ExternalLink className="w-3 h-3" />
                                </a>
                              )}
                            </div>
                          </div>
                        </div>
                      )
                    })
                  )}
                </div>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
