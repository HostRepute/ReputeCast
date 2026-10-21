import { describe, expect, it } from 'vitest'
import vectors from '../../../../internal/sms/testdata/segments.json'
import { countText, fixEncoding } from './segments'

interface Case {
  name: string
  text: string
  characters: number
  encoding: 'gsm7' | 'ucs2'
  segments: number
  fill: number[]
  left: number
  outside?: { char: string; kind: string; replacement: string; count: number; first: number }[]
  fixed?: string
}

// The same vectors as internal/sms/segments_test.go: the composer's counter
// and the server's must never disagree.
describe('countText', () => {
  for (const c of (vectors as { cases: Case[] }).cases) {
    it(c.name, () => {
      const got = countText(c.text)
      expect(got.characters).toBe(c.characters)
      expect(got.encoding).toBe(c.encoding)
      expect(got.segments).toBe(c.segments)
      expect(got.fill).toEqual(c.fill)
      expect(got.left).toBe(c.left)
      if (c.outside) {
        expect(got.outside.map((o) => ({ char: o.char, kind: o.kind, replacement: o.replacement, count: o.count, first: o.first }))).toEqual(c.outside)
      }
      if (c.fixed !== undefined) expect(fixEncoding(c.text)).toBe(c.fixed)
    })
  }
})
