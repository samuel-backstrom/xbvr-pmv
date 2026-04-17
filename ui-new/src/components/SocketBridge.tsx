import { useEffect } from 'react'
import { Wampy } from 'wampy'
import { useAppStore } from '../store'

export default function SocketBridge() {
  const addServiceLog = useAppStore((s) => s.addServiceLog)
  const addRunningScraper = useAppStore((s) => s.addRunningScraper)
  const bumpStorageRevision = useAppStore((s) => s.bumpStorageRevision)
  const clearRunningScrapers = useAppStore((s) => s.clearRunningScrapers)
  const removeRunningScraper = useAppStore((s) => s.removeRunningScraper)
  const setLogConnectionStatus = useAppStore((s) => s.setLogConnectionStatus)
  const setLastTaskMessage = useAppStore((s) => s.setLastTaskMessage)
  const setTaskLock = useAppStore((s) => s.setTaskLock)

  useEffect(() => {
    setLogConnectionStatus('connecting')

    const ws = new Wampy('/ws/', {
      realm: 'default',
      onConnect: () => {
        setLogConnectionStatus('connected')
      },
      onClose: () => {
        setLogConnectionStatus('disconnected')
      },
      onError: () => {
        setLogConnectionStatus('disconnected')
      },
      onReconnect: () => {
        setLogConnectionStatus('connecting')
      },
      onReconnectSuccess: () => {
        setLogConnectionStatus('connected')
      },
    })

    ws.subscribe('service.log', ({ argsDict }) => {
      if (!argsDict) return

      const entry = {
        level: String(argsDict.level || 'info'),
        message: String(argsDict.message || ''),
        timestamp: String(argsDict.timestamp || ''),
        data: argsDict.data || {},
      }

      addServiceLog(entry)

      const task = String(entry.data?.task || '')
      if (task === 'scrape') {
        setLastTaskMessage('scrape', entry)
      }
      if (task === 'rescan') {
        setLastTaskMessage('rescan', entry)
      }
      if (task === 'scraperProgress') {
        if (entry.message === 'DONE') {
          clearRunningScrapers()
          return
        }

        const scraperId = String(entry.data?.scraperID || '')
        if (!scraperId) return

        if (entry.data?.started) addRunningScraper(scraperId)
        if (entry.data?.completed) removeRunningScraper(scraperId)
      }
    })

    ws.subscribe('lock.change', ({ argsDict }) => {
      if (!argsDict) return

      const name = String(argsDict.name || '')
      const locked = Boolean(argsDict.locked)
      if (name === 'scrape') {
        setTaskLock('scrape', locked)
      }
      if (name === 'rescan') {
        setTaskLock('rescan', locked)
      }
    })

    ws.subscribe('state.change.optionsStorage', () => {
      bumpStorageRevision()
    })

    return () => {
      setLogConnectionStatus('disconnected')
      ws.disconnect()
    }
  }, [
    addRunningScraper,
    addServiceLog,
    bumpStorageRevision,
    clearRunningScrapers,
    removeRunningScraper,
    setLastTaskMessage,
    setLogConnectionStatus,
    setTaskLock,
  ])

  return null
}
