import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { TableScroll } from '@/components/ui/Table'
import { Segmented } from '@/components/ui/Tabs'
import { T, formatList, t as tt, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear, pct } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { MilestoneTrack } from './MilestoneTrack'
import { WeekBars, type BarDatum } from './WeekBars'
import { m, mo } from './messages'
import { mc } from './recommendations.messages'
import { GROW, GrowPage, LoadError, NewsletterTile, ReadOnlyNote, calendarDay, dayRange } from './shared'

type Overview = Schemas['GrowOverview']
type Need = Schemas['GrowNeed']
type Recs = Schemas['RecommendationsView']
type RecItem = Schemas['RecommendationItem']

const SERIES = [
  { key: 'other', color: 'var(--chart-other)' },
  { key: 'referrals', color: 'var(--chart-gmail)' },
  { key: 'partners', color: 'var(--chart-outlook)' },
]

/**
 * Grow › Overview (GrowOverview, GrowStates): the last 30 days' numbers,
 * what needs a person, new subscribers by source, and the program and
 * recommendations cards, each loading on its own.
 */
export function GrowOverviewPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/overview`, { params: { path: { workspace: slug } } })
  const o = q.data
  return (
    <GrowPage tab="overview" documentTitle={t(m.docOverview)}>
      {!o ? (
        q.isError ? (
          <LoadError error={q.error} title={t(mo.programError)} onRetry={() => void q.refetch()} />
        ) : (
          <OverviewSkeleton />
        )
      ) : !o.programId && o.recommends === 0 && o.recommendedBy === 0 ? (
        <FirstRun o={o} />
      ) : (
        <OverviewBody o={o} />
      )}
    </GrowPage>
  )
}

function useReadOnly() {
  const edit = useCan('grow:edit')
  const review = useCan('grow:review')
  const connect = useCan('grow:connect')
  return !edit && !review && !connect
}

function OverviewBody({ o }: { o: Overview }) {
  const t = useT()
  const readOnly = useReadOnly()
  return (
    <div className="gw">
      {readOnly ? (
        <ReadOnlyNote>
          {t(o.helpers.edit.length && o.helpers.connect.length ? mo.readOnlyText : mo.readOnlyTextNoHelpers, {
            editHelpers: formatList(o.helpers.edit, 'disjunction'),
            connectHelpers: formatList(o.helpers.connect, 'disjunction'),
          })}
        </ReadOnlyNote>
      ) : null}
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="k">
          {o.listName
            ? t(mo.period, { from: dayMonthDate(calendarDay(o.from)), to: dayMonthYear(calendarDay(o.to)), list: o.listName })
            : t(mo.periodNoList, { from: dayMonthDate(calendarDay(o.from)), to: dayMonthYear(calendarDay(o.to)) })}
        </span>
        <span className="k">{t(mo.updated, { time: clock(o.updatedAt) })}</span>
      </div>
      <Tiles o={o} />
      <NeedsYou o={o} readOnly={readOnly} />
      {o.listName ? (
        <div className="gw-g32 gw-st">
          <ChartCard o={o} />
          {o.programId ? <ProgramCard programId={o.programId} /> : <NoProgramCard o={o} />}
        </div>
      ) : o.programId ? (
        <ProgramCard programId={o.programId} />
      ) : (
        <NoProgramCard o={o} />
      )}
      <RecommendationsCard />
    </div>
  )
}

function Tiles({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const tl = o.tiles
  const share = tl.newSubscribers ? (tl.fromGrow / tl.newSubscribers) * 100 : 0
  return (
    <div className="gw-stats">
      <Stat label={t(mo.tileFromGrow)} value={count(tl.fromGrow)} sub={t(mo.tileFromGrowSub, { pct: pct(share, 0), count: tl.newSubscribers })} />
      {o.programId ? (
        <Stat
          label={t(mo.tileConversion)}
          value={tl.visits ? pct(tl.conversion, 1) : '–'}
          sub={tl.visits ? t(mo.tileConversionSub, { visits: tl.visits, confirmed: count(tl.confirmed) }) : t(mo.tileConversionNone)}
        />
      ) : null}
      <Stat
        label={t(mo.tilePartners)}
        value={count(tl.partnerSignups)}
        sub={tl.partners.length ? t(mo.tilePartnersSub, { partners: formatList(tl.partners) }) : t(mo.tilePartnersNone)}
      />
      {o.programId ? (
        <Stat
          label={t(mo.tileRewards)}
          value={count(tl.rewardsSent)}
          sub={tl.rewards.length ? tl.rewards.map((r) => t(mo.tileRewardsItem, { count: count(r.count), reward: r.name })).join(' · ') : t(mo.tileRewardsNone)}
        />
      ) : (
        <div className="card gw-stat">
          <span className="k">{t(mo.tileProgram)}</span>
          <span className="v">{t(mo.tileProgramNotSet)}</span>
          <Link to="/w/$workspace/grow/referrals/setup" params={{ workspace: slug }} className="s">
            {t(mo.tileProgramStart)}
          </Link>
        </div>
      )}
    </div>
  )
}

function Stat({ label, value, sub }: { label: string; value: ReactNode; sub: ReactNode }) {
  return (
    <div className="card gw-stat">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      <span className="s">{sub}</span>
    </div>
  )
}

const needLook: Record<string, { icon: IconName; bg: string; fg: string }> = {
  held: { icon: 'clock', bg: 'var(--warn-bg)', fg: 'var(--warn-fg)' },
  partner_invalid: { icon: 'link', bg: 'var(--bad-bg)', fg: 'var(--bad-fg)' },
  partner_unreachable: { icon: 'link', bg: 'var(--warn-bg)', fg: 'var(--warn-fg)' },
  unclaimed: { icon: 'mail', bg: 'var(--info-bg)', fg: 'var(--info-fg)' },
  codes_out: { icon: 'tag', bg: 'var(--bad-bg)', fg: 'var(--bad-fg)' },
  codes_low: { icon: 'tag', bg: 'var(--warn-bg)', fg: 'var(--warn-fg)' },
  webhook_failing: { icon: 'webhook', bg: 'var(--bad-bg)', fg: 'var(--bad-fg)' },
  to_fulfill: { icon: 'send', bg: 'var(--info-bg)', fg: 'var(--info-fg)' },
  stripe_missing: { icon: 'key', bg: 'var(--warn-bg)', fg: 'var(--warn-fg)' },
  file_missing: { icon: 'upload', bg: 'var(--bad-bg)', fg: 'var(--bad-fg)' },
}

/** "Needs you": each item with its action, newest problems first. */
function NeedsYou({ o, readOnly }: { o: Overview; readOnly: boolean }) {
  const t = useT()
  const items = o.needsYou
  if (!items.length) {
    return (
      <section className="card">
        <CardSection title={t(mo.needsTitle, { count: 0 })} caption={t(mo.needsChecked)} />
        <div className="gw-pad flex items-start gap-3">
          <span className="gw-ti" style={{ background: 'var(--ok-bg)', color: 'var(--ok-fg)' }} aria-hidden="true">
            <Icon name="check" size={14} strokeWidth={2.2} />
          </span>
          <div className="flex flex-col gap-0.5">
            <span className="font-semibold">{t(mo.needsNoneTitle)}</span>
            <span className="k">{t(mo.needsNoneText)}</span>
          </div>
        </div>
      </section>
    )
  }
  return (
    <section className="card">
      <CardSection title={t(mo.needsTitle, { count: items.length })} caption={t(mo.needsCaption)} />
      <div>
        {items.map((n, i) => (
          <NeedRow key={`${n.kind}-${i}`} n={n} readOnly={readOnly} workspace={o.listName} />
        ))}
      </div>
    </section>
  )
}

function NeedRow({ n, readOnly }: { n: Need; readOnly: boolean; workspace: string }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canReview = useCan('grow:review')
  const canEdit = useCan('grow:edit')
  const canConnect = useCan('grow:connect')
  const look = needLook[n.kind] ?? needLook.held!
  const params = { workspace: slug }
  const since = n.since ? `${dayMonthDate(n.since)} ${clock(n.since)}` : ''
  let title: ReactNode = null
  let text: ReactNode = null
  let action: ReactNode = null
  const btn = (label: string, to: string, search: Record<string, unknown> = {}, primary = false, allowed = true) => (
    <ButtonLink to={to as never} params={params as never} search={search as never} size="sm" variant={primary && allowed && !readOnly ? 'primary' : undefined}>
      {allowed && !readOnly ? label : t(mo.actView)}
    </ButtonLink>
  )
  switch (n.kind) {
    case 'held': {
      const shown = n.people.slice(0, 3)
      const people = formatList(shown.map((p) => t(mo.heldPerson, { name: p.name, count: p.count })))
      title = (
        <span>
          <b>{t(mo.heldTitle, { count: n.count })}</b> {people}
          {n.people.length > shown.length ? ` ${t(mo.heldMore, { count: n.people.length - shown.length })}` : ''}
        </span>
      )
      text = n.since ? t(mo.heldText, { date: dayMonthDate(n.since) }) : t(mo.heldTextNoDate)
      action = btn(t(mo.actReview), '/w/$workspace/grow/referrals/review', {}, true, canReview)
      break
    }
    case 'partner_invalid':
      title = (
        <span>
          <b>{t(mo.partnerInvalidTitle, { partner: n.partner, date: since })}</b> {t(mo.partnerInvalidHidden)}
        </span>
      )
      text =
        n.reason === 'ended'
          ? t(mo.partnerInvalidEnded)
          : n.reason === 'unknown_link'
            ? t(mo.partnerInvalidUnknown)
            : t(mo.partnerInvalidSignature, { workspace: me.workspace.name })
      action = btn(t(mo.actFix), '/w/$workspace/grow/recommendations', {}, false, canConnect)
      break
    case 'partner_unreachable':
      title = <b>{t(mo.partnerUnreachableTitle, { partner: n.partner, date: since })}</b>
      text = t(mo.partnerUnreachableText)
      action = btn(t(mo.actOpen), '/w/$workspace/grow/recommendations', {}, false, canConnect)
      break
    case 'unclaimed': {
      const people = formatList(n.people.map((p) => p.name))
      const dates = formatList(n.expires.map((d) => dayMonthDate(d)))
      title = (
        <span>
          <b>{t(mo.unclaimedTitle, { count: n.count, reward: n.reward })}</b> {t(mo.unclaimedPeople, { people })}
          {n.expires.length ? ` ${t(mo.unclaimedExpire, { count: n.expires.length, dates })}` : ''}
        </span>
      )
      text = t(mo.unclaimedText)
      action = btn(t(mo.actResend), '/w/$workspace/grow/referrals/referrers', { milestone: n.milestone }, false, canReview)
      break
    }
    case 'codes_out':
      title = (
        <span>
          <b>{t(mo.codesOutTitle, { reward: n.reward })}</b> {t(mo.codesOutWaiting, { count: n.count })}
        </span>
      )
      text = t(mo.codesOutText)
      action = btn(t(mo.actUpload), '/w/$workspace/grow/referrals/milestones', { edit: n.milestoneId }, true, canEdit)
      break
    case 'codes_low':
      title = <b>{t(mo.codesLowTitle, { reward: n.reward, count: n.codesLeft })}</b>
      text = t(mo.codesLowText)
      action = btn(t(mo.actUpload), '/w/$workspace/grow/referrals/milestones', { edit: n.milestoneId }, false, canEdit)
      break
    case 'webhook_failing':
      title = <b>{t(mo.webhookTitle, { reward: n.reward })}</b>
      text = t(mo.webhookText, { count: n.count })
      action = btn(t(mo.actFix), '/w/$workspace/grow/referrals/milestones', { edit: n.milestoneId }, false, canEdit)
      break
    case 'to_fulfill':
      title = <b>{t(mo.fulfillTitle, { count: n.count, reward: n.reward })}</b>
      text = t(mo.fulfillText)
      action = btn(t(mo.actFulfill), '/w/$workspace/grow/referrals/milestones', {}, false, canReview)
      break
    case 'stripe_missing':
      title = <b>{t(mo.stripeTitle, { reward: n.reward })}</b>
      text = t(mo.stripeText, { count: n.count })
      action = btn(t(mo.actFix), '/w/$workspace/grow/referrals/milestones', { edit: n.milestoneId }, false, canEdit)
      break
    case 'file_missing':
      title = <b>{t(mo.fileTitle, { reward: n.reward })}</b>
      text = t(mo.fileText)
      action = btn(t(mo.actFix), '/w/$workspace/grow/referrals/milestones', { edit: n.milestoneId }, false, canEdit)
      break
  }
  return (
    <div className="gw-ny">
      <span className="gw-ti" style={{ background: look.bg, color: look.fg }} aria-hidden="true">
        <Icon name={look.icon} size={13} strokeWidth={2} />
      </span>
      <div className="tx">
        {title}
        <span className="k">{text}</span>
      </div>
      <div className="act">{action}</div>
    </div>
  )
}

/** New subscribers per week by source, as a chart or a table. */
function ChartCard({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [view, setView] = useState<'chart' | 'table'>('chart')
  const names = [t(mo.seriesOther), t(mo.seriesReferrals), t(mo.seriesPartners)]
  const totals = [0, 0, 0]
  const bars: BarDatum[] = o.weeks.map((w) => {
    const values = [w.other, w.referrals, w.partners]
    values.forEach((v, i) => (totals[i] = (totals[i] ?? 0) + v))
    return { start: w.start, end: w.end, partial: w.partial, sofar: w.partial ? t(m.daysSoFar, { days: w.days }) : undefined, values }
  })
  const visible = o.programId ? [0, 1, 2] : [0, 2]
  const label = t(mo.chartLabel, { list: o.listName })
  const share = o.tiles.newSubscribers ? (o.tiles.fromGrow / o.tiles.newSubscribers) * 100 : 0
  return (
    <section className="card">
      <CardSection
        title={t(mo.chartTitle, { list: o.listName })}
        caption={t(mo.chartCaption)}
        actions={
          <Segmented
            label={t(mo.viewAs)}
            value={view}
            onChange={setView}
            items={[
              { value: 'chart', label: t(mo.viewChart) },
              { value: 'table', label: t(mo.viewTable) },
            ]}
          />
        }
      />
      <div className="gw-pad flex flex-col gap-3">
        <div className="gw-cleg">
          {visible.map((i) => (
            <span key={i}>
              <i style={{ background: SERIES[i]!.color }} />
              {t(mo.legendItem, { series: names[i]!, count: count(totals[i]) })}
            </span>
          ))}
        </div>
        {view === 'chart' ? (
          <WeekBars
            bars={bars}
            series={SERIES}
            label={`${label}: ${bars.map((b) => t(mo.chartWeekLabel, { week: dayRange(b.start, b.end), other: b.values[0]!, referrals: b.values[1]!, partners: b.values[2]! })).join('; ')}`}
            marker={o.programStart ? { day: o.programStart, text: t(mo.programStarted, { date: dayMonthDate(o.programStart) }), short: t(mo.programStartedShort, { date: dayMonthDate(o.programStart) }) } : undefined}
          />
        ) : (
          <TableScroll label={label}>
            <table className="tbl gw-tbl">
              <thead>
                <tr>
                  <th scope="col">{t(mo.colWeek)}</th>
                  {visible.map((i) => (
                    <th key={i} scope="col" className="r">
                      {names[i]}
                    </th>
                  ))}
                  <th scope="col" className="r">
                    {t(mo.colTotal)}
                  </th>
                </tr>
              </thead>
              <tbody>
                {bars.map((b) => (
                  <tr key={b.start}>
                    <td className="nw">
                      {dayRange(b.start, b.end)}
                      {b.sofar ? <span className="k"> · {b.sofar}</span> : null}
                    </td>
                    {visible.map((i) => (
                      <td key={i} className="r gw-num">
                        {count(b.values[i])}
                      </td>
                    ))}
                    <td className="r gw-num font-semibold">{count(b.values.reduce((s, v) => s + v, 0))}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </TableScroll>
        )}
        <span className="text-[13px] text-ink-nav">
          <T msg={mo.shareLine} values={{ part: count(o.tiles.fromGrow), total: count(o.tiles.newSubscribers), pct: pct(share, 0), b: (c) => <b className="font-semibold">{c}</b> }} />
        </span>
      </div>
      <div className="gw-foot">
        <span className="k">{t(mo.chartFoot)}</span>
        <Link to="/w/$workspace/audience/growth" params={{ workspace: slug }} search={o.listId ? { list: o.listId } : {}} className="gw-lnk">
          {t(mo.bySource)}
          <Icon name="arrowRight" size={14} />
        </Link>
      </div>
    </section>
  )
}

/** The referral program card: the milestone track, the top 3 and the counts. */
function ProgramCard({ programId }: { programId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/referrals/{program}/results`, { params: { path: { workspace: slug, program: programId }, query: { period: 'all' } } })
  const r = q.data
  if (!r) {
    return (
      <section className="card">
        <CardSection title={t(mo.programTitle)} />
        <div className="gw-pad">
          {q.isError ? (
            <LoadError error={q.error} title={t(mo.programError)} onRetry={() => void q.refetch()} />
          ) : (
            <Loading label={t(mo.loadingProgram)}>
              <div className="flex flex-col gap-3">
                <Skeleton w="100%" h={60} />
                <Skeleton w="80%" h={14} />
                <Skeleton w="70%" h={14} />
              </div>
            </Loading>
          )}
        </div>
      </section>
    )
  }
  const p = r.program
  const f = r.funnel
  const on = p.status === 'on'
  const ladder = r.rewards
  const top = r.top.filter((x) => x.counted > 0).slice(0, 3)
  return (
    <section className="card">
      <CardSection
        title={t(mo.programTitle)}
        caption={t(mo.programCaption, { list: p.listName, count: on ? p.links : p.subscribers })}
        actions={
          <Badge tone={on ? 'ok' : 'neutral'} dot>
            {on ? t(m.on) : t(m.off)}
          </Badge>
        }
      />
      <div className="gw-pad flex flex-col gap-[18px]">
        {ladder.length ? (
          <div className="flex flex-col gap-3">
            <h3 className="gw-hd3">{t(mo.milestonesHd)}</h3>
            <MilestoneTrack
              label={t(mo.milestonesLabel, {
                items: formatList(ladder.map((x) => t(mo.milestonesLabelItem, { referrers: x.reached, count: x.count }))),
              })}
              points={ladder.map((x) => ({ count: x.count, title: t(m.referralsN, { count: x.count }), mid: x.name, sub: t(m.referrersN, { count: x.reached }) }))}
            />
          </div>
        ) : null}
        <div className="flex flex-col gap-0.5">
          <div className="flex items-center justify-between gap-2">
            <h3 className="gw-hd3">{t(mo.topHd)}</h3>
            <span className="k">{t(mo.topCounted)}</span>
          </div>
          {top.length ? (
            top.map((x, i) => (
              <div className="gw-lrow" key={x.linkId}>
                <span className="k gw-num" style={{ width: 14, textAlign: 'right' }}>
                  {i + 1}
                </span>
                <Avatar name={x.name} size={28} color="muted" />
                <span className="tx">
                  <span className="gw-tt2">{x.name}</span>
                  <span className="k">{x.milestones.length ? t(mo.topMilestones, { list: x.milestones.join(' · ') }) : t(mo.topNoMilestone)}</span>
                </span>
                <span className="gw-num text-[15px] font-bold">{count(x.counted)}</span>
              </div>
            ))
          ) : (
            <span className="k">{t(mo.topNone)}</span>
          )}
        </div>
        <div className="flex flex-wrap gap-x-3.5 gap-y-1.5 text-[13px] text-ink-nav">
          <span>
            <T msg={mo.countsCounted} values={{ count: count(f.counted), b: (c) => <b>{c}</b> }} />
          </span>
          <span>
            <T msg={mo.countsHold} values={{ count: count(f.inHold), b: (c) => <b>{c}</b> }} />
          </span>
          <span>
            <T msg={mo.countsHeld} values={{ count: count(f.held), held: (c) => <b className={f.held ? 'gw-held' : undefined}>{c}</b> }} />
          </span>
          <span>
            <T msg={mo.countsNotCounted} values={{ count: count(f.notCounted), b: (c) => <b>{c}</b> }} />
          </span>
        </div>
      </div>
      <div className="gw-foot">
        <span className="k">
          {on && p.startedAt ? t(mo.programFoot, { date: dayMonthDate(p.startedAt), list: p.listName }) : t(mo.programFootOff, { list: p.listName })}
        </span>
        <ButtonLink to="/w/$workspace/grow/referrals" params={{ workspace: slug }} size="sm">
          {t(mo.openReferrals)}
        </ButtonLink>
      </div>
    </section>
  )
}

