import { useState, useEffect, useCallback } from 'react'
import { RefreshCw } from 'lucide-react'
import { type GmailStatus, type WhatsAppStatus, gmailStatus, syncGmail, whatsappStatus, syncWhatsApp } from '../api'
import { cn } from '../lib/utils'

interface Props {
  onSynced: () => void
}

function timeAgo(iso: string): string {
  const diff = Math.floor((Date.now() - new Date(iso).getTime()) / 1000)
  if (diff < 60) return 'just now'
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
  return `${Math.floor(diff / 86400)}d ago`
}

export default function SyncButton({ onSynced }: Props) {
  const [gmail, setGmail] = useState<GmailStatus | null>(null)
  const [wa, setWa] = useState<WhatsAppStatus | null>(null)
  const [syncing, setSyncing] = useState(false)
  const [result, setResult] = useState<string | null>(null)

  const loadStatus = useCallback(async () => {
    const [g, w] = await Promise.all([
      gmailStatus().catch(() => null),
      whatsappStatus().catch(() => null),
    ])
    setGmail(g)
    setWa(w)
  }, [])

  useEffect(() => { loadStatus() }, [loadStatus])

  const gmailConnected = gmail?.connected
  const waConnected = wa?.status === 'connected'

  if (!gmailConnected && !waConnected) return null

  async function handleSync() {
    setSyncing(true)
    setResult(null)
    try {
      const jobs = []
      if (gmailConnected) jobs.push(syncGmail())
      if (waConnected) jobs.push(syncWhatsApp())
      const results = await Promise.all(jobs)
      const total = results.reduce((sum, r) => sum + r.tasks_created, 0)
      setResult(`${total} task${total !== 1 ? 's' : ''} created`)
      await loadStatus()
      onSynced()
    } catch (e) {
      setResult(e instanceof Error ? e.message : 'Sync failed')
    } finally {
      setSyncing(false)
    }
  }

  const lastSynced = gmail?.last_synced_at ?? wa?.last_synced_at

  return (
    <div className="flex items-center gap-2">
      {result && (
        <span className="text-xs text-slate-500 dark:text-slate-400">{result}</span>
      )}
      {!result && lastSynced && (
        <span className="text-xs text-slate-400 dark:text-slate-500">
          Synced {timeAgo(lastSynced)}
        </span>
      )}
      <button
        onClick={handleSync}
        disabled={syncing}
        className={cn(
          'flex items-center gap-1.5 px-4 py-2 text-sm rounded-md bg-slate-900 text-white hover:bg-slate-700 transition-colors disabled:opacity-50',
        )}
      >
        <RefreshCw size={13} className={syncing ? 'animate-spin' : ''} />
        {syncing ? 'Syncing…' : 'Sync'}
      </button>
    </div>
  )
}
