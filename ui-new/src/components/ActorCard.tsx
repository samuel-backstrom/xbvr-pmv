import { Star, Film } from 'lucide-react'
import { useAppStore } from '../store'
import { imageUrl, type Actor } from '../api/client'

export default function ActorCard({ actor }: { actor: Actor }) {
  const openDetail = useAppStore((s) => s.openActorDetail)

  const imgSrc = actor.image_url
    ? imageUrl(actor.image_url)
    : '/ui/images/blank_female_profile.png'

  const sceneCount = actor.scenes?.length || 0

  return (
    <div
      onClick={() => openDetail(actor)}
      className="group glass rounded-xl overflow-hidden card-hover cursor-pointer"
    >
      {/* Image */}
      <div className="relative aspect-[3/4] overflow-hidden">
        <img
          src={imgSrc}
          alt={actor.name}
          className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
          loading="lazy"
          onError={(e) => { (e.target as HTMLImageElement).src = '/ui/images/blank_female_profile.png' }}
        />
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent" />

        {/* Name overlay */}
        <div className="absolute bottom-0 left-0 right-0 p-3">
          <h3 className="text-sm font-semibold text-white truncate">{actor.name}</h3>
          <div className="flex items-center gap-2 mt-1">
            <span className="text-xs text-surface-300 flex items-center gap-1">
              <Film className="w-3 h-3" />
              {sceneCount} scene{sceneCount !== 1 ? 's' : ''}
            </span>
            {actor.avail_count > 0 && (
              <span className="text-xs text-cyber-teal">
                {actor.avail_count} avail
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Rating bar */}
      {(actor.star_rating > 0 || actor.scene_rating_average > 0) && (
        <div className="px-3 py-2 flex items-center justify-between">
          {actor.star_rating > 0 && (
            <div className="flex items-center gap-1">
              {Array.from({ length: 5 }, (_, i) => (
                <Star
                  key={i}
                  className={`w-3 h-3 ${i < actor.star_rating
                    ? 'fill-cyber-amber text-cyber-amber'
                    : 'text-surface-600'
                  }`}
                />
              ))}
            </div>
          )}
          {actor.scene_rating_average > 0 && (
            <span className="text-[11px] text-surface-400">
              Avg {Math.round(actor.scene_rating_average * 4) / 4}★
            </span>
          )}
        </div>
      )}
    </div>
  )
}
