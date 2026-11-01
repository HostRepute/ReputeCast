import { render, screen } from '@testing-library/react'
import { StepTrail, Stepper } from './Stepper'

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

  it('draws a dialog\'s steps on one line: one list item per step, the separators hidden', () => {
    render(
      <StepTrail
        label="Connect steps"
        current={2}
        blocked
        steps={[
          { key: 'a', title: 'Store address' },
          { key: 'b', title: 'Create the app' },
          { key: 'c', title: 'Paste the token' },
        ]}
      />,
    )
    const list = screen.getByRole('list', { name: 'Connect steps' })
    const items = screen.getAllByRole('listitem')
    expect(items).toHaveLength(3)
    expect(items[0]).toHaveTextContent('Done: Store address')
    expect(items[1]).toHaveAttribute('aria-current', 'step')
    expect(items[1]).not.toHaveTextContent('Done:')
    expect(screen.getByText('Paste the token')).toHaveAttribute('aria-disabled', 'true')
    expect(screen.getByText('Create the app')).not.toHaveAttribute('aria-disabled')
    for (const sep of list.querySelectorAll('.sep')) expect(sep).toHaveAttribute('aria-hidden', 'true')
    expect(list.querySelectorAll('.sep')).toHaveLength(2)
  })
})
