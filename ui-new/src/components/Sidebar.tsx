import { NavLink } from 'react-router-dom'
import {
  Film, Users, FolderOpen, Settings, Search,
  PanelLeftClose, PanelLeftOpen, Activity,
} from 'lucide-react'
import { useAppStore } from '../store'

const navItems = [
  { to: '/', icon: Film, label: 'Scenes' },
  { to: '/actors', icon: Users, label: 'Actors' },
  { to: '/files', icon: FolderOpen, label: 'Files' },
  { to: '/settings', icon: Settings, label: 'Settings' },
  { to: '/logs', icon: Activity, label: 'Logs' },
]

export default function Sidebar() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed)
  const toggle = useAppStore((s) => s.toggleSidebar)
  const openCmd = useAppStore((s) => s.openCommandPalette)

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
