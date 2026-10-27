import type { Component, Editor } from 'grapesjs'
import { Link } from '@tanstack/react-router'
import { useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type ReactNode } from 'react'
import { toast } from '@/components/ui/Toast'
import { T, useT, t as tt } from '@/i18n'
import { andList } from '@/lib/format'
import { useComposer } from '../context'
import { ShareReviewButton } from '../Collab'
import { COMMENT_ATTR, NewCommentBox, ThreadCard, useComments, useMentionable } from '../Comments'
import { blockDefs } from '../editor/blocks'
import { encodeTables } from '../editor/components'
import { buildToken, chipHtml, chipLabelOf, parseToken } from '../editor/liquid'
import { blockContent, mjChildren, sectionOf, tagOf, textContent } from '../editor/labels'
import { applyTextStyle, currentTextStyle, TEXT_STYLES, type TextEditing, type TextStyle } from '../editor/textStyle'
import { SYNCED_ATTR } from '../editor/synced'
import { bodyOf, docItems, fromEditable, isBlank, textHtml, toEditable, type DocItem } from './doc'
import { applyBold, applyShortcut, caretToEnd, isDividerLine, paragraphOf } from './markdown'
import { cleanPaste, textToHtml } from './paste'
import { mw } from './write.messages'
import { mx } from '../collab.messages'
import './write.css'

/**
 * Write mode (CO-9, ComposerWrite board): the email as a document over the
 * same blocks as Design mode. Texts are typed in place (markdown shortcuts,
 * headings and lists, paste cleanup with Undo), "/" on an empty line
 * inserts a block, images take their alt text inline, and synced and
 * layout blocks say where they change. Every edit goes to the canvas's
 * components, so Design and Code show it at once.
 */

const SLASH_OPTIONS = ['heading', 'bullets', 'numbers', 'image', 'button', 'divider', 'article', 'poll', 'person', 'cond'] as const
type SlashOption = (typeof SLASH_OPTIONS)[number]

const OPTION_TEXT: Record<SlashOption, { label: keyof typeof mw; hint: keyof typeof mw; keys: string; kbd?: string; group: 1 | 2 }> = {
  heading: { label: 'optHeading', hint: 'optHeadingHint', keys: 'heading title h2', kbd: '#', group: 1 },
  bullets: { label: 'optBullets', hint: 'optBulletsHint', keys: 'bulleted list bullets ul', kbd: '-', group: 1 },
  numbers: { label: 'optNumbers', hint: 'optNumbersHint', keys: 'numbered list steps ol', kbd: '1.', group: 1 },
  image: { label: 'optImage', hint: 'optImageHint', keys: 'image picture photo', group: 1 },
  button: { label: 'optButton', hint: 'optButtonHint', keys: 'button cta link', group: 1 },
  divider: { label: 'optDivider', hint: 'optDividerHint', keys: 'divider line rule', kbd: '---', group: 1 },
  article: { label: 'optArticle', hint: 'optArticleHint', keys: 'article card post', group: 2 },
  poll: { label: 'optPoll', hint: 'optPollHint', keys: 'poll vote survey', group: 2 },
  person: { label: 'optPerson', hint: 'optPersonHint', keys: 'personalization first name merge tag token', group: 2 },
  cond: { label: 'optCond', hint: 'optCondHint', keys: 'conditional block if rule segment', group: 2 },
}

/** Options matching what is typed after "/": a word of their keys, or their name. */
export function slashMatches(q: string): SlashOption[] {
  const s = q.toLowerCase().trim()
  if (!s) return [...SLASH_OPTIONS]
  return SLASH_OPTIONS.filter((o) => {
    const it = OPTION_TEXT[o]
    return it.keys.split(' ').some((k) => k.startsWith(s)) || tt(mw[it.label]).toLowerCase().includes(s)
  })
}

const attrs = (c: Component) => c.getAttributes() as Record<string, string>

/** The attributes a new text takes from the texts around it (font, size, colour), without ReputeCast's markers. */
function textAttrs(editor: Editor, near: Component | null): string {
  const texts = (editor.getWrapper()?.findType('mj-text') ?? []) as Component[]
  const ref = (near && tagOf(near) === 'mj-text' ? near : null) ?? texts.find((x) => !sectionOf(x) || !attrs(sectionOf(x)!)[SYNCED_ATTR])
  if (!ref) return ' font-size="15px" line-height="1.55" color="#1F2937"'
  return Object.entries(attrs(ref))
    .filter(([k]) => !k.startsWith('rc-') && k !== 'id' && k !== 'css-class' && k !== 'mj-class')
    .map(([k, v]) => ` ${k}="${String(v).replace(/"/g, '&quot;')}"`)
    .join('')
}

/** Where content goes when nothing is chosen: the last column that isn't a synced block's. */
function lastColumn(editor: Editor): Component | undefined {
  const body = bodyOf(editor)
  if (!body) return undefined
  const sections = mjChildren(body).filter((s) => !attrs(s)[SYNCED_ATTR] && tagOf(s) === 'mj-section')
  const last = sections[sections.length - 1]
  const cols = last ? mjChildren(last).filter((x) => tagOf(x) === 'mj-column') : []
  return cols[cols.length - 1]
}

