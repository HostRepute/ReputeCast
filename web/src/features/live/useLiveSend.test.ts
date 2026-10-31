import { isStale, MAX_ATTEMPTS, newest, POLL_EVERY, retryDelay, STALE_AFTER } from './useLiveSend'
import type { Snapshot } from './model'

// The stream's reconnection and stale-data rules (LiveMonitorStates; review GS-14).
describe('the live stream', () => {
  it('backs off 2, 4, 8, 16 s, then every 30 s, ten times', () => {
    expect([1, 2, 3, 4, 5, 9].map(retryDelay)).toEqual([2000, 4000, 8000, 16000, 30000, 30000])
    expect(MAX_ATTEMPTS).toBe(10)
  })

  it('calls an open stream stale after 90 s of silence and then checks every 10 s', () => {
    const heard = new Date('2026-10-14T09:38:44Z')
    expect(isStale(heard, new Date('2026-10-14T09:40:14Z'))).toBe(false)
    expect(isStale(heard, new Date('2026-10-14T09:40:15Z'))).toBe(true)
    expect(isStale(undefined, new Date())).toBe(false)
    expect([STALE_AFTER, POLL_EVERY]).toEqual([90_000, 10_000])
  })

  it('keeps the newer snapshot of the same send', () => {
    const a = { campaignId: 'c', version: 4 } as Snapshot
    const b = { campaignId: 'c', version: 5 } as Snapshot
    const other = { campaignId: 'd', version: 9 } as Snapshot
    expect(newest(a, b)).toBe(b)
    expect(newest(b, a)).toBe(b)
    expect(newest(a, other)).toBe(a)
    expect(newest(undefined, a)).toBe(a)
    expect(newest(null, null)).toBeNull()
  })
})
