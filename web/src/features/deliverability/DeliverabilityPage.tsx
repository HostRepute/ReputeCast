import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { andList, clock, count, monthDay } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { AssetGrid, SkeletonGrid, StaleGrid } from './AssetCards'
import { BlocklistCard } from './BlocklistCard'
import { DmarcCard, DmarcEmptyCard } from './DmarcCard'
import { IncidentTimeline } from './IncidentTimeline'
import { ProviderHealthCard } from './ProviderHealth'
import { PostmasterCard, SndsCard } from './SignalCards'
import { eta, firstRun, type Overview } from './words'

const overviewPath = '/api/v1/workspaces/{workspace}/deliverability' as const

/** 7.8 Deliverability: HostRepute connected or not, with loading and error states. */
export function DeliverabilityPage() {
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    overviewPath,
    { params: { path: { workspace: slug } } },
    { refetchInterval: (query) => (query.state.data?.checks.running ? 1000 : false) },
  )
  const o = q.data
  useFinishedToast(o)
  return (
    <Page title="Deliverability" badges={o ? <HeaderStatus o={o} /> : null} actions={<Actions o={o} />}>
      {q.isError ? (
        <ErrorPanel title="Couldn't load Deliverability" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed; the checks keep running in the background.
        </ErrorPanel>
      ) : !o ? (
        <section className="flex flex-col gap-3.5" aria-label="Loading">
          <span className="vh" role="status">
            Loading deliverability
          </span>
          <div className="flex items-center gap-3 rounded-[10px] border border-line bg-white px-4 py-3">
            <Skeleton w={16} h={16} r={999} />
            <Skeleton w={260} h={14} />
            <Skeleton w={200} h={12} />
          </div>
          <SkeletonGrid />
        </section>
      ) : o.assets.length === 0 && o.providers.length === 0 ? (
        <NoSetup />
      ) : (
        <Body o={o} />
      )}
    </Page>
  )
}

/** Toast when a check run that was running finishes. */
function useFinishedToast(o?: Overview): void {
  const last = useRef<string | null>(null)
  useEffect(() => {
    const running = o?.checks.running?.id ?? null
    if (last.current && !running && o) {
      const listed = o.blocklists.listed
      toast({
        kind: 'success',
        title: 'Checks finished',
        detail: listed ? `${count(listed)} ${listed === 1 ? 'listing' : 'listings'} on ${count(o.blocklists.sources)} sources` : `clean on ${count(o.blocklists.sources)} sources`,
      })
    }
    last.current = running
  }, [o])
}

function HeaderStatus({ o }: { o: Overview }) {
  const hr = o.hostRepute
  const c = o.checks
  let badge
  if (hr.status === 'connected' && hr.unreachable) {
    badge = (
      <Badge tone="bad" dot>
        HostRepute unreachable · cached results
      </Badge>
    )
  } else if (hr.status === 'connected') {
    badge = (
      <Badge tone="ok" dot>
        HostRepute connected · {c.interval} checks
      </Badge>
    )
  } else if (hr.status === 'not_connected') {
    badge = (
      <Badge tone="neutral" dot>
        HostRepute not connected · provider data and built-in checks
      </Badge>
    )
  } else {
    badge = (
      <Badge tone="warn" dot>
        HostRepute needs reconnecting
      </Badge>
    )
  }
  const when =
    c.via === 'hostrepute'
      ? c.lastAt
        ? `Last full check ${clock(c.lastAt)}${c.nextAt ? ` · next ${clock(c.nextAt)}` : ''}`
        : 'First check running'
      : c.lastAt
        ? `Last check ${clock(c.lastAt)} · daily`
        : 'Daily checks'
  return (
    <>
      {badge}
      <span className="k">{when}</span>
    </>
  )
}

