import { render, screen } from '@testing-library/react'
import { Stepper } from './Stepper'

describe('Stepper (review 2026-11 FE-3b)', () => {
  it('marks the current step, reads finished ones as done, and shows each line and the foot', () => {
    render(
      <Stepper
        label="Setup steps"
        current={2}
        steps={[
          { key: 'a', title: 'Provider', line: 'Amazon SES' },
          { key: 'b', title: 'Domain', line: 'Add three records' },
          { key: 'c', title: 'Test' },
        ]}
        foot="About ten minutes"
      />,
    )
    const nav = screen.getByRole('navigation', { name: 'Setup steps' })
    const [first, second, third] = nav.querySelectorAll('.step')
    expect(first).toHaveTextContent('Done: Provider')
    expect(first).toHaveTextContent('Amazon SES')
    expect(first).not.toHaveAttribute('aria-current')
    expect(second).toHaveAttribute('aria-current', 'step')
    expect(second).toHaveTextContent('2')
    expect(third).toHaveTextContent('3Test')
    expect(screen.getByText('About ten minutes')).toBeInTheDocument()
  })
})
