import { Handle, Position, useUpdateNodeInternals, type Node, type NodeProps } from '@xyflow/react'
import { memo, type CSSProperties, type ReactNode } from 'react'
import { count } from '@/lib/format'
import { cn } from '@/lib/cn'
import {
  borderColor,
  durationMs,
  isCompact,
  isTerminal,
  nodeById,
  nodeDetail,
  nodeTitle,
  triggerKindWord,
  typeColors,
  typeName,
  wholePct,
  type NodeData,
  type StepType,
} from '../graph'
import { useCountUp, useFlowView } from '../view'
import { T, useT } from '@/i18n'
import { mf } from '../flow.messages'
import { m as smsStep } from '@/features/sms/automation.messages'
import { smsNodeLines, useSmsTextParts } from '@/features/sms/automationNode'
import { pushNodeLines, usePushEstimate } from '@/features/push/AutomationStep'
import { useWhatsAppStepEstimate, whatsAppNodeHead, whatsAppNodeLines } from '@/features/whatsapp/AutomationStep'
import { StoreNodeLine } from '@/features/commerce/automation'

export type StepNodeType = Node<NodeData, StepType>

const providerColors: Record<string, string> = { ses: 'var(--chart-ses)', postal: 'var(--chart-postal)', postmark: 'var(--chart-postmark)' }

/** Whole percent of a part. */
function share(part: number, whole: number): number {
  return whole > 0 ? (part / whole) * 100 : 0
}

const bold = (chunks: ReactNode[]) => <b className="n">{chunks}</b>
const plain = (chunks: ReactNode[]) => <span>{chunks}</span>

/**
 * One step on the canvas: a header with the step type, the title, its
 * settings and live stats (AutomationFlow board).
 */
