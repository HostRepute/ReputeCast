import { useId, useState } from 'react'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { deniedReason } from '@/lib/session'
import { pretty } from '@/features/mcpservers/tools'
import { mcpOf } from './Conversation'
import { AsstIcon } from './icons'
import { mx } from './mcp.messages'
import { m } from './messages'
import type { Confirmation, ToolRow } from './reducer'
import { StateCard } from './states'
import type { ThreadController } from './ThreadView'

export const CONFIRM_ID = 'asst-confirm'

/** The call a waiting turn asks about, while it still asks. */
export function pendingConfirmation(c: ThreadController): Confirmation | undefined {
  const conf = c.state.confirmation
  const turn = c.state.turn
  return conf && conf.state === 'pending' && turn?.state === 'waiting' && turn.id === conf.turnId ? conf : undefined
}

/**
 * The confirmation card (AssistantPanel › Confirm): server · tool, the
 * model that asked, the exact arguments as plain JSON (scrolls, copies),
 * Deny and Allow once. A decision is sent once; a closed call (409) says
 * so, and so does one that expired (24 hours without a decision).
 */
export function ConfirmCard({ c, conf }: { c: ThreadController; conf: Confirmation }) {
  const t = useT()
  const titleId = useId()
  const [copied, setCopied] = useState(false)
  const args = pretty(conf.arguments)
  const values = { server: conf.serverName, tool: conf.tool }
  const busy = c.deciding
  const expired = conf.state === 'expired'
  const copy = () => {
    void navigator.clipboard?.writeText(args).then(
      () => {
        setCopied(true)
        window.setTimeout(() => setCopied(false), 2000)
      },
      () => setCopied(false),
    )
  }
  return (
    <section className="asst-confirm" id={CONFIRM_ID} aria-labelledby={titleId} tabIndex={-1}>
      <div className="asst-confirm-head">
        <strong id={titleId}>{t(mx.title, values)}</strong>
        <Badge tone={expired ? 'neutral' : 'warn'}>{t(expired ? m.expiredBadge : mx.badge)}</Badge>
      </div>
      {expired ? null : <p className="sub">{t(mx.check, { server: conf.serverName })}</p>}
      <small className="k">{t(mx.askedBy, { model: c.model?.modelName || c.model?.model || '', name: conf.modelName })}</small>
      <div className="asst-confirm-args">
        <ScrollPre className="asst-tooldetail" label={t(mx.argsLabel, values)} vertical>
          {args}
        </ScrollPre>
        <button type="button" className="asst-confirm-copy" onClick={copy}>
          <AsstIcon name={copied ? 'check' : 'copy'} size={14} />
          <span>{copied ? t(common.copied) : t(common.copy)}</span>
        </button>
        <span className="vh" role="status">
          {copied ? t(common.copied) : ''}
        </span>
      </div>
      {expired ? null : <small>{t(mx.note)}</small>}
      {c.confProblem?.id === conf.id ? (
        <p className="asst-status asst-confirm-problem">
          <AsstIcon name="warn" size={14} />
          <span>{t(c.confProblem.code === 'mcp_tool_blocked' ? mx.confirmBlocked : mx.confirmChanged, values)}</span>
        </p>
      ) : null}
      {expired ? (
        <p className="asst-status">{t(mx.expired, values)}</p>
      ) : c.confirmClosed === conf.id ? (
        <p className="asst-status" role="status">
          {t(mx.closed)}
        </p>
      ) : (
        <div className="asst-confirm-acts">
          <Button onClick={() => void c.decide(conf, 'deny')} loading={busy === 'deny'} disabled={Boolean(busy)}>
            {t(mx.deny)}
          </Button>
          <Button variant="primary" onClick={() => void c.decide(conf, 'allow')} loading={busy === 'allow'} disabled={Boolean(busy) || c.confProblem?.id === conf.id}>
            {t(mx.allow)}
          </Button>
        </div>
      )}
    </section>
  )
}