function Actions({ o }: { o?: Overview }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/deliverability/checks', {
    onSuccess: () => qc.invalidateQueries({ queryKey: ['get', overviewPath] }),
    onError: (e) => toast({ kind: 'error', title: "Couldn't start the checks", detail: errorMessage(e) }),
  })
  const [exporting, setExporting] = useState(false)
  async function exportReport() {
    setExporting(true)
    try {
      const { data, response } = await http.GET('/api/v1/workspaces/{workspace}/deliverability/export', { params: { path: { workspace: slug } }, parseAs: 'blob' })
      const name = /filename="([^"]+)"/.exec(response.headers.get('Content-Disposition') ?? '')?.[1] ?? 'deliverability.csv'
      const url = URL.createObjectURL(data as Blob)
      const a = document.createElement('a')
      a.href = url
      a.download = name
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't export the report", detail: errorMessage(e) })
    } finally {
      setExporting(false)
    }
  }
  const running = Boolean(o?.checks.running)
  const label = o && o.hostRepute.status === 'connected' ? 'Run all checks now' : 'Run checks now'
  return (
    <>
      <Button onClick={exportReport} loading={exporting} disabled={!o}>
        Export report
      </Button>
      <Button variant="primary" onClick={() => run.mutate({ params: { path: { workspace: slug } } })} loading={run.isPending || running} disabled={!o}>
        {label}
      </Button>
    </>
  )
}

/** First run of a workspace without a sending domain or provider. */
function NoSetup() {
  const { slug } = useWorkspace()
  return (
    <EmptyState
      title="Deliverability appears once mail can be sent"
      action={
        <ButtonLink variant="primary" to="/w/$workspace/providers/connect" params={{ workspace: slug }}>
          Connect a sending provider
        </ButtonLink>
      }
    >
      Domain and IP health, blocklists, Postmaster, SNDS and DMARC show up here for your sending domains and dedicated IPs, with each provider's account health.
    </EmptyState>
  )
}

/** "Checking news.acme.io and 203.0.113.25 · 61 of 84 sources answered · about 6 s left". */
function Progress({ o }: { o: Overview }) {
  const now = useNow(1000)
  const r = o.checks.running!
  const names = r.assets.length ? r.assets : o.assets.map((a) => a.asset)
  return (
    <div className="flex items-center gap-3 rounded-[10px] border border-line bg-white px-4 py-3 max-md:flex-wrap" role="status" aria-live="polite">
      <span className="spin" aria-hidden="true" />
      <span className="font-semibold">Checking {andList(names)}</span>
      <span className="k">
        {count(r.answered)} of {count(r.total)} sources answered · {eta(r, now.getTime())}
      </span>
    </div>
  )
}

function Body({ o }: { o: Overview }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const hr = o.hostRepute
  const connected = hr.status === 'connected'
  const stale = (connected && Boolean(hr.unreachable)) || hr.status === 'revoked' || hr.status === 'expired'
  const canConnect = useCan('integrations:hostrepute')
  const delistReason = usePermissionReason('integrations:hostrepute')
  const [delisting, setDelisting] = useState<string>()
  async function delist(asset: string) {
    setDelisting(asset)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/deliverability/delistings', { params: { path: { workspace: slug } }, body: { asset } })
      await qc.invalidateQueries({ queryKey: ['get', overviewPath] })
      const r = data?.requests[0]
      if (r?.already) {
        toast({ kind: 'info', title: `Delisting already requested on ${monthDay(r.requestedAt)}`, detail: `${r.source} hasn't answered yet; HostRepute follows up.` })
      } else if (r) {
        toast({ kind: 'success', title: `Delisting requested for ${asset}`, detail: `${andList(data!.requests.map((x) => x.source))} · HostRepute follows up` })
      }
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't request delisting", detail: e instanceof ApiError ? e.message : errorMessage(e) })
    } finally {
      setDelisting(undefined)
    }
  }

  if (firstRun(o)) {
    return (
      <section className="flex flex-col gap-3.5" aria-label="Loading">
        <Progress o={o} />
        <SkeletonGrid />
      </section>
    )
  }
  return (
    <>
      {o.checks.running ? <Progress o={o} /> : null}
      {hr.status === 'revoked' || hr.status === 'expired' ? <ReconnectBanner rejectedAt={hr.tokenRejectedAt} /> : null}
      {connected && hr.unreachable ? <UnreachableAlert o={o} /> : null}
      {hr.status === 'not_connected' ? <ConnectInvite canConnect={canConnect} /> : null}
      {stale ? (
        <section className="flex flex-col gap-3.5" aria-label="Cached results">
          <StaleGrid o={o} />
          <span className="k">Send Readiness stays usable: blocklist checks are marked Unknown instead of Pass until HostRepute answers.</span>
        </section>
      ) : (
        <AssetGrid o={o} onDelist={delist} delistReason={delistReason} delisting={delisting} />
      )}
      <ProviderHealthCard providers={o.providers} connected={connected} />
      <div className="flex flex-wrap items-start gap-5 max-md:gap-3">
        <BlocklistCard b={o.blocklists} connected={connected} canConnect={canConnect} />
        <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-5 max-md:gap-3">
          {o.postmaster ? <PostmasterCard p={o.postmaster} /> : <PanelPlaceholder title="Google Postmaster Tools" what="Domain and IP reputation, spam rate and authentication at Gmail" />}
          {o.snds ? <SndsCard s={o.snds} /> : o.assets.some((a) => a.type === 'ip') ? <PanelPlaceholder title="Microsoft SNDS" what="Filter results, complaints and trap hits for your dedicated IPs at Outlook.com" /> : null}
          {connected ? null : o.dmarc.sources.length ? <DmarcCard d={o.dmarc} className="card min-w-0" /> : <DmarcEmptyCard connected={false} />}
        </div>
      </div>
      {connected ? (
        <div className="flex flex-wrap items-start gap-5 max-md:gap-3">
          {o.dmarc.sources.length ? <DmarcCard d={o.dmarc} /> : <div className="min-w-0 flex-[1_1_440px]"><DmarcEmptyCard connected rua={hr.dmarcRua} /></div>}
          <IncidentTimeline t={o.incidents} connected className="flex-[1_1_440px]" />
        </div>
      ) : (
        <IncidentTimeline t={o.incidents} connected={false} />
      )}
    </>
  )
}

