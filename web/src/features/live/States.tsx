import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, Loading, Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { Banner } from '@/components/ui/Note'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import type { LivePermissions } from './actions'
import { CountUp } from './LiveParts'
import {
  cancelledText,
  continuingProviders,
  finalTiles,
  finishedText,
  incidentTitle,
  pausedText,
  providerColor,
  providerResultLines,
  scheduledLine,
  scopeName,
  startingText,
  type Incident,
  type Snapshot,
} from './model'

/** LiveMonitorStates › connecting: the send started, the first events haven't arrived. */
export function ConnectingCard({ s }: { s: Snapshot }) {
  const t = startingText(s)
  const first = s.providers[0]
  const pct = s.recipients > 0 ? Math.max(1, (s.sent / s.recipients) * 100) : 1
  return (
    <Card className="flex flex-col gap-3 px-[22px] py-5" aria-busy="true">
      <div className="flex flex-wrap items-center gap-3.5">
        <span className="spin lv-spin" aria-hidden="true" />
        <div className="flex flex-col" role="status">
          <span className="text-[18px] font-bold">{t.title}</span>
          <span className="k">{t.body}</span>
        </div>
      </div>
      <div className="bar" style={{ height: 12, borderRadius: 6 }}>
        <i style={{ width: `${Math.min(100, pct)}%`, background: first ? providerColor[first.color] : 'var(--chart-ses)' }} />
      </div>
      <Grid min={180} gap={12}>
        {(
          [
            ['Delivered', 80],
            ['Bounced', 60],
            ['Complained', 50],
            ['Opened', 70],
            ['Clicked', 60],
          ] as const
        ).map(([label, w]) => (
          <div key={label} className="flex flex-col gap-1.5">
            <span className="k">{label}</span>
            <Skeleton w={w} h={26} />
          </div>
        ))}
      </Grid>
    </Card>
  )
}

/** LiveMonitorStates › error: the browser lost the event stream. */
export function StreamLostBanner({ title, body, onReconnect, busy }: { title: string; body: string; onReconnect: () => void; busy?: boolean }) {
  return (
    <div role="alert" className="flex flex-wrap items-center gap-4 rounded-[12px] border border-bad-line bg-[#FEF2F2] px-5 py-4">
      <svg width="32" height="32" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
        <circle cx="20" cy="20" r="19" fill="#FECACA" />
        <path d="M20 11v11M20 27v1" stroke="#991B1B" strokeWidth="3" strokeLinecap="round" />
      </svg>
      <div className="flex flex-col gap-0.5 max-md:min-w-[120px]" style={{ flex: '1 1 300px' }}>
        <span className="text-[16px] font-bold text-[#7F1D1D]">{title}</span>
        <span className="text-[13px] text-[#7F1D1D]">{body}</span>
      </div>
      <Button onClick={onReconnect} loading={busy}>
        Reconnect now
      </Button>
    </div>
  )
}

/** LiveMonitorStates › complete: what the send did, and the way to the report. */
export function FinishedView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const { slug } = useWorkspace()
  const t = finishedText(s)
  const shareReason = usePermissionReason('reports:share')
  const params = { workspace: slug, campaignId }
  return (
    <>
      <div className="flex flex-wrap items-center gap-[18px] rounded-[12px] border border-[#86EFAC] bg-ok-tint px-[22px] py-[18px]" role="status">
        <svg width="40" height="40" viewBox="0 0 40 40" aria-hidden="true" className="flex-none">
          <circle cx="20" cy="20" r="19" fill="#BBF7D0" />
          <path d="m12 20.5 5.5 5.5L28 15" stroke="#166534" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" fill="none" />
        </svg>
        <div className="flex flex-col gap-0.5 max-md:min-w-[120px]" style={{ flex: '1 1 300px' }}>
          <span className="text-[20px] font-bold tracking-[-0.01em] text-[#14532D]">{t.title}</span>
          <span className="text-[#14532D]">{t.body}</span>
        </div>
        <div className="flex flex-wrap gap-2.5">
          {shareReason ? (
            <Button disabledReason={shareReason}>Share read-only link</Button>
          ) : (
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/report" params={params} hash="share">
              Share read-only link
            </ButtonLink>
          )}
          <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/report" params={params}>
            View campaign report
          </ButtonLink>
        </div>
      </div>
      <FinalTiles s={s} />
    </>
  )
}

/** The complete state's counters, cost and the per-provider results. */
export function FinalTiles({ s }: { s: Snapshot }) {
  const lines = providerResultLines(s)
  return (
    <Grid min={200} gap={16}>
      {finalTiles(s).map((t) => (
        <div key={t.key} className="card tile">
          <span className="k">{t.label}</span>
          <span className="n text-[26px]">{/^[\d,]+$/.test(t.value) ? <CountUp value={Number(t.value.replaceAll(',', ''))} /> : t.value}</span>
          <span className="k">{t.sub}</span>
        </div>
      ))}
      {lines.length ? (
        <div className="card tile">
          <span className="k">Per sending provider</span>
          {lines.map((l) => (
            <span key={l} className="n text-[16px]">
              {l}
            </span>
          ))}
        </div>
      ) : null}
    </Grid>
  )
}

