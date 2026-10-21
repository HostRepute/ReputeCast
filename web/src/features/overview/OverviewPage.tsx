import type { ReactNode } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Grid } from '@/components/ui/Layout'
import { useNow } from '@/lib/hooks'
import { clock } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { Link } from '@tanstack/react-router'
import { StateAlert } from '../reports/StateAlert'
import { GrowthCard, TiersCard } from './AudienceCards'
import { FirstRun } from './FirstRun'
import { NextCampaignCard } from './NextCampaignCard'
import { OverviewSearch } from './OverviewSearch'
import { ProviderOverviewCard } from './ProviderCards'
import { RecentCampaigns } from './RecentCampaigns'
import { SenderHealthCard } from './SenderHealthCard'
import { HealthSkeleton, NextSkeleton, ProvidersSkeleton, CardSkeleton } from './Skeletons'
import { overviewSubtitle, refreshSentences } from './words'
import { TransactionalOverviewCard } from '@/features/transactional/OverviewCard'
import { EarnOverviewCard } from '@/features/earn/revenue/EarnCard'

/** A card whose data failed to load: what failed and a retry, the rest of the page stays. */
function CardError({ title, error, onRetry }: { title: string; error: unknown; onRetry: () => void }) {
  return (
    <section className="card flex flex-col gap-2 border-bad-line px-[22px] py-5" role="alert">
      <h2 className="h2 text-bad-fg">{title}</h2>
      <span className="text-[13px] text-ink-2">{errorMessage(error)} The rest of the Overview is current.</span>
      <div>
        <Button size="sm" onClick={onRetry}>
          Retry
        </Button>
      </div>
    </section>
  )
}

/** A card the member's role can't see: why, instead of a gap. */
function CardDenied({ title, perm }: { title: string; perm: string }) {
  const reason = usePermissionReason(perm)
  return (
    <section className="card flex flex-col gap-2 border-dashed px-[22px] py-5">
      <h2 className="h2 text-ink-3">{title}</h2>
      <span className="k">{reason}</span>
    </section>
  )
}

function Section<T>({
  q,
  perm,
  title,
  skeleton,
  children,
}: {
  q: { data?: T; isError: boolean; error: unknown; refetch: () => unknown }
  perm: string
  title: string
  skeleton: ReactNode
  children: (data: T) => ReactNode
}) {
  const can = useCan(perm)
  if (!can) return <CardDenied title={title} perm={perm} />
  if (q.data !== undefined) return <>{children(q.data)}</>
  if (q.isError) return <CardError title={`Couldn't load ${title.toLowerCase()}`} error={q.error} onRetry={() => void q.refetch()} />
  return <>{skeleton}</>
}

