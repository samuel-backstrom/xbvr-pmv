import { useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  X, Star, MapPin, Ruler, Weight, Calendar, Film,
} from 'lucide-react'
import { useAppStore } from '../store'
import { imageUrl, getActor, rateActor, type Actor } from '../api/client'
import { format, parseISO } from 'date-fns'

function InfoRow({ icon: Icon, label, value }: { icon: any; label: string; value: string }) {
  return (
    <div className="flex items-center gap-2.5 py-1.5">
      <Icon className="w-3.5 h-3.5 text-surface-500 flex-shrink-0" />
      <span className="text-xs text-surface-500 w-20 flex-shrink-0">{label}</span>
      <span className="text-sm text-surface-200">{value}</span>
    </div>
  )
}

export default function ActorDetailModal() {
  const actor = useAppStore((s) => s.selectedActor)
  const close = useAppStore((s) => s.closeActorDetail)
  const openScene = useAppStore((s) => s.openSceneDetail)
  const bumpActorListRevision = useAppStore((s) => s.bumpActorListRevision)
  const [detail, setDetail] = useState<Actor | null>(null)
  const [hoveredStar, setHoveredStar] = useState(0)

  useEffect(() => {
    if (!actor) { setDetail(null); return }
    getActor(actor.id).then(setDetail).catch(() => setDetail(actor))
  }, [actor])

  useEffect(() => {
    if (!actor) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') close()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [actor, close])

  const item = detail || actor
  if (!item) return null

  const birthDate = item.birth_date && item.birth_date !== '0001-01-01T00:00:00Z'
    ? format(parseISO(item.birth_date), 'MMM d, yyyy')
    : null

  const imgSrc = item.image_url
    ? imageUrl(item.image_url, '700x')
    : '/ui/images/blank_female_profile.png'

  const images: string[] = item.image_arr ? (() => {
    try { return JSON.parse(item.image_arr).filter((s: string) => s) }
    catch { return [] }
  })() : []

  const handleRate = async (rating: number) => {
    try {
      await rateActor(item.id, rating)
      setDetail((d) => d ? { ...d, star_rating: rating } : d)
      bumpActorListRevision()
    } catch {}
  }

  return (
    <AnimatePresence>
      {actor && (
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
            className="fixed inset-4 sm:inset-8 lg:inset-y-8 lg:left-[15%] lg:right-[15%] z-50
              glass-strong rounded-2xl overflow-hidden flex flex-col"
          >
            {/* Header */}
            <div className="flex items-center justify-between px-6 py-3 border-b border-surface-700/50">
              <h2 className="text-lg font-semibold text-surface-100">{item.name}</h2>
              <button onClick={close} className="p-1 rounded-lg hover:bg-surface-700/50 text-surface-400 hover:text-white transition-colors">
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Body */}
            <div className="flex-1 overflow-y-auto">
              <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 p-6">
                {/* Left - Image */}
                <div className="lg:col-span-1">
                  <img
                    src={imgSrc}
                    alt={item.name}
                    className="w-full rounded-xl object-cover aspect-[3/4]"
                    onError={(e) => { (e.target as HTMLImageElement).src = '/ui/images/blank_female_profile.png' }}
                  />

                  {/* Gallery thumbnails */}
                  {images.length > 1 && (
                    <div className="flex gap-1.5 mt-3 overflow-x-auto pb-2">
                      {images.slice(0, 8).map((url, i) => (
                        <img
                          key={i}
                          src={imageUrl(url, 'x100')}
                          className="w-14 h-14 object-cover rounded-lg flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity cursor-pointer"
                          alt=""
                        />
                      ))}
                    </div>
                  )}
                </div>

                {/* Right - Info */}
                <div className="lg:col-span-2 space-y-5">
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
                      {item.scene_rating_average > 0 && (
                        <span className="ml-3 text-xs text-surface-400">
                          Scene avg: {Math.round(item.scene_rating_average * 4) / 4}★
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Details grid */}
                  <div>
                    <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Details</label>
                    <div className="mt-1.5 divide-y divide-surface-700/30">
                      {birthDate && <InfoRow icon={Calendar} label="Born" value={birthDate} />}
                      {item.nationality && <InfoRow icon={MapPin} label="Nationality" value={item.nationality} />}
                      {item.height > 0 && (
                        <InfoRow icon={Ruler} label="Height" value={`${item.height} cm`} />
                      )}
                      {item.weight > 0 && (
                        <InfoRow icon={Weight} label="Weight" value={`${item.weight} kg`} />
                      )}
                      {item.ethnicity && <InfoRow icon={MapPin} label="Ethnicity" value={item.ethnicity} />}
                      {item.eye_color && <InfoRow icon={MapPin} label="Eyes" value={item.eye_color} />}
                      {item.hair_color && <InfoRow icon={MapPin} label="Hair" value={item.hair_color} />}
                    </div>
                  </div>

                  {/* Scene count */}
                  <div>
                    <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Scenes</label>
                    <div className="flex items-center gap-2 mt-1.5">
                      <Film className="w-4 h-4 text-accent" />
                      <span className="text-sm text-surface-200">
                        {item.scenes?.length || 0} scenes
                        {item.avail_count > 0 && `, ${item.avail_count} available`}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
