import { describe, expect, it } from 'vitest'
import { testState, TEST_WAIT_MS, type InboundRoute } from './ReceivingCard'

const route = (sentAt?: string, receivedAt?: string) => ({ lastTest: { sentAt, receivedAt } }) as unknown as InboundRoute

// The test message on the Receiving mail card (InboundStates): waiting while
// it's on its way, received once it's back, "Not received yet" after 2
// minutes, and nothing once the result is old.
describe('testState', () => {
  const sent = Date.parse('2026-11-12T09:41:00Z')
  it('waits for two minutes', () => {
    expect(testState(route(new Date(sent).toISOString()), sent + 3000)).toBe('waiting')
    expect(testState(route(new Date(sent).toISOString()), sent + TEST_WAIT_MS + 1)).toBe('missing')
  })
  it('says received when the test came back after it was sent', () => {
    expect(testState(route(new Date(sent).toISOString(), new Date(sent + 9000).toISOString()), sent + 10000)).toBe('received')
    // A receipt from an earlier test doesn't count.
    expect(testState(route(new Date(sent).toISOString(), new Date(sent - 60000).toISOString()), sent + 10000)).toBe('waiting')
  })
  it('shows nothing without a test or for an old one', () => {
    expect(testState(route(), sent)).toBe('none')
    expect(testState(undefined, sent)).toBe('none')
    expect(testState(route(new Date(sent).toISOString(), new Date(sent + 9000).toISOString()), sent + 11 * 60 * 1000)).toBe('none')
  })
})
