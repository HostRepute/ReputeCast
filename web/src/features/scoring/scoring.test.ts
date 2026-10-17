import { describe, expect, it } from 'vitest'
import { bandOf, bandRange } from './bands'
import { capInline, capText, ruleFor, ruleProblems, ruleWhen, signed, type Rule } from './ruleText'
import { changesOf, sameConfig, type Config } from './useScoringDraft'

const rules: Rule[] = [
  { id: 'r1', kind: 'open', points: 2, enabled: true, cap: { kind: 'points_week', points: 6 } },
  { id: 'r2', kind: 'click', points: 5, enabled: true, cap: { kind: 'points_week', points: 20 } },
  { id: 'r3', kind: 'click', contains: 'acme.io/pricing', points: 15, enabled: true, cap: { kind: 'once_month' } },
  { id: 'r6', kind: 'field', field: 'plan', op: 'is_one_of', values: ['team', 'business'], points: 10, enabled: true, cap: { kind: 'while_true' } },
]
const saved: Config = { rules, decay: { enabled: true, afterWeeks: 4, percent: 10 }, bands: { warmFrom: 20, hotFrom: 50 } }

describe('Scoring rules in words (ScoringRules)', () => {
  it('reads like the board', () => {
    expect(ruleWhen(rules[0]!)).toBe('Human open of any email')
    expect(ruleWhen(rules[2]!)).toBe('Click on a link containing acme.io/pricing')
    expect(ruleWhen(rules[3]!)).toBe('Field plan is team or business')
    expect(capText(rules[0]!)).toBe('6 a week')
    expect(capText(rules[2]!)).toBe('Once a month')
    expect(capInline({ ...rules[0]!, cap: { kind: 'none' } })).toBe('none')
    expect(signed(-20)).toBe('−20')
    expect(signed(15)).toBe('+15')
  })

  it("says why a cap can't work", () => {
    expect(ruleProblems({ ...rules[1]!, cap: { kind: 'points_week', points: 3 } }).cap).toBe(
      'A cap of 3 a week is lower than the 5 points one click gives. Raise the cap to 5 or more, or lower the points.',
    )
    expect(ruleProblems({ ...rules[1]!, points: 0 }).points).toBe('Points go from −50 to +50, and not 0.')
    expect(ruleProblems({ ...rules[2]!, contains: ' ' }).contains).toBe('Enter the text the link contains.')
    expect(ruleProblems(rules[3]!)).toEqual({})
  })

  it('starts a new rule with its kind’s defaults', () => {
    expect(ruleFor('field', 'n1', { lists: [], forms: [], apiEvents: [], fields: [{ key: 'plan', label: 'Plan', type: 'select', options: [] }] })).toMatchObject({
      field: 'plan',
      op: 'is_one_of',
      cap: { kind: 'while_true' },
    })
  })
})

describe('Bands', () => {
  it('splits 0–100 at the thresholds', () => {
    const b = { warmFrom: 20, hotFrom: 50 }
    expect([bandOf(19, b), bandOf(20, b), bandOf(49, b), bandOf(50, b)]).toEqual(['cold', 'warm', 'warm', 'hot'])
    expect(bandRange('warm', b)).toEqual([20, 49])
    expect(bandRange('hot', b)).toEqual([50, 100])
  })
})

describe('Unsaved changes', () => {
  it('names the edited rules, deletions, decay and thresholds', () => {
    const draft: Config = { ...saved, rules: saved.rules.map((r) => (r.id === 'r3' ? { ...r, points: 25 } : r)) }
    expect(changesOf(saved, draft, true)).toEqual({ rules: [3], deleted: 0, decay: false, bands: false, onlyBands: false, setup: false })
    const bands: Config = { ...saved, bands: { warmFrom: 25, hotFrom: 50 } }
    expect(changesOf(saved, bands, true).onlyBands).toBe(true)
    const fewer: Config = { ...saved, rules: saved.rules.slice(1) }
    expect(changesOf(saved, fewer, true)).toMatchObject({ deleted: 1 })
  })

  it("doesn't count a missing list of values as a change", () => {
    const same: Config = { ...saved, rules: saved.rules.map((r) => (r.kind === 'open' ? { ...r, values: [] } : r)) }
    expect(sameConfig(saved, same)).toBe(true)
  })
})
