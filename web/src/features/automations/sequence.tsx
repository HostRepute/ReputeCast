import type { Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { count, pct } from '@/lib/format'
import { nodeById, nodeTitle, type FlowGraph, type Refs } from './graph'
import { m } from './complete.messages'

type Step = Schemas['AutomationSequenceStep']

const share = (n: number, of: number) => (of > 0 ? (n / of) * 100 : undefined)

/**
 * How a flow's emails do one after another (review 2026-11 AT-13):
 * delivered, human opens, clicks, unsubscribes and the share who went on
 * to the next email.
 */
export function SequenceStats({ sequence, graph, refs }: { sequence: Step[]; graph: FlowGraph; refs: Refs }) {
  const tr = useT()
  if (sequence.length === 0) return <span className="k">{tr(m.sequenceEmpty)}</span>
  const name = (id: string) => {
    const n = nodeById(graph, id)
    return n ? nodeTitle(graph, n, refs) : id
  }
  return (
    <ol className="fe-seqlist" aria-label={tr(m.sequenceTitle)}>
      {sequence.map((s) => {
        const next = s.next !== undefined && s.next !== null ? share(s.next, s.sent) : undefined
        return (
          <li key={s.nodeId}>
            <b className="[overflow-wrap:anywhere]">{name(s.nodeId)}</b>
            <dl className="fe-seqnums">
              <div>
                <dt>{tr(m.colDelivered)}</dt>
                <dd className="n">{count(s.delivered)}</dd>
              </div>
              <div>
                <dt>{tr(m.colOpens)}</dt>
                <dd className="n">{pct(share(s.openedHuman, s.delivered))}</dd>
              </div>
              <div>
                <dt>{tr(m.colClicks)}</dt>
                <dd className="n">{pct(share(s.clicked, s.delivered))}</dd>
              </div>
              <div>
                <dt>{tr(m.colUnsubs)}</dt>
                <dd className="n">{pct(share(s.unsubscribed, s.delivered))}</dd>
              </div>
            </dl>
            {s.next === undefined || s.next === null ? (
              <span className="k">{tr(m.lastEmail)}</span>
            ) : (
              <span className="flex items-center gap-2 text-[12.5px]">
                <span className="fe-seqbar" aria-hidden="true">
                  <i style={{ width: `${Math.min(100, Math.round(next ?? 0))}%` }} />
                </span>
                <span>{tr(m.nextShare, { percent: pct(next, 0) })}</span>
              </span>
            )}
          </li>
        )
      })}
    </ol>
  )
}
