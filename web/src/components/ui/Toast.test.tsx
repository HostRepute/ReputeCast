import { act, fireEvent, render, screen } from '@testing-library/react'
import { dismissToast, toast, toastDuration, Toaster, updateToast } from './Toast'

describe('toasts', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => {
    act(() => dismissToast())
    vi.useRealTimers()
  })

  it('shows one at a time and closes success toasts after 6 s', () => {
    render(<Toaster />)
    act(() => void toast({ title: 'Saved as v16' }))
    act(() => void toast({ title: 'Saved as v17', detail: '09:41 · 2 blocks changed' }))
    expect(screen.queryByText('Saved as v16')).not.toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Saved as v17 · 09:41 · 2 blocks changed')
    act(() => vi.advanceTimersByTime(6000))
    expect(screen.queryByText(/Saved as v17/)).not.toBeInTheDocument()
  })

  it('offers Undo for deletes and keeps the toast 10 s', () => {
    const undo = vi.fn()
    render(<Toaster />)
    act(() => void toast({ kind: 'deleted', title: 'Deleted sam.rivera@outlook.com', detail: 'suppressions kept', undo }))
    act(() => vi.advanceTimersByTime(9000))
    fireEvent.click(screen.getByRole('button', { name: 'Undo' }))
    expect(undo).toHaveBeenCalledOnce()
    expect(screen.queryByText(/Deleted sam/)).not.toBeInTheDocument()
    expect(toastDuration({ title: 'x', undo })).toBe(10_000)
  })

  it('keeps errors until dismissed, as an alert with Retry', () => {
    const retry = vi.fn()
    render(<Toaster />)
    act(() => void toast({ kind: 'error', title: "Couldn't save v17", detail: 'server returned 502, your changes are still here', retry }))
    act(() => vi.advanceTimersByTime(60_000))
    expect(screen.getByRole('alert')).toHaveTextContent("Couldn't save v17")
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(retry).toHaveBeenCalledOnce()
  })

  it('shows progress with Hide and no dismiss cross, and can finish in place', () => {
    render(<Toaster />)
    let id = 0
    act(() => void (id = toast({ kind: 'progress', title: 'Re-running 31 checks', detail: 'blocklists take the longest, about 20 s' })))
    expect(screen.getByRole('button', { name: 'Hide' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Dismiss' })).not.toBeInTheDocument()
    act(() => updateToast(id, { kind: 'success', title: '31 checks done' }))
    expect(screen.getByRole('status')).toHaveTextContent('31 checks done')
    expect(screen.getByRole('button', { name: 'Dismiss' })).toBeInTheDocument()
  })
})
