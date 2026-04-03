import type { Person, Category, Task } from '../api'

export const mockPeople: Person[] = [
  { id: 1, name: 'Rotem', created_at: '2026-01-01T00:00:00Z' },
  { id: 2, name: 'Ben', created_at: '2026-01-01T00:00:00Z' },
]

export const mockCategories: Category[] = [
  { id: 1, name: 'Housekeeping' },
  { id: 2, name: 'Car' },
  { id: 3, name: 'Health' },
  { id: 4, name: 'Packages' },
  { id: 5, name: 'Finance' },
]

export const mockTask: Task = {
  id: 1,
  title: 'Clean the kitchen',
  type: 'one-off',
  due_date: '2026-04-05',
  assignee_id: 1,
  assignee_name: 'Rotem',
  category_id: 1,
  category_name: 'Housekeeping',
  status: 'pending',
  notes: '',
  source: 'manual',
  created_at: '2026-01-01T00:00:00Z',
}

export const mockRecurringTask: Task = {
  id: 2,
  title: 'Vacuum floors',
  type: 'recurring',
  recurrence: { unit: 'week', every: 1 },
  due_date: '2026-04-05',
  status: 'pending',
  notes: '',
  source: 'manual',
  created_at: '2026-01-01T00:00:00Z',
}

export const mockOverdueTask: Task = {
  id: 3,
  title: 'Pay rent',
  type: 'one-off',
  due_date: '2026-03-01',
  status: 'pending',
  notes: '',
  source: 'manual',
  created_at: '2026-01-01T00:00:00Z',
}

export const mockDoneTask: Task = {
  id: 4,
  title: 'Buy groceries',
  type: 'one-off',
  due_date: '2026-04-05',
  status: 'done',
  notes: '',
  source: 'manual',
  created_at: '2026-01-01T00:00:00Z',
}
