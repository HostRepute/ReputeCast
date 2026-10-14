import { describe, expect, it } from 'vitest'
import { affectsText, membersText, restoredText, type ListImpact } from './listDelete'

const board: ListImpact = {
  id: 'l1',
  name: 'Product Updates',
  members: 14110,
  elsewhere: 9802,
  otherLists: ['Weekly Digest'],
  otherListCount: 1,
  forms: [
    { id: 'f1', name: 'Footer mini form' },
    { id: 'f2', name: 'Newsletter signup' },
  ],
  automations: [{ id: 'a1', name: 'Joined Product Updates' }],
  campaigns: [
    { id: 'c1', name: 'A' },
    { id: 'c2', name: 'B' },
    { id: 'c3', name: 'C' },
  ],
}

describe('Delete list dialog copy', () => {
  it('reads like the Dialogs board', () => {
    expect(membersText(board)).toBe(
      '14,110 contacts are on this list. 9,802 are also on Weekly Digest and stay subscribed there; 4,308 are on no other list and will become contacts without a subscription. Suppressions are never removed.',
    )
    expect(affectsText(board)).toBe('2 signup forms, 1 automation trigger ("Joined Product Updates") and 3 scheduled campaigns use this list. They will pause.')
  })

  it('names up to three other lists, then counts the rest', () => {
    const im = { ...board, otherLists: ['Weekly Digest', 'Beta Program', 'Events'], otherListCount: 5 }
    expect(membersText(im)).toContain('also on Weekly Digest, Beta Program, Events or 2 other lists and stay subscribed there')
  })

  it('handles lists nobody else shares, lists everyone shares and empty lists', () => {
    expect(membersText({ ...board, elsewhere: 0 })).toBe(
      '14,110 contacts are on this list. All of them are on no other list and will become contacts without a subscription. Suppressions are never removed.',
    )
    expect(membersText({ ...board, elsewhere: 14110, otherLists: ['Weekly Digest', 'Beta Program'], otherListCount: 2 })).toBe(
      '14,110 contacts are on this list. All of them are also on Weekly Digest or Beta Program and stay subscribed there. Suppressions are never removed.',
    )
    expect(membersText({ ...board, members: 1, elsewhere: 0 })).toBe(
      '1 contact is on this list. It is on no other list and will become a contact without a subscription. Suppressions are never removed.',
    )
    expect(membersText({ ...board, members: 0, elsewhere: 0 })).toBe('Nobody is subscribed to this list. Suppressions are never removed.')
  })

  it('says what uses the list in the singular, and nothing when unused', () => {
    expect(affectsText({ ...board, forms: board.forms.slice(0, 1), automations: [], campaigns: [] })).toBe('1 signup form uses this list. It will pause.')
    expect(affectsText({ ...board, forms: [], automations: [], campaigns: [] })).toBe('')
  })

  it('says what a restore turned back on', () => {
    expect(restoredText({ id: 'l1', name: 'Product Updates', forms: ['Newsletter signup'], automations: ['Welcome series'], campaigns: ['Weekly Digest #142'], left: [] })).toBe(
      '1 signup form and 1 automation are live again; Weekly Digest #142 is back on schedule.',
    )
    expect(restoredText({ id: 'l1', name: 'Product Updates', forms: [], automations: [], campaigns: [], left: ['Newsletter signup'] })).toBe(
      'Its members are back. Newsletter signup stays as it is.',
    )
  })
})
