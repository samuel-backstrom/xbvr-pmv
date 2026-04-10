import { Outlet } from 'react-router-dom'
import Sidebar from './Sidebar'
import CommandPalette from './CommandPalette'
import SceneDetailModal from './SceneDetailModal'
import ActorDetailModal from './ActorDetailModal'
import { useAppStore } from '../store'

export default function Layout() {
  const collapsed = useAppStore((s) => s.sidebarCollapsed)

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar />
      <main
        className={`flex-1 overflow-y-auto transition-all duration-300
          ${collapsed ? 'ml-16' : 'ml-56'}`}
      >
        <div className="p-6 min-h-screen">
          <Outlet />
        </div>
      </main>
      <CommandPalette />
      <SceneDetailModal />
      <ActorDetailModal />
    </div>
  )
}
