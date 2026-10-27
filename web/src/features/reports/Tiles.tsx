import type { ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { cn } from '@/lib/cn'
import { clock, count, shortDate } from '@/lib/format'
import { useOptionalWorkspace } from '@/lib/session'
import { T, useT } from '@/i18n'
import { PollTiles, type PollResults } from '@/features/polls/report'
import { sentByLanguage } from '@/features/languages/report'
import { m } from './messages'
import { bounceSub, comparison, notReportedLine, pct1, pct2, sentByLine, type CampaignReport, type TileComparison } from './words'

/** "What SMTP can and can't do": the adapters guide, whose table says what each provider reports. */
const SMTP_GUIDE = 'https://github.com/hostrepute/reputecast/blob/main/docs/adapters.md'

type Column = Schemas['ReportProviderColumn']

/** One headline tile (`.card.tile`: 14px 16px, 3px gaps) with its comparison. */
function Tile({
  label,
  value,
  sub,
  extra,
  cmp,
  muted,
  behind,
  asOf,
  badge,
}: {
  label: string
  value: string
  sub: ReactNode
  extra?: ReactNode
  cmp?: TileComparison
  muted?: boolean
  behind?: boolean
  asOf?: string
  badge?: ReactNode
}) {
  const t = useT()
  return (
    <div className={cn('card rep-tile', behind && 'border-bad-line')}>
      <span className="k">{label}</span>
      <span className={cn('n text-[22px]', muted && 'text-ink-3')}>{value}</span>
      {asOf ? (
        <Badge tone="warn" className="self-start">
          {t(m.asOf, { time: asOf })}
        </Badge>
      ) : (
        <>
          <span className="k">{sub}</span>
          {extra}
          {badge}
          {cmp ? (
            <span className={cn('rep-cmp', cmp.tone)}>
              <span aria-hidden="true">{cmp.arrow} </span>
              {cmp.direction ? <span className="vh">{cmp.direction} </span> : null}
              {cmp.text}
            </span>
          ) : null}
        </>
      )}
    </div>
  )
}

/** The provider whose own tracking counted every open and click of the send, if one did. */
export function trackedBy(r: CampaignReport): Column | undefined {
  const cols = r.providers.columns.filter((c) => c.sent > 0)
  return cols.length === 1 && cols[0]!.trackedBy === 'provider' ? cols[0] : undefined
}

/** Whether every message of the send went through a relay whose deliveries are inferred. */
function inferredOnly(r: CampaignReport): Column | undefined {
  const cols = r.providers.columns.filter((c) => c.sent > 0)
  return cols.length === 1 && cols[0]!.deliveryInferred ? cols[0] : undefined
}

/**
 * The headline tiles. Once the results are final there are eight, each with
 * its change against the last issues; while they still come in, six that
 * say "so far" (ReportStates). While provider events lag, the counts they
 * feed say "as of".
 */
export function HeadlineTiles({ r, polls, onBots }: { r: CampaignReport; polls?: PollResults[]; onBots?: () => void }) {
  const t = useT()
  const h = r.headline
  const final = r.campaign.final
  const lag = r.backlog ? clock(r.backlog.since) : undefined
  const current = lag ? t(m.current) : undefined
  // Opens and clicks lag too when a provider that is behind tracked them.
  const lagIds = new Set(r.backlog?.providers.map((p) => p.providerId) ?? [])
  const opensLag = lag && r.providers.columns.some((c) => c.trackedBy === 'provider' && lagIds.has(c.providerId)) ? lag : undefined
  const tracker = trackedBy(r)
  const relay = inferredOnly(r)
  const from = (text: string) => (tracker ? t(m.fromProvider, { text, provider: tracker.short || tracker.name }) : text)
  const cmp = (metric: Schemas['ReportComparison']['metric']) => (final ? comparison(r.comparisons, metric) : undefined)
  const measurable = h.humanOpenRateMeasurable !== undefined
  const humanSub = measurable
    ? t(final ? m.humanSub : m.humanSoFar, { rate: pct1(h.humanOpenRate), measurable: pct1(h.humanOpenRateMeasurable) })
    : t(m.humanOfDelivered, { rate: pct1(h.humanOpenRate) })
  const deliveredSub = final
    ? h.undelivered > 0
      ? t(m.deliveredSub, { rate: pct1(h.deliveredRate), count: count(h.undelivered) })
      : pct1(h.deliveredRate)
    : h.undelivered > 0
      ? t(m.deliveredRetrying, { rate: pct1(h.deliveredRate), count: count(h.undelivered) })
      : t(m.deliveredSoFar, { rate: pct1(h.deliveredRate) })
  const notReported = notReportedLine(h)
  const complaintsSub = final
    ? h.complaintRateReported !== undefined && notReported
      ? t(m.complaintsSub, { rate: pct2(h.complaintRate), reported: pct2(h.complaintRateReported) })
      : t(m.unsubSub, { rate: pct2(h.complaintRate) })
    : [t(m.complaintsSoFar, { rate: pct2(h.complaintRate) }), notReported].filter(Boolean).join(' · ')
  const bots =
    h.botClicks > 0 ? (
      <span className="k">
        {onBots ? (
          <a
            href="#rep-click-map"
            onClick={(e) => {
              e.preventDefault()
              onBots()
            }}
          >
            {t(m.botFiltered, { n: h.botClicks, count: count(h.botClicks) })}
          </a>
        ) : (
          t(m.botFiltered, { n: h.botClicks, count: count(h.botClicks) })
        )}
      </span>
    ) : null
  const bounces = relay ? t(m.fromBounceMailbox, { text: bounceSub(h, final) }) : bounceSub(h, final)
  const delivered = (
    <Tile
      label={t(m.delivered)}
      value={count(h.delivered)}
      sub={deliveredSub}
      cmp={cmp('delivered')}
      behind={Boolean(lag)}
      asOf={lag}
      badge={
        relay ? (
          <Badge tone="neutral" className="self-start">
            {t(m.inferred)}
          </Badge>
        ) : undefined
      }
    />
  )
  const human = <Tile label={t(m.humanOpens)} value={count(h.humanOpens)} sub={opensLag ? undefined : (current ?? from(humanSub))} cmp={cmp('humanOpens')} behind={Boolean(opensLag)} asOf={opensLag} />
  const privacy = (
    <Tile
      label={t(m.privacyOpens)}
      value={count(h.privacyOpens)}
      muted
      sub={opensLag ? undefined : (current ?? t(final ? m.privacySub : m.privacySoFar, { rate: pct1(h.privacyOpenRate) }))}
      cmp={cmp('privacyOpens')}
      behind={Boolean(opensLag)}
      asOf={opensLag}
    />
  )
  const clicks = (
    <Tile
      label={t(m.humanClicks)}
      value={count(h.clicks)}
      sub={opensLag ? undefined : (current ?? from(t(final ? m.clicksPeople : m.clicksSoFar, { rate: pct1(h.clickRate), cto: pct1(h.clickToOpen) })))}
      extra={final ? bots : null}
      cmp={cmp('clicks')}
      behind={Boolean(opensLag)}
      asOf={opensLag}
    />
  )
  const complaints = (
    <Tile
      label={t(m.complaints)}
      value={count(h.complaints)}
      sub={complaintsSub}
      extra={final && notReported ? <span className="k">{notReported}</span> : null}
      cmp={cmp('complaints')}
      behind={Boolean(lag)}
      asOf={lag}
    />
  )
  const bounceTile = <Tile label={t(m.bounces)} value={count(h.bounces)} sub={bounces} cmp={cmp('bounces')} behind={Boolean(lag)} asOf={lag} />
  if (!final) {
    return (
      <div className={cn('rep-tiles six', lag && 'opacity-75')}>
        {delivered}
        {polls?.length ? <PollTiles polls={polls} /> : null}
        {human}
        {privacy}
        {clicks}
        {bounceTile}
        {complaints}
      </div>
    )
  }
  return (
    <div className={cn('rep-tiles', lag && 'opacity-75')}>
      <Tile label={t(m.sent)} value={count(h.sent)} sub={sentByLanguage(r.languages) ?? (sentByLine(h) || '—')} cmp={cmp('sent')} />
      {delivered}
      {polls?.length ? <PollTiles polls={polls} /> : null}
      {human}
      {privacy}
      {clicks}
      <Tile label={t(m.unsubscribes)} value={count(h.unsubscribes)} sub={current ?? t(m.unsubSub, { rate: pct2(h.unsubscribeRate) })} cmp={cmp('unsubscribes')} />
      {complaints}
      {bounceTile}
    </div>
  )
}

/** Under the tiles: the scanner opens left out, and what the arrows compare with. */
export function TileNotes({ r }: { r: CampaignReport }) {
  const t = useT()
  const h = r.headline
  if (!r.campaign.final) return null
  const c = r.comparisons
  const right = c.tiles.length && c.issues > 0 ? t(m.cmpNote, { n: c.issues }) : c.unavailable === 'no_history' ? t(m.cmpNoHistory) : null
  const left = h.machineOpens > 0 ? t(m.scannerNote, { count: count(h.machineOpens) }) : null
  if (!left && !right) return null
  return (
    <div className="k flex flex-wrap justify-between gap-x-4 gap-y-1">
      <span>{left}</span>
      <span>{right}</span>
    </div>
  )
}

/** "What “of measurable” means": delivered split into human opens, measurable without one, and privacy-protected. */
export function MeasurableDisclosure({ r }: { r: CampaignReport }) {
  const t = useT()
  const h = r.headline
  if (h.delivered <= 0 || h.humanOpenRateMeasurable === undefined) return null
  const rest = Math.max(0, h.measurable - h.humanOpens)
  const share = (n: number) => (n / h.delivered) * 100
  const values = {
    delivered: count(h.delivered),
    human: count(h.humanOpens),
    rest: count(rest),
    privacy: count(h.privacyOpens),
  }
  return (
    <details className="card rep-opx">
      <summary>
        <span className="chev" aria-hidden="true">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
            <path d="m6 4 4 4-4 4" />
          </svg>
        </span>
        {t(m.measurableTitle)}
      </summary>
      <div className="rep-opx-b">
        <div className="flex flex-col gap-1.5">
          <span className="k">{t(m.ofDeliveredCount, { count: values.delivered })}</span>
          <div className="rep-stk" role="img" aria-label={t(m.stackLabel, values)}>
            <i style={{ width: `${share(h.humanOpens)}%`, background: 'var(--ink)' }} />
            <i style={{ width: `${share(h.privacyOpens)}%`, background: '#B4BAC5' }} />
          </div>
          <div className="rep-leg" aria-hidden="true">
            <span>
              <span className="rep-sw" style={{ background: 'var(--ink)' }} />
              {t(m.stackHuman, { count: values.human, rate: pct1(share(h.humanOpens)) })}
            </span>
            <span>
              <span className="rep-sw" style={{ background: 'var(--line-2)', border: '1px solid var(--line-strong)' }} />
              {t(m.stackMeasurable, { count: values.rest, rate: pct1(share(rest)) })}
            </span>
            <span>
              <span className="rep-sw" style={{ background: '#B4BAC5' }} />
              {t(m.stackPrivacy, { count: values.privacy, rate: pct1(share(h.privacyOpens)) })}
            </span>
          </div>
        </div>
        <p>
          {t(m.measurableBody, {
            delivered: values.delivered,
            privacy: values.privacy,
            measurable: count(h.measurable),
            human: values.human,
            rateMeasurable: pct1(h.humanOpenRateMeasurable),
            rate: pct1(h.humanOpenRate),
          })}
        </p>
        <p>{t(m.measurableScanners, { scanners: count(h.machineOpens) })}</p>
      </div>
    </details>
  )
}

/** The board's grey note with an info mark. */
export function InfoNote({ title, children, id }: { title?: ReactNode; children: ReactNode; id?: string }) {
  return (
    <div className="rep-note" id={id}>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="#5B6472" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
        <circle cx="8" cy="8" r="6.2" />
        <path d="M8 7.2v3.8M8 5v.1" />
      </svg>
      <div className="flex min-w-0 flex-col gap-0.5">
        {title ? <span className="font-semibold text-ink">{title}</span> : null}
        <span>{children}</span>
      </div>
    </div>
  )
}

/**
 * Tracking source notes (ReportStates): a provider whose own tracking
 * counted the opens and clicks, and a relay whose deliveries are inferred.
 */
export function TrackingNotes({ r }: { r: CampaignReport }) {
  const t = useT()
  const ws = useOptionalWorkspace()
  const cols = r.providers.columns.filter((c) => c.sent > 0)
  const tracked = cols.filter((c) => c.trackedBy === 'provider')
  const relays = cols.filter((c) => c.deliveryInferred)
  if (!tracked.length && !relays.length) return null
  const providersLink = (c: ReactNode[]) =>
    ws ? (
      <Link to="/w/$workspace/providers" params={{ workspace: ws.slug }}>
        {c}
      </Link>
    ) : (
      <>{c}</>
    )
  return (
    <div className="flex flex-col gap-2.5">
      {tracked.map((c) => {
        const provider = c.short || c.name
        return (
          <InfoNote key={`t-${c.providerId || c.name}`} title={t(m.trackedByProvider, { provider })}>
            {t(m.trackedByProviderBody, { provider })}{' '}
            {ws ? <T msg={m.changeInProvidersLink} values={{ a: providersLink }} /> : null}
          </InfoNote>
        )
      })}
      {relays.map((c) => (
        <InfoNote key={`r-${c.providerId || c.name}`} title={cols.length > 1 ? t(m.inferredTitleVia, { provider: c.short || c.name }) : t(m.inferredTitle)}>
          {t(m.inferredBody)}{' '}
          <a href={SMTP_GUIDE} target="_blank" rel="noreferrer">
            {t(m.smtpCan)}
            <span className="vh"> {t(m.newTab)}</span>
          </a>
        </InfoNote>
      ))}
    </div>
  )
}

/**
 * "Results are still coming in · final on Fri 24 Oct, 12:41." with "Email
 * me when it's final" (ReportStates). A read-only link shows it without the
 * button.
 */
export function StillComing({ finalAt, issues, action }: { finalAt: string; issues: number; action?: ReactNode }) {
  const t = useT()
  return (
    <div className="rep-banner" role="status">
      <Badge tone="info" dot>
        {t(m.stillComing)}
      </Badge>
      <span className="min-w-0 flex-[1_1_320px]">
        <b>{t(m.stillComingTitle, { day: shortDate(finalAt), time: clock(finalAt) })}</b> {t(m.stillComingBody, { n: issues })}
      </span>
      {action}
    </div>
  )
}

