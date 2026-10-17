import { describe, expect, it } from 'vitest'
import { buckets } from './GrowthChart'
import { isImport, sourceLine, sourceName, type Source } from './sourceText'

const src = (s: Partial<Source>): Source => ({ key: 'k', kind: 'form', name: '', imported: false, hosts: [], ...s })

describe('Growth sources in words (GrowthSources)', () => {
  it('names each kind of source', () => {
    expect(sourceName(src({ kind: 'api_key', name: 'Website signup form' }))).toBe('API key “Website signup form”')
    expect(sourceName(src({ kind: 'person' }))).toBe('Added by a person')
    expect(sourceName(src({ kind: 'subscribe_page' }))).toBe('Subscribe page')
    expect(sourceLine(src({ kind: 'form', formKind: 'embedded', hosts: ['acme.io', 'blog.acme.io'] }))).toBe('Embedded form · acme.io and blog.acme.io')
    expect(sourceLine(src({ kind: 'subscribe_page', url: 'https://news.acme.io/subscribe' }))).toBe('Hosted · news.acme.io/subscribe')
    expect(sourceLine(src({ kind: 'api_key', keyPrefix: 'rc_live_8f2c…a91d' }))).toBe('API · rc_live_8f2c…a91d')
    expect(sourceLine(src({ kind: 'import', importCount: 39402, tookImportDay: 7391 }))).toBe('Import · 7,391 took the import day as their join date')
    expect(sourceLine(src({ kind: 'import', tookImportDay: 2118 }))).toBe('Import · 2,118 contacts')
  })

  it('says a member picked the list only on forms that offer a choice', () => {
    expect(sourceLine(src({ kind: 'form', listChoice: true, hosts: ['acme.io'] }), 'Beta Program')).toBe('Picked Beta Program on the form')
    expect(sourceLine(src({ kind: 'form', formKind: 'embedded', hosts: ['blog.acme.io'] }), 'Beta Program')).toBe('Embedded form · blog.acme.io')
    expect(sourceLine(src({ kind: 'preference_center' }), 'Beta Program')).toBe('Added Beta Program to an existing subscription')
  })

  it('treats imports and what they carried as imports', () => {
    expect(isImport(src({ kind: 'import' }))).toBe(true)
    expect(isImport(src({ kind: 'carried', carried: 'import:csv' }))).toBe(true)
    expect(isImport(src({ kind: 'carried', carried: 'form:docs' }))).toBe(false)
  })
})

describe('The joins chart', () => {
  const day = (d: string, j: number) => ({ day: d, joined: [j, 1], left: [0, 1], imports: 0 })
  it('keeps days for a month and turns a year into weeks', () => {
    const month = Array.from({ length: 30 }, (_, i) => day(`2026-09-${String(i + 1).padStart(2, '0')}`, 2))
    expect(buckets(month)).toHaveLength(30)
    const year = Array.from({ length: 365 }, (_, i) => day(`d${i}`, 1))
    const weeks = buckets(year)
    expect(weeks).toHaveLength(53)
    expect(weeks[0]).toMatchObject({ day: 'd0', days: 7, joined: [7, 7], left: [0, 7] })
    expect(weeks[52]).toMatchObject({ days: 1 })
  })
})