/** A Postmaster or SNDS panel without data. */
function PanelPlaceholder({ title, what }: { title: string; what: string }) {
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col gap-2.5 border-dashed px-4.5 py-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2">{title}</h2>
        <Badge tone="neutral">No data yet</Badge>
      </div>
      <span className="k">{what}. Connect HostRepute to bring it in; it appears here after the first sync.</span>
      <div className="flex">
        <ButtonLink size="sm" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
          Connect HostRepute
        </ButtonLink>
      </div>
    </section>
  )
}

/** Not connected: everything works, HostRepute adds more (DeliverabilityNoHostRepute). */
function ConnectInvite({ canConnect }: { canConnect: boolean }) {
  const { slug } = useWorkspace()
  const reason = usePermissionReason('integrations:hostrepute')
  return (
    <section className="card flex flex-wrap items-center gap-5 border-accent-line bg-accent-tint px-5.5 py-5" aria-labelledby="h-invite">
      <svg width="44" height="44" viewBox="0 0 44 44" aria-hidden="true">
        <rect width="44" height="44" rx="11" fill="#0F766E" />
        <path d="M22 9 33 13.5v8c0 6-4.6 10.6-11 13-6.4-2.4-11-7-11-13v-8z" fill="none" stroke="#fff" strokeWidth="2" strokeLinejoin="round" />
        <path d="m16.5 22 3.6 3.6 7.4-7.6" fill="none" stroke="#fff" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
      <div className="flex flex-[1_1_360px] flex-col gap-1">
        <h2 className="m-0 text-[16px] font-bold" id="h-invite">
          Everything on this page works without HostRepute. Connecting it adds more.
        </h2>
        <span className="text-[13px] text-ink-nav">
          HostRepute is our sibling email-reputation service. Connected, it checks 84 blocklists every hour instead of 8 once a day, adds delisting workflows, collects your DMARC
          reports, and feeds all of it into Send Readiness. No paywall in ReputeCast: every feature here stays on either way.
        </span>
      </div>
      <div className="flex flex-wrap gap-2.5">
        <ButtonAnchor href="https://hostrepute.com">What HostRepute checks</ButtonAnchor>
        {canConnect ? (
          <ButtonLink variant="primary" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
            Connect HostRepute
          </ButtonLink>
        ) : (
          <Button variant="primary" disabledReason={reason}>
            Connect HostRepute
          </Button>
        )}
      </div>
    </section>
  )
}

