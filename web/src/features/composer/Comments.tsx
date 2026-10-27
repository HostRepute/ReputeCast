import type { Component, Editor } from 'grapesjs'
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { toast } from '@/components/ui/Toast'
import { t as tt, useT } from '@/i18n'
import { clock, shortDate, zonedParts } from '@/lib/format'
import { useMe } from '@/lib/session'
import { useComposer } from './context'
import { sectionName, tagOf, textContent } from './editor/labels'
import { mx } from './collab.messages'

/**
 * Comments on a campaign (CO-10, Composer board's Comments tab): threads on
 * a block, which carries rc-comment="<id>" (left out of the compiled email),
 * or on the whole email; replies with @mentions that notify the people
 * named; resolving. Reviewers with a review link comment too, marked as
 * reviewers.
 */

export const COMMENT_ATTR = 'rc-comment'

type Thread = Schemas['ComposerThread']
type Member = Schemas['ComposerMember']

const commentsPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/comments' as const

/** The campaign's threads, refreshed every 20 seconds while the composer is open. */
export function useComments() {
  const c = useComposer()
  return $api.useQuery(
    'get',
    commentsPath,
    { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } },
    { enabled: c.kind !== 'transactional', refetchInterval: 20_000, refetchOnWindowFocus: true },
  )
}

function useMembers(enabled: boolean) {
  const c = useComposer()
  return $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/mentionable',
    { params: { path: { workspace: c.slug, campaignId: c.campaign.id } } },
    { enabled: enabled && c.kind !== 'transactional', staleTime: 5 * 60_000 },
  )
}

/** "09:36" today, "Yesterday 16:12", else "Tue 20 Oct 09:36". */
export function commentTime(at: string, now = new Date()): string {
  const key = (d: string | Date) => {
    const p = zonedParts(d)
    return `${p.year}-${p.month}-${p.day}`
  }
  if (key(at) === key(now)) return clock(at)
  if (key(at) === key(new Date(now.getTime() - 86_400_000))) return tt(mx.yesterdayAt, { time: clock(at) })
  return `${shortDate(at)} ${clock(at)}`
}

function clip(s: string, n = 60): string {
  const one = s.replace(/\s+/g, ' ').trim()
  return one.length > n ? `${one.slice(0, n - 1).trimEnd()}…` : one
}

/** The word for what a block is ("button", "heading"), for labels and markers. */
function whatOf(comp: Component): string {
  const tag = tagOf(comp)
  if (tag === 'mj-text' && comp.getEl()?.querySelector('h1, h2, h3, h4')) return tt(mx.what, { tag: 'heading' })
  return tt(mx.what, { tag: tag.replace(/-/g, '_') })
}

/** What a thread is on: "the button “Read the docs”", "the heading “Retiring the v1 API”". */
export function anchorLabelOf(comp: Component): string {
  const tag = tagOf(comp)
  const quoted = (text: string) => (clip(text) ? tt(mx.anchorQuoted, { what: whatOf(comp), text: clip(text) }) : tt(mx.anchorPlain, { what: whatOf(comp) }))
  switch (tag) {
    case 'mj-button':
      return quoted(textContent(comp))
    case 'mj-image':
      return quoted(String((comp.getAttributes() as Record<string, unknown>).alt ?? ''))
    case 'mj-text': {
      const heading = comp.getEl()?.querySelector('h1, h2, h3, h4')
      return quoted(heading?.textContent ?? textContent(comp))
    }
    case 'mj-section':
    case 'mj-wrapper':
    case 'mj-hero':
      return comp.parent() && tagOf(comp.parent()) === 'mj-body' ? quoted(sectionName(comp)) : quoted('')
    default:
      return quoted('')
  }
}

/** A short random id for rc-comment (the server takes [a-z0-9]{0,40}). */
function newAnchor(): string {
  const bytes = new Uint8Array(8)
  crypto.getRandomValues(bytes)
  return Array.from(bytes, (b) => (b % 36).toString(36)).join('')
}

