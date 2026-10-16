import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Table, Td, Th } from '@/components/ui/Table'
import { formatList, useT } from '@/i18n'
import { clock, count, pct, shortDate, zonedParts } from '@/lib/format'
import { cn } from '@/lib/cn'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'
import { dateTime, IssueStatusBadge, signed, useDigestHistory, useDigestIssues, type Digest, type DigestIssue, type HistoryItem } from './shared'

/** The status cell's words after the badge: "2 new posts so far", "no new posts since it was turned on"… */
function statusDetail(t: ReturnType<typeof useT>, d: Digest, is: DigestIssue, firstSkip: boolean): string {
  switch (is.status) {
    case 'collecting':
      return t(m.postsSoFar, { count: is.posts })
    case 'waiting':
      return is.approval ? t(m.waitingFor, { names: formatList(d.approvals.approvers, 'disjunction') }) : ''
    case 'held':
      return t(m.heldReason, { reason: is.holdReason || 'readiness' })
    case 'draft':
      return t(m.draftNotSent)
    case 'not_sent':
      return t(m.notSentReason)
    case 'skipped':
      if (is.skipReason === 'person') return is.skipNote ? t(m.skipByNote, { name: is.skippedBy || '', note: is.skipNote }) : t(m.skipBy, { name: is.skippedBy || '' })
      if (is.skipReason === 'no_items' && firstSkip) return t(m.skipReasonFirst)
      return t(m.skipReason, { reason: is.skipReason || 'other' })
  }
  return ''
}

/** "#2" or "–": skipped and unsent issues use no number. */
function issueNumber(is: DigestIssue, d: Digest): string {
  if (is.number) return `#${is.number}`
  if (d.next && d.next.id === is.id) return `#${d.next.number}`
  return '–'
}

function Delta({ children }: { children: ReactNode }) {
  return <span className="k block text-[12px]">{children}</span>
}

