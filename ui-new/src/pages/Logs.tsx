import { useEffect, useState, useRef } from 'react'
import { motion } from 'framer-motion'
import {
  Terminal, Pause, Play, Trash2, ArrowDown, Search,
  AlertTriangle, Info, Bug, XCircle,
} from 'lucide-react'

interface LogEntry {
  time: string
  level: string
  msg: string
  [key: string]: any
}

const LEVEL_STYLES: Record<string, { bg: string; text: string; icon: any }> = {
  info: { bg: 'bg-cyber-blue/10', text: 'text-cyber-blue', icon: Info },
  warning: { bg: 'bg-cyber-amber/10', text: 'text-cyber-amber', icon: AlertTriangle },
  error: { bg: 'bg-cyber-red/10', text: 'text-cyber-red', icon: XCircle },
  debug: { bg: 'bg-surface-700/30', text: 'text-surface-400', icon: Bug },
}

export default function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [paused, setPaused] = useState(false)
  const [search, setSearch] = useState('')
  const [levelFilter, setLevelFilter] = useState<string>('all')
  const [autoScroll, setAutoScroll] = useState(true)
  const containerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${protocol}//${window.location.host}/api/log`)
    wsRef.current = ws

    ws.onmessage = (event) => {
      try {
        const entry = JSON.parse(event.data) as LogEntry
        setLogs((prev) => {
          const next = [...prev, entry]
          // Keep last 1000 entries
          return next.length > 1000 ? next.slice(-1000) : next
        })
      } catch {}
    }

    ws.onerror = () => {}
    ws.onclose = () => {}

    return () => {
      ws.close()
    }
  }, [])

  // Auto-scroll
  useEffect(() => {
    if (autoScroll && !paused && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight
    }
  }, [logs, autoScroll, paused])

  const filtered = logs.filter((log) => {
    if (levelFilter !== 'all' && log.level !== levelFilter) return false
    if (search && !log.msg.toLowerCase().includes(search.toLowerCase())) return false
    return true
  })

  const displayLogs = paused ? filtered : filtered

  return (
    <div className="space-y-4 h-full flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold text-surface-100">Logs</h1>
          <span className="text-sm text-surface-500">{logs.length} entries</span>
          {!paused && (
            <span className="flex items-center gap-1.5 text-xs text-cyber-teal">
              <span className="w-1.5 h-1.5 rounded-full bg-cyber-teal animate-pulse" />
              Live
            </span>
          )}
        </div>

        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="w-4 h-4 text-surface-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Filter logs..."
              className="input-dark pl-10 w-48 text-sm"
            />
          </div>

          <div className="flex items-center border border-surface-700 rounded-lg overflow-hidden">
            {['all', 'info', 'warning', 'error', 'debug'].map((level) => (
              <button
                key={level}
                onClick={() => setLevelFilter(level)}
                className={`px-2.5 py-1.5 text-xs font-medium capitalize transition-colors
                  ${levelFilter === level ? 'bg-accent text-white' : 'text-surface-400 hover:text-surface-200 hover:bg-surface-700/50'}`}
              >
                {level}
              </button>
            ))}
          </div>

          <button
            onClick={() => setPaused(!paused)}
            className={`btn-ghost text-sm flex items-center gap-1.5 ${paused ? 'text-cyber-amber' : ''}`}
          >
            {paused ? <Play className="w-4 h-4" /> : <Pause className="w-4 h-4" />}
            {paused ? 'Resume' : 'Pause'}
          </button>

          <button
            onClick={() => setAutoScroll(!autoScroll)}
            className={`btn-ghost text-sm flex items-center gap-1.5 ${autoScroll ? 'text-accent' : ''}`}
          >
            <ArrowDown className="w-4 h-4" />
          </button>

          <button
            onClick={() => setLogs([])}
            className="btn-ghost text-sm flex items-center gap-1.5 text-surface-400 hover:text-cyber-red"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Log viewer */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        className="glass rounded-xl flex-1 min-h-0 overflow-hidden flex flex-col"
      >
        <div
          ref={containerRef}
          className="flex-1 overflow-y-auto p-1 font-mono text-xs"
        >
          {displayLogs.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-surface-500">
              <Terminal className="w-8 h-8 mb-2 opacity-50" />
              <p className="text-sm">
                {logs.length === 0 ? 'Waiting for log events...' : 'No matching logs'}
              </p>
            </div>
          ) : (
            displayLogs.map((log, i) => {
              const style = LEVEL_STYLES[log.level] || LEVEL_STYLES.info
              const Icon = style.icon
              return (
                <div
                  key={i}
                  className={`flex items-start gap-2 px-3 py-1.5 rounded hover:bg-surface-700/20 transition-colors ${
                    log.level === 'error' ? 'bg-cyber-red/5' : ''
                  }`}
                >
                  <span className="text-surface-600 flex-shrink-0 w-20 pt-0.5">
                    {log.time ? new Date(log.time).toLocaleTimeString() : '--:--:--'}
                  </span>
                  <span className={`flex-shrink-0 w-5 pt-0.5 ${style.text}`}>
                    <Icon className="w-3.5 h-3.5" />
                  </span>
                  <span className={`flex-shrink-0 w-14 uppercase text-[10px] font-semibold pt-0.5 ${style.text}`}>
                    {log.level}
                  </span>
                  <span className="text-surface-300 flex-1 break-all leading-relaxed">{log.msg}</span>
                </div>
              )
            })
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between px-4 py-2 border-t border-surface-700/50 text-[11px] text-surface-500">
          <span>{displayLogs.length} / {logs.length} entries</span>
          <span>Max 1000 entries kept in memory</span>
        </div>
      </motion.div>
    </div>
  )
}
