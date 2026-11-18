import { useState } from 'react'
import { ApiError, errorMessage } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Note } from '@/components/ui/Note'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { toast } from '@/components/ui/Toast'
import { formatList, useT, type MessageDescriptor } from '@/i18n'
import { common } from '@/i18n/common'
import { useWorkspace } from '@/lib/session'
import { decideTool, useRefreshMCP, type MCPDecision, type MCPServer, type MCPTool } from './api'
import { m } from './messages'
import { changedParts, hints, isChanged, pretty, shortHash, type ChangedPart } from './tools'

type Translate = ReturnType<typeof useT>

/** A tool's state, as the table's badge says it. */
export function toolState(tool: MCPTool): { label: MessageDescriptor; tone: 'ok' | 'warn' | 'neutral' } {
  if (tool.removed) return { label: m.stateRemoved, tone: 'neutral' }
  if (isChanged(tool)) return { label: m.stateChanged, tone: 'warn' }
  switch (tool.decision) {
    case 'read_only':
      return { label: m.stateApproved, tone: 'ok' }
    case 'writes':
      return { label: m.stateAsks, tone: 'warn' }
    case 'blocked':
      return { label: m.stateBlocked, tone: 'neutral' }
    default:
      return { label: m.statePending, tone: 'warn' }
  }
}

export const kindLabel: Record<MCPTool['decision'], MessageDescriptor> = {
  read_only: m.kindReadOnly,
  writes: m.kindWrites,
  blocked: m.kindBlocked,
  pending: m.kindPending,
}

/** The tool as its server sent it, in plain text: the description, inputSchema and the annotations. */
export function untrustedText(t: Translate, tool: Pick<MCPTool, 'description' | 'inputSchema' | 'annotations'>): string {
  const lines = [tool.description.trim() || t(m.noDescription), '', t(m.preSchema, { schema: pretty(tool.inputSchema) })]
  const h = hints(tool.annotations)
  if (h) lines.push(t(m.preAnnotations, { annotations: h }))
  return lines.join('\n')
}

/** A 409 mcp_tool_reviewed: the tool changed while it was reviewed. */
function isStale(e: unknown): boolean {
  return e instanceof ApiError && e.status === 409 && e.code === 'mcp_tool_reviewed'
}

function useDecide(server: MCPServer) {
  const { slug } = useWorkspace()
  const t = useT()
  const refresh = useRefreshMCP()
  return async (tool: MCPTool, decision: MCPDecision): Promise<'ok' | 'stale' | 'failed'> => {
    try {
      // The hash is the version the person read: one that changed since is refused.
      await decideTool(slug, server.id, tool.id, { decision, hash: tool.hash })
      await refresh()
      toast({ kind: 'success', title: t(decision === 'read_only' ? m.savedReadOnly : decision === 'writes' ? m.savedWrites : m.savedBlocked, { tool: tool.name }) })
      return 'ok'
    } catch (e) {
      if (isStale(e)) {
        await refresh()
        return 'stale'
      }
      toast({ kind: 'error', title: t(m.decideFailed, { tool: tool.name }), detail: errorMessage(e) })
      return 'failed'
    }
  }
}

/**
 * Review (MCPServers › Review): the server's description and schema as
 * untrusted plain text, and the decision, sent with the hash shown. A tool
 * that changed meanwhile comes back refreshed, saying so. Read-only for
 * people who can't manage integrations.
 */
export function ReviewDialog({ server, tool, canManage, onClose }: { server: MCPServer; tool: MCPTool; canManage: boolean; onClose: () => void }) {
  const t = useT()
  const decide = useDecide(server)
  const [choice, setChoice] = useState<MCPDecision>(tool.decision === 'pending' ? 'writes' : tool.decision)
  const [busy, setBusy] = useState(false)
  // The version the person was shown when they last chose; a newer one says so.
  const [stale, setStale] = useState(false)
  const save = async () => {
    setBusy(true)
    const r = await decide(tool, choice)
    setBusy(false)
    if (r === 'ok') onClose()
    else if (r === 'stale') setStale(true)
  }
  const state = toolState(tool)
  const options: [MCPDecision, MessageDescriptor][] = [
    ['read_only', m.optReadOnly],
    ['writes', m.optWrites],
    ['blocked', m.optBlocked],
  ]
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && !busy && onClose()}
      title={canManage ? t(m.reviewTitle, { tool: tool.name }) : tool.name}
      description={t(m.reviewFrom, { server: server.name })}
      footer={
        canManage ? (
          <>
            <Button onClick={onClose} disabled={busy}>
              {t(common.cancel)}
            </Button>
            <Button variant="primary" onClick={() => void save()} loading={busy}>
              {t(common.save)}
            </Button>
          </>
        ) : (
          <Button onClick={onClose}>{t(common.close)}</Button>
        )
      }
    >
      <div className="mcp-dlg">
        {stale ? (
          <Note tone="warn" role="alert">
            {t(m.stale)}
          </Note>
        ) : null}
        <ScrollPre className="mcp-pre" label={t(m.untrustedLabel, { tool: tool.name })} vertical>
          {untrustedText(t, tool)}
        </ScrollPre>
        <span className="k mono text-[12px]">{t(m.version, { hash: shortHash(tool.hash) })}</span>
        {canManage ? (
          <>
            <fieldset className="mcp-radios">
              <legend className="vh">{t(m.kindLegend)}</legend>
              {options.map(([value, label]) => (
                <label key={value} className="mcp-radio">
                  <input type="radio" name={`kind-${tool.id}`} value={value} checked={choice === value} onChange={() => setChoice(value)} />
                  {t(label)}
                </label>
              ))}
            </fieldset>
            <small className="k">{t(m.pinNote)}</small>
          </>
        ) : (
          <span>
            <Badge tone={state.tone} dot>
              {t(state.label)}
            </Badge>
          </span>
        )}
      </div>
    </Dialog>
  )
}

