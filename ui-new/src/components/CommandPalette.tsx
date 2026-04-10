import { useEffect, useState, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { Search, Film, X } from 'lucide-react'
import { useAppStore } from '../store'
import { searchScenes, imageUrl, type Scene } from '../api/client'

export default function CommandPalette() {
  const open = useAppStore((s) => s.commandPaletteOpen)
  const close = useAppStore((s) => s.closeCommandPalette)
  const openDetail = useAppStore((s) => s.openSceneDetail)
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Scene[]>([])
  const [loading, setLoading] = useState(false)
  const [selected, setSelected] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  // Keyboard shortcut
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        useAppStore.getState().openCommandPalette()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  // Focus input when opened
  useEffect(() => {
    if (open) {
      setTimeout(() => inputRef.current?.focus(), 50)
      setQuery('')
      setResults([])
      setSelected(0)
    }
  }, [open])

  // Debounced search
  useEffect(() => {
    if (!query.trim()) { setResults([]); return }
    clearTimeout(timerRef.current)
    timerRef.current = setTimeout(async () => {
      setLoading(true)
      try {
        const data = await searchScenes(query)
        setResults(data.scenes || [])
        setSelected(0)
      } catch { setResults([]) }
      setLoading(false)
    }, 250)
    return () => clearTimeout(timerRef.current)
  }, [query])

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') close()
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelected((s) => Math.min(s + 1, results.length - 1))
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelected((s) => Math.max(s - 1, 0))
    }
    if (e.key === 'Enter' && results[selected]) {
      openDetail(results[selected])
      close()
    }
  }

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50"
            onClick={close}
          />
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: -20 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: -20 }}
            transition={{ duration: 0.15 }}
            className="fixed top-[15%] left-1/2 -translate-x-1/2 w-full max-w-2xl z-50"
          >
            <div className="glass-strong rounded-2xl shadow-2xl shadow-black/40 overflow-hidden">
              {/* Search input */}
              <div className="flex items-center gap-3 px-5 py-4 border-b border-surface-700/50">
                <Search className={`w-5 h-5 flex-shrink-0 ${loading ? 'text-accent animate-pulse' : 'text-surface-400'}`} />
                <input
                  ref={inputRef}
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  onKeyDown={handleKey}
                  placeholder="Search scenes, actors, sites..."
                  className="flex-1 bg-transparent text-surface-100 text-lg placeholder-surface-500
                    outline-none"
                />
                <button onClick={close} className="text-surface-500 hover:text-surface-300">
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Results */}
              {results.length > 0 && (
                <ul className="max-h-96 overflow-y-auto py-2">
                  {results.map((scene, i) => (
                    <li
                      key={scene.id}
                      onClick={() => { openDetail(scene); close() }}
                      onMouseEnter={() => setSelected(i)}
                      className={`flex items-center gap-4 px-5 py-3 cursor-pointer transition-colors
                        ${i === selected ? 'bg-accent/10' : 'hover:bg-surface-700/30'}`}
                    >
                      <img
                        src={imageUrl(scene.cover_url, '120x')}
                        className="w-16 h-10 object-cover rounded-md flex-shrink-0"
                        alt=""
                        onError={(e) => { (e.target as HTMLImageElement).src = '/ui/images/blank.png' }}
                      />
                      <div className="flex-1 min-w-0">
                        <div className="text-sm font-medium text-surface-100 truncate">{scene.title}</div>
                        <div className="text-xs text-surface-400 mt-0.5">
                          {scene.site}
                          {scene.cast?.length > 0 && ` · ${scene.cast.map(c => c.name).join(', ')}`}
                        </div>
                      </div>
                      {scene.duration > 0 && (
                        <span className="text-xs text-surface-500 flex-shrink-0">{scene.duration}m</span>
                      )}
                    </li>
                  ))}
                </ul>
              )}

              {/* Empty state */}
              {query && !loading && results.length === 0 && (
                <div className="py-12 text-center text-surface-500">
                  <Film className="w-8 h-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">No scenes found</p>
                </div>
              )}

              {/* Footer */}
              <div className="flex items-center gap-4 px-5 py-2.5 border-t border-surface-700/50 text-[11px] text-surface-500">
                <span><kbd className="font-mono bg-surface-700 rounded px-1 py-0.5">↑↓</kbd> navigate</span>
                <span><kbd className="font-mono bg-surface-700 rounded px-1 py-0.5">↵</kbd> open</span>
                <span><kbd className="font-mono bg-surface-700 rounded px-1 py-0.5">esc</kbd> close</span>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
