import { create } from 'zustand'
import type { Scene, Actor } from '../api/client'

export interface LogEntry {
  timestamp: string
  level: string
  message: string
  data?: Record<string, any>
  [key: string]: any
}

type LogConnectionStatus = 'connecting' | 'connected' | 'disconnected'
type TaskKind = 'scrape' | 'rescan'

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

  // List invalidation
  sceneListRevision: number
  actorListRevision: number
  bumpSceneListRevision: () => void
  bumpActorListRevision: () => void

  // Sidebar
  sidebarCollapsed: boolean
  toggleSidebar: () => void

  // Live logs
  serviceLogs: LogEntry[]
  logConnectionStatus: LogConnectionStatus
  addServiceLog: (entry: LogEntry) => void
  clearServiceLogs: () => void
  setLogConnectionStatus: (status: LogConnectionStatus) => void
  logsPaused: boolean
  pausedServiceLogs: LogEntry[] | null
  logsSearch: string
  logsLevelFilter: string
  logsAutoScroll: boolean
  setLogsPaused: (paused: boolean) => void
  setPausedServiceLogs: (logs: LogEntry[] | null) => void
  setLogsSearch: (search: string) => void
  setLogsLevelFilter: (level: string) => void
  setLogsAutoScroll: (enabled: boolean) => void

  // Background task state
  lockScrape: boolean
  lockRescan: boolean
  lastScrapeMessage: LogEntry | null
  lastRescanMessage: LogEntry | null
  runningScrapers: string[]
  storageRevision: number
  setTaskLock: (task: TaskKind, locked: boolean) => void
  setLastTaskMessage: (task: TaskKind, entry: LogEntry) => void
  addRunningScraper: (scraperId: string) => void
  removeRunningScraper: (scraperId: string) => void
  clearRunningScrapers: () => void
  bumpStorageRevision: () => void
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

  sceneListRevision: 0,
  actorListRevision: 0,
  bumpSceneListRevision: () =>
    set((state) => ({ sceneListRevision: state.sceneListRevision + 1 })),
  bumpActorListRevision: () =>
    set((state) => ({ actorListRevision: state.actorListRevision + 1 })),

  sidebarCollapsed: false,
  toggleSidebar: () => set((s) => ({ sidebarCollapsed: !s.sidebarCollapsed })),

  serviceLogs: [],
  logConnectionStatus: 'connecting',
  addServiceLog: (entry) =>
    set((state) => {
      const serviceLogs = [...state.serviceLogs, entry]
      return {
        serviceLogs: serviceLogs.length > 1000 ? serviceLogs.slice(-1000) : serviceLogs,
      }
    }),
  clearServiceLogs: () => set({ serviceLogs: [] }),
  setLogConnectionStatus: (status) => set({ logConnectionStatus: status }),
  logsPaused: false,
  pausedServiceLogs: null,
  logsSearch: '',
  logsLevelFilter: 'all',
  logsAutoScroll: true,
  setLogsPaused: (paused) => set({ logsPaused: paused }),
  setPausedServiceLogs: (logs) => set({ pausedServiceLogs: logs }),
  setLogsSearch: (search) => set({ logsSearch: search }),
  setLogsLevelFilter: (level) => set({ logsLevelFilter: level }),
  setLogsAutoScroll: (enabled) => set({ logsAutoScroll: enabled }),

  lockScrape: false,
  lockRescan: false,
  lastScrapeMessage: null,
  lastRescanMessage: null,
  runningScrapers: [],
  storageRevision: 0,
  setTaskLock: (task, locked) =>
    set(task === 'scrape' ? { lockScrape: locked } : { lockRescan: locked }),
  setLastTaskMessage: (task, entry) =>
    set(task === 'scrape' ? { lastScrapeMessage: entry } : { lastRescanMessage: entry }),
  addRunningScraper: (scraperId) =>
    set((state) => ({
      runningScrapers: state.runningScrapers.includes(scraperId)
        ? state.runningScrapers
        : [...state.runningScrapers, scraperId],
    })),
  removeRunningScraper: (scraperId) =>
    set((state) => ({
      runningScrapers: state.runningScrapers.filter((id) => id !== scraperId),
    })),
  clearRunningScrapers: () => set({ runningScrapers: [] }),
  bumpStorageRevision: () => set((state) => ({ storageRevision: state.storageRevision + 1 })),
}))
