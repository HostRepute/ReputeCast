import type { Component, Editor } from 'grapesjs'
import type { ComponentPluginOptions } from 'grapesjs-mjml'
import { useSyncExternalStore } from 'react'
import { ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { t as tt } from '@/i18n'
import { dayMonthDate, clock } from '@/lib/format'
import type { BlockDef } from '@/features/composer/editor/blocks'
import { mb } from './messages'

/**
 * Grow's blocks on the composer's canvas (ComposerGrowBlocks): Referral and
 * Recommendations in any email, Reward in milestone emails. Each is a
 * marker, `<mj-raw><!-- rc:referral {…} --></mj-raw>`, that the send
 * expands per reader (internal/referrals, internal/recommendations). On the
 * canvas the server renders the block for the "Preview as" reader; a block
 * that would be hidden shows why, dimmed, with its actions.
 */
export type GrowKind = 'referral' | 'recommendations' | 'reward'

export const GROW_TYPES: Record<GrowKind, string> = { referral: 'rc-referral', recommendations: 'rc-recommendations', reward: 'rc-reward' }
const KINDS = Object.keys(GROW_TYPES) as GrowKind[]
const ATTR = 'data-rc-grow'
const MARKER = /^\s*rc:(referral|recommendations|reward)(?:\s+(\{[\s\S]*\}))?\s*$/

export type ReferralSettings = Schemas['ReferralBlock']
export type RecsSettings = Schemas['RecommendationsBlock']
export type RewardSettings = Schemas['RewardBlock']
type Raw = Record<string, unknown>

/** A new block's settings (the server fills in the rest). */
const NEW: Record<GrowKind, Raw> = {
  referral: { layout: 'progress', share: ['email', 'x', 'linkedin', 'whatsapp'], button: 'Share your link', progress: true },
  recommendations: { which: 'all', ids: [], layout: 'stacked', heading: 'Newsletters we recommend', intro: 'Picked by the team.' },
  reward: { button: 'Get your reward', image: true, description: true, expiry: true, hub: true },
}

/** The marker for a block: JSON in an HTML comment, with "--" kept out of it. */
export function growMarker(kind: GrowKind, settings: Raw): string {
  let json = JSON.stringify(settings)
  while (json.includes('--')) json = json.replace(/--/g, '-\\u002d')
  return `<mj-raw><!-- rc:${kind} ${json} --></mj-raw>`
}

/** The Grow group of the Blocks panel; the Reward block only in milestone emails. */
export function growBlockDefs(milestone = false): BlockDef[] {
  const out: BlockDef[] = [
    {
      id: 'rc-referral',
      label: tt(mb.blockReferral),
      group: 'grow',
      icon: 'growReferral',
      kind: 'section',
      content: growMarker('referral', NEW.referral),
      keywords: 'referral refer invite share friends rewards link grow',
    },
    {
      id: 'rc-recommendations',
      label: tt(mb.blockRecommendations),
      group: 'grow',
      icon: 'growRecs',
      kind: 'section',
      content: growMarker('recommendations', NEW.recommendations),
      keywords: 'recommendations recommend newsletters partners grow',
    },
  ]
  if (milestone)
    out.unshift({ id: 'rc-reward', label: tt(mb.blockReward), group: 'grow', icon: 'growReward', kind: 'section', content: growMarker('reward', NEW.reward), keywords: 'reward milestone download code grow' })
  return out
}

export function growKindOf(c: Component | null | undefined): GrowKind | null {
  const type = c?.get('type')
  return KINDS.find((k) => GROW_TYPES[k] === type) ?? null
}

export function isGrowBlock(c: Component | null | undefined): boolean {
  return growKindOf(c) !== null
}

/** The settings a block carries (what its marker says; the server fills in defaults). */
export function growRaw(c: Component): Raw {
  const raw = (c.get('attributes') as Raw | undefined)?.[ATTR]
  if (typeof raw === 'string') {
    try {
      const v = JSON.parse(raw) as unknown
      if (v && typeof v === 'object') return v as Raw
    } catch {
      /* none */
    }
  }
  return {}
}

/** Changes a block's settings (an attribute change: undoable, saved, tracked). */
export function setGrowRaw(c: Component, settings: Raw) {
  c.addAttributes({ [ATTR]: JSON.stringify(settings) })
}

/** "Referral block · card with progress", for the selection and the layers. */
export function growBlockLabel(c: Component): string {
  const kind = growKindOf(c)
  const raw = growRaw(c)
  if (kind === 'referral') return tt(mb.labelReferral, { layout: String(raw.layout ?? 'progress') })
  if (kind === 'recommendations') return tt(mb.labelRecommendations, { layout: String(raw.layout ?? 'stacked') })
  return tt(mb.labelReward)
}

/** The block's name in the layer tree and the warnings ("Referral block"). */
export function growSectionLabel(c: Component): string {
  const kind = growKindOf(c)
  return kind === 'referral' ? tt(mb.nameReferral) : kind === 'recommendations' ? tt(mb.nameRecommendations) : tt(mb.nameReward)
}

// --- Preview and rendering ------------------------------------------------------

export interface GrowPreview {
  /** The workspace; empty reads it from the address (/w/{slug}/…). */
  slug: string
  /** The reader the blocks render for: "" for the sample, null until the composer picks one. */
  contactId: string | null
  /** The milestone email being edited (the Reward block reads its reward). */
  milestoneId: string
}

let preview: GrowPreview = { slug: '', contactId: null, milestoneId: '' }
const listeners = new Set<() => void>()
const notify = () => listeners.forEach((l) => l())

function subscribe(l: () => void) {
  listeners.add(l)
  return () => listeners.delete(l)
}

/** Sets who and what the canvas previews Grow blocks as; their views render again. */
export function setGrowPreview(next: Partial<GrowPreview>) {
  const merged = { ...preview, ...next }
  if (merged.slug === preview.slug && merged.contactId === preview.contactId && merged.milestoneId === preview.milestoneId) return
  preview = merged
  notify()
}

export function useGrowPreview(): GrowPreview {
  return useSyncExternalStore(subscribe, () => preview)
}

function slugNow(): string {
  if (preview.slug) return preview.slug
  const parts = window.location.pathname.split('/')
  return parts[1] === 'w' ? (parts[2] ?? '') : ''
}

export type GrowRender =
  | { status: 'error'; at: number; message: string; code?: number }
  | { status: 'ok'; at: number; referral?: Schemas['GrowBlockRender']; recs?: Schemas['BlockRender'] }

// Renders by block, reader and settings; undefined while the first one loads.
const cache = new Map<string, GrowRender>()
const inflight = new Set<string>()
// A render older than this is fetched again the next time its block draws.
const FRESH_MS = 60_000

function keyOf(kind: GrowKind, raw: Raw): string {
  const who = kind === 'recommendations' ? '' : (preview.contactId ?? '')
  return `${slugNow()}|${kind}|${who}|${kind === 'recommendations' ? '' : preview.milestoneId}|${JSON.stringify(raw)}`
}

async function fetchRender(kind: GrowKind, raw: Raw, key: string) {
  if (inflight.has(key)) return
  inflight.add(key)
  const workspace = slugNow()
  try {
    if (kind === 'recommendations') {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/blocks/recommendations', { params: { path: { workspace } }, body: { settings: raw } })
      cache.set(key, { status: 'ok', at: Date.now(), recs: data })
    } else {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/grow/blocks/render', {
        params: { path: { workspace } },
        body: { kind, settings: raw, contactId: preview.contactId || undefined, milestoneId: preview.milestoneId || undefined },
      })
      cache.set(key, { status: 'ok', at: Date.now(), referral: data })
    }
  } catch (e) {
    cache.set(key, { status: 'error', at: Date.now(), message: errorMessage(e), code: e instanceof ApiError ? e.status : undefined })
  }
  inflight.delete(key)
  notify()
}

