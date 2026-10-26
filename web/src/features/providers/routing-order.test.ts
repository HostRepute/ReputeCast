import { addToSplit, aboutCount, changeText, moveSplit, onlyOrderChanged, removeFromSplit, setWeight, splitPair, splitTotal, weightsProblem, type Draft, type StreamDraft } from './routing-model'

const ses = 'ses-0000-0000-0000-000000000001'
const postal = 'po-00-0000-0000-0000-000000000003'
const postmark = 'pm-00-0000-0000-0000-000000000002'

const stream = (splits: { providerId: string; weight: number }[], extra: Partial<StreamDraft> = {}): StreamDraft => ({
  splits,
  fallback: '',
  failover: 'pause_and_ask',
  conditions: [],
  priority: true,
  ...extra,
})

const draft = (broadcast: StreamDraft): Draft => ({
  broadcast,
  automation: stream([{ providerId: ses, weight: 100 }], { fallback: postmark, failover: 'automatic' }),
  system: stream([{ providerId: postmark, weight: 100 }], { fallback: ses, failover: 'automatic' }),
  transactional: stream([{ providerId: postmark, weight: 100 }], { failover: 'automatic' }),
})

const providers = [
  { id: ses, short: 'SES' },
  { id: postal, short: 'Postal' },
  { id: postmark, short: 'Postmark' },
] as never

describe('the ordered split (PR-14)', () => {
  const saved = stream([
    { providerId: ses, weight: 90 },
    { providerId: postal, weight: 10 },
  ])

  it('reorders rows without touching weights', () => {
    const moved = moveSplit(saved, 1, 0)
    expect(moved.splits.map((x) => x.providerId)).toEqual([postal, ses])
    expect(splitTotal(moved)).toBe(100)
    expect(onlyOrderChanged(moved, saved)).toBe(true)
    expect(moveSplit(saved, 0, 5)).toBe(saved)
  })

  it('changes one weight and says when the total is off', () => {
    const d = setWeight(saved, postal, 15)
    expect(splitPair(d)).toBe('90/15')
    expect(weightsProblem(draft(d))).toEqual({ stream: 'broadcast', total: 105 })
    expect(weightsProblem(draft(setWeight(d, ses, 85)))).toBeUndefined()
  })

  it('adds a provider at 0% and gives a removed one’s weight to the first row', () => {
    const added = addToSplit(saved, postmark)
    expect(added.splits).toEqual([...saved.splits, { providerId: postmark, weight: 0 }])
    expect(removeFromSplit(saved, postal).splits).toEqual([{ providerId: ses, weight: 100 }])
  })

  it('previews the impact before saving', () => {
    const before = draft(saved)
    const after = draft(setWeight(setWeight(saved, ses, 85), postal, 15))
    expect(changeText(after, before, providers, { moved: 1560, from: 'SES', toward: 'Postal' })).toBe(
      'Broadcasts split 90/10 → 85/15 · about 1,600 Broadcast recipients move from SES to Postal; nobody else changes provider',
    )
    expect(changeText(after, before, providers)).toBe('Broadcasts split 90/10 → 85/15 · only the difference moves; nobody else changes provider')
    expect(changeText(draft(moveSplit(saved, 1, 0)), before, providers)).toBe('Broadcasts order changed to Postal, SES · nobody changes provider')
    expect(changeText(draft(setWeight(saved, postal, 20)), before, providers)).toBe('Weights add up to 110%. Make them 100% to save.')
    expect(aboutCount(30)).toBe('fewer than 100')
  })
})
