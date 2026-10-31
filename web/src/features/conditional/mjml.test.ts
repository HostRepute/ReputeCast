import type { Catalog, MessageDescriptor } from '@/i18n'
import { prepareCompile } from './mjml'

// The canvas shows a block marked rc-omit (a Sponsor block without a
// booking); the compiled email leaves it out, as the server's PrepareCompile
// does (internal/mjml).
describe('prepareCompile', () => {
  const doc =
    '<mjml><mj-body><mj-section><mj-column><mj-text>Story</mj-text></mj-column></mj-section>' +
    '<!-- rc:sponsor --><mj-section rc-sponsor="1" rc-omit="1" padding="12px 24px"><mj-column><mj-text>No sponsor is booked for #144.</mj-text></mj-column></mj-section>' +
    '<mj-section rc-omit="0"><mj-column><mj-text>Kept</mj-text></mj-column></mj-section></mj-body></mjml>'

  it('leaves out blocks marked rc-omit and strips the attribute elsewhere', () => {
    const out = prepareCompile(doc)
    expect(out).not.toContain('No sponsor is booked')
    expect(out).not.toContain('rc-')
    expect(out).toContain('Story')
    expect(out).toContain('Kept')
  })

  it('keeps marking conditional blocks around an omitted one', () => {
    const rule = JSON.stringify({ id: 'beta01', show: 'only', match: 'all', conditions: [] }).replace(/"/g, '&quot;')
    const out = prepareCompile(doc.replace('<mj-section rc-omit="0">', `<mj-section rc-visible="${rule}">`))
    expect(out).toContain('<mj-raw><!--rc:if beta01--></mj-raw>')
    expect(out).toContain('<mj-raw><!--rc:fi beta01--></mj-raw>')
    expect(out).not.toContain('No sponsor is booked')
  })
})

// X-1: who sees a block says what the reader's role can't do, or who can, in
// words (campaigns:edit never shows).
const permissionKey = /\b[a-z]+:(?:view|edit|send|pause|approve|publish|content|create|import|export|delete|verify|manage|sql|share|hostrepute|invite|use|run|review|connect|transfer|usage|library)\b/
const english = import.meta.glob<Record<string, Record<string, MessageDescriptor>>>('./**/*messages.ts', { eager: true })
const vietnamese = import.meta.glob<Catalog>('./**/*messages.vi.ts', { eager: true, import: 'default' })

describe("conditional content copy", () => {
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
