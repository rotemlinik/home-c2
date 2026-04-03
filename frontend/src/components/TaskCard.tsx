import { useState, useRef, useEffect } from 'react'
import { type Task, type SnoozePreset, doneTask, reopenTask, snoozeTask, deleteTask } from '../api'
import { cn, formatDate, isOverdue, isDueToday } from '../lib/utils'
import { CheckCircle, Clock, Trash2, RefreshCw, Pencil, AlarmClock, Undo2 } from 'lucide-react'
import Modal from './Modal'

interface Props {
  task: Task
  onRefresh: () => void
  onEdit: (task: Task) => void
  selected?: boolean
  onSelect?: (id: number, checked: boolean) => void
  onHoverSelect?: (id: number, checked: boolean) => void
}

const SNOOZE_PRESETS: { value: SnoozePreset; label: string }[] = [
  { value: 'tomorrow', label: 'Tomorrow' },
  { value: '3days', label: 'In 3 days' },
  { value: 'week', label: 'Next week' },
  { value: 'month', label: 'Next month' },
]

const CATEGORY_COLORS: Record<string, string> = {
  Housekeeping: 'bg-blue-100 text-blue-700',
  Car: 'bg-orange-100 text-orange-700',
  Health: 'bg-green-100 text-green-700',
  Packages: 'bg-purple-100 text-purple-700',
  Finance: 'bg-yellow-100 text-yellow-700',
  Social: 'bg-pink-100 text-pink-700',
}

