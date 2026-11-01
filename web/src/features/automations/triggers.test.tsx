import { render, screen } from '@testing-library/react'
import type { Schemas } from '@/api/client'
import { NewKindFields } from './triggers'

const options = {
  segments: [
    { id: 'quick', name: 'Engaged readers', checkMinutes: 10, evalSeconds: 1 },
    { id: 'slow', name: 'Big SQL segment', checkMinutes: 30, evalSeconds: 90 },
  ],
  campaigns: [],
} as unknown as Schemas['AutomationOptions']

// Review 2026-11 AT-15: the trigger says when its segment is checked less often.
describe('Enters a segment', () => {
  it('says a slow segment is checked less often', () => {
    const { rerender } = render(<NewKindFields t={{ kind: 'segment_entered', segmentId: 'quick' }} set={() => {}} options={options} disabled={false} />)
    expect(screen.queryByText(/checked every 30 minutes/)).not.toBeInTheDocument()
    rerender(<NewKindFields t={{ kind: 'segment_entered', segmentId: 'slow' }} set={() => {}} options={options} disabled={false} />)
    expect(screen.getByText('This segment takes 90 seconds to evaluate, so it is checked every 30 minutes instead of 10. Contacts enter the flow up to that long after they join it.')).toBeInTheDocument()
  })
})
