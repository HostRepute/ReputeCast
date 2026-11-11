import { describe, expect, it } from 'vitest'
import { addStep, canMove, moveField, nextChoiceKey, removeStep, splitSteps, syncSteps, tagOf, type FormStep } from './steps'
import { emptyTargeting, listOf, scheduleSummary, targetingSummary, type Targeting } from './targeting'
import { abInputOf, abSentence, addVariant, evenSplit, sameAB, splitText, type ABDraft, type ABResult } from './variants'

const three = (): FormStep[] => [
  { id: 's1', fields: ['email'] },
  { id: 's2', fields: ['topics'] },
  { id: 's3', fields: ['first_name'] },
]

describe('form steps (M60 G-23)', () => {
  it('keeps the steps in line with the fields', () => {
    // A gone field drops out, a field in two steps stays in the first, new ones join the last.
    const steps = syncSteps(
      [
        { id: 's1', fields: ['email', 'gone'] },
        { id: 's2', fields: ['email', 'first_name'] },
      ],
      ['email', 'first_name', 'company'],
    )
    expect(steps).toEqual([
      { id: 's1', fields: ['email'] },
      { id: 's2', fields: ['first_name', 'company'] },
    ])
    expect(syncSteps([{ id: 's1', fields: ['email'] }], ['email'])).toBeUndefined()
    expect(syncSteps(undefined, ['email'])).toBeUndefined()
  })

  it('splits one step into two: the email first', () => {
    expect(splitSteps(['email', 'first_name', 'company'])).toEqual([
      { id: 's1', fields: ['email'] },
      { id: 's2', fields: ['first_name', 'company'] },
    ])
  })

  it('adds steps up to four, with free ids, and removes into the step before', () => {
    const four = addStep(three())
    expect(four.map((s) => s.id)).toEqual(['s1', 's2', 's3', 's4'])
    expect(addStep(four)).toBe(four)
    expect(removeStep(three(), 1)).toEqual([
      { id: 's1', fields: ['email', 'topics'] },
      { id: 's3', fields: ['first_name'] },
    ])
    expect(removeStep(three(), 0)[0]).toEqual({ id: 's2', fields: ['email', 'topics'] })
    const two = three().slice(0, 2)
    expect(removeStep(two, 1)).toBe(two)
  })

  it('moves a field past the edge of its step into the next one', () => {
    expect(moveField(three(), 0, 0, 1)).toEqual([
      { id: 's1', fields: [] },
      { id: 's2', fields: ['email', 'topics'] },
      { id: 's3', fields: ['first_name'] },
    ])
    expect(moveField(three(), 2, 0, -1)[1]).toEqual({ id: 's2', fields: ['topics', 'first_name'] })
    expect(canMove(three(), 0, 0, -1)).toBe(false)
    expect(canMove(three(), 2, 0, 1)).toBe(false)
    expect(canMove(three(), 1, 0, 1)).toBe(true)
  })

  it('gives a choice question a free key and its answers tags', () => {
    expect(nextChoiceKey([{ key: 'email', label: '', required: true }, { key: 'q1', label: 'Topics', required: false }], ['plan'])).toBe('q2')
    expect(tagOf('Product news')).toBe('product-news')
    expect(tagOf('  Thời sự & Kỹ thuật ')).toBe('thoi-su-ky-thuat')
  })
})

describe('where and when (M60 G-24)', () => {
  it('says nothing set as one sentence', () => {
    expect(targetingSummary(emptyTargeting())).toBe('Shown on every page, to everyone.')
  })

  it('joins the parts the board shows', () => {
    const x: Targeting = {
      ...emptyTargeting(),
      pages: {
        mode: 'only',
        rules: [
          { op: 'starts_with', value: '/pricing' },
          { op: 'starts_with', value: '/blog/' },
          { op: 'equals', value: '/changelog' },
        ],
      },
      devices: ['mobile'],
      countries: { mode: 'only', codes: ['US', 'CA'] },
      schedule: { endsAt: '2026-11-30T12:00:00Z' },
    }
    expect(targetingSummary(x)).toBe('Shown on 3 pages · mobile · from US and CA · until 30 Nov')
    // Every device ticked is no device rule.
    expect(targetingSummary({ ...emptyTargeting(), devices: ['desktop', 'tablet', 'mobile'] })).toBe('Shown on every page, to everyone.')
  })

  it("says a hosted form's schedule", () => {
    expect(scheduleSummary({})).toBe('Open now, with no end date.')
    expect(scheduleSummary({ endsAt: '2026-11-30T12:00:00Z' })).toBe('Closes 30 Nov')
  })

  it('reads comma-separated values', () => {
    expect(listOf(' us, CA ,,de ')).toEqual(['us', 'CA', 'de'])
  })
})

describe('A/B tests (M60 G-25)', () => {
  const none: ABDraft = { variants: [], autoEnd: false, minViews: 500, confidence: 0.95 }
  const content = { heading: 'Get the Weekly Digest', description: 'Every Tuesday.', button: 'Subscribe', popupDelay: 0, popupScroll: 0, steps: three() }

  it('splits evenly to 100', () => {
    expect(evenSplit(2)).toEqual([50, 50])
    expect(evenSplit(3)).toEqual([34, 33, 33])
    expect(splitText([{ weight: 34 }, { weight: 33 }, { weight: 33 }])).toBe('34/33/33')
  })

  it("adds B as a copy of A's copy and steps", () => {
    const { draft, key } = addVariant(none, content)
    expect(key).toBe('B')
    expect(draft.variants.map((v) => [v.key, v.weight])).toEqual([
      ['A', 50],
      ['B', 50],
    ])
    expect(draft.variants[1]!.content?.heading).toBe('Get the Weekly Digest')
    expect(draft.variants[1]!.content?.steps).not.toBe(content.steps)
    expect(sameAB(draft, none)).toBe(false)
    const next = addVariant(draft, content)
    expect(next.key).toBe('C')
    expect(next.draft.variants.map((v) => v.weight)).toEqual([34, 33, 33])
  })

  it("saves each variant's steps in line with the form's fields", () => {
    const { draft } = addVariant(none, content)
    const input = abInputOf(draft, ['email', 'topics'])
    expect(input.variants[0]).toEqual({ key: 'A', weight: 50 })
    expect(input.variants[1]!.content?.steps).toEqual([
      { id: 's1', fields: ['email'] },
      { id: 's2', fields: ['topics'] },
      { id: 's3', fields: [] },
    ])
  })

  const result = (over: Partial<ABResult>): ABResult => ({
    status: 'running',
    autoEnd: false,
    confidence: 97,
    doubleOptIn: true,
    fewestViews: 4096,
    lift: 28,
    minViews: 500,
    targetConfidence: 0.95,
    variants: [
      { key: 'A', weight: 50, views: 4096, submissions: 131, confirmed: 118, rate: 3.2 },
      { key: 'B', weight: 50, views: 4104, submissions: 168, confirmed: 152, rate: 4.1 },
    ],
    ...over,
  })

  it('says where the test stands', () => {
    expect(abSentence(result({ outcome: 'winner', leader: 'B', runnerUp: 'A' }))).toBe('B gets 4.1% vs 3.2% for A (+28%), 97% confidence.')
    expect(abSentence(result({ outcome: 'not_significant' }))).toBe('Not significant yet.')
    expect(abSentence(result({ outcome: 'too_small', fewestViews: 212 }))).toBe('Too few views yet: 212 of 500 per variant.')
  })
})
