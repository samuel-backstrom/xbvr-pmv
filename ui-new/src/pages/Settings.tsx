import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import {
  Settings as SettingsIcon, FolderOpen, RefreshCw, Globe,
  Database, Palette, Monitor, Volume2, Shield,
} from 'lucide-react'

interface AppConfig {
  server: { port: number; bindAddress: string }
  library: { path: string[] }
  [key: string]: any
}

export default function Settings() {
  const [version, setVersion] = useState('')
  const [activeTab, setActiveTab] = useState('general')

  useEffect(() => {
    fetch('/api/options/state')
      .then(r => r.json())
      .then(d => setVersion(d.currentState?.server?.version || ''))
      .catch(() => {})
  }, [])

  const tabs = [
    { id: 'general', label: 'General', icon: SettingsIcon },
    { id: 'folders', label: 'Folders', icon: FolderOpen },
    { id: 'sites', label: 'Sites', icon: Globe },
    { id: 'cache', label: 'Cache', icon: Database },
    { id: 'interface', label: 'Interface', icon: Monitor },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <h1 className="text-xl font-bold text-surface-100">Settings</h1>
        {version && (
          <span className="badge bg-surface-700/50 text-surface-400 text-xs">v{version}</span>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Sidebar tabs */}
        <div className="lg:col-span-1">
          <nav className="glass rounded-xl p-2 space-y-0.5">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors
                    ${activeTab === tab.id
                      ? 'bg-accent/15 text-accent'
                      : 'text-surface-400 hover:text-surface-200 hover:bg-surface-700/30'
                    }`}
                >
                  <Icon className="w-4 h-4" />
                  {tab.label}
                </button>
              )
            })}
          </nav>

          {/* Quick actions */}
          <div className="glass rounded-xl p-4 mt-4 space-y-3">
            <h3 className="text-xs text-surface-500 uppercase tracking-wider font-medium">Quick actions</h3>
            <button
              onClick={() => fetch('/api/scan/all', { method: 'POST' })}
              className="btn-primary w-full text-sm flex items-center justify-center gap-2"
            >
              <RefreshCw className="w-4 h-4" /> Rescan library
            </button>
            <button
              onClick={() => fetch('/api/task/scrape', { method: 'POST' })}
              className="btn-ghost w-full text-sm flex items-center justify-center gap-2"
            >
              <Globe className="w-4 h-4" /> Scrape sites
            </button>
          </div>
        </div>

        {/* Content */}
        <div className="lg:col-span-3">
          <motion.div
            key={activeTab}
            initial={{ opacity: 0, x: 10 }}
            animate={{ opacity: 1, x: 0 }}
            className="glass rounded-xl p-6"
          >
            {activeTab === 'general' && (
              <div className="space-y-6">
                <div>
                  <h3 className="text-lg font-semibold text-surface-100 mb-4">General settings</h3>
                  <p className="text-sm text-surface-400 leading-relaxed">
                    Configure your XBVR instance. Settings changes are applied through the main XBVR
                    configuration interface. This page provides quick access to common operations.
                  </p>
                </div>

                <div className="space-y-4">
                  <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                    <div className="flex items-center gap-3">
                      <Database className="w-5 h-5 text-accent" />
                      <div>
                        <div className="text-sm font-medium text-surface-200">Database</div>
                        <div className="text-xs text-surface-500">Manage your database</div>
                      </div>
                    </div>
                    <button className="btn-ghost text-xs">Optimize</button>
                  </div>

                  <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                    <div className="flex items-center gap-3">
                      <Shield className="w-5 h-5 text-cyber-teal" />
                      <div>
                        <div className="text-sm font-medium text-surface-200">DLNA</div>
                        <div className="text-xs text-surface-500">DLNA server status</div>
                      </div>
                    </div>
                    <span className="badge bg-cyber-teal/15 text-cyber-teal text-xs">Active</span>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'folders' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-surface-100">Library folders</h3>
                <p className="text-sm text-surface-400">
                  Manage your media library folders through the main XBVR settings interface.
                  Use the rescan button to re-index your library after making changes.
                </p>
                <div className="p-6 rounded-xl border border-dashed border-surface-600 text-center">
                  <FolderOpen className="w-8 h-8 text-surface-500 mx-auto mb-2" />
                  <p className="text-sm text-surface-400">
                    Folder management available in main settings
                  </p>
                </div>
              </div>
            )}

            {activeTab === 'sites' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-surface-100">Site scrapers</h3>
                <p className="text-sm text-surface-400">
                  Configure which sites to scrape for scene metadata. Access the full scraper
                  configuration through the main XBVR options panel.
                </p>
                <button
                  onClick={() => fetch('/api/task/scrape', { method: 'POST' })}
                  className="btn-primary text-sm flex items-center gap-2"
                >
                  <RefreshCw className="w-4 h-4" /> Run all scrapers
                </button>
              </div>
            )}

            {activeTab === 'cache' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-surface-100">Cache management</h3>
                <p className="text-sm text-surface-400">
                  Manage cached data including images and metadata.
                </p>
                <div className="space-y-3">
                  <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                    <div>
                      <div className="text-sm font-medium text-surface-200">Image cache</div>
                      <div className="text-xs text-surface-500">Cached scene and actor images</div>
                    </div>
                    <button
                      onClick={() => fetch('/api/task/clean-cache', { method: 'POST' })}
                      className="btn-ghost text-xs text-cyber-red hover:bg-cyber-red/10"
                    >
                      Clear
                    </button>
                  </div>
                  <div className="flex items-center justify-between p-4 rounded-xl bg-surface-800/50 border border-surface-700/30">
                    <div>
                      <div className="text-sm font-medium text-surface-200">Search index</div>
                      <div className="text-xs text-surface-500">Full-text search index</div>
                    </div>
                    <button
                      onClick={() => fetch('/api/task/index', { method: 'POST' })}
                      className="btn-ghost text-xs"
                    >
                      Rebuild
                    </button>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'interface' && (
              <div className="space-y-6">
                <h3 className="text-lg font-semibold text-surface-100">Interface</h3>
                <p className="text-sm text-surface-400">
                  This new UI is a modern alternative to the classic XBVR interface. Both UIs
                  connect to the same backend and share the same data.
                </p>
                <div className="p-4 rounded-xl bg-accent/5 border border-accent/20">
                  <div className="flex items-center gap-2 mb-2">
                    <Palette className="w-4 h-4 text-accent" />
                    <span className="text-sm font-medium text-accent">New UI</span>
                  </div>
                  <p className="text-xs text-surface-400 leading-relaxed">
                    Built with React, TypeScript, Tailwind CSS, and Framer Motion.
                    Features a modern glass-morphism design with a cinematic dark theme.
                  </p>
                </div>
              </div>
            )}
          </motion.div>
        </div>
      </div>
    </div>
  )
}
