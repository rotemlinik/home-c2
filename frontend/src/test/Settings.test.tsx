import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import Settings from '../pages/Settings'
import { mockPeople } from './fixtures'

vi.mock('../api', () => ({
  listPeople: vi.fn(),
  createPerson: vi.fn(),
  deletePerson: vi.fn(),
  gmailStatus: vi.fn(),
  gmailDisconnect: vi.fn(),
  whatsappStatus: vi.fn(),
  whatsappConnect: vi.fn(),
  whatsappDisconnect: vi.fn(),
  whatsappQR: vi.fn(),
  whatsappListContacts: vi.fn(),
  whatsappAddContact: vi.fn(),
  whatsappRemoveContact: vi.fn(),
}))

import * as api from '../api'

beforeEach(() => {
  vi.mocked(api.listPeople).mockResolvedValue(mockPeople)
  vi.mocked(api.createPerson).mockResolvedValue({ id: 3, name: 'Alice', created_at: '2026-01-01T00:00:00Z' })
  vi.mocked(api.deletePerson).mockResolvedValue(undefined)
  vi.mocked(api.gmailStatus).mockResolvedValue({ connected: false })
  vi.mocked(api.whatsappStatus).mockResolvedValue({ status: 'disconnected' })
  vi.mocked(api.whatsappListContacts).mockResolvedValue([])
})

afterEach(() => {
  vi.clearAllMocks()
})

function renderSettings(dark = false, onDarkChange = vi.fn()) {
  return render(<Settings dark={dark} onDarkChange={onDarkChange} />)
}

describe('Settings', () => {
  it('shows loading state initially', () => {
    vi.mocked(api.listPeople).mockReturnValue(new Promise(() => {}))
    vi.mocked(api.gmailStatus).mockReturnValue(new Promise(() => {}))
    vi.mocked(api.whatsappStatus).mockReturnValue(new Promise(() => {}))
    vi.mocked(api.whatsappListContacts).mockReturnValue(new Promise(() => {}))
    renderSettings()
    expect(screen.getByText(/loading/i)).toBeInTheDocument()
  })

  it('renders people list after load', async () => {
    renderSettings()
    await waitFor(() => expect(screen.getByText('Rotem')).toBeInTheDocument())
    expect(screen.getByText('Ben')).toBeInTheDocument()
  })

  it('shows error when listPeople fails', async () => {
    vi.mocked(api.listPeople).mockRejectedValue(new Error('Cannot load people'))
    renderSettings()
    await waitFor(() => expect(screen.getByText('Cannot load people')).toBeInTheDocument())
  })

  it('add person form calls createPerson with trimmed name and reloads list', async () => {
    renderSettings()
    const user = userEvent.setup()
    await waitFor(() => screen.getByLabelText('New person name'))
    await user.type(screen.getByLabelText('New person name'), '  Alice  ')
    await user.click(screen.getByRole('button', { name: /add/i }))
    await waitFor(() => expect(api.createPerson).toHaveBeenCalledWith('Alice'))
    await waitFor(() => expect(api.listPeople).toHaveBeenCalledTimes(2))
  })

  it('add button disabled when input empty', async () => {
    renderSettings()
    await waitFor(() => screen.getByRole('button', { name: /add/i }))
    expect(screen.getByRole('button', { name: /add/i })).toBeDisabled()
  })

  it('delete person shows modal; confirming calls deletePerson and reloads', async () => {
    renderSettings()
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByLabelText('Remove Rotem')).toBeInTheDocument())
    await user.click(screen.getByLabelText('Remove Rotem'))
    // Modal should appear
    expect(screen.getByText(/Remove Rotem\?/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(api.deletePerson).toHaveBeenCalledWith(mockPeople[0].id))
    await waitFor(() => expect(api.listPeople).toHaveBeenCalledTimes(2))
  })

  it('delete cancelled when modal Cancel clicked', async () => {
    renderSettings()
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByLabelText('Remove Rotem')).toBeInTheDocument())
    await user.click(screen.getByLabelText('Remove Rotem'))
    expect(screen.getByText(/Remove Rotem\?/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(api.deletePerson).not.toHaveBeenCalled()
  })

  it('shows error when deletePerson fails', async () => {
    vi.mocked(api.deletePerson).mockRejectedValue(new Error('Cannot delete'))
    renderSettings()
    const user = userEvent.setup()
    await waitFor(() => expect(screen.getByLabelText('Remove Rotem')).toBeInTheDocument())
    await user.click(screen.getByLabelText('Remove Rotem'))
    await user.click(screen.getByRole('button', { name: 'Remove' }))
    await waitFor(() => expect(screen.getByText('Cannot delete')).toBeInTheDocument())
  })

  it('dark mode toggle calls onDarkChange with toggled value', async () => {
    const onDarkChange = vi.fn()
    renderSettings(false, onDarkChange)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Switch to dark mode'))
    expect(onDarkChange).toHaveBeenCalledWith(true)
  })
})
