import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { useNow } from '@/lib/hooks'
import { count, money, until } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { OverviewIssueCard } from '@/features/digests/IssueParts'
import { useT } from '@/i18n'
import { m } from './messages'
import { clockIn, dateIn, minutes, readinessBadge, routeLine } from './words'

/** "Next scheduled campaign" with its Send Readiness, route and estimates (Main board). */
export function NextCampaignCard({ n }: { n: Schemas['OverviewNextCampaign'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const now = useNow(60_000)
  const editReason = usePermissionReason('campaigns:edit')
  const c = n.campaign
  // A recurring digest's issue that goes out before the next scheduled campaign takes the card.
  if (n.issue && (!c || new Date(n.issue.sendAt) < new Date(c.scheduledAt))) return <OverviewIssueCard issue={n.issue} />
  if (!c) {
    return (
      <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-next-h">
        <h2 className="h2" id="ov-next-h">
          {t(m.nextCampaign)}
        </h2>
        <p className="m-0 text-ink-2">
          {t(m.nothingScheduled)} {n.drafts > 0 ? t(m.draftsWaiting, { n: n.drafts, count: count(n.drafts) }) : t(m.scheduleOne)}
        </p>
        <div className="flex flex-wrap gap-2.5">
          {n.drafts > 0 ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns" params={{ workspace: slug }} search={{ status: 'drafts' }}>
              {t(m.openDrafts)}
            </ButtonLink>
          ) : editReason ? (
            <Button variant="primary" disabledReason={editReason}>
              {t(m.newCampaign)}
            </Button>
          ) : (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }}>
              {t(m.newCampaign)}
            </ButtonLink>
          )}
        </div>
      </section>
    )
  }
  const params = { workspace: slug, campaignId: c.id }
  const badge = readinessBadge(c.readiness)
  const issues = c.readiness?.issues ?? []
  const blocked = issues.some((i) => i.status === 'block')
  const when = `${dateIn(c.scheduledAt, c.timezone)} · ${clockIn(c.scheduledAt, c.timezone)} ${c.timezone}${new Date(c.scheduledAt) > now ? ` · ${until(c.scheduledAt, now)}` : ''}`
  return (
    <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-next-h">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="h2" id="ov-next-h">
          {t(m.nextCampaign)}
        </h2>
        <Badge tone={badge.tone} dot>
          {badge.text}
        </Badge>
      </div>
      <div className="flex flex-col gap-1">
        <Link to="/w/$workspace/campaigns/$campaignId/edit" params={params} className="text-[18px] font-semibold text-ink no-underline hover:text-ink">
          {c.name}
        </Link>
        <span className="k">{when}</span>
      </div>
      <div className="grid gap-3 [grid-template-columns:repeat(4,minmax(0,1fr))] max-md:[grid-template-columns:repeat(auto-fit,minmax(140px,1fr))]">
        <div>
          <div className="k">{t(m.recipients)}</div>
          <div className="n text-[18px]">{count(c.recipients)}</div>
        </div>
        <div>
          <div className="k">{t(m.route)}</div>
          <div className="n text-[18px]">{routeLine(c.routes)}</div>
        </div>
        <div>
          <div className="k">{t(m.estCost)}</div>
          <div className="n text-[18px]">{money(c.cost)}</div>
        </div>
        <div>
          <div className="k">{t(m.estDuration)}</div>
          <div className="n text-[18px]">{minutes(c.durationMinutes)}</div>
        </div>
      </div>
      {issues.length ? (
        <div
          className={
            blocked
              ? 'flex flex-col gap-1.5 rounded-lg border border-[#FECACA] bg-[#FEF2F2] p-3 text-[13px]'
              : 'flex flex-col gap-1.5 rounded-lg border border-[#FDE68A] bg-[#FFFBEB] p-3 text-[13px]'
          }
        >
          {issues.map((i, k) => (
            <div key={k} className="flex gap-2 max-md:flex-wrap">
              <Badge tone={i.status === 'block' ? 'bad' : 'warn'} className="h-5">
                {i.status === 'block' ? t(m.issueBlock) : t(m.issueWarn)}
              </Badge>
              <span>{i.title}</span>
            </div>
          ))}
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2.5">
        <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={params}>
          {t(m.openReadiness)}
        </ButtonLink>
        {/* The composer opens read-only for members who can't edit campaigns. */}
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={params}>
          {editReason ? t(m.viewCampaign) : t(m.editCampaign)}
        </ButtonLink>
      </div>
    </section>
  )
}

/** First run: nothing to send yet (OverviewEmpty). */
export function PendingNextCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const editReason = usePermissionReason('campaigns:edit')
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-5 py-4.5" aria-labelledby="ov-next-p">
      <h3 className="h2 text-ink-3" id="ov-next-p">
        {t(m.nextCampaign)}
      </h3>
      <span className="k">{t(m.nextPending)}</span>
      {/* PR-13: a campaign can be written before a provider is connected. */}
      {editReason ? (
        <Button size="sm" className="self-start max-md:min-h-11" disabledReason={editReason}>
          {t(m.newCampaign)}
        </Button>
      ) : (
        <ButtonLink size="sm" className="self-start max-md:min-h-11" to="/w/$workspace/campaigns/new" params={{ workspace: slug }}>
          {t(m.newCampaign)}
        </ButtonLink>
      )}
    </section>
  )
}
