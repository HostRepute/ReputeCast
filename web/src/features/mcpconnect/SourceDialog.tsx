import { useRef, useState } from 'react'
import { $api, ApiError, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select } from '@/components/ui/Field'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { useSharingReview } from '@/features/ai/SharingReview'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { logTime } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { mc } from './messages'
import { mp } from './proposals.messages'

const path = '/api/v1/workspaces/{workspace}/mcp-connect/connections/{id}/sources' as const
type SourceKind = Schemas['MCPSourceSelection']['kind']

export function SourceDialog({ connection, onClose }: { connection: Schemas['MCPConnection']; onClose: () => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const [reviewSharing, sharingDialog] = useSharingReview(connection.clientName, true)
  const editorial = connection.scopes.includes('editorial:read')
  const readCampaign = useCan('campaigns:view')
  const canReport = useCan('reports:view') && readCampaign && connection.scopes.includes('reports:read')
  const canCampaign = readCampaign && editorial
  const canPost = useCan('website:view') && editorial
  const canTemplate = useCan('templates:view') && editorial
  const canAudience = useCan('subscribers:view') && editorial
  const canFlow = useCan('automations:view') && editorial
  const canSegment = canAudience && !me.scope.lists?.length
  const [kind, setKind] = useState<SourceKind>(canCampaign ? 'campaign' : canPost ? 'post' : canTemplate ? 'template' : canAudience ? 'list' : canFlow ? 'automation' : 'report')
  const [search, setSearch] = useState('')
  const [term, setTerm] = useState('')
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState('')
  const [shared, setShared] = useState<Schemas['MCPSourceHandle']>()
  const [uncertain, setUncertain] = useState(false)
  const [denied, setDenied] = useState(false)
  const submitting = useRef(false)
  const saved = $api.useQuery('get', path, { params: { path: { workspace: slug, id: connection.id } } }, { retry: false })
  const campaignKind = kind === 'campaign' || kind === 'report'
  const campaigns = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns', { params: { path: { workspace: slug }, query: { q: term, page, pageSize: 20, channel: 'email', ...(kind === 'report' ? { status: 'sent' } : {}) } } }, { enabled: (kind === 'campaign' && canCampaign) || (kind === 'report' && canReport) })
  const posts = $api.useQuery('get', '/api/v1/workspaces/{workspace}/website/posts', { params: { path: { workspace: slug }, query: { q: term, page, pageSize: 20, source: 'web' } } }, { enabled: kind === 'post' && canPost })
  const templates = $api.useQuery('get', '/api/v1/workspaces/{workspace}/templates', { params: { path: { workspace: slug } } }, { enabled: kind === 'template' && canTemplate })
  const voice = $api.useQuery('get', '/api/v1/workspaces/{workspace}/brand-kit/voice', { params: { path: { workspace: slug } } }, { enabled: kind === 'voice' && canCampaign })
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } }, { enabled: kind === 'list' && canAudience })
  const segments = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } }, { enabled: kind === 'segment' && canSegment })
  const flows = $api.useQuery('get', '/api/v1/workspaces/{workspace}/automations', { params: { path: { workspace: slug }, query: { status: 'all' } } }, { enabled: kind === 'automation' && canFlow })
  const q = campaignKind ? campaigns : kind === 'post' ? posts : kind === 'template' ? templates : kind === 'list' ? lists : kind === 'segment' ? segments : kind === 'automation' ? flows : voice
  const localItems = kind === 'template' ? templates.data?.templates.filter((item) => !item.builtin) : kind === 'list' ? lists.data?.items : kind === 'segment' ? segments.data?.items.filter((item) => item.mode === 'builder') : flows.data?.items
  const matches = localItems?.filter((item) => item.name.toLocaleLowerCase().includes(term.toLocaleLowerCase()))
  const voiceItems = voice.data?.revision ? [{ id: me.workspace.id, name: `${t(mp.kind, { kind: 'voice' })} · ${t(mp.revision, { revision: voice.data.revision })}` }] : []
  const items = campaignKind ? campaigns.data?.items.map((c) => ({ id: c.id, name: c.name })) : kind === 'post' ? posts.data?.posts.map((p) => ({ id: p.id, name: p.title })) : kind === 'voice' ? voiceItems : matches?.slice((page - 1) * 20, page * 20)
  const total = campaignKind ? campaigns.data?.total : kind === 'post' ? posts.data?.total : kind === 'voice' ? voiceItems.length : matches?.length
  const select = $api.useMutation('post', path, { retry: false })
  const locked = select.isPending || saved.isFetching || uncertain
  const reset = () => { setSelected(''); setShared(undefined); setDenied(false) }
  const share = () => {
    if (submitting.current || locked || !selected || shared || saved.error) return
    submitting.current = true
    setDenied(false)
    return select.mutateAsync({ params: { path: { workspace: slug, id: connection.id } }, body: { sharingReviewed: true, kind, id: selected } }, {
      onSuccess: (result) => { setShared(result); void saved.refetch() },
      onError: (error) => { if (error instanceof ApiError && [403, 404, 409, 422].includes(error.status)) setDenied(true); else setUncertain(true) },
      onSettled: () => { submitting.current = false },
    })
  }
  return <Dialog open size="lg" title={t(mp.selected)} description={connection.clientName} dismissible={!select.isPending && !saved.isFetching} onOpenChange={(open) => { if (!open && !select.isPending && !saved.isFetching) onClose() }} footer={<>
    <Button disabled={select.isPending || saved.isFetching} onClick={onClose}>{t(mc.cancel)}</Button><Button variant="primary" loading={select.isPending} disabled={locked || !!shared || !selected || !!q.error || !!saved.error} onClick={() => reviewSharing(async () => share())}>{t(mp.select)}</Button>
  </>}>
    <div className="flex min-w-0 flex-col gap-4">
      <Note>{t(kind === 'report' ? mp.reportBody : mp.selectedBody)}</Note>
      {uncertain || saved.error ? <ErrorPanel title={t(mc.uncertainTitle)} actions={<Button loading={saved.isFetching} onClick={async () => {
        const result = await saved.refetch()
        if (!result.isSuccess) return
        const found = result.data.items.find((s) => s.id === selected && s.kind === kind)
        setShared(found)
        setUncertain(false)
      }}>{t(mc.retry)}</Button>}>{t(mp.sourceError)}</ErrorPanel> : null}
      {denied ? <Note tone="bad">{t(mp.sourceDenied)}</Note> : null}
      {shared ? <Note tone="ok">{t(mp.shared, { date: logTime(shared.expiresAt) })}</Note> : null}
      <Field label={t(mp.sourceKind)}><Select disabled={locked} value={kind} onChange={(e) => { setKind(e.target.value as SourceKind); setSearch(''); setTerm(''); setPage(1); reset() }}>
        {canCampaign ? <option value="campaign">{t(mp.kind, { kind: 'campaign' })}</option> : null}{canPost ? <option value="post">{t(mp.kind, { kind: 'post' })}</option> : null}
        {canTemplate ? <option value="template">{t(mp.kind, { kind: 'template' })}</option> : null}{canCampaign ? <option value="voice">{t(mp.kind, { kind: 'voice' })}</option> : null}
        {canAudience ? <option value="list">{t(mp.kind, { kind: 'list' })}</option> : null}{canSegment ? <option value="segment">{t(mp.kind, { kind: 'segment' })}</option> : null}{canFlow ? <option value="automation">{t(mp.kind, { kind: 'automation' })}</option> : null}
        {canReport ? <option value="report">{t(mp.kind, { kind: 'report' })}</option> : null}
      </Select></Field>
      {kind === 'template' || kind === 'voice' ? <Note>{t(mp.referenceOnly)}</Note> : null}
      {kind === 'list' || kind === 'segment' || kind === 'automation' ? <Note>{t(mp.workflowSource)}</Note> : null}
      {kind !== 'voice' ? <form className="flex flex-wrap items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (!locked) { setTerm(search.trim()); setPage(1); reset() } }}>
        <Field label={t(mp.search)} className="min-w-0 flex-1"><Input maxLength={200} value={search} disabled={locked} onChange={(e) => setSearch(e.target.value)} /></Field><Button type="submit" disabled={locked}>{t(mp.find)}</Button>
      </form> : null}
      {q.error ? <ErrorPanel title={t(mp.sourceFailed)} actions={<Button onClick={() => void q.refetch()}>{t(mc.retry)}</Button>} /> : q.isPending ? <Loading label={t(common.loading)}><Skeleton h={48} /></Loading> : items?.length ? <Field label={t(mp.choose)}><Select disabled={locked} value={selected} onChange={(e) => { setSelected(e.target.value); setShared(undefined); setDenied(false) }}>
        <option value="">{t(mp.choose)}</option>{items.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
      </Select></Field> : <p role="status" className="k m-0">{t(mp.noSources)}</p>}
      {page > 1 || (total ?? 0) > 20 ? <nav className="flex flex-wrap items-center gap-2" aria-label={t(mp.choose)}><Button disabled={page === 1 || locked} onClick={() => { setPage(page - 1); reset() }}>{t(mp.previousPage)}</Button><span className="k">{t(mp.page, { page })}</span><Button disabled={page * 20 >= (total ?? 0) || locked} onClick={() => { setPage(page + 1); reset() }}>{t(mp.nextPage)}</Button></nav> : null}
      {saved.data?.items.length ? <ul className="m-0 space-y-2 pl-5">{saved.data.items.map((s) => <li key={s.handle} className="[overflow-wrap:anywhere]">{s.name || t(mp.kind, { kind: s.kind })} · {t(mc.expires, { date: logTime(s.expiresAt) })}</li>)}</ul> : null}
      {sharingDialog}
    </div>
  </Dialog>
}
