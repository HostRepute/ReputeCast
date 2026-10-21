import type { Component, Editor } from 'grapesjs'
import { afterEach, describe, expect, it } from 'vitest'
import { blockDefs } from './blocks'
import { blockAdded, blockContent, contentMarker, markerOf, tagOf } from './labels'

/** A headless GrapesJS with the MJML plugin, as the composer loads it. */
async function editorWith(body: string): Promise<Editor> {
  const { default: grapesjs } = await import('grapesjs')
  const mjmlModule = await import('grapesjs-mjml')
  const mjmlPlugin = ((mjmlModule.default as unknown as { default?: typeof mjmlModule.default }).default ?? mjmlModule.default) as typeof mjmlModule.default
  const container = document.createElement('div')
  document.body.appendChild(container)
  return grapesjs.init({
    container,
    headless: true,
    storageManager: false,
    telemetry: false,
    plugins: [(ed: Editor) => mjmlPlugin(ed, { blocks: [], resetBlocks: true })],
    components: `<mjml><mj-body>${body}</mj-body></mjml>`,
  })
}

const bodyOf = (ed: Editor) => ed.getWrapper()!.findType('mj-body')[0]!
const sections = (ed: Editor) => bodyOf(ed).components().filter((c: Component) => tagOf(c) === 'mj-section')
const asList = (added: Component | Component[]) => (Array.isArray(added) ? added : [added])
const productCard = () => blockDefs('500 Market St, San Francisco, CA 94105').find((d) => d.id === 'rc-product')!
const hero = '<!-- rc:hero --><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section>'

let editors: Editor[] = []
afterEach(() => {
  editors.forEach((e) => e.destroy())
  editors = []
})

async function canvas(body: string): Promise<Editor> {
  const ed = await editorWith(body)
  editors.push(ed)
  return ed
}

describe('a block keeps its <!-- rc:… --> marker on the canvas', () => {
  it('reads and splits the marker off a block content', () => {
    const def = productCard()
    expect(contentMarker(def.content)).toBe('product-card')
    expect(contentMarker('<mj-section></mj-section>')).toBe('')
    const parts = blockContent(def.content) as unknown[]
    expect(parts[0]).toEqual({ type: 'comment', content: ' rc:product-card ' })
    expect(String(parts[1]).startsWith('<mj-section')).toBe(true)
    // Content without a marker goes in as it is.
    expect(blockContent('<mj-section></mj-section>')).toBe('<mj-section></mj-section>')
  })

  it('a block clicked in is recognised by its marker', async () => {
    const ed = await canvas(hero)
    const body = bodyOf(ed)
    // The bug: the parser drops a comment that leads a string.
    const lost = asList(body.append(productCard().content, { at: body.components().length }))
    expect(markerOf(lost.find((c) => tagOf(c) === 'mj-section')!)).toBe('')
    lost.forEach((c) => c.remove())
    // As the Blocks panel adds it on a click.
    const added = blockAdded(ed, body.append(blockContent(productCard().content), { at: body.components().length }))
    expect(added && tagOf(added)).toBe('mj-section')
    expect(markerOf(added!)).toBe('product-card')
    expect(ed.getSelected()).toBe(added)
    expect(markerOf(sections(ed)[0]!)).toBe('hero')
    expect(ed.getHtml()).toMatch(/<!-- rc:hero -->[\s\S]*<!-- rc:product-card -->\s*<mj-section/)
  })

  it('a block dropped in gets its marker back', async () => {
    const ed = await canvas(hero)
    const body = bodyOf(ed)
    // A drop adds the block's content as one string (the marker is lost)…
    const dropped = body.append(productCard().content, { at: body.components().length })
    // …and the drop handler puts it back, with the block's own content.
    const added = blockAdded(ed, dropped, productCard().content)
    expect(markerOf(added!)).toBe('product-card')
    expect(ed.getSelected()).toBe(added)
    expect(bodyOf(ed).components().filter((c: Component) => c.get('type') === 'comment')).toHaveLength(2)
  })

  it('a block dropped between a marker and its section moves before the marker', async () => {
    const ed = await canvas(hero)
    const body = bodyOf(ed)
    // The canvas's drop index of "just above the hero" is after its marker.
    blockAdded(ed, body.append(productCard().content, { at: 1 }), productCard().content)
    const [card, first] = sections(ed)
    expect(markerOf(card!)).toBe('product-card')
    expect(markerOf(first!)).toBe('hero')
    expect(ed.getHtml()).toMatch(/<!-- rc:product-card -->\s*<mj-section[\s\S]*<!-- rc:hero -->\s*<mj-section/)
  })

  it('a block without a marker dropped above a marked section leaves it its marker', async () => {
    const ed = await canvas('<!-- rc:footer --><mj-section><mj-column><mj-text>Unsubscribe</mj-text></mj-column></mj-section>')
    const body = bodyOf(ed)
    blockAdded(ed, body.append('<mj-section><mj-column><mj-text>New</mj-text></mj-column></mj-section>', { at: 1 }), '<mj-section></mj-section>')
    const [plain, footer] = sections(ed)
    expect(markerOf(plain!)).toBe('')
    expect(markerOf(footer!)).toBe('footer')
  })
})
