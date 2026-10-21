import type { Component } from 'grapesjs'
import { describe, expect, it } from 'vitest'
import { fillProductCard, isProductCard } from './composer'

/*
 * A small stand-in for GrapesJS components: the tag, type, content and
 * attributes the composer's helpers read and write.
 */
interface Node {
  tag?: string
  type?: string
  content?: string
  attrs: Record<string, string>
  html?: string
  kids: Node[]
  parent?: Node
}

const wrapped = new Map<Node, Component>()
function wrap(n: Node): Component {
  let c = wrapped.get(n)
  if (c) return c
  const fake = {
    get: (k: string) => (k === 'tagName' ? (n.tag ?? '') : k === 'type' ? (n.type ?? 'default') : k === 'content' ? (n.content ?? '') : undefined),
    set: (k: string, v: string) => {
      if (k === 'content') n.content = v
      else n.attrs[`prop:${k}`] = v
    },
    parent: () => (n.parent ? wrap(n.parent) : undefined),
    getAttributes: () => n.attrs,
    addAttributes: (a: Record<string, string>) => Object.assign(n.attrs, a),
    removeAttributes: (keys: string[]) => keys.forEach((k) => delete n.attrs[k]),
    components: (html?: string) => {
      if (html !== undefined) {
        n.html = html
        n.kids = []
        return []
      }
      const models = n.kids.map(wrap)
      return { models, length: models.length, filter: (f: (x: Component) => boolean) => models.filter(f), forEach: (f: (x: Component) => void) => models.forEach(f) }
    },
  }
  c = fake as unknown as Component
  wrapped.set(n, c)
  return c
}

function el(tag: string, kids: Node[] = [], attrs: Record<string, string> = {}): Node {
  const n: Node = { tag, attrs, kids }
  kids.forEach((k) => (k.parent = n))
  return n
}
const comment = (text: string): Node => ({ type: 'comment', content: text, attrs: {}, kids: [] })
/** The Product card block (editor/blocks.ts): an image column beside the name, the price line and a button. */
const card = () =>
  el('mj-section', [
    el('mj-column', [el('mj-image', [], { src: '/media/starters/product.png', alt: 'Describe the product' })]),
    el('mj-column', [el('mj-text'), el('mj-text'), el('mj-button', [], { href: 'https://example.com' })]),
  ])
function body(...kids: Node[]) {
  return el('mj-body', kids)
}

describe('the Product card', () => {
  it('is the section after an rc:product-card marker', () => {
    const s = card()
    body(comment(' rc:product-card '), s)
    expect(isProductCard(wrap(s))).toBe(true)
  })
  it('is a section shaped like the block when the marker is gone (a block inserted from the panel)', () => {
    const s = card()
    body(el('mj-section'), s)
    expect(isProductCard(wrap(s))).toBe(true)
  })
  it('isn’t another block, even shaped like it', () => {
    const cards = card()
    body(comment(' rc:product-cards '), cards)
    expect(isProductCard(wrap(cards))).toBe(false)
    const article = el('mj-section', [el('mj-column', [el('mj-image'), el('mj-text'), el('mj-text'), el('mj-button')])])
    body(article)
    expect(isProductCard(wrap(article))).toBe(false)
    const mixed = el('mj-section', [el('mj-column', [el('mj-image'), el('mj-text')]), el('mj-column', [el('mj-text'), el('mj-text'), el('mj-button')])])
    body(mixed)
    expect(isProductCard(wrap(mixed))).toBe(false)
    expect(isProductCard(wrap(el('mj-text')))).toBe(false)
  })
  it('takes the product’s cover, name, price line and page', () => {
    const s = card()
    body(s)
    fillProductCard(wrap(s), { src: '/media/lk/launch-kit.png', alt: 'Cover of the Newsletter Launch Kit', name: 'Tips & tricks <2026>', price: 'Pay what you want · from $5', href: 'https://news.acme.io/launch-kit' })
    const [left, right] = s.kids
    expect(left!.kids[0]!.attrs).toMatchObject({ src: '/media/lk/launch-kit.png', alt: 'Cover of the Newsletter Launch Kit', 'prop:src': '/media/lk/launch-kit.png' })
    expect(right!.kids[0]!.html).toBe('Tips &amp; tricks &lt;2026&gt;')
    expect(right!.kids[1]!.html).toBe('Pay what you want · from $5')
    expect(right!.kids[2]!.attrs.href).toBe('https://news.acme.io/launch-kit')
  })
})
