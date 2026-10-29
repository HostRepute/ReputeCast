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

  // R4.A3: a section saved as a named synced block reads as one synced
  // card too; its words change in the brand kit, not here.
  it('reads a named synced block as one synced card, not texts to edit', async () => {
    const { editor } = await headlessMjmlEditor(
      email.replace('<mj-section rc-name="Feature highlights">', '<mj-section rc-name="Feature highlights" rc-synced="b-0123456789ab">'),
    )
    const items = docItems(editor)
    expect(items.map((x) => x.kind)).toEqual(['synced', 'image', 'text', 'synced', 'divider', 'other', 'synced'])
    expect(items[3]).toMatchObject({ kind: 'synced', synced: 'b-0123456789ab' })
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
    expect(chip.textContent).toBe('first_name · fallback “there”')
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
    // As the browser reads it: spaces of any kind, comments and broken tags hold nothing.
    expect(isBlank('<p> &nbsp; <br></p><!-- <b>note</b> -->')).toBe(true)
    expect(isBlank('<p>&nbsp;x</p>')).toBe(false)
    expect(isBlank('<p>a &lt; b</p>')).toBe(false)
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

describe('a text going into the page', () => {
  it('loses what could run there: handlers, scripts and javascript: links', async () => {
    const { editor } = await headlessMjmlEditor(email)
    const out = toEditable('<p><img src="x" onerror="alert(1)"> <a href="javascript:alert(2)" onclick="x()">link</a> {{ first_name }}</p><script>alert(3)</script><iframe srcdoc="<b>x</b>"></iframe>', editor)
    expect(out).not.toMatch(/onerror|onclick|javascript:|<script|<iframe|srcdoc/)
    expect(out).toContain('<img src="x">')
    expect(out).toContain('<a>link</a>')
    expect(out).toContain('data-rc-liquid="{{ first_name }}"')
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

  // R4.A1, CO-9: a paste from Notion, as its clipboard holds it.
  it('keeps headings, links and lists from Notion, one list per list, and drops its markup', () => {
    const notion =
      '<meta charset="utf-8"><!-- notionvc: 4f2a9c1e-7d3b-4e5f-9a8b-0c1d2e3f4a5b -->' +
      '<h1 id="1c9e" class="">Retiring the v1 API</h1>' +
      '<p id="2d0f" class="">On <strong>30 November</strong> the v1 endpoints stop answering. Read the <a href="https://docs.acme.io/migrate-v2">migration guide</a>.</p>' +
      '<ul id="3e1a" class="bulleted-list"><li style="list-style-type:disc">Requests to v1 return <code>410 Gone</code></li></ul>' +
      '<ul id="4f2b" class="bulleted-list"><li style="list-style-type:disc">API keys keep working</li></ul>' +
      '<ol type="1" id="5a3c" class="numbered-list" start="1"><li>Update the SDK</li></ol>' +
      '<ol type="1" id="6b4d" class="numbered-list" start="2"><li>Switch the base URL</li></ol>' +
      '<ul id="7c5e" class="to-do-list"><li><div class="checkbox checkbox-off"></div><span class="to-do-children-unchecked">Tell the team</span></li></ul>' +
      '<p id="8d6f" class=""></p>' +
      '<blockquote id="9e7a" class="">Questions? <em>Reply to this email.</em></blockquote>' +
      '<figure id="af8b" class="image"><img src="https://prod-files-secure.s3.us-west-2.amazonaws.com/x/roadmap.png"></figure>' +
      '<!-- notionvc: 4f2a9c1e-7d3b-4e5f-9a8b-0c1d2e3f4a5b -->'
    const out = cleanPaste(notion)
    expect(out.source).toBe('Notion')
    expect(out.kept).toEqual(['headings', 'links', 'lists'])
    expect(out.images).toBe(1)
    expect(out.removed).toBeGreaterThanOrEqual(10)
    expect(out.html).toBe(
      '<h2>Retiring the v1 API</h2>' +
        '<p>On <strong>30 November</strong> the v1 endpoints stop answering. Read the <a href="https://docs.acme.io/migrate-v2">migration guide</a>.</p>' +
        '<ul><li>Requests to v1 return 410 Gone</li><li>API keys keep working</li></ul>' +
        '<ol><li>Update the SDK</li><li>Switch the base URL</li></ol>' +
        '<ul><li>Tell the team</li></ul>' +
        '<p>Questions? <em>Reply to this email.</em></p>',
    )
    expect(out.html).not.toMatch(/notionvc|class=|style=|id=|img|checkbox/)
  })

  it('reads the paste in an inert document, so nothing in it loads or runs', () => {
    const out = cleanPaste('<p>Hi <img src="x" onerror="window.__ran = true"><b onclick="x()">there</b></p>')
    expect((window as unknown as { __ran?: boolean }).__ran).toBeUndefined()
    expect(out.html).toBe('Hi <strong>there</strong>')
  })

  it('keeps no comment, even one a pattern would leave behind', () => {
    const out = cleanPaste('<p>a<!-- hidden -->b<!--[if !supportLists]-->c<!--[endif]--></p><!<!---->--<script>x()</script>-->')
    // The broken comment's rest is only text, and the script goes.
    expect(out.html).toBe('<p>abc</p><p>----&gt;</p>')
  })

  it('drops unsafe links and keeps Liquid ones', () => {
    expect(cleanPaste('<a href="javascript:alert(1)">x</a> and <a href="{{ preferences_url }}">prefs</a>').html).toBe('x and <a href="{{ preferences_url }}">prefs</a>')
  })

  it('turns plain text into paragraphs', () => {
    expect(textToHtml('One\nline two\n\nNext <b>')).toBe('<p>One<br>line two</p><p>Next &lt;b&gt;</p>')
  })
})