function StepNodeImpl({ id, type, data, selected }: NodeProps<StepNodeType>) {
  const v = useFlowView()
  const t = useCountUp()
  const tr = useT()
  const smsParts = useSmsTextParts(type === 'sms')
  // Who the push and WhatsApp steps reach (X22): one request per flow, and per WhatsApp step.
  const pushEst = usePushEstimate(v.automationId, type === 'push')
  const waEst = useWhatsAppStepEstimate(data.whatsapp, type === 'whatsapp')
  const updateNodeInternals = useUpdateNodeInternals()
  const node = nodeById(v.graph, id) ?? { id, type, data, position: { x: 0, y: 0 } }
  const st = v.stats.get(id)
  const isNew = v.published && !v.liveIds.has(id)
  const neverPublished = !v.publishedIds.has(id)
  const n = (x: number) => count(Math.round(x * t))
  const p = (x: number) => wholePct(x * t)
  const color = typeColors[type]
  const delay = v.entrance?.nodes.get(id)
  const hasIssue = v.issues.has(id)
  const style: CSSProperties & Record<string, string> = { '--fl-d': `${delay ?? 0}s` }
  const border = borderColor(type)
  if (border && !selected && !hasIssue) style.borderColor = border

  // Header: the step type plus what matters most here.
  let header = typeName(type)
  if (type === 'trigger' && data.trigger?.kind) header += ` · ${triggerKindWord(data.trigger.kind)}`
  const variant = v.graph.edges.find((e) => e.target === id && nodeById(v.graph, e.source)?.type === 'split')?.data.path
  const providers = st?.providers ?? []
  const showProviders = type === 'email' && v.showStats && providers.length > 1 && !neverPublished
  if (variant && type === 'email') header += ` · ${variant}`
  else if (isNew) header += ` · ${tr(mf.newInVersion, { version: v.draftVersion })}`
  else if (showProviders) header += ` · ${data.email?.route ? tr(mf.routeOverride) : tr(mf.automationsRoute)}`
  else if (type === 'sms') header += ` · ${tr(smsStep.byCountry)}`
  else if (type === 'whatsapp') header += ` · ${whatsAppNodeHead(data.whatsapp, tr)}`

  const title = nodeTitle(v.graph, node, v.refs)
  const detail = nodeDetail(node)
  const lines: ReactNode[] = []
  const stat = v.showStats && v.hasStats && !neverPublished
  const noStats = v.showStats && v.hasStats && neverPublished && hasStatsLine(type, data)
  switch (type) {
    case 'trigger':
      if (stat || detail)
        lines.push(
          <div className="st" key="s">
            {stat ? (
              <span>
                <T msg={mf.statEntered} values={{ count: n(st?.entered ?? 0), b: bold }} />
              </span>
            ) : null}
            {detail ? <span>{detail}</span> : null}
            {stat && data.trigger?.kind === 'abandoned_checkout' ? <StoreNodeLine kind="trigger" /> : null}
          </div>,
        )
      break
    case 'delay':
      if (stat && durationMs(data.delay) >= 3_600_000)
        lines.push(
          <div className="st" key="s">
            <span>
              <T msg={mf.statWaiting} values={{ count: n(st?.entered ?? 0), b: bold }} />
            </span>
          </div>,
        )
      break
    case 'email':
      if (stat) {
        const delivered = st?.delivered ?? 0
        lines.push(
          <div className="st" key="s">
            <span>
              <T msg={mf.statSent} values={{ count: n(st?.sent ?? 0), b: bold }} />
            </span>
            <span>
              <T msg={showProviders ? mf.statHumanOpens : mf.statOpens} values={{ rate: p(share(st?.openedHuman ?? 0, delivered)), b: bold }} />
            </span>
            <span>
              <T msg={mf.statClicks} values={{ rate: p(share(st?.clicked ?? 0, delivered)), b: bold }} />
            </span>
          </div>,
        )
        if (showProviders)
          lines.push(
            <div className="st" key="p">
              {providers.map((pr) =>
                pr.fallback ? (
                  <span key={pr.providerId} className="badge mut">
                    <span className="dot" style={{ color: providerColors[adapterOf(v, pr.providerId)] ?? 'var(--ink-3)' }} aria-hidden="true" />
                    <T msg={mf.statFailover} values={{ provider: pr.short, count: n(pr.sent), n: plain }} />
                  </span>
                ) : (
                  <span key={pr.providerId} className="badge pur">
                    {pr.short} <span>{n(pr.sent)}</span>
                  </span>
                ),
              )}
            </div>,
          )
      }
      break
    case 'goal':
      if (stat)
        lines.push(
          <div className="st" key="s">
            {data.condition?.kind === 'placed_order' ? (
              <StoreNodeLine kind="goal" />
            ) : (
              <span>
                <T msg={mf.statReached} values={{ count: n(st?.goalReached ?? 0), rate: p(share(st?.goalReached ?? 0, v.entered)), b: bold }} />
              </span>
            )}
            {data.condition?.kind === 'placed_order' ? <span>{tr(mf.statOfEntered, { rate: p(share(st?.goalReached ?? 0, v.entered)), count: n(v.entered) })}</span> : null}
          </div>,
        )
      break
    case 'exit':
      if (stat)
        lines.push(
          <div className="st" key="s">
            <span>
              <T msg={mf.statExited} values={{ count: n(st?.exited ?? 0), b: bold }} />
            </span>
          </div>,
        )
      break
    case 'filter':
      if (stat)
        lines.push(
          <div className="st" key="s">
            <span>
              <T msg={mf.statPassed} values={{ count: n(st?.completed ?? 0), b: bold }} />
            </span>
            <span>
              <T msg={mf.statFilteredOut} values={{ count: n(st?.dropped ?? 0), b: bold }} />
            </span>
          </div>,
        )
      break
    case 'webhook':
    case 'list':
      if (detail || (stat && type === 'list'))
        lines.push(
          <div className="st" key="s">
            {detail ? <span className="mono">{detail}</span> : null}
            {stat && type === 'list' ? (
              <span>
                <T msg={mf.statPassed} values={{ count: n(st?.entered ?? 0), b: bold }} />
              </span>
            ) : null}
            {stat && type === 'webhook' && (st?.failed ?? 0) > 0 ? (
              <span>
                <T msg={mf.statFailedCalls} values={{ count: n(st?.failed ?? 0), b: bold }} />
              </span>
            ) : null}
          </div>,
        )
      break
    case 'sms': {
      const parts = smsNodeLines(data.sms, tr, smsParts)
      if (parts.length)
        lines.push(
          <div className="st" key="s">
            {parts.map((x) => (
              <span key={x}>{x}</span>
            ))}
          </div>,
        )
      break
    }
    case 'push':
    case 'whatsapp': {
      const parts = type === 'push' ? pushNodeLines(data.push, tr, pushEst) : whatsAppNodeLines(data.whatsapp, tr, waEst)
      if (parts.length)
        lines.push(
          <div className="st" key="s">
            {parts.map((x) => (
              <span key={x}>{x}</span>
            ))}
          </div>,
        )
      break
    }
    case 'branch':
      if (detail)
        lines.push(
          <div className="st" key="s">
            <span>{detail}</span>
          </div>,
        )
      break
  }
  if (noStats)
    lines.push(
      <div className="st" key="n">
        <span>{tr(mf.noStatsUntilPublished)}</span>
      </div>,
    )
  return (
    <div
      className={cn('fn', isCompact(type) && 'compact', isNew && 'new', selected && 'sel', hasIssue && 'issue', delay === undefined && 'later')}
      style={style}
      onAnimationEnd={(e) => {
        // React Flow measured the handles while the step was still sliding in;
        // measure again now that it sits still, so edges meet the handles.
        if (e.animationName === 'fl-node-in' && e.target === e.currentTarget) updateNodeInternals(id)
      }}
    >
      <span className="hit" aria-hidden="true" />
      {type !== 'trigger' ? <Handle type="target" position={Position.Top} className="fn-hb i" /> : null}
      {/* A goal leads on where the flow carries past it (the Abandoned cart starter, X20). */}
      {!isTerminal(type) || (type === 'goal' && v.graph.edges.some((e) => e.source === id)) ? (
        <Handle type="source" position={Position.Bottom} className="fn-hb o" />
      ) : null}
      <div className="hd" style={{ color }}>
        <span className={cn('dot', type === 'trigger' && v.live && 'live')} aria-hidden="true" />
        {header}
      </div>
      <div className="t">{title}</div>
      {lines}
    </div>
  )
}

function hasStatsLine(type: StepType, data: NodeData): boolean {
  switch (type) {
    case 'email':
    case 'goal':
    case 'exit':
    case 'filter':
    case 'list':
      return true
    case 'delay':
      return durationMs(data.delay) >= 3_600_000
  }
  return false
}

function adapterOf(v: ReturnType<typeof useFlowView>, providerId: string): string {
  const name = v.refs.providers.get(providerId)?.name.toLowerCase() ?? ''
  if (name.includes('ses')) return 'ses'
  if (name.includes('postmark')) return 'postmark'
  if (name.includes('postal')) return 'postal'
  return ''
}

export const StepNode = memo(StepNodeImpl)

export const nodeTypes = {
  trigger: StepNode,
  delay: StepNode,
  email: StepNode,
  branch: StepNode,
  filter: StepNode,
  split: StepNode,
  webhook: StepNode,
  list: StepNode,
  goal: StepNode,
  exit: StepNode,
  sms: StepNode,
  push: StepNode,
  whatsapp: StepNode,
}
