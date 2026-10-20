import { Link } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { TableScroll } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, decimal, pct } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { OffBanner, ProgramHeader, type Period } from './ProgramHeader'
import { WeekBars } from './WeekBars'
import { m } from './messages'
import { mr } from './referrals.messages'
import { GROW, GrowPage, LoadError, ProgramNav, ReadOnlyNote, calendarDay, dayRange, mono, orNames, useReferralPrograms } from './shared'

type Results = Schemas['ReferralResults']

export interface ResultsSearch {
  program?: string
  period?: Period
  grouping?: 'day' | 'week'
}

/**
 * Referrals › Results (Referrals, ReferralsTablet, ReferralsPhone,
 * ReferralStates): the program header, the funnel, confirmed referrals over
 * time, channels, top referrers, rewards per milestone and quality.
 */
export function ResultsPage({ search, onSearch }: { search: ResultsSearch; onSearch: (s: ResultsSearch) => void }) {
  const t = useT()
  const { list, program, q } = useReferralPrograms(search.program)
  return (
    <GrowPage tab="referrals" documentTitle={t(m.docReferrals)}>
      {!list ? (
        q.isError ? (
          <LoadError error={q.error} title={t(mr.loadError)} onRetry={() => void q.refetch()} />
        ) : (
          <ResultsSkeleton />
        )
      ) : !program ? (
        <NoProgram helpers={list.helpers.edit} />
      ) : !program.startedAt && program.status === 'off' ? (
        <SettingUp program={program} />
      ) : (
        <Results programId={program.id} search={search} onSearch={onSearch} />
      )}
    </GrowPage>
  )
}

