import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

function toDateObj(d: string): Date {
  // Normalize to YYYY-MM-DD in case SQLite returns datetime with time component
  return new Date(d.slice(0, 10) + 'T12:00:00')
}

export function formatDate(d: string | undefined): string {
  if (!d) return '—'
  const date = toDateObj(d)
  if (isNaN(date.getTime())) return '—'
  return date.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export function isOverdue(due_date: string | undefined): boolean {
  if (!due_date) return false
  return toDateObj(due_date) < new Date(new Date().toDateString())
}

export function isDueToday(due_date: string | undefined): boolean {
  if (!due_date) return false
  return due_date === new Date().toISOString().slice(0, 10)
}