/** Puts content in after `after` (or at the end of the last column); returns what went in. */
function insertContent(editor: Editor, after: Component | null, mjml: string): Component | undefined {
  const parent = after?.parent() ?? lastColumn(editor)
  if (!parent) return insertSection(editor, after, `<mj-section background-color="#ffffff" padding="16px 24px"><mj-column>${mjml}</mj-column></mj-section>`)
  const at = after && after.parent() === parent ? parent.components().indexOf(after) + 1 : parent.components().length
  const added = parent.append(blockContent(encodeTables(mjml)), { at }) as Component | Component[]
  return Array.isArray(added) ? added.find((x) => tagOf(x)) : added
}

/** Puts a section in after the one `after` is in (before the synced footer at the end). */
function insertSection(editor: Editor, after: Component | null, mjml: string): Component | undefined {
  const body = bodyOf(editor)
  if (!body) return undefined
  const kids = mjChildren(body)
  const sec = sectionOf(after)
  let at: number
  if (sec) at = body.components().indexOf(sec) + 1
  else {
    const footer = kids.find((s) => attrs(s)[SYNCED_ATTR] === 'footer')
    at = footer ? body.components().indexOf(footer) : body.components().length
  }
  const added = body.append(blockContent(encodeTables(mjml)), { at }) as Component | Component[]
  return Array.isArray(added) ? added.find((x) => tagOf(x)) : added
}

/** The editable element a text component is typed in. */
const editables = new WeakMap<Component, HTMLElement>()

function textEditing(el: HTMLElement): TextEditing {
  const doc = el.ownerDocument
  return {
    el,
    doc,
    selection: () => doc.getSelection(),
    exec: (cmd, value) => {
      if (typeof doc.execCommand === 'function') doc.execCommand(cmd, false, value)
    },
  }
}

interface WriteState {
  editor: Editor
  canEdit: boolean
  active: Component | null
  setActive(c: Component | null): void
  focusCid: string | null
  setFocusCid(cid: string | null): void
  commit(comp: Component): void
  openLine(after: Component | null): void
  inserted: { cid: string; label: string } | null
  markInserted(cid: string, label: string, focus: boolean): void
  commentFor: Component | null
  setCommentFor(c: Component | null): void
}