/** The block a thread is on, if it is still in the email. */
export function componentOfAnchor(editor: Editor | null, anchor: string): Component | undefined {
  if (!editor || !anchor) return undefined
  return editor.getWrapper()?.find(`[${COMMENT_ATTR}="${anchor}"]`)[0]
}

/** Mentions still written in a body ("@Marcus Chen"), from the people picked. */
function mentionsIn(body: string, picked: Map<string, string>): string[] {
  return [...picked].filter(([, name]) => body.includes(`@${name}`)).map(([id]) => id)
}

/**
 * A comment box with @mention suggestions: "@" then letters lists the
 * workspace's people; ↑ ↓ choose, Enter or Tab picks, Esc closes.
 */
function MentionBox({
  label,
  placeholder,
  value,
  onChange,
  picked,
  members,
  autoFocus,
  hiddenLabel,
}: {
  label: string
  placeholder: string
  value: string
  onChange: (v: string) => void
  picked: Map<string, string>
  members: Member[]
  autoFocus?: boolean
  hiddenLabel?: boolean
}) {
  const t = useT()
  const id = useId()
  const ref = useRef<HTMLTextAreaElement>(null)
  const [query, setQuery] = useState<string | null>(null)
  const [active, setActive] = useState(0)
  const meId = useMe().data?.user.id
  const matches = useMemo(() => {
    if (query === null) return []
    const q = query.toLowerCase()
    return members.filter((m) => m.id !== meId && (!q || m.name.toLowerCase().split(/\s+/).some((w) => w.startsWith(q)) || m.name.toLowerCase().startsWith(q))).slice(0, 6)
  }, [query, members, meId])
  const open = query !== null
  useEffect(() => {
    if (autoFocus) ref.current?.focus()
  }, [autoFocus])
  const track = (text: string, caret: number) => {
    const m = /(?:^|\s)@([\p{L}\p{M}'’.-]{0,30}(?: [\p{L}\p{M}'’.-]{0,30})?)$/u.exec(text.slice(0, caret))
    setQuery(m ? m[1]! : null)
    setActive(0)
  }
  const pick = (m: Member) => {
    const el = ref.current
    const caret = el?.selectionStart ?? value.length
    const before = value.slice(0, caret).replace(/@([^@]*)$/, `@${m.name} `)
    const next = before + value.slice(caret)
    picked.set(m.id, m.name)
    onChange(next)
    setQuery(null)
    window.requestAnimationFrame(() => {
      el?.focus()
      el?.setSelectionRange(before.length, before.length)
    })
  }
  const onKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      setQuery(null)
    } else if (matches.length && e.key === 'ArrowDown') {
      e.preventDefault()
      setActive((a) => (a + 1) % matches.length)
    } else if (matches.length && e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((a) => (a - 1 + matches.length) % matches.length)
    } else if (matches.length && (e.key === 'Enter' || e.key === 'Tab')) {
      e.preventDefault()
      pick(matches[active]!)
    }
  }
  const listId = `${id}-people`
  return (
    <div className="mn">
      <label className={hiddenLabel ? 'sr-only' : 'lbl'} htmlFor={`${id}-box`}>
        {label}
      </label>
      <textarea
        id={`${id}-box`}
        ref={ref}
        className="in"
        rows={2}
        placeholder={placeholder}
        value={value}
        maxLength={4000}
        aria-activedescendant={open && matches.length ? `${listId}-${active}` : undefined}
        aria-controls={open ? listId : undefined}
        onChange={(e) => {
          onChange(e.target.value)
          track(e.target.value, e.target.selectionStart)
        }}
        onKeyDown={onKeyDown}
        onBlur={() => window.setTimeout(() => setQuery(null), 150)}
      />
      {open ? (
        matches.length ? (
          <ul className="mnl" id={listId} role="listbox" aria-label={t(mx.mentionsList)}>
            {matches.map((m, i) => (
              <li key={m.id} id={`${listId}-${i}`} role="option" aria-selected={i === active}>
                <button type="button" tabIndex={-1} onMouseDown={(e) => e.preventDefault()} onClick={() => pick(m)}>
                  <Avatar name={m.name} color={m.color} size={22} />
                  {m.name}
                  <span className="k" style={{ marginLeft: 'auto' }}>
                    {m.role}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        ) : query ? (
          <p className="k m-0 mt-1" id={listId} role="status">
            {t(mx.noMatch, { q: query })}
          </p>
        ) : null
      ) : null}
    </div>
  )
}

function useInvalidate() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: ['get', commentsPath] })
}