let undecided = 0

/** The block's current render, fetching it when it's missing or old. */
function renderOf(kind: GrowKind, raw: Raw): GrowRender | undefined {
  if (kind !== 'recommendations' && preview.contactId === null) {
    // The composer picks the reader as soon as it knows the program's
    // readers; without one in a few seconds, the sample.
    if (!undecided)
      undecided = window.setTimeout(() => {
        undecided = 0
        if (preview.contactId === null) setGrowPreview({ contactId: '' })
      }, 3000)
    return undefined
  }
  if (kind === 'reward' && !preview.milestoneId) return { status: 'error', at: 0, message: tt(mb.rewardOnly) }
  const key = keyOf(kind, raw)
  const r = cache.get(key)
  if (!r || Date.now() - r.at > FRESH_MS) void fetchRender(kind, raw, key)
  return r
}

/** The render of a component's block, for its inspector; it follows the preview. */
export function useGrowRender(c: Component | null): GrowRender | undefined {
  const kind = c ? growKindOf(c) : null
  const raw = c ? growRaw(c) : {}
  return useSyncExternalStore(subscribe, () => (kind ? cache.get(keyOf(kind, raw)) : undefined))
}

/** Forgets every render (after the program or the recommendations change), and draws again. */
export function refreshGrowBlocks() {
  cache.clear()
  notify()
}

// --- Canvas HTML -----------------------------------------------------------------

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
const sans = "font-family:'IBM Plex Sans',system-ui,sans-serif"
const btn = `display:inline-flex;align-items:center;min-height:28px;padding:0 10px;border:1px solid #D4D8DF;border-radius:7px;background:#fff;color:#0B1220;${sans};font-size:12px;font-weight:600;text-decoration:none;cursor:pointer;white-space:nowrap`