/** No program yet (recommendations only): the five-step guide and the suggested milestones. */
function NoProgramCard({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  return (
    <section className="card">
      <CardSection
        title={t(mo.programTitle)}
        caption={t(mo.programCaptionNone)}
        actions={
          <Badge tone="neutral" dot>
            {t(mo.notSetUp)}
          </Badge>
        }
      />
      <div className="gw-pad flex flex-col gap-4">
        <span className="text-ink-2">{o.listName ? t(mo.noProgramText, { list: o.listName }) : t(mo.noProgramTextNoList)}</span>
        <div>
          {canEdit ? (
            <ButtonLink to="/w/$workspace/grow/referrals/setup" params={{ workspace: slug }} variant="primary">
              {t(mo.firstReferralsTitle)}
            </ButtonLink>
          ) : null}
        </div>
        <div className="flex flex-col gap-3">
          <h3 className="gw-hd3">{t(mo.suggested)}</h3>
          <MilestoneTrack
            label={t(mo.suggestedLabel)}
            points={[
              { count: 1, title: t(m.referralsN, { count: 1 }), sub: t(mo.suggest1) },
              { count: 3, title: t(m.referralsN, { count: 3 }), sub: t(mo.suggest3) },
              { count: 10, title: t(m.referralsN, { count: 10 }), sub: t(mo.suggest10) },
            ]}
          />
        </div>
      </div>
    </section>
  )
}

/** The state of a recommendation as a badge. */
export function recBadge(it: Pick<RecItem, 'state'>): { tone: Tone; label: string } {
  switch (it.state) {
    case 'connected':
      return { tone: 'ok', label: mcText('stateConnected') }
    case 'showing':
      return { tone: 'ok', label: mcText('stateShowing') }
    case 'paused':
      return { tone: 'neutral', label: mcText('statePaused') }
    case 'partner_paused':
      return { tone: 'warn', label: mcText('statePartnerPaused') }
    case 'invalid':
      return { tone: 'bad', label: mcText('stateInvalid') }
    case 'limit':
      return { tone: 'warn', label: mcText('stateLimit') }
    case 'unreachable':
      return { tone: 'warn', label: mcText('stateUnreachable') }
    case 'ended':
      return { tone: 'neutral', label: mcText('stateEnded') }
  }
  return { tone: 'neutral', label: it.state }
}

function mcText(k: keyof typeof mc): string {
  return tt(mc[k])
}

/** "Partner on ReputeCast · 2,322 shown · 168 requests · 119 confirmed". */
export function recLine(it: RecItem): string {
  const since = it.since ? `${dayMonthDate(it.since)} ${clock(it.since)}` : ''
  if (it.kind === 'link') {
    if (it.state === 'paused') return tt(mc.lineLinkPaused, { date: since })
    return tt(mc.lineLink, { shown: count(it.shown), clicks: it.clicks })
  }
  switch (it.state) {
    case 'invalid':
      return tt(mc.linePartnerHidden, { date: since })
    case 'paused':
    case 'partner_paused':
      return tt(mc.linePartnerPaused, { date: since })
    case 'limit':
      return tt(mc.linePartnerLimit)
    case 'unreachable':
      return tt(mc.linePartnerUnreachable, { date: since })
  }
  return tt(mc.linePartner, { shown: count(it.shown), requests: it.requests, confirmed: count(it.confirmed) })
}

/** The recommendations card: what you recommend, who recommends you and your profile. */
function RecommendationsCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${GROW}/recommendations`, { params: { path: { workspace: slug } } })
  const v: Recs | undefined = q.data
  const status = q.error instanceof ApiError ? q.error.status : undefined
  return (
    <section className="card">
      <CardSection title={t(mo.recsTitle)} caption={t(mo.recsCaption)} />
      {!v ? (
        <div className="gw-pad">
          {q.isError ? (
            <LoadError error={q.error} title={t(mo.recsError)} onRetry={() => void q.refetch()}>
              {status ? t(mo.recsErrorText, { status }) : t(mo.recsErrorTextNoStatus)}
            </LoadError>
          ) : (
            <Loading label={t(mo.loadingRecs)}>
              <div className="gw-g2">
                <Skeleton w="100%" h={120} />
                <Skeleton w="100%" h={120} />
              </div>
            </Loading>
          )}
        </div>
      ) : (
        <div className="gw-pad">
          <div className="gw-g2" style={{ gap: '16px 28px' }}>
            <div className="flex min-w-0 flex-col gap-1">
              <h3 className="gw-hd3">{t(mo.youRecommend, { count: v.items.length })}</h3>
              <div>
                {v.items.length ? (
                  v.items.map((it) => {
                    const b = recBadge(it)
                    return (
                      <div className="gw-lrow" key={it.id}>
                        <NewsletterTile name={it.name} initials={it.initials} logo={it.logo?.url} />
                        <span className="tx">
                          <span className="gw-tt2">{it.name}</span>
                          <span className="k">{recLine(it)}</span>
                        </span>
                        <Badge tone={b.tone} dot>
                          {b.label}
                        </Badge>
                      </div>
                    )
                  })
                ) : (
                  <span className="k">{t(mo.recNone)}</span>
                )}
              </div>
            </div>
            <div className="flex min-w-0 flex-col gap-1">
              <h3 className="gw-hd3">{t(mo.recommendingYou, { count: v.recommendingYou.length })}</h3>
              {v.recommendingYou.length ? (
                v.recommendingYou.map((r) => {
                  const b = recBadge(r)
                  const rate = r.requests ? pct((r.confirmed / r.requests) * 100, 0) : '–'
                  return (
                    <div className="gw-lrow" key={r.linkId}>
                      <NewsletterTile name={r.name} />
                      <span className="tx">
                        <span className="gw-tt2">{r.name}</span>
                        <span className="k">
                          {r.state === 'paused'
                            ? t(mc.lineYouPaused, { date: dayMonthDate(r.since), requests: r.requests, confirmed: count(r.confirmed) })
                            : t(mc.lineYou, { date: dayMonthDate(r.since), requests: r.requests, confirmed: count(r.confirmed), rate, limit: count(r.dailyLimit) })}
                        </span>
                      </span>
                      <Badge tone={b.tone} dot>
                        {b.label}
                      </Badge>
                    </div>
                  )
                })
              ) : (
                <span className="k">{t(mo.recNone)}</span>
              )}
              {v.profile.name ? (
                <>
                  <h3 className="gw-hd3" style={{ marginTop: 12 }}>
                    {t(mo.yourProfile)}
                  </h3>
                  <div className="gw-lrow">
                    <NewsletterTile name={v.profile.name} initials={v.profile.initials} logo={v.profile.logo?.url} me />
                    <span className="tx">
                      <span className="gw-tt2">{v.profile.name}</span>
                      <span className="k">
                        {v.profile.description
                          ? t(mo.profileLine, { description: v.profile.description, list: v.profile.listName })
                          : t(mo.profileLineNoDescription, { list: v.profile.listName })}
                      </span>
                    </span>
                  </div>
                </>
              ) : null}
            </div>
          </div>
        </div>
      )}
      <div className="gw-foot">
        <span className="k">{v ? recsWhere(v) : ''}</span>
        <ButtonLink to="/w/$workspace/grow/recommendations" params={{ workspace: slug }} size="sm">
          {t(mo.openRecommendations)}
        </ButtonLink>
      </div>
    </section>
  )
}

function recsWhere(v: Recs): string {
  const where = formatList(v.blocks.map((b) => (b.automation ? `${b.automation} · ${b.name}` : b.name)))
  if (v.confirmPage && v.blocks.length) return tt(mo.recsFoot, { where })
  if (v.confirmPage) return tt(mo.recsFootConfirm)
  if (v.blocks.length) return tt(mo.recsFootBlock, { where })
  return tt(mo.recsFootNowhere)
}

/** First run: nothing set up yet. */
function FirstRun({ o }: { o: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('grow:edit')
  return (
    <div className="gw">
      <section className="card flex flex-col gap-5 px-7 py-8">
        <div className="flex flex-col gap-1.5">
          <h2 className="m-0 text-[22px] font-bold tracking-[-0.02em]">{t(mo.firstTitle)}</h2>
          <span className="max-w-[760px] text-ink-2">{o.listName ? t(mo.firstLead, { list: o.listName }) : t(mo.firstLeadNoList)}</span>
        </div>
        <div className="gw-g2">
          <FirstCard
            icon="grow"
            title={t(mo.firstReferralsTitle)}
            text={t(mo.firstReferralsText)}
            action={
              canEdit ? (
                <ButtonLink to="/w/$workspace/grow/referrals/setup" params={{ workspace: slug }} variant="primary">
                  {t(mo.firstReferralsTitle)}
                </ButtonLink>
              ) : null
            }
          />
          <FirstCard
            icon="star"
            title={t(mo.firstRecsTitle)}
            text={t(mo.firstRecsText)}
            action={
              canEdit ? (
                <ButtonLink to="/w/$workspace/grow/recommendations" params={{ workspace: slug }} search={{ add: true }}>
                  {t(mo.addRecommendation)}
                </ButtonLink>
              ) : null
            }
          />
        </div>
        <span className="k">{t(mo.noMarketplace)}</span>
      </section>
    </div>
  )
}

function FirstCard({ icon, title, text, action }: { icon: IconName; title: string; text: string; action: ReactNode }) {
  return (
    <div className="gw-sbox" style={{ padding: '18px 20px' }}>
      <span className="gw-ti" style={{ width: 36, height: 36, background: 'var(--accent-soft)', color: 'var(--accent)' }} aria-hidden="true">
        <Icon name={icon} size={18} />
      </span>
      <h3 className="m-0 text-[16px] font-semibold">{title}</h3>
      <span className="text-ink-2">{text}</span>
      {action ? <div className="mt-1">{action}</div> : null}
    </div>
  )
}

function OverviewSkeleton() {
  const t = useT()
  return (
    <Loading label={t(mo.loadingLabel)}>
      <div className="gw">
        <div className="gw-stats">
          {[0, 1, 2, 3].map((i) => (
            <div className="card gw-stat" key={i}>
              <Skeleton w="60%" h={12} />
              <Skeleton w="40%" h={24} style={{ marginTop: 6 }} />
              <Skeleton w="70%" h={10} style={{ marginTop: 6 }} />
            </div>
          ))}
        </div>
        <Skeleton w="100%" h={120} r={10} />
        <div className="gw-g32">
          <Skeleton w="100%" h={340} r={10} />
          <Skeleton w="100%" h={340} r={10} />
        </div>
      </div>
    </Loading>
  )
}