/** 7.1 Overview: sender health, the next send, providers, growth, tiers and recent campaigns. */
export function OverviewPage() {
  const { slug } = useWorkspace()
  const now = useNow(60_000)
  const path = { workspace: slug }
  const can = { health: useCan('deliverability:view'), campaigns: useCan('campaigns:view'), providers: useCan('providers:view'), audience: useCan('subscribers:view') }
  const opts = (enabled: boolean) => ({ enabled, refetchInterval: 60_000 })
  const health = $api.useQuery('get', '/api/v1/workspaces/{workspace}/overview/sender-health', { params: { path } }, opts(can.health))
  const next = $api.useQuery('get', '/api/v1/workspaces/{workspace}/overview/next-campaign', { params: { path } }, opts(can.campaigns))
  const providers = $api.useQuery('get', '/api/v1/workspaces/{workspace}/overview/providers', { params: { path } }, opts(can.providers))
  const audience = $api.useQuery('get', '/api/v1/workspaces/{workspace}/overview/audience', { params: { path, query: { days: 30 } } }, opts(can.audience))
  const recent = $api.useQuery('get', '/api/v1/workspaces/{workspace}/overview/recent-campaigns', { params: { path } }, opts(can.campaigns))
  const editReason = usePermissionReason('campaigns:edit')

  const firstRun = providers.data?.connected === 0
  const providersLoading = providers.isPending && can.providers
  const subtitle = overviewSubtitle(now, health.data?.domain, health.data?.state === 'pending' || (firstRun && !health.data?.domain))
  const stale = providers.data?.providers.find((p) => p.refresh)
  const alert = stale ? refreshSentences(stale, clock) : null

  const newCampaign = firstRun ? (
    <Button variant="primary" disabledReason="Connect a sending provider first: the composer unlocks when a provider is assigned to the Broadcasts stream.">
      New campaign
    </Button>
  ) : editReason ? (
    <Button variant="primary" disabledReason={editReason}>
      New campaign
    </Button>
  ) : (
    <ButtonLink variant="primary" to="/w/$workspace/campaigns/new" params={path}>
      New campaign
    </ButtonLink>
  )

  return (
    <Page
      title="Overview"
      subtitle={subtitle}
      contentClassName={firstRun ? 'max-w-[1080px] min-[1101px]:py-7' : undefined}
      actions={
        <>
          <OverviewSearch />
          {newCampaign}
        </>
      }
    >
      {firstRun ? (
        <FirstRun audience={audience.data} />
      ) : (
        <>
          {alert && stale ? (
            <StateAlert
              title={alert.title}
              actions={
                <>
                  <Button size="sm" onClick={() => void providers.refetch()}>
                    Retry now
                  </Button>
                  <ButtonLink size="sm" to="/w/$workspace/providers/events" params={path} search={{ provider: stale.card.id }}>
                    View event log
                  </ButtonLink>
                  {stale.statusUrl ? (
                    <ButtonAnchor size="sm" href={stale.statusUrl}>
                      {stale.statusLabel}
                    </ButtonAnchor>
                  ) : null}
                </>
              }
            >
              {alert.body}
            </StateAlert>
          ) : null}
          <Grid min={380} gap={20}>
            <Section q={health} perm="deliverability:view" title="Sender health" skeleton={<HealthSkeleton />}>
              {(h) => <SenderHealthCard h={h} />}
            </Section>
            <Section q={next} perm="campaigns:view" title="Next scheduled campaign" skeleton={<NextSkeleton />}>
              {(n) => <NextCampaignCard n={n} />}
            </Section>
          </Grid>
          <section
            className="flex flex-col gap-3"
            aria-labelledby={providersLoading ? undefined : 'ov-providers-h'}
            aria-label={providersLoading ? 'Sending providers' : undefined}
          >
            {/* While the cards load, only their skeletons show (OverviewStates). */}
            {providersLoading ? null : (
              <div className="flex flex-wrap items-center justify-between gap-3">
                <h2 className="h2" id="ov-providers-h">
                  Sending providers
                </h2>
                <Link to="/w/$workspace/providers" params={path} className="text-[13px] font-medium">
                  Manage providers &amp; routing
                </Link>
              </div>
            )}
            <Section q={providers} perm="providers:view" title="Sending providers" skeleton={<ProvidersSkeleton />}>
              {(p) => (
                <Grid min={340} gap={16}>
                  {p.providers.map((c) => (
                    <ProviderOverviewCard key={c.card.id} p={c} monthStart={p.monthStart} />
                  ))}
                </Grid>
              )}
            </Section>
            {health.isPending && providers.isPending ? (
              <p className="k m-0">Checking blocklists and provider APIs… usually under 5 seconds.</p>
            ) : null}
          </section>
          <TransactionalOverviewCard />
          <EarnOverviewCard />
          <Grid min={380} gap={20}>
            <Section q={audience} perm="subscribers:view" title="Subscriber growth" skeleton={<CardSkeleton lines={5} />}>
              {(a) => <GrowthCard a={a} />}
            </Section>
            <Section q={audience} perm="subscribers:view" title="Engagement tiers" skeleton={<CardSkeleton lines={5} />}>
              {(a) => <TiersCard a={a} />}
            </Section>
          </Grid>
          <Section q={recent} perm="campaigns:view" title="Recent campaigns" skeleton={<CardSkeleton lines={4} />}>
            {(r) => <RecentCampaigns items={r.items} />}
          </Section>
        </>
      )}
    </Page>
  )
}