/** A quiet box in the block's place: loading, or a note. */
function quiet(text: string): string {
  return `<div style="margin:16px 24px;padding:18px;border:1px dashed #C9CED6;border-radius:8px;background:#F9FAFB;${sans};font-size:12.5px;line-height:1.5;color:#5B6472;text-align:center">${esc(text)}</div>`
}

const eyeOff =
  '<svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#5B6472" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" style="vertical-align:-2px;margin-right:6px"><path d="M2 2l12 12M6.6 6.6a2 2 0 0 0 2.8 2.8M4.2 4.3C2.9 5.2 2 6.5 1.5 8c1.2 3 3.8 5 6.5 5 1.2 0 2.4-.4 3.4-1M7 3.1C7.3 3 7.7 3 8 3c2.7 0 5.3 2 6.5 5-.3.8-.7 1.5-1.2 2.1"/></svg>'

/** The block hidden for the "Preview as" reader, with the reason. */
function hiddenFor(title: string, text: string): string {
  return (
    `<div style="margin:16px 24px;padding:16px 18px;border:1px dashed #C9CED6;border-radius:8px;background:#fff;${sans};text-align:center">` +
    `<div style="font-size:13px;font-weight:600;color:#0B1220;margin-bottom:4px">${eyeOff}${esc(title)}</div>` +
    `<div style="font-size:12.5px;line-height:1.5;color:#374151">${esc(text)}</div></div>`
  )
}

/** A block that will be hidden: dimmed, in a dashed frame, with the reason and its actions. */
function willHide(reason: string, inner: string, actions: string): string {
  return (
    `<div style="position:relative;margin:12px 16px;padding:12px;border:2px dashed #F59E0B;border-radius:6px;background:#FFFBEB">` +
    `<div style="display:inline-flex;align-items:center;gap:6px;margin:0 0 8px;padding:4px 10px;border:1px solid #FCD34D;border-radius:6px;background:#FEF3C7;color:#92400E;${sans};font-size:12px;font-weight:600">` +
    `<span aria-hidden="true">!</span>${esc(reason)}</div>` +
    (inner ? `<div style="opacity:.5">${inner}</div>` : '') +
    (actions ? `<div style="display:flex;flex-wrap:wrap;align-items:center;gap:8px;padding:8px 8px 2px;${sans};font-size:12px;color:#374151">${actions}</div>` : '') +
    `</div>`
  )
}

function action(kind: 'remove' | 'open', label: string, href = ''): string {
  return kind === 'remove'
    ? `<button type="button" data-rc-grow-action="remove" style="${btn}">${esc(label)}</button>`
    : `<a href="${esc(href)}" data-rc-grow-action="open" style="${btn}">${esc(label)}</a>`
}

function referralHtml(r: Schemas['GrowBlockRender']): string {
  const slug = slugNow()
  switch (r.hidden) {
    case 'not_on_list':
      return hiddenFor(tt(mb.hiddenForTitle, { name: r.for }), tt(mb.hiddenForText, { list: r.listName }))
    case 'no_program':
      return willHide(tt(mb.hiddenNoProgram), '', action('remove', tt(mb.removeBlock)) + action('open', tt(mb.setUpReferrals), `/w/${slug}/grow/referrals`))
    case 'program_off': {
      const when = r.offAt ? (r.offBy ? tt(mb.offLine, { date: dayMonthDate(r.offAt), time: clock(r.offAt), name: r.offBy }) : tt(mb.offLineNoBy, { date: dayMonthDate(r.offAt), time: clock(r.offAt) })) : ''
      return willHide(
        tt(mb.hiddenOff),
        r.html,
        action('remove', tt(mb.removeBlock)) + action('open', tt(mb.referralSettings), `/w/${slug}/grow/referrals/settings?program=${encodeURIComponent(r.programId)}`) + (when ? `<span>${esc(when)}</span>` : ''),
      )
    }
  }
  return r.html || quiet(tt(mb.nothingToDraw))
}

function recsHtml(r: Schemas['BlockRender']): string {
  if (r.shows > 0) return r.html
  const rows = r.hidden
    .filter((h) => h.state !== 'not_in_emails')
    .map(
      (h) =>
        `<div style="display:flex;align-items:center;gap:8px;padding:4px 0;${sans};font-size:12.5px;color:#374151">` +
        `<span style="display:inline-flex;align-items:center;justify-content:center;width:22px;height:22px;border-radius:5px;background:#EFF6FF;color:#1D4ED8;font-size:10px;font-weight:700">${esc(h.initials)}</span>` +
        `<span><b style="font-weight:600;color:#0B1220">${esc(h.name)}</b> · ${esc(hiddenReason(h.state, h.pausedBy))}</span></div>`,
    )
    .join('')
  return willHide(tt(mb.noRecommendations), rows ? `<div style="padding:4px 8px">${rows}</div>` : '', action('open', tt(mb.manageRecommendations), `/w/${slugNow()}/grow/recommendations`))
}

