import { describe, expect, it } from 'vitest'
import { canvasHead, checkMjmlTags, expandSelfClosing, formatMjml, headCss, headMeta, isEmptyBody, joinMjml, splitMjml, withHeadMeta, withVisibilityStyles } from './mjmlDoc'

const doc = `<mjml>
  <mj-head>
    <mj-title>Old subject</mj-title>
    <mj-preview>Old preview</mj-preview>
    <mj-attributes><mj-all font-family="Georgia"></mj-all></mj-attributes>
    <mj-style>.x { color: red; }</mj-style>
  </mj-head>
  <mj-body width="600px">
    <mj-section><mj-column><mj-image src="a.png" alt="A" /><mj-text>Hi</mj-text></mj-column></mj-section>
  </mj-body>
</mjml>`

describe('splitMjml / joinMjml', () => {
  it('keeps the head next to the canvas body and joins them back', () => {
    const parts = splitMjml(doc)
    expect(parts.head).toContain('<mj-attributes>')
    expect(parts.body.startsWith('<mjml><mj-body width="600px">')).toBe(true)
    // Self-closing tags become pairs, so an HTML parser doesn't nest the siblings.
    expect(parts.body).toContain('<mj-image src="a.png" alt="A"></mj-image><mj-text>Hi</mj-text>')
    const joined = joinMjml(parts.head, parts.body)
    expect(splitMjml(joined)).toEqual(parts)
  })

  it('wraps loose content and handles an empty document', () => {
    expect(splitMjml('').body).toBe('<mjml><mj-body width="600px"></mj-body></mjml>')
    expect(splitMjml('<mj-section></mj-section>').body).toBe('<mjml><mj-body width="600px"><mj-section></mj-section></mj-body></mjml>')
  })
})

describe('head helpers', () => {
  it('writes the title and preview from the subject and preheader, escaped', () => {
    const head = withHeadMeta(splitMjml(doc).head, { title: 'Q4 & <more>', preview: '' })
    expect(headMeta(head)).toEqual({ title: 'Q4 & <more>', preview: undefined })
    expect(head).toContain('<mj-title>Q4 &amp; &lt;more&gt;</mj-title>')
  })

  it('passes only rendering parts to the canvas', () => {
    const h = canvasHead(splitMjml(doc).head)
    expect(h).toContain('<mj-attributes>')
    expect(h).toContain('<mj-style>')
    expect(h).not.toContain('mj-title')
    expect(headCss(splitMjml(doc).head)).toBe('.x { color: red; }')
  })

  it('adds and removes the visibility rule with its use', () => {
    const head = splitMjml(doc).head
    const both = withVisibilityStyles(head, { mobile: true, desktop: true })
    expect(both).toContain('.rc-hide-mobile')
    expect(both).toContain('.rc-hide-desktop')
    // The canvas dims hidden components instead of hiding them.
    expect(headCss(both)).toBe('.x { color: red; }')
    const mobile = withVisibilityStyles(both, { mobile: true, desktop: false })
    expect(mobile).not.toContain('rc-hide-desktop')
    expect(withVisibilityStyles(mobile, { mobile: false, desktop: false })).not.toContain('rc:visibility')
  })
})

describe('isEmptyBody', () => {
  it('tells the empty canvas apart', () => {
    expect(isEmptyBody('')).toBe(true)
    expect(isEmptyBody('<mjml><mj-body></mj-body></mjml>')).toBe(true)
    expect(isEmptyBody(doc)).toBe(false)
  })
})

describe('checkMjmlTags', () => {
  it('finds an unclosed element with its line', () => {
    expect(checkMjmlTags('<mjml>\n<mj-body>\n<mj-section>\n<mj-column>\n</mj-section>\n</mj-body>\n</mjml>')).toEqual({ line: 4, message: 'unclosed <mj-column>' })
    expect(checkMjmlTags('<mjml><mj-body></mj-body></mjml></mj-body>')).toEqual({ line: 1, message: '</mj-body> has no opening tag' })
  })

  it('ignores HTML inside text and comments', () => {
    expect(checkMjmlTags('<mjml><mj-body><!-- <mj-section> --><mj-section><mj-column><mj-text><div><b>x</mj-text></mj-column></mj-section></mj-body></mjml>')).toBeNull()
    expect(checkMjmlTags(doc)).toBeNull()
  })
})

describe('formatMjml', () => {
  it('indents elements and keeps text content on one line', () => {
    const out = formatMjml('<mjml><mj-body><mj-section><mj-column><mj-text>Hi <b>there</b></mj-text></mj-column></mj-section></mj-body></mjml>')
    expect(out).toBe(
      ['<mjml>', '  <mj-body>', '    <mj-section>', '      <mj-column>', '        <mj-text>Hi <b>there</b></mj-text>', '      </mj-column>', '    </mj-section>', '  </mj-body>', '</mjml>', ''].join('\n'),
    )
    expect(checkMjmlTags(formatMjml(doc))).toBeNull()
  })
})

describe('expandSelfClosing', () => {
  it('leaves comments and closed tags alone', () => {
    expect(expandSelfClosing('<!-- <mj-image /> --><mj-spacer height="10px"/>')).toBe('<!-- <mj-image /> --><mj-spacer height="10px"></mj-spacer>')
  })
})
