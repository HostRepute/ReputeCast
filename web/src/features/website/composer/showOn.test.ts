import { describe, expect, it } from 'vitest'
import { prepareCompile } from '@/features/conditional/mjml'

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
