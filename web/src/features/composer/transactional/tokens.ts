import type { Editor } from 'grapesjs'
import { useEffect, useRef } from 'react'
import { liquidOutputs, localsOf, tokenView, type TokenContext, type TokenView } from './variables'

/**
 * Variables on the canvas (TransactionalTemplate board): each `{{ … }}`
 * shows as a chip with what it prints for the chosen test data. Sensitive
 * values are amber; undeclared variables and missing required ones red.
 *
 * Only the canvas DOM changes: the chips are put back to their Liquid
 * before a text is edited (GrapesJS reads the DOM when editing ends), and
 * texts being edited are left alone. The MJML, the model and saves never
 * see them.
 */
const CLS = 'rc-tok'
const SRC = 'data-rc-src'

const STYLE = `.${CLS}{padding:0 5px;border-radius:5px;background:#EDE9FE;color:#5B21B6;font-family:'IBM Plex Mono',ui-monospace,SFMono-Regular,monospace;font-size:.86em;font-weight:500;line-height:1.55;white-space:nowrap;-webkit-box-decoration-break:clone;box-decoration-break:clone}
.${CLS}.s{background:#FEF3C7;color:#92400E}
.${CLS}.e{background:#FEE2E2;color:#991B1B;text-decoration:underline wavy #DC2626;text-underline-offset:3px}
.${CLS}.m{background:#EEF0F3;color:#4B5563}`

/** Puts the Liquid of every chip under `root` back. */
export function undecorate(root: Element) {
  const spans = root.querySelectorAll(`span.${CLS}`)
  if (!spans.length) return
  spans.forEach((s) => s.replaceWith(s.ownerDocument.createTextNode(s.getAttribute(SRC) ?? s.textContent ?? '')))
  root.normalize()
}

/** Turns the outputs in the text under `root` into chips (texts being edited are skipped). */
export function decorate(root: Element, ctx: TokenContext, title: (v: TokenView, src: string) => string) {
  const doc = root.ownerDocument
  const walker = doc.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(n) {
      const p = n.parentElement
      if (!p || !n.nodeValue?.includes('{{')) return NodeFilter.FILTER_REJECT
      if (p.closest(`span.${CLS}, [contenteditable="true"], script, style, title, textarea`)) return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    },
  })
  const nodes: Text[] = []
  while (walker.nextNode()) nodes.push(walker.currentNode as Text)
  for (const node of nodes) {
    const src = node.nodeValue ?? ''
    const outs = liquidOutputs(src)
    if (!outs.length) continue
    const frag = doc.createDocumentFragment()
    let at = 0
    for (const o of outs) {
      if (o.start > at) frag.append(src.slice(at, o.start))
      const raw = src.slice(o.start, o.end)
      const v = tokenView(o.expr, ctx)
      const span = doc.createElement('span')
      span.className = v.tone ? `${CLS} ${v.tone}` : CLS
      span.setAttribute(SRC, raw)
      span.title = title(v, raw)
      span.textContent = v.text
      frag.append(span)
      at = o.end
    }
    if (at < src.length) frag.append(src.slice(at))
    node.replaceWith(frag)
  }
}

function ensureStyle(doc: Document) {
  if (doc.getElementById('rc-tok-style')) return
  const el = doc.createElement('style')
  el.id = 'rc-tok-style'
  el.textContent = STYLE
  doc.head?.appendChild(el)
}

/**
 * Keeps the canvas chips in step with the editor, the variables and the
 * preview data. Loop items and assigns are read from the canvas itself.
 */
export function useCanvasTokens(editor: Editor | null, base: Omit<TokenContext, 'locals'> | null, title: (v: TokenView, src: string) => string) {
  const titleRef = useRef(title)
  useEffect(() => {
    titleRef.current = title
  })
  useEffect(() => {
    if (!editor || !base) return
    let observer: MutationObserver | null = null
    let raf = 0
    const opts: MutationObserverInit = { childList: true, subtree: true, characterData: true }
    const body = () => editor.Canvas.getBody() as HTMLElement | undefined
    // Our own changes happen with the observer off, so they don't loop.
    const quietly = (fn: (b: HTMLElement) => void) => {
      const b = body()
      if (!b) return
      observer?.disconnect()
      fn(b)
      observer?.observe(b, opts)
    }
    const run = () => {
      raf = 0
      quietly((b) => {
        ensureStyle(b.ownerDocument)
        // Liquid tags stay as text on the canvas, so the loops and assigns can be read from it.
        decorate(b, { ...base, locals: localsOf(b.textContent ?? '') }, (v, src) => titleRef.current(v, src))
      })
    }
    const schedule = () => {
      if (!raf) raf = requestAnimationFrame(run)
    }
    const attach = () => {
      observer?.disconnect()
      const b = body()
      if (!b) return
      observer = new MutationObserver(schedule)
      observer.observe(b, opts)
      schedule()
    }
    // A text about to be edited gets its Liquid back first: the double click
    // reaches the document (capture) before GrapesJS starts editing.
    const onDblClick = (e: Event) => {
      const el = (e.target as Element | null)?.closest?.('[data-gjs-type]')
      if (el) quietly(() => undecorate(el))
    }
    const onRteEnable = (view: { el?: HTMLElement }) => {
      if (view?.el) quietly(() => undecorate(view.el!))
    }
    let doc: Document | undefined
    const bindDoc = () => {
      doc?.removeEventListener('dblclick', onDblClick, true)
      doc = editor.Canvas.getDocument() ?? undefined
      doc?.addEventListener('dblclick', onDblClick, true)
    }
    const onFrame = () => {
      bindDoc()
      attach()
    }
    bindDoc()
    attach()
    editor.on('rte:enable', onRteEnable)
    editor.on('rte:disable', schedule)
    editor.on('canvas:frame:load:body', onFrame)
    return () => {
      cancelAnimationFrame(raf)
      observer?.disconnect()
      doc?.removeEventListener('dblclick', onDblClick, true)
      // The editor may already be destroyed (the composer closing).
      try {
        editor.off('rte:enable', onRteEnable)
        editor.off('rte:disable', schedule)
        editor.off('canvas:frame:load:body', onFrame)
        const b = body()
        if (b) undecorate(b)
      } catch {
        /* nothing left to clean */
      }
    }
  }, [editor, base])
}
