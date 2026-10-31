import { act, fireEvent, render, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { dismissToast, toast, Toaster } from '@/components/ui/Toast'
import { setCatalogLoader, setLocale } from '@/i18n'
import vietnamese from './messages.vi'
import { fixesToast, undoReason } from './parts'
import { AppliedFixes, RunningBanner } from './Verdict'

const today = new Date()
today.setHours(9, 46, 0, 0)

function batch(patch: Partial<Schemas['ReadinessFixBatch']> = {}): Schemas['ReadinessFixBatch'] {
  return {
    id: '0199f0a4-0000-7000-8000-000000000001',
    appliedAt: today.toISOString(),
    appliedBy: 'Linh Tran',
    applied: ['This send goes 100% via Amazon SES', 'Added utm_campaign=weekly-digest-142 to 2 links'],
    count: 2,
    version: 15,
    undoable: true,
    ...patch,
  }
}

describe('Undo for fixes applied together (R1.U3)', () => {
  afterEach(async () => {
    await setLocale('en')
  })

  it('says who applied which fixes, when, and the version they saved', () => {
    render(<AppliedFixes batch={batch()} me="Linh Tran" undo={<Button>Undo these fixes</Button>} />)
    expect(
      screen.getByText('You applied 2 fixes at 09:46, saved as version 15: This send goes 100% via Amazon SES · Added utm_campaign=weekly-digest-142 to 2 links.'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Undo these fixes' })).toBeInTheDocument()
  })

  it('names someone else, and reads how many when the fixes were applied in another language', () => {
    render(<AppliedFixes batch={batch({ applied: [], count: 1, version: undefined })} me="Marcus Chen" />)
    expect(screen.getByText('Linh Tran applied 1 fix at 09:46.')).toBeInTheDocument()
  })

  it('shows the fixes under the re-running banner, with the lock', () => {
    render(
      <RunningBanner
        progress={{ done: 7, total: 11 } as Schemas['ReadinessProgress']}
        lock="Sending is locked while the checks run again after your fix."
        applied={<AppliedFixes batch={batch()} me="Linh Tran" undo={<Button>Undo these fixes</Button>} />}
      />,
    )
    const banner = screen.getByRole('status')
    expect(banner).toHaveTextContent('Running Send Readiness · 7 of 11 checks done')
    expect(banner).toHaveTextContent('You applied 2 fixes at 09:46, saved as version 15')
    expect(screen.getByRole('button', { name: 'Undo these fixes' })).toBeInTheDocument()
  })

  it('offers Undo in the toast after the fixes, and keeps it 10 s', () => {
    const undo = vi.fn()
    render(<Toaster />)
    let id = 0
    act(() => void (id = toast(fixesToast(['This send goes 100% via Amazon SES', 'Added utm_campaign=weekly-digest-142 to 2 links'], undo))))
    expect(screen.getByRole('status')).toHaveTextContent('2 fixes applied')
    fireEvent.click(screen.getByRole('button', { name: 'Undo' }))
    expect(undo).toHaveBeenCalledOnce()
    act(() => dismissToast(id))
    // Without an undo (the fixes changed a language version), no Undo.
    act(() => void (id = toast(fixesToast(['Marked Vietnamese as up to date']))))
    expect(screen.getByRole('status')).toHaveTextContent('Marked Vietnamese as up to date')
    expect(screen.queryByRole('button', { name: 'Undo' })).not.toBeInTheDocument()
    act(() => dismissToast(id))
  })

  it('lets people undo what they could apply: a route change needs the right to send', () => {
    const can = { edit: true, send: true, editable: true }
    const helpers = { send: ['Marcus Chen'], edit: ['Linh Tran'] }
    expect(undoReason({ route: true }, can, helpers)).toBeUndefined()
    expect(undoReason({ route: true }, { ...can, send: false }, helpers)).toBe('Marcus Chen can change the route.')
    expect(undoReason({}, { ...can, send: false }, helpers)).toBeUndefined()
    expect(undoReason({}, { ...can, edit: false }, helpers)).toBe('Linh Tran can change it.')
    expect(undoReason({}, { ...can, editable: false }, helpers)).toBe('This campaign already started sending, so it can’t change.')
  })

  it('reads in Vietnamese', async () => {
    setCatalogLoader('vi', async () => vietnamese)
    await setLocale('vi')
    render(<AppliedFixes batch={batch()} me="Linh Tran" />)
    expect(
      screen.getByText(
        'Bạn đã áp dụng 2 bản sửa lúc 09:46, lưu thành phiên bản 15: This send goes 100% via Amazon SES · Added utm_campaign=weekly-digest-142 to 2 links.',
      ),
    ).toBeInTheDocument()
  })
})