/** A text typed in place. */
function WriteText({ comp, w }: { comp: Component; w: WriteState }) {
  const t = useT()
  const ref = useRef<HTMLDivElement>(null)
  const written = useRef<string | null>(null)
  const timer = useRef(0)
  const html = textHtml(comp)
  // The model is written as it is typed; the element is only redrawn when the
  // model changed from elsewhere (undo, Design), so the caret stays put.
  useLayoutEffect(() => {
    const el = ref.current
    if (!el || html === written.current) return
    el.innerHTML = toEditable(html, w.editor)
    written.current = html
  }, [html, w.editor])
  useEffect(() => {
    const el = ref.current
    if (!el) return
    editables.set(comp, el)
    if (w.focusCid === comp.cid) {
      el.focus()
      const first = el.querySelector('h2, h3, li, p') ?? el
      caretToEnd(first)
      w.setFocusCid(null)
    }
  }, [comp, w])
  const commit = useCallback(() => {
    window.clearTimeout(timer.current)
    const el = ref.current
    if (!el) return
    const next = fromEditable(el)
    if (next === written.current) return
    comp.components(next)
    written.current = textHtml(comp)
  }, [comp])
  useEffect(() => () => commit(), [commit])
  const onInput = () => {
    const el = ref.current!
    const sel = el.ownerDocument.getSelection()
    const node = sel?.anchorNode ?? null
    // "**words**": bold.
    if (node && node.nodeType === 3 && /\*\*[^*\n]+\*\*$/.test((node as Text).data.slice(0, sel!.anchorOffset))) {
      const strong = applyBold(node as Text)
      if (strong?.nextSibling) {
        const r = el.ownerDocument.createRange()
        r.setStart(strong.nextSibling, Math.min(1, strong.nextSibling.textContent?.length ?? 0))
        r.collapse(true)
        sel!.removeAllRanges()
        sel!.addRange(r)
      }
    }
    // "# ", "## ", "- ", "1. " at the start of a paragraph.
    const block = paragraphOf(sel?.anchorNode ?? null, el)
    if (block && /^(#{1,2}|[-*]|1\.)[ \u00a0]/.test(block.textContent ?? '')) {
      const target = applyShortcut(block)
      if (target) caretToEnd(target)
    }
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(commit, 400)
  }
  const onKeyDown = (e: ReactKeyboardEvent<HTMLDivElement>) => {
    const el = ref.current!
    // Alt and an arrow key move the block (as the gutter's handle does).
    if (e.altKey && (e.key === 'ArrowUp' || e.key === 'ArrowDown')) {
      e.preventDefault()
      commit()
      if (moveBlock(comp, e.key === 'ArrowUp' ? -1 : 1)) w.setFocusCid(comp.cid)
      return
    }
    if (e.key === 'Enter' && !e.shiftKey) {
      // "---" alone on a line: a divider after this text.
      const block = paragraphOf(el.ownerDocument.getSelection()?.anchorNode ?? null, el)
      if (block && isDividerLine(block)) {
        e.preventDefault()
        block.remove()
        commit()
        const div = blockDefs('').find((d) => d.id === 'mj-divider')!.content
        insertContent(w.editor, comp, div)
        w.openLine(comp)
      }
    }
  }
  const onPaste = (e: React.ClipboardEvent<HTMLDivElement>) => {
    const data = e.clipboardData
    const raw = data.getData('text/html')
    const text = data.getData('text/plain')
    if (!raw && !text) return
    e.preventDefault()
    commit()
    const before = textHtml(comp)
    const cleaned = raw ? cleanPaste(raw) : { html: textToHtml(text), source: '', kept: [], removed: 0, images: 0 }
    const el = ref.current!
    const doc = el.ownerDocument
    if (typeof doc.execCommand === 'function' && doc.queryCommandSupported?.('insertHTML')) doc.execCommand('insertHTML', false, cleaned.html)
    else {
      const sel = doc.getSelection()
      const range = sel && sel.rangeCount && el.contains(sel.anchorNode) ? sel.getRangeAt(0) : null
      const frag = doc.createRange().createContextualFragment(cleaned.html)
      if (range) {
        range.deleteContents()
        range.insertNode(frag)
      } else el.appendChild(frag)
    }
    commit()
    const detail = [
      cleaned.kept.length ? tt(mw.pasteKept, { kept: andList(cleaned.kept.map((k) => tt(k === 'headings' ? mw.keptHeadings : k === 'links' ? mw.keptLinks : mw.keptLists))) }) : '',
      cleaned.removed ? tt(mw.pasteRemoved, { count: cleaned.removed }) : '',
      cleaned.images ? tt(mw.pasteImages, { count: cleaned.images }) : '',
    ]
      .filter(Boolean)
      .join(' · ')
    toast({
      kind: 'success',
      title: cleaned.source ? tt(mw.pastedFrom, { source: cleaned.source }) : tt(mw.pasted),
      detail: detail || undefined,
      action: {
        label: tt(mw.undoPaste),
        onClick: () => {
          comp.components(before)
          toast({ kind: 'info', title: tt(mw.pasteUndone), detail: tt(mw.pasteUndoneDetail) })
        },
      },
    })
  }
  const onBlur = () => {
    commit()
    // An emptied new block goes again.
    if (w.inserted?.cid === comp.cid && isBlank(textHtml(comp))) comp.remove()
  }
  const thread = useThreadFor(comp)
  return (
    <div
      ref={ref}
      className={`wtext${thread ? ' wcm' : ''}`}
      contentEditable={w.canEdit}
      suppressContentEditableWarning
      role="textbox"
      tabIndex={0}
      aria-multiline="true"
      aria-readonly={!w.canEdit || undefined}
      aria-label={t(mw.textBlock)}
      spellCheck
      onInput={onInput}
      onKeyDown={onKeyDown}
      onPaste={onPaste}
      onFocus={() => {
        w.setActive(comp)
        try {
          ref.current?.ownerDocument.execCommand?.('defaultParagraphSeparator', false, 'p')
        } catch {
          // Older engines: their default separator stays.
        }
      }}
      onBlur={onBlur}
    />
  )
}

/** The open thread on a block, if any. */
function useThreadFor(comp: Component) {
  const q = useComments()
  const anchor = attrs(comp)[COMMENT_ATTR]
  return anchor ? q.data?.threads.find((x) => x.anchor === anchor && !x.resolvedAt) : undefined
}

function WriteImage({ comp, hero, w }: { comp: Component; hero: boolean; w: WriteState }) {
  const t = useT()
  const id = useId()
  const a = attrs(comp)
  const alt = a.alt ?? ''
  const src = a.src ?? ''
  const name = decodeURIComponent(src.split('/').pop()?.split('?')[0] ?? '') || t(mw.imageWord)
  const decorative = a['rc-decorative'] !== undefined
  const [draft, setDraft] = useState(alt)
  const [state, setState] = useState<'none' | 'added' | 'decorative'>(alt ? 'added' : decorative ? 'decorative' : 'none')
  const width = parseInt(a.width ?? '', 10)
  const word = hero ? t(mw.imageHero) : name
  const label = alt ? t(mw.imageAlt, { name: hero ? t(mw.imageHero) : t(mw.imageWord), alt }) : width ? t(mw.imageSize, { name: word, width }) : word
  const needsAlt = !alt && !decorative
  // An image that came without alt text keeps its alt box (ComposerWrite).
  const [asked] = useState(needsAlt)
  const save = (v: string) => {
    const next = v.trim()
    if (next === alt) return
    comp.addAttributes({ alt: next })
    setState(next ? 'added' : 'none')
  }
  return (
    <>
      <div className={`wimg${state === 'none' ? ' nalt' : ''}`} style={{ height: hero ? 180 : 150, position: 'relative' }}>
        {label}
        {asked ? <span className={`wbadge${state === 'none' ? '' : ' ok'}`}>{state === 'none' ? t(mw.noAlt) : state === 'decorative' ? t(mw.decorativeDone) : t(mw.altAdded)}</span> : null}
      </div>
      {asked ? (
        <div className="walt">
          <label className="sr-only" htmlFor={`${id}-alt`}>
            {t(mw.altFor, { name })}
          </label>
          <input
            id={`${id}-alt`}
            className="in"
            placeholder={t(mw.altPlaceholder)}
            value={draft}
            disabled={!w.canEdit}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={() => save(draft)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') save(draft)
            }}
          />
          <button
            type="button"
            className="btn sm"
            disabled={!w.canEdit}
            onClick={() => {
              comp.addAttributes({ alt: '' })
              setDraft('')
              setState('decorative')
            }}
          >
            {t(mw.decorative)}
          </button>
        </div>
      ) : (
        <span className="wcap">{t(mw.altCaption, { alt })}</span>
      )}
    </>
  )
}

