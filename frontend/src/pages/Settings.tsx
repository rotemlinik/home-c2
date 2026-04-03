import { useEffect, useState, useCallback, useRef } from 'react'
import {
  type Person, type GmailStatus, type WhatsAppStatus, type WhatsAppContact,
  listPeople, createPerson, deletePerson,
  gmailStatus, gmailDisconnect,
  whatsappStatus, whatsappConnect, whatsappDisconnect, whatsappQR,
  whatsappListContacts, whatsappAddContact, whatsappRemoveContact,
} from '../api'
import { Trash2, Plus, Moon, Sun, Mail, MessageCircle, CheckCircle, XCircle } from 'lucide-react'
import Modal from '../components/Modal'

interface Props {
  dark: boolean
  onDarkChange: (v: boolean) => void
}

export default function Settings({ dark, onDarkChange }: Props) {
  const [people, setPeople] = useState<Person[]>([])
  const [newName, setNewName] = useState('')
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [gmail, setGmail] = useState<GmailStatus | null>(null)
  const [confirmRemove, setConfirmRemove] = useState<{ id: number; name: string } | null>(null)
  const [confirmDisconnect, setConfirmDisconnect] = useState(false)

  // WhatsApp state
  const [wa, setWa] = useState<WhatsAppStatus | null>(null)
  const [waQR, setWaQR] = useState<string | null>(null)
  const [waContacts, setWaContacts] = useState<WhatsAppContact[]>([])
  const [newContactJid, setNewContactJid] = useState('')
  const [newContactLabel, setNewContactLabel] = useState('')
  const [waConnecting, setWaConnecting] = useState(false)
  const [confirmWaDisconnect, setConfirmWaDisconnect] = useState(false)
  const qrPollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopQRPoll = useCallback(() => {
    if (qrPollRef.current) {
      clearInterval(qrPollRef.current)
      qrPollRef.current = null
    }
  }, [])

  const startQRPoll = useCallback(() => {
    stopQRPoll()
    qrPollRef.current = setInterval(async () => {
      try {
        const [status, qr] = await Promise.all([whatsappStatus(), whatsappQR()])
        setWa(status)
        setWaQR(qr.qr_data)
        if (status.status === 'connected') {
          stopQRPoll()
          setWaConnecting(false)
          setWaQR(null)
        } else if (status.status === 'disconnected') {
          stopQRPoll()
          setWaConnecting(false)
          setWaQR(null)
        }
      } catch {
        stopQRPoll()
        setWaConnecting(false)
      }
    }, 2000)
  }, [stopQRPoll])

  useEffect(() => () => stopQRPoll(), [stopQRPoll])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [ppl, g, waStatus, contacts] = await Promise.all([
        listPeople(),
        gmailStatus().catch(() => null),
        whatsappStatus().catch(() => null),
        whatsappListContacts().catch(() => []),
      ])
      setPeople(ppl)
      setGmail(g)
      setWa(waStatus)
      setWaContacts(contacts)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    if (!newName.trim()) return
    setSaving(true)
    setError(null)
    try {
      await createPerson(newName.trim())
      setNewName('')
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to add person')
    } finally {
      setSaving(false)
    }
  }

  async function handleDelete(id: number) {
    setConfirmRemove(null)
    setError(null)
    try {
      await deletePerson(id)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to remove person')
    }
  }

  async function handleDisconnect() {
    setConfirmDisconnect(false)
    await gmailDisconnect()
    load()
  }

  async function handleWaConnect() {
    setWaConnecting(true)
    setWaQR(null)
    try {
      await whatsappConnect()
      startQRPoll()
    } catch {
      setWaConnecting(false)
    }
  }

  async function handleWaDisconnect() {
    setConfirmWaDisconnect(false)
    await whatsappDisconnect().catch(() => {})
    setWa(null)
    setWaQR(null)
    load()
  }

  async function handleAddContact(e: React.FormEvent) {
    e.preventDefault()
    if (!newContactJid.trim()) return
    try {
      const contact = await whatsappAddContact(newContactJid.trim(), newContactLabel.trim())
      setWaContacts(prev => [...prev, contact])
      setNewContactJid('')
      setNewContactLabel('')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to add contact')
    }
  }

  async function handleRemoveContact(id: number) {
    await whatsappRemoveContact(id).catch(() => {})
    setWaContacts(prev => prev.filter(c => c.id !== id))
  }

  return (
    <div className="space-y-8 max-w-md">
      <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Settings</h1>

      {/* Appearance */}
      <section>
        <h2 className="text-sm font-semibold text-slate-700 dark:text-slate-400 uppercase tracking-wide mb-3">Appearance</h2>
        <div className="flex items-center justify-between bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg px-4 py-3">
          <div className="flex items-center gap-2 text-sm text-slate-800 dark:text-slate-200">
            {dark ? <Moon size={15} /> : <Sun size={15} />}
            {dark ? 'Dark mode' : 'Light mode'}
          </div>
          <button
            onClick={() => onDarkChange(!dark)}
            aria-label={dark ? 'Switch to light mode' : 'Switch to dark mode'}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors cursor-pointer ${dark ? 'bg-slate-700' : 'bg-slate-200'}`}
          >
            <span className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${dark ? 'translate-x-6' : 'translate-x-1'}`} />
          </button>
        </div>
      </section>

      {/* Integrations */}
      <section>
        <h2 className="text-sm font-semibold text-slate-700 dark:text-slate-400 uppercase tracking-wide mb-3">Integrations</h2>
        <div className="space-y-2">

          {/* Gmail */}
          <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg px-4 py-3 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Mail size={16} className="text-slate-500 dark:text-slate-400" />
              <div>
                <p className="text-sm font-medium text-slate-800 dark:text-slate-200">Gmail</p>
                {gmail?.connected && gmail.email && (
                  <p className="text-xs text-slate-400 dark:text-slate-500">{gmail.email}</p>
                )}
              </div>
              {gmail?.connected
                ? <CheckCircle size={14} className="text-green-500" />
                : <XCircle size={14} className="text-slate-300 dark:text-slate-600" />
              }
            </div>
            {gmail?.connected ? (
              <button
                onClick={() => setConfirmDisconnect(true)}
                className="text-xs text-slate-400 hover:text-red-500 dark:hover:text-red-400 transition-colors cursor-pointer"
              >
                Disconnect
              </button>
            ) : (
              <a
                href="/api/integrations/gmail/auth"
                className="text-xs px-3 py-1.5 bg-slate-900 dark:bg-slate-700 text-white rounded-md hover:bg-slate-700 dark:hover:bg-slate-600 transition-colors"
              >
                Connect
              </a>
            )}
          </div>

          {/* WhatsApp */}
          <div className="bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <MessageCircle size={16} className="text-slate-500 dark:text-slate-400" />
                <div>
                  <p className="text-sm font-medium text-slate-800 dark:text-slate-200">WhatsApp</p>
                  {wa?.status === 'connected' && wa.phone && (
                    <p className="text-xs text-slate-400 dark:text-slate-500">{wa.phone}</p>
                  )}
                  {wa?.status === 'pending_qr' && (
                    <p className="text-xs text-slate-400 dark:text-slate-500">Scan QR code…</p>
                  )}
                </div>
                {wa?.status === 'connected'
                  ? <CheckCircle size={14} className="text-green-500" />
                  : <XCircle size={14} className="text-slate-300 dark:text-slate-600" />
                }
              </div>
              {wa?.status === 'connected' ? (
                <button
                  onClick={() => setConfirmWaDisconnect(true)}
                  className="text-xs text-slate-400 hover:text-red-500 dark:hover:text-red-400 transition-colors cursor-pointer"
                >
                  Disconnect
                </button>
              ) : (
                <button
                  onClick={handleWaConnect}
                  disabled={waConnecting}
                  className="text-xs px-3 py-1.5 bg-slate-900 dark:bg-slate-700 text-white rounded-md hover:bg-slate-700 dark:hover:bg-slate-600 disabled:opacity-50 transition-colors"
                >
                  {waConnecting ? 'Connecting…' : 'Connect'}
                </button>
              )}
            </div>

            {/* QR Code */}
            {waQR && (
              <div className="mt-4 flex flex-col items-center gap-2">
                <p className="text-xs text-slate-500 dark:text-slate-400">Open WhatsApp → Linked Devices → Link a device</p>
                <img
                  src={`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(waQR)}`}
                  alt="WhatsApp QR code"
                  className="rounded-lg border border-slate-200 dark:border-slate-700"
                  width={200}
                  height={200}
                />
              </div>
            )}

            {/* Contact whitelist */}
            {wa?.status === 'connected' && (
              <div className="mt-4 border-t border-slate-100 dark:border-slate-700 pt-4">
                <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wide mb-2">Monitored contacts</p>
                <div className="space-y-1.5 mb-3">
                  {waContacts.length === 0 && (
                    <p className="text-xs text-slate-400 italic">No contacts yet — add a phone number below.</p>
                  )}
                  {waContacts.map(c => (
                    <div key={c.id} className="flex items-center justify-between text-sm">
                      <div>
                        <span className="text-slate-800 dark:text-slate-200">{c.label || c.jid}</span>
                        {c.label && <span className="text-xs text-slate-400 ml-2">{c.jid}</span>}
                      </div>
                      <button
                        onClick={() => handleRemoveContact(c.id)}
                        className="text-slate-300 dark:text-slate-600 hover:text-red-500 transition-colors cursor-pointer"
                      >
                        <Trash2 size={13} />
                      </button>
                    </div>
                  ))}
                </div>
                <form onSubmit={handleAddContact} className="flex flex-col gap-2">
                  <input
                    className="border border-slate-300 dark:border-slate-600 rounded-md px-3 py-1.5 text-sm bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-400"
                    value={newContactJid}
                    onChange={e => setNewContactJid(e.target.value)}
                    placeholder="Phone number (e.g. 972501234567@s.whatsapp.net)"
                  />
                  <div className="flex gap-2">
                    <input
                      className="flex-1 border border-slate-300 dark:border-slate-600 rounded-md px-3 py-1.5 text-sm bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-400"
                      value={newContactLabel}
                      onChange={e => setNewContactLabel(e.target.value)}
                      placeholder="Label (e.g. Husband)"
                    />
                    <button
                      type="submit"
                      disabled={!newContactJid.trim()}
                      className="flex items-center gap-1 px-3 py-1.5 bg-slate-900 dark:bg-slate-700 text-white text-sm rounded-md hover:bg-slate-700 disabled:opacity-50 transition-colors"
                    >
                      <Plus size={13} />
                      Add
                    </button>
                  </div>
                </form>
              </div>
            )}
          </div>
        </div>
      </section>

      {/* People */}
      <section>
        <h2 className="text-sm font-semibold text-slate-700 dark:text-slate-400 uppercase tracking-wide mb-3">People</h2>

        {error && <p className="text-sm text-red-600 dark:text-red-400 mb-3">{error}</p>}

        {loading ? (
          <p className="text-sm text-slate-400 italic mb-4">Loading…</p>
        ) : (
          <div className="space-y-2 mb-4">
            {people.map(p => (
              <div key={p.id} className="flex items-center justify-between bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg px-4 py-2.5">
                <span className="text-sm text-slate-800 dark:text-slate-200">{p.name}</span>
                <button
                  onClick={() => setConfirmRemove({ id: p.id, name: p.name })}
                  aria-label={`Remove ${p.name}`}
                  className="text-slate-300 dark:text-slate-600 hover:text-red-500 dark:hover:text-red-400 transition-colors cursor-pointer"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <form onSubmit={handleAdd} className="flex gap-2">
          <input
            id="new-person-name"
            className="flex-1 border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 text-sm bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-400"
            value={newName}
            onChange={e => setNewName(e.target.value)}
            placeholder="Add person..."
            aria-label="New person name"
          />
          <button
            type="submit"
            disabled={saving || !newName.trim()}
            className="flex items-center gap-1 px-3 py-2 bg-slate-900 dark:bg-slate-700 text-white text-sm rounded-md hover:bg-slate-700 dark:hover:bg-slate-600 disabled:opacity-50 transition-colors"
          >
            <Plus size={14} />
            Add
          </button>
        </form>
      </section>

      {confirmRemove && (
        <Modal title="Remove person" onClose={() => setConfirmRemove(null)}>
          <p className="text-sm text-slate-600 dark:text-slate-300 mb-6">
            Remove {confirmRemove.name}?
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setConfirmRemove(null)}
              className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={() => handleDelete(confirmRemove.id)}
              className="px-4 py-2 text-sm text-white bg-red-800 rounded-md hover:bg-red-900 transition-colors"
            >
              Remove
            </button>
          </div>
        </Modal>
      )}

      {confirmDisconnect && (
        <Modal title="Disconnect Gmail" onClose={() => setConfirmDisconnect(false)}>
          <p className="text-sm text-slate-600 dark:text-slate-300 mb-6">
            Disconnect Gmail? You'll need to re-authorize to sync emails again.
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setConfirmDisconnect(false)}
              className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleDisconnect}
              className="px-4 py-2 text-sm text-white bg-red-800 rounded-md hover:bg-red-900 transition-colors"
            >
              Disconnect
            </button>
          </div>
        </Modal>
      )}

      {confirmWaDisconnect && (
        <Modal title="Disconnect WhatsApp" onClose={() => setConfirmWaDisconnect(false)}>
          <p className="text-sm text-slate-600 dark:text-slate-300 mb-6">
            Disconnect WhatsApp? The device pairing will be removed and you'll need to scan a new QR code to reconnect.
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setConfirmWaDisconnect(false)}
              className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleWaDisconnect}
              className="px-4 py-2 text-sm text-white bg-red-800 rounded-md hover:bg-red-900 transition-colors"
            >
              Disconnect
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}
