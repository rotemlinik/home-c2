import { useState, useRef, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '../lib/utils'

interface Option {
  value: string
  label: string
}

interface Props {
  value: string
  options: Option[]
  onChange: (value: string) => void
}

export default function FilterSelect({ value, options, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  const selected = options.find(o => o.value === value) ?? options[0]

  useEffect(() => {
    if (!open) return
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [open])

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(o => !o)}
        className={cn(
          'flex items-center gap-2 px-3 py-1.5 text-sm border rounded-md bg-white transition-colors',
          open ? 'border-slate-400 dark:border-slate-500 text-slate-900 dark:text-slate-100 bg-white dark:bg-slate-800' : 'border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 hover:border-slate-300 dark:hover:border-slate-600'
        )}
      >
        {selected.label}
        <ChevronDown size={14} className={cn('text-slate-400 transition-transform', open && 'rotate-180')} />
      </button>

      {open && (
        <div className="absolute left-0 top-[calc(100%+4px)] bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-md shadow-lg z-20 min-w-full">
          {options.map(o => (
            <button
              key={o.value}
              onClick={() => { onChange(o.value); setOpen(false) }}
              className={cn(
                'w-full text-left px-3 py-1.5 text-sm transition-colors whitespace-nowrap',
                o.value === value ? 'text-slate-900 dark:text-white font-medium bg-slate-50 dark:bg-slate-700' : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
              )}
            >
              {o.label}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}
