import { approvalMode, heldNote, heldSentence } from './ApprovalCard'
import {
  acknowledgedKeys,
  acknowledgements,
  agoWords,
  blockedLine,
  confirmNote,
  countsLine,
  countsShort,
  fixAllLabel,
  linkParts,
  localDateWords,
  minutes,
  parsePercent,
  primaryFix,
  scheduleSentence,
  scheduleShort,
  sendLabel,
  sendsIn,
  thresholdError,
  waitShort,
  waitWords,
  type Check,
  type Review,
  type Run,
  type Thresholds,
} from './logic'

const check = (over: Partial<Check>): Check => ({ key: 'k', group: 'content', status: 'pass', title: 'A check', fixes: [], ...over }) as Check

describe('verdict copy', () => {
  it('counts checks the way the banners say them', () => {
    expect(countsLine({ passed: 23, warnings: 2, blocks: 0, unknown: 0, total: 25 })).toBe('23 checks passed, 2 warnings, 0 blocks')
    expect(countsLine({ passed: 1, warnings: 1, blocks: 1, unknown: 0, total: 3 })).toBe('1 check passed, 1 warning, 1 block')
    expect(countsShort({ passed: 25, warnings: 2, blocks: 0 })).toBe('25 passed · 2 warnings · 0 blocks')
  })

  it('says how many blocks have a one-click fix', () => {
    const fix = { kind: 'apply', action: 'insert_footer', label: 'Insert footer block' } as Check['fixes'][number]
    const all = [check({ status: 'block', fixes: [fix] }), check({ status: 'block', fixes: [fix] }), check({ status: 'block', fixes: [fix] })]
    const counts = { passed: 17, warnings: 2, blocks: 3, unknown: 0, total: 22 }
    expect(blockedLine(counts, all)).toBe(
      '3 blocking issues, 2 warnings, 17 checks passed. Sending stays disabled until every block is fixed. Each one has a one-click fix below.',
    )
    expect(blockedLine(counts, [all[0]!, check({ status: 'block' }), check({ status: 'block' })])).toMatch(/1 of them has a one-click fix below\.$/)
    expect(blockedLine({ ...counts, blocks: 1 }, [all[0]!])).toMatch(/It has a one-click fix below\.$/)
  })

  it('labels the actions as on the boards', () => {
    expect(fixAllLabel('warn', 2)).toBe('Fix both warnings')
    expect(fixAllLabel('block', 3)).toBe('Apply all 3 fixes')
    expect(fixAllLabel('block', 1)).toBe('Apply the fix')
    expect(sendLabel('at', 31604)).toBe('Schedule send to 31,604 recipients')
    expect(sendLabel('now', 47842)).toBe('Send to 47,842 recipients')
    expect(confirmNote(2)).toBe("You'll confirm the audience and the two warnings on the next step.")
    expect(confirmNote(0)).toBe("You'll confirm the audience on the next step.")
  })
})

describe('schedule and time words', () => {
  it('writes local dates without shifting them across zones', () => {
    expect(localDateWords('2026-10-14')).toBe('Wednesday 14 Oct 2026')
    expect(scheduleSentence({ mode: 'at', date: '2026-11-04', time: '09:00', timezone: 'America/New_York' } as Review['schedule'])).toBe(
      'Wednesday 4 Nov 2026 at 09:00 America/New_York',
    )
    expect(scheduleShort({ mode: 'at', date: '2026-11-04', time: '09:00', timezone: 'America/New_York' } as Review['schedule'])).toBe('Nov 4, 09:00')
    expect(scheduleShort({ mode: 'now' } as Review['schedule'])).toBe('now')
  })

  it('counts down and back', () => {
    const now = new Date('2026-10-12T11:00:00Z')
    expect(sendsIn('2026-10-14T09:19:00Z', now)).toBe('1 d 22 h 19 m')
    expect(sendsIn('2026-10-12T10:00:00Z', now)).toBe('now')
    expect(agoWords('2026-10-12T09:00:00Z', now)).toBe('2 hours ago')
    expect(agoWords('2026-10-11T10:00:00Z', now)).toBe('yesterday')
    expect(waitWords(170)).toBe('about 3 hours')
    expect(waitShort(160)).toBe('2 h 40 min')
    expect(minutes(34)).toBe('34 min')
    expect(minutes(72)).toBe('1 h 12 min')
  })
})

