import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { ApiError } from '@/api/client'
import { BulkResultDialog, ConfirmDialog, DialogOption, DialogRow } from './ConfirmDialog'

function DeleteList({ onConfirm }: { onConfirm: () => Promise<unknown> | void }) {
  const [open, setOpen] = useState(true)
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={setOpen}
      badge="Delete list"
      title='Delete "Product Updates"?'
      description="14,110 contacts are on this list. Suppressions are never removed."
      confirmLabel="Delete list"
      onConfirm={onConfirm}
      typedConfirmation={{ label: 'Type the list name to confirm', expected: 'Product Updates', hint: 'Deleted lists can be restored from the audit log for 30 days.' }}
    >
      <DialogRow label="Affects" first>
        2 signup forms, 1 automation trigger and 3 scheduled campaigns use this list. They will pause.
      </DialogRow>
    </ConfirmDialog>
  )
}

describe('ConfirmDialog', () => {
  it('states the consequences first and asks for the name before deleting', async () => {
    const onConfirm = vi.fn()
    render(<DeleteList onConfirm={onConfirm} />)
    const dialog = screen.getByRole('dialog', { name: 'Delete "Product Updates"?' })
    expect(dialog).toHaveTextContent('Delete list')
    expect(dialog).toHaveTextContent('Affects')
    const button = screen.getByRole('button', { name: 'Delete list' })
    expect(button).toHaveAttribute('aria-disabled', 'true')
    fireEvent.click(button)
    expect(onConfirm).not.toHaveBeenCalled()

    const field = screen.getByLabelText('Type the list name to confirm')
    expect(field).toHaveAccessibleDescription('Deleted lists can be restored from the audit log for 30 days.')
    fireEvent.change(field, { target: { value: 'Product Upd' } })
    expect(button).toHaveAttribute('aria-disabled', 'true')
    fireEvent.change(field, { target: { value: 'Product Updates' } })
    expect(button).not.toHaveAttribute('aria-disabled')
    fireEvent.click(button)
    await waitFor(() => expect(onConfirm).toHaveBeenCalledOnce())
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('keeps the dialog open with the reason when the action fails', async () => {
    const onConfirm = vi.fn().mockRejectedValue(new ApiError(409, { status: 409, detail: 'The list is used by a running automation. Pause it first.' } as never))
    render(<DeleteList onConfirm={onConfirm} />)
    fireEvent.change(screen.getByLabelText('Type the list name to confirm'), { target: { value: 'Product Updates' } })
    fireEvent.click(screen.getByRole('button', { name: 'Delete list' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('The list is used by a running automation. Pause it first.')
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('requires a reason when asked, and passes it on', async () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        open
        onOpenChange={() => {}}
        tone="warn"
        badge="Emergency bypass"
        title="Send without approval?"
        description="The policy needs an approver. Bypassing is logged with your name."
        confirmLabel="Send now"
        onConfirm={onConfirm}
        reason={{ label: 'Why it can’t wait', minLength: 10 }}
      />,
    )
    const send = screen.getByRole('button', { name: 'Send now' })
    expect(send).toHaveAttribute('aria-disabled', 'true')
    fireEvent.change(screen.getByLabelText('Why it can’t wait'), { target: { value: 'Outage notice to customers' } })
    fireEvent.click(send)
    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith({ reason: 'Outage notice to customers' }))
  })

  it('renders choices as labelled radios', () => {
    render(
      <ConfirmDialog open onOpenChange={() => {}} badge="Cancel send" title="Cancel the rest of Weekly Digest #142?" description="19,594 of 31,604 are already delivered." confirmLabel="Cancel remaining sends" onConfirm={() => {}}>
        <DialogOption name="c" checked onChange={() => {}} title="Cancel the remaining 12,010" description="Marks the campaign as partially sent." />
        <DialogOption name="c" checked={false} onChange={() => {}} title="Pause instead" description="Keeps the queue for 24 hours." />
      </ConfirmDialog>,
    )
    expect(screen.getByRole('radio', { name: /Cancel the remaining 12,010/ })).toBeChecked()
    expect(screen.getByRole('radio', { name: /Pause instead/ })).not.toBeChecked()
  })
})

describe('BulkResultDialog', () => {
  it('reports what was done and skipped, with Undo for the done part', async () => {
    const onUndo = vi.fn()
    render(
      <BulkResultDialog
        open
        onOpenChange={() => {}}
        title='Moved 2,114 of 2,118 contacts to "Re-permission"'
        description="4 could not be moved: 2 are hard-bounced and stay suppressed, 2 were deleted by another member while this ran."
        done={{ count: 2114, text: 'Tagged re-permission, removed from Broadcasts until they confirm' }}
        skipped={{ count: 4, text: 'Skipped' }}
        onUndo={onUndo}
      />,
    )
    const dialog = screen.getByRole('dialog')
    expect(dialog).toHaveTextContent('Bulk action · partial result')
    expect(dialog).toHaveTextContent('2,114Tagged re-permission')
    fireEvent.click(screen.getByRole('button', { name: 'Undo for the 2,114' }))
    await waitFor(() => expect(onUndo).toHaveBeenCalledOnce())
  })
})
