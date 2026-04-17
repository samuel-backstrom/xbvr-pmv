import { useEffect, useState, useCallback, useRef } from 'react'
import { motion } from 'framer-motion'
import {
  SlidersHorizontal, ChevronDown, ArrowUpDown, X, Star,
} from 'lucide-react'
import ActorCard from '../components/ActorCard'
import { useAppStore } from '../store'
import { getActors, getActorFilters, type Actor } from '../api/client'

type SortOption = { label: string; field: string; dir: string }
const SORTS: SortOption[] = [
  { label: 'Name A-Z', field: 'name', dir: 'asc' },
  { label: 'Scene count', field: 'scene_count', dir: 'desc' },
  { label: 'Rating', field: 'star_rating', dir: 'desc' },
  { label: 'Added date', field: 'added_date', dir: 'desc' },
  { label: 'Birth date', field: 'birth_date', dir: 'desc' },
]

const CARD_SIZES = [
  { label: 'S', cols: 'grid-cols-3 sm:grid-cols-4 md:grid-cols-5 lg:grid-cols-6 xl:grid-cols-8' },
  { label: 'M', cols: 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6' },
  { label: 'L', cols: 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5' },
]

export default function Actors() {
  const actorListRevision = useAppStore((s) => s.actorListRevision)
  const [actors, setActors] = useState<Actor[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit] = useState(48)
  const [sort, setSort] = useState(SORTS[0])
  const [cardSize, setCardSize] = useState(1)
  const [sortOpen, setSortOpen] = useState(false)
  const sortRef = useRef<HTMLDivElement>(null)

  const [filtersOpen, setFiltersOpen] = useState(false)
  const [ratingFilter, setRatingFilter] = useState(0)

  const hasActiveFilters = ratingFilter > 0

  const fetchActors = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, any> = {
        sort: sort.field,
        order: sort.dir,
        offset: (page - 1) * limit,
        limit,
      }
      if (ratingFilter > 0) params.rating = ratingFilter

      const data = await getActors(params)
      setActors(data.actors || [])
      setTotal(data.results || 0)
    } catch {
      setActors([])
    }
    setLoading(false)
  }, [sort, page, limit, ratingFilter, actorListRevision])

  useEffect(() => {
    fetchActors()
  }, [fetchActors])

  useEffect(() => {
    setPage(1)
  }, [sort, ratingFilter])

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (sortRef.current && !sortRef.current.contains(e.target as Node)) {
        setSortOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  const totalPages = Math.ceil(total / limit)

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold text-surface-100">Actors</h1>
          <span className="text-sm text-surface-500">{total.toLocaleString()} total</span>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setFiltersOpen(!filtersOpen)}
            className={`btn-ghost text-sm flex items-center gap-1.5 ${hasActiveFilters ? 'text-accent' : ''}`}
          >
            <SlidersHorizontal className="w-4 h-4" />
            Filters
            {hasActiveFilters && <span className="w-1.5 h-1.5 rounded-full bg-accent" />}
          </button>

          <div className="relative" ref={sortRef}>
            <button
              onClick={() => setSortOpen(!sortOpen)}
              className="btn-ghost text-sm flex items-center gap-1.5"
            >
              <ArrowUpDown className="w-4 h-4" />
              {sort.label}
              <ChevronDown className={`w-3 h-3 transition-transform ${sortOpen ? 'rotate-180' : ''}`} />
            </button>
            {sortOpen && (
              <div className="absolute right-0 mt-1 w-48 glass-strong rounded-xl py-1 z-20 shadow-xl">
                {SORTS.map((s) => (
                  <button
                    key={s.field}
                    onClick={() => { setSort(s); setSortOpen(false) }}
                    className={`w-full text-left px-4 py-2 text-sm transition-colors
                      ${sort.field === s.field ? 'text-accent bg-accent/10' : 'text-surface-300 hover:bg-surface-700/50 hover:text-surface-100'}`}
                  >
                    {s.label}
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="flex items-center border border-surface-700 rounded-lg overflow-hidden">
            {CARD_SIZES.map((s, i) => (
              <button
                key={s.label}
                onClick={() => setCardSize(i)}
                className={`px-2.5 py-1.5 text-xs font-medium transition-colors
                  ${cardSize === i ? 'bg-accent text-white' : 'text-surface-400 hover:text-surface-200 hover:bg-surface-700/50'}`}
              >
                {s.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Filter panel */}
      {filtersOpen && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: 'auto' }}
          className="glass rounded-xl p-4 space-y-3"
        >
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-surface-200">Filters</span>
            {hasActiveFilters && (
              <button onClick={() => setRatingFilter(0)} className="text-xs text-accent hover:text-accent-hover flex items-center gap-1">
                <X className="w-3 h-3" /> Clear
              </button>
            )}
          </div>
          <div>
            <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Min rating</label>
            <div className="flex items-center gap-1 mt-1">
              {Array.from({ length: 5 }, (_, i) => {
                const val = i + 1
                return (
                  <button key={i} onClick={() => setRatingFilter(ratingFilter === val ? 0 : val)} className="p-0.5">
                    <Star className={`w-4 h-4 transition-colors ${val <= ratingFilter
                      ? 'fill-cyber-amber text-cyber-amber' : 'text-surface-600 hover:text-surface-400'}`} />
                  </button>
                )
              })}
              {ratingFilter > 0 && <span className="text-xs text-surface-400 ml-1">& up</span>}
            </div>
          </div>
        </motion.div>
      )}

      {/* Actor grid */}
      {loading && actors.length === 0 ? (
        <div className="flex items-center justify-center py-32">
          <div className="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : actors.length === 0 ? (
        <div className="text-center py-32">
          <p className="text-surface-400 text-lg">No actors found</p>
          <p className="text-surface-600 text-sm mt-1">Try adjusting your filters</p>
        </div>
      ) : (
        <>
          <div className={`grid ${CARD_SIZES[cardSize].cols} gap-4`}>
            {actors.map((actor, i) => (
              <motion.div
                key={actor.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: Math.min(i * 0.02, 0.5) }}
              >
                <ActorCard actor={actor} />
              </motion.div>
            ))}
          </div>

          {totalPages > 1 && (
            <div className="flex items-center justify-center gap-2 pt-4 pb-8">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="btn-ghost text-sm disabled:opacity-30 disabled:cursor-not-allowed"
              >
                Previous
              </button>
              <div className="flex items-center gap-1">
                {Array.from({ length: Math.min(totalPages, 7) }, (_, i) => {
                  let pageNum: number
                  if (totalPages <= 7) pageNum = i + 1
                  else if (page <= 4) pageNum = i + 1
                  else if (page >= totalPages - 3) pageNum = totalPages - 6 + i
                  else pageNum = page - 3 + i
                  return (
                    <button
                      key={pageNum}
                      onClick={() => setPage(pageNum)}
                      className={`w-9 h-9 rounded-lg text-sm font-medium transition-colors
                        ${page === pageNum ? 'bg-accent text-white' : 'text-surface-400 hover:bg-surface-700/50 hover:text-surface-200'}`}
                    >
                      {pageNum}
                    </button>
                  )
                })}
              </div>
              <button
                onClick={() => setPage(Math.min(totalPages, page + 1))}
                disabled={page === totalPages}
                className="btn-ghost text-sm disabled:opacity-30 disabled:cursor-not-allowed"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
