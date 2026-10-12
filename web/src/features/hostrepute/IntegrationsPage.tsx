import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { ApiError, errorMessage, http } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { KeyValues } from '@/components/ui/Data'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { ago, clock, count, monthDay, trimPct, usDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ConnectCard } from './ConnectCard'
import { numberWord, roleLabel, ShieldMark, useIntegrations, useRefreshConnection, useStartOAuth, type Connection, type Integrations } from './shared'

/** The answer to "Add to monitoring" when the HostRepute plan is full. */
interface Limit {
  asset: string
  limit: number
  used: number
  team: string
  swap?: string
  swapRole?: string
}

/** Settings › Integrations: HostRepute and the other integrations. */
export function IntegrationsPage() {
  const { slug } = useWorkspace()
  const q = useIntegrations()
  const [limit, setLimit] = useState<Limit | null>(null)
  return (
    <Page title="Settings" documentTitle="Integrations · Settings" tabs={settingsTabs(slug)}>
      {q.isError ? (
        <ErrorPanel title="Couldn't load integrations" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : !q.data ? (
        <section className="card flex flex-col gap-3 px-4.5 py-4" aria-busy="true">
          <span className="vh" role="status">
            Loading integrations
          </span>
          <Skeleton w={140} h={16} />
          <Skeleton w={320} />
          <Skeleton w="100%" h={180} />
        </section>
      ) : (
        <>
          <HostReputeSection conn={q.data.hostRepute} onLimit={setLimit} />
          {limit ? <LimitCard limit={limit} onClose={() => setLimit(null)} /> : null}
          <OtherIntegrations d={q.data} />
        </>
      )}
    </Page>
  )
}

function HostReputeSection({ conn, onLimit }: { conn: Connection; onLimit: (l: Limit | null) => void }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const oauth = useStartOAuth()
  const useKey = () => void navigate({ to: '/w/$workspace/deliverability/connect', params: { workspace: slug }, search: { key: true } })
  if (conn.status === 'not_connected') return <ConnectCard conn={conn} onConnect={oauth.start} onUseKey={useKey} connecting={oauth.pending} />
  if (conn.status === 'revoked' || conn.status === 'expired') return <RevokedCard conn={conn} onReconnect={oauth.start} onUseKey={useKey} reconnecting={oauth.pending} />
  return <HostReputeCard conn={conn} onLimit={onLimit} />
}

/** "Clean on 84 blocklists · Postmaster high reputation" and the other status lines. */
function assetStatus(a: Connection['assets'][number], sources: number) {
  if (!a.monitored) return null
  if (a.listings.length) {
    return (
      <span className="row flex-nowrap gap-1.5">
        <Badge tone="warn">Listed</Badge>
        <span>
          {a.listings.length === 1 ? '1 listing' : `${a.listings.length} listings`} ({a.listings.join(', ')}) · delisting link in Deliverability
        </span>
      </span>
    )
  }
  if (a.status === 'pending') return <span className="k">First check running</span>
  const parts: string[] = []
  if (a.role === 'organization_domain') {
    parts.push('Clean')
    if (a.dmarcPolicy) parts.push(`DMARC p=${a.dmarcPolicy}${a.dmarcAligned != null ? `, ${trimPct(a.dmarcAligned)} aligned` : ''}`)
  } else {
    parts.push(`Clean on ${count(sources)} blocklists`)
    if (a.postmasterReputation) parts.push(`Postmaster ${a.postmasterReputation} reputation`)
  }
  return parts.join(' · ')
}

/** Connected (SettingsIntegrations): account, scopes, monitored assets. */
function HostReputeCard({ conn, onLimit }: { conn: Connection; onLimit: (l: Limit | null) => void }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshConnection()
  const reason = usePermissionReason('integrations:hostrepute')
  const [adding, setAdding] = useState<string>()
  const [syncing, setSyncing] = useState(false)
  const [disconnect, setDisconnect] = useState(false)
  const scopes = conn.scopes.filter((s) => s.granted)
  async function add(asset: string) {
    setAdding(asset)
    onLimit(null)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/assets', { params: { path: { workspace: slug } }, body: { asset } })
      await refresh()
      toast({ title: `${asset} added to monitoring`, detail: 'HostRepute checks it within the hour' })
    } catch (e) {
      if (e instanceof ApiError && e.code === 'monitor_limit_reached') {
        const m = (e.problem.meta ?? {}) as Record<string, unknown>
        onLimit({ asset, limit: Number(m.limit), used: Number(m.used), team: String(m.team ?? conn.teamName), swap: m.swap as string | undefined, swapRole: m.swapRole as string | undefined })
      } else {
        toast({ kind: 'error', title: `Couldn't add ${asset}`, detail: errorMessage(e) })
      }
    } finally {
      setAdding(undefined)
    }
  }
  async function syncNow() {
    setSyncing(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/sync', { params: { path: { workspace: slug } } })
      toast({ kind: 'progress', title: 'Syncing with HostRepute', duration: 4000 })
      window.setTimeout(() => void refresh(), 2500)
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't sync", detail: errorMessage(e) })
    } finally {
      setSyncing(false)
    }
  }
  const tokenWord = conn.syncError ? 'HostRepute unreachable' : 'token valid'
  const meta = [
    conn.method === 'api_key' ? 'API key' : 'OAuth',
    conn.connectedAt ? `connected ${usDate(conn.connectedAt)}${conn.connectedBy ? ` by ${conn.connectedBy}` : ''}` : null,
    tokenWord,
    conn.lastSyncAt ? `last sync ${ago(conn.lastSyncAt)}` : 'first sync running',
  ]
  return (
    <section className="card" aria-labelledby="h-hostrepute">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-hostrepute">
            HostRepute
          </h2>
          <span className="k">Reputation monitoring that feeds Send Readiness and Deliverability</span>
        </div>
      </div>
      <div className="pad col gap-3.5">
        <div className="row justify-between">
          <div className="row gap-3">
            <ShieldMark />
            <div className="col gap-px">
              <span className="font-semibold">Connected to HostRepute team “{conn.teamName}”</span>
              <span className="k">{meta.filter(Boolean).join(' · ')}</span>
            </div>
          </div>
          {conn.syncError ? (
            <Badge tone="warn" dot>
              Unreachable
            </Badge>
          ) : (
            <Badge tone="ok" dot>
              Connected
            </Badge>
          )}
        </div>
        {conn.syncError ? (
          <Note tone="warn">
            {conn.syncError}
            {conn.syncErrorAt ? ` at ${clock(conn.syncErrorAt)}` : ''}. ReputeCast retries every hour and runs its built-in blocklist check meanwhile.
          </Note>
        ) : null}
        <KeyValues
          className="text-[12.5px]"
          items={[
            [
              'Granted scopes',
              <>
                {scopes.map((s, i) => (
                  <span key={s.key}>
                    {i > 0 ? ' · ' : ''}
                    <span className="mono">{s.key}</span> {s.use}
                  </span>
                ))}
              </>,
            ],
            ['Sent to HostRepute', 'Sending domains and dedicated IPs to monitor. Never contacts, never email content.'],
            ['Alerts', conn.alertsToNotifications ? 'Delivered into ReputeCast notifications (in-app and email per member preference)' : 'Not delivered into ReputeCast notifications'],
          ]}
        />
        <Table density="compact" className="[&_td]:!whitespace-normal [&_td_.badge]:!whitespace-nowrap" caption="Assets monitored by HostRepute">
          <thead>
            <tr>
              <Th>Asset</Th>
              <Th>Type</Th>
              <Th>Monitoring</Th>
              <Th>Last sync</Th>
              <Th>Status</Th>
            </tr>
          </thead>
          <tbody>
            {conn.assets.map((a) => (
              <tr key={a.asset} className={a.monitored ? undefined : 'bg-[#FFFBEB]'}>
                <Td>
                  <span className="mono">{a.asset}</span>
                </Td>
                <Td>{roleLabel(a)}</Td>
                <Td>{a.monitored ? <Badge tone="ok">Monitored</Badge> : <Badge tone="warn">Not monitored</Badge>}</Td>
                <Td>{a.monitored && a.lastSyncAt ? ago(a.lastSyncAt) : <span className="k">–</span>}</Td>
                <Td>
                  {a.monitored ? (
                    assetStatus(a, conn.sources)
                  ) : (
                    <Button size="sm" variant="primary" onClick={() => add(a.asset)} loading={adding === a.asset} disabledReason={reason}>
                      Add to monitoring
                    </Button>
                  )}
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
        <div className="row gap-2">
          <Button size="sm" onClick={syncNow} loading={syncing} disabledReason={reason}>
            Sync now
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/deliverability" params={{ workspace: slug }}>
            Open Deliverability
          </ButtonLink>
          <Button size="sm" variant="danger" onClick={() => setDisconnect(true)} disabledReason={reason}>
            Disconnect
          </Button>
        </div>
      </div>
      <DisconnectDialog open={disconnect} onOpenChange={setDisconnect} team={conn.teamName ?? ''} />
    </section>
  )
}

/** Token revoked or expired → Reconnect (HostReputeStates). */
function RevokedCard({ conn, onReconnect, onUseKey, reconnecting }: { conn: Connection; onReconnect: () => void; onUseKey: () => void; reconnecting: boolean }) {
  const reason = usePermissionReason('integrations:hostrepute')
  const at = conn.tokenRejectedAt
  return (
    <section className="card" aria-labelledby="h-hr-rev">
      <div className="sec">
        <div className="col gap-0.5">
          <h2 className="h2" id="h-hr-rev">
            Token revoked or expired
          </h2>
          <span className="k">
            HostRepute team “{conn.teamName}” · {conn.method === 'api_key' ? 'API key' : 'OAuth'}
          </span>
        </div>
        <Badge tone="warn">Stale since {at ? monthDay(at) : 'the last sync'}</Badge>
      </div>
      <div className="pad col gap-3">
        <Note tone="warn" label="Reconnect" role="alert">
          HostRepute rejected the token{at ? ` on ${monthDay(at)}, ${clock(at)}` : ''} (revoked from the HostRepute side or expired). Blocklist, Postmaster, SNDS and DMARC data is
          stale since then; Send Readiness marks the reputation check as “unknown”, not as a Block.
        </Note>
        <KeyValues
          className="text-[12.5px]"
          items={[
            ['Still working', 'Everything in ReputeCast · provider-side health, bounce and complaint rates, authentication checks'],
            ['Paused', 'Blocklist checks, Postmaster and SNDS panels, DMARC summary, HostRepute alerts'],
          ]}
        />
        <div className="row gap-2">
          <Button size="sm" variant="primary" onClick={onReconnect} loading={reconnecting} disabledReason={reason}>
            <ShieldMark />
            Reconnect with HostRepute
          </Button>
          <Button size="sm" onClick={onUseKey} disabledReason={reason}>
            Use an API key instead
          </Button>
        </div>
      </div>
    </section>
  )
}

/** HostRepute's own monitoring limit, shown honestly (HostReputeStates). */
function LimitCard({ limit, onClose }: { limit: Limit; onClose: () => void }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshConnection()
  const [busy, setBusy] = useState(false)
  async function swap() {
    if (!limit.swap) return
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/assets/swap', { params: { path: { workspace: slug } }, body: { remove: limit.swap, add: limit.asset } })
      await refresh()
      onClose()
      toast({ title: `${limit.asset} monitored instead of ${limit.swap}`, detail: 'HostRepute checks it within the hour' })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't swap the assets", detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const role = limit.swapRole === 'organization_domain' ? 'organization domain' : limit.swapRole?.replaceAll('_', ' ')
  return (
    <section className="card" aria-labelledby="h-hr-limit">
      <div className="sec">
        <div className="col gap-0.5">
          <h2 className="h2" id="h-hr-limit">
            HostRepute monitoring limit reached
          </h2>
          <span className="k">Adding {limit.asset} to monitoring</span>
        </div>
        <Badge tone="warn">
          {limit.used} of {limit.limit} used
        </Badge>
      </div>
      <div className="pad col gap-3">
        <Note tone="neutral" role="status">
          Your HostRepute team “{limit.team}” monitors {limit.used} of {limit.limit} assets allowed by its plan, so {limit.asset} can't be added right now. This is a HostRepute
          limit, not a ReputeCast one: ReputeCast stays fully usable and the {numberWord(limit.used)} monitored assets keep syncing.
        </Note>
        <KeyValues
          className="text-[12.5px]"
          items={[
            [
              'Options',
              limit.swap
                ? `Raise the limit in HostRepute, or swap an asset: ${limit.swap} (${role}) could make room for ${limit.asset}, which actually sends mail.`
                : 'Raise the limit in HostRepute, or stop monitoring an asset there to make room.',
            ],
          ]}
        />
        <div className="row gap-2">
          <ButtonAnchor size="sm" href="https://app.hostrepute.com">
            Manage the plan in HostRepute
          </ButtonAnchor>
          {limit.swap ? (
            <Button size="sm" onClick={swap} loading={busy}>
              Swap {limit.swap} for {limit.asset}
            </Button>
          ) : null}
          <Button size="sm" onClick={onClose}>
            Not now
          </Button>
        </div>
      </div>
    </section>
  )
}

/** Disconnect, listing what stops working first (HostReputeStates). */
function DisconnectDialog({ open, onOpenChange, team }: { open: boolean; onOpenChange: (o: boolean) => void; team: string }) {
  const { slug } = useWorkspace()
  const refresh = useRefreshConnection()
  const [busy, setBusy] = useState(false)
  async function disconnect() {
    setBusy(true)
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/integrations/hostrepute', { params: { path: { workspace: slug } } })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'deleted', title: 'HostRepute disconnected', detail: `team “${team}” keeps monitoring on hostrepute.com` })
    } catch (e) {
      toast({ kind: 'error', title: "Couldn't disconnect HostRepute", detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Disconnect HostRepute?"
      description={<span className="text-[13.5px]">ReputeCast forgets the token. Your HostRepute team and its monitoring keep running on hostrepute.com; nothing is deleted there.</span>}
      dismissible={!busy}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button variant="danger-primary" onClick={disconnect} loading={busy}>
            Disconnect
          </Button>
        </>
      }
    >
      <span className="text-[12px] font-semibold text-ink-2">Stops working in ReputeCast</span>
      <ul className="m-0 flex flex-col gap-[3px] pl-[18px] text-[13.5px] text-ink-nav">
        <li>Blocklist check in Send Readiness (shown as “not checked”)</li>
        <li>Postmaster, SNDS and DMARC panels in Deliverability</li>
        <li>HostRepute alerts in notifications</li>
        <li>“Add to monitoring” for new domains and IPs</li>
      </ul>
      <span className="text-[12px] font-semibold text-ink-2">Keeps working</span>
      <ul className="m-0 pl-[18px] text-[13.5px] text-ink-nav">
        <li>Every other feature, including provider health, routing, guarded sending and reports</li>
      </ul>
    </Dialog>
  )
}

/** Other integrations: webhooks, MCP, Postmaster and SNDS. */
function OtherIntegrations({ d }: { d: Integrations }) {
  const { slug } = useWorkspace()
  const params = { workspace: slug }
  const via = (v: string) =>
    v === 'hostrepute' ? <Badge tone="ok">Via HostRepute</Badge> : v === 'direct' ? <Badge tone="ok">Connected directly</Badge> : <Badge tone="neutral">Not connected</Badge>
  return (
    <section className="card" aria-labelledby="h-other">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-other">
            Other integrations
          </h2>
          <span className="k">Everything here is part of the open-source edition</span>
        </div>
      </div>
      <Table caption="Other integrations">
        <thead>
          <tr>
            <Th>Integration</Th>
            <Th>Status</Th>
            <Th>Used for</Th>
            <Th>
              <span className="vh">Actions</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <Td>
              <span className="font-semibold">Outgoing webhooks</span>
            </Td>
            <Td>
              {d.webhooks.endpoints ? (
                <Badge tone="ok">
                  {count(d.webhooks.endpoints)} {d.webhooks.endpoints === 1 ? 'endpoint' : 'endpoints'}
                  {d.webhooks.failing ? ` · ${count(d.webhooks.failing)} failing` : ''}
                </Badge>
              ) : (
                <Badge tone="neutral">No endpoints</Badge>
              )}
            </Td>
            <Td>Normalized events to your systems and Slack</Td>
            <Td>
              <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={params}>
                Manage
              </ButtonLink>
            </Td>
          </tr>
          <tr>
            <Td>
              <span className="font-semibold">MCP server</span>
            </Td>
            <Td>
              {d.mcp.agents ? (
                <Badge tone="ok">
                  On · {count(d.mcp.agents)} {d.mcp.agents === 1 ? 'agent' : 'agents'}
                </Badge>
              ) : (
                <Badge tone="neutral">No agents</Badge>
              )}
            </Td>
            <Td>AI agents with scoped permissions</Td>
            <Td>
              <ButtonLink size="sm" to="/w/$workspace/settings/keys" params={params}>
                Manage
              </ButtonLink>
            </Td>
          </tr>
          <tr>
            <Td>
              <span className="font-semibold">Google Postmaster Tools</span>
            </Td>
            <Td>{via(d.postmaster.via)}</Td>
            <Td>Domain reputation, spam rate, feedback loop</Td>
            <Td>
              <ButtonLink size="sm" to="/w/$workspace/deliverability" params={params}>
                View
              </ButtonLink>
            </Td>
          </tr>
          <tr>
            <Td>
              <span className="font-semibold">Microsoft SNDS</span>
            </Td>
            <Td>{via(d.snds.via)}</Td>
            <Td>IP reputation for Outlook.com</Td>
            <Td>
              <ButtonLink size="sm" to="/w/$workspace/deliverability" params={params}>
                View
              </ButtonLink>
            </Td>
          </tr>
        </tbody>
      </Table>
    </section>
  )
}
