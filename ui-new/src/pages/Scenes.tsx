import { useEffect, useState, useCallback, useRef } from 'react'
import { motion } from 'framer-motion'
import {
  SlidersHorizontal, ChevronDown, Grid3X3, LayoutGrid, X,
  ArrowUpDown, Eye, Heart, Bookmark, Scroll, Star,
} from 'lucide-react'
import SceneCard from '../components/SceneCard'
import { useAppStore } from '../store'
import {
  getScenes, getSceneFilters, type Scene, type FilterOptions,
} from '../api/client'

type SortOption = { label: string; field: string; dir: string }
const SORTS: SortOption[] = [
  { label: 'Release date', field: 'release_date_text', dir: 'desc' },
  { label: 'Added date', field: 'added_date', dir: 'desc' },
  { label: 'Rating', field: 'star_rating', dir: 'desc' },
  { label: 'Scene rating', field: 'scene_rating', dir: 'desc' },
  { label: 'Duration', field: 'duration', dir: 'desc' },
  { label: 'Title A-Z', field: 'title', dir: 'asc' },
  { label: 'Random', field: 'random', dir: 'desc' },
]

const CARD_SIZES = [
  { label: 'S', cols: 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6' },
  { label: 'M', cols: 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5' },
  { label: 'L', cols: 'grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4' },
]

type ListFilter = 'watchlist' | 'favourite' | 'wishlist' | null
type AvailFilter = 'available' | 'downloaded' | null

export default function Scenes() {
  const sceneListRevision = useAppStore((s) => s.sceneListRevision)
  const [scenes, setScenes] = useState<Scene[]>([])
  const [loading, setLoading] = useState(true)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [limit] = useState(40)
  const [sort, setSort] = useState(SORTS[0])
  const [cardSize, setCardSize] = useState(1)
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [sortOpen, setSortOpen] = useState(false)
  const sortRef = useRef<HTMLDivElement>(null)

  // Filters
  const [filterOptions, setFilterOptions] = useState<FilterOptions | null>(null)
  const [selectedSites, setSelectedSites] = useState<string[]>([])
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [selectedCast, setSelectedCast] = useState<string[]>([])
  const [listFilter, setListFilter] = useState<ListFilter>(null)
  const [availFilter, setAvailFilter] = useState<AvailFilter>(null)
  const [isScripted, setIsScripted] = useState(false)
  const [isWatched, setIsWatched] = useState<boolean | null>(null)
  const [ratingFilter, setRatingFilter] = useState(0)

  const hasActiveFilters = selectedSites.length > 0 || selectedTags.length > 0 ||
    selectedCast.length > 0 || listFilter || availFilter || isScripted ||
    isWatched !== null || ratingFilter > 0

  // Load filters
  useEffect(() => {
    getSceneFilters().then(setFilterOptions).catch(() => {})
  }, [])

  // Load scenes
  const fetchScenes = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, any> = {
        sort: sort.field,
        order: sort.dir,
        offset: (page - 1) * limit,
        limit,
      }
      if (selectedSites.length > 0) params.sites = selectedSites
      if (selectedTags.length > 0) params.tags = selectedTags
      if (selectedCast.length > 0) params.cast = selectedCast
      if (listFilter) params.list = listFilter
      if (availFilter === 'available') {
        params.is_available = true
        params.is_accessible = true
      }
      if (availFilter === 'downloaded') params.is_accessible = true
      if (isScripted) params.is_scripted = true
      if (isWatched === true) params.is_watched = true
      if (isWatched === false) params.is_watched = false
      if (ratingFilter > 0) params.rating = ratingFilter

      const data = await getScenes(params)
      setScenes(data.scenes || [])
      setTotal(data.results || 0)
    } catch {
      setScenes([])
    }
    setLoading(false)
  }, [sort, page, limit, selectedSites, selectedTags, selectedCast, listFilter, availFilter, isScripted, isWatched, ratingFilter, sceneListRevision])

  useEffect(() => {
    fetchScenes()
  }, [fetchScenes])

  // Reset page on filter change
  useEffect(() => {
    setPage(1)
  }, [sort, selectedSites, selectedTags, selectedCast, listFilter, availFilter, isScripted, isWatched, ratingFilter])

  // Close sort dropdown on outside click
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

  const clearFilters = () => {
    setSelectedSites([])
    setSelectedTags([])
    setSelectedCast([])
    setListFilter(null)
    setAvailFilter(null)
    setIsScripted(false)
    setIsWatched(null)
    setRatingFilter(0)
  }

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold text-surface-100">Scenes</h1>
          <span className="text-sm text-surface-500">{total.toLocaleString()} total</span>
        </div>

        <div className="flex items-center gap-2">
          {/* Filter toggle */}
          <button
            onClick={() => setFiltersOpen(!filtersOpen)}
            className={`btn-ghost text-sm flex items-center gap-1.5 ${hasActiveFilters ? 'text-accent' : ''}`}
          >
            <SlidersHorizontal className="w-4 h-4" />
            Filters
            {hasActiveFilters && (
              <span className="w-1.5 h-1.5 rounded-full bg-accent" />
            )}
          </button>

          {/* Sort dropdown */}
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

          {/* Card size */}
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
          exit={{ opacity: 0, height: 0 }}
          className="glass rounded-xl p-4 space-y-4"
        >
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-surface-200">Filters</span>
            {hasActiveFilters && (
              <button onClick={clearFilters} className="text-xs text-accent hover:text-accent-hover flex items-center gap-1">
                <X className="w-3 h-3" /> Clear all
              </button>
            )}
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {/* Quick filters */}
            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Quick filters</label>
              <div className="flex flex-wrap gap-1.5">
                <button
                  onClick={() => setAvailFilter(availFilter === 'available' ? null : 'available')}
                  className={`badge cursor-pointer transition-colors ${availFilter === 'available' ? 'bg-accent/20 text-accent' : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'}`}
                >
                  <Eye className="w-3 h-3" /> Available
                </button>
                <button
                  onClick={() => setIsScripted(!isScripted)}
                  className={`badge cursor-pointer transition-colors ${isScripted ? 'bg-cyber-teal/20 text-cyber-teal' : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'}`}
                >
                  <Scroll className="w-3 h-3" /> Scripted
                </button>
                <button
                  onClick={() => setListFilter(listFilter === 'favourite' ? null : 'favourite')}
                  className={`badge cursor-pointer transition-colors ${listFilter === 'favourite' ? 'bg-cyber-pink/20 text-cyber-pink' : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'}`}
                >
                  <Heart className="w-3 h-3" /> Favourite
                </button>
                <button
                  onClick={() => setListFilter(listFilter === 'wishlist' ? null : 'wishlist')}
                  className={`badge cursor-pointer transition-colors ${listFilter === 'wishlist' ? 'bg-cyber-amber/20 text-cyber-amber' : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'}`}
                >
                  <Bookmark className="w-3 h-3" /> Wishlist
                </button>
                <button
                  onClick={() => setListFilter(listFilter === 'watchlist' ? null : 'watchlist')}
                  className={`badge cursor-pointer transition-colors ${listFilter === 'watchlist' ? 'bg-cyber-blue/20 text-cyber-blue' : 'bg-surface-700/50 text-surface-400 hover:text-surface-200'}`}
                >
                  <Eye className="w-3 h-3" /> Watchlist
                </button>
              </div>
            </div>

            {/* Rating filter */}
            <div className="space-y-2">
              <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">Min rating</label>
              <div className="flex items-center gap-1">
                {Array.from({ length: 5 }, (_, i) => {
                  const val = i + 1
                  return (
                    <button
                      key={i}
                      onClick={() => setRatingFilter(ratingFilter === val ? 0 : val)}
                      className="p-0.5"
                    >
                      <Star className={`w-4 h-4 transition-colors ${val <= ratingFilter
                        ? 'fill-cyber-amber text-cyber-amber'
                        : 'text-surface-600 hover:text-surface-400'
                      }`} />
                    </button>
                  )
                })}
                {ratingFilter > 0 && (
                  <span className="text-xs text-surface-400 ml-1">& up</span>
                )}
              </div>
            </div>

            {/* Sites */}
            {filterOptions?.sites && filterOptions.sites.length > 0 && (
              <div className="space-y-2">
                <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                  Sites {selectedSites.length > 0 && `(${selectedSites.length})`}
                </label>
                <select
                  className="input-dark text-sm w-full"
                  value=""
                  onChange={(e) => {
                    if (e.target.value && !selectedSites.includes(e.target.value)) {
                      setSelectedSites([...selectedSites, e.target.value])
                    }
                  }}
                >
                  <option value="">Select site...</option>
                  {filterOptions.sites.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
                {selectedSites.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {selectedSites.map((s) => (
                      <span key={s} className="badge bg-accent/15 text-accent text-xs">
                        {s}
                        <button onClick={() => setSelectedSites(selectedSites.filter(x => x !== s))}>
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Tags */}
            {filterOptions?.tags && filterOptions.tags.length > 0 && (
              <div className="space-y-2">
                <label className="text-xs text-surface-500 uppercase tracking-wider font-medium">
                  Tags {selectedTags.length > 0 && `(${selectedTags.length})`}
                </label>
                <select
                  className="input-dark text-sm w-full"
                  value=""
                  onChange={(e) => {
                    if (e.target.value && !selectedTags.includes(e.target.value)) {
                      setSelectedTags([...selectedTags, e.target.value])
                    }
                  }}
                >
                  <option value="">Select tag...</option>
                  {filterOptions.tags.map((t) => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
                {selectedTags.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {selectedTags.map((t) => (
                      <span key={t} className="badge bg-cyber-blue/15 text-cyber-blue text-xs">
                        {t}
                        <button onClick={() => setSelectedTags(selectedTags.filter(x => x !== t))}>
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </motion.div>
      )}

      {/* Scene grid */}
      {loading && scenes.length === 0 ? (
        <div className="flex items-center justify-center py-32">
          <div className="w-8 h-8 border-2 border-accent border-t-transparent rounded-full animate-spin" />
        </div>
      ) : scenes.length === 0 ? (
        <div className="text-center py-32">
          <p className="text-surface-400 text-lg">No scenes found</p>
          <p className="text-surface-600 text-sm mt-1">Try adjusting your filters</p>
        </div>
      ) : (
        <>
          <div className={`grid ${CARD_SIZES[cardSize].cols} gap-4`}>
            {scenes.map((scene, i) => (
              <motion.div
                key={scene.id}
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: Math.min(i * 0.03, 0.5) }}
              >
                <SceneCard scene={scene} />
              </motion.div>
            ))}
          </div>

          {/* Pagination */}
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
                  if (totalPages <= 7) {
                    pageNum = i + 1
                  } else if (page <= 4) {
                    pageNum = i + 1
                  } else if (page >= totalPages - 3) {
                    pageNum = totalPages - 6 + i
                  } else {
                    pageNum = page - 3 + i
                  }
                  return (
                    <button
                      key={pageNum}
                      onClick={() => setPage(pageNum)}
                      className={`w-9 h-9 rounded-lg text-sm font-medium transition-colors
                        ${page === pageNum
                          ? 'bg-accent text-white'
                          : 'text-surface-400 hover:bg-surface-700/50 hover:text-surface-200'
                        }`}
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
