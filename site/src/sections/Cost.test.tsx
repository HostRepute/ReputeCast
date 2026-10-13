import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { Cost } from './Cost'

describe('the cost calculator', () => {
  it('labels every input and starts from the board defaults', () => {
    render(<Cost />)
    expect(screen.getByLabelText('Emails per month')).toHaveValue('200000')
    expect(screen.getByLabelText('Amazon SES · $ per 1,000')).toHaveValue('0.10')
    expect(screen.getByLabelText('Kit plan · $ per month')).toHaveValue('300')
    expect(screen.getByRole('heading', { name: 'Monthly sending cost at 200,000 emails' })).toBeInTheDocument()
  })

  it('recomputes as the volume and prices are edited', () => {
    render(<Cost />)
    fireEvent.change(screen.getByLabelText('Emails per month'), { target: { value: '1,000,000' } })
    expect(screen.getByRole('heading', { name: 'Monthly sending cost at 1,000,000 emails' })).toBeInTheDocument()
    const ses = screen.getByText('Amazon SES').closest('li')!
    expect(within(ses).getByText('$100.00')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Amazon SES · $ per 1,000'), { target: { value: '0.20' } })
    expect(within(screen.getByText('Amazon SES').closest('li')!).getByText('$200.00')).toBeInTheDocument()
  })

  it('announces a summary once typing stops, not on every keystroke', () => {
    vi.useFakeTimers()
    try {
      render(<Cost />)
      const status = screen.getByRole('status')
      fireEvent.change(screen.getByLabelText('Emails per month'), { target: { value: '5' } })
      fireEvent.change(screen.getByLabelText('Emails per month'), { target: { value: '50000' } })
      expect(status).toHaveTextContent('')
      act(() => {
        vi.advanceTimersByTime(1000)
      })
      expect(status).toHaveTextContent('Updated for 50,000 emails a month. Lowest: Amazon SES at $5.00.')
    } finally {
      vi.useRealTimers()
    }
  })
})
