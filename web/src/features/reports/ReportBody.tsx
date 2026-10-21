import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { TableScroll, Th } from '@/components/ui/Table'
import { cn } from '@/lib/cn'
import { clock, count, money, pct } from '@/lib/format'
import { providerColor } from '../providers/common'
import { LanguageClickMap, ResultsPerLanguage, sentByLanguage } from '@/features/languages/report'
import { HowAnswersCounted, PollsError, PollSections, PollTiles, useMapPolls, type PollResults } from '@/features/polls/report'
import { m as pm } from '@/features/polls/report.messages'
import { T, useT } from '@/i18n'
import { ClickMap } from './ClickMap'
import { m as rm } from './messages'
import { CampaignStoreRevenue } from '@/features/commerce/revenue'
import { EngagementChart } from './EngagementChart'
import {
  abBadge,
  abExplanation,
  bounceLine,
  fewerNote,
  mailboxColor,
  pct1,
  pct2,
  providerNote,
  seconds,
  sentByLine,
  splitCaption,
  variantLine,
  type CampaignReport,
} from './words'

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

/** One headline tile (`.card.tile`: 14px 16px, 3px gaps). */
function Tile({ label, value, sub, muted, behind, asOf }: { label: string; value: string; sub: ReactNode; muted?: boolean; behind?: boolean; asOf?: string }) {
  const t = useT()
  return (
    <div className={cn('card flex flex-col gap-[3px] px-4 py-3.5', behind && 'border-bad-line')}>
      <span className="k">{label}</span>
      <span className={cn('n text-[22px]', muted && 'text-ink-3')}>{value}</span>
      {asOf ? (
        <Badge tone="warn" className="self-start">
          {t(rm.asOf, { time: asOf })}
        </Badge>
      ) : (
        <span className="k">{sub}</span>
      )}
    </div>
  )
}

/** The eight headline tiles. While provider events lag, the counts they feed say "as of". */
export function HeadlineTiles({ r, polls }: { r: CampaignReport; polls?: PollResults[] }) {
  const t = useT()
  const h = r.headline
  const final = r.campaign.final
  const lag = r.backlog ? clock(r.backlog.since) : undefined
  const current = lag ? t(rm.current) : undefined
  return (
    <Grid min={240} gap={12} className={cn(lag && 'opacity-75')}>
      <Tile label={t(rm.sent)} value={count(h.sent)} sub={sentByLanguage(r.languages) ?? (sentByLine(h) || '—')} />
      <Tile label={t(rm.delivered)} value={count(h.delivered)} sub={pct1(h.deliveredRate)} behind={Boolean(lag)} asOf={lag} />
      {polls?.length ? <PollTiles polls={polls} /> : null}
      <Tile label={t(rm.humanOpens)} value={count(h.humanOpens)} sub={current ?? t(rm.ofDelivered, { rate: pct1(h.humanOpenRate) })} />
      <Tile label={t(rm.machineOpens)} value={count(h.machineOpens)} muted sub={current ?? t(rm.machineSub, { rate: pct1(h.machineOpenRate) })} />
      <Tile label={t(rm.clicks)} value={count(h.clicks)} sub={current ?? t(rm.clicksSub, { rate: pct1(h.clickRate), cto: pct1(h.clickToOpen) })} />
      <Tile label={t(rm.unsubscribes)} value={count(h.unsubscribes)} sub={current ?? pct2(h.unsubscribeRate)} />
      <Tile label={t(rm.complaints)} value={count(h.complaints)} sub={pct2(h.complaintRate)} behind={Boolean(lag)} asOf={lag} />
      <Tile label={t(rm.bounces)} value={count(h.bounces)} sub={bounceLine(h, final)} behind={Boolean(lag)} asOf={lag} />
    </Grid>
  )
}

/** A card with the add-on boards' section bar on top. */
function TableCard({ title, caption, footer, className, children, id }: { title: string; caption: string; footer?: string | null; className?: string; children: ReactNode; id: string }) {
  return (
    <section className={cn('card min-w-0', className)} aria-labelledby={id}>
      <div className="sec">
        <h2 className="h2" id={id}>
          {title}
        </h2>
        <span className="k">{caption}</span>
      </div>
      <TableScroll label={title}>{children}</TableScroll>
      {footer ? <div className="k border-t border-line-2 px-4.5 py-2.5">{footer}</div> : null}
    </section>
  )
}

