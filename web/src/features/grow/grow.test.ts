import { describe, expect, it } from 'vitest'
import { trackPos } from './MilestoneTrack'
import { tileInitials } from './shared'

describe('the milestone track', () => {
  // Milestones sit evenly: 1, 3 and 10 at a third, two thirds and the end.
  const counts = [1, 3, 10]
  it('puts a count between its milestones', () => {
    expect(trackPos(counts, 0)).toBe(0)
    expect(trackPos(counts, 1)).toBeCloseTo(33.33, 1)
    // Sam: 2 counted is half way from 1 to 3.
    expect(trackPos(counts, 2)).toBeCloseTo(50, 1)
    expect(trackPos(counts, 3)).toBeCloseTo(66.67, 1)
    expect(trackPos(counts, 10)).toBe(100)
    expect(trackPos(counts, 23)).toBe(100)
  })
  it('handles no milestones', () => {
    expect(trackPos([], 5)).toBe(0)
  })
})

describe('newsletter tiles', () => {
  it('uses two initials without a leading "The"', () => {
    expect(tileInitials('Ops Weekly')).toBe('OW')
    expect(tileInitials('The Deliverability Desk')).toBe('DD')
    expect(tileInitials('Fabrikam Engineering Letter')).toBe('FE')
  })
})
