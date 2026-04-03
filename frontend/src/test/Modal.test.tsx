import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import Modal from '../components/Modal'

function renderModal(onClose = vi.fn()) {
  return render(
    <Modal title="Test Modal" onClose={onClose}>
      <p>Modal content</p>
    </Modal>
  )
}

describe('Modal', () => {
  it('renders title and children', () => {
    renderModal()
    expect(screen.getByText('Test Modal')).toBeInTheDocument()
    expect(screen.getByText('Modal content')).toBeInTheDocument()
  })

  it('close button calls onClose', async () => {
    const onClose = vi.fn()
    renderModal(onClose)
    const user = userEvent.setup()
    await user.click(screen.getByLabelText('Close'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('backdrop click calls onClose', async () => {
    const onClose = vi.fn()
    renderModal(onClose)
    const user = userEvent.setup()
    // The backdrop is the absolute div behind the panel
    const backdrop = document.querySelector('.absolute.inset-0') as HTMLElement
    await user.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Escape key calls onClose', async () => {
    const onClose = vi.fn()
    renderModal(onClose)
    const user = userEvent.setup()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('has role="dialog" and aria-modal', () => {
    renderModal()
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(dialog).toHaveAttribute('aria-modal', 'true')
  })
})
