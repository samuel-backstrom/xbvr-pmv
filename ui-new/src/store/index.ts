import { create } from 'zustand'
import type { Scene, Actor } from '../api/client'

interface AppState {
  // Command palette
  commandPaletteOpen: boolean
  openCommandPalette: () => void
  closeCommandPalette: () => void

  // Scene detail modal
  selectedScene: Scene | null
  openSceneDetail: (scene: Scene) => void
  closeSceneDetail: () => void

  // Actor detail modal
  selectedActor: Actor | null
  openActorDetail: (actor: Actor) => void
  closeActorDetail: () => void

  // Sidebar
  sidebarCollapsed: boolean
  toggleSidebar: () => void
}

export const useAppStore = create<AppState>((set) => ({
  commandPaletteOpen: false,
  openCommandPalette: () => set({ commandPaletteOpen: true }),
  closeCommandPalette: () => set({ commandPaletteOpen: false }),

  selectedScene: null,
  openSceneDetail: (scene) => set({ selectedScene: scene }),
  closeSceneDetail: () => set({ selectedScene: null }),

  selectedActor: null,
  openActorDetail: (actor) => set({ selectedActor: actor }),
  closeActorDetail: () => set({ selectedActor: null }),

  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),
}))
