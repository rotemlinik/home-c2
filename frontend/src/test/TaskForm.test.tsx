import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import TaskForm from '../components/TaskForm'
import { mockPeople, mockCategories, mockTask } from './fixtures'

vi.mock('../api', () => ({
  createTask: vi.fn(),
  updateTask: vi.fn(),
}))

import * as api from '../api'

beforeEach(() => {
  vi.mocked(api.createTask).mockResolvedValue(mockTask)
  vi.mocked(api.updateTask).mockResolvedValue(mockTask)
})

afterEach(() => {
  vi.clearAllMocks()
})

function renderForm(props: Partial<Parameters<typeof TaskForm>[0]> = {}) {
  const defaults = {
    people: mockPeople,
    categories: mockCategories,
    onDone: vi.fn(),
    onCancel: vi.fn(),
  }
  return render(<TaskForm {...defaults} {...props} />)
}

describe('TaskForm', () => {
  it('renders all form fields (title, type, due date, assignee, category, notes)', () => {
    renderForm()
    expect(screen.getByLabelText(/title/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/type/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/due date/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/assignee/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/category/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/notes/i)).toBeInTheDocument()
  })

  it('recurrence fields hidden when type is one-off', () => {
    renderForm()
    expect(screen.queryByLabelText('Repeat every N')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Repeat unit')).not.toBeInTheDocument()
  })

  it('recurrence fields shown when type is recurring', async () => {
    renderForm()
    const user = userEvent.setup()
    await user.selectOptions(screen.getByLabelText(/type/i), 'recurring')
    expect(screen.getByLabelText('Repeat every N')).toBeInTheDocument()
    expect(screen.getByLabelText('Repeat unit')).toBeInTheDocument()
  })

  it('submit with empty title does not call createTask', async () => {
    renderForm()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /create/i }))
    expect(api.createTask).not.toHaveBeenCalled()
  })

  it('submit with valid data calls createTask with correct payload', async () => {
    renderForm()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText(/title/i), 'New task')
    await user.type(screen.getByLabelText(/due date/i), '2026-04-10')
    await user.click(screen.getByRole('button', { name: /create/i }))
    await waitFor(() =>
      expect(api.createTask).toHaveBeenCalledWith(
        expect.objectContaining({ title: 'New task', type: 'one-off' })
      )
    )
  })

  it('submit calls onDone after success', async () => {
    const onDone = vi.fn()
    renderForm({ onDone })
    const user = userEvent.setup()
    await user.type(screen.getByLabelText(/title/i), 'New task')
    await user.click(screen.getByRole('button', { name: /create/i }))
    await waitFor(() => expect(onDone).toHaveBeenCalled())
  })

  it('update mode: pre-fills fields from task prop, calls updateTask on submit', async () => {
    renderForm({ task: mockTask })
    const user = userEvent.setup()
    expect(screen.getByLabelText(/title/i)).toHaveValue('Clean the kitchen')
    await user.click(screen.getByRole('button', { name: /update/i }))
    await waitFor(() => expect(api.updateTask).toHaveBeenCalledWith(mockTask.id, expect.any(Object)))
  })

  it('cancel button calls onCancel', async () => {
    const onCancel = vi.fn()
    renderForm({ onCancel })
    const user = userEvent.setup()
    await user.click(screen.getByRole('button', { name: /cancel/i }))
    expect(onCancel).toHaveBeenCalled()
  })

  it('shows error message when createTask rejects', async () => {
    vi.mocked(api.createTask).mockRejectedValue(new Error('Server error'))
    renderForm()
    const user = userEvent.setup()
    await user.type(screen.getByLabelText(/title/i), 'New task')
    await user.click(screen.getByRole('button', { name: /create/i }))
    await waitFor(() => expect(screen.getByText('Server error')).toBeInTheDocument())
  })
})
