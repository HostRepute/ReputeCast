import type { Component, Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'
import { http, type Schemas } from '@/api/client'
import { t as tt } from '@/i18n'
import { mc } from './messages'

/**
 * The Discount code block (M59 G-22, ComposerTimedBlocks): a marker,
 * `<mj-raw><!-- rc:coupon {"pool":"welcome15",…} --></mj-raw>`, that the
 * send expands per reader (internal/coupons, liquid.RegisterBlock) into the
 * label, the reader's code in a dashed box and the button to the address
 * that applies it. The canvas draws it with the pool's sample code.
 */
export const COUPON_TYPE = 'rc-coupon'
const ATTR = 'data-rc-coupon'
const MARKER = /^\s*rc:coupon(?:\s+(\{[\s\S]*\}))?\s*$/

export interface CouponBlock {
  pool: string
  label: string
  button: string
  buttonUrl: boolean
}

type Pool = Schemas['CouponPoolListItem']

/** A new block's settings, in the app's language. */
export function newCoupon(pool = ''): CouponBlock {
  return { pool, label: tt(mc.defaultLabel), button: tt(mc.defaultButton), buttonUrl: true }
}

/** The marker: JSON in an HTML comment, with "--" kept out of it. */
export function couponMarker(b: CouponBlock): string {
  let json = JSON.stringify(b)
  while (json.includes('--')) json = json.replace(/--/g, '-\\u002d')
  return `<mj-raw><!-- rc:coupon ${json} --></mj-raw>`
}

export function isCouponBlock(c: Component | null | undefined): boolean {
  return c?.get('type') === COUPON_TYPE
}

export function couponOf(c: Component): CouponBlock {
  const raw = (c.get('attributes') as Record<string, unknown> | undefined)?.[ATTR]
  if (typeof raw === 'string') {
    try {
      return { ...newCoupon(), ...(JSON.parse(raw) as Partial<CouponBlock>) }
    } catch {
      /* the defaults */
    }
  }
  return newCoupon()
}

/** Changes a block's settings (an attribute change: undoable, saved). */
export function setCoupon(c: Component, b: CouponBlock) {
  c.addAttributes({ [ATTR]: JSON.stringify(b) })
}

// --- The workspace's pools, for the canvas and the inspector --------------------

let pools: Pool[] | null = null
let loading = false
const listeners = new Set<() => void>()

function slugNow(): string {
  const parts = window.location.pathname.split('/')
  return parts[1] === 'w' ? (parts[2] ?? '') : ''
}

/** The pools (loaded once; refreshCouponPools loads them again). */
export function couponPools(): Pool[] | null {
  if (!pools && !loading) {
    loading = true
    void http
      .GET('/api/v1/workspaces/{workspace}/coupons/pools', { params: { path: { workspace: slugNow() } } })
      .then(({ data }) => {
        pools = data?.items ?? []
      })
      .catch(() => {
        pools = []
      })
      .finally(() => {
        loading = false
        listeners.forEach((l) => l())
      })
  }
  return pools
}

export function refreshCouponPools() {
  pools = null
  couponPools()
}

export function subscribeCouponPools(l: () => void) {
  listeners.add(l)
  return () => {
    listeners.delete(l)
  }
}

// --- Canvas HTML --------------------------------------------------------------

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
const sans = "font-family:'IBM Plex Sans',system-ui,sans-serif"

/** The block as readers see it, with the pool's sample code (as the send's ExpandBlock draws it). */
export function couponHtml(b: CouponBlock, pool: Pool | undefined): string {
  if (!pool) {
    const why = b.pool ? tt(mc.canvasUnknown, { slug: b.pool }) : tt(mc.canvasPick)
    return `<div style="margin:8px 24px 24px;padding:18px;border:1px dashed #C9CED6;border-radius:8px;background:#F9FAFB;${sans};font-size:12.5px;line-height:1.5;color:#5B6472;text-align:center">${esc(why)}</div>`
  }
  const code = `${pool.prefix}SAMPLE1`
  const label = b.label ? `<div style="${sans};font-size:13px;font-weight:600;color:#4B5563;margin-bottom:8px">${esc(b.label)}</div>` : ''
  const hasUrl = pool.source !== 'upload' || !!pool.settings.applyUrl
  const button =
    b.buttonUrl && b.button && hasUrl
      ? `<div style="margin-top:14px"><span style="display:inline-block;padding:9px 14px;border-radius:6px;background:#0F766E;color:#FFFFFF;${sans};font-size:13px;font-weight:600">${esc(b.button)}</span></div>`
      : ''
  return (
    `<div style="padding:8px 24px 24px;text-align:center">${label}` +
    `<div style="display:inline-block;padding:12px 18px;border:2px dashed #6B7280;border-radius:8px;background:#F9FAFB;font-family:'IBM Plex Mono',ui-monospace,monospace;font-size:20px;font-weight:600;letter-spacing:.08em;color:#0B1220;user-select:all;max-width:100%;box-sizing:border-box;overflow-wrap:anywhere">${esc(code)}</div>` +
    `${button}</div>`
  )
}

type View = {
  model: Component
  el: HTMLElement
  render(): unknown
  listenTo(obj: unknown, ev: string, fn: () => void): void
}

export function loadCouponBlock(editor: Editor, { coreMjmlModel, coreMjmlView }: ComponentPluginOptions) {
  editor.Components.addType(COUPON_TYPE, {
    isComponent: (el: HTMLElement) => {
      if ((el.tagName || '').toLowerCase() !== 'mj-raw') return false
      const comment = Array.from(el.childNodes).find((n) => n.nodeType === Node.COMMENT_NODE) as Comment | undefined
      const found = comment ? MARKER.exec(comment.data) : null
      if (!found) return false
      let raw = JSON.stringify(newCoupon())
      if (found[1]) {
        try {
          raw = JSON.stringify({ ...newCoupon(), ...(JSON.parse(found[1].replace(/-\\u002d/g, '--')) as Partial<CouponBlock>) })
        } catch {
          /* the defaults */
        }
      }
      return { type: COUPON_TYPE, tagName: 'mj-raw', components: [], attributes: { [ATTR]: raw } }
    },
    model: {
      ...coreMjmlModel,
      defaults: {
        name: 'Discount code',
        tagName: 'mj-raw',
        draggable: '[data-gjs-type="mj-column"]',
        droppable: false,
        editable: false,
        stylable: false,
        'style-default': {},
        style: {},
        attributes: {},
      },
      handleAttributeChange() {},
      handleStyleChange() {},
      // The email keeps the marker; each reader gets their own code.
      toHTML(this: Component) {
        return couponMarker(couponOf(this))
      },
    },
    view: {
      ...coreMjmlView,
      tagName: 'div',
      attributes: { style: 'pointer-events: all; display: block;', 'data-rc-coupon-view': '' },
      init(this: View) {
        coreMjmlView.init.call(this)
        const rerender = () => this.render()
        const off = subscribeCouponPools(rerender)
        this.listenTo(this.model, 'removed', off)
        this.listenTo(this.model, `change:attributes:${ATTR}`, rerender)
      },
      getMjmlTemplate() {
        // Compiled on its own at body level: the HTML is drawn inside the
        // column the block sits in.
        return { start: '<mjml><mj-body>', end: '</mj-body></mjml>' }
      },
      getInnerMjmlTemplate(this: View) {
        const b = couponOf(this.model)
        const list = couponPools()
        const html = list ? couponHtml(b, list.find((p) => p.slug === b.pool)) : couponHtml(b, undefined)
        return { start: `<mj-raw>${html}</mj-raw>`, end: '' }
      },
      getTemplateFromEl(sandboxEl: HTMLElement) {
        // The body's wrapper div holds the block; it has no child
        // components, which render into an empty slot.
        const root = sandboxEl.querySelector('div') ?? sandboxEl
        return `${root.innerHTML}<div data-rc-slot="" style="display:none"></div>`
      },
      getChildrenSelector() {
        return '[data-rc-slot]'
      },
    },
  })
}

/** The Discount code blocks in an editor. */
export function couponBlocksIn(editor: Editor | null | undefined): Component[] {
  return editor?.getWrapper()?.findType(COUPON_TYPE) ?? []
}
