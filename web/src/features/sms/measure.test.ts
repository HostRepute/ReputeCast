import { describe, expect, it } from 'vitest'
import type { Catalog, MessageDescriptor } from '@/i18n'
import { findFields, findLinks, measure, renderWith } from './measure'

const prefix = 'Acme:'
const optOut = 'Reply STOP to opt out'

describe('measure', () => {
  it('counts the summit text as the composer board does: 130 characters, 1 segment', () => {
    const body = '8 days to Acme Summit 2026! The agenda is live: acme.io/summit/agenda See you 22 Oct in Chicago or online.'
    const m = measure(prefix, body, optOut, 'go.acme.io')
    expect(m.count.characters).toBe(130)
    expect(m.count.segments).toBe(1)
    expect(m.count.left).toBe(30)
    expect(m.links).toHaveLength(1)
    expect(m.links[0]!.host).toBe('acme.io')
  })

  it('flags a public shortener and leaves it unshortened', () => {
    const links = findLinks('The agenda is live: bit.ly/3Acme26 See you soon.')
    expect(links[0]!.shortener).toBe(true)
  })

  it('never takes an email address for a link', () => {
    expect(findLinks('Write to support@acme.io/help today')).toHaveLength(0)
  })

  it('counts a field at its fallback and renders a name', () => {
    const body = 'Hi {{ first_name | default: "there" }}! Only 8 days to go.'
    expect(findFields(body)).toEqual([{ key: 'first_name', fallback: 'there', hasFallback: true }])
    const m = measure(prefix, body, optOut, '')
    expect(m.count.characters).toBe('Acme: Hi there! Only 8 days to go. Reply STOP to opt out'.length)
    expect(m.base).toBe('Acme: Hi ! Only 8 days to go. Reply STOP to opt out'.length)
    expect(renderWith(body, { first_name: 'Christopher' })).toBe('Hi Christopher! Only 8 days to go.')
  })
})

// X-1: texting says what the reader's role can't do, or who can, in words
// (providers:manage and sms:send never show).
const permissionKey = /\b[a-z]+:(?:view|edit|send|pause|approve|publish|content|create|import|export|delete|verify|manage|sql|share|hostrepute|invite|use|run|review|connect|transfer|usage|library)\b/
const english = import.meta.glob<Record<string, Record<string, MessageDescriptor>>>('./**/*messages.ts', { eager: true })
const vietnamese = import.meta.glob<Catalog>('./**/*messages.vi.ts', { eager: true, import: 'default' })

describe("SMS copy", () => {
  it('names no permission key, in English or Vietnamese', () => {
    const en = Object.values(english).flatMap((file) => Object.values(file).flatMap((group) => Object.values(group)))
    const vi = Object.values(vietnamese).flatMap((catalog) => Object.entries(catalog))
    expect(en.length).toBeGreaterThan(0)
    expect(vi.length).toBeGreaterThan(0)
    for (const d of en) {
      expect(d.message, d.id).not.toMatch(permissionKey)
      // Nor does a call site pass one in: no {perm}, {perms} or {permission} to fill.
      expect(d.message, d.id).not.toMatch(/\{perm(?:s|ission)?\}/)
    }
    for (const [id, text] of vi) expect(text, id).not.toMatch(permissionKey)
  })
})