function changedTitle(parts: ChangedPart[] | undefined): MessageDescriptor {
  if (!parts || !parts.length) return m.changedUnknown
  const has = (p: ChangedPart['part']) => parts.some((x) => x.part === p)
  if (has('description') && has('schema')) return m.changedBoth
  if (has('description')) return m.changedDescription
  if (has('schema')) return m.changedSchema
  return m.changedAnnotations
}

const partLabel: Record<ChangedPart['part'], MessageDescriptor> = { description: m.partDescription, schema: m.partSchema, annotations: m.partAnnotations }

/**
 * A tool that changed after approval (MCPServers › Changed): what it said
 * then and says now, part by part, in plain text; Block the tool or Approve
 * the new version (the same kind, at the new hash). Managers decide; the
 * rest are told who can.
 */
export function ChangedCard({ server, tool, canManage, helpers, onReview, id }: { server: MCPServer; tool: MCPTool; canManage: boolean; helpers: string[]; onReview: () => void; id: string }) {
  const t = useT()
  const decide = useDecide(server)
  const [busy, setBusy] = useState<MCPDecision>()
  const [stale, setStale] = useState(false)
  const parts = changedParts(tool)
  const titleId = `${id}-h`
  const run = async (d: MCPDecision) => {
    setBusy(d)
    const r = await decide(tool, d)
    setBusy(undefined)
    setStale(r === 'stale')
  }
  // The decision it had: approving again keeps it.
  const kept: MCPDecision = tool.decision === 'writes' ? 'writes' : 'read_only'
  return (
    <section className="card mcp-changed" id={id} aria-labelledby={titleId} tabIndex={-1}>
      <Badge tone="warn" dot>
        {t(m.changedBadge)}
      </Badge>
      <h2 className="h2" id={titleId}>
        {t(changedTitle(parts), { tool: tool.name })}
      </h2>
      {stale ? (
        <Note tone="warn" role="alert">
          {t(m.stale)}
        </Note>
      ) : null}
      {parts && parts.length ? (
        parts.map((p) => (
          <div key={p.part} className="grid gap-1.5">
            {parts.length > 1 || p.part !== 'description' ? <span className="mcp-part">{t(partLabel[p.part])}</span> : null}
            <div className="mcp-diff">
              {p.part === 'description' ? (
                <>
                  {/* The board's lines: "Before Search the Acme wiki." in plain text. */}
                  <p className="removed">
                    <strong>{t(m.before)}</strong> {p.before || t(m.noDescription)}
                  </p>
                  <p className="added">
                    <strong>{t(m.after)}</strong> {p.after || t(m.noDescription)}
                  </p>
                </>
              ) : (
                <>
                  <div className="removed">
                    <strong>{t(m.before)}</strong>
                    <pre className="code">{p.before}</pre>
                  </div>
                  <div className="added">
                    <strong>{t(m.after)}</strong>
                    <pre className="code">{p.after}</pre>
                  </div>
                </>
              )}
            </div>
          </div>
        ))
      ) : (
        <>
          <p className="m-0">{t(m.noApproved)}</p>
          <ScrollPre className="mcp-pre" label={t(m.untrustedLabel, { tool: tool.name })} vertical>
            {untrustedText(t, tool)}
          </ScrollPre>
        </>
      )}
      <p className="k m-0 max-w-[72ch]">{t(m.changedBody)}</p>
      <div className="mcp-acts">
        {canManage ? (
          parts && parts.length ? (
            <>
              <Button variant="danger" onClick={() => void run('blocked')} loading={busy === 'blocked'} disabled={Boolean(busy)}>
                {t(m.blockTool)}
              </Button>
              <Button onClick={() => void run(kept)} loading={busy === kept} disabled={Boolean(busy)}>
                {t(m.approveNew)}
              </Button>
            </>
          ) : (
            <Button onClick={onReview}>{t(m.reviewLabel, { tool: tool.name })}</Button>
          )
        ) : (
          <small className="k">{helpers.length ? t(m.changedHelpers, { helpers: formatList(helpers, 'disjunction') }) : t(m.changedNobody)}</small>
        )}
      </div>
    </section>
  )
}
