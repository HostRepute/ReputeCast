import { useNavigate } from '@tanstack/react-router'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { andList, clock, count, dayMonthDate, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { browserName, throwIfForbidden } from './common'
import { InfoIcon } from './SetupCards'
import { m } from './report.messages'
import './push.css'

type Report = Schemas['PushReport']

const SITE_COLORS = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#8b94a3']

const statusBadge: Partial<Record<Report['status'], { tone: Tone; msg: MessageDescriptor }>> = {
  sent: { tone: 'ok', msg: m.statusSent },
  sending: { tone: 'info', msg: m.statusSending },
  scheduled: { tone: 'info', msg: m.statusScheduled },
  cancelled: { tone: 'neutral', msg: m.statusCancelled },
  failed: { tone: 'bad', msg: m.statusFailed },
}

/**
 * A push campaign's report (brief §5.12): sent, shown, clicks,
 * unsubscribed within 24 hours, expired subscriptions removed, by browser
 * and by site, and the clicks over the first 24 hours.
 */
export function PushReportPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/report', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  const r = q.data
  return (
    <Page
      title={t(m.title)}
      documentTitle={r ? t(m.docTitle, { name: r.name }) : t(m.title)}
      breadcrumb={[{ label: t(m.campaignsCrumb), to: '/w/$workspace/campaigns', params: { workspace: slug } }, { label: r?.name ?? t(m.title) }]}
    >
      {r ? <Report r={r} /> : q.isError ? <ReportError error={q.error} onRetry={() => void q.refetch()} /> : <ReportSkeleton />}
    </Page>
  )
}