/** Token revoked or expired: the Deliverability banner (HostReputeStates). */
function ReconnectBanner({ rejectedAt }: { rejectedAt?: string }) {
  const { slug } = useWorkspace()
  const reason = usePermissionReason('integrations:hostrepute')
  const when = rejectedAt ? ` on ${monthDay(rejectedAt)}, ${clock(rejectedAt)}` : ''
  return (
    <section className="note w items-center" role="status" aria-label="HostRepute needs reconnecting">
      <Badge tone="warn">Reconnect</Badge>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <span className="min-w-0 flex-[1_1_420px]">
          HostRepute rejected the token{when} (revoked from the HostRepute side or expired). Blocklist, Postmaster, SNDS and DMARC data is stale since then; Send Readiness marks the
          reputation check as “unknown”, not as a Block.
        </span>
        {reason ? null : (
          <ButtonLink size="sm" variant="primary" to="/w/$workspace/deliverability/connect" params={{ workspace: slug }}>
            Reconnect with HostRepute
          </ButtonLink>
        )}
      </div>
    </section>
  )
}

/** HostRepute unreachable: cached results are shown (DeliverabilityStates › error). */
function UnreachableAlert({ o }: { o: Overview }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const run = $api.useMutation('post', '/api/v1/workspaces/{workspace}/deliverability/checks', {
    onSuccess: () => qc.invalidateQueries({ queryKey: ['get', overviewPath] }),
    onError: (e) => toast({ kind: 'error', title: "Couldn't start the checks", detail: errorMessage(e) }),
  })
  const hr = o.hostRepute
  const last = o.checks.lastAt ?? hr.lastSyncAt
  const stale = ['Blocklist', 'DMARC-report']
  const current = ['Provider account health']
  if (o.postmaster?.via === 'direct') current.push('Google Postmaster')
  else if (o.postmaster) stale.push('Google Postmaster')
  if (o.snds?.via === 'direct') current.push('Microsoft SNDS')
  else if (o.snds) stale.push('Microsoft SNDS')
  stale.push('incident')
  const fallback = o.assets.find((a) => a.fallback)?.fallback
  const fallbackListed = o.assets.reduce((n, a) => n + (a.fallback?.listed ?? 0), 0)
  return (
    <div role="alert" className="flex items-start gap-3.5 rounded-[10px] border border-[#FECACA] bg-[#FEF2F2] px-4 py-3.5">
      <svg width="20" height="20" viewBox="0 0 20 20" fill="none" stroke="#B91C1C" strokeWidth="1.8" aria-hidden="true">
        <circle cx="10" cy="10" r="8" />
        <path d="M10 6v5M10 13.5v.5" strokeLinecap="round" />
      </svg>
      <div className="flex flex-[1_1_300px] flex-col gap-1">
        <span className="font-semibold text-bad-fg">{hr.unreachable?.message}</span>
        <span className="text-[13px] text-[#7F1D1D]">
          {andList(stale)} data below {stale.length > 1 ? 'are' : 'is'} from the last successful check{last ? ` at ${clock(last)}` : ''}. {andList(current)}{' '}
          {current.length > 1 ? 'are' : 'is'} fetched directly and {current.length > 1 ? 'are' : 'is'} current.{' '}
          {fallback?.checkedAt
            ? `The built-in ${fallback.sources}-source blocklist check ran as a fallback and ${fallbackListed ? `found ${fallbackListed} ${fallbackListed === 1 ? 'listing' : 'listings'}` : 'found nothing new'}.`
            : 'The built-in blocklist check runs as a fallback.'}
        </span>
        <div className="mt-1.5 flex flex-wrap gap-2.5">
          <Button size="sm" variant="primary" onClick={() => run.mutate({ params: { path: { workspace: slug } } })} loading={run.isPending}>
            Retry
          </Button>
          <ButtonAnchor size="sm" href="https://status.hostrepute.com">
            HostRepute status
          </ButtonAnchor>
          <Button size="sm" onClick={() => run.mutate({ params: { path: { workspace: slug }, query: { builtin: true } } })}>
            Use built-in checks until it's back
          </Button>
        </div>
      </div>
    </div>
  )
}
