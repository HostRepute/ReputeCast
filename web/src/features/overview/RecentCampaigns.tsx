import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Table, Td, Th } from '@/components/ui/Table'
import { count, pct, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { RouteBadges } from '../campaigns/RouteBadges'
import { CampaignStatusBadge } from '../campaigns/StatusBadge'
import { ChannelTag } from '@/features/channels/ChannelFilter'

/** "Recent campaigns": the latest sends, as the campaigns list shows them (Main board). */
export function RecentCampaigns({ items }: { items: Schemas['CampaignListItem'][] }) {
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col" aria-labelledby="ov-recent-h">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line-2 px-5 py-4">
        <h2 className="h2" id="ov-recent-h">
          Recent campaigns
        </h2>
        <Link to="/w/$workspace/campaigns" params={{ workspace: slug }} className="text-[13px] font-medium">
          All campaigns
        </Link>
      </div>
      {items.length ? (
        <Table phoneWrap>
          <thead>
            <tr>
              <Th>Campaign</Th>
              <Th>Sent</Th>
              <Th>Recipients</Th>
              <Th>Delivered</Th>
              <Th>Human opens</Th>
              <Th>Clicks</Th>
              <Th>Sending providers</Th>
              <Th>Status</Th>
            </tr>
          </thead>
          <tbody>
            {items.map((c) => {
              const live = c.status === 'sending' || c.status === 'paused'
              return (
                <tr key={c.id}>
                  <Td>
                    <Link
                      to={
                        // Phase 8: a text campaign's report is the SMS report.
                        c.channel === 'sms' ? '/w/$workspace/campaigns/$campaignId/sms-report' : live ? '/w/$workspace/campaigns/$campaignId/live' : '/w/$workspace/campaigns/$campaignId/report'
                      }
                      params={{ workspace: slug, campaignId: c.id }}
                      className="font-semibold text-ink no-underline hover:text-ink"
                    >
                      {c.name}
                    </Link>
                    <div className="k">
                      {c.channel === 'sms' ? <ChannelTag channel="sms" /> : null}
                      {c.audience || (c.origin ? `imported from ${c.origin}` : '—')}
                    </div>
                  </Td>
                  <Td>{c.startedAt ? shortDate(c.startedAt) : '—'}</Td>
                  <Td num>{c.recipients != null ? count(c.recipients) : '—'}</Td>
                  <Td num>{c.delivered != null ? pct(c.delivered) : '—'}</Td>
                  <Td num>{c.humanOpens != null ? pct(c.humanOpens) : '—'}</Td>
                  <Td num>{c.clicks != null ? pct(c.clicks) : '—'}</Td>
                  <Td>{c.origin ? <span className="k">{c.origin[0]!.toUpperCase() + c.origin.slice(1)} · imported</span> : <RouteBadges routes={c.routes} sent />}</Td>
                  <Td>{c.status === 'sent' ? <Badge tone="ok">Sent</Badge> : <CampaignStatusBadge status={c.status} blocked={c.blocked} />}</Td>
                </tr>
              )
            })}
          </tbody>
        </Table>
      ) : (
        <p className="k m-0 px-5 py-4">Campaigns appear here once they go out, with their delivery, human opens and clicks.</p>
      )}
    </section>
  )
}
