export interface Person {
  id: number
  name: string
  created_at: string
}

export interface Category {
  id: number
  name: string
}

export interface Recurrence {
  unit: 'day' | 'week' | 'month'
  every: number
}

export interface Task {
  id: number
  title: string
  type: 'one-off' | 'recurring'
  recurrence?: Recurrence
  due_date: string
  assignee_id?: number
  category_id?: number
  status: 'pending' | 'done' | 'snoozed'
  completed_by?: number
  completed_at?: string
  notes: string
  source: 'manual' | 'gmail' | 'imessage' | 'whatsapp'
  source_ref?: string
  parent_id?: number
  created_at: string
  assignee_name?: string
  category_name?: string
}

export interface GmailStatus {
  connected: boolean
  email?: string
  last_synced_at?: string
  tasks_created_last_sync?: number
}

export interface SyncResult {
  tasks_created: number
  emails_read: number
}

export interface SyncLogEntry {
  source: string
  synced_at: string
  emails_read: number
  tasks_created: number
  error?: string
}

export type SnoozePreset = 'tomorrow' | '3days' | 'week' | 'month'

const base = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(base + path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) throw new Error(await res.text())
  if (res.status === 204) return undefined as T
  return res.json()
}

// Tasks
export const listTasks = (params?: {
  view?: 'today' | 'upcoming' | 'all'
  status?: string
  assignee?: string
  category?: string
}) => {
  const qs = new URLSearchParams()
  if (params?.view) qs.set('view', params.view)
  if (params?.status) qs.set('status', params.status)
  if (params?.assignee) qs.set('assignee', params.assignee)
  if (params?.category) qs.set('category', params.category)
  return request<Task[]>(`/tasks?${qs}`)
}

export const createTask = (body: {
  title: string
  type: 'one-off' | 'recurring'
  recurrence?: Recurrence
  due_date: string
  assignee_id?: number
  category_id?: number
  notes?: string
}) => request<Task>('/tasks', { method: 'POST', body: JSON.stringify(body) })

export const updateTask = (id: number, body: Partial<Task>) =>
  request<Task>(`/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(body) })

export const doneTask = (id: number, completed_by?: number) =>
  request<Task>(`/tasks/${id}/done`, { method: 'POST', body: JSON.stringify({ completed_by }) })

export const reopenTask = (id: number) =>
  request<Task>(`/tasks/${id}/reopen`, { method: 'POST' })

export const snoozeTask = (id: number, preset: SnoozePreset) =>
  request<Task>(`/tasks/${id}/snooze`, { method: 'POST', body: JSON.stringify({ preset }) })

export const deleteTask = (id: number) =>
  request<void>(`/tasks/${id}`, { method: 'DELETE' })

export const taskHistory = (id: number) =>
  request<Task[]>(`/tasks/${id}/history`)

// People
export const listPeople = () => request<Person[]>('/people')
export const createPerson = (name: string) =>
  request<Person>('/people', { method: 'POST', body: JSON.stringify({ name }) })
export const deletePerson = (id: number) =>
  request<void>(`/people/${id}`, { method: 'DELETE' })

// Categories
export const listCategories = () => request<Category[]>('/categories')

// Gmail integration
export const gmailStatus = () => request<GmailStatus>('/integrations/gmail/status')
export const gmailDisconnect = () => request<void>('/integrations/gmail', { method: 'DELETE' })
export const syncGmail = () => request<SyncResult>('/sync/gmail', { method: 'POST' })
export const syncHistory = () => request<SyncLogEntry[]>('/sync/history')

// WhatsApp integration
export interface WhatsAppStatus {
  status: 'disconnected' | 'pending_qr' | 'connected'
  phone?: string
  last_synced_at?: string
}

export interface WhatsAppQR {
  qr_data: string | null
  expires_at?: string
}

export interface WhatsAppContact {
  id: number
  jid: string
  label: string
  created_at: string
}

export const whatsappStatus = () => request<WhatsAppStatus>('/integrations/whatsapp/status')
export const whatsappConnect = () => request<WhatsAppStatus>('/integrations/whatsapp/connect', { method: 'POST' })
export const whatsappDisconnect = () => request<void>('/integrations/whatsapp/disconnect', { method: 'POST' })
export const whatsappQR = () => request<WhatsAppQR>('/integrations/whatsapp/qr')
export const whatsappListContacts = () => request<WhatsAppContact[]>('/integrations/whatsapp/contacts')
export const whatsappAddContact = (jid: string, label: string) =>
  request<WhatsAppContact>('/integrations/whatsapp/contacts', { method: 'POST', body: JSON.stringify({ jid, label }) })
export const whatsappRemoveContact = (id: number) =>
  request<void>(`/integrations/whatsapp/contacts/${id}`, { method: 'DELETE' })
export const syncWhatsApp = () => request<SyncResult>('/sync/whatsapp', { method: 'POST' })
