import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import TaskCard from '../components/TaskCard'
import { mockTask, mockRecurringTask, mockOverdueTask, mockDoneTask } from './fixtures'

vi.mock('../api', () => ({
  doneTask: vi.fn(),
  reopenTask: vi.fn(),
  snoozeTask: vi.fn(),
  deleteTask: vi.fn(),
}))

import * as api from '../api'

beforeEach(() => {
  vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  vi.mocked(api.doneTask).mockResolvedValue(mockTask)
  vi.mocked(api.reopenTask).mockResolvedValue(mockTask)
  vi.mocked(api.snoozeTask).mockResolvedValue(mockTask)
  vi.mocked(api.deleteTask).mockResolvedValue(undefined)
})

afterEach(() => {
  vi.useRealTimers()
  vi.clearAllMocks()
})

function renderCard(task = mockTask, onRefresh = vi.fn(), onEdit = vi.fn()) {
  return render(<TaskCard task={task} onRefresh={onRefresh} onEdit={onEdit} />)
}

describe('TaskCard', () => {
  it('renders task title', () => {
    renderCard()
    expect(screen.getByText('Clean the kitchen')).toBeInTheDocument()
  })

  it('renders category badge', () => {
    renderCard()
    expect(screen.getByText('Housekeeping')).toBeInTheDocument()
  })

  it('renders assignee name', () => {
    renderCard()
    expect(screen.getByText(/Rotem/)).toBeInTheDocument()
  })

  it('renders due date formatted', () => {
    renderCard()
    // formatDate('2026-04-05') => '05 Apr 2026'
    expect(screen.getByText(/05 Apr 2026/)).toBeInTheDocument()
  })

  it('renders overdue styling (red border/bg) when past due', () => {
    const { container } = renderCard(mockOverdueTask)
    const card = container.firstChild as HTMLElement
    expect(card.className).toMatch(/border-red/)
  })

  it('renders done styling (opacity/strikethrough) when done', () => {
    const { container } = renderCard(mockDoneTask)
    const card = container.firstChild as HTMLElement
    expect(card.className).toMatch(/opacity/)
  })

  it('renders recurring badge for recurring tasks', () => {
    renderCard(mockRecurringTask)
    const svgs = document.querySelectorAll('svg')
    // RefreshCw icon should be present for recurring tasks
    expect(svgs.length).toBeGreaterThan(0)
    expect(screen.getByText(/every 1 week/)).toBeInTheDocument()
  })

  it('clicking mark-done button calls doneTask then onRefresh', async () => {
    const onRefresh = vi.fn()
    renderCard(mockTask, onRefresh)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Mark done'))
    await waitFor(() => expect(api.doneTask).toHaveBeenCalledWith(mockTask.id))
    await waitFor(() => expect(onRefresh).toHaveBeenCalled())
  })

  it('clicking edit button calls onEdit with the task', async () => {
    const onEdit = vi.fn()
    renderCard(mockTask, vi.fn(), onEdit)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Edit task'))
    expect(onEdit).toHaveBeenCalledWith(mockTask)
  })

  it('clicking snooze button opens dropdown with presets', async () => {
    renderCard()
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Snooze task'))
    expect(screen.getByText('Tomorrow')).toBeInTheDocument()
    expect(screen.getByText('In 3 days')).toBeInTheDocument()
    expect(screen.getByText('Next week')).toBeInTheDocument()
    expect(screen.getByText('Next month')).toBeInTheDocument()
  })

  it('clicking a snooze preset calls snoozeTask with correct preset then onRefresh', async () => {
    const onRefresh = vi.fn()
    renderCard(mockTask, onRefresh)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Snooze task'))
    await user.click(screen.getByText('Tomorrow'))
    await waitFor(() => expect(api.snoozeTask).toHaveBeenCalledWith(mockTask.id, 'tomorrow'))
    await waitFor(() => expect(onRefresh).toHaveBeenCalled())
  })

  it('snooze dropdown closes when clicking outside', async () => {
    renderCard()
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Snooze task'))
    expect(screen.getByText('Tomorrow')).toBeInTheDocument()
    await user.click(document.body)
    await waitFor(() => expect(screen.queryByText('Tomorrow')).not.toBeInTheDocument())
  })

  it('clicking delete shows confirmation modal; confirming calls deleteTask then onRefresh', async () => {
    const onRefresh = vi.fn()
    renderCard(mockTask, onRefresh)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Delete task'))
    expect(screen.getByText(/Delete "Clean the kitchen"\?/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Delete' }))
    await waitFor(() => expect(api.deleteTask).toHaveBeenCalledWith(mockTask.id))
    await waitFor(() => expect(onRefresh).toHaveBeenCalled())
  })

  it('clicking delete then cancel does not delete', async () => {
    renderCard()
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Delete task'))
    expect(screen.getByText(/Delete "Clean the kitchen"\?/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(api.deleteTask).not.toHaveBeenCalled()
  })

  it('shows reopen button for done tasks and calls reopenTask on click', async () => {
    const onRefresh = vi.fn()
    renderCard(mockDoneTask, onRefresh)
    const reopenBtn = screen.getByLabelText('Reopen task')
    expect(reopenBtn).toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(reopenBtn)
    await waitFor(() => expect(api.reopenTask).toHaveBeenCalledWith(mockDoneTask.id))
    await waitFor(() => expect(onRefresh).toHaveBeenCalled())
  })

  it('does not show reopen button for pending tasks', () => {
    renderCard(mockTask)
    expect(screen.queryByLabelText('Reopen task')).not.toBeInTheDocument()
  })

  it('does not show edit button for done tasks', () => {
    renderCard(mockDoneTask)
    expect(screen.queryByLabelText('Edit task')).not.toBeInTheDocument()
  })

  it('shows edit button for pending tasks', () => {
    renderCard(mockTask)
    expect(screen.getByLabelText('Edit task')).toBeInTheDocument()
  })

  it('shows tooltips on action buttons', () => {
    renderCard(mockTask)
    expect(screen.getByTitle('Edit')).toBeInTheDocument()
    expect(screen.getByTitle('Delete')).toBeInTheDocument()
    expect(screen.getByTitle('Snooze')).toBeInTheDocument()
  })

  it('shows reopen tooltip for done tasks', () => {
    renderCard(mockDoneTask)
    expect(screen.getByTitle('Reopen')).toBeInTheDocument()
  })
})
