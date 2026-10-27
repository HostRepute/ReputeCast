import { describe, expect, it } from 'vitest'
import { headlessMjmlEditor } from '../editor/testEditor'
import { docItems, fromEditable, isBlank, textHtml, toEditable } from './doc'
import { applyBold, applyShortcut, isDividerLine, shortcutOf } from './markdown'
import { cleanPaste, textToHtml } from './paste'

// CO-9: Write mode's document, markdown shortcuts and paste cleanup.

const email = `<mjml><mj-body>
<mj-section rc-synced="header"><mj-column><mj-image src="/media/logo.png" alt="Acme"></mj-image></mj-column></mj-section>
<mj-section><mj-column><mj-image src="/media/hero.png" alt="Q4 roadmap illustration"></mj-image></mj-column></mj-section>
<mj-section><mj-column><mj-text font-size="15px"><h1>Three launches</h1><p>Hi {{ first_name | default: "there" }}, here's what shipped.</p></mj-text></mj-column></mj-section>
<mj-section rc-name="Feature highlights"><mj-column><mj-text><h3>Feature flags</h3></mj-text><mj-button href="https://docs.acme.io/flags">Read the docs</mj-button></mj-column><mj-column><mj-text><h3>Audit log exports</h3></mj-text><mj-button href="https://acme.io/changelog">See the changelog</mj-button></mj-column></mj-section>
<mj-section><mj-column><mj-divider></mj-divider><mj-social><mj-social-element name="x" href="https://x.com/acme">X</mj-social-element></mj-social></mj-column></mj-section>
<mj-section rc-synced="footer"><mj-column><mj-text>Acme Inc · <a href="{{ unsubscribe_url }}">Unsubscribe</a></mj-text></mj-column></mj-section>
</mj-body></mjml>`

describe('write mode document', () => {
  it('reads the email in order: synced cards, the hero, texts, columns, buttons', async () => {
    const { editor } = await headlessMjmlEditor(email)
    const items = docItems(editor)
    expect(items.map((x) => x.kind)).toEqual(['synced', 'image', 'text', 'column', 'text', 'button', 'column', 'text', 'button', 'divider', 'other', 'synced'])
    expect(items[0]).toMatchObject({ kind: 'synced', synced: 'header' })
    expect(items[1]).toMatchObject({ kind: 'image', hero: true })
    expect(items[3]).toMatchObject({ kind: 'column', section: 'Feature highlights', n: 1, count: 2 })
    expect(items[6]).toMatchObject({ kind: 'column', n: 2, count: 2 })
    expect(items[11]).toMatchObject({ kind: 'synced', synced: 'footer' })
  })

  it('edits a text through chips and back without touching anything else', async () => {
    const { editor } = await headlessMjmlEditor(email)
    const before = editor.getHtml()
    const text = docItems(editor).find((x) => x.kind === 'text')!
    if (text.kind !== 'text') throw new Error('no text')
    const html = textHtml(text.comp)
    expect(html).toContain('{{ first_name | default: "there" }}')
    const el = document.createElement('div')
    el.innerHTML = toEditable(html, editor)
    // The token shows as a chip that can't be typed into…
    const chip = el.querySelector('[data-rc-liquid]')!
    expect(chip.getAttribute('contenteditable')).toBe('false')
    expect(chip.textContent).toBe('first_name · “there”')
    // …and goes back as the same Liquid.
    expect(fromEditable(el)).toBe(html)
    // Nothing edited: the email is the same.
    text.comp.components(fromEditable(el))
    expect(editor.getHtml()).toBe(before)
    // An edit changes that text only.
    el.querySelector('p')!.append(' And more.')
    text.comp.components(fromEditable(el))
    const after = editor.getHtml()
    expect(after).toContain('And more.')
    expect(after.replace(' And more.', '')).toBe(before)
  })

  it('knows an emptied text', () => {
    expect(isBlank('<h2><br></h2>')).toBe(true)
    expect(isBlank('<p>&nbsp;</p>')).toBe(true)
    expect(isBlank('<p>{{ first_name }}</p>')).toBe(false)
    expect(isBlank('<img src="a.png">')).toBe(false)
  })
})

