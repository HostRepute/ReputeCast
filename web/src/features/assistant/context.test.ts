import { describe, expect, it } from 'vitest'
import { screenOf, withSuggestion } from './context'

describe('the screen the assistant opens on', () => {
  it('suggests the draft in the composer, as on its review', () => {
    expect(screenOf('/w/acme/campaigns/c1/edit')).toEqual({ screen: 'campaign.edit', kind: 'campaign', resource: { kind: 'campaign', id: 'c1' } })
    expect(screenOf('/w/acme/campaigns/c1/review').resource).toEqual({ kind: 'campaign', id: 'c1' })
  })

  it("takes a contextual button's resource on the page it was pressed on only", () => {
    const report = screenOf('/w/acme/campaigns/c1/report')
    const suggest = { kind: 'campaign' as const, id: 'c2', path: '/w/acme/campaigns/c1/report' }
    expect(withSuggestion(report, suggest, '/w/acme/campaigns/c1/report')).toEqual({ ...report, kind: 'campaign', resource: { kind: 'campaign', id: 'c2' } })
    // Elsewhere the route's own suggestion stands, and nothing is attached either way.
    expect(withSuggestion(report, suggest, '/w/acme/campaigns')).toBe(report)
    expect(withSuggestion(report, undefined, '/w/acme/campaigns/c1/report')).toBe(report)
  })
})
