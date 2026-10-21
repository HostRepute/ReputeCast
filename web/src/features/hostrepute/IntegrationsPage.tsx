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
import { m as md } from '@/features/deliverability/messages'
import { m as hub } from '@/features/integrations/messages'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, clock, count, monthDay, trimPct, usDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { ConnectCard } from './ConnectCard'
import { m } from './messages'
import { plainNote, roleLabel, ShieldMark, useIntegrations, useRefreshConnection, useStartOAuth, type Connection } from './shared'

/** The answer to "Add to monitoring" when the HostRepute plan is full. */
interface Limit {
  asset: string
  limit: number
  used: number
  team: string
  swap?: string
  swapRole?: string
}

/**
 * Settings › Integrations › HostRepute (SettingsIntegrations): today's
 * connected view, now the HostRepute card's page in the integrations hub.
 * The other integrations it used to list are cards of the catalog.
 */
export function IntegrationsPage() {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useIntegrations()
  const [limit, setLimit] = useState<Limit | null>(null)
  const name = t(hub.hostrepute)
  return (
    <Page
      title={name}
      documentTitle={t(m.documentTitle)}
      breadcrumb={[
        { label: t(hub.settings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(hub.title), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: name },
      ]}
    >
      {q.isError ? (
        <ErrorPanel title={t(m.integrationsLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.nothingChanged)}
        </ErrorPanel>
      ) : !q.data ? (
        <section className="card flex flex-col gap-3 px-4.5 py-4" aria-busy="true">
          <span className="vh" role="status">
            {t(m.integrationsLoading)}
          </span>
          <Skeleton w={140} h={16} />
          <Skeleton w={320} />
          <Skeleton w="100%" h={180} />
        </section>
      ) : (
        <>
          <HostReputeSection conn={q.data.hostRepute} onLimit={setLimit} />
          {limit ? <LimitCard limit={limit} onClose={() => setLimit(null)} /> : null}
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
  if (conn.status === 'not_connected') return <ConnectCard conn={conn} onConnect={oauth.start} onUseKey={useKey} connecting={oauth.pending} compact />
  if (conn.status === 'revoked' || conn.status === 'expired') return <RevokedCard conn={conn} onReconnect={oauth.start} onUseKey={useKey} reconnecting={oauth.pending} />
  return <HostReputeCard conn={conn} onLimit={onLimit} />
}

/** "Clean on 84 blocklists · Postmaster high reputation" and the other status lines. */
function assetStatus(a: Connection['assets'][number], sources: number) {
  if (!a.monitored) return null
  if (a.listings.length) {
    return (
      <span className="row flex-nowrap gap-1.5">
        <Badge tone="warn">{tt(m.listed)}</Badge>
        <span>{tt(m.listings, { count: a.listings.length, sources: a.listings.join(', ') })}</span>
      </span>
    )
  }
  if (a.status === 'pending') return <span className="k">{tt(m.firstCheck)}</span>
  const parts: string[] = []
  if (a.role === 'organization_domain') {
    parts.push(tt(m.clean))
    if (a.dmarcPolicy) parts.push(a.dmarcAligned != null ? tt(m.dmarcPolicyAligned, { policy: a.dmarcPolicy, aligned: trimPct(a.dmarcAligned) }) : tt(m.dmarcPolicy, { policy: a.dmarcPolicy }))
  } else {
    parts.push(tt(m.cleanOn, { count: count(sources) }))
    if (a.postmasterReputation) parts.push(tt(m.postmasterReputation, { level: a.postmasterReputation.toLowerCase(), raw: a.postmasterReputation }))
  }
  return parts.join(' · ')
}

/** Connected (SettingsIntegrations): account, scopes, monitored assets. */
function HostReputeCard({ conn, onLimit }: { conn: Connection; onLimit: (l: Limit | null) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
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
      toast({ title: t(m.added, { asset }), detail: t(m.addedDetail) })
    } catch (e) {
      if (e instanceof ApiError && e.code === 'monitor_limit_reached') {
        const meta = (e.problem.meta ?? {}) as Record<string, unknown>
        onLimit({ asset, limit: Number(meta.limit), used: Number(meta.used), team: String(meta.team ?? conn.teamName), swap: meta.swap as string | undefined, swapRole: meta.swapRole as string | undefined })
      } else {
        toast({ kind: 'error', title: t(m.addFailed, { asset }), detail: errorMessage(e) })
      }
    } finally {
      setAdding(undefined)
    }
  }
  async function syncNow() {
    setSyncing(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/integrations/hostrepute/sync', { params: { path: { workspace: slug } } })
      toast({ kind: 'progress', title: t(m.syncing), duration: 4000 })
      window.setTimeout(() => void refresh(), 2500)
    } catch (e) {
      toast({ kind: 'error', title: t(m.syncFailed), detail: errorMessage(e) })
    } finally {
      setSyncing(false)
    }
  }
  const meta = [
    conn.method === 'api_key' ? t(m.methodApiKey) : 'OAuth',
    conn.connectedAt
      ? conn.connectedBy
        ? t(m.metaConnectedBy, { date: usDate(conn.connectedAt), name: conn.connectedBy })
        : t(m.metaConnected, { date: usDate(conn.connectedAt) })
      : null,
    conn.syncError ? t(m.metaUnreachable) : t(m.metaTokenValid),
    conn.lastSyncAt ? t(m.metaLastSync, { ago: ago(conn.lastSyncAt) }) : t(m.metaFirstSync),
  ]
  return (
    <section className="card" aria-labelledby="h-hostrepute">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="h-hostrepute">
            {t(hub.hostrepute)}
          </h2>
          <span className="k">{t(m.sectionLine)}</span>
        </div>
      </div>
      <div className="pad col gap-3.5">
        <div className="row justify-between">
          <div className="row gap-3">
            <ShieldMark />
            <div className="col gap-px">
              <span className="font-semibold">{t(m.connectedTo, { team: conn.teamName ?? '' })}</span>
              <span className="k">{meta.filter(Boolean).join(' · ')}</span>
            </div>
          </div>
          {conn.syncError ? (
            <Badge tone="warn" dot>
              {t(m.unreachable)}
            </Badge>
          ) : (
            <Badge tone="ok" dot>
              {t(m.statusConnected)}
            </Badge>
          )}
        </div>
        {conn.syncError ? (
          <Note tone="warn">
            {t(m.syncError, { error: conn.syncError, hasTime: conn.syncErrorAt ? 'yes' : 'no', time: conn.syncErrorAt ? clock(conn.syncErrorAt) : '' })}
          </Note>
        ) : null}
        <KeyValues
          className="text-[12.5px]"
          items={[
            [
              t(m.grantedScopes),
              <>
                {scopes.map((s, i) => (
                  <span key={s.key}>
                    {i > 0 ? ' · ' : ''}
                    <span className="mono">{s.key}</span> {s.use}
                  </span>
                ))}
              </>,
            ],
            [t(m.sentToHostRepute), t(m.sentToHostReputeValue)],
            [t(hub.hrAdds), t(hub.hrAddsValue)],
            [t(m.alerts), conn.alertsToNotifications ? t(m.alertsOn) : t(m.alertsOff)],
          ]}
        />
        <Table density="compact" className="hr-assets [&_td]:!whitespace-normal [&_td_.badge]:!whitespace-nowrap" caption={t(m.assetsCaption)}>
          <thead>
            <tr>
              <Th>{t(m.colAsset)}</Th>
              <Th>{t(m.colType)}</Th>
              <Th>{t(m.colMonitoring)}</Th>
              <Th>{t(m.colLastSync)}</Th>
              <Th>{t(m.colStatus)}</Th>
            </tr>
          </thead>
          <tbody>
            {conn.assets.map((a) => (
              <tr key={a.asset} className={a.monitored ? undefined : 'bg-[#FFFBEB]'}>
                <Td>
                  <span className="mono">{a.asset}</span>
                </Td>
                <Td>{roleLabel(a)}</Td>
                <Td>{a.monitored ? <Badge tone="ok">{t(m.monitored)}</Badge> : <Badge tone="warn">{t(m.notMonitored)}</Badge>}</Td>
                <Td>{a.monitored && a.lastSyncAt ? ago(a.lastSyncAt) : <span className="k">–</span>}</Td>
                <Td>
                  {a.monitored ? (
                    assetStatus(a, conn.sources)
                  ) : (
                    <Button size="sm" variant="primary" onClick={() => add(a.asset)} loading={adding === a.asset} disabledReason={reason}>
                      {t(m.addToMonitoring)}
                    </Button>
                  )}
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
        <div className="row gap-2">
          <Button size="sm" onClick={syncNow} loading={syncing} disabledReason={reason}>
            {t(m.syncNow)}
          </Button>
          <ButtonLink size="sm" to="/w/$workspace/deliverability" params={{ workspace: slug }}>
            {t(m.openDeliverability)}
          </ButtonLink>
          <Button size="sm" variant="danger" onClick={() => setDisconnect(true)} disabledReason={reason}>
            {t(m.disconnect)}
          </Button>
        </div>
      </div>
      <DisconnectDialog open={disconnect} onOpenChange={setDisconnect} team={conn.teamName ?? ''} />
    </section>
  )
}

/** Token revoked or expired → Reconnect (HostReputeStates). */
function RevokedCard({ conn, onReconnect, onUseKey, reconnecting }: { conn: Connection; onReconnect: () => void; onUseKey: () => void; reconnecting: boolean }) {
  const t = useT()
  const reason = usePermissionReason('integrations:hostrepute')
  const at = conn.tokenRejectedAt
  return (
    <section className="card" aria-labelledby="h-hr-rev">
      <div className="sec">
        <div className="col gap-0.5">
          <h2 className="h2" id="h-hr-rev">
            {t(m.revokedTitle)}
          </h2>
          <span className="k">{t(m.revokedLine, { team: conn.teamName ?? '', method: conn.method === 'api_key' ? t(m.methodApiKey) : 'OAuth' })}</span>
        </div>
        <Badge tone="warn">{at ? t(m.staleSince, { date: monthDay(at) }) : t(m.staleSinceLastSync)}</Badge>
      </div>
      <div className="pad col gap-3">
        <Note tone="warn" label={t(m.reconnectLabel)} role="alert">
          {at ? t(md.reconnectBodyAt, { date: monthDay(at), time: clock(at) }) : t(md.reconnectBody)}
        </Note>
        <KeyValues
          className="text-[12.5px]"
          items={[
            [t(m.stillWorking), t(m.stillWorkingValue)],
            [t(m.paused), t(m.pausedValue)],
          ]}
        />
        <div className="row gap-2">
          <Button size="sm" variant="primary" onClick={onReconnect} loading={reconnecting} disabledReason={reason}>
            <ShieldMark />
            {t(m.reconnectWith)}
          </Button>
          <Button size="sm" onClick={onUseKey} disabledReason={reason}>
            {t(m.useApiKey)}
          </Button>
        </div>
      </div>
    </section>
  )
}

/** HostRepute's own monitoring limit, shown honestly (HostReputeStates). */
function LimitCard({ limit, onClose }: { limit: Limit; onClose: () => void }) {
  const t = useT()
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
      toast({ title: t(m.swapped, { asset: limit.asset, swap: limit.swap }), detail: t(m.addedDetail) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.swapFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const role = t(m.swapRole, { role: limit.swapRole ?? '', raw: limit.swapRole?.replaceAll('_', ' ') ?? '' })
  return (
    <section className="card" aria-labelledby="h-hr-limit">
      <div className="sec">
        <div className="col gap-0.5">
          <h2 className="h2" id="h-hr-limit">
            {t(m.limitTitle)}
          </h2>
          <span className="k">{t(m.limitAdding, { asset: limit.asset })}</span>
        </div>
        <Badge tone="warn">{t(m.limitUsed, { used: count(limit.used), limit: count(limit.limit) })}</Badge>
      </div>
      <div className="pad col gap-3">
        <Note tone="neutral" className={plainNote} role="status">
          {t(m.limitNote, { team: limit.team, used: count(limit.used), limit: count(limit.limit), asset: limit.asset, usedCount: limit.used })}
        </Note>
        <KeyValues
          className="text-[12.5px]"
          items={[[t(m.options), limit.swap ? t(m.optionsSwap, { swap: limit.swap, role, asset: limit.asset }) : t(m.optionsNoSwap)]]}
        />
        <div className="row gap-2">
          <ButtonAnchor size="sm" href="https://app.hostrepute.com">
            {t(m.managePlan)}
          </ButtonAnchor>
          {limit.swap ? (
            <Button size="sm" onClick={swap} loading={busy}>
              {t(m.swapButton, { swap: limit.swap, asset: limit.asset })}
            </Button>
          ) : null}
          <Button size="sm" onClick={onClose}>
            {t(m.notNow)}
          </Button>
        </div>
      </div>
    </section>
  )
}

/** Disconnect, listing what stops working first (HostReputeStates). */
function DisconnectDialog({ open, onOpenChange, team }: { open: boolean; onOpenChange: (o: boolean) => void; team: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshConnection()
  const [busy, setBusy] = useState(false)
  async function disconnect() {
    setBusy(true)
    try {
      await http.DELETE('/api/v1/workspaces/{workspace}/integrations/hostrepute', { params: { path: { workspace: slug } } })
      await refresh()
      onOpenChange(false)
      toast({ kind: 'deleted', title: t(m.disconnected), detail: t(m.disconnectedDetail, { team }) })
    } catch (e) {
      toast({ kind: 'error', title: t(m.disconnectFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      eyebrow={<span className="k">{t(m.disconnectEyebrow)}</span>}
      title={t(m.disconnectTitle)}
      description={<span className="text-[13.5px]">{t(m.disconnectBody)}</span>}
      className="card border-bad-line"
      hideClose
      dismissible={!busy}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(common.cancel)}</Button>
          <Button variant="danger-primary" onClick={disconnect} loading={busy}>
            {t(m.disconnect)}
          </Button>
        </>
      }
    >
      <span className="text-[12px] font-semibold text-ink-2">{t(m.stopsWorking)}</span>
      <ul className="m-0 flex list-disc flex-col gap-[3px] pl-[18px] text-[13.5px] text-ink-nav">
        <li>{t(m.stopsReadiness)}</li>
        <li>{t(m.stopsPanels)}</li>
        <li>{t(m.stopsAlerts)}</li>
        <li>{t(m.stopsAddToMonitoring)}</li>
      </ul>
      <span className="text-[12px] font-semibold text-ink-2">{t(m.keepsWorking)}</span>
      <ul className="m-0 list-disc pl-[18px] text-[13.5px] text-ink-nav">
        <li>{t(m.keepsEverything)}</li>
      </ul>
    </Dialog>
  )
}
