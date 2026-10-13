import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { cn } from '@/lib/cn'
import { clock, count, money } from '@/lib/format'
import { providerColor } from '../providers/common'
import { ClickMap } from './ClickMap'
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

export interface ReportActions {
  /** Why the current member can't add addresses to the suppression list. */
  suppressReason?: string
  onSuppress: (row: BounceRow) => void
  onReview: (row: BounceRow) => void
}

/** One headline tile (`.card.tile`: 14px 16px, 3px gaps). */
function Tile({ label, value, sub, muted, behind, asOf }: { label: string; value: string; sub: ReactNode; muted?: boolean; behind?: boolean; asOf?: string }) {
  return (
    <div className={cn('card flex flex-col gap-[3px] px-4 py-3.5', behind && 'border-bad-line')}>
      <span className="k">{label}</span>
      <span className={cn('n text-[22px]', muted && 'text-ink-3')}>{value}</span>
      {asOf ? (
        <Badge tone="warn" className="self-start">
          as of {asOf}
        </Badge>
      ) : (
        <span className="k">{sub}</span>
      )}
    </div>
  )
}

/** The eight headline tiles. While provider events lag, the counts they feed say "as of". */
export function HeadlineTiles({ r }: { r: CampaignReport }) {
  const h = r.headline
  const final = r.campaign.final
  const lag = r.backlog ? clock(r.backlog.since) : undefined
  const current = lag ? 'current' : undefined
  return (
    <Grid min={240} gap={12} className={cn(lag && 'opacity-75')}>
      <Tile label="Sent" value={count(h.sent)} sub={sentByLine(h) || '—'} />
      <Tile label="Delivered" value={count(h.delivered)} sub={pct1(h.deliveredRate)} behind={Boolean(lag)} asOf={lag} />
      <Tile label="Human opens" value={count(h.humanOpens)} sub={current ?? `${pct1(h.humanOpenRate)} of delivered`} />
      <Tile label="Machine opens" value={count(h.machineOpens)} muted sub={current ?? `${pct1(h.machineOpenRate)} · Apple Mail prefetch, scanners`} />
      <Tile label="Clicks" value={count(h.clicks)} sub={current ?? `${pct1(h.clickRate)} · click-to-open ${pct1(h.clickToOpen)}`} />
      <Tile label="Unsubscribes" value={count(h.unsubscribes)} sub={current ?? pct2(h.unsubscribeRate)} />
      <Tile label="Complaints" value={count(h.complaints)} sub={pct2(h.complaintRate)} behind={Boolean(lag)} asOf={lag} />
      <Tile label="Bounces" value={count(h.bounces)} sub={bounceLine(h, final)} behind={Boolean(lag)} asOf={lag} />
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
      <div className="tscroll">{children}</div>
      {footer ? <div className="k border-t border-line-2 px-4.5 py-2.5">{footer}</div> : null}
    </section>
  )
}

