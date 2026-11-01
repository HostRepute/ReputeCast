import { act, renderHook } from '@testing-library/react'
import { useOptimistic } from './hooks'

// Review 2026-11 FE-7b: a switch shows its new state at once and goes back
// when the save fails.
describe('useOptimistic', () => {
  it('shows the new value while saving and keeps it once saved', async () => {
    let finish: (ok: boolean) => void = () => {}
    const { result, rerender } = renderHook(({ server }) => useOptimistic(server), { initialProps: { server: false } })
    let saved: Promise<boolean> = Promise.resolve(false)
    act(() => {
      saved = result.current[1](true, () => new Promise<boolean>((r) => (finish = r)))
    })
    expect(result.current[0]).toBe(true)
    await act(async () => {
      finish(true)
      await saved
    })
    expect(result.current[0]).toBe(true)
    // The server catches up; its value is shown from then on.
    rerender({ server: true })
    expect(result.current[0]).toBe(true)
    rerender({ server: false })
    expect(result.current[0]).toBe(false)
  })

  it('goes back when the save fails or throws', async () => {
    const { result } = renderHook(() => useOptimistic(false))
    await act(async () => {
      expect(await result.current[1](true, async () => false)).toBe(false)
    })
    expect(result.current[0]).toBe(false)
    await act(async () => {
      expect(await result.current[1](true, () => Promise.reject(new Error('refused')))).toBe(false)
    })
    expect(result.current[0]).toBe(false)
  })
})
