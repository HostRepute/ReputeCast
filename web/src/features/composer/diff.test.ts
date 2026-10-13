import { describe, expect, it } from 'vitest'
import { diffLines } from './diff'

describe('diffLines', () => {
  it('marks added and removed lines', () => {
    const out = diffLines('a\nb\nc', 'a\nB\nc\nd')
    expect(out).toEqual([
      { kind: 'same', text: 'a' },
      { kind: 'del', text: 'b' },
      { kind: 'add', text: 'B' },
      { kind: 'same', text: 'c' },
      { kind: 'add', text: 'd' },
    ])
  })

  it('folds long unchanged runs around the changes', () => {
    const a = Array.from({ length: 30 }, (_, i) => `line ${i}`)
    const b = [...a]
    b[15] = 'changed'
    const out = diffLines(a.join('\n'), b.join('\n'), 2)
    expect(out.filter((l) => l.kind === 'fold').map((l) => l.text)).toEqual(['… 13 unchanged lines', '… 12 unchanged lines'])
    expect(out.find((l) => l.kind === 'add')?.text).toBe('changed')
  })

  it('says nothing changed for equal texts', () => {
    expect(diffLines('x\ny', 'x\ny').every((l) => l.kind === 'same' || l.kind === 'fold')).toBe(true)
  })
})
