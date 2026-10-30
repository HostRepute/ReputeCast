import { describe, expect, it } from 'vitest'
import { isKnownField, knownFields, suggestField } from './fields'

const tokens = [
  { key: 'first_name', label: 'First name', kind: 'contact' as const },
  { key: 'last_name', label: 'Last name', kind: 'contact' as const },
  { key: 'plan', label: 'Plan', kind: 'custom' as const },
]

describe('fields an email may print', () => {
  it("knows the workspace's fields and what ReputeCast fills in", () => {
    const known = knownFields(tokens)
    for (const k of ['first_name', 'plan', 'unsubscribe_url', 'contact.plan', 'rc_poll_42', 'rc_survey_x', 'forloop.index']) expect(isKnownField(k, known), k).toBe(true)
    for (const k of ['frist_name', 'issue', 'company']) expect(isKnownField(k, known), k).toBe(false)
    // A digest template prints its feed.
    expect(isKnownField('issue', knownFields(tokens, true))).toBe(true)
  })

  it('suggests what a mistyped field meant', () => {
    const known = knownFields(tokens)
    expect(suggestField('frist_name', known)).toBe('first_name')
    expect(suggestField('lastname', known)).toBe('last_name')
    expect(suggestField('plna', known)).toBe('plan')
    expect(suggestField('company', known)).toBeUndefined()
  })
})