function CommentHead({ author, color, at, reviewer }: { author: string; color?: string; at: string; reviewer: boolean }) {
  const t = useT()
  return (
    <div className="ch">
      <Avatar name={author} color={reviewer ? 'neutral' : color} size={28} />
      <b>{author}</b>
      <span className="k">{commentTime(at)}</span>
      {reviewer ? <Badge tone="neutral">{t(mx.reviewer)}</Badge> : null}
    </div>
  )
}

function ThreadCard({ thread, members, focused, canResolve }: { thread: Thread; members: Member[]; focused: boolean; canResolve: boolean }) {
  const c = useComposer()
  const t = useT()
  const invalidate = useInvalidate()
  const [replying, setReplying] = useState(focused && !thread.resolvedAt)
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const [picked] = useState(() => new Map<string, string>())
  const ref = useRef<HTMLElement>(null)
  const first = thread.comments[0]!
  const target = componentOfAnchor(c.editor, thread.anchor)
  useEffect(() => {
    if (focused) ref.current?.scrollIntoView({ block: 'nearest' })
  }, [focused])
  const reply = async () => {
    setBusy(true)
    try {
      await http.POST(commentsPath, {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: { body: text, threadId: thread.id, mentions: mentionsIn(text, picked) },
      })
      setText('')
      picked.clear()
      setReplying(false)
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(mx.commentFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const resolve = async (resolved: boolean) => {
    setBusy(true)
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/campaigns/{campaignId}/comments/{threadId}/resolved', {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id, threadId: thread.id } },
        body: { resolved },
      })
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(mx.resolveFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const anchor = thread.anchorLabel ? t(mx.onAnchor, { label: thread.anchorLabel }) : thread.anchor ? '' : t(mx.onEmail)
  return (
    <article
      ref={ref}
      id={`thread-${thread.id}`}
      className={`cm${focused ? ' on' : ''}${thread.resolvedAt ? ' done' : ''}`}
      aria-label={thread.anchorLabel ? t(mx.threadOn, { label: thread.anchorLabel }) : t(mx.threadOnEmail)}
      tabIndex={-1}
    >
      <CommentHead author={first.author} color={first.authorColor} at={first.createdAt} reviewer={first.reviewer} />
      {anchor ? (
        target ? (
          <button type="button" className="anc" title={t(mx.selectBlock)} onClick={() => c.editor?.select(target)}>
            {anchor}
          </button>
        ) : (
          // Only once the canvas is there can a block be missing from it.
          <span className="anc">{c.editor && thread.anchor && thread.anchorLabel ? t(mx.anchorGone, { label: anchor }) : anchor}</span>
        )
      ) : null}
      <p className="tx">{first.body}</p>
      {thread.comments.slice(1).map((r) => (
        <div key={r.id} className="cr">
          <CommentHead author={r.author} color={r.authorColor} at={r.createdAt} reviewer={r.reviewer} />
          <p className="tx">{r.body}</p>
        </div>
      ))}
      {thread.resolvedAt ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="k" style={{ flex: '1 1 140px' }}>
            {t(mx.resolvedBy, { name: thread.resolvedBy || '—' })}
          </span>
          {canResolve ? (
            <button type="button" className="btn sm" disabled={busy} onClick={() => void resolve(false)}>
              {t(mx.reopen)}
            </button>
          ) : null}
        </div>
      ) : replying ? (
        <>
          <MentionBox
            label={t(mx.replyTo, { name: first.author })}
            hiddenLabel
            placeholder={t(mx.replyPlaceholder)}
            value={text}
            onChange={setText}
            picked={picked}
            members={members}
            autoFocus={!focused}
          />
          <div className="flex flex-wrap items-center gap-2">
            <span className="k" style={{ flex: '1 1 140px' }}>
              {t(mx.mentionsNote)}
            </span>
            {canResolve ? (
              <button type="button" className="btn sm" disabled={busy} onClick={() => void resolve(true)}>
                {t(mx.resolve)}
              </button>
            ) : null}
            <button type="button" className="btn sm pri" disabled={busy || !text.trim()} aria-busy={busy || undefined} onClick={() => void reply()}>
              {busy ? t(mx.posting) : t(mx.reply)}
            </button>
          </div>
        </>
      ) : (
        <div className="flex gap-2">
          <button type="button" className="btn sm" onClick={() => setReplying(true)}>
            {t(mx.reply)}
          </button>
          {canResolve ? (
            <button type="button" className="btn sm" disabled={busy} onClick={() => void resolve(true)}>
              {t(mx.resolve)}
            </button>
          ) : null}
        </div>
      )}
    </article>
  )
}

/** A new thread: on the selected block, or the whole email. */
function NewComment({ members }: { members: Member[] }) {
  const c = useComposer()
  const t = useT()
  const invalidate = useInvalidate()
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const [picked] = useState(() => new Map<string, string>())
  const sel = c.selected && tagOf(c.selected) && tagOf(c.selected) !== 'mj-body' ? c.selected : null
  const label = sel ? anchorLabelOf(sel) : ''
  const post = async () => {
    setBusy(true)
    try {
      // The block keeps the thread's id; people who can't edit comment on
      // it by name only.
      let anchor = ''
      if (sel && c.canEdit) {
        const attrs = sel.getAttributes() as Record<string, string>
        anchor = attrs[COMMENT_ATTR] || newAnchor()
        if (!attrs[COMMENT_ATTR]) sel.addAttributes({ [COMMENT_ATTR]: anchor })
      }
      await http.POST(commentsPath, {
        params: { path: { workspace: c.slug, campaignId: c.campaign.id } },
        body: { body: text, anchor, anchorLabel: label, mentions: mentionsIn(text, picked) },
      })
      setText('')
      picked.clear()
      await invalidate()
      if (anchor) void c.save({ background: true })
    } catch (e) {
      toast({ kind: 'error', title: t(mx.commentFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="cm new">
      <span className="anc">{sel ? t(mx.onAnchor, { label }) : t(mx.onEmail)}</span>
      <MentionBox label={t(mx.newLabel)} hiddenLabel placeholder={t(mx.newPlaceholder)} value={text} onChange={setText} picked={picked} members={members} />
      <div className="flex flex-wrap items-center gap-2">
        <span className="k" style={{ flex: '1 1 140px' }}>
          {t(mx.mentionsNote)}
        </span>
        <button type="button" className="btn sm pri" disabled={busy || !text.trim()} aria-busy={busy || undefined} onClick={() => void post()}>
          {busy ? t(mx.posting) : t(mx.comment)}
        </button>
      </div>
    </div>
  )
}

/** The inspector's Comments tab. `thread` is a deep link's (a mention's email). */
export function CommentsTab({ thread }: { thread?: string }) {
  const c = useComposer()
  const t = useT()
  const q = useComments()
  const members = useMembers(true).data?.items ?? []
  const [showResolved, setShowResolved] = useState(false)
  const canResolve = c.campaign.access.canEdit
  const list = q.data
  const open = list?.threads.filter((x) => !x.resolvedAt) ?? []
  const resolved = list?.threads.filter((x) => x.resolvedAt) ?? []
  const focusedResolved = Boolean(thread && resolved.some((x) => x.id === thread))
  let body: ReactNode
  if (q.isPending) body = <div className="k" style={{ padding: '14px 12px' }} aria-busy="true" />
  else if (q.isError)
    body = (
      <div className="flex items-center gap-2" style={{ padding: '14px 12px' }} role="alert">
        <span className="k">{t(mx.loadFailed)}</span>
        <button type="button" className="btn sm" onClick={() => void q.refetch()}>
          {t(mx.retry)}
        </button>
      </div>
    )
  else
    body = (
      <>
        {open.map((x) => (
          <ThreadCard key={x.id} thread={x} members={members} focused={x.id === thread} canResolve={canResolve} />
        ))}
        {!open.length && !resolved.length ? (
          <p className="k m-0" style={{ padding: '10px 12px' }}>
            {t(mx.noComments)}
          </p>
        ) : null}
        {resolved.length ? (
          <div className="flex items-center gap-2" style={{ padding: '10px 12px' }}>
            <span className="k">{t(mx.resolvedCount, { count: resolved.length })}</span>
            <button type="button" className="btn sm" aria-expanded={showResolved || focusedResolved} onClick={() => setShowResolved((s) => !s)}>
              {showResolved || focusedResolved ? t(mx.hideResolved) : t(mx.showResolved)}
            </button>
          </div>
        ) : null}
        {showResolved || focusedResolved ? resolved.map((x) => <ThreadCard key={x.id} thread={x} members={members} focused={x.id === thread} canResolve={canResolve} />) : null}
        {canResolve ? null : (
          <p className="k m-0" style={{ padding: '0 12px 10px' }}>
            {t(mx.whoResolves)}
          </p>
        )}
      </>
    )
  return (
    <div className="flex flex-col" aria-label={t(mx.commentsRegion)}>
      <NewComment members={members} />
      {body}
    </div>
  )
}

/** The tab's label: "Comments · 2" (open threads). */
export function useCommentsTabLabel(): string {
  const t = useT()
  const q = useComments()
  return t(mx.tabComments, { count: q.data?.open ?? 0 })
}

type Box = { top: number; left: number; width: number; height: number }

/**
 * The avatar of the person who started a thread, on the block it is on
 * (Composer board: "Comment by Marcus Chen on this button. Open comments").
 */
export function CommentOverlays({ host, tick, boxOf }: { host: HTMLElement | null; tick: number; boxOf: (editor: Editor, el: Element | null | undefined, host: HTMLElement | null) => Box | null }) {
  const c = useComposer()
  const t = useT()
  const q = useComments()
  const ed = c.editor
  const placed = useMemo(() => {
    if (!ed || c.kind === 'transactional') return []
    const out: { thread: Thread; box: Box; what: string }[] = []
    for (const thread of q.data?.threads ?? []) {
      if (thread.resolvedAt || !thread.anchor) continue
      const comp = componentOfAnchor(ed, thread.anchor)
      const el = comp?.getEl()
      const inner = tagOf(comp) === 'mj-button' ? el?.querySelector('a, p') : tagOf(comp) === 'mj-image' ? el?.querySelector('img') : null
      const box = comp ? boxOf(ed, inner ?? el, host) : null
      if (comp && box) out.push({ thread, box, what: whatOf(comp) })
    }
    return out
  }, [ed, host, tick, q.data, c.kind, boxOf]) // eslint-disable-line react-hooks/exhaustive-deps
  return (
    <>
      {placed.map(({ thread, box, what }) => {
        const first = thread.comments[0]!
        return (
          <button
            key={thread.id}
            type="button"
            className="cpin"
            style={{ top: box.top + Math.max(0, box.height - 30) / 2, left: box.left + box.width + 8 }}
            aria-label={t(mx.marker, { name: first.author, what })}
            onClick={() => {
              c.setRightTab('comments')
              window.requestAnimationFrame(() => document.getElementById(`thread-${thread.id}`)?.focus())
            }}
          >
            <Avatar name={first.author} color={first.reviewer ? 'neutral' : first.authorColor} size={30} />
          </button>
        )
      })}
    </>
  )
}