/** "Waiting for you · Linear · create_issue  Go to the request" (the board's status line). */
export function WaitingLine({ conf }: { conf: Confirmation }) {
  const t = useT()
  return (
    <p className="asst-status asst-waiting">
      <AsstIcon name="clock" size={14} />
      <span>{t(mx.waiting, { server: conf.serverName, tool: conf.tool })}</span>
      <a
        href={`#${CONFIRM_ID}`}
        onClick={(e) => {
          e.preventDefault()
          const card = document.getElementById(CONFIRM_ID)
          card?.scrollIntoView({ block: 'nearest' })
          card?.focus({ preventScroll: true })
        }}
      >
        {t(mx.goTo)}
      </a>
    </p>
  )
}

const MCP_STATES = new Set(['mcp_sign_in', 'mcp_tool_changed', 'mcp_tool_blocked', 'mcp_unreachable', 'mcp_timeout', 'mcp_too_large'])

/**
 * What a turn's MCP calls ran into (AssistantStates › Sign in again, Tool
 * changed…): one card per server and problem, with what can be done.
 */
export function MCPStates({ c, rows }: { c: ThreadController; rows: ToolRow[] }) {
  const t = useT()
  const seen = new Set<string>()
  const cards = rows.flatMap((row) => {
    const mcp = mcpOf(row)
    if (!mcp || !row.code || !MCP_STATES.has(row.code)) return []
    const key = `${row.code}:${mcp.key}`
    if (seen.has(key)) return []
    seen.add(key)
    return [{ key, code: row.code, mcp }]
  })
  if (!cards.length) return null
  return (
    <>
      {cards.map(({ key, code, mcp }) => {
        const values = { server: mcp.server, tool: mcp.tool }
        const server = c.servers.find((s) => s.key === mcp.key)
        switch (code) {
          case 'mcp_sign_in':
            return (
              <StateCard
                key={key}
                tone="warn"
                badge={t(mx.signInBadge)}
                title={t(mx.signInTitle, values)}
                actions={
                  server ? (
                    <Button variant="primary" onClick={() => void c.signIn(server)}>
                      {t(mx.signIn, values)}
                    </Button>
                  ) : (
                    <ButtonLink variant="primary" to="/w/$workspace/settings/integrations/mcp-servers" params={{ workspace: c.slug }}>
                      {t(mx.signIn, values)}
                    </ButtonLink>
                  )
                }
              >
                <p>{t(mx.signInBody, values)}</p>
              </StateCard>
            )
          case 'mcp_tool_changed':
            return (
              <StateCard
                key={key}
                tone="warn"
                badge={t(mx.changedBadge)}
                title={t(mx.changedTitle)}
                actions={
                  c.canManage ? (
                    <ButtonLink variant="primary" to="/w/$workspace/settings/integrations/mcp-servers" params={{ workspace: c.slug }} search={server ? { server: server.id } : {}}>
                      {t(mx.reviewChange)}
                    </ButtonLink>
                  ) : (
                    <Button variant="primary" disabledReason={deniedReason('integrations:manage', c.helpers)}>
                      {t(mx.reviewChange)}
                    </Button>
                  )
                }
              >
                <p>{t(mx.changedBody, values)}</p>
              </StateCard>
            )
          case 'mcp_tool_blocked':
            return (
              <StateCard key={key} tone="neutral" badge={t(mx.blockedBadge)} title={t(mx.blockedTitle, values)}>
                <p>{t(mx.blockedBody)}</p>
              </StateCard>
            )
          case 'mcp_unreachable':
            return (
              <StateCard key={key} tone="bad" badge={t(mx.failedBadge)} title={t(mx.unreachableTitle, values)}>
                <p>{t(mx.unreachableBody, values)}</p>
              </StateCard>
            )
          case 'mcp_timeout':
            return (
              <StateCard key={key} tone="bad" badge={t(mx.failedBadge)} title={t(mx.timeoutTitle, values)}>
                <p>{t(mx.timeoutBody, values)}</p>
              </StateCard>
            )
          default:
            return (
              <StateCard key={key} tone="bad" badge={t(mx.failedBadge)} title={t(mx.tooLargeTitle, values)}>
                <p>{t(mx.tooLargeBody, values)}</p>
              </StateCard>
            )
        }
      })}
    </>
  )
}