describe('thresholds', () => {
  const t: Thresholds = { bounce: 2, complaint: 0.1, mailboxComplaint: 0.1, providerErrors: 5, scope: 'route' }
  it('reads what people type', () => {
    expect(parsePercent('2,5 %')).toBe(2.5)
    expect(parsePercent('0.10%')).toBe(0.1)
    expect(parsePercent('two')).toBeNull()
  })
  it('explains values it refuses', () => {
    expect(thresholdError('bounce', null, t)).toBe('Enter a percentage, for example 2.0.')
    expect(thresholdError('bounce', 25, t)).toBe('Keep it between 0.1 % and 20 %.')
    expect(thresholdError('mailboxComplaint', 0.6, t)).toBe('Keep it within five times the overall complaint rate.')
    expect(thresholdError('complaint', 0.1, t)).toBeUndefined()
  })
})

describe('confirmation', () => {
  it('asks for every warning, the ones with a one-click fix first', () => {
    const run = {
      checks: [
        check({ key: 'reputation.ip', status: 'warn', ack: 'I understand the Postal IP is listed on UCEPROTECT L3, a low-impact list.' }),
        check({
          key: 'audience.unengaged',
          status: 'warn',
          ack: 'I understand 412 unengaged contacts are included.',
          fixes: [{ kind: 'apply', action: 'exclude_sunset', label: 'Exclude 412 unengaged' } as Check['fixes'][number]],
        }),
        check({ key: 'content.size' }),
      ],
    } as Run
    expect(acknowledgements(run).map((a) => a.key)).toEqual(['audience.unengaged', 'reputation.ip'])
  })

  it('confirms warnings worded the same with one checkbox', () => {
    const both = 'I understand the hero image has no alt text and “Plan your migration” is 2.1:1, below 4.5:1.'
    const run = {
      checks: [
        check({ key: 'accessibility.alt_text:1-0-0', group: 'accessibility', status: 'warn', ack: both, ackKey: 'a1' }),
        check({ key: 'accessibility.contrast:6-0-0', group: 'accessibility', status: 'warn', ack: both, ackKey: 'a1' }),
        check({ key: 'accessibility.moderate', group: 'accessibility', status: 'warn', ack: 'I understand 4 moderate accessibility issues remain.', ackKey: 'a2' }),
      ],
    } as Run
    const acks = acknowledgements(run)
    expect(acks.map((a) => a.keys)).toEqual([['accessibility.alt_text:1-0-0', 'accessibility.contrast:6-0-0'], ['accessibility.moderate']])
    expect(acknowledgedKeys(acks, ['accessibility.alt_text:1-0-0'])).toEqual(['accessibility.alt_text:1-0-0', 'accessibility.contrast:6-0-0'])
  })

  it('matches confirmations by key, whatever language they read in', () => {
    // Two warnings that happen to read the same in a translation stay two
    // confirmations; the server's key decides.
    const run = {
      checks: [
        check({ key: 'links.shortener', status: 'warn', ack: 'Tôi hiểu.', ackKey: 'k1' }),
        check({ key: 'links.empty', status: 'warn', ack: 'Tôi hiểu.', ackKey: 'k2' }),
      ],
    } as Run
    expect(acknowledgements(run).map((a) => a.keys)).toEqual([['links.shortener'], ['links.empty']])
  })

  it('never marks an image decorative in bulk', () => {
    const alt = check({
      key: 'accessibility.alt_text:1-0-0',
      status: 'warn',
      fixes: [
        { kind: 'link', label: 'Write alt text', href: '/w/acme/campaigns/c/edit' },
        { kind: 'apply', action: 'accessibility_decorative', label: 'Mark as decorative' },
      ] as Check['fixes'],
    })
    expect(primaryFix(alt)).toBeUndefined()
  })

  it('never removes a sponsor’s block in bulk, and updates it', () => {
    const missing = check({
      key: 'content.sponsor.creative',
      status: 'block',
      fixes: [
        { kind: 'link', label: 'Open booking', href: '/w/acme/earn/sponsors?booking=b' },
        { kind: 'apply', action: 'sponsor_remove_block', label: 'Remove the block', tone: 'danger' },
      ] as Check['fixes'],
    })
    expect(primaryFix(missing)).toBeUndefined()
    const stale = check({
      key: 'content.sponsor.stale',
      status: 'block',
      fixes: [
        { kind: 'apply', action: 'sponsor_refresh_block', label: 'Update the block', tone: 'primary' },
        { kind: 'apply', action: 'sponsor_remove_block', label: 'Remove the block', tone: 'danger' },
      ] as Check['fixes'],
    })
    expect(primaryFix(stale)?.action).toBe('sponsor_refresh_block')
  })

  it('sets web addresses apart', () => {
    expect(linkParts('1 link redirects through bit.ly/3xK9tQ. Shorteners hide it.')).toEqual([
      { text: '1 link redirects through ', link: false },
      { text: 'bit.ly/3xK9tQ', link: true },
      { text: '. Shorteners hide it.', link: false },
    ])
  })
})

