import { useEffect, useState } from 'react'
import { AnimatePresence, motion } from 'framer-motion'
import { CheckCircle2, Edit3, PlusCircle, Tag, X } from 'lucide-react'
import { format, parseISO } from 'date-fns'
import {
  createCustomScene,
  matchFile,
  type Scene,
  type SceneFile,
} from '../api/client'
import { deriveSceneTitleFromFilename } from '../pages/files-utils.js'

export default function FileCreateSceneModal({
  file,
  onClose,
  onCreated,
}: {
  file: SceneFile | null
  onClose: () => void
  onCreated?: (scene: Scene, openDetail: boolean) => void | Promise<void>
}) {
  const [title, setTitle] = useState('')
  const [sceneId, setSceneId] = useState('')
  const [pmvhavenURL, setPmvHavenURL] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (!file) return
    setTitle(deriveSceneTitleFromFilename(file.filename))
    setSceneId('')
    setPmvHavenURL('')
  }, [file])

  useEffect(() => {
    if (!file) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [file, onClose])

  const createdAt = file ? format(parseISO(file.created_time), 'yyyy-MM-dd') : ''

  const handleCreate = async (openScene: boolean) => {
    if (!file || saving) return
    setSaving(true)
    try {
      const scene = await createCustomScene({
        title: title.trim(),
        id: sceneId.trim() || undefined,
        filename: file.filename,
        pmvhaven_url: pmvhavenURL.trim() || undefined,
      })
      await matchFile(file.id, scene.scene_id)
      onClose()
      if (onCreated) {
        await onCreated(scene, openScene)
      }
    } catch {
      // Let the user retry from the modal.
    } finally {
      setSaving(false)
    }
  }

  return (
    <AnimatePresence>
      {file && (
        <>
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm"
            onClick={onClose}
          />
          <motion.div
            initial={{ opacity: 0, y: 28, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 28, scale: 0.98 }}
            transition={{ type: 'spring', damping: 25, stiffness: 300 }}
            className="fixed inset-4 md:inset-8 z-50 glass-strong rounded-2xl overflow-hidden flex flex-col"
          >
            <div className="flex items-center justify-between gap-3 px-5 py-4 border-b border-surface-700/50">
              <div className="min-w-0">
                <div className="flex items-center gap-2 text-sm text-surface-400">
                  <PlusCircle className="w-4 h-4 text-cyber-teal" />
                  Create custom scene
                </div>
                <h2 className="text-base sm:text-lg font-semibold text-surface-100 truncate">
                  {file.filename}
                </h2>
              </div>
              <button
                onClick={onClose}
                className="p-2 rounded-lg hover:bg-surface-700/50 text-surface-400 hover:text-white transition-colors"
                aria-label="Close create scene dialog"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)] gap-0 flex-1 min-h-0">
              <div className="border-b lg:border-b-0 lg:border-r border-surface-700/40 p-5 space-y-4 overflow-y-auto">
                <div className="rounded-xl bg-surface-800/50 border border-surface-700/40 p-4 space-y-2">
                  <div className="text-xs uppercase tracking-wider text-surface-500 font-medium">File</div>
                  <div className="font-mono text-sm text-surface-100 break-all">{file.filename}</div>
                  <div className="font-mono text-[11px] text-surface-500 break-all">{file.path}</div>
                  <div className="flex flex-wrap gap-2 text-xs text-surface-400">
                    <span className="badge bg-surface-700/50">{file.video_width > 0 ? `${file.video_width}x${file.video_height}` : 'No resolution'}</span>
                    <span className="badge bg-surface-700/50">{createdAt}</span>
                  </div>
                </div>

                <div className="text-xs text-surface-500">
                  The new scene will be created, the file will be matched to it, and you can optionally open the scene details immediately.
                </div>
              </div>

              <div className="overflow-y-auto min-h-0 p-5 space-y-4">
                <label className="block">
                  <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
                    Title
                  </div>
                  <input
                    value={title}
                    onChange={(e) => setTitle(e.target.value)}
                    className="input-dark w-full"
                    placeholder="Scene title"
                  />
                </label>

                <label className="block">
                  <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
                    Scene ID
                  </div>
                  <div className="relative">
                    <Tag className="w-4 h-4 text-surface-500 absolute left-3 top-1/2 -translate-y-1/2" />
                    <input
                      value={sceneId}
                      onChange={(e) => setSceneId(e.target.value)}
                      className="input-dark w-full pl-10"
                      placeholder="Optional custom scene id"
                    />
                  </div>
                </label>

                <label className="block">
                  <div className="text-xs uppercase tracking-wider text-surface-500 mb-1.5 font-medium">
                    PMVHaven URL
                  </div>
                  <input
                    value={pmvhavenURL}
                    onChange={(e) => setPmvHavenURL(e.target.value)}
                    className="input-dark w-full"
                    placeholder="https://pmvhaven.com/video/..."
                  />
                </label>

                <div className="flex flex-wrap gap-2 pt-2">
                  <button
                    onClick={() => void handleCreate(false)}
                    disabled={saving || !title.trim()}
                    className="btn-primary flex items-center gap-1.5 disabled:opacity-60"
                  >
                    {saving ? <CheckCircle2 className="w-4 h-4 animate-pulse" /> : <PlusCircle className="w-4 h-4" />}
                    Create
                  </button>
                  <button
                    onClick={() => void handleCreate(true)}
                    disabled={saving || !title.trim()}
                    className="btn-ghost flex items-center gap-1.5 disabled:opacity-60"
                  >
                    <Edit3 className="w-4 h-4" />
                    Create & View
                  </button>
                </div>
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