function Report({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const dup = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/duplicate')
  const badge = statusBadge[r.status]
  const sites = andList(r.sites)
  const ttl = r.ttlMinutes % 60 === 0 ? t(m.hours, { n: r.ttlMinutes / 60 }) : t(m.minutes, { n: r.ttlMinutes })
  const date = r.sentAt ? t(m.sentAt, { date: dayMonthDate(r.sentAt), time: clock(r.sentAt) }) : ''
  const line =
    r.status === 'sent' && r.sentAt
      ? t(r.tag ? m.sentLine : m.sentLineNoTag, { date, name: r.sentBy, sites, tag: r.tag, ttl })
      : r.status === 'sending' && r.sentAt
        ? t(m.sendingLine, { date, sites })
        : t(m.notSentLine, { sites })
  const skipped = Object.entries(r.skipped).filter(([, n]) => n > 0)
  return (
    <>
      <div className="card flex flex-wrap items-center gap-3 px-[18px] py-3.5">
        <span className="flex min-w-0 flex-[1_1_360px] flex-col gap-0.5">
          <span className="text-[15px] font-bold">{r.name}</span>
          <span className="k">{line}</span>
        </span>
        {badge ? (
          <Badge tone={badge.tone} dot>
            {t(badge.msg)}
          </Badge>
        ) : null}
        <ButtonAnchor size="sm" href={`/api/v1/workspaces/${encodeURIComponent(slug)}/push-campaigns/${r.id}/report.csv`} newTab={false}>
          {t(m.exportCsv)}
        </ButtonAnchor>
        {r.canSend ? (
          <Button
            size="sm"
            loading={dup.isPending}
            busyLabel={t(m.duplicating)}
            onClick={() =>
              dup.mutate(
                { params: { path: { workspace: slug, campaignId: r.id } } },
                {
                  onSuccess: (c) => void navigate({ to: '/w/$workspace/campaigns/$campaignId/push', params: { workspace: slug, campaignId: c.id } }),
                  onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                },
              )
            }
          >
            {t(m.duplicate)}
          </Button>
        ) : null}
      </div>
      <div className="px-stats">
        <Tile label={t(m.tileSent)} value={count(r.sent)} sub={t(m.tileSentSub)} />
        <Tile label={t(m.tileShown)} value={count(r.shown)} sub={t(m.tileShownSub, { pct: pct(r.shownPct) })} />
        <Tile label={t(m.tileClicks)} value={count(r.clicks)} sub={t(m.tileClicksSub, { pct: pct(r.clicksPct) })} />
        <Tile label={t(m.tileUnsub)} value={count(r.unsubscribed)} sub={pct(r.unsubscribedPct, 2)} />
        <Tile label={t(m.tileExpired)} value={count(r.expired)} sub={t(m.tileExpiredSub)} />
        {r.held ? <Tile label={t(m.tileHeld)} value={count(r.held)} sub={t(m.tileHeldSub)} /> : null}
      </div>
      {skipped.length ? (
        <div className="flex flex-wrap gap-2">
          {skipped.map(([reason, n]) => (
            <span key={reason} className="chip">
              {t(reason === 'frequency_cap' ? m.skippedCap : reason === 'quiet_hours' ? m.skippedQuiet : reason === 'worker_missing' ? m.skippedWorker : m.skippedOther, {
                n: count(n),
              })}
            </span>
          ))}
        </div>
      ) : null}
      <div className="px-grid">
        <section className="card">
          <div className="sec">
            <h2 className="h2">{t(m.byBrowser)}</h2>
          </div>
          <Table aria-label={t(m.byBrowser)}>
            <thead>
              <tr>
                <Th>{t(m.colBrowser)}</Th>
                <Th align="right">{t(m.colSent)}</Th>
                <Th align="right">{t(m.colShown)}</Th>
                <Th align="right">{t(m.colShownRate)}</Th>
                <Th align="right">{t(m.colClicks)}</Th>
              </tr>
            </thead>
            <tbody>
              {r.byBrowser.map((b) => (
                <tr key={b.browser}>
                  <Td>{t(browserName(b.browser, true))}</Td>
                  <Td num align="right">
                    {count(b.sent)}
                  </Td>
                  <Td num align="right">
                    {count(b.shown)}
                  </Td>
                  <Td num align="right">
                    {pct(b.shownPct)}
                  </Td>
                  <Td num align="right">
                    {count(b.clicks)}
                  </Td>
                </tr>
              ))}
              <tr>
                <Td className="font-bold">{t(m.total)}</Td>
                <Td num align="right" className="font-bold">
                  {count(r.sent)}
                </Td>
                <Td num align="right" className="font-bold">
                  {count(r.shown)}
                </Td>
                <Td num align="right" className="font-bold">
                  {pct(r.shownPct)}
                </Td>
                <Td num align="right" className="font-bold">
                  {count(r.clicks)}
                </Td>
              </tr>
            </tbody>
          </Table>
          <div className="px-foot">
            <span className="k">{t(m.shownFoot)}</span>
          </div>
        </section>
        <div className="flex min-w-0 flex-col gap-4">
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(m.bySite)}</h2>
            </div>
            <div className="pad flex flex-col gap-2.5">
              <div className="flex flex-col gap-2.5">
                <div className="px-split" role="img" aria-label={r.bySite.map((s) => t(m.siteLegend, { host: s.host, count: count(s.sent) })).join(', ')}>
                  {r.bySite.map((s, i) => (s.sent ? <i key={s.host} style={{ flex: s.sent, background: SITE_COLORS[i % SITE_COLORS.length] }} /> : null))}
                </div>
                <div className="px-legend">
                  {r.bySite.map((s, i) => (
                    <span key={s.host}>
                      <i style={{ background: SITE_COLORS[i % SITE_COLORS.length] }} />
                      {t(m.siteLegend, { host: s.host, count: count(s.sent) })}
                    </span>
                  ))}
                </div>
              </div>
              {r.sitesNotIn.length ? (
                <span className="k">{t(r.sitesNotIn.length === 1 ? m.notInAudience : m.notInAudiencePlural, { sites: andList(r.sitesNotIn) })}</span>
              ) : null}
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(m.clicksTitle)}</h2>
            </div>
            <div className="pad">{r.clicks24.length > 1 && r.clicks ? <ClicksChart r={r} /> : <span className="k">{t(m.noClicksYet)}</span>}</div>
          </section>
        </div>
      </div>
      <div className="note n" role="note">
        <InfoIcon />
        <div>
          <T msg={m.shownNote} values={{ b: (c) => <b>{c}</b> }} />
        </div>
      </div>
    </>
  )
}

function Tile({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="card px-stat">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      <span className="s">{sub}</span>
    </div>
  )
}

