import { describe, expect, it } from 'vitest'
import { inBounds, lastMatched, numberText, parseNumber, ruleAction, ruleDuration, rulesOn, ruleWhen, savedText } from './traffic-model'

// Copy of the ProvidersTrafficRules board.
describe('ProvidersTrafficRules', () => {
  it('writes a reply rule as the board does', () => {
    const yahoo = { mailbox: 'yahoo' as const, count: 10, windowMinutes: 10, action: 'pause' as const, durationMinutes: 120 }
    expect(`${ruleWhen(yahoo)} → ${ruleAction(yahoo)} · ${ruleDuration(yahoo)}`).toBe('After 10 replies in 10 min → Pause Yahoo · for 2 h')
    const outlook = { mailbox: 'outlook' as const, count: 1, windowMinutes: 10, action: 'pause_route' as const, durationMinutes: 0 }
    expect(`${ruleWhen(outlook)} → ${ruleAction(outlook)} · ${ruleDuration(outlook)}`).toBe(
      'On the first reply → Pause Outlook on that route and alert · until someone resumes',
    )
    const gmail = { mailbox: 'gmail' as const, count: 5, windowMinutes: 10, action: 'slow' as const, durationMinutes: 30 }
    expect(`${ruleWhen(gmail)} → ${ruleAction(gmail)} · ${ruleDuration(gmail)}`).toBe('After 5 replies in 10 min → Slow Gmail to 50% · for 30 min')
    expect(ruleAction({ mailbox: 'icloud', action: 'alert' })).toBe('Alert only')
  })

  it('says when a rule last matched', () => {
    expect(lastMatched({ lastMatchedAt: '2026-09-23T09:14:00Z', lastCampaign: 'Weekly Digest #139' })).toBe('23 Sep · Weekly Digest #139')
    expect(lastMatched({})).toBe('never')
  })

  it('counts the rules that are on', () => {
    expect(rulesOn([{ enabled: true }, { enabled: true }, { enabled: false }])).toBe('2 of 3 on')
  })

  it('says who saved last, or that there are unsaved changes', () => {
    const now = new Date('2026-10-14T09:00:00Z')
    expect(savedText({ savedAt: '2026-10-12T09:00:00Z', savedBy: 'Linh Nguyen' }, false, now)).toBe('Saved 2 days ago by Linh Nguyen')
    expect(savedText({ savedAt: '2026-10-12T09:00:00Z', savedBy: 'Linh Nguyen' }, true, now)).toBe('Unsaved changes')
    expect(savedText({}, false, now)).toBe('Defaults, not changed yet')
  })

  it('reads typed numbers with points or commas', () => {
    expect([parseNumber('0.5'), parseNumber('0,10'), parseNumber(' 15 '), parseNumber(''), parseNumber('abc')]).toEqual([0.5, 0.1, 15, undefined, undefined])
    expect([inBounds('0.5', [0.01, 20]), inBounds('25', [0.01, 20]), inBounds('', [0, 1])]).toEqual([true, false, false])
    expect([numberText(0.5), numberText(0.1, 2), numberText(1, 1), numberText(15)]).toEqual(['0.5', '0.10', '1.0', '15'])
  })
})
