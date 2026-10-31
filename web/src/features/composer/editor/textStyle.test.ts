import { describe, expect, it, vi } from 'vitest'
import { applyTextStyle, blockAt, currentTextStyle, textStyleOf, type TextEditing } from './textStyle'

// CO-8: the text toolbar's headings (Text style) on the text being edited.
// jsdom has no execCommand: formatBlock is done by hand here, as browsers do.

function editing(html: string, caretIn: string): TextEditing & { exec: ReturnType<typeof vi.fn> } {
  const el = document.createElement('div')
  el.contentEditable = 'true'
  el.innerHTML = html
  document.body.appendChild(el)
  const place = (node: Node) => {
    const r = document.createRange()
    r.selectNodeContents(node)
    r.collapse(false)
    const s = document.getSelection()!
    s.removeAllRanges()
    s.addRange(r)
  }
  place(el.querySelector(caretIn)!.firstChild!)
  const exec = vi.fn((cmd: string, value?: string) => {
    if (cmd !== 'formatBlock') return
    const block = blockAt(document.getSelection()!.anchorNode, el)!
    const next = document.createElement(value!.replace(/[<>]/g, ''))
    next.append(...Array.from(block.childNodes))
    block.replaceWith(next)
    place(next.lastChild!)
  })
  return { exec, selection: () => document.getSelection(), el, doc: document }
}

describe('text styles', () => {
  it('reads the style of the block the caret is in', () => {
    const div = document.createElement('div')
    div.innerHTML = '<h1>Big</h1><h2>Title</h2><h3>Sub</h3><p><small>Fine print</small></p><p>Body <small>aside</small></p>'
    const [h1, h2, h3, small, p] = Array.from(div.children) as HTMLElement[]
    expect([h1, h2, h3, small, p].map((b) => textStyleOf(b!))).toEqual(['heading', 'heading', 'subheading', 'small', 'p'])
    expect(textStyleOf(null)).toBe('p')
  })

  it('makes a paragraph a heading, a subheading and small print, and back', () => {
    const rte = editing('<p>Retiring the v1 API</p><p>On 30 November.</p>', 'p')
    expect(currentTextStyle(rte)).toBe('p')
    applyTextStyle(rte, 'heading')
    expect(rte.el.innerHTML).toBe('<h2>Retiring the v1 API</h2><p>On 30 November.</p>')
    expect(currentTextStyle(rte)).toBe('heading')
    applyTextStyle(rte, 'subheading')
    expect(rte.el.innerHTML).toBe('<h3>Retiring the v1 API</h3><p>On 30 November.</p>')
    applyTextStyle(rte, 'small')
    expect(rte.el.innerHTML).toBe('<p><small>Retiring the v1 API</small></p><p>On 30 November.</p>')
    expect(currentTextStyle(rte)).toBe('small')
    applyTextStyle(rte, 'p')
    expect(rte.el.innerHTML).toBe('<p>Retiring the v1 API</p><p>On 30 November.</p>')
  })

  it('keeps an h1 as it is when Heading is picked again', () => {
    const rte = editing('<h1>Weekly digest</h1>', 'h1')
    applyTextStyle(rte, 'heading')
    expect(rte.exec).not.toHaveBeenCalled()
    expect(rte.el.innerHTML).toBe('<h1>Weekly digest</h1>')
  })

  it('tells the editor about changes it makes itself', () => {
    const rte = editing('<p>Footer words</p>', 'p')
    const input = vi.fn()
    rte.el.addEventListener('input', input)
    applyTextStyle(rte, 'small')
    expect(input).toHaveBeenCalledTimes(1)
  })
})
