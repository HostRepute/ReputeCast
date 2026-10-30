import { describe, expect, it } from 'vitest'
import { prepareCompile } from '@/features/conditional/mjml'
import type { Catalog, MessageDescriptor } from '@/i18n'

// Show on (the publication website): a Web only block is the post's, so the
// email the composer compiles never has it; an Email only block stays.
describe('prepareCompile and Show on', () => {
  const doc = `<mjml><mj-body>
    <mj-section><mj-column><mj-text>Both</mj-text></mj-column></mj-section>
    <mj-section rc-show-on="web"><mj-column><mj-text>Read more on the site</mj-text><mj-text rc-visible='{"id":"web005","show":"only"}'>Inside</mj-text></mj-column></mj-section>
    <mj-section><mj-column><mj-text rc-show-on="web">Web line</mj-text><mj-text rc-show-on="email">Email line</mj-text></mj-column></mj-section>
  </mj-body></mjml>`

  it('leaves Web only blocks out, with what they hold', () => {
    const out = prepareCompile(doc)
    for (const gone of ['Read more on the site', 'Inside', 'web005', 'Web line', 'rc-show-on']) expect(out).not.toContain(gone)
    for (const kept of ['Both', 'Email line']) expect(out).toContain(kept)
  })

  it('leaves a document without them alone', () => {
    const plain = '<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>'
    expect(prepareCompile(plain)).toBe(plain)
  })
})

// X-1: the website and its posts say what the reader's role can't do, or who
// can, in words (website:manage never shows). The whole website feature, from here.
const permissionKey = /\b[a-z]+:(?:view|edit|send|pause|approve|publish|content|create|import|export|delete|verify|manage|sql|share|hostrepute|invite|use|run|review|connect|transfer|usage|library)\b/
const english = import.meta.glob<Record<string, Record<string, MessageDescriptor>>>('../**/*messages.ts', { eager: true })
const vietnamese = import.meta.glob<Catalog>('../**/*messages.vi.ts', { eager: true, import: 'default' })

describe("the website's copy", () => {
  it('names no permission key, in English or Vietnamese', () => {
    const en = Object.values(english).flatMap((file) => Object.values(file).flatMap((group) => Object.values(group)))
    const vi = Object.values(vietnamese).flatMap((catalog) => Object.entries(catalog))
    expect(en.length).toBeGreaterThan(0)
    expect(vi.length).toBeGreaterThan(0)
    for (const d of en) {
      expect(d.message, d.id).not.toMatch(permissionKey)
      // Nor does a call site pass one in: no {perm}, {perms} or {permission}
      // to fill, but for the post rule's note, whose {perm} Settings › Approvals
      // fills with words (“publishing posts and pages”).
      if (d.id !== 'website.elsewhere.policyPostNote') expect(d.message, d.id).not.toMatch(/\{perm(?:s|ission)?\}/)
    }
    for (const [id, text] of vi) expect(text, id).not.toMatch(permissionKey)
  })
})