describe('markdown shortcuts', () => {
  const para = (html: string) => {
    const root = document.createElement('div')
    root.innerHTML = `<p>${html}</p>`
    return { root, p: root.firstElementChild as HTMLElement }
  }

  it('reads the shortcuts', () => {
    expect(shortcutOf('# Title')).toEqual({ kind: 'heading', rest: 'Title' })
    expect(shortcutOf('## Sub')).toEqual({ kind: 'subheading', rest: 'Sub' })
    expect(shortcutOf('- item')).toEqual({ kind: 'bullets', rest: 'item' })
    expect(shortcutOf('*\u00a0item')).toEqual({ kind: 'bullets', rest: 'item' })
    expect(shortcutOf('1. step')).toEqual({ kind: 'numbers', rest: 'step' })
    expect(shortcutOf('#hashtag')).toBeNull()
  })

  it('turns "# " into a heading and "- " into a list, keeping the words', () => {
    let { root, p } = para('# Retiring <b>v1</b>')
    expect(applyShortcut(p)?.tagName).toBe('H2')
    expect(root.innerHTML).toBe('<h2>Retiring <b>v1</b></h2>')
    ;({ root, p } = para('- first point'))
    expect(applyShortcut(p)?.tagName).toBe('LI')
    expect(root.innerHTML).toBe('<ul><li>first point</li></ul>')
    ;({ root, p } = para('1. step'))
    applyShortcut(p)
    expect(root.innerHTML).toBe('<ol><li>step</li></ol>')
    expect(applyShortcut(para('plain words').p)).toBeNull()
  })

  it('makes **words** bold and knows a divider line', () => {
    const { root, p } = para('make it **bold**')
    applyBold(p.firstChild as Text)
    expect(root.innerHTML).toBe('<p>make it <strong>bold</strong>&nbsp;</p>')
    expect(isDividerLine(para('---').p)).toBe(true)
    expect(isDividerLine(para('-- not').p)).toBe(false)
  })
})

describe('paste cleanup', () => {
  it('keeps headings, links, lists, bold and italics from Google Docs, and drops the styles', () => {
    const gdocs =
      '<meta charset="utf-8"><b style="font-weight:normal;" id="docs-internal-guid-1234"><h2 dir="ltr" style="line-height:1.38;margin-top:18pt"><span style="font-size:16pt;font-family:Arial;color:#000000;font-weight:400">What changes</span></h2>' +
      '<ul style="margin-top:0;margin-bottom:0"><li dir="ltr" style="list-style-type:disc"><p dir="ltr" style="line-height:1.38"><span style="font-size:11pt;font-family:Arial">Requests to v1 return </span><span style="font-size:11pt;font-family:Arial;font-weight:700">410 Gone</span></p></li></ul>' +
      '<p dir="ltr" style="line-height:1.38"><span style="font-size:11pt">See the </span><a href="https://docs.acme.io/migrate-v2" style="text-decoration:none"><span style="color:#1155cc;text-decoration:underline">migration guide</span></a><span style="font-style:italic"> soon</span>.</p>' +
      '<p><img src="https://lh7-us.googleusercontent.com/x.png" width="600"></p><script>alert(1)</script></b>'
    const out = cleanPaste(gdocs)
    expect(out.source).toBe('Google Docs')
    expect(out.kept).toEqual(['headings', 'links', 'lists'])
    expect(out.images).toBe(1)
    expect(out.removed).toBeGreaterThanOrEqual(10)
    expect(out.html).toBe(
      '<h2>What changes</h2><ul><li><p>Requests to v1 return <strong>410 Gone</strong></p></li></ul><p>See the <a href="https://docs.acme.io/migrate-v2">migration guide</a><em> soon</em>.</p>',
    )
    expect(out.html).not.toMatch(/style=|script|img/)
  })

  it('drops unsafe links and keeps Liquid ones', () => {
    expect(cleanPaste('<a href="javascript:alert(1)">x</a> and <a href="{{ preferences_url }}">prefs</a>').html).toBe('x and <a href="{{ preferences_url }}">prefs</a>')
  })

  it('turns plain text into paragraphs', () => {
    expect(textToHtml('One\nline two\n\nNext <b>')).toBe('<p>One<br>line two</p><p>Next &lt;b&gt;</p>')
  })
})
