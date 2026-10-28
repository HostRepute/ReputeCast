import { fireEvent, render, screen, within } from '@testing-library/react'
import { PassedCard } from './Issues'
import type { Check, Run } from './logic'

const spf = {
  key: 'auth.spf',
  group: 'authentication',
  status: 'pass',
  title: 'SPF authorizes both sending providers',
  fixes: [],
  raw: ['$ dig +short news.acme.io TXT', '"v=spf1 include:amazonses.com include:spf.mta.acme.io -all"'],
} as unknown as Check
const dkim = {
  key: 'auth.dkim',
  group: 'authentication',
  status: 'pass',
  title: 'DKIM signing on both providers, 2048-bit keys',
  fixes: [],
  technical: [
    { label: 'SES', value: '3 Easy DKIM CNAMEs verified' },
    { label: 'Postal', value: 'selector postal-2026 verified' },
  ],
} as unknown as Check
const footer = { key: 'compliance.footer', group: 'compliance', status: 'pass', title: 'Footer has an unsubscribe link and a postal address', fixes: [] } as unknown as Check
const run = {
  checks: [spf, dkim, footer],
  groups: [
    { key: 'authentication', title: 'Authentication', caption: '', status: 'pass', passed: 2, total: 2, summary: '' },
    { key: 'compliance', title: 'Compliance', caption: '', status: 'pass', passed: 1, total: 1, summary: '' },
  ],
} as unknown as Run

describe('technical details of passed checks (R1.U2)', () => {
  it('keeps records out of the sentence and shows them on request', async () => {
    render(<PassedCard run={run} />)
    fireEvent.click(screen.getByText('Show all 3 checks'))
    const row = (await screen.findByText('SPF authorizes both sending providers')).closest('.rv-row') as HTMLElement
    const details = within(row).getByText('Technical details').closest('details') as HTMLDetailsElement
    // The sentence carries no record: it's only in the technical details.
    const sentence = Array.from(row.querySelectorAll('.rv-rt > span'))
      .map((e) => e.textContent)
      .join(' ')
    expect(sentence).toBe('SPF authorizes both sending providers')
    expect(details).toHaveTextContent('v=spf1 include:amazonses.com include:spf.mta.acme.io -all')
    expect(details.open).toBe(false)
    fireEvent.click(within(row).getByText('Technical details'))
    expect(details.open).toBe(true)
    expect(within(row).getByText(/^\$ dig \+short news\.acme\.io TXT/)).toHaveTextContent('"v=spf1 include:amazonses.com include:spf.mta.acme.io -all"')
    // DKIM: each provider's keys.
    const keys = screen.getByText('DKIM signing on both providers, 2048-bit keys').closest('.rv-row') as HTMLElement
    expect(within(keys).getByText('selector postal-2026 verified')).toBeInTheDocument()
    // A check with nothing technical offers no details.
    const plain = screen.getByText('Footer has an unsubscribe link and a postal address').closest('.rv-row') as HTMLElement
    expect(within(plain).queryByText('Technical details')).not.toBeInTheDocument()
  })
})
