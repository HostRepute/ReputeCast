import { Link } from '@tanstack/react-router'
import { memo, useId, useState, type ReactNode } from 'react'
import { ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { HOSTREPUTE } from '@/lib/hostrepute'
import { useWorkspace } from '@/lib/session'
import { splitModelName } from '@/features/mcpservers/api'
import { AsstIcon } from './icons'
import { MarkdownText, StreamingMarkdown } from './Markdown'
import { mx } from './mcp.messages'
import { m } from './messages'
import { turnActive, type ThreadMessage, type ThreadState, type ToolRow } from './reducer'
import { StateCard } from './states'

/** One block of the conversation: the person's message, an answer (a turn's tools and text), or the summary marker. */
export type Item =
  | { kind: 'user'; key: string; text: string; pending?: boolean }
  | { kind: 'answer'; key: string; turnId?: string; tools: ToolRow[]; texts: ThreadMessage[]; live?: string; stopped?: string; status?: 'queued' | 'answering' | 'waiting' }
  | { kind: 'summary'; key: string }

/** The conversation's blocks, in order, from a thread's state (pure). */
export function buildItems(s: ThreadState): Item[] {
  const items: Item[] = []
  const answerOf = (turnId: string | undefined, key: string) => {
    const last = items[items.length - 1]
    if (last && last.kind === 'answer' && last.turnId === turnId) return last
    const a: Item = { kind: 'answer', key, turnId, tools: [], texts: [] }
    items.push(a)
    return a
  }
  for (const msg of s.messages) {
    if (msg.role === 'user') items.push({ kind: 'user', key: msg.id, text: msg.text })
    else if (msg.role === 'summary') items.push({ kind: 'summary', key: msg.id })
    else {
      const a = answerOf(msg.turnId, msg.id)
      if (msg.role === 'tool') a.tools.push(...(msg.tools ?? []))
      else a.texts.push(msg)
    }
  }
  if (s.pending) items.push({ kind: 'user', key: 'pending', text: s.pending, pending: true })
  const turn = s.turn
  if (turn && (turnActive(turn) || s.live || (s.toolsTurn === turn.id && s.tools.length) || s.stopped?.turnId === turn.id)) {
    const a = answerOf(turn.id, `turn-${turn.id}`)
    if (s.toolsTurn === turn.id) a.tools.push(...s.tools.filter((x) => !a.tools.some((y) => y.callId === x.callId)))
    if (s.live?.turnId === turn.id) a.live = s.live.text
    if (s.stopped?.turnId === turn.id) a.stopped = s.stopped.text
    if (turnActive(turn)) a.status = turn.state as 'queued' | 'answering' | 'waiting'
  } else if (s.pending) {
    items.push({ kind: 'answer', key: 'pending-answer', tools: [], texts: [], status: 'queued' })
  }
  return items
}

const knownTools = new Set(['read_report', 'read_attached', 'search_help', 'describe_screen', 'suggest_link', 'propose_edit', 'draft_agent_task', 'subject_ideas', 'rewrite'])

/**
 * An MCP tool's row: its server and tool, from its label ("Linear ·
 * create_issue") and its name for the model (mcp__linear__create_issue).
 */
export function mcpOf(row: Pick<ToolRow, 'name' | 'label'>): { key: string; server: string; tool: string; label: string } | undefined {
  const x = splitModelName(row.name)
  if (!x) return undefined
  const suffix = ` · ${x.tool}`
  if (row.label && row.label.endsWith(suffix)) return { key: x.key, server: row.label.slice(0, -suffix.length), tool: x.tool, label: row.label }
  // A tool whose own name isn't safe for the model (mcp__linear__note_b_ for "note<b>") keeps it in the label.
  const at = row.label ? row.label.lastIndexOf(' · ') : -1
  if (row.label && at > 0) return { key: x.key, server: row.label.slice(0, at), tool: row.label.slice(at + 3), label: row.label }
  return { key: x.key, server: x.key, tool: x.tool, label: `${x.key}${suffix}` }
}

/** "Read report · September Product Update · 0.3 s"; "Linear · search_issues · 1.2 s". */
function useToolLine() {
  const t = useT()
  return (row: ToolRow) => {
    const mcp = mcpOf(row)
    if (mcp) {
      if (row.code === 'denied') return t(mx.rowDenied, { label: mcp.label, server: mcp.server })
      if (row.code === 'outcome_unknown') return t(mx.rowUnknown, { label: mcp.label, server: mcp.server })
      const line = row.ok ? mcp.label : t(mx.rowProblem, { label: mcp.label, code: row.code ?? '' })
      return row.ms >= 100 ? `${line} · ${t(m.seconds, { s: (row.ms / 1000).toFixed(1) })}` : line
    }
    const verb = knownTools.has(row.name) ? t(m.tool, { name: row.name }) : row.name
    const parts = [verb]
    if (row.ok) {
      if (row.label) parts.push(row.label)
    } else {
      parts.push(t(m.toolFailed, { code: row.code ?? '' }))
    }
    if (row.ms >= 100) parts.push(t(m.seconds, { s: (row.ms / 1000).toFixed(1) }))
    return parts.join(' · ')
  }
}

/** A tool row: muted, expandable to what was called and what came back (the projected detail). */
function ToolRowView({ row }: { row: ToolRow }) {
  const t = useT()
  const line = useToolLine()
  const [open, setOpen] = useState(false)
  const detail = [`${row.name} · ${row.callId}`, `→ ${row.ok ? row.label || row.name : (row.code ?? '')}`, t(m.milliseconds, { ms: row.ms })].join('\n')
  return (
    <>
      <button type="button" className={row.ok ? 'asst-tool' : 'asst-tool failed'} aria-expanded={open} onClick={() => setOpen((o) => !o)}>
        <AsstIcon name={row.ok ? 'tool' : 'x'} size={14} />
        <span>{line(row)}</span>
        <AsstIcon name="chevron" size={14} className={open ? 'asst-flip' : undefined} />
      </button>
      {open ? <pre className="asst-tooldetail">{detail}</pre> : null}
    </>
  )
}

/** A handoff into an existing screen (it only navigates). */
function HandoffCard({ card }: { card: NonNullable<ToolRow['card']> }) {
  const t = useT()
  const { slug } = useWorkspace()
  // The app only links to its own screens in this workspace.
  if (!card.path.startsWith(`/w/${slug}/`) && card.path !== `/w/${slug}`) return null
  const desc = card.kind === 'proposal' ? m.cardProposal : card.kind === 'agent_task' ? m.cardAgent : m.cardLink
  const label = card.kind === 'proposal' ? m.reviewProposal : card.kind === 'agent_task' ? m.startAgent : m.openCard
  return (
    <div className="asst-handoff">
      <AsstIcon name={card.kind === 'link' ? 'link' : 'mail'} />
      <div className="grow">
        <strong>{card.title}</strong>
        <small>{t(desc)}</small>
      </div>
      <Link to={card.path as never} className="btn">
        {t(label)}
      </Link>
    </div>
  )
}

function HostReputeCard({ text }: { text: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <StateCard
      tone="neutral"
      badge={t(m.hostReputeBadge)}
      title={t(m.hostReputeTitle)}
      actions={
        <>
          <ButtonAnchor href={HOSTREPUTE} variant="primary">
            {t(m.openHostRepute)}
          </ButtonAnchor>
          <ButtonLink to="/w/$workspace/deliverability" params={{ workspace: slug }}>
            {t(m.openDeliverability)}
          </ButtonLink>
        </>
      }
    >
      <p>{text}</p>
    </StateCard>
  )
}

type AnswerItem = Extract<Item, { kind: 'answer' }>
const sameList = (a: readonly unknown[], b: readonly unknown[]) => a.length === b.length && a.every((x, i) => x === b[i])

/**
 * An answer renders again only when it changed: a streaming answer's
 * deltas, or a keystroke in the composer, leave the finished ones alone
 * (buildItems makes new items each time, from the same messages and rows).
 */
const Answer = memo(
  AnswerView,
  (a, b) =>
    a.item.key === b.item.key &&
    a.item.status === b.item.status &&
    a.item.live === b.item.live &&
    a.item.stopped === b.item.stopped &&
    sameList(a.item.tools, b.item.tools) &&
    sameList(a.item.texts, b.item.texts),
)

function AnswerView({ item }: { item: AnswerItem }) {
  const t = useT()
  const id = useId()
  const streaming = item.status !== undefined
  const cards = item.tools.map((r) => r.card).filter((c): c is NonNullable<ToolRow['card']> => Boolean(c))
  return (
    <article className="asst-msg ai" aria-labelledby={id} aria-busy={streaming || undefined}>
      <h3 className="vh" id={id}>
        {t(m.answer)}
      </h3>
      {item.tools.map((row) => (
        <ToolRowView key={row.callId} row={row} />
      ))}
      {item.texts.map((msg) =>
        msg.card === 'hostrepute' ? (
          <HostReputeCard key={msg.id} text={msg.text} />
        ) : (
          <div key={msg.id} className="asst-md">
            <MarkdownText text={msg.text} />
            {msg.partial ? <p className="asst-status">{t(m.partialNote)}</p> : null}
          </div>
        ),
      )}
      {item.live ? (
        <div className="asst-md">
          <StreamingMarkdown text={item.live} />
        </div>
      ) : null}
      {item.stopped ? (
        <div className="asst-md">
          <MarkdownText text={item.stopped} />
        </div>
      ) : null}
      {cards.map((c, i) => (
        <HandoffCard key={i} card={c} />
      ))}
      {/* A waiting turn says so under the conversation, with its card. */}
      {streaming && !item.live && item.status !== 'waiting' ? (
        <p className="asst-status">
          <span className="asst-pulse" aria-hidden="true" />
          {item.status === 'queued' ? t(m.queued) : t(m.answering)}
        </p>
      ) : null}
    </article>
  )
}

const UserMessage = memo(UserMessageView, (a, b) => a.item.key === b.item.key && a.item.text === b.item.text && a.item.pending === b.item.pending)

function UserMessageView({ item }: { item: Extract<Item, { kind: 'user' }> }) {
  const t = useT()
  const id = useId()
  return (
    <article className="asst-msg user" aria-labelledby={id}>
      <h3 className="vh" id={id}>
        {t(m.you)}
      </h3>
      <p>{item.text}</p>
    </article>
  )
}

/**
 * The conversation (AssistantPanel): the person's bubbles, the answers with
 * their tool rows, handoff cards and Markdown, and the summary marker.
 * Streaming isn't read out token by token: the log is aria-live off and a
 * separate polite region announces the start, the end and confirmations.
 */
export function Conversation({ items, children }: { items: Item[]; children?: ReactNode }) {
  const t = useT()
  return (
    <>
      {items.map((it) =>
        it.kind === 'user' ? (
          <UserMessage key={it.key} item={it} />
        ) : it.kind === 'summary' ? (
          <div key={it.key} className="asst-summary" role="separator" aria-label={t(m.summaryMarker)}>
            <span>{t(m.summaryMarker)}</span>
          </div>
        ) : (
          <Answer key={it.key} item={it} />
        ),
      )}
      {children}
    </>
  )
}
