import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { Dialog } from './Dialog'

function Opener({ mounted }: { mounted?: boolean }) {
  const [open, setOpen] = useState(false)
  const dialog = <Dialog open={open} onOpenChange={setOpen} title="Change phone" />
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        Change phone
      </button>
      {mounted ? dialog : open ? dialog : null}
    </>
  )
}

describe('Dialog', () => {
  // Most dialogs open from a plain button, not a Radix trigger.
  for (const mounted of [true, false]) {
    it(`gives focus back to the button that opened it (${mounted ? 'kept mounted' : 'mounted on open'})`, async () => {
      render(<Opener mounted={mounted} />)
      const button = screen.getByRole('button', { name: 'Change phone' })
      button.focus()
      fireEvent.click(button)
      const dialog = await screen.findByRole('dialog', { name: 'Change phone' })
      await waitFor(() => expect(dialog.contains(document.activeElement)).toBe(true))
      fireEvent.keyDown(dialog, { key: 'Escape' })
      await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
      expect(document.activeElement).toBe(button)
    })
  }
})
