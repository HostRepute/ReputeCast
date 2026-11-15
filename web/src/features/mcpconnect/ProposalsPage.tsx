import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { mc } from './messages'
import { ConnectionDisclosure } from './parts'
import { mp } from './proposals.messages'

export const proposalBase = '/api/v1/workspaces/{workspace}/mcp-connect/proposals' as const

export function ProposalsPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const [cursor, setCursor] = useState('')
  const now = useNow()
  const q = $api.useQuery('get', proposalBase, { params: { path: { workspace: slug }, query: { cursor } } })
  return <Page title={t(mp.title)} subtitle={t(mp.lead)} actions={<Link className="btn" to="/w/$workspace/settings/connections" params={{ workspace: slug }}>{t(mc.title)}</Link>}>
    <Note tone="info" className="mb-5">{t(mp.noSend)}</Note>
    <div className="mcp-connect-grid"><div className="flex min-w-0 flex-col gap-5">
      {q.error ? <ErrorPanel title={t(mp.unavailable)} actions={<Button onClick={() => void q.refetch()}>{t(mc.retry)}</Button>} /> : !q.data ? <Loading label={t(common.loading)}><Skeleton h={200} /></Loading> : !q.data.items.length ? <EmptyState title={t(mp.empty)}>{t(mp.emptyBody)}</EmptyState> : (['new', 'existing'] as const).map((section) => {
        const items = q.data.items.filter((item) => item.change === (section === 'existing'))
        return items.length ? <section key={section} aria-label={t(section === 'new' ? mp.newDrafts : mp.existing)} className="flex flex-col gap-3">
          <h2 className="h2">{t(section === 'new' ? mp.newDrafts : mp.existing)}</h2>
          {items.map((item) => <Card key={item.id} className="flex min-w-0 flex-col gap-3 p-5 max-md:p-4">
            <div className="flex flex-wrap items-start justify-between gap-3"><h3 className="m-0 min-w-0 text-sm font-semibold [overflow-wrap:anywhere]">{item.name || t(mp.kind, { kind: item.kind })}</h3><Badge tone={item.decision ? 'neutral' : 'warn'}>{t(mp.status, { decision: item.decision })}</Badge></div>
            <p className="k m-0 [overflow-wrap:anywhere]">{item.clientName} · {t(item.verified ? mc.verified : mc.unverified)}</p>
            <p className="k m-0">{t(mc.expires, { date: logTime(item.expiresAt) })}</p>
            {item.purged || (!item.decision && (item.revoked || new Date(item.expiresAt) <= now)) ? <p className="k m-0">{t(mp.blocked, { reason: item.purged ? 'purged' : item.revoked ? 'revoked' : 'expired' })}</p> : null}
            <div><Link className="btn" to="/w/$workspace/proposals/$proposal" params={{ workspace: slug, proposal: item.id }}>{t(mp.review)}</Link></div>
          </Card>)}
        </section> : null
      })}
      {cursor || q.data?.nextCursor ? <nav className="flex flex-wrap gap-2" aria-label={t(mp.title)}>{cursor ? <Button onClick={() => setCursor('')}>{t(mp.back)}</Button> : null}{q.data?.nextCursor ? <Button onClick={() => setCursor(q.data!.nextCursor!)}>{t(mp.next)}</Button> : null}</nav> : null}
    </div><ConnectionDisclosure /></div>
  </Page>
}
