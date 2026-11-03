import { describe, expect, it } from 'vitest'
import type { MessageDescriptor } from '@/i18n'
import { mc } from './messages'
import { hiddenName } from './WebPanel'

// The web checks name each block left out of a post's web version; a poll
// stays in the inbox, where each answer is the recipient's own link.
describe('hiddenName', () => {
  const t = ((d: MessageDescriptor, v?: Record<string, unknown>) => (v?.name ? `${d.message}:${String(v.name)}` : d.message)) as Parameters<typeof hiddenName>[0]

  it('names a poll as a poll', () => {
    expect(hiddenName(t, { kind: 'poll', reason: 'email_only', line: 3, name: 'Poll' })).toBe(mc.hiddenPoll.message)
  })

  it('names other blocks as before', () => {
    expect(hiddenName(t, { kind: 'footer', reason: 'email_only', line: 9 })).toBe(mc.hiddenFooter.message)
    expect(hiddenName(t, { kind: 'block', reason: 'conditional', line: 4, name: 'Beta corner' })).toBe(`${mc.hiddenConditional.message}:Beta corner`)
  })
})