describe('approval states', () => {
  const review = (status: string, request?: Partial<NonNullable<Review['approval']['request']>>, over: Partial<Review> = {}): Review =>
    ({
      campaign: { status },
      run: { verdict: 'warnings' },
      access: { canSend: false },
      approval: { required: true, valid: request?.status === 'approved', request: request ? { status: 'pending', requestedAt: '2026-10-12T09:00:00Z', ...request } : undefined },
      ...over,
    }) as unknown as Review

  it('follows the request of a draft', () => {
    expect(approvalMode(review('draft'))).toBe('required')
    expect(approvalMode(review('approval', { status: 'pending' }))).toBe('waiting')
    expect(approvalMode(review('approved', { status: 'approved' }))).toBe('approved')
    expect(approvalMode(review('draft', { status: 'invalidated' }))).toBe('invalidated')
    expect(approvalMode({ ...review('draft'), access: { canSend: true } } as Review)).toBe('required')
    expect(approvalMode({ ...review('draft'), run: { verdict: 'blocked' } } as Review)).toBe('none')
  })

  it("keeps a scheduled send's time but leads with an approval that no longer covers it", () => {
    expect(approvalMode(review('scheduled', { status: 'approved' }))).toBe('none')
    expect(approvalMode(review('scheduled', { status: 'invalidated' }))).toBe('invalidated')
    expect(approvalMode(review('scheduled', { status: 'pending' }))).toBe('waiting')
    // A request settled before the campaign was scheduled doesn't hold it.
    const scheduled = { scheduled: { by: 'Linh Nguyen', at: '2026-10-12T10:00:00Z' } }
    expect(approvalMode(review('scheduled', { status: 'rejected' }, scheduled))).toBe('none')
    expect(approvalMode(review('scheduled', { status: 'rejected', requestedAt: '2026-10-12T11:00:00Z' }, scheduled))).toBe('required')
    expect(approvalMode(review('sending', { status: 'invalidated' }))).toBe('none')
  })

  it('says why a send was held in one sentence', () => {
    expect(heldSentence('Sending would hurt inbox placement for every recipient.')).toBe(
      'Sending would hurt inbox placement for every recipient, so the approved send was held.',
    )
    expect(heldSentence('')).toBe('The approved send was held.')
  })

  it('says what fixing a held send does to the approval', () => {
    expect(heldNote([check({ group: 'links', status: 'block' })])).toMatch(/^Fixing the link changes the content/)
    expect(heldNote([check({ group: 'routing', status: 'block' })])).toMatch(/^Changing the route invalidates the approval/)
    expect(heldNote([check({ group: 'links', status: 'block' }), check({ group: 'content', status: 'block' })])).toMatch(/^Fixing a block in the campaign/)
  })
})
