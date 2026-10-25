import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { clearJournal, forgetRecovered, journalDiffers, journalKey, journalWriter, pruneJournals, readJournal, recoverJournal, writeJournal, type JournalEntry } from './journal'

const key = journalKey('u1', 'c1')

function entry(patch: Partial<JournalEntry> = {}): JournalEntry {
  return {
    v: 1,
    campaignId: 'c1',
    language: '',
    baseVersion: 13,
    startedAt: '2026-10-25T09:31:00Z',
    updatedAt: '2026-10-25T09:35:00Z',
    subject: 'S',
    preheader: 'P',
    mjml: '<mjml><mj-body></mj-body></mjml>',
    abTest: null,
    ...patch,
  }
}

beforeEach(() => window.localStorage.clear())
afterEach(() => vi.useRealTimers())

describe('the composer journal', () => {
  it('keeps typing within two seconds, so a crash mid-typing loses at most that', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-10-25T09:31:00Z'))
    let text = ''
    const w = journalWriter({ key, campaignId: 'c1', snapshot: () => ({ baseVersion: 14, subject: 'S', preheader: '', mjml: `<mj-text>${text}</mj-text>`, abTest: null }) })
    // Someone types a character every 150 ms for 20 seconds…
    for (let i = 0; i < 134; i++) {
      text += 'a'
      w.touch()
      vi.advanceTimersByTime(150)
      // …and at every moment the journal is at most two seconds behind.
      const kept = readJournal(key)
      const keptChars = kept ? kept.mjml.length - '<mj-text></mj-text>'.length : 0
      if (i >= 14) expect(text.length - keptChars).toBeLessThanOrEqual(Math.ceil(2000 / 150))
    }
    // …then the tab crashes: nothing else runs. Reopening finds the edits.
    const kept = readJournal(key)!
    expect(kept.startedAt).toBe('2026-10-25T09:31:00.000Z')
    expect(Date.parse(kept.updatedAt) - Date.parse(kept.startedAt)).toBeGreaterThan(18_000)
    expect(text.length - (kept.mjml.length - '<mj-text></mj-text>'.length)).toBeLessThanOrEqual(14)
  })

  it('writes at once when the tab is hidden, and forgets edits once they are saved', () => {
    vi.useFakeTimers()
    const w = journalWriter({ key, campaignId: 'c1', snapshot: () => ({ baseVersion: 14, subject: 'Typed', preheader: '', mjml: '<mjml/>', abTest: null }) })
    w.touch()
    expect(readJournal(key)).toBeNull()
    w.flush()
    expect(readJournal(key)?.subject).toBe('Typed')
    w.clear()
    expect(readJournal(key)).toBeNull()
    // Nothing pending: a flush writes nothing.
    w.flush()
    expect(readJournal(key)).toBeNull()
  })

  it('keeps one journal per person, campaign and language version', () => {
    expect(journalKey('u1', 'c1')).not.toBe(journalKey('u2', 'c1'))
    expect(journalKey('u1', 'c1', 'es')).not.toBe(journalKey('u1', 'c1'))
  })

  it('tells edits worth restoring from what is already saved', () => {
    const e = entry({ mjml: '<mjml>\n  <mj-body></mj-body>\n</mjml>' })
    expect(journalDiffers(e, { subject: 'S', preheader: 'P', mjml: '<mjml> <mj-body></mj-body> </mjml>' })).toBe(false)
    expect(journalDiffers(e, { subject: 'S2', preheader: 'P', mjml: e.mjml })).toBe(true)
  })

  it('ignores broken entries and drops journals nobody came back to', () => {
    window.localStorage.setItem(key, '{not json')
    expect(readJournal(key)).toBeNull()
    writeJournal(journalKey('u1', 'old'), entry({ campaignId: 'old', updatedAt: '2026-09-01T00:00:00Z' }))
    writeJournal(journalKey('u1', 'new'), entry({ campaignId: 'new', updatedAt: '2026-10-24T00:00:00Z' }))
    pruneJournals(Date.parse('2026-10-25T00:00:00Z'))
    expect(readJournal(journalKey('u1', 'old'))).toBeNull()
    expect(readJournal(journalKey('u1', 'new'))?.campaignId).toBe('new')
    clearJournal(journalKey('u1', 'new'))
  })

  it('offers the last session\'s edits back once, kept aside from the new ones', () => {
    const loaded = { subject: 'S', preheader: 'P', mjml: '<mjml><mj-body><mj-text>Saved</mj-text></mj-body></mjml>' }
    writeJournal(key, entry({ mjml: '<mjml><mj-body><mj-text>Typed, never saved</mj-text></mj-body></mjml>', updatedAt: new Date().toISOString() }))
    const found = recoverJournal(key, loaded)
    expect(found?.mjml).toContain('Typed, never saved')
    // Undecided, the offer stays for the next time the campaign opens…
    expect(recoverJournal(key, loaded)?.mjml).toContain('Typed, never saved')
    // …unless newer edits were journaled since: those are what the person did last.
    writeJournal(key, entry({ mjml: 'newer edits', updatedAt: new Date().toISOString() }))
    expect(recoverJournal(key, loaded)?.mjml).toBe('newer edits')
    forgetRecovered(key)
    expect(recoverJournal(key, loaded)).toBeNull()
  })

  it('forgets a journal that holds nothing new', () => {
    const loaded = { subject: 'S', preheader: 'P', mjml: '<mjml/>' }
    writeJournal(key, entry({ mjml: '<mjml/>', updatedAt: new Date().toISOString() }))
    expect(recoverJournal(key, loaded)).toBeNull()
    expect(readJournal(key)).toBeNull()
  })
})