function WriteButton({ comp, w }: { comp: Component; w: WriteState }) {
  const t = useT()
  const ref = useRef<HTMLSpanElement>(null)
  const text = textContent(comp)
  const href = attrs(comp).href ?? ''
  useLayoutEffect(() => {
    if (ref.current && ref.current.textContent !== text) ref.current.textContent = text
  }, [text])
  const commit = () => {
    const v = (ref.current?.textContent ?? '').trim()
    if (!v || v === text) {
      if (ref.current) ref.current.textContent = text
      return
    }
    comp.components(v.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;'))
  }
  const thread = useThreadFor(comp)
  return (
    <div className="wbtnb">
      <span
        ref={ref}
        className={`ebtn${thread ? ' wcm' : ''}`}
        contentEditable={w.canEdit}
        suppressContentEditableWarning
        role="textbox"
        tabIndex={0}
        aria-readonly={!w.canEdit || undefined}
        aria-label={t(mw.buttonText)}
        onFocus={() => w.setActive(comp)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            ;(e.target as HTMLElement).blur()
          }
        }}
      />
      <span className="url">{href.replace(/^https?:\/\//, '')}</span>
      <span className="k">{t(mw.buttonTag)}</span>
    </div>
  )
}

/** Add a block below, and move (drag, or Alt and the arrow keys). */
function Gutter({ comp, w }: { comp: Component; w: WriteState }) {
  const t = useT()
  if (!w.canEdit) return null
  return (
    <div className="gut">
      <button type="button" aria-label={t(mw.addBelow)} onClick={() => w.openLine(comp)}>
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
          <path d="M8 3v10M3 8h10" />
        </svg>
      </button>
      <button
        type="button"
        aria-label={t(mw.move)}
        onKeyDown={(e) => {
          if (!e.altKey || (e.key !== 'ArrowUp' && e.key !== 'ArrowDown')) return
          e.preventDefault()
          moveBlock(comp, e.key === 'ArrowUp' ? -1 : 1)
        }}
      >
        <svg width="12" height="14" viewBox="0 0 12 16" fill="currentColor" aria-hidden="true">
          <circle cx="4" cy="4" r="1.2" />
          <circle cx="8" cy="4" r="1.2" />
          <circle cx="4" cy="8" r="1.2" />
          <circle cx="8" cy="8" r="1.2" />
          <circle cx="4" cy="12" r="1.2" />
          <circle cx="8" cy="12" r="1.2" />
        </svg>
      </button>
    </div>
  )
}

/** Moves a block one place up or down in its column. */
export function moveBlock(comp: Component, by: -1 | 1): boolean {
  const parent = comp.parent()
  if (!parent) return false
  const siblings = mjChildren(parent)
  const i = siblings.indexOf(comp)
  const other = siblings[i + by]
  if (!other) return false
  const all = parent.components()
  const at = all.indexOf(other) + (by > 0 ? 1 : 0)
  comp.move(parent, { at: by > 0 ? at : all.indexOf(other) })
  return true
}

/** The empty line: type "/" for the block menu (↑ ↓ choose, Enter insert, Esc close). */
function SlashLine({ after, w, onDone }: { after: Component | null; w: WriteState; onDone: () => void }) {
  const t = useT()
  const id = useId()
  const c = useComposer()
  const [value, setValue] = useState('')
  const [act, setAct] = useState(0)
  const [fields, setFields] = useState(false)
  const ref = useRef<HTMLInputElement>(null)
  const open = value.startsWith('/')
  const q = open ? value.slice(1) : ''
  const vis = slashMatches(q)
  const active = vis.length ? Math.min(act, vis.length - 1) : -1
  useEffect(() => {
    if (after) ref.current?.focus()
  }, [after])
  const insert = (o: SlashOption) => {
    const ed = w.editor
    const base = textAttrs(ed, after)
    const defs = blockDefs(c.context.postalAddress)
    let added: Component | undefined
    let focusText = false
    switch (o) {
      case 'heading':
        added = insertContent(ed, after, `<mj-text${base}><h2><br></h2></mj-text>`)
        focusText = true
        break
      case 'bullets':
        added = insertContent(ed, after, `<mj-text${base}><ul><li><br></li></ul></mj-text>`)
        focusText = true
        break
      case 'numbers':
        added = insertContent(ed, after, `<mj-text${base}><ol><li><br></li></ol></mj-text>`)
        focusText = true
        break
      case 'image':
        added = insertContent(ed, after, defs.find((d) => d.id === 'mj-image')!.content)
        if (added) {
          c.setAssetTarget({ component: added, attr: 'src' })
          c.openDialog('assets')
        }
        break
      case 'button':
        added = insertContent(ed, after, defs.find((d) => d.id === 'mj-button')!.content)
        break
      case 'divider':
        added = insertContent(ed, after, defs.find((d) => d.id === 'mj-divider')!.content)
        break
      case 'article':
        added = insertSection(ed, after, defs.find((d) => d.id === 'rc-article')!.content)
        break
      case 'poll':
        added = insertSection(ed, after, defs.find((d) => d.id === 'rc-poll')!.content)
        break
      case 'person':
        setFields(true)
        setValue('')
        return
      case 'cond': {
        added = insertSection(ed, after, defs.find((d) => d.id === 'mj-1-column')!.content)
        if (added) {
          // The rule is set in Design: Settings › Who sees this section.
          c.setView('visual')
          ed.select(added)
          c.setRightTab('settings')
        }
        break
      }
    }
    setValue('')
    if (!added) return
    const label = tt(mw[OPTION_TEXT[o].label])
    w.markInserted(added.cid, label, focusText)
    toast({ kind: 'success', title: tt(mw.inserted, { block: label }), detail: tt(mw.insertedWhere) })
    onDone()
  }
  const insertField = (token: string) => {
    const label = chipLabelOf(w.editor, token)
    const target = w.active && tagOf(w.active) === 'mj-text' ? w.active : null
    setFields(false)
    if (target) {
      const el = editables.get(target)
      if (el) {
        el.focus()
        caretToEnd(el.querySelector('p:last-child, h2:last-child, li:last-child') ?? el)
        const doc = el.ownerDocument
        if (typeof doc.execCommand === 'function' && doc.queryCommandSupported?.('insertHTML')) doc.execCommand('insertHTML', false, chipHtml(token, label))
        else (el.querySelector('p:last-child, li:last-child') ?? el).insertAdjacentHTML('beforeend', chipHtml(token, label))
        target.components(fromEditable(el))
        return
      }
    }
    const added = insertContent(w.editor, after, `<mj-text${textAttrs(w.editor, after)}><p>${token}</p></mj-text>`)
    if (added) w.markInserted(added.cid, tt(mw.optPerson), true)
  }
  const onKeyDown = (e: ReactKeyboardEvent<HTMLInputElement>) => {
    if (open) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setAct(vis.length ? (active + 1) % vis.length : 0)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setAct(vis.length ? (active + vis.length - 1) % vis.length : 0)
      } else if (e.key === 'Enter') {
        e.preventDefault()
        if (active >= 0) insert(vis[active]!)
      } else if (e.key === 'Escape') {
        e.preventDefault()
        e.stopPropagation()
        // Esc keeps what was typed as text.
        setValue(value.slice(1))
      }
      return
    }
    if (e.key === 'Enter' && value.trim()) {
      // Words typed here become a paragraph.
      e.preventDefault()
      const esc = value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      const added = insertContent(w.editor, after, `<mj-text${textAttrs(w.editor, after)}><p>${esc}</p></mj-text>`)
      setValue('')
      if (added) w.markInserted(added.cid, tt(mw.textBlock), true)
    } else if (e.key === 'Escape' && fields) {
      setFields(false)
    }
  }
  const listId = `${id}-list`
  const groups = [1, 2] as const
  return (
    <div className="wlinebox">
      <label className="sr-only" htmlFor={`${id}-line`}>
        {t(mw.emptyLine)}
      </label>
      <input
        id={`${id}-line`}
        ref={ref}
        className="wline"
        value={value}
        placeholder={t(mw.slashPlaceholder)}
        autoComplete="off"
        role="combobox"
        aria-autocomplete="list"
        aria-expanded={open}
        aria-controls={open ? listId : undefined}
        aria-activedescendant={open && active >= 0 ? `${id}-${vis[active]}` : undefined}
        disabled={!w.canEdit}
        onChange={(e) => {
          setValue(e.target.value)
          setAct(0)
        }}
        onKeyDown={onKeyDown}
      />
      {open ? (
        <div className="slash">
          <div id={listId} role="listbox" aria-label={t(mw.insertBlock)}>
            {groups.map((g) =>
              vis.some((o) => OPTION_TEXT[o].group === g) ? (
                <div key={g} role="group" aria-labelledby={`${id}-g${g}`}>
                  <div className="sgh" id={`${id}-g${g}`}>
                    {g === 1 ? t(mw.groupText) : t(mw.groupEmail)}
                  </div>
                  {vis
                    .filter((o) => OPTION_TEXT[o].group === g)
                    .map((o) => (
                      <div
                        key={o}
                        id={`${id}-${o}`}
                        role="option"
                        aria-selected={vis[active] === o}
                        tabIndex={-1}
                        onMouseDown={(e) => e.preventDefault()}
                        onClick={() => insert(o)}
                        onKeyDown={() => {}}
                      >
                        <span className="sl">{t(mw[OPTION_TEXT[o].label])}</span>
                        <span className="sh">
                          {t(mw[OPTION_TEXT[o].hint])}
                          {OPTION_TEXT[o].kbd ? <span className="kbd">{OPTION_TEXT[o].kbd}</span> : null}
                        </span>
                      </div>
                    ))}
                </div>
              ) : null,
            )}
            {!vis.length ? (
              <div className="none" role="option" aria-disabled="true" aria-selected="false">
                {t(mw.noMatch, { q })}
              </div>
            ) : null}
          </div>
          <div className="sf" aria-hidden="true">
            <span>
              <span className="kbd">↑</span>
              <span className="kbd">↓</span>
              {t(mw.keysChoose)}
            </span>
            <span>
              <span className="kbd">{t(mw.keyEnter)}</span>
              {t(mw.keysInsert)}
            </span>
            <span>
              <span className="kbd">{t(mw.keyEsc)}</span>
              {t(mw.keysClose)}
            </span>
          </div>
        </div>
      ) : null}
      {fields ? <FieldMenu onPick={insertField} onClose={() => setFields(false)} /> : null}
    </div>
  )
}

