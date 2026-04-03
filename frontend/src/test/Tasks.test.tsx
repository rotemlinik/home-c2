import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import Tasks from '../pages/Tasks'
import { mockTask, mockPeople, mockCategories } from './fixtures'

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
  vi.mocked(api.listTasks).mockResolvedValue([mockTask])
  vi.mocked(api.listPeople).mockResolvedValue(mockPeople)
  vi.mocked(api.listCategories).mockResolvedValue(mockCategories)
  vi.mocked(api.doneTask).mockResolvedValue(mockTask)
  vi.mocked(api.createTask).mockResolvedValue(mockTask)
  vi.mocked(api.updateTask).mockResolvedValue(mockTask)
  vi.mocked(api.snoozeTask).mockResolvedValue(mockTask)
  vi.mocked(api.deleteTask).mockResolvedValue(undefined)
})

afterEach(() => {
  vi.clearAllMocks()
})

describe('Tasks', () => {
  it('shows loading state initially', () => {
    vi.mocked(api.listTasks).mockReturnValue(new Promise(() => {}))
    render(<Tasks />)
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('renders task list after load', async () => {
    render(<Tasks />)
    await waitFor(() => expect(screen.getByText('Clean the kitchen')).toBeInTheDocument())
  })

  it('shows "No tasks found." for empty list', async () => {
    vi.mocked(api.listTasks).mockResolvedValue([])
    render(<Tasks />)
    await waitFor(() => expect(screen.getByText('No tasks found.')).toBeInTheDocument())
  })

  it('shows error when API fails', async () => {
    vi.mocked(api.listTasks).mockRejectedValue(new Error('API error'))
    render(<Tasks />)
    await waitFor(() => expect(screen.getByText('API error')).toBeInTheDocument())
  })

  it('status filter change triggers new load with correct status param', async () => {
    render(<Tasks />)
    const user = userEvent.setup()
    await waitFor(() => expect(api.listTasks).toHaveBeenCalledTimes(1))
    // The status filter is the first FilterSelect - click to open
    const filterButtons = screen.getAllByRole('button')
    // First filter button is status which shows "Pending" by default
    const statusButton = filterButtons.find(b => b.textContent?.includes('Pending'))!
    await user.click(statusButton)
    await user.click(screen.getByText('Done'))
    await waitFor(() =>
      expect(api.listTasks).toHaveBeenLastCalledWith(
        expect.objectContaining({ status: 'done' })
      )
    )
  })

  it('category filter change triggers reload', async () => {
    render(<Tasks />)
    const user = userEvent.setup()
    await waitFor(() => expect(api.listTasks).toHaveBeenCalledTimes(1))
    const categoryButton = screen.getAllByRole('button').find(b =>
      b.textContent?.includes('All categories')
    )!
    await user.click(categoryButton)
    // Find Housekeeping as a button (in the dropdown), not the task badge
    const housekeepingBtns = screen.getAllByRole('button').filter(b =>
      b.textContent?.trim() === 'Housekeeping'
    )
    await user.click(housekeepingBtns[0])
    await waitFor(() => expect(api.listTasks).toHaveBeenCalledTimes(2))
  })

  it('Clear button resets all filters to defaults', async () => {
    render(<Tasks />)
    const user = userEvent.setup()
    await waitFor(() => expect(api.listTasks).toHaveBeenCalledTimes(1))
    // Change status filter to something different so Clear button appears
    const statusButton = screen.getAllByRole('button').find(b => b.textContent?.includes('Pending'))!
    await user.click(statusButton)
    await user.click(screen.getByText('Done'))
    await waitFor(() => screen.getByText('Clear'))
    await user.click(screen.getByText('Clear'))
    // Status should go back to pending (the default)
    await waitFor(() =>
      expect(api.listTasks).toHaveBeenLastCalledWith(
        expect.objectContaining({ status: 'pending' })
      )
    )
  })

  it('"Add task" opens create modal', async () => {
    render(<Tasks />)
    const user = userEvent.setup()
    await waitFor(() => screen.getByText(/add task/i))
    await user.click(screen.getByText(/add task/i))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByText('New task')).toBeInTheDocument()
  })

  it('edit action opens edit modal', async () => {
    render(<Tasks />)
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByLabelText('Edit task')).toBeInTheDocument())
    await user.click(screen.getByLabelText('Edit task'))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    expect(screen.getByText('Edit task')).toBeInTheDocument()
  })

  it('after task created, modal closes and list reloads', async () => {
    render(<Tasks />)
    const user = userEvent.setup()
    await waitFor(() => screen.getByText(/add task/i))
    await user.click(screen.getByText(/add task/i))
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument())
    // Fill in and submit the form
    await user.type(screen.getByLabelText(/title/i), 'New task')
    await user.click(screen.getByRole('button', { name: /create/i }))
    await waitFor(() => expect(api.createTask).toHaveBeenCalled())
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    await waitFor(() => expect(api.listTasks).toHaveBeenCalledTimes(2))
  })
})
