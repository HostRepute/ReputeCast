import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { m } from './approval.messages'

/**
 * The approval review of a request to publish a transactional template: the
 * version it lets its author publish, what changes, the breaking-change
 * check and the test renders it was asked on.
 */
export function TransactionalApprovalCard({ snapshot: s }: { snapshot: Schemas['ApprovalTransactionalSnapshot'] }) {
  const t = useT()
  return (
    <section className="card" aria-labelledby="ap-transactional">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-transactional">
            {t(m.cardTitle)}
          </h2>
          <span className="k">{t(m.cardCaption)}</span>
        </div>
        {s.published_version ? <Badge tone="ok">{t(m.published, { version: s.published_version })}</Badge> : null}
      </div>
      <div className="pad col gap-3!">
        <dl className="kv">
          <dt>{t(m.version)}</dt>
          <dd>
            {s.liveVersion ? t(m.versionReplaces, { version: s.version, key: s.key, live: s.liveVersion }) : t(m.versionValue, { version: s.version, key: s.key })}
          </dd>
          {s.category ? (
            <>
              <dt>{t(m.category)}</dt>
              <dd>{s.category}</dd>
            </>
          ) : null}
        </dl>
        {s.subjectAfter || s.variables.length > 0 || s.contentSummary ? (
          <div className="diff">
            {s.subjectAfter ? (
              <>
                <div className="ctx font-semibold">{t(m.subject)}</div>
                {s.subjectBefore ? <div className="del">− {s.subjectBefore}</div> : null}
                <div className="add">+ {s.subjectAfter}</div>
              </>
            ) : null}
            {s.variables.length > 0 ? (
              <>
                <div className="ctx font-semibold">{t(m.variables)}</div>
                {s.variables.map((v) => (
                  <div key={v} className={v.startsWith('+') ? 'add' : v.startsWith('−') ? 'del' : 'ctx'}>
                    {v}
                  </div>
                ))}
              </>
            ) : null}
            <div className="ctx font-semibold">{t(m.content)}</div>
            <div className="ctx">{s.contentSummary || t(m.noContentChange)}</div>
          </div>
        ) : null}
        {s.breaking ? (
          <div className="note w" role="note">
            <span>
              <b>{t(m.breaking)}</b> · {s.breaking}
            </span>
          </div>
        ) : null}
        {s.testRenders.length > 0 ? (
          <div className="flex flex-col gap-1.5">
            <span className="k">{t(m.testRenders)}</span>
            <div className="flex flex-wrap gap-1.5">
              {s.testRenders.map((r) => (
                <Badge key={r.name} tone={r.ok ? 'ok' : 'bad'}>
                  {r.name} · {r.ok ? t(m.renders) : t(m.fails)}
                </Badge>
              ))}
            </div>
          </div>
        ) : null}
        <span className="k">{t(m.checksLine, { passed: s.passed, warnings: s.warnings })}</span>
      </div>
    </section>
  )
}

/** The decision card's note for a template publish request (the campaign note talks about audiences and schedules). */
export function TransactionalDecisionNote({ author }: { author: string }) {
  const t = useT()
  return (
    <div className="note n text-[12.5px]">
      <span>{t(m.decisionNote, { name: author.split(' ')[0] })}</span>
    </div>
  )
}

/** What rejecting or requesting changes does to a template publish request. */
export function TransactionalDecisionDetail({ kind, author }: { kind: 'reject' | 'request_changes'; author: string }) {
  const t = useT()
  return <>{t(kind === 'reject' ? m.rejectDetail : m.changesDetail, { name: author })}</>
}