function NoProgram({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  return (
    <EmptyState
      title={t(mr.noneTitle)}
      action={
        canEdit ? (
          <ButtonLink to="/w/$workspace/grow/referrals/setup" params={{ workspace: slug }} variant="primary">
            {t(mr.noneAction)}
          </ButtonLink>
        ) : null
      }
    >
      <p className="m-0">{t(mr.noneText)}</p>
      {canEdit ? null : (
        <p className="k m-0 mt-2">
          <T msg={helpers.length ? mr.noneEditNeeds : mr.noneEditNeedsNoHelpers} values={{ helpers: orNames(helpers), mono }} />
        </p>
      )}
    </EmptyState>
  )
}

function SettingUp({ program }: { program: Schemas['ReferralProgramSummary'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  return (
    <EmptyState
      title={t(mr.resumeTitle, { list: program.listName })}
      action={
        canEdit ? (
          <ButtonLink to="/w/$workspace/grow/referrals/setup" params={{ workspace: slug }} search={{ program: program.id }} variant="primary">
            {t(mr.resumeAction)}
          </ButtonLink>
        ) : null
      }
    >
      {t(mr.resumeText, { step: program.setupStep })}
    </EmptyState>
  )
}

function Results({ programId, search, onSearch }: { programId: string; search: ResultsSearch; onSearch: (s: ResultsSearch) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const period = search.period ?? 'all'
  const grouping = search.grouping ?? 'week'
  const q = $api.useQuery(
    'get',
    `${GROW}/referrals/{program}/results`,
    { params: { path: { workspace: slug, program: programId }, query: { period, grouping } } },
    { placeholderData: (prev) => prev },
  )
  const r = q.data
  const canEdit = useCan('grow:edit')
  const canReview = useCan('grow:review')
  const readOnly = !canEdit && !canReview
  if (!r) {
    return (
      <div className="gw">
        <ProgramNav current="results" programId={search.program} />
        {q.isError ? (
          <LoadError
            error={q.error}
            title={t(mr.loadError)}
            onRetry={() => void q.refetch()}
            secondary={
              <ButtonLink to="/w/$workspace/grow" params={{ workspace: slug }}>
                {t(m.growOverview)}
              </ButtonLink>
            }
          />
        ) : (
          <ResultsSkeleton />
        )}
      </div>
    )
  }
  const p = r.program
  return (
    <div className="gw">
      <ProgramNav current="results" held={p.held} programId={search.program} />
      <OffBanner p={p} />
      {readOnly ? (
        <ReadOnlyNote>
          <T msg={p.helpers.edit.length ? mr.readOnlyText : mr.readOnlyTextNoHelpers} values={{ helpers: orNames(p.helpers.edit), mono }} />
        </ReadOnlyNote>
      ) : null}
      <ProgramHeader p={p} period={period} onPeriod={(v) => onSearch({ ...search, period: v === 'all' ? undefined : v })} helpers={p.helpers.edit} />
      {!r.enough ? (
        <TooLittle r={r} programId={search.program} />
      ) : (
        <>
          <Funnel r={r} programId={search.program} />
          <div className="gw-g2 gw-st">
            <OverTime r={r} grouping={grouping} onGrouping={(g) => onSearch({ ...search, grouping: g === 'week' ? undefined : g })} />
            <Channels r={r} />
          </div>
          <TopReferrers r={r} programId={search.program} />
          <div className="gw-g2 gw-st">
            <Rewards r={r} programId={search.program} />
            <Quality r={r} />
          </div>
        </>
      )}
    </div>
  )
}

function periodName(t: ReturnType<typeof useT>, p: string): string {
  return p === '7d' ? t(mr.period7) : p === '30d' ? t(mr.period30) : t(mr.periodAll)
}

function periodLower(t: ReturnType<typeof useT>, p: string): string {
  return p === '7d' ? t(mr.periodLower7) : p === '30d' ? t(mr.periodLower30) : t(mr.periodLowerAll)
}

/** Not enough to read: "Results show after 20 confirmed referrals". */
function TooLittle({ r, programId }: { r: Results; programId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const f = r.funnel
  const days = Math.max(1, Math.round((new Date(r.to).getTime() - new Date(r.from).getTime()) / 86_400_000))
  const search = programId ? { program: programId } : {}
  return (
    <section className="card flex flex-col gap-3 px-7 py-8">
      <h2 className="m-0 text-[18px] font-bold">{t(mr.littleTitle)}</h2>
      <span className="max-w-[720px] text-ink-2">{t(mr.littleText, { days, visits: f.visits, signups: f.signups, confirmed: count(f.confirmed) })}</span>
      <div className="flex max-w-[420px] flex-col gap-1.5">
        <div className="gw-bar" style={{ height: 10 }}>
          <i style={{ width: `${Math.min(100, (f.confirmed / 20) * 100)}%`, background: 'var(--accent)' }} />
        </div>
        <span className="k">{t(mr.littleProgress, { count: f.confirmed })}</span>
      </div>
      <span className="text-ink-2">
        <T
          msg={mr.littleLinks}
          values={{
            referrers: (c) => (
              <Link to="/w/$workspace/grow/referrals/referrers" params={{ workspace: slug }} search={search}>
                {c}
              </Link>
            ),
            review: (c) => (
              <Link to="/w/$workspace/grow/referrals/review" params={{ workspace: slug }} search={search}>
                {c}
              </Link>
            ),
          }}
        />
      </span>
    </section>
  )
}

function reasonText(t: ReturnType<typeof useT>, r: Schemas['ReferralReasonCount']): string {
  const n = { count: count(r.count) }
  switch (r.reason) {
    case 'self':
      return t(mr.reasonSelf, n)
    case 'same_ip':
      return t(mr.reasonSameIp, n)
    case 'disposable':
      return t(mr.reasonDisposable, n)
    case 'left':
      return r.leftKind === 'hard_bounce' ? t(mr.reasonLeftBounce, n) : r.leftKind === 'complaint' ? t(mr.reasonLeftComplaint, n) : t(mr.reasonLeft, n)
    case 'already':
      return t(mr.reasonAlready, n)
    case 'never_confirmed':
      return t(mr.reasonNever, n)
    case 'reviewer':
      return t(mr.reasonReviewer, n)
  }
  return String(r.count)
}

/** Visits → signups → confirmed → counted, with what branches off. */
function Funnel({ r, programId }: { r: Results; programId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const f = r.funnel
  const p = r.program
  const hold = p.counting.holdDays
  const caption = hold
    ? t(mr.funnelCaption, { period: periodName(t, r.period), from: dayMonthDate(r.from), to: dayMonthDate(r.to), days: hold })
    : t(mr.funnelCaptionNoHold, { period: periodName(t, r.period), from: dayMonthDate(r.from), to: dayMonthDate(r.to) })
  const arrow = (
    <span className="fa" aria-hidden="true">
      <Icon name="chevronRight" size={14} />
    </span>
  )
  return (
    <section className="card">
      <CardSection title={t(mr.funnelTitle)} caption={caption} />
      <div className="gw-fun" role="list" aria-label={t(mr.funnelLabel)}>
        <div role="listitem">
          <span className="k">{t(mr.visits)}</span>
          <span className="fv">{count(f.visits)}</span>
          <span className="k text-ink-nav">{t(mr.visitsSub, { weeks: f.weeks })}</span>
          <div className="fb">
            <div>
              <span className="text-ink-2">{t(mr.visitsNote)}</span>
            </div>
          </div>
        </div>
        <div role="listitem">
          {arrow}
          <span className="k">{t(mr.signups)}</span>
          <span className="fv">{count(f.signups)}</span>
          <span className="k text-ink-nav">{t(mr.signupsSub, { pct: pct(f.visits ? (f.signups / f.visits) * 100 : 0, 1) })}</span>
          <div className="fb">
            <div>
              <b>{count(f.unconfirmed)}</b>
              <span>{t(mr.unconfirmed)}</span>
            </div>
          </div>
        </div>
        <div role="listitem">
          {arrow}
          <span className="k">{t(mr.confirmed)}</span>
          <span className="fv">{count(f.confirmed)}</span>
          <span className="k text-ink-nav">{t(mr.confirmedSub, { pct: pct(f.signups ? (f.confirmed / f.signups) * 100 : 0, 0) })}</span>
          <div className="fb">
            <div>
              <b>{count(f.inHold)}</b>
              <span>{f.holdEndsBy ? t(mr.inHold, { date: dayMonthDate(f.holdEndsBy) }) : t(mr.inHoldNoDate)}</span>
            </div>
            <div>
              <b className={f.held ? 'gw-held' : undefined}>{count(f.held)}</b>
              <span>
                {t(mr.heldReview)}
                {f.held ? (
                  <>
                    {' · '}
                    <Link to="/w/$workspace/grow/referrals/review" params={{ workspace: slug }} search={programId ? { program: programId } : {}}>
                      {t(mr.review)}
                    </Link>
                  </>
                ) : null}
              </span>
            </div>
            <div>
              <b>{count(f.notCounted)}</b>
              <span>{t(mr.notCounted)}</span>
            </div>
            {f.reasons.length ? (
              <ul>
                {f.reasons.map((x) => (
                  <li key={`${x.reason}-${x.leftKind}`}>{reasonText(t, x)}</li>
                ))}
              </ul>
            ) : null}
          </div>
        </div>
        <div role="listitem">
          {arrow}
          <span className="k">{t(mr.counted)}</span>
          <span className="fv">{count(f.counted)}</span>
          <span className="k text-ink-nav">{t(mr.countedSub, { pct: pct(f.confirmed ? (f.counted / f.confirmed) * 100 : 0, 0) })}</span>
          <div className="fb">
            <div>
              <b>{count(f.rewardsSent)}</b>
              <span>{t(mr.rewardsSent)}</span>
            </div>
            <div>
              <b>{count(f.referrers)}</b>
              <span>{t(mr.referrersCounted)}</span>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}

/** Confirmed referrals per week or day. */
function OverTime({ r, grouping, onGrouping }: { r: Results; grouping: 'day' | 'week'; onGrouping: (g: 'day' | 'week') => void }) {
  const t = useT()
  const bars = r.overTime.map((b) => ({
    start: b.start,
    end: b.end,
    partial: b.partial,
    sofar: b.partial ? t(m.daysSoFar, { days: b.days }) : undefined,
    values: [b.confirmed],
  }))
  const total = r.overTime.reduce((s, b) => s + b.confirmed, 0)
  const days = r.overTime.reduce((s, b) => s + b.days, 0)
  const items = r.overTime.map((b) => t(mr.overTimeItem, { when: grouping === 'day' ? dayMonthDate(calendarDay(b.start)) : dayRange(b.start, b.end), count: b.confirmed })).join(', ')
  return (
    <section className="card">
      <CardSection
        title={t(mr.overTimeTitle)}
        caption={grouping === 'day' ? t(mr.overTimeDay, { period: periodLower(t, r.period) }) : t(mr.overTimeWeek, { period: periodLower(t, r.period) })}
        actions={
          <Segmented
            label={t(mr.groupBy)}
            value={grouping}
            onChange={onGrouping}
            items={[
              { value: 'day', label: t(mr.groupDay) },
              { value: 'week', label: t(mr.groupWeek) },
            ]}
          />
        }
      />
      <div className="gw-pad flex flex-col gap-2.5">
        <WeekBars bars={bars} series={[{ key: 'confirmed', color: 'var(--chart-gmail)' }]} label={t(mr.overTimeLabel, { grouping, items })} day={grouping === 'day'} />
      </div>
      <div className="gw-foot">
        <span className="k">{t(mr.overTimeFoot, { count: total, days, perDay: decimal(days ? total / days : 0, 0) })}</span>
      </div>
    </section>
  )
}

const CHANNELS: Record<string, keyof typeof mr> = {
  copy: 'channelCopy',
  x: 'channelX',
  linkedin: 'channelLinkedin',
  whatsapp: 'channelWhatsapp',
  email: 'channelEmail',
  other: 'channelOther',
}

/** Visits and signups by where links were shared. */
function Channels({ r }: { r: Results }) {
  const t = useT()
  const max = Math.max(1, ...r.channels.map((c) => c.visits))
  const visits = r.channels.reduce((s, c) => s + c.visits, 0)
  const signups = r.channels.reduce((s, c) => s + c.signups, 0)
  return (
    <section className="card">
      <CardSection title={t(mr.channelsTitle)} caption={t(mr.channelsCaption, { period: periodLower(t, r.period) })} />
      <div className="gw-pad flex flex-col gap-[13px]">
        {r.channels.map((c) => (
          <div key={c.channel} className="flex flex-col gap-[5px]">
            <div className="flex flex-wrap justify-between gap-2.5 text-[13px]">
              <span className="font-semibold">{t(mr[CHANNELS[c.channel] ?? 'channelOther'])}</span>
              <span>
                <span className="gw-num font-semibold">{count(c.visits)}</span>{' '}
                <span className="k">
                  {c.signups
                    ? t(mr.channelLineRate, { visits: c.visits, signups: c.signups, pct: pct(c.visits ? (c.signups / c.visits) * 100 : 0, 1) })
                    : t(mr.channelLineNoRate, { visits: c.visits })}
                </span>
              </span>
            </div>
            <div className="gw-bar">
              <i style={{ width: `${(c.visits / max) * 100}%`, background: 'var(--chart-gmail)' }} />
            </div>
          </div>
        ))}
      </div>
      <div className="gw-foot">
        <span className="k">
          <T msg={mr.channelsFoot} values={{ signups, visits, mono }} />
        </span>
      </div>
    </section>
  )
}

export function MilestoneBadges({ list }: { list: number[] }) {
  const t = useT()
  return (
    <span className="gw-ms">
      {list.map((n) => (
        <Badge key={n} tone="neutral" title={t(mr.milestoneReachedTitle, { count: n })}>
          {n}
        </Badge>
      ))}
    </span>
  )
}

/** The top referrers, linking to their panels. */
function TopReferrers({ r, programId }: { r: Results; programId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const dash = <span className="k">{t(m.none)}</span>
  const num = (v: number) => (v ? count(v) : dash)
  const to = '/w/$workspace/grow/referrals/referrers' as const
  const panel = (linkId: string) => ({ ...(programId ? { program: programId } : {}), referrer: linkId })
  return (
    <section className="card">
      <CardSection title={t(mr.topTitle)} caption={t(mr.topCaption)} />
      {r.top.length ? (
        <>
          <div className="max-md:hidden">
            <TableScroll label={t(mr.topTitle)}>
              <table className="tbl gw-tbl">
                <thead>
                  <tr>
                    <th scope="col">{t(mr.colReferrer)}</th>
                    <th scope="col" className="r">
                      {t(mr.colCounted)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colInHold)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colHeld)}
                    </th>
                    <th scope="col">{t(mr.colMilestones)}</th>
                    <th scope="col">{t(mr.colLastReferral)}</th>
                  </tr>
                </thead>
                <tbody>
                  {r.top.map((x) => (
                    <tr key={x.linkId}>
                      <td>
                        <Link to={to} params={{ workspace: slug }} search={panel(x.linkId)} className="block text-ink no-underline">
                          <span className="flex min-w-0 items-center gap-2.5">
                            <Avatar name={x.name} size={28} color="muted" />
                            <span className="flex min-w-0 flex-col leading-[1.3]">
                              <span className="font-semibold">{x.name}</span>
                              {x.email ? <span className="k mono whitespace-nowrap">{x.email}</span> : null}
                            </span>
                          </span>
                        </Link>
                      </td>
                      <td className="r gw-num">{num(x.counted)}</td>
                      <td className="r gw-num">{num(x.inHold)}</td>
                      <td className="r gw-num">{x.held ? <span className="gw-held">{count(x.held)}</span> : dash}</td>
                      <td>
                        <MilestoneBadges list={x.milestones} />
                        {x.unclaimed ? <div className="k mt-[3px]">{t(mr.unclaimed, { reward: x.unclaimed })}</div> : null}
                      </td>
                      <td className="nw">{x.lastReferral ? dayMonthDate(x.lastReferral) : dash}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </TableScroll>
          </div>
          <div className="md:hidden">
            {r.top.map((x) => (
              <Link key={x.linkId} to={to} params={{ workspace: slug }} search={panel(x.linkId)} className="gw-prow no-underline">
                <Avatar name={x.name} size={28} color="muted" />
                <span className="tx">
                  <span className="font-semibold">{x.name}</span>
                  <span className="k">
                    {[
                      t(mr.topPhoneLine, { counted: count(x.counted) }),
                      x.inHold ? t(mr.topPhoneHold, { count: count(x.inHold) }) : null,
                      x.held ? <span key="h" className="gw-held">{t(mr.topPhoneHeld, { count: count(x.held) })}</span> : null,
                      x.lastReferral ? t(mr.topPhoneLast, { date: dayMonthDate(x.lastReferral) }) : null,
                    ]
                      .filter(Boolean)
                      .map((v, i) => (
                        <span key={i}>
                          {i ? ' · ' : ''}
                          {v}
                        </span>
                      ))}
                  </span>
                </span>
                <MilestoneBadges list={x.milestones} />
              </Link>
            ))}
          </div>
        </>
      ) : (
        <div className="gw-pad k">{t(mr.topEmpty)}</div>
      )}
      <div className="gw-foot">
        <span className="k">{t(mr.topFoot, { referrers: r.funnel.referrers, three: count(r.withThree), ten: count(r.withTen) })}</span>
        <ButtonLink to={to} params={{ workspace: slug }} search={programId ? { program: programId } : {}} size="sm">
          {t(mr.allReferrers)}
        </ButtonLink>
      </div>
    </section>
  )
}

function usedText(t: ReturnType<typeof useT>, r: Schemas['ReferralRewardRow']): string {
  switch (r.type) {
    case 'download':
      return t(mr.usedDownload, { count: count(r.used) })
    case 'coupon':
      return t(mr.usedCoupon, { count: count(r.used) })
    case 'manual':
      return t(mr.usedManual, { count: count(r.used) })
  }
  return t(mr.usedOther, { count: count(r.used) })
}

/** Rewards per milestone: reached, sent, used, waiting, failed. */
function Rewards({ r, programId }: { r: Results; programId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const dash = <span className="k">{t(m.none)}</span>
  const sent = r.rewards.reduce((s, x) => s + x.sent, 0)
  const waiting = r.rewards.reduce((s, x) => s + x.waiting, 0)
  const failed = r.rewards.reduce((s, x) => s + x.failed, 0)
  return (
    <section className="card">
      <CardSection title={t(mr.rewardsTitle)} caption={t(mr.rewardsCaption)} />
      {r.rewards.length ? (
        <>
          <div className="max-md:hidden">
            <TableScroll label={t(mr.rewardsTitle)}>
              <table className="tbl gw-tbl">
                <thead>
                  <tr>
                    <th scope="col">{t(mr.colReward)}</th>
                    <th scope="col" className="r">
                      {t(mr.colReached)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colSent)}
                    </th>
                    <th scope="col">{t(mr.colUsed)}</th>
                    <th scope="col" className="r">
                      {t(mr.colWaiting)}
                    </th>
                    <th scope="col" className="r">
                      {t(mr.colFailed)}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {r.rewards.map((x) => (
                    <tr key={x.milestoneId}>
                      <td>
                        <span className="flex flex-col gap-px">
                          <span className="gw-tt2">{x.name}</span>
                          <span className="k">{t(m.referralsN, { count: x.count })}</span>
                        </span>
                      </td>
                      <td className="r gw-num">{count(x.reached)}</td>
                      <td className="r gw-num">{count(x.sent)}</td>
                      <td className="nw">{usedText(t, x)}</td>
                      <td className="r gw-num">{x.waiting ? <span className="gw-held">{count(x.waiting)}</span> : dash}</td>
                      <td className="r gw-num">{x.failed ? <span className="text-bad-fg font-semibold">{count(x.failed)}</span> : dash}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </TableScroll>
          </div>
          <div className="md:hidden">
            {r.rewards.map((x) => (
              <div key={x.milestoneId} className="flex flex-col gap-0.5 border-t border-line-2 px-4 py-[11px]">
                <span className="gw-tt2">
                  {x.name} <span className="k font-medium">· {t(m.referralsN, { count: x.count })}</span>
                </span>
                <span className="k text-ink-nav">
                  {x.waiting || x.failed
                    ? t(mr.rewardsPhoneSome, { reached: count(x.reached), sent: count(x.sent), used: usedText(t, x), waiting: count(x.waiting), failed: count(x.failed) })
                    : t(mr.rewardsPhoneNone, { reached: count(x.reached), sent: count(x.sent), used: usedText(t, x) })}
                </span>
              </div>
            ))}
          </div>
          <div className="border-t border-line-2 px-[18px] py-3">
            <span className="k text-ink-2">{t(mr.rewardsWaitingNote)}</span>
          </div>
        </>
      ) : (
        <div className="gw-pad k">{t(mr.noMilestones)}</div>
      )}
      <div className="gw-foot">
        <span className="k">
          {waiting || failed ? t(mr.rewardsFootFailed, { sent: count(sent), waiting, failed: count(failed) }) : t(mr.rewardsFootNone, { sent: count(sent) })}
        </span>
        <Link to="/w/$workspace/grow/referrals/milestones" params={{ workspace: slug }} search={programId ? { program: programId } : {}} className="gw-lnk">
          {t(mr.milestonesLink)}
        </Link>
      </div>
    </section>
  )
}

/** Referred subscribers against the whole list and other new subscribers. */
function Quality({ r }: { r: Results }) {
  const t = useT()
  const q = r.quality
  const row = (label: string, value: number, text: string, color: string, bold: boolean, scale: number): ReactNode => (
    <div className="flex flex-nowrap items-center gap-2.5">
      <span className="k" style={{ width: 96, flex: 'none' }}>
        {label}
      </span>
      <div className="gw-bar flex-1">
        <i style={{ width: `${Math.min(100, (value / scale) * 100)}%`, background: color }} />
      </div>
      <span className="gw-num text-right text-[13px]" style={{ width: 120, flex: 'none', fontWeight: bold ? 600 : 400, color: bold ? undefined : 'var(--ink-nav)' }}>
        {text}
      </span>
    </div>
  )
  const cmp = (title: string, note: string, line: Schemas['ReferralQualityLine'], others: string, counts: boolean) => {
    const scale = Math.max(line.referred, line.others, 0.0001) * 1.18
    return (
      <div className="flex flex-col gap-1.5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-[13px] font-semibold">{title}</span>
          <span className="k">{note}</span>
        </div>
        {row(
          t(mr.referred),
          line.referred,
          counts ? t(mr.qualityValue, { pct: pct(line.referred, 1), n: count(line.referredN), of: count(line.referredOf) }) : pct(line.referred, 1),
          'var(--chart-gmail)',
          true,
          scale,
        )}
        {row(
          others,
          line.others,
          counts ? t(mr.qualityValue, { pct: pct(line.others, 1), n: count(line.othersN), of: count(line.othersOf) }) : pct(line.others, 1),
          'var(--chart-other)',
          false,
          scale,
        )}
      </div>
    )
  }
  return (
    <section className="card">
      <CardSection title={t(mr.qualityTitle)} caption={t(mr.qualityCaption)} />
      <div className="gw-pad flex flex-col gap-4">
        {q.enough ? (
          <>
            {q.campaign && q.opens.referredOf > 0 ? (
              cmp(t(mr.qualityOpens), q.campaign, q.opens, t(mr.wholeList), false)
            ) : (
              <div className="flex flex-col gap-1">
                <span className="text-[13px] font-semibold">{t(mr.qualityOpens)}</span>
                <span className="k">{q.campaign ? t(mr.qualityNoReferred, { campaign: q.campaign }) : t(mr.qualityNoIssue)}</span>
              </div>
            )}
            {cmp(t(mr.qualityUnsub), t(mr.qualitySameWeeks), q.unsubscribed, t(mr.otherSources), true)}
            {cmp(t(mr.qualityBounced), t(mr.qualitySameWeeks), q.bounced, t(mr.otherSources), true)}
          </>
        ) : (
          <span className="text-ink-2">{t(mr.qualityWaiting, { count: q.referred })}</span>
        )}
      </div>
      <div className="gw-foot">
        <span className="k">{q.enough ? t(mr.qualityEnough, { count: count(q.referred) }) : null}</span>
      </div>
    </section>
  )
}

function ResultsSkeleton() {
  const t = useT()
  return (
    <Loading label={t(mr.loadingLabel)}>
      <div className="gw">
        <Skeleton w="100%" h={74} r={10} />
        <Skeleton w="100%" h={220} r={10} />
        <div className="gw-g2">
          <Skeleton w="100%" h={300} r={10} />
          <Skeleton w="100%" h={300} r={10} />
        </div>
      </div>
    </Loading>
  )
}
