import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import FilterSelect from '../components/FilterSelect'

const options = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'done', label: 'Done' },
]

function renderSelect(value = 'all', onChange = vi.fn()) {
  return render(<FilterSelect value={value} options={options} onChange={onChange} />)
}

describe('FilterSelect', () => {
  it('renders selected option label', () => {
    renderSelect('pending')
    expect(screen.getByText('Pending')).toBeInTheDocument()
  })

  it('click opens dropdown', async () => {
    renderSelect()
    const user = userEvent.setup()
    const button = screen.getByRole('button')
    await user.click(button)
    expect(screen.getAllByRole('button').length).toBeGreaterThan(1)
  })

  it('shows all options in dropdown', async () => {
    renderSelect()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button'))
    // All three options should appear in the dropdown buttons
    const buttons = screen.getAllByRole('button')
    const labels = buttons.map(b => b.textContent)
    expect(labels.some(l => l?.includes('All'))).toBe(true)
    expect(labels.some(l => l?.includes('Pending'))).toBe(true)
    expect(labels.some(l => l?.includes('Done'))).toBe(true)
  })

  it('clicking option calls onChange and closes dropdown', async () => {
    const onChange = vi.fn()
    renderSelect('all', onChange)
    const user = userEvent.setup()
    await user.click(screen.getByRole('button'))
    const pendingBtn = screen.getAllByText('Pending')[0]
    await user.click(pendingBtn)
    expect(onChange).toHaveBeenCalledWith('pending')
    // Dropdown should close - only one button (the trigger) remains
    expect(screen.getAllByRole('button')).toHaveLength(1)
  })

  it('clicking outside closes dropdown', async () => {
    renderSelect()
    const user = userEvent.setup()
    await user.click(screen.getByRole('button'))
    expect(screen.getAllByRole('button').length).toBeGreaterThan(1)
    // Click outside
    await user.click(document.body)
    expect(screen.getAllByRole('button')).toHaveLength(1)
  })

  it('selected option has visual indicator (font-medium or bg)', async () => {
    renderSelect('pending')
    const user = userEvent.setup()
    await user.click(screen.getByRole('button'))
    const buttons = screen.getAllByRole('button')
    // Find the button with "Pending" text (there are two now)
    const pendingButtons = buttons.filter(b => b.textContent?.includes('Pending'))
    // One should have font-medium or bg-slate class
    const hasIndicator = pendingButtons.some(
      b => b.className.includes('font-medium') || b.className.includes('bg-slate')
    )
    expect(hasIndicator).toBe(true)
  })
})
