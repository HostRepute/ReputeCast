import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { useNow } from '@/lib/hooks'
import { count, money, until } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { clockIn, dateIn, minutes, readinessBadge, routeLine } from './words'

/** "Next scheduled campaign" with its Send Readiness, route and estimates (Main board). */
export function NextCampaignCard({ n }: { n: Schemas['OverviewNextCampaign'] }) {
  const { slug } = useWorkspace()
  const now = useNow(60_000)
  const editReason = usePermissionReason('campaigns:edit')
  const c = n.campaign
  if (!c) {
    return (
      <section className="card flex flex-col gap-3.5 px-[22px] py-5" aria-labelledby="ov-next-h">
        <h2 className="h2" id="ov-next-h">
          Next scheduled campaign
        </h2>
        <p className="m-0 text-ink-2">
          Nothing is scheduled.{' '}
          {n.drafts > 0
            ? `${count(n.drafts)} ${n.drafts === 1 ? 'draft is' : 'drafts are'} waiting; schedule one and its readiness, route and cost show here.`
            : 'Schedule a campaign and its readiness, route and cost show here.'}
        </p>
        <div className="flex flex-wrap gap-2.5">
          {n.drafts > 0 ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns" params={{ workspace: slug }} search={{ status: 'drafts' }}>
              Open drafts
            </ButtonLink>
          ) : editReason ? (
            <Button variant="primary" disabledReason={editReason}>
              New campaign
            </Button>
          ) : (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={{ workspace: slug }}>
              New campaign
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
          Next scheduled campaign
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
          <div className="k">Recipients</div>
          <div className="n text-[18px]">{count(c.recipients)}</div>
        </div>
        <div>
          <div className="k">Route</div>
          <div className="n text-[18px]">{routeLine(c.routes)}</div>
        </div>
        <div>
          <div className="k">Est. cost</div>
          <div className="n text-[18px]">{money(c.cost)}</div>
        </div>
        <div>
          <div className="k">Est. duration</div>
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
                {i.status === 'block' ? 'Block' : 'Warn'}
              </Badge>
              <span>{i.title}</span>
            </div>
          ))}
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2.5">
        <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={params}>
          Open Send Readiness
        </ButtonLink>
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={params}>
          Edit campaign
        </ButtonLink>
      </div>
    </section>
  )
}

/** First run: nothing to send yet (OverviewEmpty). */
export function PendingNextCard() {
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-5 py-4.5" aria-labelledby="ov-next-p">
      <h3 className="h2 text-ink-3" id="ov-next-p">
        Next scheduled campaign
      </h3>
      <span className="k">No campaigns yet. The composer unlocks when a provider is assigned to the Broadcasts stream.</span>
    </section>
  )
}
