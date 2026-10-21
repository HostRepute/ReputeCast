import type { Component } from 'grapesjs'
import type { Schemas } from '@/api/client'
import { ATTR_NAME, ATTR_VISIBLE } from '@/features/conditional/mjml'
import { tagOf } from '@/features/composer/editor/labels'

/**
 * The Sponsor block (ComposerEarnBlocks, brief §3.12): an mj-section that
 * fills itself from the issue's booking (internal/sponsors block.go). Its
 * own settings are attributes the compile step drops:
 *
 *   rc-sponsor         the block (the slot's id, or "1")
 *   rc-sponsor-layout  logo-left, image-top or text
 *   rc-sponsor-label   plain ("Sponsored") or name ("Sponsored by …")
 *   rc-sponsor-hash    what filled it, to tell when the booking changed
 *   rc-omit            set while there is no booking: the canvas shows why,
 *                      the email leaves the block out
 *
 * The content (the creative, the waiting placeholder or the note) is the
 * server's MJML; the block's name and Visibility are the writer's.
 */
export type SponsorBlock = Schemas['SponsorBlock']
export type SponsorLayout = SponsorBlock['layout']
export type SponsorLabelMode = SponsorBlock['labelMode']

export const ATTR_SPONSOR = 'rc-sponsor'
export const ATTR_LAYOUT = 'rc-sponsor-layout'
export const ATTR_LABEL = 'rc-sponsor-label'
export const ATTR_HASH = 'rc-sponsor-hash'
export const SPONSOR_MARKER = 'sponsor'

export interface SponsorSettingsValue {
  layout: SponsorLayout
  label: SponsorLabelMode
  hash: string
}

function attrsOf(c: Component): Record<string, unknown> {
  return (c.getAttributes() ?? {}) as Record<string, unknown>
}

/** The Sponsor block a component is, null when it isn't one. */
export function sponsorOf(c: Component | null | undefined): SponsorSettingsValue | null {
  if (!c || tagOf(c) !== 'mj-section') return null
  const a = attrsOf(c)
  if (a[ATTR_SPONSOR] === undefined || a[ATTR_SPONSOR] === '') return null
  const layout = String(a[ATTR_LAYOUT] ?? '')
  return {
    layout: layout === 'image-top' || layout === 'text' ? layout : 'logo-left',
    label: a[ATTR_LABEL] === 'plain' ? 'plain' : 'name',
    hash: String(a[ATTR_HASH] ?? ''),
  }
}

/** Every Sponsor block on the canvas (an email has one; a copy may add more). */
export function sponsorBlocks(body: Component | undefined): Component[] {
  const out: Component[] = []
  const walk = (x: Component) => {
    if (sponsorOf(x)) out.push(x)
    else x.components().forEach(walk)
  }
  if (body) walk(body)
  return out
}

const plex = `'IBM Plex Sans',Helvetica,Arial,sans-serif`

/**
 * The block as the Blocks panel inserts it: the card the server draws
 * (internal/sponsors block.go) with a note, until its settings fill it from
 * the booking.
 */
export function sponsorSection(note: string): string {
  const text = note.replace(/&/g, '&amp;').replace(/</g, '&lt;')
  return (
    `<!-- rc:${SPONSOR_MARKER} -->\n` +
    `<mj-section ${ATTR_SPONSOR}="1" ${ATTR_LAYOUT}="logo-left" ${ATTR_LABEL}="name" ${ATTR_NAME}="Sponsor" background-color="#ffffff" padding="12px 24px">` +
    `<mj-column><mj-text padding="0px">` +
    `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="border:1px dashed #D4D8DF;border-radius:8px;background-color:#F9FAFB;border-collapse:separate;"><tr><td valign="top" style="padding:16px 18px;">` +
    `<div style="font-family:${plex};font-size:12px;font-weight:700;letter-spacing:0.06em;text-transform:uppercase;color:#5B6472;padding-bottom:6px;">Sponsored</div>` +
    `<p style="font-family:${plex};font-size:13.5px;line-height:1.5;color:#374151;margin:0;">${text}</p>` +
    `</td></tr></table></mj-text></mj-column></mj-section>`
  )
}

function decode(v: string): string {
  return v.replace(/&(quot|amp|lt|gt|#39|#34|apos);/g, (m, e: string) => ({ quot: '"', amp: '&', lt: '<', gt: '>', '#39': "'", '#34': '"', apos: "'" })[e] ?? m)
}

/** Splits the server's block into its section's attributes and its content. */
export function splitBlock(mjml: string): { attrs: Record<string, string>; inner: string } | null {
  const m = /^\s*<mj-section\b([^>]*)>([\s\S]*)<\/mj-section>\s*$/.exec(mjml)
  if (!m) return null
  const attrs: Record<string, string> = {}
  for (const a of m[1]!.matchAll(/([a-z][a-z0-9-]*)="([^"]*)"/g)) attrs[a[1]!] = decode(a[2]!)
  return { attrs, inner: m[2]! }
}

/** The writer's own attributes, which a refill keeps. */
const KEEP = [ATTR_NAME, ATTR_VISIBLE, 'css-class']

/**
 * Fills a Sponsor block with the server's content: the section's attributes
 * (with what filled it) and its columns. The block's name, Visibility and
 * CSS classes stay. Returns false when the content didn't parse.
 */
export function applySponsor(comp: Component, blk: SponsorBlock): boolean {
  const parts = splitBlock(blk.mjml)
  if (!parts) return false
  const own = attrsOf(comp)
  const next: Record<string, unknown> = { ...parts.attrs, [ATTR_HASH]: blk.hash }
  for (const k of KEEP) if (own[k] !== undefined && own[k] !== '') next[k] = own[k]
  if (!next[ATTR_NAME]) next[ATTR_NAME] = 'Sponsor'
  comp.setAttributes(next)
  comp.components(parts.inner)
  return true
}

/** Whether a block holds what the booking has now. */
export function isCurrent(comp: Component, blk: SponsorBlock): boolean {
  return sponsorOf(comp)?.hash === blk.hash
}
