import { act, fireEvent, render, renderHook, screen } from '@testing-library/react'
import { SelectAll } from '@/components/ui/SelectAll'
import { useSelection } from './selection'

describe('One bulk-selection pattern (review 2026-11 FE-3b)', () => {
  it('keeps only rows still shown, in the order shown', () => {
    const { result, rerender } = renderHook(({ ids }) => useSelection(ids), { initialProps: { ids: ['a', 'b', 'c'] } })
    act(() => result.current.toggle('c', true))
    act(() => result.current.toggle('a', true))
    expect(result.current.selected).toEqual(['a', 'c'])
    expect(result.current.some).toBe(true)
    expect(result.current.all).toBe(false)
    // Another page or filter: rows out of view drop out for good.
    rerender({ ids: ['b', 'c'] })
    expect(result.current.selected).toEqual(['c'])
    rerender({ ids: ['a', 'b', 'c'] })
    expect(result.current.selected).toEqual(['c'])
    act(() => result.current.setMany(['a', 'b', 'c'], true))
    expect(result.current.all).toBe(true)
    act(() => result.current.clear())
    expect(result.current.some).toBe(false)
  })

  it('draws the page checkbox mixed while some rows are picked', () => {
    const onChange = vi.fn()
    const { rerender } = render(<SelectAll all={false} some label="Select every campaign on this page" onChange={onChange} />)
    const box = screen.getByRole('checkbox', { name: 'Select every campaign on this page' }) as HTMLInputElement
    expect(box.indeterminate).toBe(true)
    fireEvent.click(box)
    expect(onChange).toHaveBeenCalledWith(true)
    rerender(<SelectAll all some label="Select every campaign on this page" onChange={onChange} />)
    expect(box.indeterminate).toBe(false)
    expect(box).toBeChecked()
  })
})