/** Issues: sent, skipped, held, waiting and paused, each compared with the sent issue before it. */
export function IssuesCard({ d, onSkip, onPreview }: { d: Digest; onSkip: (issueId: string, number: number) => void; onPreview: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useDigestIssues(d.id)
  const skipReason = usePermissionReason('campaigns:pause', d.pauseHelpers)
  const items = q.data?.items ?? []
  // The oldest skip before anything was sent is the "since it was turned on" one.
  const firstSkipID = [...items].reverse().find((x) => x.status === 'skipped' && x.skipReason === 'no_items')?.id
  const firstSentNumber = Math.min(...items.filter((x) => x.number).map((x) => x.number!))
  const open = (is: DigestIssue) => is.status === 'collecting' || is.status === 'waiting' || is.status === 'held' || is.status === 'prepared' || is.status === 'preparing'
  const actions = (is: DigestIssue) => {
    const params = is.campaignId ? { workspace: slug, campaignId: is.campaignId } : undefined
    if (is.status === 'sent' || is.status === 'sending') {
      return (
        <>
          {params ? (
            <ButtonLink size="sm" to={is.status === 'sending' ? '/w/$workspace/campaigns/$campaignId/live' : '/w/$workspace/campaigns/$campaignId/report'} params={params}>
              {t(m.report)}
            </ButtonLink>
          ) : null}
          {params ? (
            <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/edit" params={params}>
              {t(m.open)}
            </ButtonLink>
          ) : null}
        </>
      )
    }
    if (is.status === 'draft' && params) {
      return (
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={params}>
          {t(m.reviewAndSend)}
        </ButtonLink>
      )
    }
    if (open(is)) {
      return (
        <>
          <Button size="sm" onClick={onPreview}>
            {t(m.preview)}
          </Button>
          <Button size="sm" disabledReason={skipReason} onClick={() => onSkip(is.id, is.number ?? (d.next?.number ?? d.sentIssues + 1))}>
            {t(m.skip)}
          </Button>
        </>
      )
    }
    return null
  }
  const approvedBy = (is: DigestIssue): ReactNode => {
    if (is.approval?.decidedBy && is.approval.status !== 'pending') {
      return (
        <span className="flex flex-col">
          <span>{is.approval.decidedBy.name}</span>
          {is.approval.decidedAt ? <span className="k">{clock(is.approval.decidedAt)}</span> : null}
        </span>
      )
    }
    if (is.status === 'collecting' && is.needsApproval) return <span className="k">{t(m.needsApprovalAt, { time: d.next?.prepareAt ? clock(d.next.prepareAt) : '–' })}</span>
    return <span className="k">–</span>
  }
  return (
    <section className="card" aria-labelledby="digest-issues">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="digest-issues">
            {t(m.issues)}
          </h2>
          <span className="k">{t(m.issuesSub)}</span>
        </div>
      </div>
      {q.isError ? (
        <p className="pad k m-0">{errorMessage(q.error)}</p>
      ) : !q.data ? (
        <div className="pad flex flex-col gap-3" aria-busy="true">
          {Array.from({ length: 3 }, (_, i) => (
            <Skeleton key={i} w="100%" h={28} />
          ))}
        </div>
      ) : items.length === 0 ? (
        <div className="pad flex flex-col items-center gap-2 py-8 text-center">
          <Icon name="calendar" size={24} />
          <span className="font-semibold">{t(m.noIssuesYet)}</span>
        </div>
      ) : (
        <>
          <div className="max-md:hidden">
            <Table caption={t(m.issuesLabel, { name: d.name })} className="[&_td]:align-top">
              <thead>
                <tr>
                  <Th>{t(m.colIssue)}</Th>
                  <Th>{t(m.colStatus)}</Th>
                  <Th align="right">{t(m.colPosts)}</Th>
                  <Th align="right">{t(m.colRecipients)}</Th>
                  <Th align="right">{t(m.colHumanOpens)}</Th>
                  <Th align="right">{t(m.colClicks)}</Th>
                  <Th>{t(m.colApprovedBy)}</Th>
                  <Th>
                    <span className="vh">{t(m.open)}</span>
                  </Th>
                </tr>
              </thead>
              <tbody>
                {items.map((is) => (
                  <tr key={is.id}>
                    <Td>
                      <span className="flex flex-col">
                        <span className="font-semibold">{issueNumber(is, d)}</span>
                        {is.sendAt ? <span className="k whitespace-nowrap">{dateTime(is.sendAt)}</span> : null}
                      </span>
                    </Td>
                    <Td>
                      <span className="flex flex-wrap items-center gap-2">
                        <IssueStatusBadge status={is.status} />
                        <span className="k">{statusDetail(t, d, is, is.id === firstSkipID)}</span>
                      </span>
                    </Td>
                    <Td align="right">
                      <span className="n font-semibold">{count(is.posts)}</span>
                      {is.status === 'sent' || is.status === 'sending' ? (
                        is.postsDelta != null && is.number !== firstSentNumber ? (
                          <Delta>{t(m.postsDelta, { delta: signed(is.postsDelta), number: (is.number ?? 1) - 1 })}</Delta>
                        ) : is.firstIssue ? (
                          <Delta>{t(m.firstIssue)}</Delta>
                        ) : null
                      ) : null}
                    </Td>
                    <Td align="right">
                      {is.recipients != null ? (
                        <>
                          <span className="n font-semibold">{is.estimated ? t(m.about, { count: count(is.recipients) }) : count(is.recipients)}</span>
                          {is.recipientsDelta != null ? <Delta>{signed(is.recipientsDelta)}</Delta> : null}
                        </>
                      ) : (
                        <span className="k">–</span>
                      )}
                    </Td>
                    <Td align="right">
                      {is.humanOpens != null ? (
                        <>
                          <span className="n">{pct(is.humanOpens)}</span>
                          {is.humanOpensDelta != null ? <Delta>{t(m.pts, { delta: signed(is.humanOpensDelta, 1) })}</Delta> : null}
                        </>
                      ) : (
                        <span className="k">–</span>
                      )}
                    </Td>
                    <Td align="right">
                      {is.clicks != null ? (
                        <>
                          <span className="n">{pct(is.clicks)}</span>
                          {is.clicksDelta != null ? <Delta>{t(m.pts, { delta: signed(is.clicksDelta, 1) })}</Delta> : null}
                        </>
                      ) : (
                        <span className="k">–</span>
                      )}
                    </Td>
                    <Td>{approvedBy(is)}</Td>
                    <Td>
                      <div className="row-actions flex flex-wrap justify-end gap-1.5">{actions(is)}</div>
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          </div>
          <ul className="m-0 flex list-none flex-col p-0 md:hidden" aria-label={t(m.issuesLabel, { name: d.name })}>
            {items.map((is) => (
              <li key={is.id} className="flex flex-col gap-1.5 border-b border-line-2 px-4 py-3 last:border-b-0">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-semibold">{is.sendAt && issueNumber(is, d) !== '–' ? t(m.issueListDate, { number: issueNumber(is, d).slice(1), date: dateTime(is.sendAt) }) : is.sendAt ? dateTime(is.sendAt) : '–'}</span>
                  <IssueStatusBadge status={is.status} />
                </div>
                <span className="k">
                  {is.status === 'sent' && is.recipients != null
                    ? is.approval?.decidedBy
                      ? t(m.issueListLine, {
                          count: is.posts,
                          recipients: count(is.recipients),
                          opens: pct(is.humanOpens),
                          clicks: pct(is.clicks),
                          name: is.approval.decidedBy.name,
                          time: is.approval.decidedAt ? clock(is.approval.decidedAt) : '–',
                        })
                      : t(m.issueListLineNoApproval, { count: is.posts, recipients: count(is.recipients), opens: pct(is.humanOpens), clicks: pct(is.clicks) })
                    : is.status === 'collecting' && is.recipients != null
                      ? t(m.issueListCollecting, { posts: t(m.postsSoFar, { count: is.posts }), recipients: count(is.recipients) })
                      : statusDetail(t, d, is, is.id === firstSkipID)}
                </span>
                {actions(is) ? <div className="flex flex-wrap gap-1.5">{actions(is)}</div> : null}
              </li>
            ))}
          </ul>
          <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-[18px] py-2.5">
            <span className="k">{d.archive ? t(m.issuesFoot) : t(m.issuesFootPrivate)}</span>
            {d.archive ? (
              <Link to="/w/$workspace/forms/archive" params={{ workspace: slug }} className="lnk text-[13px] font-semibold">
                {t(m.publicArchive)}
              </Link>
            ) : null}
          </div>
        </>
      )}
    </section>
  )
}

/** Human opens by issue: the last sent issues, one bar each, and the issue being collected. */
export function OpensChart({ d }: { d: Digest }) {
  const t = useT()
  const q = useDigestIssues(d.id)
  const sent = (q.data?.items ?? []).filter((x) => x.status === 'sent' && x.humanOpens != null && x.number).slice(0, 6).reverse()
  const next = d.next
  const last = sent[sent.length - 1]
  const prev = sent[sent.length - 2]
  const max = 50
  return (
    <section className="card" aria-labelledby="digest-chart">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="digest-chart">
            {t(m.chartTitle)}
          </h2>
          <span className="k">{t(m.chartSub)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {sent.length === 0 ? (
          <p className="k m-0">{t(m.chartEmpty)}</p>
        ) : (
          <div
            className="flex gap-2 pt-2"
            role="img"
            aria-label={t(m.chartLabel, { values: sent.map((x) => t(m.chartValue, { number: x.number!, value: pct(x.humanOpens) })).join(', ') })}
          >
            <div className="k relative h-[150px] w-8 shrink-0 text-right text-[11px]" aria-hidden="true">
              <span className="absolute right-0 top-0 -translate-y-1/2">50%</span>
              <span className="absolute right-0 top-1/2 -translate-y-1/2">25%</span>
              <span className="absolute bottom-0 right-0 translate-y-1/2">0%</span>
            </div>
            <div className="relative flex h-[150px] flex-1 items-end justify-around gap-4 border-b border-line" aria-hidden="true">
              <span className="absolute inset-x-0 top-0 border-t border-line-2" />
              <span className="absolute inset-x-0 top-1/2 border-t border-line-2" />
              {sent.map((x) => (
                <div key={x.id} className="relative flex h-full max-w-[90px] flex-1 flex-col items-center justify-end gap-1">
                  <span className="text-[12px] font-bold">{pct(x.humanOpens)}</span>
                  <div className="w-full rounded-t bg-accent" style={{ height: `${Math.min(100, ((x.humanOpens ?? 0) / max) * 100)}%` }} />
                </div>
              ))}
              {next && d.state === 'on' ? (
                <div className="relative flex h-full max-w-[90px] flex-1 flex-col items-center justify-end gap-1">
                  <span className="k text-[11px]">{t(m.chartCollecting)}</span>
                  <div className="h-[60%] w-full rounded-t border border-dashed border-line-strong" />
                </div>
              ) : null}
            </div>
          </div>
        )}
        {sent.length ? (
          <div className="ml-10 flex justify-around gap-4" aria-hidden="true">
            {sent.map((x) => (
              <div key={x.id} className="flex max-w-[90px] flex-1 flex-col items-center text-center">
                <span className="text-[12px] font-semibold">{t(m.chartBar, { number: x.number!, date: x.sendAt ? shortDate(x.sendAt) : '' })}</span>
                <span className="k text-[11px]">{t(m.chartClicks, { value: pct(x.clicks) })}</span>
              </div>
            ))}
            {next && d.state === 'on' ? (
              <div className="flex max-w-[90px] flex-1 flex-col items-center text-center">
                <span className="text-[12px] font-semibold">{t(m.chartBar, { number: next.number, date: next.sendAt ? shortDate(next.sendAt) : '' })}</span>
                <span className="k text-[11px]">{t(m.chartNotSent)}</span>
              </div>
            ) : null}
          </div>
        ) : null}
        {last && prev && last.humanOpens != null && prev.humanOpens != null ? (
          <p className="k m-0 text-[12px]">
            {t(m.chartCompare, {
              number: last.number!,
              previous: prev.number!,
              opens: signed((last.humanOpens ?? 0) - (prev.humanOpens ?? 0), 1),
              clicks: signed((last.clicks ?? 0) - (prev.clicks ?? 0), 1),
              recipients: signed((last.recipients ?? 0) - (prev.recipients ?? 0)),
            })}
          </p>
        ) : sent.length ? (
          <p className="k m-0 text-[12px]">{t(m.chartNote)}</p>
        ) : null}
      </div>
    </section>
  )
}

/** The digest's recent history, newest first: the audit log keeps the settings changes. */
export function HistoryCard({ d }: { d: Digest }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useDigestHistory(d.id, 5)
  const items = q.data?.items ?? []
  return (
    <section className="card" aria-labelledby="digest-history">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="digest-history">
            {t(m.history)}
          </h2>
          <span className="k">{t(m.historySub)}</span>
        </div>
        <Link to="/w/$workspace/settings/audit" params={{ workspace: slug }} search={{ category: 'digests' }} className="lnk text-[13px] font-semibold">
          {t(m.auditLog)}
        </Link>
      </div>
      {!q.data ? (
        <div className="pad flex flex-col gap-3">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} w="100%" h={30} />
          ))}
        </div>
      ) : items.length === 0 ? (
        <p className="pad k m-0">{t(m.historyEmpty)}</p>
      ) : (
        <ol className="m-0 flex list-none flex-col p-0">
          {items.map((h, i) => (
            <HistoryRow key={i} h={h} d={d} />
          ))}
        </ol>
      )}
    </section>
  )
}

function HistoryRow({ h, d }: { h: HistoryItem; d: Digest }) {
  const t = useT()
  let icon: IconName = 'arrowRight'
  let tone = 'bg-info-bg text-info-fg'
  let title = ''
  let sub = ''
  const next = d.next?.number ?? d.sentIssues + 1
  switch (h.kind) {
    case 'left_out':
      icon = 'more'
      tone = 'bg-neutral-bg text-neutral-fg'
      title = t(m.hLeftOut, { title: h.title ?? '' })
      sub =
        h.leftOutBy === 'category' && h.draftList
          ? t(m.hLeftOutCategoryDraft, { category: h.category ?? '', list: h.draftList })
          : h.leftOutBy === 'tag'
            ? t(m.hLeftOutTag)
            : h.leftOutBy === 'age'
              ? t(m.hLeftOutAge)
              : t(m.hLeftOutCategory, { category: h.category ?? '' })
      break
    case 'goes_in':
      title = t(m.hGoesIn, { title: h.title ?? '', number: next })
      sub =
        h.via === 'plugin'
          ? t(m.hGoesInPlugin, { time: h.published ? clock(h.published) : clock(h.at) })
          : t(m.hGoesInRead, { date: h.published ? shortDate(h.published) : shortDate(h.at), time: clock(h.at) })
      break
    case 'sent':
      icon = 'check'
      tone = 'bg-ok-bg text-ok-fg'
      title = t(m.hSent, { number: h.number ?? 0, count: count(h.recipients ?? 0) })
      sub = h.approvedBy ? t(m.hSentApproved, { name: h.approvedBy, time: h.approvedAt ? clock(h.approvedAt) : '', owner: h.owner ?? '' }) : t(m.hSentNoApproval, { owner: h.owner ?? '' })
      break
    case 'skipped':
    case 'not_sent':
      icon = 'close'
      tone = 'bg-neutral-bg text-neutral-fg'
      title = h.kind === 'skipped' ? t(m.hSkipped, { date: h.sendAt ? shortDate(h.sendAt) : shortDate(h.at) }) : t(m.hNotSent, { date: h.sendAt ? shortDate(h.sendAt) : shortDate(h.at) })
      sub = h.kind === 'skipped' ? t(m.skipReason, { reason: h.skipReason || 'other' }) : t(m.notSentReason)
      break
    case 'turned_on': {
      icon = 'check'
      tone = 'bg-ok-bg text-ok-fg'
      title = t(m.hTurnedOn, { name: h.actor ?? '' })
      const mode = h.mode || 'auto'
      if (h.builtBy && h.builtAt) {
        const sameDay = shortDate(h.builtAt) === shortDate(h.at)
        const hour = zonedParts(h.builtAt).hour
        sub = !sameDay
          ? t(m.hBuiltOn, { mode, name: h.builtBy, date: shortDate(h.builtAt) })
          : hour < 12
            ? t(m.hBuiltMorning, { mode, name: h.builtBy })
            : t(m.hBuiltAfternoon, { mode, name: h.builtBy })
      } else {
        sub = t(m.hTurnedOnMode, { mode })
      }
      break
    }
    case 'paused':
      icon = 'pause'
      tone = 'bg-warn-bg text-warn-fg'
      title = h.actor && h.pauseReason === 'person' ? t(m.hPaused, { name: h.actor }) : t(m.hPausedAuto)
      sub = h.note ? `“${h.note}”` : t(m.hPausedReason, { reason: h.pauseReason || 'person' })
      break
    case 'created':
      icon = 'plus'
      tone = 'bg-neutral-bg text-neutral-fg'
      title = t(m.hCreated, { name: h.actor ?? '' })
      sub = t(m.hCreatedMode, { mode: h.mode || 'auto' })
      break
  }
  // Posts by weekday and time ("Wed 07:31"), issues and turning on by date ("Fri 10 Oct").
  const when = h.kind === 'left_out' || h.kind === 'goes_in' || h.kind === 'paused' ? t(m.sendTimeValue, { day: String(zonedParts(h.at).weekday), time: clock(h.at) }) : shortDate(h.at)
  return (
    <li className="flex items-start gap-3 border-b border-line-2 px-[18px] py-3 last:border-b-0">
      <span className={cn('grid h-6 w-6 shrink-0 place-items-center rounded-full', tone)} aria-hidden="true">
        <Icon name={icon} size={12} strokeWidth={2.2} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="font-semibold">{title}</span>
        <span className="k">{sub}</span>
      </div>
      <time className="mono shrink-0 text-[12px] text-ink-3" dateTime={h.at} title={dateTime(h.at)}>
        {when}
      </time>
    </li>
  )
}
