import { fireEvent, render, screen } from '@testing-library/react'
import { Pager } from './Pager'

describe('Pager (review 2026-11 FE-3b)', () => {
  it('fetches the next page, says it is loading, and goes once the last page is in', () => {
    const fetchNextPage = vi.fn(() => Promise.resolve())
    const { rerender } = render(<Pager query={{ hasNextPage: true, isFetchingNextPage: false, fetchNextPage }} label="Load 50 more" />)
    fireEvent.click(screen.getByRole('button', { name: 'Load 50 more' }))
    expect(fetchNextPage).toHaveBeenCalledTimes(1)
    rerender(<Pager query={{ hasNextPage: true, isFetchingNextPage: true, fetchNextPage }} label="Load 50 more" />)
    expect(screen.getByRole('button')).toHaveAttribute('aria-busy', 'true')
    fireEvent.click(screen.getByRole('button'))
    expect(fetchNextPage).toHaveBeenCalledTimes(1)
    rerender(<Pager query={{ hasNextPage: false, isFetchingNextPage: false, fetchNextPage }} label="Load 50 more" />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })
})
