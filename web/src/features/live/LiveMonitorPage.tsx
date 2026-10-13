import { useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { ApiError } from '@/api/client'
import { Page, type Crumb } from '@/components/shell/Page'
import { ForbiddenState, NotFoundState, problemReference, ServerErrorState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { clockSeconds } from '@/lib/format'
import { useMediaQuery, useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { CancelSendDialog, useLiveActions, useLivePermissions, type LiveActions, type LivePermissions } from './actions'
import { DecisionView } from './Decision'
import { LiveBody } from './LiveParts'
import { MobileLive } from './Mobile'
import { eventsCaption, headerBadges, isStarting, lostText, openIncident, startedLine, type CaptionKind, type Incident, type Snapshot } from './model'
import {
  CancelledView,
  ConnectingCard,
  FailedView,
  FinishedView,
  IdleView,
  IncidentBanner,
  LiveSkeleton,
  PausedBanner,
  ScheduledView,
  StreamLostBanner,
} from './States'
import { MAX_ATTEMPTS, useLiveSend, type StreamState } from './useLiveSend'
import './live.css'

export interface LiveSearch {
  /** "live" keeps the live view while an incident waits for a decision. */
  view?: 'live'
}

/**
 * 7.6 Live sending monitor (boards LiveMonitor, LiveMonitorPaused,
 * LiveMonitorOutage, LiveMonitorStates, LiveMonitorMobile): the snapshot
 * from the API, kept live by the event stream.
 */
export function LiveMonitorPage({ campaignId, search }: { campaignId: string; search: LiveSearch }) {
  const { slug } = useWorkspace()
  const live = useLiveSend(slug, campaignId)
  if (!live.snapshot) {
    if (live.query.isError) return <LiveError error={live.query.error} retry={() => void live.query.refetch()} campaignId={campaignId} />
    return (
      <Page title="Live sending monitor" breadcrumb={crumbs(slug, 'Loading…')}>
        <LiveSkeleton />
      </Page>
    )
  }
  return <LiveScreen s={live.snapshot} campaignId={campaignId} search={search} helpers={live.helpers} stream={live.stream} apply={live.apply} />
}

function crumbs(slug: string, name: string): Crumb[] {
  return [{ label: 'Campaigns', to: '/w/$workspace/campaigns', params: { workspace: slug } }, { label: name }]
}

/** Which body the screen shows. */
type Mode = 'decision' | 'connecting' | 'live' | 'finished' | 'cancelled' | 'failed' | 'scheduled' | 'idle'

function modeOf(s: Snapshot, incident: Incident | undefined, view: LiveSearch['view'], now: Date): Mode {
  switch (s.state) {
    case 'finished':
      return 'finished'
    case 'cancelled':
      return 'cancelled'
    case 'failed':
      return 'failed'
    case 'scheduled':
      return 'scheduled'
    case 'idle':
      return 'idle'
  }
  if (incident && view !== 'live') return 'decision'
  if (isStarting(s, now)) return 'connecting'
  return 'live'
}

function LiveScreen({
  s,
  campaignId,
  search,
  helpers,
  stream,
  apply,
}: {
  s: Snapshot
  campaignId: string
  search: LiveSearch
  helpers: Snapshot['helpers']
  stream: StreamState & { reconnect: () => void }
  apply: (s: Snapshot) => void
}) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const now = useNow(1000)
  const phone = useMediaQuery('(max-width: 767px)')
  const actions = useLiveActions(slug, campaignId, apply)
  const can = useLivePermissions(helpers)
  const [cancelling, setCancelling] = useState(false)
  const incident = openIncident(s)
  const mode = modeOf(s, incident, search.view, now)
  const active = ['preparing', 'sending', 'partial', 'paused'].includes(s.state)
  const lost = active && (stream.status === 'retrying' || stream.status === 'stopped') && stream.lostAt !== undefined
  const asOf = clockSeconds(s.at)
  const setView = (view: LiveSearch['view']) =>
    void navigate({ to: '/w/$workspace/campaigns/$campaignId/live', params: { workspace: slug, campaignId }, search: view ? { view } : {}, replace: true })
  const pause = () => void actions.pause().catch(() => undefined)
  const resume = () => void actions.resume().catch(() => undefined)
  const cancelOpen = () => setCancelling(true)

  const lostBanner =
    lost && stream.lostAt ? (
      <StreamLostBanner
        {...lostText(stream.lostAt, s.at, stream.attempt, MAX_ATTEMPTS, stream.status === 'stopped', now)}
        onReconnect={stream.reconnect}
      />
    ) : null
  const pausedBanner =
    s.state === 'paused' && !incident ? <PausedBanner s={s} can={can} onResume={resume} onCancel={cancelOpen} busy={actions.pending} /> : null
  const incidentBanner = incident && mode !== 'decision' ? <IncidentBanner s={s} incident={incident} onDecide={() => setView(undefined)} /> : null

  const dialog = (
    <CancelSendDialog open={cancelling} onOpenChange={setCancelling} s={s} incident={incident} actions={actions} canPause={!can.pause} />
  )

  if (phone && (mode === 'live' || mode === 'connecting')) {
    return (
      <>
        <MobileLive
          s={s}
          now={now}
          streaming={!lost}
          incident={incident}
          connecting={mode === 'connecting'}
          banner={
            lostBanner || pausedBanner || incidentBanner ? (
              <>
                {lostBanner}
                {pausedBanner}
                {incidentBanner}
              </>
            ) : null
          }
          can={can}
          onPause={pause}
          onResume={resume}
          onCancel={cancelOpen}
          busy={actions.pending}
        />
        {dialog}
      </>
    )
  }

  let body: ReactNode
  switch (mode) {
    case 'decision':
      body = <DecisionView s={s} incident={incident!} actions={actions} can={can} onCancelAll={cancelOpen} />
      break
    case 'connecting':
      body = (
        <>
          {lostBanner}
          <ConnectingCard s={s} />
        </>
      )
      break
    case 'finished':
      body = <FinishedView s={s} campaignId={campaignId} />
      break
    case 'cancelled':
      body = <CancelledView s={s} campaignId={campaignId} />
      break
    case 'failed':
      body = <FailedView s={s} campaignId={campaignId} />
      break
    case 'scheduled':
      body = <ScheduledView s={s} campaignId={campaignId} />
      break
    case 'idle':
      body = <IdleView s={s} campaignId={campaignId} />
      break
    default:
      body = (
        <>
          {lostBanner}
          {pausedBanner}
          {incidentBanner}
          <LiveBody s={s} campaignId={campaignId} streaming={!lost} stale={lost} asOf={asOf} incident={incident} />
        </>
      )
  }

  return (
    <Page
      title="Live sending monitor"
      documentTitle={`${s.name} · Live`}
      breadcrumb={crumbs(slug, s.name)}
      subtitle={startedLine(s) || undefined}
      badges={<HeaderBadges s={s} incident={incident} mode={mode} />}
      actions={<HeaderActions s={s} mode={mode} incident={incident} now={now} lost={lost} can={can} actions={actions} onPause={pause} onCancel={cancelOpen} onBack={() => setView('live')} />}
    >
      {body}
      {dialog}
    </Page>
  )
}

/** The state words after the breadcrumb (none once a send has ended, as on LiveMonitorStates). */
function HeaderBadges({ s, incident, mode }: { s: Snapshot; incident?: Incident; mode: Mode }) {
  if (mode === 'finished' || mode === 'cancelled' || mode === 'failed' || mode === 'connecting' || mode === 'idle') return null
  const badges = headerBadges(s, incident)
  return (
    <>
      {badges.map((b) =>
        b.live ? (
          <span key={b.text} className="badge ok">
            <span className="dot pulse-live" aria-hidden="true" />
            {b.text}
          </span>
        ) : (
          <Badge key={b.text} tone={b.tone} dot={!b.pulse} pulse={b.pulse}>
            {b.text}
          </Badge>
        ),
      )}
    </>
  )
}

/** The top bar's events line and actions for each view. */
function HeaderActions({
  s,
  mode,
  incident,
  now,
  lost,
  can,
  actions,
  onPause,
  onCancel,
  onBack,
}: {
  s: Snapshot
  mode: Mode
  incident?: Incident
  now: Date
  lost: boolean
  can: LivePermissions
  actions: LiveActions
  onPause: () => void
  onCancel: () => void
  onBack: () => void
}) {
  if (mode === 'decision') {
    const kind: CaptionKind = incident?.kind === 'provider_outage' ? 'outage' : 'threshold'
    return (
      <>
        <span className="k">{eventsCaption(s, now, kind, !lost)}</span>
        <Button onClick={onBack}>Back to live view</Button>
      </>
    )
  }
  if (mode !== 'live' && mode !== 'connecting') return null
  const paused = s.state === 'paused' && !incident
  return (
    <>
      <span className="k">{eventsCaption(s, now, 'live', !lost)}</span>
      {s.options.canPause ? (
        <Button onClick={onPause} disabledReason={can.pause} loading={actions.pending} title="Pausing needs campaigns:pause">
          Pause
        </Button>
      ) : null}
      {s.options.canCancel && !paused ? (
        <Button variant="danger" onClick={onCancel} disabledReason={can.cancel}>
          Cancel send
        </Button>
      ) : null}
    </>
  )
}

/** 403, 404 or 500 while loading the monitor. */
function LiveError({ error, retry, campaignId }: { error: unknown; retry: () => void; campaignId: string }) {
  const { slug } = useWorkspace()
  const status = error instanceof ApiError ? error.status : 0
  const back = (
    <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
      Back to campaigns
    </ButtonLink>
  )
  let body: ReactNode
  if (error instanceof ApiError && status === 403 && error.problem.permission) {
    body = <ForbiddenState permission={error.problem.permission} title="Your role can't watch this send" readOnly={back} />
  } else if (status === 404) {
    body = (
      <NotFoundState path={`/campaigns/${campaignId}/live`} reference={problemReference(error)} primary={back}>
        The campaign may have been deleted, or the link is from another workspace.
      </NotFoundState>
    )
  } else {
    body = <ServerErrorState reference={problemReference(error)} savedNote="Sending continues on the server." onRetry={retry} back={back} />
  }
  return (
    <Page title={status === 404 ? 'Not found' : status === 403 ? 'No access' : 'Live sending monitor'} breadcrumb={crumbs(slug, 'Live sending monitor')}>
      {body}
    </Page>
  )
}
