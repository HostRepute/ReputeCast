import { useRef, useState, type ReactNode } from 'react'
import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { TableScroll, Th } from '@/components/ui/Table'
import { cn } from '@/lib/cn'
import { andList, count, money, pct } from '@/lib/format'
import { useOptionalWorkspace } from '@/lib/session'
import { providerColor } from '../providers/common'
import { LanguageClickMap, ResultsPerLanguage } from '@/features/languages/report'
import { HowAnswersCounted, PollsError, PollSections, useMapPolls, type PollResults } from '@/features/polls/report'
import { m as pm } from '@/features/polls/report.messages'
import { T, useT } from '@/i18n'
import { ABTestCard } from './ABTestCard'
import { ClickMap } from './ClickMap'
import { m as rm } from './messages'
import { CampaignStoreRevenue } from '@/features/commerce/revenue'
import { EngagementChart } from './EngagementChart'
import { Growth, Insights } from './Insights'
import { HeadlineTiles, InfoNote, MeasurableDisclosure, TileNotes, TrackingNotes } from './Tiles'
import { fewerNote, mailboxColor, pct1, pct2, providerNote, seconds, splitCaption, type CampaignReport } from './words'

type BounceRow = Schemas['ReportBounceRow']

/** A campaign's polls (design expansion, X11): their results, or why they're missing. */
export interface ReportPolls {
  items?: PollResults[]
  error?: unknown
  retry(): void
}

export interface ReportActions {
  /** Why the current member can't add addresses to the suppression list. */
  suppressReason?: string
  onSuppress: (row: BounceRow) => void
  onReview: (row: BounceRow) => void
}

/** A card with the add-on boards' section bar on top. */
function TableCard({ title, caption, footer, className, children, id }: { title: string; caption: string; footer?: ReactNode; className?: string; children: ReactNode; id: string }) {
  return (
    <section className={cn('card min-w-0', className)} aria-labelledby={id}>
      <div className="sec">
        <h2 className="h2" id={id}>
          {title}
        </h2>
        <span className="k">{caption}</span>
      </div>
      <TableScroll label={title}>{children}</TableScroll>
      {footer ? <div className="k flex flex-col gap-1 border-t border-line-2 px-4.5 py-2.5">{footer}</div> : null}
    </section>
  )
}

/** "n/a" with why, for a rate a mailbox provider never reports (review F-2). */
function NotReported() {
  const t = useT()
  return (
    <td className="r k">
      {t(rm.na)}
      <span className="vh">{t(rm.naNotReported)}</span>
    </td>
  )
}

/** The notes under the mailbox table: how providers are told apart, Apple Mail, and complaints not reported. */
function mailboxNotes(t: ReturnType<typeof useT>, rows: Schemas['ReportMailboxRow'][], slug?: string): ReactNode[] {
  const out: ReactNode[] = [<span key="mx">{t(rm.mxNote)}</span>]
  const icloud = rows.find((r) => r.key === 'icloud')
  const gmail = rows.find((r) => r.key === 'gmail')
  if (icloud?.privacyOpenRate !== undefined && gmail?.privacyOpenRate !== undefined && icloud.humanOpenRateMeasurable !== undefined) {
    out.push(
      <span key="app">
        {t(rm.privacyAppNote, { icloud: pct1(icloud.privacyOpenRate), gmail: pct1(gmail.privacyOpenRate), rate: pct1(icloud.humanOpenRateMeasurable), count: count(icloud.measurable) })}
      </span>,
    )
  }
  const na = rows.filter((r) => r.complaintRateNa === 'not_reported').map((r) => r.name)
  if (na.length) {
    const deliverability = (c: ReactNode[]) =>
      slug ? (
        <Link to="/w/$workspace/deliverability" params={{ workspace: slug }}>
          {c}
        </Link>
      ) : (
        <>{c}</>
      )
    out.push(
      <span key="na">{na.includes('Gmail') ? <T msg={rm.naNoteGmail} values={{ providers: andList(na), a: deliverability }} /> : t(rm.naNote, { providers: andList(na) })}</span>,
    )
  }
  return out
}