/** Why a recommendation can't show now ("daily limit reached, until 00:00 UTC"). */
export function hiddenReason(state: string, pausedBy?: string): string {
  switch (state) {
    case 'paused':
      return pausedBy ? tt(mb.reasonPausedBy, { name: pausedBy }) : tt(mb.reasonPaused)
    case 'partner_paused':
      return tt(mb.reasonPartnerPaused)
    case 'invalid':
      return tt(mb.reasonInvalid)
    case 'limit':
      return tt(mb.reasonLimit)
    case 'ended':
      return tt(mb.reasonEnded)
    case 'unreachable':
      return tt(mb.reasonUnreachable)
    case 'not_in_emails':
      return tt(mb.reasonConfirmOnly)
  }
  return state
}

/** The canvas HTML of a block: its render for the preview, or why it's missing. */
function canvasHtml(kind: GrowKind, raw: Raw, last: string): string {
  const r = renderOf(kind, raw)
  if (!r) return last || quiet(tt(mb.loading))
  if (r.status === 'error') return quiet(kind === 'reward' && !preview.milestoneId ? r.message : tt(mb.renderFailed))
  if (r.referral) return referralHtml(r.referral)
  if (r.recs) return recsHtml(r.recs)
  return quiet(tt(mb.renderFailed))
}

// --- GrapesJS types ----------------------------------------------------------------

type View = {
  model: Component
  el: HTMLElement
  render(): unknown
  listenTo(obj: unknown, ev: string, fn: () => void): void
  rcLast?: string
}

function onCanvasClick(view: View, e: MouseEvent) {
  const target = (e.target as Element | null)?.closest?.('[data-rc-grow-action]')
  if (!target) return
  e.preventDefault()
  e.stopPropagation()
  if (target.getAttribute('data-rc-grow-action') === 'remove') {
    if (view.model.get('removable') !== false) view.model.remove()
    return
  }
  const href = target.getAttribute('href')
  // Grow opens beside the composer, so the draft stays open.
  if (href) window.open(href, '_blank', 'noopener')
}

export function loadGrowBlocks(editor: Editor, { coreMjmlModel, coreMjmlView }: ComponentPluginOptions) {
  for (const kind of KINDS) {
    const type = GROW_TYPES[kind]
    editor.Components.addType(type, {
      isComponent: (el: HTMLElement) => {
        if ((el.tagName || '').toLowerCase() !== 'mj-raw') return false
        const comment = Array.from(el.childNodes).find((n) => n.nodeType === Node.COMMENT_NODE) as Comment | undefined
        const m = comment ? MARKER.exec(comment.data) : null
        if (!m || m[1] !== kind) return false
        let raw = '{}'
        if (m[2]) {
          try {
            raw = JSON.stringify(JSON.parse(m[2].replace(/-\\u002d/g, '--')))
          } catch {
            /* the defaults */
          }
        }
        return { type, tagName: 'mj-raw', components: [], attributes: { [ATTR]: raw } }
      },
      model: {
        ...coreMjmlModel,
        defaults: {
          name: kind === 'referral' ? 'Referral block' : kind === 'recommendations' ? 'Recommendations block' : 'Reward block',
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
        // Attributes aren't mirrored into styles (the block has none).
        handleAttributeChange() {},
        handleStyleChange() {},
        // The email keeps the marker; each reader gets their own block.
        toHTML(this: Component) {
          return growMarker(kind, growRaw(this))
        },
      },
      view: {
        ...coreMjmlView,
        tagName: 'section',
        attributes: { style: 'pointer-events: all; display: block;', 'data-rc-grow-view': kind },
        init(this: View) {
          coreMjmlView.init.call(this)
          const rerender = () => this.render()
          listeners.add(rerender)
          this.listenTo(this.model, 'removed', () => listeners.delete(rerender))
          this.el.addEventListener('click', (e) => onCanvasClick(this, e))
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
        getInnerMjmlTemplate(this: View) {
          const html = canvasHtml(kind, growRaw(this.model), this.rcLast ?? '')
          this.rcLast = html
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
}

/** The Grow blocks in an editor, for the toolbar's "Preview as" and the checks. */
export function growBlocksIn(editor: Editor | null | undefined): Component[] {
  const wrapper = editor?.getWrapper()
  if (!wrapper) return []
  return KINDS.flatMap((k) => wrapper.findType(GROW_TYPES[k]))
}