/** The clicks so far, hour by hour over the first 24 hours (the board's area chart). */
function ClicksChart({ r }: { r: Report }) {
  const t = useT()
  const W = 560
  const H = 170
  const x0 = 44
  const x1 = 554
  const y0 = 148
  const y1 = 8
  const max = Math.max(1, ...r.clicks24.map((p) => p.clicks))
  const x = (min: number) => x0 + ((x1 - x0) * Math.min(min, 1440)) / 1440
  const y = (v: number) => y0 - ((y0 - y1) * v) / max
  const pts = r.clicks24.map((p) => `${x(p.minutes).toFixed(1)},${y(p.clicks).toFixed(1)}`)
  const lastMin = r.clicks24[r.clicks24.length - 1]?.minutes ?? 0
  const area = `M${x0},${y0} L${pts.join(' L')} L${x(lastMin).toFixed(1)},${y0} Z`
  const ticks = [0, 0.25, 0.5, 0.75, 1].map((f) => Math.round(max * f))
  const font = 'IBM Plex Sans,system-ui,sans-serif'
  return (
    <svg width="100%" viewBox={`0 0 ${W} ${H}`} role="img" aria-label={t(m.clicksTitle)} className="block max-w-full">
      {ticks.map((v, i) => (
        <g key={i}>
          <line x1={x0} x2={x1} y1={y(v)} y2={y(v)} stroke="#EEF0F3" />
          <text x={36} y={y(v) + 4} textAnchor="end" fontSize="11" fill="#5B6472" fontFamily={font}>
            {count(v)}
          </text>
        </g>
      ))}
      <path d={area} fill="#0F766E" opacity=".10" />
      <polyline points={pts.join(' ')} fill="none" stroke="#0F766E" strokeWidth="2" strokeLinejoin="round" strokeLinecap="round" />
      <text x={x0} y={166} textAnchor="start" fontSize="11" fill="#5B6472" fontFamily={font}>
        {r.sentAt ? clock(r.sentAt) : t(m.clicksStart)}
      </text>
      <text x={(x0 + x1) / 2} y={166} textAnchor="middle" fontSize="11" fill="#5B6472" fontFamily={font}>
        {t(m.plusHours, { n: 12 })}
      </text>
      <text x={x1} y={166} textAnchor="end" fontSize="11" fill="#5B6472" fontFamily={font}>
        {t(m.plusHours, { n: 24 })}
      </text>
    </svg>
  )
}

function ReportSkeleton() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="flex flex-col gap-4">
      <div className="card flex flex-col gap-2 px-[18px] py-3.5">
        <Skeleton w={260} h={15} />
        <Skeleton w={380} h={11} />
      </div>
      <div className="px-stats">
        {[0, 1, 2].map((i) => (
          <div key={i} className="card px-stat">
            <Skeleton w={80} h={11} />
            <Skeleton w={64} h={20} className="mt-1" />
            <Skeleton w={120} h={11} className="mt-1" />
          </div>
        ))}
      </div>
      <div className="card px-pad flex flex-col gap-5">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="flex gap-3">
            <Skeleton w="28%" h={12} />
            <Skeleton w="55%" h={12} />
            <Skeleton w="12%" h={12} />
          </div>
        ))}
      </div>
    </Loading>
  )
}

/** The report's load error: the campaign was sent; its numbers are safe. */
export function ReportError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = error instanceof ApiError ? error.status : 0
  const ref = error instanceof ApiError ? (error.problem.instance?.split('/').pop() ?? '') : ''
  return (
    <section className="card flex flex-col items-center gap-2.5 p-7 text-center" role="alert">
      <Icon name="alertCircle" size={28} className="text-bad-strong" />
      <span className="text-[16px] font-bold">{t(m.errorTitle)}</span>
      <span className="k max-w-[440px] text-ink-2!">{status ? t(status >= 500 ? m.errorBody5xx : m.errorBody, { status }) : t(m.errorOffline)}</span>
      <div className="flex gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
          {t(m.campaigns)}
        </ButtonLink>
      </div>
      {ref ? <span className="k mono">{t(m.request, { id: ref })}</span> : null}
    </section>
  )
}