function MailboxTable({ rows }: { rows: Schemas['ReportMailboxRow'][] }) {
  const icloud = rows.some((r) => r.key === 'icloud')
  return (
    <TableCard
      id="rep-mb-h"
      title="Results per mailbox provider"
      caption="where recipients read it"
      className="flex-[1_1_560px]"
      footer={icloud ? "iCloud's machine opens are Apple Mail Privacy Protection prefetches; its human-open figure is the honest one." : null}
    >
      <table className="tbl rep-tbl phone-wrap">
        <thead>
          <tr>
            <th scope="col">Mailbox provider</th>
            <th scope="col" className="r">
              Delivered
            </th>
            <th scope="col" className="r">
              Human opens
            </th>
            <th scope="col" className="r">
              Machine opens
            </th>
            <th scope="col" className="r">
              Clicks
            </th>
            <th scope="col" className="r">
              Bounces
            </th>
            <th scope="col" className="r">
              Complaints
            </th>
            <th scope="col" className="r">
              Unsubs
            </th>
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

function ProviderTable({ t }: { t: Schemas['ReportProviderTable'] }) {
  const cols = t.columns
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
    <TableCard id="rep-sp-h" title="Results per sending provider" caption={splitCaption(t)} className="flex-[1_1_440px]" footer={providerNote(t)}>
      <table className="tbl rep-tbl phone-wrap">
        <thead>
          <tr>
            <th scope="col">Metric</th>
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
          {row('Sent', (c) => count(c.sent))}
          {row('Delivered', (c) => pct1(c.deliveredRate))}
          {row('Human opens', (c) => pct1(c.humanOpenRate))}
          {row('Clicks', (c) => pct1(c.clickRate))}
          {row('Bounces', (c) => pct1(c.bounceRate))}
          {row('Complaints', (c) => pct2(c.complaintRate))}
          {row('Median time to delivery', (c) => seconds(c.medianDeliveryMs))}
          {row('Cost', (c) => money(c.cost))}
        </tbody>
      </table>
    </TableCard>
  )
}

const bounceTone: Record<string, Tone> = { bad: 'bad', warn: 'warn', neutral: 'neutral' }

function SuppressionCell({ row, actions }: { row: BounceRow; actions?: ReportActions }) {
  switch (row.suppression) {
    case 'auto':
      return <Badge tone="ok">Auto-suppressed</Badge>
    case 'retry':
      return <Badge tone="neutral">Retry next send</Badge>
    case 'suppressed':
      return <Badge tone="ok">Suppressed</Badge>
    case 'add':
      return actions ? (
        <>
          <Button size="sm" className="rep-noprint" disabledReason={actions.suppressReason} onClick={() => actions.onSuppress(row)}>
            Add {count(row.unsuppressed)} to suppression
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
            Review
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
  return (
    <Badge tone="neutral" className={print ? 'rep-printonly' : undefined}>
      Not suppressed
    </Badge>
  )
}

function BounceTable({ b, headlineBounces, actions }: { b: Schemas['ReportBounceTable']; headlineBounces: number; actions?: ReportActions }) {
  return (
    <TableCard id="rep-bt-h" title={`Bounces by type · ${count(headlineBounces)}`} caption="each provider's original code mapped to one taxonomy" className="flex-[2_1_560px]">
      {b.rows.length ? (
        <table className="tbl rep-tbl phone-wrap">
          <thead>
            <tr>
              <th scope="col">ReputeCast type</th>
              <th scope="col" className="r">
                Count
              </th>
              <th scope="col">Original codes</th>
              <th scope="col">Suppression</th>
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
          {headlineBounces > 0
            ? 'Bounce details by type and code weren’t part of this campaign’s import; the count above is the imported total.'
            : 'Nothing bounced.'}
        </p>
      )}
    </TableCard>
  )
}

const reasonColor: Record<string, string> = { ink: '#0B1220', muted: '#8b94a3', bad: '#B91C1C' }

function Unsubscribes({ u, imported }: { u: Schemas['ReportUnsubscribes']; imported: boolean }) {
  const note = fewerNote(u.choseFewer)
  return (
    <section className="card flex min-w-0 flex-[1_1_320px] flex-col gap-2.5 px-5 py-4.5" aria-labelledby="rep-un-h">
      <div className="flex items-center justify-between max-md:flex-wrap">
        <h2 className="h2" id="rep-un-h">
          Unsubscribe reasons · {count(u.total)}
        </h2>
        {imported && !u.reasons.length ? null : <span className="k">asked on the unsubscribe page</span>}
      </div>
      {u.reasons.length ? (
        <div className="flex flex-col gap-2 text-[13px]">
          {u.reasons.map((r) => (
            <div key={r.key}>
              <div className="flex justify-between max-md:flex-wrap">
                <span>{r.label}</span>
                <span className="n">{count(r.count)}</span>
              </div>
              <div className="bar" role="img" aria-label={`${r.label}: ${Math.round(r.share)}% of unsubscribes`}>
                <i style={{ width: `${Math.round(r.share)}%`, background: reasonColor[r.tone] }} />
              </div>
            </div>
          ))}
        </div>
      ) : (
        <span className="k">
          {u.total === 0
            ? 'Nobody unsubscribed from this campaign.'
            : imported
              ? 'Unsubscribe reasons weren’t part of this campaign’s import; the count above is the imported total.'
              : 'Nobody has said why yet.'}
        </span>
      )}
      {note ? <span className="k">{note}</span> : null}
    </section>
  )
}

function ABTestCard({ t }: { t: Schemas['ReportABTest'] }) {
  const badge = abBadge(t)
  const lines = abExplanation(t)
  return (
    <section className="card flex flex-col" aria-labelledby="rep-ab-h">
      <div className="sec">
        <h2 className="h2" id="rep-ab-h">
          A/B test · {t.label}
        </h2>
        <Badge tone={badge.tone} dot>
          {badge.text}
        </Badge>
      </div>
      <Grid min={300} gap={0}>
        {t.variants.map((v) => (
          <div key={v.key} className={cn('flex flex-col gap-2 border-r border-line-2 px-4.5 py-4', v.winner && 'bg-ok-tint')}>
            <div className="flex items-center justify-between gap-2.5 max-md:flex-wrap">
              <Badge tone={v.winner ? 'ok' : 'neutral'}>{v.winner ? `${v.key} · winner` : v.key}</Badge>
              <span className="k">{variantLine(v)}</span>
            </div>
            <span className="font-semibold">{v.label}</span>
            <div className="flex gap-4 text-[13px] max-md:flex-wrap">
              <span>
                <b className="n">{pct1(v.humanOpenRate)}</b> human opens
              </span>
              <span>
                <b className="n">{pct1(v.clickRate)}</b> clicks
              </span>
              <span>
                <b className="n">{pct2(v.complaintRate)}</b> complaints
              </span>
            </div>
            <div className="bar h-2" aria-hidden="true">
              <i style={{ width: `${v.humanOpenRate ?? 0}%`, background: v.winner ? 'var(--accent)' : '#8b94a3' }} />
            </div>
          </div>
        ))}
        <div className="flex flex-col gap-1.5 px-4.5 py-4 text-[13px]">
          <span className="font-semibold">{t.state === 'testing' ? 'How the winner is picked' : 'How the winner was picked'}</span>
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

/** Everything below the header: the report's cards in the board's order. */
export function ReportBody({ r, actions }: { r: CampaignReport; actions?: ReportActions }) {
  return (
    <>
      <HeadlineTiles r={r} />
      <div className="flex flex-wrap items-stretch gap-5">
        <EngagementChart e={r.engagement} startedAt={r.campaign.startedAt} final={r.campaign.final} />
        <ClickMap m={r.clickMap} clicks={r.headline.clicks} imported={Boolean(r.campaign.origin)} />
      </div>
      <div className="flex flex-wrap items-start gap-5">
        <MailboxTable rows={r.mailboxes} />
        {r.providers.columns.length ? <ProviderTable t={r.providers} /> : null}
      </div>
      <div className="flex flex-wrap items-start gap-5">
        <BounceTable b={r.bounces} headlineBounces={r.headline.bounces} actions={actions} />
        <Unsubscribes u={r.unsubscribes} imported={Boolean(r.campaign.origin)} />
      </div>
      {r.abTest ? <ABTestCard t={r.abTest} /> : null}
    </>
  )
}
