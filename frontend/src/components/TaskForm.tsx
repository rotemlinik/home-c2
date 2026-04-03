import { useState } from 'react'
import { type Category, type Person, createTask, updateTask, type Task } from '../api'

interface Props {
  people: Person[]
  categories: Category[]
  task?: Task
  onDone: () => void
  onCancel: () => void
}

const inputCls = 'w-full border border-slate-300 dark:border-slate-600 rounded-md px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-400'
const labelCls = 'block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1'

export default function TaskForm({ people, categories, task, onDone, onCancel }: Props) {
  const [title, setTitle] = useState(task?.title ?? '')
  const [type, setType] = useState<'one-off' | 'recurring'>(task?.type ?? 'one-off')
  const [dueDate, setDueDate] = useState(task?.due_date ?? '')
  const [assigneeId, setAssigneeId] = useState<string>(task?.assignee_id?.toString() ?? '')
  const [categoryId, setCategoryId] = useState<string>(task?.category_id?.toString() ?? '')
  const [notes, setNotes] = useState(task?.notes ?? '')
  const [recUnit, setRecUnit] = useState<'day' | 'week' | 'month'>(task?.recurrence?.unit ?? 'week')
  const [recEvery, setRecEvery] = useState<number>(task?.recurrence?.every ?? 1)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!title.trim()) return
    setSaving(true)
    setError(null)
    try {
      const body = {
        title: title.trim(),
        type,
        due_date: dueDate,
        assignee_id: assigneeId ? Number(assigneeId) : undefined,
        category_id: categoryId ? Number(categoryId) : undefined,
        notes,
        recurrence: type === 'recurring' ? { unit: recUnit, every: recEvery } : undefined,
      }
      if (task) {
        await updateTask(task.id, body)
      } else {
        await createTask(body)
      }
      onDone()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save task')
    } finally {
      setSaving(false)
    }
  }

  return (
    <form onSubmit={submit} className="space-y-4">
      <div>
        <label htmlFor="task-title" className={labelCls}>Title *</label>
        <input
          id="task-title"
          className={inputCls}
          value={title}
          onChange={e => setTitle(e.target.value)}
          placeholder="What needs doing?"
          autoFocus
          required
        />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="task-type" className={labelCls}>Type</label>
          <select
            id="task-type"
            className={inputCls}
            value={type}
            onChange={e => setType(e.target.value as 'one-off' | 'recurring')}
          >
            <option value="one-off">One-off</option>
            <option value="recurring">Recurring</option>
          </select>
        </div>

        <div>
          <label htmlFor="task-due-date" className={labelCls}>Due date</label>
          <input
            id="task-due-date"
            type="date"
            className={inputCls}
            value={dueDate}
            onChange={e => setDueDate(e.target.value)}
          />
        </div>
      </div>

      {type === 'recurring' && (
        <div className="flex items-center gap-2">
          <span className="text-sm text-slate-600 dark:text-slate-400">Every</span>
          <input
            id="task-rec-every"
            type="number"
            min={1}
            aria-label="Repeat every N"
            className="w-16 border border-slate-300 dark:border-slate-600 rounded-md px-2 py-1.5 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-400"
            value={recEvery}
            onChange={e => setRecEvery(Number(e.target.value))}
          />
          <select
            id="task-rec-unit"
            aria-label="Repeat unit"
            className="border border-slate-300 dark:border-slate-600 rounded-md px-2 py-1.5 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-slate-100 focus:outline-none focus:ring-2 focus:ring-slate-400"
            value={recUnit}
            onChange={e => setRecUnit(e.target.value as 'day' | 'week' | 'month')}
          >
            <option value="day">day(s)</option>
            <option value="week">week(s)</option>
            <option value="month">month(s)</option>
          </select>
        </div>
      )}

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="task-assignee" className={labelCls}>Assignee</label>
          <select
            id="task-assignee"
            className={inputCls}
            value={assigneeId}
            onChange={e => setAssigneeId(e.target.value)}
          >
            <option value="">Unassigned</option>
            {people.map(p => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
        </div>

        <div>
          <label htmlFor="task-category" className={labelCls}>Category</label>
          <select
            id="task-category"
            className={inputCls}
            value={categoryId}
            onChange={e => setCategoryId(e.target.value)}
          >
            <option value="">No category</option>
            {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </div>
      </div>

      <div>
        <label htmlFor="task-notes" className={labelCls}>Notes</label>
        <textarea
          id="task-notes"
          className={`${inputCls} resize-none`}
          rows={2}
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes..."
        />
      </div>

      {error && (
        <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
      )}

      <div className="flex justify-end gap-2 pt-2">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-sm text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={saving}
          className="px-4 py-2 bg-slate-900 text-white text-sm rounded-md hover:bg-slate-700 disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving...' : task ? 'Update' : 'Create'}
        </button>
      </div>
    </form>
  )
}
