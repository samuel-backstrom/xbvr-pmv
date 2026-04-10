import { Star, Clock, Scroll, Eye, Heart } from 'lucide-react'
import { useAppStore } from '../store'
import { imageUrl, type Scene } from '../api/client'
import { format, parseISO } from 'date-fns'

export default function SceneCard({ scene }: { scene: Scene }) {
  const openDetail = useAppStore((s) => s.openSceneDetail)

  const scriptCount = scene.file?.filter(f => f.type === 'script').length || 0
  const releaseDate = scene.release_date && scene.release_date !== '0001-01-01T00:00:00Z'
    ? format(parseISO(scene.release_date), 'yyyy-MM-dd')
    : null

  return (
    <div
      onClick={() => openDetail(scene)}
      className="group glass rounded-xl overflow-hidden card-hover cursor-pointer"
    >
      {/* Image */}
      <div className="relative aspect-video overflow-hidden">
        <img
          src={imageUrl(scene.cover_url)}
          alt={scene.title}
          className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
          loading="lazy"
          onError={(e) => { (e.target as HTMLImageElement).src = '/ui/images/blank.png' }}
        />

        {/* Gradient overlay */}
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent
          opacity-60 group-hover:opacity-80 transition-opacity" />

        {/* Badges */}
        <div className="absolute bottom-2 left-2 right-2 flex flex-wrap items-end gap-1.5">
          {scene.is_watched && (
            <span className="badge bg-accent/80 text-white backdrop-blur-sm">
              <Eye className="w-3 h-3" /> Watched
            </span>
          )}
          {scene.is_scripted && (
            <span className="badge bg-cyber-teal/80 text-white backdrop-blur-sm">
              <Scroll className="w-3 h-3" /> Script
              {scriptCount > 1 && <span className="ml-0.5">{scriptCount}</span>}
            </span>
          )}
          {scene.duration > 0 && (
            <span className="badge bg-black/60 text-surface-200 backdrop-blur-sm ml-auto">
              <Clock className="w-3 h-3" /> {scene.duration}m
            </span>
          )}
        </div>

        {/* Availability indicator */}
        {!scene.is_available && (
          <div className="absolute top-2 right-2">
            <div className="w-2 h-2 rounded-full bg-surface-500" title="Not available" />
          </div>
        )}
      </div>

      {/* Content */}
      <div className="p-3">
        <h3 className="text-sm font-medium text-surface-100 truncate group-hover:text-white transition-colors">
          {scene.title}
        </h3>

        <div className="flex items-center justify-between mt-1.5">
          <span className="text-xs text-accent-hover font-medium">{scene.site}</span>
          {releaseDate && (
            <span className="text-[11px] text-surface-500">{releaseDate}</span>
          )}
        </div>

        {scene.cast && scene.cast.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {scene.cast.slice(0, 3).map((c) => (
              <span key={c.id} className="text-[11px] text-surface-400 bg-surface-700/50 rounded px-1.5 py-0.5">
                {c.name}
              </span>
            ))}
            {scene.cast.length > 3 && (
              <span className="text-[11px] text-surface-500">+{scene.cast.length - 3}</span>
            )}
          </div>
        )}

        {scene.star_rating > 0 && (
          <div className="flex items-center gap-1 mt-2">
            {Array.from({ length: 5 }, (_, i) => (
              <Star
                key={i}
                className={`w-3 h-3 ${i < scene.star_rating
                  ? 'fill-cyber-amber text-cyber-amber'
                  : 'text-surface-600'
                }`}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
