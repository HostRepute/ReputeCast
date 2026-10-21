import { $api, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { ButtonAnchor } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Table, Td, Th } from '@/components/ui/Table'
import { ReportError } from '@/features/push/ReportPage'
import { m as pm } from '@/features/push/report.messages'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, dayMonthYear, money, pct } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { throwIfForbidden } from './common'
import { m } from './campaign.messages'
import '@/features/push/push.css'
import './whatsapp.css'

type Report = Schemas['WhatsAppReport']

/** A WhatsApp campaign's report: sent, delivered, read, failed with Meta's codes, replies, opt-outs and cost. */
export function WhatsAppReportPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/report', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  const r = q.data
  return (
    <Page
      title={t(m.reportTitle)}
      documentTitle={r ? t(m.docTitleReport, { name: r.name }) : t(m.reportTitle)}
      breadcrumb={[{ label: t(m.campaignsCrumb), to: '/w/$workspace/campaigns', params: { workspace: slug } }, { label: r?.name ?? t(m.reportTitle) }]}
    >
      {r ? (
        <Body r={r} />
      ) : q.isError ? (
        <ReportError error={q.error} onRetry={() => void q.refetch()} />
      ) : (
        <Loading label={t(m.loading)}>
          <Skeleton w="100%" h={320} />
        </Loading>
      )}
    </Page>
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

function Body({ r }: { r: Report }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <>
      <div className="card flex flex-wrap items-center gap-3 px-[18px] py-3.5">
        <span className="flex min-w-0 flex-[1_1_360px] flex-col gap-0.5">
          <span className="text-[15px] font-bold">{r.name}</span>
          <span className="k">
            {r.sentAt ? t(m.sentLine, { date: `${dayMonthDate(r.sentAt)}, ${clock(r.sentAt)}`, name: r.sentBy, template: r.template }) : t(m.notSentLine, { template: r.template })}
          </span>
        </span>
        <Badge tone={r.status === 'sent' ? 'ok' : r.status === 'failed' ? 'bad' : 'info'} dot>
          {t(
            r.status === 'sent'
              ? pm.statusSent
              : r.status === 'failed'
                ? pm.statusFailed
                : r.status === 'cancelled'
                  ? pm.statusCancelled
                  : r.status === 'scheduled'
                    ? pm.statusScheduled
                    : r.status === 'sending' || r.status === 'paused'
                      ? pm.statusSending
                      : pm.statusDraft,
          )}
        </Badge>
        <ButtonAnchor size="sm" href={`/api/v1/workspaces/${encodeURIComponent(slug)}/whatsapp-campaigns/${r.id}/report.csv`} newTab={false}>
          {t(m.exportCsv)}
        </ButtonAnchor>
      </div>
      <div className="px-stats">
        <Tile label={t(m.tileSent)} value={count(r.sent)} sub={t(m.tileSentSub)} />
        <Tile label={t(m.tileDelivered)} value={count(r.delivered)} sub={t(m.tileDeliveredSub, { pct: pct(r.deliveredPct) })} />
        <Tile label={t(m.tileRead)} value={count(r.read)} sub={t(m.tileReadSub, { pct: pct(r.readPct) })} />
        <Tile label={t(m.tileFailed)} value={count(r.failed)} sub={t(m.tileFailedSub)} />
        <Tile label={t(m.tileReplies)} value={count(r.replies)} sub={t(m.tileRepliesSub, { buttons: r.buttons, optOuts: r.optOuts })} />
        <Tile label={t(m.tileCost)} value={money(r.cost)} sub={t(m.tileCostSub, { estimate: money(r.estimate), date: dayMonthYear(`${r.ratesAsOf.slice(0, 10)}T12:00:00Z`) })} />
      </div>
      <div className="px-grid">
        <section className="card">
          <div className="sec">
            <h2 className="h2">{t(m.byCountry)}</h2>
          </div>
          <Table aria-label={t(m.byCountry)}>
            <thead>
              <tr>
                <Th>{t(m.colCountry)}</Th>
                <Th align="right">{t(m.colSent)}</Th>
                <Th align="right">{t(m.colDelivered)}</Th>
                <Th align="right">{t(m.colRead)}</Th>
                <Th align="right">{t(m.colFailed)}</Th>
                <Th align="right">{t(m.colCost)}</Th>
              </tr>
            </thead>
            <tbody>
              {r.byCountry.map((c) => (
                <tr key={c.country}>
                  <Td className="mono">{c.country}</Td>
                  <Td num align="right">
                    {count(c.sent)}
                  </Td>
                  <Td num align="right">
                    {count(c.delivered)}
                  </Td>
                  <Td num align="right">
                    {count(c.read)}
                  </Td>
                  <Td num align="right">
                    {count(c.failed)}
                  </Td>
                  <Td num align="right">
                    {money(c.cost)}
                  </Td>
                </tr>
              ))}
            </tbody>
          </Table>
        </section>
        <section className="card">
          <div className="sec">
            <h2 className="h2">{t(m.failures)}</h2>
          </div>
          {r.failures.length ? (
            <Table aria-label={t(m.failures)}>
              <thead>
                <tr>
                  <Th>{t(m.colCode)}</Th>
                  <Th>{t(m.colReason)}</Th>
                  <Th align="right">{t(m.colCount)}</Th>
                </tr>
              </thead>
              <tbody>
                {r.failures.map((f) => (
                  <tr key={f.code}>
                    <Td className="mono">{f.code}</Td>
                    <Td wrap>{f.reason}</Td>
                    <Td num align="right">
                      {count(f.count)}
                    </Td>
                  </tr>
                ))}
              </tbody>
            </Table>
          ) : (
            <div className="pad">
              <span className="k">{t(m.noFailures)}</span>
            </div>
          )}
        </section>
      </div>
      <div className="note n" role="note">
        <div>
          <T msg={m.readNote} values={{ b: (c) => <b>{c}</b> }} />
        </div>
      </div>
    </>
  )
}