/** The workspace's personalization fields, to insert as a chip with its fallback. */
function FieldMenu({ onPick, onClose }: { onPick: (token: string) => void; onClose: () => void }) {
  const c = useComposer()
  const t = useT()
  const ref = useRef<HTMLDivElement>(null)
  useEffect(() => {
    const el = ref.current
    el?.querySelector('button')?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      e.stopPropagation()
      onClose()
    }
    el?.addEventListener('keydown', onKey)
    return () => el?.removeEventListener('keydown', onKey)
  }, [onClose])
  const tokens = c.context.tokens.filter((x) => x.kind !== 'system')
  return (
    <div ref={ref} className="slash" role="group" aria-label={t(mw.fields)}>
      <div className="sgh">{t(mw.fields)}</div>
      {tokens.map((x) => {
        const token = buildToken({ ...parseToken(`{{ ${x.key} }}`), fallback: x.fallback ?? undefined })
        return (
          <button key={x.key} type="button" className="wopt" onClick={() => onPick(token)}>
            <span className="sl">{x.label}</span>
            {x.fallback ? <span className="sh">“{x.fallback}”</span> : null}
          </button>
        )
      })}
    </div>
  )
}

function Toolbar({ w }: { w: WriteState }) {
  const c = useComposer()
  const t = useT()
  const um = w.editor.UndoManager
  void c.rev
  const [link, setLink] = useState<string | null>(null)
  const [fields, setFields] = useState(false)
  const el = w.active ? editables.get(w.active) : undefined
  const style: TextStyle = el ? currentTextStyle(textEditing(el)) : 'p'
  const run = (fn: (rte: TextEditing) => void) => {
    if (!w.active || !el) return
    el.focus()
    fn(textEditing(el))
    w.commit(w.active)
  }
  const q = useComments()
  const disabled = !w.canEdit
  return (
    <div className="wtools">
      <div className="wt-fmt" role="toolbar" aria-label={t(mw.formatting)}>
        <button type="button" className="btn ic" aria-label={um.hasUndo() ? t(mw.undo) : t(mw.undoNone)} aria-disabled={!um.hasUndo() || undefined} onClick={() => um.hasUndo() && um.undo()}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
            <path d="M6 4 2 8l4 4M2 8h8a4 4 0 0 1 0 8" />
          </svg>
        </button>
        <button type="button" className="btn ic" aria-label={um.hasRedo() ? t(mw.redo) : t(mw.redoNone)} aria-disabled={!um.hasRedo() || undefined} onClick={() => um.hasRedo() && um.redo()}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden="true">
            <path d="m10 4 4 4-4 4M14 8H6a4 4 0 0 0 0 8" />
          </svg>
        </button>
        <span className="sep" aria-hidden="true" />
        <select className="in" aria-label={t(mw.textStyle)} value={style} disabled={disabled || !el} onChange={(e) => run((rte) => applyTextStyle(rte, e.target.value as TextStyle))}>
          {TEXT_STYLES.map((s) => (
            <option key={s} value={s}>
              {s === 'p' ? t(mw.paragraph) : s === 'heading' ? t(mw.heading) : s === 'subheading' ? t(mw.subheading) : t(mw.small)}
            </option>
          ))}
        </select>
        <span className="sep" aria-hidden="true" />
        <button type="button" className="btn ic" aria-label={t(mw.bold)} disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => run((rte) => rte.exec('bold'))}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinejoin="round" aria-hidden="true">
            <path d="M4.5 2.5h4.2a2.8 2.8 0 0 1 0 5.6H4.5zM4.5 8.1h5a2.9 2.9 0 0 1 0 5.8h-5z" />
          </svg>
        </button>
        <button type="button" className="btn ic" aria-label={t(mw.italic)} disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => run((rte) => rte.exec('italic'))}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" aria-hidden="true">
            <path d="M7 2.5h6M3 13.5h6M10 2.5 6 13.5" />
          </svg>
        </button>
        <button type="button" className="btn ic" aria-label={t(mw.link)} aria-expanded={link !== null} disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => setLink(link === null ? '' : null)}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" aria-hidden="true">
            <path d="M6.5 9.5a3 3 0 0 0 4.2 0l2.3-2.3a3 3 0 0 0-4.2-4.2l-.9.9M9.5 6.5a3 3 0 0 0-4.2 0L3 8.8A3 3 0 0 0 7.2 13l.9-.9" />
          </svg>
        </button>
        <span className="sep" aria-hidden="true" />
        <button type="button" className="btn ic" aria-label={t(mw.bullets)} disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => run((rte) => rte.exec('insertUnorderedList'))}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" aria-hidden="true">
            <path d="M6 4h8M6 8h8M6 12h8" />
            <circle cx="2.5" cy="4" r=".6" />
            <circle cx="2.5" cy="8" r=".6" />
            <circle cx="2.5" cy="12" r=".6" />
          </svg>
        </button>
        <button type="button" className="btn ic" aria-label={t(mw.numbers)} disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => run((rte) => rte.exec('insertOrderedList'))}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" aria-hidden="true">
            <path d="M6.5 4h7.5M6.5 8h7.5M6.5 12h7.5M2 3l1-.5v3M2 7.5h2L2 10h2M2 12h1.6l-.9 1 .9 1H2" />
          </svg>
        </button>
        <span className="sep" aria-hidden="true" />
        <button type="button" className="btn tx" aria-label={t(mw.personalizeLabel)} aria-expanded={fields} disabled={disabled} onMouseDown={(e) => e.preventDefault()} onClick={() => setFields((f) => !f)}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
            <circle cx="8" cy="5.5" r="2.5" />
            <path d="M3 14a5 5 0 0 1 10 0" />
          </svg>
          {t(mw.personalize)}
        </button>
        <button type="button" className="btn tx" aria-label={t(mw.commentLabel)} disabled={!w.active} onClick={() => w.setCommentFor(w.active)}>
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" aria-hidden="true">
            <path d="M2.5 3.5h11v7.5H7l-3 2.5V11H2.5z" />
          </svg>
          {t(mw.comment)}
        </button>
      </div>
      <div className="wt-col">
        <span className="cmc">
          <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinejoin="round" aria-hidden="true">
            <path d="M2.5 3.5h11v7.5H7l-3 2.5V11H2.5z" />
          </svg>
          {t(mx.tabComments, { count: q.data?.open ?? 0 })}
        </span>
        <ShareReviewButton />
      </div>
      {link !== null ? (
        <form
          className="wlink"
          onSubmit={(e) => {
            e.preventDefault()
            const href = link.trim()
            run((rte) => rte.exec(href ? 'createLink' : 'unlink', href || undefined))
            setLink(null)
          }}
        >
          <label className="sr-only" htmlFor="wlink-href">
            {t(mw.linkAddress)}
          </label>
          <input id="wlink-href" className="in" type="url" placeholder={t(mw.linkPlaceholder)} value={link} onChange={(e) => setLink(e.target.value)} />
          <button type="submit" className="btn sm pri">
            {link.trim() ? t(mw.linkApply) : t(mw.linkRemove)}
          </button>
        </form>
      ) : null}
      {fields ? (
        <div className="wfields">
          <FieldMenu
            onPick={(token) => {
              setFields(false)
              run((rte) => {
                const label = chipLabelOf(w.editor, token)
                if (typeof rte.doc.queryCommandSupported === 'function' && rte.doc.queryCommandSupported('insertHTML')) rte.exec('insertHTML', chipHtml(token, label))
                else rte.el.insertAdjacentHTML('beforeend', chipHtml(token, label))
              })
            }}
            onClose={() => setFields(false)}
          />
        </div>
      ) : null}
    </div>
  )
}