function MailboxTable({ rows }: { rows: Schemas['ReportMailboxRow'][] }) {
  const t = useT()
  const ws = useOptionalWorkspace()
  const sub = (s: string) => <span className="rep-sub">{s}</span>
  return (
    <TableCard id="rep-mb-h" title={t(rm.mailboxTitle)} caption={t(rm.mailboxCaption)} footer={mailboxNotes(t, rows, ws?.slug)}>
      <table className="tbl rep-tbl phone-wrap">
        <thead>
          <tr>
            <Th>{t(rm.colMailbox)}</Th>
            <Th align="right">{t(rm.delivered)}</Th>
            <Th align="right">
              {t(rm.humanOpens)}
              {sub(t(rm.ofMeasurable))}
            </Th>
            <Th align="right">
              {t(rm.privacyOpens)}
              {sub(t(rm.ofDeliveredSub))}
            </Th>
            <Th align="right">{t(rm.humanClicks)}</Th>
            <Th align="right">{t(rm.bounces)}</Th>
            <Th align="right">{t(rm.complaints)}</Th>
            <Th align="right">{t(rm.colUnsubs)}</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((m) => (
            <tr key={m.key}>
              <th scope="row">
                <span className="flex items-center gap-2 whitespace-nowrap">
                  <span className="dot" style={{ color: mailboxColor[m.key] }} aria-hidden="true" />
                  {m.name}
                </span>
                <span className="k ml-4 block whitespace-nowrap font-medium">{t(rm.sentCount, { count: count(m.sent) })}</span>
              </th>
              <td className="r n">{pct1(m.deliveredRate)}</td>
              <td className="r n">{pct1(m.humanOpenRateMeasurable ?? m.humanOpenRate)}</td>
              <td className="r n">{pct1(m.privacyOpenRate)}</td>
              <td className="r n">{pct1(m.clickRate)}</td>
              <td className="r n">{pct1(m.bounceRate)}</td>
              {m.complaintRateNa === 'not_reported' ? <NotReported /> : <td className="r n">{pct2(m.complaintRate)}</td>}
              <td className="r n">{pct2(m.unsubscribeRate)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </TableCard>
  )
}

function ProviderTable({ t: table }: { t: Schemas['ReportProviderTable'] }) {
  const t = useT()
  const cols = table.columns
  const row = (label: string, value: (c: Schemas['ReportProviderColumn']) => ReactNode) => (
    <tr>
      <th scope="row">{label}</th>
      {cols.map((c) => (
        <td key={c.providerId || c.name} className="r n">
          {value(c)}
        </td>
      ))}
    </tr>
  )
  const note = providerNote(table)
  const costs = cols.some((c) => c.cost !== undefined)
  return (
    <TableCard id="rep-sp-h" title={t(rm.providerTitle)} caption={splitCaption(table)} className="flex-[3_1_440px]" footer={note ? <span>{note}</span> : null}>
      <table className="tbl rep-tbl phone-wrap">
        <thead>
          <tr>
            <Th>{t(rm.colMetric)}</Th>
            {cols.map((c) => (
              <th key={c.providerId || c.name} scope="col" className="r">
                <span className="inline-flex items-center gap-1.5">
                  <span className="dot" style={{ color: providerColor[c.color] }} aria-hidden="true" />
                  {c.name}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {row(t(rm.sent), (c) => count(c.sent))}
          {row(t(rm.delivered), (c) => pct1(c.deliveredRate))}
          {row(t(rm.humanOpensMeasurable), (c) => pct1(c.humanOpenRateMeasurable ?? c.humanOpenRate))}
          {row(t(rm.privacyOpens), (c) => pct1(c.privacyOpenRate))}
          {row(t(rm.humanClicks), (c) => pct1(c.clickRate))}
          {row(t(rm.bounces), (c) => pct1(c.bounceRate))}
          {row(t(rm.complaints), (c) => pct2(c.complaintRate))}
          {row(t(rm.medianDelivery), (c) => seconds(c.medianDeliveryMs))}
          {costs ? row(t(rm.cost), (c) => money(c.cost)) : null}
        </tbody>
      </table>
    </TableCard>
  )
}

const bounceTone: Record<string, Tone> = { bad: 'bad', warn: 'warn', neutral: 'neutral' }

function SuppressionCell({ row, actions }: { row: BounceRow; actions?: ReportActions }) {
  const t = useT()
  switch (row.suppression) {
    case 'auto':
      return <Badge tone="ok">{t(rm.autoSuppressed)}</Badge>
    case 'retry':
      return <Badge tone="neutral">{t(rm.retryNext)}</Badge>
    case 'suppressed':
      return <Badge tone="ok">{t(rm.suppressed)}</Badge>
    case 'investigate':
      // A block for the sender's reputation: the addresses are valid (review AN-7).
      return actions ? (
        <>
          <Button size="sm" className="rep-noprint" aria-describedby="rep-blocked-note" onClick={() => actions.onReview(row)}>
            {t(rm.investigate)}
          </Button>
          <NotSuppressed print />
        </>
      ) : (
        <NotSuppressed />
      )
    case 'add':
      return actions ? (
        <>
          <Button size="sm" className="rep-noprint" disabledReason={actions.suppressReason} onClick={() => actions.onSuppress(row)}>
            {t(rm.addToSuppression, { count: count(row.unsuppressed) })}
          </Button>
          <NotSuppressed print />
        </>
      ) : (
        <NotSuppressed />
      )
    default:
      return actions ? (
        <>
          <Button size="sm" className="rep-noprint" onClick={() => actions.onReview(row)}>
            {t(rm.review)}
          </Button>
          <NotSuppressed print />
        </>
      ) : (
        <NotSuppressed />
      )
  }
}

/** What a printed report (or a read-only link) says where the screen offers an action. */
function NotSuppressed({ print }: { print?: boolean }) {
  const t = useT()
  return (
    <Badge tone="neutral" className={print ? 'rep-printonly' : undefined}>
      {t(rm.notSuppressed)}
    </Badge>
  )
}

function BounceTable({ b, headlineBounces, actions }: { b: Schemas['ReportBounceTable']; headlineBounces: number; actions?: ReportActions }) {
  const t = useT()
  return (
    <TableCard id="rep-bt-h" title={t(rm.bouncesTitle, { count: count(headlineBounces) })} caption={t(rm.bouncesCaption)}>
      {b.rows.length ? (
        <table className="tbl rep-tbl phone-wrap">
          <thead>
            <tr>
              <Th>{t(rm.colType)}</Th>
              <Th align="right">{t(rm.colCount)}</Th>
              <Th>{t(rm.colSuppression)}</Th>
              <Th>{t(rm.colCodes)}</Th>
            </tr>
          </thead>
          <tbody>
            {b.rows.map((r) => {
              const codes = r.codes.map((c) => [c.provider, c.type, c.code].filter(Boolean).join(' ')).join(' · ')
              const blocked = r.suppression === 'investigate'
              return [
                <tr key={r.type} className={cn(blocked && 'hl')}>
                  <td>
                    <Badge tone={bounceTone[r.tone] ?? 'neutral'}>{r.label}</Badge>
                  </td>
                  <td className="r n">{count(r.count)}</td>
                  <td>
                    <SuppressionCell row={r} actions={actions} />
                  </td>
                  <td className={cn('mono text-[12px]', codes.length > 50 && 'wrap')}>{codes}</td>
                </tr>,
                blocked ? (
                  <tr key={`${r.type}-note`} className="nt">
                    <td colSpan={4}>
                      <InfoNote id="rep-blocked-note">{t(rm.blockedNote)}</InfoNote>
                    </td>
                  </tr>
                ) : null,
              ]
            })}
          </tbody>
        </table>
      ) : (
        <p className="k m-0 px-4.5 py-4">{headlineBounces > 0 ? t(rm.bouncesImported) : t(rm.nothingBounced)}</p>
      )}
    </TableCard>
  )
}

const reasonColor: Record<string, string> = { ink: '#0B1220', muted: '#8b94a3', bad: '#B91C1C' }

function Unsubscribes({ u, imported }: { u: Schemas['ReportUnsubscribes']; imported: boolean }) {
  const t = useT()
  const note = fewerNote(u.choseFewer)
  return (
    <section className="card flex min-w-0 flex-[2_1_320px] flex-col gap-2.5 px-5 py-4.5" aria-labelledby="rep-un-h">
      <div className="flex items-center justify-between gap-2 max-md:flex-wrap">
        <h2 className="h2" id="rep-un-h">
          {t(rm.unsubTitle, { count: count(u.total) })}
        </h2>
        {imported && !u.reasons.length ? null : <span className="k">{t(rm.unsubCaption)}</span>}
      </div>
      {u.reasons.length ? (
        <div className="flex flex-col gap-2 text-[13px]">
          {u.reasons.map((r) => (
            <div key={r.key}>
              <div className="flex justify-between max-md:flex-wrap">
                <span>{r.label}</span>
                <span className="n">{count(r.count)}</span>
              </div>
              <div className="bar" role="img" aria-label={t(rm.unsubShare, { reason: r.label, share: pct(Math.round(r.share), 0) })}>
                <i style={{ width: `${Math.round(r.share)}%`, background: reasonColor[r.tone] }} />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <span className="k">{u.total === 0 ? t(rm.unsubNobody) : imported ? t(rm.unsubImported) : t(rm.unsubNoReason)}</span>
      )}
      {note ? <span className="k">{note}</span> : null}
    </section>
  )
}

/**
 * Everything below the header: the report's cards in the board's order.
 * `earn` is the Revenue and Sponsor sections (EarnInReport), after the
 * click map; shared read-only reports leave it out, and say with `email`
 * whether the link shows the email itself.
 */
export function ReportBody({
  r,
  actions,
  polls,
  earn,
  banner,
  email = true,
}: {
  r: CampaignReport
  actions?: ReportActions
  polls?: ReportPolls
  earn?: ReactNode
  /** "Results are still coming in" with its actions, above the tiles. */
  banner?: ReactNode
  email?: boolean
}) {
  const t = useT()
  const items = polls?.items?.length ? polls.items : undefined
  const mapPolls = useMapPolls(items)
  const [view, setView] = useState<'human' | 'bot'>('human')
  const [focus, setFocus] = useState<number>()
  const map = useRef<HTMLElement>(null)
  const toMap = (v: 'human' | 'bot', position?: number) => {
    setView(v)
    setFocus(position)
    map.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    if (position === undefined) map.current?.focus({ preventScroll: true })
  }
  const clickMap = (
    <LanguageClickMap r={r} campaignId={r.campaign.id}>
      {(m, picker, note) => (
        <ClickMap
          ref={map}
          m={m}
          clicks={r.headline.clicks}
          imported={Boolean(r.campaign.origin)}
          emailShared={email}
          picker={picker}
          languageNote={note}
          polls={mapPolls}
          pollNote={<span className="k">{t(pm.mapNote)}</span>}
          view={view}
          onView={(v) => {
            setView(v)
            setFocus(undefined)
          }}
          focus={focus}
        />
      )}
    </LanguageClickMap>
  )
  const remaining = r.abTest?.state === 'decided' || r.abTest?.state === 'no_winner' ? r.abTest.remaining : undefined
  return (
    <>
      {banner}
      <section className="flex flex-col gap-2.5" aria-label={t(rm.headlineLabel)}>
        <TrackingNotes r={r} />
        <HeadlineTiles r={r} polls={items} onBots={() => toMap('bot')} />
        <TileNotes r={r} />
        {r.campaign.final ? <MeasurableDisclosure r={r} /> : null}
      </section>
      <div className="flex flex-wrap items-start gap-5">
        <Insights r={r} onShowLink={actions ? (p) => toMap('human', p) : undefined} />
        <Growth r={r} campaignId={actions ? r.campaign.id : undefined} />
      </div>
      <EngagementChart e={r.engagement} startedAt={r.campaign.startedAt} final={r.campaign.final} remaining={remaining} />
      {clickMap}
      {items ? (
        // With a poll (PollResults): how answers are counted, then the Poll section.
        <>
          <HowAnswersCounted polls={items} />
          <PollSections campaignId={r.campaign.id} polls={items} />
        </>
      ) : null}
      {polls?.error ? <PollsError error={polls.error} onRetry={polls.retry} /> : null}
      {earn}
      {/* Store orders credited to the email (X20), after Earn's Revenue and Sponsor sections. */}
      <CampaignStoreRevenue campaignId={r.campaign.id} campaign={r.campaign.name} />
      <MailboxTable rows={r.mailboxes} />
      <div className="flex flex-wrap items-start gap-5">
        {r.providers.columns.length ? <ProviderTable t={r.providers} /> : null}
        <Unsubscribes u={r.unsubscribes} imported={Boolean(r.campaign.origin)} />
      </div>
      {r.languages ? <ResultsPerLanguage r={r} /> : null}
      <BounceTable b={r.bounces} headlineBounces={r.headline.bounces} actions={actions} />
      {r.abTest ? <ABTestCard r={r} /> : null}
    </>
  )
}
