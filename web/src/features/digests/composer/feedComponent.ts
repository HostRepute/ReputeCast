import type { Component, Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'
import { t as tt } from '@/i18n'
import { dc } from './messages'

/**
 * The Feed items block on the composer's canvas (digest mode). In the
 * template it is a marker, `<mj-raw><!-- rc:feed-items {…} --></mj-raw>`,
 * that each issue's preparation replaces with its posts (internal/digests
 * block.go). On the canvas it shows the posts of the "Preview with" choice,
 * rendered by the server (POST …/digests/{id}/render) and compiled here by
 * the same mjml-browser as the rest of the canvas.
 */
export const FEED_TYPE = 'rc-feed-items'
const BLOCK_ATTR = 'data-rc-block'
const MARKER = /^\s*rc:feed-items\s+(\{[\s\S]*\})\s*$/

/** The block's settings, as the marker carries them (DigestFeedBlock). */
export interface FeedBlock {
  layout: 'list' | 'cards' | 'grid'
  fields: string[]
  excerptWords: number
  buttonText: string
  max: number
  order: 'newest' | 'oldest'
  overflow: 'drop' | 'keep'
  fallback?: string
  fallbackName?: string
}

export const DEFAULT_BLOCK: FeedBlock = {
  layout: 'cards',
  fields: ['image', 'title', 'date', 'author', 'excerpt', 'button'],
  excerptWords: 40,
  buttonText: 'Read the post',
  max: 6,
  order: 'newest',
  overflow: 'drop',
}

/** The marker for a block: JSON in an HTML comment, with "--" kept out of it. */
export function markerFor(b: FeedBlock): string {
  let json = JSON.stringify(b)
  while (json.includes('--')) json = json.replace(/--/g, '-\\u002d')
  return `<mj-raw><!-- rc:feed-items ${json} --></mj-raw>`
}

/** Reads a template's Feed items block (the first marker), if it has one. */
export function blockOfMjml(mjml: string): FeedBlock | null {
  const m = /<!--\s*rc:feed-items\s+(\{[\s\S]*?\})\s*-->/.exec(mjml)
  if (!m) return null
  try {
    return { ...DEFAULT_BLOCK, ...(JSON.parse(m[1]!) as Partial<FeedBlock>) }
  } catch {
    return { ...DEFAULT_BLOCK }
  }
}

/** The block settings of a Feed items component. */
export function blockOf(c: Component): FeedBlock {
  const raw = (c.get('attributes') as Record<string, unknown> | undefined)?.[BLOCK_ATTR]
  if (typeof raw === 'string') {
    try {
      return { ...DEFAULT_BLOCK, ...(JSON.parse(raw) as Partial<FeedBlock>) }
    } catch {
      /* the default */
    }
  }
  return { ...DEFAULT_BLOCK }
}

/** Changes a Feed items component's settings (an attribute change: undoable, saved, tracked). */
export function setBlock(c: Component, b: FeedBlock) {
  c.addAttributes({ [BLOCK_ATTR]: JSON.stringify(b) })
}

export function isFeedItems(c: Component | null | undefined): boolean {
  return Boolean(c && c.get('type') === FEED_TYPE)
}

/** The canvas selection's label: "Feed items · Cards". */
export function feedItemsLabel(c: Component): string {
  return tt(dc.selectionLabel, { layout: blockOf(c).layout })
}

const chipStyle =
  "position:absolute;left:12px;display:inline-flex;align-items:center;gap:5px;height:20px;padding:0 8px;border-radius:999px;border:1px solid #99F6E4;background:#F0FDFA;color:#0F766E;font-family:'IBM Plex Sans',system-ui,sans-serif;font-size:11px;font-weight:600;line-height:1;white-space:nowrap;z-index:2;pointer-events:none"
const chipIcon =
  '<svg width="11" height="11" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="4" cy="12" r="1.3"></circle><path d="M3 7.5a5.5 5.5 0 0 1 5.5 5.5M3 3a10 10 0 0 1 10 10"></path></svg>'

/**
 * Marks each post on the canvas with a "Feed item 1 of 2" chip, above its
 * column. The chips are the canvas's only: the template keeps the marker.
 */
function addChips(root: HTMLElement, layout: FeedBlock['layout']) {
  const posts = Array.from(root.querySelectorAll<HTMLElement>('.rc-feed-post'))
  posts.forEach((el, i) => {
    const first = layout === 'grid' ? i < 2 : i === 0
    const chip = el.ownerDocument.createElement('span')
    // In the top padding of the first row; on the divider of the others (the grid has none).
    chip.setAttribute('style', `${chipStyle};top:${first ? -24 : layout === 'grid' ? -20 : 4}px`)
    chip.setAttribute('data-rc-chip', '')
    chip.innerHTML = chipIcon
    chip.append(el.ownerDocument.createTextNode(tt(dc.feedItemOf, { number: i + 1, total: posts.length })))
    el.style.position = 'relative'
    el.prepend(chip)
  })
}

// The canvas preview: the posts' sections, shared by every Feed items
// component of the canvas (a template has one).
let previewMjml = ''
const listeners = new Set<() => void>()

/** Shows these sections in the Feed items block on the canvas. */
export function setFeedPreview(mjml: string) {
  if (mjml === previewMjml) return
  previewMjml = mjml
  listeners.forEach((l) => l())
}

/** A gray placeholder while the posts load. */
const placeholder = `<mj-section background-color="#ffffff" padding="24px"><mj-column><mj-text align="center" color="#5B6472" font-size="13px">Feed items</mj-text></mj-column></mj-section>`

type View = {
  model: Component
  el: HTMLElement
  render(): unknown
  listenTo(obj: unknown, ev: string, fn: () => void): void
}

export function loadFeedItems(editor: Editor, { coreMjmlModel, coreMjmlView }: ComponentPluginOptions) {
  editor.Components.addType(FEED_TYPE, {
    isComponent: (el: HTMLElement) => {
      if ((el.tagName || '').toLowerCase() !== 'mj-raw') return false
      const comment = Array.from(el.childNodes).find((n) => n.nodeType === Node.COMMENT_NODE) as Comment | undefined
      const m = comment ? MARKER.exec(comment.data) : null
      if (!m) return false
      let block = DEFAULT_BLOCK
      try {
        block = { ...DEFAULT_BLOCK, ...(JSON.parse(m[1]!.replace(/-\\u002d/g, '--')) as Partial<FeedBlock>) }
      } catch {
        /* the default */
      }
      return { type: FEED_TYPE, tagName: 'mj-raw', components: [], attributes: { [BLOCK_ATTR]: JSON.stringify(block) } }
    },
    model: {
      ...coreMjmlModel,
      defaults: {
        name: 'Feed items',
        tagName: 'mj-raw',
        draggable: '[data-gjs-type="mj-body"], [data-gjs-type="mj-wrapper"]',
        droppable: false,
        editable: false,
        copyable: false,
        stylable: false,
        'style-default': {},
        style: {},
        attributes: {},
      },
      init(this: Component, ...a: unknown[]) {
        ;(coreMjmlModel.init as (this: Component, ...a: unknown[]) => void).apply(this, a)
        // The image library sets "src" on the component it was opened for: for
        // Feed items that is the fallback image.
        this.on('change:attributes:src', () => {
          const src = (this.get('attributes') as Record<string, unknown>).src
          if (typeof src !== 'string' || !src) return
          const attrs = { ...(this.get('attributes') as Record<string, unknown>) }
          delete attrs.src
          attrs[BLOCK_ATTR] = JSON.stringify({ ...blockOf(this), fallback: src, fallbackName: decodeURIComponent(src.split('/').pop() ?? '') })
          this.set('attributes', attrs)
        })
      },
      // Attributes aren't mirrored into styles (the block has none).
      handleAttributeChange() {},
      handleStyleChange() {},
      // The template keeps the marker; issues get the posts.
      toHTML(this: Component) {
        return markerFor(blockOf(this))
      },
    },
    view: {
      ...coreMjmlView,
      tagName: 'section',
      attributes: { style: 'pointer-events: all; display: block;', 'data-rc-feed': '' },
      init(this: View) {
        coreMjmlView.init.call(this)
        const rerender = () => this.render()
        listeners.add(rerender)
        this.listenTo(this.model, 'removed', () => listeners.delete(rerender))
      },
      getMjmlTemplate(this: View) {
        const parent = this.model.parent()
        const parentView = parent?.view as unknown as { getInnerMjmlTemplate?: () => { start: string; end: string } } | undefined
        if (parentView?.getInnerMjmlTemplate && parent?.get('tagName') === 'mj-body') {
          const body = coreMjmlView.getInnerMjmlTemplate.call(parentView) as { start: string; end: string }
          return { start: `<mjml>${body.start}`, end: `${body.end}</mjml>` }
        }
        return { start: '<mjml><mj-body>', end: '</mj-body></mjml>' }
      },
      getInnerMjmlTemplate() {
        return { start: previewMjml || placeholder, end: '' }
      },
      getTemplateFromEl(this: View, sandboxEl: HTMLElement) {
        // The body's wrapper div holds the sections. The block has no child
        // components: they render into an empty slot, so the posts stay.
        const root = sandboxEl.querySelector('div') ?? sandboxEl
        addChips(root, blockOf(this.model).layout)
        return `${root.innerHTML}<div data-rc-slot="" style="display:none"></div>`
      },
      getChildrenSelector() {
        return '[data-rc-slot]'
      },
    },
  })
}