/** The synced Header or Footer: changes in the brand kit only. */
function SyncedCard({ kind }: { kind: 'header' | 'footer' }) {
  const c = useComposer()
  const t = useT()
  return (
    <div className="wsync">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <path d="M13 6.5A5 5 0 0 0 3.6 4.8M3 9.5a5 5 0 0 0 9.4 1.7" />
        <path d="M13.5 2.5v4h-4M2.5 13.5v-4h4" />
      </svg>
      <span>
        <T msg={mw.synced} values={{ kind, b: (chunks: ReactNode[]) => <b>{chunks}</b> }} />
      </span>
      <Link to="/w/$workspace/settings/brand-kit" params={{ workspace: c.slug }}>
        {t(mw.editInBrandKit)}
      </Link>
    </div>
  )
}

/** A block Write mode doesn't type in (a poll, a feed, social links): its name, and Edit in Design. */
function OtherCard({ comp, label }: { comp: Component; label: string }) {
  const c = useComposer()
  const t = useT()
  return (
    <div className="wsync">
      <span>
        <T msg={mw.other} values={{ block: label, b: (chunks: ReactNode[]) => <b>{chunks}</b> }} />
      </span>
      <button
        type="button"
        className="btn sm"
        onClick={() => {
          c.setView('visual')
          c.editor?.select(comp)
        }}
      >
        {t(mw.editInDesign)}
      </button>
    </div>
  )
}

