import { useState, useEffect } from 'react'
import Tasks from './pages/Tasks'
import Settings from './pages/Settings'
import { ListTodo, Settings as SettingsIcon } from 'lucide-react'
import { cn } from './lib/utils'

type Page = 'tasks' | 'settings'

const NAV = [
  { id: 'tasks' as Page, label: 'Tasks', Icon: ListTodo },
  { id: 'settings' as Page, label: 'Settings', Icon: SettingsIcon },
]

export default function App() {
  const [page, setPage] = useState<Page>('tasks')
  const [dark, setDark] = useState(() => localStorage.getItem('theme') === 'dark')

  useEffect(() => {
    document.documentElement.classList.toggle('dark', dark)
    localStorage.setItem('theme', dark ? 'dark' : 'light')
  }, [dark])

  return (
    <div className="min-h-screen flex bg-slate-100 dark:bg-slate-950">
      {/* Sidebar */}
      <nav className="w-56 bg-white dark:bg-slate-900 border-r border-slate-200 dark:border-slate-800 flex flex-col shrink-0">
        <div className="px-5 py-5 border-b border-slate-100 dark:border-slate-800">
          <h1 className="text-base font-semibold text-slate-900 dark:text-slate-100">🏠 Home C2</h1>
        </div>
        <div className="flex-1 px-3 py-4 space-y-1">
          {NAV.map(({ id, label, Icon }) => (
            <button
              key={id}
              onClick={() => setPage(id)}
              className={cn(
                'w-full flex items-center gap-3 px-3 py-2 text-sm rounded-md transition-colors text-left',
                page === id
                  ? 'bg-slate-100 dark:bg-slate-700 text-slate-900 dark:text-white font-medium'
                  : 'text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 hover:text-slate-900 dark:hover:text-slate-100'
              )}
            >
              <Icon size={16} />
              {label}
            </button>
          ))}
        </div>
      </nav>

      {/* Main */}
      <main className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto px-6 py-8">
          {page === 'tasks' && <Tasks />}
          {page === 'settings' && <Settings dark={dark} onDarkChange={setDark} />}
        </div>
      </main>
    </div>
  )
}
