import { useCallback, useEffect, useState } from 'react'
import { type Task, type Category, type Person, listTasks, listCategories, listPeople, deleteTask } from '../api'
import TaskCard from '../components/TaskCard'
import TaskForm from '../components/TaskForm'
import Modal from '../components/Modal'
import { Plus, Filter, Trash2 } from 'lucide-react'
import FilterSelect from '../components/FilterSelect'
import SyncButton from '../components/SyncButton'

type StatusFilter = '' | 'pending' | 'done' | 'snoozed'
type DateFilter = '' | 'today' | 'week' | 'month'

function dateWindowEnd(filter: DateFilter): string | null {
  const d = new Date()
  if (filter === 'today') return d.toISOString().slice(0, 10)
  if (filter === 'week') {
    d.setDate(d.getDate() + (7 - d.getDay()))
    return d.toISOString().slice(0, 10)
  }
  if (filter === 'month') {
    return new Date(d.getFullYear(), d.getMonth() + 1, 0).toISOString().slice(0, 10)
  }
  return null
}

export default function Tasks() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [people, setPeople] = useState<Person[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [showCreate, setShowCreate] = useState(false)
  const [editTask, setEditTask] = useState<Task | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [filterCategory, setFilterCategory] = useState('')
  const [filterStatus, setFilterStatus] = useState<StatusFilter>('pending')
  const [filterDate, setFilterDate] = useState<DateFilter>('')
  const [batchMode, setBatchMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [batchDeleting, setBatchDeleting] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [ts, ppl, cats] = await Promise.all([
        listTasks({
          view: 'all',
          status: filterStatus || undefined,
          category: filterCategory || undefined,
        }),
        listPeople(),
        listCategories(),
      ])
      setTasks(ts)
      setPeople(ppl)
      setCategories(cats)
      setSelectedIds(new Set())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load tasks')
    } finally {
      setLoading(false)
    }
  }, [filterCategory, filterStatus])

  useEffect(() => { load() }, [load])

  function toggleSelect(id: number, checked: boolean) {
    setSelectedIds(prev => {
      const next = new Set(prev)
      checked ? next.add(id) : next.delete(id)
      if (next.size === 0) setBatchMode(false)
      return next
    })
  }

  function enterBatchWithSelect(id: number) {
    setBatchMode(true)
    setSelectedIds(new Set([id]))
  }

  function toggleSelectAll() {
    if (selectedIds.size === tasks.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(tasks.map(t => t.id)))
    }
  }

  async function confirmBatchDelete() {
    setShowDeleteConfirm(false)
    setBatchDeleting(true)
    try {
      await Promise.all([...selectedIds].map(id => deleteTask(id)))
      load()
    } finally {
      setBatchDeleting(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-slate-100">Tasks</h1>
        <div className="flex items-center gap-2">
          <SyncButton onSynced={load} />
          <button
            onClick={() => setShowCreate(true)}
            className="flex items-center gap-2 px-4 py-2 bg-slate-900 text-white text-sm rounded-md hover:bg-slate-700 transition-colors"
          >
            <Plus size={16} />
            Add task
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <Filter size={14} className="text-slate-400" />

        <FilterSelect
          value={filterStatus}
          onChange={v => setFilterStatus(v as StatusFilter)}
          options={[
            { value: 'pending', label: 'Pending' },
            { value: 'done', label: 'Done' },
            { value: 'snoozed', label: 'Snoozed' },
            { value: '', label: 'All' },
          ]}
        />

        <FilterSelect
          value={filterCategory}
          onChange={setFilterCategory}
          options={[
            { value: '', label: 'All categories' },
            ...categories.map(c => ({ value: String(c.id), label: c.name })),
          ]}
        />

        <FilterSelect
          value={filterDate}
          onChange={v => setFilterDate(v as DateFilter)}
          options={[
            { value: '', label: 'All time' },
            { value: 'today', label: 'Today' },
            { value: 'week', label: 'This week' },
            { value: 'month', label: 'This month' },
          ]}
        />

        {(filterCategory || filterDate || filterStatus !== 'pending') && (
          <button
            onClick={() => { setFilterCategory(''); setFilterDate(''); setFilterStatus('pending') }}
            className="text-xs text-slate-400 hover:text-slate-700 transition-colors"
          >
            Clear
          </button>
        )}
      </div>

      {/* Batch action bar */}
      {batchMode && (
        <div className="flex items-center justify-between px-3 py-2 bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md">
          <span className="text-sm text-slate-600 dark:text-slate-300">
            {selectedIds.size} selected
          </span>
          <div className="flex items-center gap-3">
            <button
              onClick={toggleSelectAll}
              className="text-sm text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
            >
              {selectedIds.size === tasks.length ? 'Deselect all' : 'Select all'}
            </button>
            <button
              onClick={() => setShowDeleteConfirm(true)}
              disabled={batchDeleting || selectedIds.size === 0}
              className="flex items-center gap-1.5 text-sm text-red-500 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300 disabled:opacity-50 transition-colors"
            >
              <Trash2 size={13} />
              Delete
            </button>
            <button
              onClick={() => { setBatchMode(false); setSelectedIds(new Set()) }}
              className="text-sm text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 transition-colors"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* States */}
      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}
      {loading ? (
        <p className="text-sm text-slate-400 italic">Loading…</p>
      ) : tasks.length === 0 ? (
        <p className="text-sm text-slate-400 italic">No tasks found.</p>
      ) : (
        <div className="space-y-8">
          {[...people.map(p => p.name), null].map(name => {
            const windowEnd = dateWindowEnd(filterDate)
            const group = tasks.filter(t => {
              if (name === null ? !!t.assignee_name : t.assignee_name !== name) return false
              if (windowEnd && t.due_date && t.due_date > windowEnd) return false
              return true
            })
            if (group.length === 0) return null
            return (
              <section key={name ?? '__unassigned'}>
                <h2 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wide mb-3">
                  {name ?? 'Unassigned'}
                </h2>
                <div className="space-y-2">
                  {group.map(t => (
                    <TaskCard
                      key={t.id}
                      task={t}
                      onRefresh={load}
                      onEdit={setEditTask}
                      selected={batchMode ? selectedIds.has(t.id) : undefined}
                      onSelect={batchMode ? toggleSelect : undefined}
                      onHoverSelect={!batchMode ? enterBatchWithSelect : undefined}
                    />
                  ))}
                </div>
              </section>
            )
          })}
        </div>
      )}

      {(showCreate || editTask) && (
        <Modal
          title={editTask ? 'Edit task' : 'New task'}
          onClose={() => { setShowCreate(false); setEditTask(null) }}
        >
          <TaskForm
            people={people}
            categories={categories}
            task={editTask ?? undefined}
            onDone={() => { setShowCreate(false); setEditTask(null); load() }}
            onCancel={() => { setShowCreate(false); setEditTask(null) }}
          />
        </Modal>
      )}

      {showDeleteConfirm && (
        <Modal title="Delete tasks" onClose={() => setShowDeleteConfirm(false)}>
          <p className="text-sm text-slate-600 dark:text-slate-300 mb-6">
            Delete {selectedIds.size} task{selectedIds.size !== 1 ? 's' : ''}? This cannot be undone.
          </p>
          <div className="flex justify-end gap-2">
            <button
              onClick={() => setShowDeleteConfirm(false)}
              className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={confirmBatchDelete}
              className="px-4 py-2 text-sm text-white bg-red-800 rounded-md hover:bg-red-900 transition-colors"
            >
              Delete
            </button>
          </div>
        </Modal>
      )}
    </div>
  )
}