function ColumnTag({ section, n, count }: { section: string; n: number; count: number }) {
  const t = useT()
  return (
    <div className="wcoltag">
      <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" aria-hidden="true">
        <rect x="2" y="3" width="5" height="10" rx="1" />
        <rect x="9" y="3" width="5" height="10" rx="1" />
      </svg>
      {t(mw.column, { section, n, count })}
    </div>
  )
}

/** The open thread on a block, beside it (below it on narrower screens). */
function BlockThread({ comp, w }: { comp: Component; w: WriteState }) {
  const thread = useThreadFor(comp)
  const members = useMentionable()
  const c = useComposer()
  if (w.commentFor === comp)
    return (
      <div className="wth">
        <NewCommentBox target={comp} members={members} onDone={() => w.setCommentFor(null)} />
      </div>
    )
  if (!thread) return null
  return (
    <div className="wth">
      <ThreadCard thread={thread} members={members} focused={false} canResolve={c.campaign.access.canEdit} compact />
    </div>
  )
}

export function WriteView() {
  const c = useComposer()
  const t = useT()
  const editor = c.editor
  const [active, setActive] = useState<Component | null>(null)
  const [focusCid, setFocusCid] = useState<string | null>(null)
  const [lineAfter, setLineAfter] = useState<Component | null>(null)
  const [inserted, setInserted] = useState<{ cid: string; label: string } | null>(null)
  const [commentFor, setCommentFor] = useState<Component | null>(null)
  void c.rev
  // eslint-disable-next-line react-hooks/exhaustive-deps -- the canvas changes bump rev
  const items = useMemo(() => docItems(editor), [editor, c.rev])
  const w = useMemo<WriteState | null>(
    () =>
      editor
        ? {
            editor,
            canEdit: c.canEdit,
            active,
            setActive,
            focusCid,
            setFocusCid,
            commit: (comp) => {
              const el = editables.get(comp)
              if (el) comp.components(fromEditable(el))
            },
            openLine: (after) => setLineAfter(after),
            inserted,
            commentFor,
            setCommentFor,
            markInserted: (cid: string, label: string, focus: boolean) => {
              setInserted({ cid, label })
              setLineAfter(null)
              if (focus) setFocusCid(cid)
            },
          }
        : null,
    [editor, c.canEdit, active, focusCid, inserted, commentFor],
  )
  if (!w) return <div className="wmain" aria-busy="true" />
  const lineAt = lineAfter ? lineAfter.cid : null
  // The empty line at the end goes before the synced footer.
  const footerAt = items.findIndex((x) => x.kind === 'synced' && x.synced === 'footer')
  const endAt = footerAt >= 0 ? footerAt : items.length
  const line = <SlashLine key={lineAt ?? 'end'} after={lineAfter} w={w} onDone={() => setLineAfter(null)} />
  const render = (x: DocItem): ReactNode => {
    switch (x.kind) {
      case 'synced':
        return <SyncedCard kind={x.synced} />
      case 'column':
        return null
      case 'text':
        return <WriteText comp={x.comp} w={w} />
      case 'image':
        return <WriteImage comp={x.comp} hero={x.hero} w={w} />
      case 'button':
        return <WriteButton comp={x.comp} w={w} />
      case 'divider':
        return <hr className="wdiv" aria-label={t(mw.divider)} />
      case 'other':
        return <OtherCard comp={x.comp} label={x.label} />
    }
  }
  return (
    <main id="main" className="wmain" tabIndex={-1}>
      <div className="wcol">
        <Toolbar w={w} />
        <p className="whint">
          <T msg={mw.hint} values={{ k: (chunks: ReactNode[]) => <span className="kbd">{chunks}</span> }} />
        </p>
        <div className="wdocw">
          <article className="wdoc" aria-label={t(mw.docLabel)}>
            {!items.some((x) => x.kind === 'text' || x.kind === 'image' || x.kind === 'button') ? <p className="k">{t(mw.empty)}</p> : null}
            {items.map((x, i) => (
              <div key={x.key} className={x.kind === 'column' ? 'contents' : 'wb'}>
                {i === endAt && !lineAt ? line : null}
                {x.kind === 'column' ? (
                  <ColumnTag section={x.section} n={x.n} count={x.count} />
                ) : (
                  <>
                    {x.kind !== 'synced' ? <Gutter comp={x.comp} w={w} /> : null}
                    {render(x)}
                    {inserted && x.kind !== 'synced' && x.comp.cid === inserted.cid && x.kind !== 'text' ? (
                      <div className="wsync new">
                        <span>
                          <T msg={mw.newBlock} values={{ block: inserted.label, b: (chunks: ReactNode[]) => <b>{chunks}</b> }} />
                        </span>
                      </div>
                    ) : null}
                    {x.kind !== 'synced' ? <BlockThread comp={x.comp} w={w} /> : null}
                  </>
                )}
                {x.kind !== 'column' && x.kind !== 'synced' && lineAt === x.comp.cid ? line : null}
              </div>
            ))}
            {endAt === items.length && !lineAt ? <div className="wb">{line}</div> : null}
          </article>
        </div>
      </div>
    </main>
  )
}
