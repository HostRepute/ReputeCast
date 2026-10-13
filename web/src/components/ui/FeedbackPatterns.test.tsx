import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { Button } from './Button'
import { DisabledReason } from './DisabledReason'
import { ErrorSummary } from './ErrorSummary'
import { Field, Input } from './Field'
import { TooltipProvider } from './Menu'
import { NoResults } from './NoResults'

describe('ErrorSummary', () => {
  it('takes focus, counts the problems and moves focus to each field', () => {
    render(
      <>
        <ErrorSummary
          errors={[
            { field: 'reply-to', message: 'Reply-to address needs a domain' },
            { field: 'sender', message: 'Sender domain already exists' },
            { field: 'webhook', message: 'Webhook URL must be https' },
          ]}
        />
        <input id="reply-to" aria-label="Reply-to address" />
        <input id="sender" aria-label="Sender domain" />
        <input name="webhook" aria-label="Webhook URL" />
      </>,
    )
    const summary = screen.getByRole('alert')
    expect(summary).toHaveFocus()
    expect(summary).toHaveAccessibleName('3 things to fix before saving')
    fireEvent.click(screen.getByRole('link', { name: 'Sender domain already exists' }))
    expect(screen.getByLabelText('Sender domain')).toHaveFocus()
    fireEvent.click(screen.getByRole('link', { name: 'Webhook URL must be https' }))
    expect(screen.getByLabelText('Webhook URL')).toHaveFocus()
  })

  it('renders nothing without errors', () => {
    const { container } = render(<ErrorSummary errors={[]} />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('Field validation states', () => {
  it('says what passed in green and what failed next to the field', () => {
    render(
      <>
        <Field label="Daily cap" success="Within the SES quota for us-east-1 (50,000 per 24 h).">
          <Input defaultValue="50,000" />
        </Field>
        <Field label="Webhook URL" error="Must be https.">
          <Input defaultValue="http://hooks.acme.io/rc" />
        </Field>
      </>,
    )
    const cap = screen.getByLabelText('Daily cap')
    expect(cap).toHaveClass('okb')
    expect(cap).toHaveAccessibleDescription('Within the SES quota for us-east-1 (50,000 per 24 h).')
    const hook = screen.getByLabelText('Webhook URL')
    expect(hook).toHaveAttribute('aria-invalid', 'true')
    expect(hook).not.toHaveClass('okb')
  })
})

describe('DisabledReason', () => {
  it('keeps the control visible and focusable, explains why, and ignores clicks', () => {
    const onClick = vi.fn()
    render(
      <TooltipProvider>
        <DisabledReason reason="Editors can't change routing · ask Linh or Marcus">
          <Button onClick={onClick}>Save routing</Button>
        </DisabledReason>
      </TooltipProvider>,
    )
    const button = screen.getByRole('button', { name: 'Save routing' })
    expect(button).toHaveAttribute('aria-disabled', 'true')
    expect(button).toHaveAccessibleDescription("Editors can't change routing · ask Linh or Marcus")
    fireEvent.click(button)
    expect(onClick).not.toHaveBeenCalled()
  })

  it('shows the reason in a bubble on focus', async () => {
    render(<Button disabledReason="Editors can't change routing · ask Linh or Marcus">Save routing</Button>)
    fireEvent.focus(screen.getByRole('button', { name: 'Save routing' }))
    await waitFor(() => expect(document.querySelector('.tip.why')).toHaveTextContent("Editors can't change routing · ask Linh or Marcus"))
  })

  it('keeps the same button, and focus, when the reason goes away', () => {
    const { rerender } = render(<Button disabledReason="Type acme-newsletter to confirm.">Delete workspace</Button>)
    const button = screen.getByRole('button', { name: 'Delete workspace' })
    button.focus()
    expect(button).toHaveAccessibleDescription('Type acme-newsletter to confirm.')
    rerender(<Button disabledReason={undefined}>Delete workspace</Button>)
    expect(screen.getByRole('button', { name: 'Delete workspace' })).toBe(button)
    expect(button).toHaveFocus()
    expect(button).not.toHaveAttribute('aria-disabled')
    expect(button).not.toHaveAccessibleDescription()
  })

  it('leaves an allowed control alone', () => {
    render(
      <DisabledReason reason={undefined}>
        <Button>Save routing</Button>
      </DisabledReason>,
    )
    expect(screen.getByRole('button', { name: 'Save routing' })).not.toHaveAttribute('aria-disabled')
  })
})

describe('Button busyLabel', () => {
  it('swaps the label without changing the width', () => {
    const { rerender } = render(<Button busyLabel="Saving…">Save</Button>)
    expect(screen.getByRole('button')).toHaveAccessibleName('Save')
    rerender(
      <Button busyLabel="Saving…" loading>
        Save
      </Button>,
    )
    const busy = screen.getByRole('button')
    expect(busy).toHaveAccessibleName('Saving…')
    expect(busy).toHaveAttribute('aria-busy', 'true')
    // Both labels stay in the layout; the idle one is kept invisible.
    expect(busy.querySelector('.reserve')).toHaveTextContent('Save')
  })
})

describe('NoResults', () => {
  it('says what was searched, why nothing matched and the ways out', () => {
    render(
      <NoResults title='No contacts match "acme.io" in Beta Program' actions={<Button>Clear filters</Button>}>
        2,115 contacts are on Beta Program; none has an acme.io address.
      </NoResults>,
    )
    expect(screen.getByRole('status')).toHaveTextContent('No contacts match "acme.io" in Beta Program')
    expect(screen.getByRole('heading', { name: 'No contacts match "acme.io" in Beta Program' })).toBeInTheDocument()
  })
})
