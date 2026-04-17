import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  X, Star, Clock, Calendar, ExternalLink, Eye, EyeOff,
  Heart, Bookmark, Scroll, ChevronLeft, ChevronRight, Globe,
} from 'lucide-react'
import { useAppStore } from '../store'
import { imageUrl, getScene, rateScene, type Scene } from '../api/client'
import { format, parseISO } from 'date-fns'

export default function SceneDetailModal() {
  const scene = useAppStore((s) => s.selectedScene)
  const close = useAppStore((s) => s.closeSceneDetail)
  const bumpSceneListRevision = useAppStore((s) => s.bumpSceneListRevision)
  const [detail, setDetail] = useState<Scene | null>(null)
  const [imgIdx, setImgIdx] = useState(0)
  const [hoveredStar, setHoveredStar] = useState(0)

  useEffect(() => {
    if (!scene) { setDetail(null); return }
    setImgIdx(0)
    getScene(scene.id).then(setDetail).catch(() => setDetail(scene))
  }, [scene])

  useEffect(() => {
    if (!scene) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [scene, close])

  const item = detail || scene
  if (!item) return null

  const releaseDate = item.release_date && item.release_date !== '0001-01-01T00:00:00Z'
    ? format(parseISO(item.release_date), 'MMM d, yyyy')
    : null

  const videoFiles = item.file?.filter(f => f.type === 'video') || []
  const scriptFiles = item.file?.filter(f => f.type === 'script') || []

  const handleRate = async (rating: number) => {
    try {
      await rateScene(item.id, rating)
      setDetail((d) => d ? { ...d, star_rating: rating } : d)
      bumpSceneListRevision()
    } catch {}
  }

  return (
    <AnimatePresence>
      {scene && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50"
            onClick={close}
          />
          <motion.div
            initial={{ opacity: 0, y: 40 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 40 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="fixed inset-4 sm:inset-8 lg:inset-y-8 lg:left-[12%] lg:right-[12%] z-50
              glass-strong rounded-2xl overflow-hidden flex flex-col"
          >
            {/* Header bar */}
            <div className="flex items-center justify-between px-6 py-3 border-b border-surface-700/50">
              <h2 className="text-lg font-semibold text-surface-100 truncate pr-4">
                {item.title}
              </h2>
              <button onClick={close} className="p-1 rounded-lg hover:bg-surface-700/50 text-surface-400 hover:text-white transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Body */}
            <div className="flex-1 overflow-y-auto">
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-0">
                {/* Left - Image */}
                <div className="relative bg-black">
                  <img
                    src={imageUrl(item.cover_url, '1600x')}
                    alt={item.title}
                    className="w-full h-full object-contain max-h-[60vh]"
                    onError={(e) => { (e.target as HTMLImageElement).src = '/ui/images/blank.png' }}
                  />
                </div>

                {/* Right - Details */}
                <div className="p-6 space-y-5 overflow-y-auto max-h-[60vh]">
                  {/* Meta row */}
                  <div className="flex flex-wrap items-center gap-3 text-sm">
                    {item.site && (
                      <a href={item.scene_url} target="_blank" rel="noreferrer"
                        className="badge bg-accent/15 text-accent-hover hover:bg-accent/25 transition-colors">
                        <Globe className="w-3 h-3" /> {item.site} <ExternalLink className="w-3 h-3 ml-0.5" />
                      </a>
                    )}
                    {releaseDate && (
                      <span className="badge bg-surface-700/50 text-surface-300">
                        <Calendar className="w-3 h-3" /> {releaseDate}
                      </span>
                    )}
                    {item.duration > 0 && (
                      <span className="badge bg-surface-700/50 text-surface-300">
                        <Clock className="w-3 h-3" /> {item.duration} min
                      </span>
                    )}
                  </div>

                  {/* Rating */}
                  <div>
                    <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Rating</label>
                    <div className="flex items-center gap-1 mt-1">
                      {Array.from({ length: 5 }, (_, i) => {
                        const val = i + 1
                        const filled = val <= (hoveredStar || item.star_rating)
                        return (
                          <button
                            key={i}
                            onClick={() => handleRate(val === item.star_rating ? 0 : val)}
                            onMouseEnter={() => setHoveredStar(val)}
                            onMouseLeave={() => setHoveredStar(0)}
                            className="p-0.5"
                          >
                            <Star className={`w-5 h-5 transition-colors ${filled
                              ? 'fill-cyber-amber text-cyber-amber'
                              : 'text-surface-600 hover:text-surface-400'
                            }`} />
                          </button>
                        )
                      })}
                    </div>
                  </div>

                  {/* Cast */}
                  {item.cast && item.cast.length > 0 && (
                    <div>
                      <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Cast</label>
                      <div className="flex flex-wrap gap-1.5 mt-1.5">
                        {item.cast.map((c) => (
                          <span key={c.id} className="badge bg-cyber-amber/10 text-cyber-amber cursor-pointer hover:bg-cyber-amber/20 transition-colors">
                            {c.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Tags */}
                  {item.tags && item.tags.length > 0 && (
                    <div>
                      <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Tags</label>
                      <div className="flex flex-wrap gap-1.5 mt-1.5">
                        {item.tags.map((t) => (
                          <span key={t.id} className="badge bg-cyber-blue/10 text-cyber-blue">
                            {t.name}
                          </span>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Synopsis */}
                  {item.synopsis && (
                    <div>
                      <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Synopsis</label>
                      <p className="text-sm text-surface-300 mt-1 leading-relaxed">{item.synopsis}</p>
                    </div>
                  )}

                  {/* Files */}
                  {videoFiles.length > 0 && (
                    <div>
                      <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                        Files ({videoFiles.length})
                      </label>
                      <div className="space-y-2 mt-1.5">
                        {videoFiles.map((f) => (
                          <div key={f.id} className="flex items-center gap-3 p-2 rounded-lg bg-surface-700/30 text-sm">
                            <div className="flex-1 min-w-0">
                              <div className="text-surface-200 truncate text-xs font-mono">{f.filename}</div>
                            </div>
                            <span className="text-xs text-surface-400 flex-shrink-0">
                              {f.video_width}x{f.video_height}
                            </span>
                            <span className="text-xs text-surface-500 flex-shrink-0">
                              {(f.size / (1024 * 1024 * 1024)).toFixed(1)} GB
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
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