function MailboxTable({ rows }: { rows: Schemas['ReportMailboxRow'][] }) {
  const t = useT()
  const icloud = rows.some((r) => r.key === 'icloud')
  return (
    <TableCard
      id="rep-mb-h"
      title={t(rm.mailboxTitle)}
      caption={t(rm.mailboxCaption)}
      className="flex-[1_1_560px]"
      footer={icloud ? t(rm.mailboxIcloud) : null}
    >
      <table className="tbl rep-tbl phone-wrap">
        <thead>
          <tr>
            <Th>{t(rm.colMailbox)}</Th>
            <Th align="right">{t(rm.delivered)}</Th>
            <Th align="right">{t(rm.humanOpens)}</Th>
            <Th align="right">{t(rm.machineOpens)}</Th>
            <Th align="right">{t(rm.clicks)}</Th>
            <Th align="right">{t(rm.bounces)}</Th>
            <Th align="right">{t(rm.complaints)}</Th>
            <Th align="right">{t(rm.colUnsubs)}</Th>
          </tr>
        </thead>
        <tbody>
          {rows.map((m) => (
            <tr key={m.key}>
              <td>
                <span className="flex items-center gap-2 whitespace-nowrap">
                  <span className="dot" style={{ color: mailboxColor[m.key] }} aria-hidden="true" />
                  {m.name}
                </span>
                <div className="k ml-4">{count(m.sent)}</div>
              </td>
              <td className="r n">{pct1(m.deliveredRate)}</td>
              <td className="r n">{pct1(m.humanOpenRate)}</td>
              <td className="r n">{pct1(m.machineOpenRate)}</td>
              <td className="r n">{pct1(m.clickRate)}</td>
              <td className="r n">{pct1(m.bounceRate)}</td>
              <td className="r n">{pct2(m.complaintRate)}</td>
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
  const row = (label: string, value: (c: Schemas['ReportProviderColumn']) => string) => (
    <tr>
      <td>{label}</td>
      {cols.map((c) => (
        <td key={c.providerId || c.name} className="r n">
          {value(c)}
        </td>
      ))}
    </tr>
  )
  return (
    <TableCard id="rep-sp-h" title={t(rm.providerTitle)} caption={splitCaption(table)} className="flex-[1_1_440px]" footer={providerNote(table)}>
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
          {row(t(rm.humanOpens), (c) => pct1(c.humanOpenRate))}
          {row(t(rm.clicks), (c) => pct1(c.clickRate))}
          {row(t(rm.bounces), (c) => pct1(c.bounceRate))}
          {row(t(rm.complaints), (c) => pct2(c.complaintRate))}
          {row(t(rm.medianDelivery), (c) => seconds(c.medianDeliveryMs))}
          {row(t(rm.cost), (c) => money(c.cost))}
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
    <TableCard id="rep-bt-h" title={t(rm.bouncesTitle, { count: count(headlineBounces) })} caption={t(rm.bouncesCaption)} className="flex-[2_1_560px]">
      {b.rows.length ? (
        <table className="tbl rep-tbl phone-wrap">
          <thead>
            <tr>
              <Th>{t(rm.colType)}</Th>
              <Th align="right">{t(rm.colCount)}</Th>
              <Th>{t(rm.colCodes)}</Th>
              <Th>{t(rm.colSuppression)}</Th>
            </tr>
          </thead>
          <tbody>
            {b.rows.map((r) => {
              const codes = r.codes.map((c) => [c.provider, c.type, c.code].filter(Boolean).join(' ')).join(' · ')
              return (
                <tr key={r.type}>
                  <td>
                    <Badge tone={bounceTone[r.tone] ?? 'neutral'}>{r.label}</Badge>
                  </td>
                  <td className="r n">{count(r.count)}</td>
                  <td className={cn('mono text-[12px]', codes.length > 50 && 'wrap')}>{codes}</td>
                  <td>
                    <SuppressionCell row={r} actions={actions} />
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      ) : (
        <p className="k m-0 px-4.5 py-4">
          {headlineBounces > 0 ? t(rm.bouncesImported) : t(rm.nothingBounced)}
        </p>
      )}
    </TableCard>
  )
}

const reasonColor: Record<string, string> = { ink: '#0B1220', muted: '#8b94a3', bad: '#B91C1C' }

function Unsubscribes({ u, imported }: { u: Schemas['ReportUnsubscribes']; imported: boolean }) {
  const t = useT()
  const note = fewerNote(u.choseFewer)
  return (
    <section className="card flex min-w-0 flex-[1_1_320px] flex-col gap-2.5 px-5 py-4.5" aria-labelledby="rep-un-h">
      <div className="flex items-center justify-between max-md:flex-wrap">
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
        <span className="k">
          {u.total === 0 ? t(rm.unsubNobody) : imported ? t(rm.unsubImported) : t(rm.unsubNoReason)}
        </span>
      )}
      {note ? <span className="k">{note}</span> : null}
    </section>
  )
}

function ABTestCard({ t: test }: { t: Schemas['ReportABTest'] }) {
  const t = useT()
  const badge = abBadge(test)
  const lines = abExplanation(test)
  const b = (c: ReactNode[]) => <b className="n">{c}</b>
  return (
    <section className="card flex flex-col" aria-labelledby="rep-ab-h">
      <div className="sec">
        <h2 className="h2" id="rep-ab-h">
          {t(rm.abTitle, { label: test.label })}
        </h2>
        <Badge tone={badge.tone} dot>
          {badge.text}
        </Badge>
      </div>
      <Grid min={300} gap={0}>
        {test.variants.map((v) => (
          <div key={v.key} className={cn('flex flex-col gap-2 border-r border-line-2 px-4.5 py-4', v.winner && 'bg-ok-tint')}>
            <div className="flex items-center justify-between gap-2.5 max-md:flex-wrap">
              <Badge tone={v.winner ? 'ok' : 'neutral'}>{v.winner ? t(rm.abVariantWinner, { variant: v.key }) : v.key}</Badge>
              <span className="k">{variantLine(v)}</span>
            </div>
            <span className="font-semibold">{v.label}</span>
            {/* rep-abm: in a longer language the three numbers wrap as wholes (styles/expansion.css). */}
            <div className="rep-abm flex gap-4 text-[13px] max-md:flex-wrap">
              <span>
                <T msg={rm.abHumanOpens} values={{ rate: pct1(v.humanOpenRate), b }} />
              </span>
              <span>
                <T msg={rm.abClicks} values={{ rate: pct1(v.clickRate), b }} />
              </span>
              <span>
                <T msg={rm.abComplaints} values={{ rate: pct2(v.complaintRate), b }} />
              </span>
            </div>
            <div className="bar h-2" aria-hidden="true">
              <i style={{ width: `${v.humanOpenRate ?? 0}%`, background: v.winner ? 'var(--accent)' : '#8b94a3' }} />
            </div>
          </div>
        ))}
        <div className="flex flex-col gap-1.5 px-4.5 py-4 text-[13px]">
          <span className="font-semibold">{test.state === 'testing' ? t(rm.abHowIs) : t(rm.abHowWas)}</span>
          {lines.map((l) => (
            <span key={l} className="k">
              {l}
            </span>
          ))}
        </div>
      </Grid>
    </section>
  )
}

/**
 * Everything below the header: the report's cards in the board's order.
 * `earn` is the Revenue and Sponsor sections (EarnInReport), after the
 * click map; shared read-only reports leave it out.
 */
export function ReportBody({ r, actions, polls, earn }: { r: CampaignReport; actions?: ReportActions; polls?: ReportPolls; earn?: ReactNode }) {
  const t = useT()
  const items = polls?.items?.length ? polls.items : undefined
  const mapPolls = useMapPolls(items)
  const clickMap = (
    <LanguageClickMap r={r} campaignId={r.campaign.id}>
      {(m, picker, note) => (
        <ClickMap
          m={m}
          clicks={r.headline.clicks}
          imported={Boolean(r.campaign.origin)}
          picker={picker}
          languageNote={note}
          polls={mapPolls}
          pollNote={<span className="k">{t(pm.mapNote)}</span>}
        />
      )}
    </LanguageClickMap>
  )
  const engagement = <EngagementChart e={r.engagement} startedAt={r.campaign.startedAt} final={r.campaign.final} />
  return (
    <>
      <HeadlineTiles r={r} polls={items} />
      {items ? (
        // With a poll (PollResults): the click map beside how answers are
        // counted, then the Poll section.
        <>
          <div className="flex flex-wrap items-stretch gap-5">{engagement}</div>
          <div className="flex flex-wrap items-stretch gap-5">
            {clickMap}
            <HowAnswersCounted polls={items} />
          </div>
          <PollSections campaignId={r.campaign.id} polls={items} />
        </>
      ) : (
        <div className="flex flex-wrap items-stretch gap-5">
          {engagement}
          {clickMap}
        </div>
      )}
      {polls?.error ? <PollsError error={polls.error} onRetry={polls.retry} /> : null}
      {earn}
      {/* Store orders credited to the email (X20), after Earn's Revenue and Sponsor sections. */}
      <CampaignStoreRevenue campaignId={r.campaign.id} campaign={r.campaign.name} />
      <div className="flex flex-wrap items-start gap-5">
        <MailboxTable rows={r.mailboxes} />
        {r.providers.columns.length ? <ProviderTable t={r.providers} /> : null}
      </div>
      {r.languages ? <ResultsPerLanguage r={r} /> : null}
      <div className="flex flex-wrap items-start gap-5">
        <BounceTable b={r.bounces} headlineBounces={r.headline.bounces} actions={actions} />
        <Unsubscribes u={r.unsubscribes} imported={Boolean(r.campaign.origin)} />
      </div>
      {r.abTest ? <ABTestCard t={r.abTest} /> : null}
    </>
  )
}
