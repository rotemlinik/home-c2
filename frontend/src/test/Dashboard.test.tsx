import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import Dashboard from '../pages/Dashboard'
import { mockTask, mockOverdueTask, mockPeople, mockCategories } from './fixtures'

vi.mock('../api', () => ({
  listTasks: vi.fn(),
  listPeople: vi.fn(),
  listCategories: vi.fn(),
  doneTask: vi.fn(),
  createTask: vi.fn(),
  updateTask: vi.fn(),
  snoozeTask: vi.fn(),
  deleteTask: vi.fn(),
}))

import * as api from '../api'

beforeEach(() => {
  vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  vi.mocked(api.listTasks).mockResolvedValue([])
  vi.mocked(api.listPeople).mockResolvedValue(mockPeople)
  vi.mocked(api.listCategories).mockResolvedValue(mockCategories)
  vi.mocked(api.doneTask).mockResolvedValue(mockTask)
  vi.mocked(api.createTask).mockResolvedValue(mockTask)
  vi.mocked(api.updateTask).mockResolvedValue(mockTask)
  vi.mocked(api.snoozeTask).mockResolvedValue(mockTask)
  vi.mocked(api.deleteTask).mockResolvedValue(undefined)
})

afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

describe('Dashboard', () => {
  it('shows loading state initially', () => {
    // Make the promise never resolve immediately so we see loading
    vi.mocked(api.listTasks).mockReturnValue(new Promise(() => {}))
    render(<Dashboard />)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('shows overdue section with tasks when overdue tasks exist (due_date < today)', async () => {
    vi.mocked(api.listTasks).mockResolvedValue([mockOverdueTask])
    render(<Dashboard />)
    await waitFor(() => expect(screen.getByRole('heading', { name: /overdue/i })).toBeInTheDocument())
    expect(screen.getByText('Pay rent')).toBeInTheDocument()
  })

  it('shows "Due today" section', async () => {
    vi.mocked(api.listTasks).mockResolvedValue([])
    render(<Dashboard />)
    await waitFor(() => expect(screen.getByRole('heading', { name: /due today/i })).toBeInTheDocument())
  })

  it('shows "Nothing due today." when no upcoming tasks', async () => {
    vi.mocked(api.listTasks).mockResolvedValue([])
    render(<Dashboard />)
    await waitFor(() => expect(screen.getByText('Nothing due today.')).toBeInTheDocument())
  })

  it('shows error message when listTasks rejects', async () => {
    vi.mocked(api.listTasks).mockRejectedValue(new Error('Network failure'))
    render(<Dashboard />)
    await waitFor(() => expect(screen.getByText('Network failure')).toBeInTheDocument())
  })

  it('"Add task" button opens create modal', async () => {
    render(<Dashboard />)
    const user = userEvent.setup()
    await waitFor(() => screen.getByText(/add task/i))
    await user.click(screen.getByText(/add task/i))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByText('New task')).toBeInTheDocument()
  })

  it('clicking edit on a task opens edit modal with task data', async () => {
    // Task due on the same day as today or after goes to "upcoming"
    const todayTask = { ...mockTask, due_date: '2026-04-01' }
    vi.mocked(api.listTasks).mockResolvedValue([todayTask])
    render(<Dashboard />)
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByText('Clean the kitchen')).toBeInTheDocument())
    await user.click(screen.getByLabelText('Edit task'))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByText('Edit task')).toBeInTheDocument()
  })

  it('marking a task done calls doneTask and refreshes the list', async () => {
    const todayTask = { ...mockTask, due_date: '2026-04-01' }
    vi.mocked(api.listTasks).mockResolvedValue([todayTask])
    render(<Dashboard />)
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByLabelText('Mark done')).toBeInTheDocument())
    await user.click(screen.getByLabelText('Mark done'))
    await waitFor(() => expect(api.doneTask).toHaveBeenCalledWith(todayTask.id))
    await waitFor(() => expect(api.listTasks).toHaveBeenCalledTimes(2))
  })
})
