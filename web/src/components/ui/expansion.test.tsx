import { render, screen } from '@testing-library/react'
import { TooltipProvider } from './Menu'
import { twoLineWidth } from './Table'
import { Truncate } from './Truncate'

describe('text expansion', () => {
  it('splits a header into two lines at its best space', () => {
    const chars = (s: string) => s.length
    expect(twoLineWidth('Lượt mở bởi người đọc', chars)).toBe(11)
    expect(twoLineWidth('Gửi thành công', chars)).toBe(9)
    expect(twoLineWidth('⟦Šéñðîñĝ þŕövîðéŕš ······⟧', chars)).toBe(17)
    // A single word never wraps, so it needs no width.
    expect(twoLineWidth('Recipients', chars)).toBeUndefined()
  })

  it('keeps truncated text whole for screen readers', () => {
    render(
      <TooltipProvider>
        <Truncate text="Weekly Digest #142 · the long name of a campaign">
          <a href="#c">Weekly Digest #142 · the long name of a campaign</a>
        </Truncate>
        <Truncate text="jo@contractor.dev" max={120} />
      </TooltipProvider>,
    )
    const link = screen.getByRole('link', { name: 'Weekly Digest #142 · the long name of a campaign' })
    expect(link).toHaveClass('ellipsis')
    const email = screen.getByText('jo@contractor.dev')
    expect(email).toHaveClass('ellipsis')
    expect(email.style.getPropertyValue('--ellipsis-max')).toBe('120px')
  })
})
