import type { Component, Editor } from 'grapesjs'
import { useEffect } from 'react'
import { t } from '@/i18n'
import { tagOf } from '@/features/composer/editor/labels'
import { mc } from './messages'
import { PREVIEW_MARKER } from './postBlocks'

/** The blocks the site counts (internal/website/render): each is one block of a post. */
const COUNTED = new Set(['mj-text', 'mj-image', 'mj-button', 'mj-divider', 'mj-table', 'mj-social', 'mj-navbar', 'mj-carousel', 'mj-accordion'])

/** Whether a component is the "Free preview ends here" block. */
export function isMarker(c: Component | null | undefined): boolean {
  return Boolean(c && tagOf(c) === 'mj-raw' && (c.getAttributes() as Record<string, string>)['rc-preview'] === 'end')
}

/** The marker on the canvas and how many blocks come before it, as the site counts them. */
export function findMarker(editor: Editor | null): { marker: Component; before: number } | null {
  const body = editor?.getWrapper()
  if (!body) return null
  let count = 0
  let found: { marker: Component; before: number } | null = null
  const walk = (c: Component) => {
    if (found) return
    if (isMarker(c)) {
      found = { marker: c, before: count }
      return
    }
    if (COUNTED.has(tagOf(c))) count++
    c.components().forEach((x: Component) => walk(x))
  }
  walk(body)
  return found
}

/** Inserts the marker after a content block (the selection, else the third block). */
export function placeMarker(editor: Editor | null, after: Component | null): boolean {
  const body = editor?.getWrapper()
  if (!editor || !body) return false
  let target = after && COUNTED.has(tagOf(after)) ? after : null
  if (!target) {
    const blocks: Component[] = []
    const walk = (c: Component) => {
      if (COUNTED.has(tagOf(c))) blocks.push(c)
      c.components().forEach((x: Component) => walk(x))
    }
    walk(body)
    target = blocks[Math.min(2, blocks.length - 1)] ?? null
  }
  const parent = target?.parent()
  if (!target || !parent) return false
  parent.append(PREVIEW_MARKER, { at: parent.components().indexOf(target) + 1 })
  return true
}

/**
 * Draws the marker on the canvas as a dashed line with its label ("Free
 * preview ends here · 3 blocks"), recounted on every change.
 */
export function useMarkerOnCanvas(editor: Editor | null, rev: number) {
  useEffect(() => {
    const doc = editor?.Canvas.getDocument()
    if (!doc?.head) return
    let el = doc.getElementById('rc-marker-style') as HTMLStyleElement | null
    if (!el) {
      el = doc.createElement('style')
      el.id = 'rc-marker-style'
      doc.head.appendChild(el)
    }
    el.textContent = `[rc-preview="end"]{display:block!important;position:relative;height:30px;margin:6px 0}
[rc-preview="end"]::before{content:"";position:absolute;left:0;right:0;top:50%;border-top:2px dashed #6D28D9}
[rc-preview="end"]::after{content:attr(data-rc-label);position:absolute;left:50%;top:50%;transform:translate(-50%,-50%);background:#EDE9FE;color:#5B21B6;border:1px solid #DDD6FE;border-radius:999px;padding:3px 12px;font:600 12px/1.4 'IBM Plex Sans',system-ui,sans-serif;white-space:nowrap}`
  }, [editor])
  useEffect(() => {
    const found = findMarker(editor)
    const el = found?.marker.getEl()
    if (el) el.setAttribute('data-rc-label', `☆ ${t(mc.markerLabel, { count: found!.before })}`)
  }, [editor, rev])
}

/**
 * Post mode's canvas in the site's theme: the site draws a post's text in
 * its own type (internal/website/render ignores the email's fonts and
 * spacing), so the canvas does too.
 */
export function useSiteTheme(editor: Editor | null) {
  useEffect(() => {
    const doc = editor?.Canvas.getDocument()
    if (!doc?.head) return
    let el = doc.getElementById('rc-site-theme') as HTMLStyleElement | null
    if (!el) {
      el = doc.createElement('style')
      el.id = 'rc-site-theme'
      doc.head.appendChild(el)
    }
    el.textContent = `[data-gjs-type="mj-text"] > td{padding:0 40px!important}
[data-gjs-type="mj-text"] > td > div{font-family:'IBM Plex Sans',system-ui,sans-serif!important;font-size:19px!important;line-height:1.7!important;color:#1F2937!important}
[data-gjs-type="mj-text"] p{margin:0 0 18px!important}
[data-gjs-type="mj-text"] h2,[data-gjs-type="mj-text"] h1{font-size:27px!important;line-height:1.25!important;font-weight:700!important;color:#0B1220!important;letter-spacing:-.01em;margin:28px 0 12px!important}
[data-gjs-type="mj-text"] h3{font-size:21px!important;line-height:1.3!important;font-weight:700!important;color:#0B1220!important;margin:22px 0 10px!important}
[data-gjs-type="mj-text"] ul,[data-gjs-type="mj-text"] ol{padding-left:24px;margin:0 0 18px}
[data-gjs-type="mj-text"] code{font-family:'IBM Plex Mono',ui-monospace,monospace!important;font-size:.88em;background:#F5F3FF;color:#4338CA;padding:2px 6px;border-radius:5px}
[data-gjs-type="mj-text"] a{color:#4338CA}
[rc-preview="end"]{margin:6px 40px!important}`
  }, [editor])
}
