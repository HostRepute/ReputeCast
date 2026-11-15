import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { mc } from './messages'
import { capabilities, ConnectionDisclosure, ConnectionIdentity } from './parts'
import { mp } from './proposals.messages'
import { SourceDialog } from './SourceDialog'

const base = '/api/v1/workspaces/{workspace}/mcp-connect' as const
type Connection = Schemas['MCPConnection']

export function ConnectionsPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const manager = useCan('mcp:manage')
  const campaigns = useCan('campaigns:view')
  const posts = useCan('website:view')
  const templates = useCan('templates:view')
  const audience = useCan('subscribers:view')
  const automations = useCan('automations:view')
  const reports = useCan('reports:view') && campaigns
  const [cursor, setCursor] = useState('')
  const [revoking, setRevoking] = useState<Connection>()
  const [selecting, setSelecting] = useState<Connection>()
  const [notice, setNotice] = useState(false)
  const q = $api.useQuery('get', `${base}/connections`, { params: { path: { workspace: slug }, query: { cursor } } })
  const options = $api.useQuery('get', `${base}/options`, { params: { path: { workspace: slug } } }, { enabled: manager, retry: false })
  const now = useNow()
  const tabs = settingsTabs(slug).map((tab) => tab.to === '/w/$workspace/settings/keys' ? { ...tab, current: true } : tab)
  return <Page title={t(mc.title)} subtitle={t(mc.lead)} tabs={tabs} actions={<><Link className="btn" to="/w/$workspace/proposals" params={{ workspace: slug }}>{t(mp.title)}</Link><Link className="btn" to="/w/$workspace/settings/mcp" params={{ workspace: slug }}>{t(mc.legacy)}</Link></>}>
    {notice ? <Note tone="ok">{t(mc.revokedNotice)}</Note> : null}
    <div className="mcp-connect-grid"><div className="flex min-w-0 flex-col gap-5">
      {options.data ? <Card className="flex flex-col gap-3 p-5 max-md:p-4"><Field label={t(mc.endpoint)} hint={t(mc.endpointHint)}><Input className="mono" readOnly value={options.data.endpoint} onFocus={(e) => e.currentTarget.select()} /></Field></Card> : null}
      {q.error ? <ErrorPanel title={t(mc.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(mc.retry)}</Button>} /> : !q.data ? <Loading label={t(common.loading)} className="card flex flex-col gap-4 p-5"><Skeleton h={28} /><Skeleton h={160} /></Loading> : q.data.items.length === 0 ? <EmptyState title={t(mc.empty)}>{t(mc.emptyBody)}</EmptyState> : q.data.items.map((connection) => {
        const expired = new Date(connection.expiresAt).getTime() <= now.getTime()
        return <Card key={connection.id} className="flex flex-col gap-4 p-5 max-md:p-4">
          <ConnectionIdentity name={connection.clientName} verified={connection.verified} />
          <div className="flex flex-wrap items-center gap-2"><Badge tone={connection.revokedAt || expired ? 'neutral' : 'ok'}>{t(connection.revokedAt ? mc.revoked : expired ? mc.expired : mc.active)}</Badge><span className="k">{t(mc.connectedBy, { name: connection.userName })}</span></div>
          {connection.revokedReason === 'refresh_reuse' ? <Note tone="warn">{t(mc.reused)}</Note> : null}
          <CardHeader level={3} title={t(mc.capabilities)} />
          <ul className="my-0 space-y-1 pl-5">{capabilities.filter((c) => connection.scopes.includes(c.key)).map((c) => <li key={c.key}>{t(c.title)} <code className="text-ink-3">{c.key}</code></li>)}</ul>
          <dl className="mcp-connect-facts"><dt className="text-ink-3">{t(mc.resourceScope)}</dt><dd>{connection.lists.length ? connection.lists.map((id) => options.data?.lists.find((l) => l.id === id)?.name ?? id).join(', ') : t(mc.allLists)}</dd>
            <dt className="text-ink-3">{t(mc.restrictDomains)}</dt><dd>{connection.senderDomains.length ? connection.senderDomains.join(', ') : t(mc.allDomains)}</dd>
            <dt className="text-ink-3">{t(mc.ips)}</dt><dd>{connection.allowedIps.length ? connection.allowedIps.join(', ') : t(mc.anyIP)}</dd></dl>
          <p className="k m-0">{t(mc.expires, { date: logTime(connection.expiresAt) })}</p>
          {!connection.revokedAt && !expired ? <div className="flex flex-wrap gap-2">{(connection.scopes.includes('editorial:read') && (campaigns || posts || templates || audience || automations)) || (connection.scopes.includes('reports:read') && reports) ? <Button onClick={() => setSelecting(connection)}>{t(mp.selected)}</Button> : null}<Button onClick={() => { setNotice(false); setRevoking(connection) }}>{t(mc.revoke)}</Button></div> : null}
        </Card>
      })}
      {cursor || q.data?.nextCursor ? <div className="flex flex-wrap gap-2">{cursor ? <Button onClick={() => setCursor('')}>{t(mc.back)}</Button> : null}{q.data?.nextCursor ? <Button onClick={() => setCursor(q.data!.nextCursor!)}>{t(mc.next)}</Button> : null}</div> : null}
    </div><ConnectionDisclosure /></div>
    {revoking ? <RevokeDialog connection={revoking} onClose={() => setRevoking(undefined)} onRevoked={() => { setRevoking(undefined); setNotice(true) }} /> : null}
    {selecting ? <SourceDialog connection={selecting} onClose={() => setSelecting(undefined)} /> : null}
  </Page>
}

function RevokeDialog({ connection, onClose, onRevoked }: { connection: Connection; onClose: () => void; onRevoked: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [uncertain, setUncertain] = useState(false)
  const [checking, setChecking] = useState(false)
  const revoke = $api.useMutation('delete', `${base}/connections/{id}`, { retry: false, onSuccess: async () => {
    await qc.invalidateQueries({ queryKey: ['get', `${base}/connections`] })
    toast({ kind: 'success', title: t(mc.revokedNotice) })
    onRevoked()
  }, onError: () => setUncertain(true) })
  return <Dialog open onOpenChange={(open) => { if (!open && !revoke.isPending && !checking) onClose() }} dismissible={!revoke.isPending && !checking} title={t(mc.revokeTitle)} description={connection.clientName} footer={<>
    <Button disabled={revoke.isPending || checking} onClick={onClose}>{t(mc.cancel)}</Button>
    <Button variant="danger" loading={revoke.isPending} disabled={uncertain || checking} onClick={() => revoke.mutate({ params: { path: { workspace: slug, id: connection.id } } })}>{t(mc.revoke)}</Button>
  </>}><p className="m-0">{t(mc.revokeBody)}</p>
    {uncertain ? <ErrorPanel title={t(mc.uncertainTitle)} className="mt-4" actions={<Button loading={checking} onClick={async () => {
      setChecking(true)
      try {
        const saved = await qc.fetchQuery({ ...$api.queryOptions('get', `${base}/connections/{id}`, { params: { path: { workspace: slug, id: connection.id } } }), staleTime: 0, retry: false })
        if (saved.revokedAt) { await qc.invalidateQueries({ queryKey: ['get', `${base}/connections`] }); onRevoked() }
        else setUncertain(false)
      } catch {
        setUncertain(true)
      } finally { setChecking(false) }
    }}>{t(mc.retry)}</Button>}>{t(mc.uncertainBody)}</ErrorPanel> : null}
  </Dialog>
}