/** A cancelled send: what went out before it stopped. */
export function CancelledView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const { slug } = useWorkspace()
  const t = cancelledText(s)
  const params = { workspace: slug, campaignId }
  return (
    <>
      <Banner
        tone="info"
        title={t.title}
        actions={
          s.sent > 0 ? (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/report" params={params}>
              View campaign report
            </ButtonLink>
          ) : (
            <ButtonLink variant="primary" to="/w/$workspace/campaigns" params={{ workspace: slug }}>
              Back to campaigns
            </ButtonLink>
          )
        }
      >
        {t.body}
      </Banner>
      {s.sent > 0 ? <FinalTiles s={s} /> : null}
    </>
  )
}

/** A send that stopped on an error. */
export function FailedView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const { slug } = useWorkspace()
  return (
    <>
      <Banner
        tone="bad"
        title={`${s.name} stopped before it finished`}
        actions={
          <ButtonLink to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId }}>
            View campaign report
          </ButtonLink>
        }
      >
        {s.error ? `${s.error} ` : ''}
        {count(s.sent)} of {count(s.recipients)} were sent; the rest is marked &ldquo;not sent&rdquo;. Nothing more goes out, and what was sent keeps reporting opens and clicks.
      </Banner>
      {s.sent > 0 ? <FinalTiles s={s} /> : null}
    </>
  )
}

/** A scheduled campaign: the monitor comes alive when it starts. */
export function ScheduledView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const { slug } = useWorkspace()
  return (
    <Banner
      tone="info"
      title={scheduledLine(s.scheduledAt)}
      actions={
        <ButtonLink to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
          Review &amp; send
        </ButtonLink>
      }
    >
      {s.name} goes to {count(s.recipients)} recipients. When it starts, Send Readiness runs once more; then progress, counts per mailbox provider and sending provider and the auto-pause thresholds appear here, live.
    </Banner>
  )
}

/** A draft or a campaign waiting for approval: nothing is sending. */
export function IdleView({ s, campaignId }: { s: Snapshot; campaignId: string }) {
  const { slug } = useWorkspace()
  return (
    <EmptyState
      title="Nothing is sending yet"
      action={
        <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }}>
          Review &amp; send
        </ButtonLink>
      }
    >
      {s.name} hasn&apos;t started. Once it does, this page shows its progress, the counts per mailbox provider and sending provider, the live events and the auto-pause thresholds.
    </EmptyState>
  )
}

/** A send paused by a person: who, when, why, and the way back. */
export function PausedBanner({ s, can, onResume, onCancel, busy }: { s: Snapshot; can: LivePermissions; onResume: () => void; onCancel: () => void; busy?: boolean }) {
  const t = pausedText(s)
  return (
    <Banner
      tone="warn"
      title={t.title}
      actions={
        <>
          <Button variant="primary" onClick={onResume} disabledReason={can.resume} loading={busy}>
            Resume sending
          </Button>
          <Button variant="danger" onClick={onCancel} disabledReason={can.cancel}>
            Cancel send
          </Button>
        </>
      }
    >
      {t.body}
    </Banner>
  )
}

/** An open incident seen from the live view: what happened and the way to decide. */
export function IncidentBanner({ s, incident, onDecide }: { s: Snapshot; incident: Incident; onDecide: () => void }) {
  const outage = incident.kind === 'provider_outage'
  const who = scopeName(incident)
  const others = continuingProviders(s, incident).map((p) => p.short)
  const rest = outage ? (others.length ? `${others.join(' and ')} keeps sending.` : 'Nothing else is sending.') : 'Everything else keeps sending.'
  return (
    <Banner
      tone={outage ? 'warn' : 'bad'}
      title={incidentTitle(incident)}
      actions={
        <Button variant="primary" onClick={onDecide}>
          Choose how to continue
        </Button>
      }
    >
      {count(incident.held)} {who ? `${who} ` : ''}recipients are held until someone decides. {rest}
    </Banner>
  )
}

/** First load: the live view's shape in grey. */
export function LiveSkeleton() {
  return (
    <Loading label="Loading the live monitor…" className="flex flex-col gap-5 max-md:gap-3">
      <Card className="flex flex-col gap-3 px-[22px] py-5">
        <div className="flex items-baseline gap-3.5">
          <Skeleton w={80} h={34} />
          <Skeleton w={180} h={16} />
        </div>
        <Skeleton h={12} r={6} className="w-full" />
        <div className="flex gap-[18px]">
          <Skeleton w={160} h={14} />
          <Skeleton w={130} h={14} />
          <Skeleton w={100} h={14} />
        </div>
      </Card>
      <Grid min={200} gap={16}>
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="card tile">
            <Skeleton w={70} h={12} />
            <Skeleton w={90} h={26} />
            <Skeleton w={110} h={12} />
          </div>
        ))}
      </Grid>
      <Card className={cn('flex flex-col gap-3 px-[18px] py-3.5')}>
        <Skeleton w={160} h={16} />
        {[0, 1, 2, 3, 4].map((i) => (
          <Skeleton key={i} h={14} className="w-full" />
        ))}
      </Card>
    </Loading>
  )
}

/** The decision views' badge for a whole campaign paused by a threshold. */
export function HeldBadge({ incident }: { incident: Incident }) {
  const who = scopeName(incident)
  return (
    <Badge tone="bad" pulse>
      {who ? `Auto-paused · ${who}` : 'Auto-paused'}
    </Badge>
  )
}
