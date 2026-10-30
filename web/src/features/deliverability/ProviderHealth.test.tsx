import { render, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { ProviderHealthCard } from './ProviderHealth'

function provider(patch: Partial<Schemas['DeliverabilityProviderHealth']>): Schemas['DeliverabilityProviderHealth'] {
  return {
    providerId: 'p-postal',
    adapter: 'postal',
    name: 'Postal (self-hosted)',
    status: 'healthy',
    badge: 'Healthy',
    tone: 'ok',
    bounce: { label: 'Bounce rate', thresholds: 'pause 5%', tone: 'ok', usage: 12, value: 0.6 },
    complaint: { label: 'Complaint rate', thresholds: 'pause 0.3%', tone: 'ok', usage: 10, value: 0.03 },
    ...patch,
  }
}

describe("a provider's rates on Deliverability (R1.L5c)", () => {
  it('compares rates with enough mail against the thresholds', () => {
    render(<ProviderHealthCard providers={[provider({})]} connected />)
    expect(screen.getByRole('meter', { name: 'Bounce rate · pause 5%: 0.6%' })).toBeInTheDocument()
    expect(screen.getByRole('meter', { name: 'Complaint rate · pause 0.3%: 0.03%' })).toBeInTheDocument()
  })

  it('shows rates on too little mail without a bar, and says why under them', () => {
    const notJudged = 'Not enough mail yet: 3 messages in the last 7 days, and rates are judged from 500.'
    render(
      <ProviderHealthCard
        providers={[
          provider({
            bounce: { label: 'Bounce rate', thresholds: 'pause 5%', tone: 'ok', usage: 0, value: 33.33 },
            complaint: { label: 'Complaint rate', thresholds: 'pause 0.3%', tone: 'ok', usage: 0, value: 0 },
            notJudged,
          }),
        ]}
        connected
      />,
    )
    expect(screen.getByText('33.33%')).toBeInTheDocument()
    expect(screen.queryByRole('meter')).not.toBeInTheDocument()
    expect(screen.getByText(notJudged)).toBeInTheDocument()
  })
})
