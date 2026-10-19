import { describe, expect, it } from 'vitest'
import { diffWords } from '@/features/languages/compare'
import { phrases } from './rewrite'

describe('phrases', () => {
  it('joins word changes separated by spaces into one removal and one addition', () => {
    expect(phrases(diffWords('a b c d', 'a x y d'))).toEqual([
      { kind: 'same', text: 'a ' },
      { kind: 'del', text: 'b c' },
      { kind: 'add', text: 'x y' },
      { kind: 'same', text: ' d' },
    ])
  })

  it('keeps unchanged text as it is', () => {
    expect(phrases(diffWords('the same words', 'the same words'))).toEqual([{ kind: 'same', text: 'the same words' }])
  })

  it('keeps a shortened tail as one removal', () => {
    const out = phrases(diffWords('plan it now, and sleep through it tonight', 'plan it now'))
    expect(out[0]).toEqual({ kind: 'same', text: 'plan it ' })
    expect(out.filter((p) => p.kind === 'del')).toHaveLength(1)
  })
})
