import { NavLink } from 'react-router-dom'
import {
  Film, Users, FolderOpen, Settings, Search, Disc3,
  PanelLeftClose, PanelLeftOpen, Activity, Database, ScanSearch,
} from 'lucide-react'
import { useAppStore } from '../store'

const navItems = [
  { to: '/', icon: Film, label: 'Scenes' },
  { to: '/actors', icon: Users, label: 'Actors' },
  { to: '/files', icon: FolderOpen, label: 'Files' },
  { to: '/pmv', icon: Disc3, label: 'PMV' },
  { to: '/settings', icon: Settings, label: 'Settings' },
  { to: '/logs', icon: Activity, label: 'Logs' },
]

export default function Sidebar() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed)
  const toggle = useAppStore((s) => s.toggleSidebar)
  const openCmd = useAppStore((s) => s.openCommandPalette)
  const lockRescan = useAppStore((s) => s.lockRescan)
  const lockScrape = useAppStore((s) => s.lockScrape)
  const lastRescanMessage = useAppStore((s) => s.lastRescanMessage)
  const lastScrapeMessage = useAppStore((s) => s.lastScrapeMessage)
  const runningScrapers = useAppStore((s) => s.runningScrapers)

  const statusItems = [
    {
      key: 'rescan',
      label: 'Files',
      icon: ScanSearch,
      locked: lockRescan,
      message: lastRescanMessage?.message || 'Idle',
      tone: lockRescan ? 'text-cyber-amber' : 'text-surface-400',
    },
    {
      key: 'scrape',
      label: 'Data',
      icon: Database,
      locked: lockScrape,
      message: lastScrapeMessage?.message || 'Idle',
      tone: lockScrape ? 'text-cyber-teal' : 'text-surface-400',
    },
  ]

  return (
    <aside
      className={`fixed left-0 top-0 h-screen z-40 flex flex-col
        bg-surface-900/95 backdrop-blur-xl border-r border-surface-700/50
        transition-all duration-300 ease-out
        ${collapsed ? 'w-16' : 'w-56'}`}
    >
      {/* Logo */}
      <div className={`flex items-center h-14 px-4 border-b border-surface-700/50
        ${collapsed ? 'justify-center' : 'gap-3'}`}>
        <div className="w-8 h-8 rounded-lg gradient-accent flex items-center justify-center flex-shrink-0">
          <Film className="w-4 h-4 text-white" />
        </div>
        {!collapsed && (
          <span className="font-bold text-lg tracking-tight">
            XB<span className="gradient-text">VR</span>
          </span>
        )}
      </div>

      {/* Search trigger */}
      <div className="px-3 pt-4 pb-2">
        <button
          onClick={openCmd}
          className={`w-full flex items-center gap-2 rounded-lg text-surface-400
            hover:text-surface-200 hover:bg-surface-700/50 transition-colors
            ${collapsed ? 'justify-center p-2' : 'px-3 py-2'}`}
        >
          <Search className="w-4 h-4 flex-shrink-0" />
          {!collapsed && (
            <>
              <span className="text-sm">Search...</span>
              <kbd className="ml-auto text-[10px] bg-surface-700 rounded px-1.5 py-0.5 font-mono text-surface-400">
                ⌘K
              </kbd>
            </>
          )}
        </button>
      </div>

      {/* Nav items */}
      <nav className="flex-1 px-3 py-2 space-y-1">
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-lg transition-all duration-150
              ${collapsed ? 'justify-center p-2.5' : 'px-3 py-2.5'}
              ${isActive
                ? 'bg-accent/15 text-accent-hover border border-accent/20'
                : 'text-surface-400 hover:text-surface-100 hover:bg-surface-700/40 border border-transparent'
              }`
            }
          >
            <Icon className="w-[18px] h-[18px] flex-shrink-0" />
            {!collapsed && <span className="text-sm font-medium">{label}</span>}
          </NavLink>
        ))}
      </nav>

      {!collapsed && (
        <div className="px-3 pb-3">
          <div className="rounded-xl border border-surface-700/40 bg-surface-800/30 p-3 space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-[11px] uppercase tracking-wider text-surface-500 font-medium">Background</span>
              {runningScrapers.length > 0 && (
                <span className="text-[10px] text-cyber-teal font-medium">
                  {runningScrapers.length} scraper{runningScrapers.length === 1 ? '' : 's'}
                </span>
              )}
            </div>
            {statusItems.map(({ key, label, icon: Icon, locked, message, tone }) => (
              <div key={key} className="rounded-lg bg-surface-900/40 px-2.5 py-2 border border-surface-700/30">
                <div className="flex items-center gap-2">
                  <Icon className={`w-3.5 h-3.5 ${tone} ${locked ? 'animate-pulse' : ''}`} />
                  <span className={`text-xs font-medium ${tone}`}>{label}</span>
                </div>
                <div className="mt-1 text-[11px] text-surface-500 line-clamp-2">
                  {message}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Collapse toggle */}
      <div className="px-3 py-3 border-t border-surface-700/50">
        <button
          onClick={toggle}
          className={`w-full flex items-center gap-2 rounded-lg text-surface-500
            hover:text-surface-200 hover:bg-surface-700/50 transition-colors
            ${collapsed ? 'justify-center p-2' : 'px-3 py-2'}`}
        >
          {collapsed
            ? <PanelLeftOpen className="w-4 h-4" />
            : <PanelLeftClose className="w-4 h-4" />
          }
          {!collapsed && <span className="text-sm">Collapse</span>}
        </button>
      </div>
    </aside>
  )
}
