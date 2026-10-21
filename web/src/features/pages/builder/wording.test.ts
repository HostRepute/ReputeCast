import { t } from '@/i18n'
import { setDisplayTimeZone } from '@/lib/format'
import { dayTimeText, livePlaceholder, localTime, wallText, whenText, zoneLabel } from './wording'

describe('dates as the builder says them', () => {
  beforeEach(() => setDisplayTimeZone('America/New_York'))
  afterAll(() => setDisplayTimeZone(undefined))

  it('shows a countdown’s or an event’s time on its own clock, with the zone’s city', () => {
    expect(wallText('2026-10-06', '09:00', 'America/New_York')).toEqual({ day: '6 Oct', time: '09:00', zone: 'New York' })
    // The same wall clock time, whatever the editor's own zone.
    setDisplayTimeZone('Asia/Ho_Chi_Minh')
    expect(wallText('2026-10-06', '09:00', 'America/New_York')).toEqual({ day: '6 Oct', time: '09:00', zone: 'New York' })
    expect(wallText('2026-11-12', '17:00', 'UTC')).toEqual({ day: '12 Nov', time: '17:00', zone: 'UTC' })
  })

  it('says when the draft was saved: the time today, the day before', () => {
    const now = new Date('2026-10-19T16:00:00Z')
    expect(whenText(t, '2026-10-19T14:41:00Z', now)).toBe('10:41')
    expect(whenText(t, '2026-10-08T13:12:00Z', now)).toBe('8 Oct, 09:12')
    expect(whenText(t, '2026-10-08T13:12:00Z', now, true)).toBe('8 Oct')
    expect(whenText(t, 'not a date', now)).toBe('')
    expect(dayTimeText(t, '2026-10-19T14:41:00Z')).toBe('19 Oct, 10:41')
  })

  it('gives an event’s time in the editor’s zone, as readers see theirs', () => {
    expect(localTime(new Date('2026-11-12T17:00:00Z'), new Date('2026-11-12T17:30:00Z'))).toEqual({ time: '12:00–12:30', city: 'New York' })
    setDisplayTimeZone('UTC')
    expect(localTime(new Date('2026-11-12T17:00:00Z'), new Date('2026-11-12T17:30:00Z'))).toBeNull()
  })

  it('names time zones with their short name on the day, UTC alone', () => {
    const day = new Date('2026-10-06T12:00:00Z')
    expect(zoneLabel(t, 'America/New_York', day)).toBe('America/New_York · EDT')
    expect(zoneLabel(t, 'UTC', day)).toBe('UTC')
  })

  it('says what an ended countdown leaves on the page', () => {
    const at = { at: '2026-10-06T09:00', tz: 'America/New_York' }
    expect(livePlaceholder(t, 'rc-countdown', { ...at, end: 'reveal' }, 'empty')).toBe('<p class="pg-empty">The countdown ended 6 Oct, 09:00 New York. Readers see what it revealed.</p>')
    expect(livePlaceholder(t, 'rc-countdown', { ...at, end: 'hide' }, 'empty')).toBe('<p class="pg-empty">The countdown ended 6 Oct, 09:00 New York. Readers no longer see it.</p>')
    expect(livePlaceholder(t, 'rc-product', {}, 'loading')).toBe('<p class="pg-empty">Drawing the block…</p>')
  })
})