export default function TaskCard({ task, onRefresh, onEdit, selected, onSelect, onHoverSelect }: Props) {
  const [showSnooze, setShowSnooze] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [loading, setLoading] = useState(false)
  const snoozeRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!showSnooze) return
    function handleClickOutside(e: MouseEvent) {
      if (snoozeRef.current && !snoozeRef.current.contains(e.target as Node)) {
        setShowSnooze(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [showSnooze])

  const overdue = task.status === 'pending' && isOverdue(task.due_date)
  const today = task.status === 'pending' && isDueToday(task.due_date)

  async function handleDone() {
    setLoading(true)
    try {
      await doneTask(task.id)
      onRefresh()
    } finally {
      setLoading(false)
    }
  }

  async function handleReopen() {
    setLoading(true)
    try {
      await reopenTask(task.id)
      onRefresh()
    } finally {
      setLoading(false)
    }
  }

  async function handleSnooze(preset: SnoozePreset) {
    setShowSnooze(false)
    setLoading(true)
    try {
      await snoozeTask(task.id, preset)
      onRefresh()
    } finally {
      setLoading(false)
    }
  }

  async function handleDelete() {
    setShowDeleteConfirm(false)
    setLoading(true)
    try {
      await deleteTask(task.id)
      onRefresh()
    } finally {
      setLoading(false)
    }
  }

  const catColor = task.category_name ? (CATEGORY_COLORS[task.category_name] ?? 'bg-slate-100 text-slate-600') : ''

  return (
  <>
    <div
      className={cn(
        'bg-white dark:bg-slate-800 rounded-lg border p-4 flex items-start gap-3 transition-all min-h-[88px]',
        overdue ? 'border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950' : 'border-slate-200 dark:border-slate-700',
        task.status === 'done' && 'opacity-60',
        loading && 'opacity-50 pointer-events-none'
      )}
    >
      {/* Checkbox (batch mode or hover) + Done button */}
      {onSelect ? (
        <input
          type="checkbox"
          checked={selected ?? false}
          onChange={e => onSelect(task.id, e.target.checked)}
          aria-label={`Select ${task.title}`}
          className="mt-1 shrink-0 h-4 w-4 rounded border-slate-300 dark:border-slate-600 cursor-pointer accent-slate-600 dark:accent-slate-400"
        />
      ) : (
        <>
          {task.status !== 'done' && onHoverSelect && (
            <input
              type="checkbox"
              checked={false}
              onChange={() => onHoverSelect(task.id, true)}
              aria-label={`Select ${task.title}`}
              className="mt-1 shrink-0 h-4 w-4 rounded border-slate-300 dark:border-slate-600 cursor-pointer accent-slate-600 dark:accent-slate-400"
            />
          )}
          <button
            onClick={handleDone}
            disabled={task.status === 'done'}
            aria-label="Mark done"
            className={cn(
              'mt-0.5 shrink-0 transition-colors',
              task.status === 'done' ? 'text-green-500 cursor-default' : 'text-slate-300 dark:text-slate-600 hover:text-green-500 cursor-pointer'
            )}
          >
            <CheckCircle size={20} />
          </button>
        </>
      )}

      {/* Content */}
      <div className="flex-1 min-w-0">
        <div className="flex items-start gap-2 flex-wrap">
          <span className={cn('text-sm font-medium text-slate-900 dark:text-slate-100', task.status === 'done' && 'line-through text-slate-400 dark:text-slate-500')}>
            {task.title}
          </span>

          {task.type === 'recurring' && (
            <span className="text-xs text-slate-400 flex items-center gap-0.5 mt-0.5">
              <RefreshCw size={10} />
              {task.recurrence && `every ${task.recurrence.every} ${task.recurrence.unit}${task.recurrence.every > 1 ? 's' : ''}`}
            </span>
          )}
          {task.source === 'gmail' && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-red-50 dark:bg-red-950 text-red-500 dark:text-red-400 font-medium">Gmail</span>
          )}
          {task.source === 'imessage' && (
            <span className="text-xs px-1.5 py-0.5 rounded bg-blue-50 dark:bg-blue-950 text-blue-500 dark:text-blue-400 font-medium">iMessage</span>
          )}
        </div>

        <div className="flex items-center gap-2 mt-1.5 flex-wrap">
          {task.category_name && (
            <span className={cn('text-xs px-2 py-0.5 rounded-full font-medium', catColor)}>
              {task.category_name}
            </span>
          )}
          {task.assignee_name && (
            <span className="text-xs text-slate-500 dark:text-slate-400">→ {task.assignee_name}</span>
          )}
          {task.due_date && (
            <span className={cn(
              'text-xs flex items-center gap-1',
              overdue ? 'text-red-600 dark:text-red-400 font-medium' : today ? 'text-amber-600 dark:text-amber-400 font-medium' : 'text-slate-400 dark:text-slate-500'
            )}>
              <Clock size={10} />
              {overdue ? 'Overdue · ' : today ? 'Today · ' : ''}{formatDate(task.due_date)}
            </span>
          )}
          {task.status === 'snoozed' && (
            <span className="text-xs text-slate-400 italic">snoozed</span>
          )}
        </div>

        {task.notes && (
          <p className="text-xs text-slate-400 mt-1 truncate">{task.notes}</p>
        )}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-1 shrink-0">
        {/* Reopen */}
        {task.status === 'done' && (
          <button
            onClick={handleReopen}
            title="Reopen"
            aria-label="Reopen task"
            className="p-1.5 text-slate-400 dark:text-slate-500 hover:text-amber-500 dark:hover:text-amber-400 rounded transition-colors cursor-pointer"
          >
            <Undo2 size={14} />
          </button>
        )}

        {/* Edit (not for completed tasks) */}
        {task.status !== 'done' && (
          <button
            onClick={() => onEdit(task)}
            title="Edit"
            aria-label="Edit task"
            className="p-1.5 text-slate-400 dark:text-slate-500 hover:text-slate-700 dark:hover:text-slate-200 rounded transition-colors cursor-pointer"
          >
            <Pencil size={14} />
          </button>
        )}

        {/* Snooze */}
        {task.status === 'pending' && (
          <div className="relative" ref={snoozeRef}>
            <button
              onClick={() => setShowSnooze(s => !s)}
              title="Snooze"
              aria-label="Snooze task"
              aria-expanded={showSnooze}
              className="p-1.5 text-slate-400 dark:text-slate-500 hover:text-slate-700 dark:hover:text-slate-200 rounded transition-colors cursor-pointer"
            >
              <AlarmClock size={13} />
            </button>
            {showSnooze && (
              <div className="absolute right-0 top-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md shadow-lg z-10 min-w-36">
                <p className="text-xs text-slate-400 dark:text-slate-500 px-3 pt-2 pb-1 font-medium uppercase tracking-wide">Snooze until</p>
                {SNOOZE_PRESETS.map(p => (
                  <button
                    key={p.value}
                    onClick={() => handleSnooze(p.value)}
                    className="w-full text-left px-3 py-1.5 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Delete */}
        <button
          onClick={() => setShowDeleteConfirm(true)}
          title="Delete"
          aria-label="Delete task"
          className="p-1.5 text-slate-400 dark:text-slate-500 hover:text-red-500 dark:hover:text-red-400 rounded transition-colors cursor-pointer"
        >
          <Trash2 size={14} />
        </button>
      </div>

    </div>

    {showDeleteConfirm && (
      <Modal title="Delete task" onClose={() => setShowDeleteConfirm(false)}>
        <p className="text-sm text-slate-600 dark:text-slate-300 mb-6">
          Delete "{task.title}"? This cannot be undone.
        </p>
        <div className="flex justify-end gap-2">
          <button
            onClick={() => setShowDeleteConfirm(false)}
            className="px-4 py-2 text-sm text-slate-600 dark:text-slate-300 border border-slate-200 dark:border-slate-700 rounded-md hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleDelete}
            className="px-4 py-2 text-sm text-white bg-red-800 rounded-md hover:bg-red-900 transition-colors"
          >
            Delete
          </button>
        </div>
      </Modal>
    )}
  </>
  )
}
