import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useRef, useState } from 'react'
import { $api, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { DraftEmailReview } from '@/features/ai/DraftFromBrief'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { logTime } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { mc } from './messages'
import { ConnectionDisclosure, ConnectionIdentity } from './parts'
import { mp } from './proposals.messages'
import { proposalBase } from './ProposalsPage'
import { WorkflowReview } from './WorkflowReview'

type Decision = Schemas['MCPProposalDecision']['decision']
const labels = { kept: mp.keep, discarded: mp.discard, accepted: mp.accept, rejected: mp.reject }
const descriptions = { kept: mp.keepBody, discarded: mp.discardBody, accepted: mp.acceptBody, rejected: mp.rejectBody }

export function ProposalPage({ id }: { id: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const q = $api.useQuery('get', `${proposalBase}/{id}`, { params: { path: { workspace: slug, id } } }, { retry: false, gcTime: 0, refetchOnWindowFocus: false })
  const [confirm, setConfirm] = useState<Decision>()
  const [busy, setBusy] = useState(false)
  const [uncertain, setUncertain] = useState(false)
  const [copyNotice, setCopyNotice] = useState<'copied' | 'copyFailed'>()
  const submitting = useRef(false)
  const heading = useRef<HTMLDivElement>(null)
  const now = useNow(1000)
  const p = q.data
  const workflow = p?.kind === 'automation' || p?.kind === 'segment'
  const label = (decision: Decision) => workflow && !p?.targetId && decision === 'kept' ? mp.keepBundle : workflow && !p?.targetId && decision === 'discarded' ? mp.discardBundle : labels[decision]
  const description = (decision: Decision) => workflow && decision === 'kept' ? mp.keepWorkflow : workflow && decision === 'discarded' ? mp.discardWorkflow : descriptions[decision]
  const blocked = p?.blocked || (!p?.decision && p && new Date(p.expiresAt) <= now ? 'expired' : '')
  const resourceID = p?.resultId || p?.targetId
  const refreshRelated = () => qc.invalidateQueries({ predicate: (query) => {
    const path = query.queryKey[1]
    return typeof path === 'string' && (path === proposalBase || ['/website', '/campaigns', '/segments', '/automations', '/approvals'].some((part) => path.includes(part)))
  } })
  const close = () => {
    if (submitting.current) return
    setConfirm(undefined)
    requestAnimationFrame(() => { if (document.activeElement === document.body) heading.current?.focus() })
  }
  const check = async () => {
    const result = await q.refetch()
    if (!result.isSuccess) return
    setUncertain(false)
    if (result.data.decision || result.data.blocked || !result.data.canDecide) {
      setConfirm(undefined)
      requestAnimationFrame(() => heading.current?.focus())
    }
    if (result.data.decision) await refreshRelated()
  }
  const decide = async () => {
    if (!confirm || submitting.current || uncertain || !p?.canDecide || p.decision || q.error || ((confirm === 'kept' || confirm === 'accepted') && blocked)) return
    submitting.current = true
    setBusy(true)
    try {
      await http.POST(`${proposalBase}/{id}/decision`, { params: { path: { workspace: slug, id } }, body: { decision: confirm } })
      const saved = await q.refetch()
      if (saved.isSuccess) { setUncertain(false); setConfirm(undefined); requestAnimationFrame(() => heading.current?.focus()) }
      else setUncertain(true)
      await refreshRelated()
    } catch { setUncertain(true) }
    finally { submitting.current = false; setBusy(false) }
  }
  const recovery = uncertain || q.error ? <ErrorPanel title={t(uncertain ? mc.uncertainTitle : mp.unavailable)} actions={<Button loading={q.isFetching} disabled={busy} onClick={() => void check()}>{t(mc.retry)}</Button>}>{t(uncertain ? mc.uncertainBody : mp.unavailableBody)}</ErrorPanel> : null
  return <Page title={p?.name || t(mp.review)} subtitle={t(mp.lead)} actions={<Link className="btn" to="/w/$workspace/proposals" params={{ workspace: slug }}>{t(mp.title)}</Link>}>
    {!confirm ? recovery : null}
    {!p ? q.isPending ? <Loading label={t(common.loading)}><Skeleton h={300} /></Loading> : null : <div className="mcp-connect-grid">
      <div className="flex min-w-0 flex-col gap-5">
        <Card className="flex min-w-0 flex-col gap-4 p-5 max-md:p-4">
          <ConnectionIdentity name={p.clientName} verified={p.verified} />
          <div ref={heading} tabIndex={-1} role="status" className="flex flex-wrap items-center gap-2 rounded"><Badge tone={p.decision ? 'neutral' : 'warn'}>{t(mp.status, { decision: p.decision })}</Badge><span className="k">{t(mp.kind, { kind: p.kind })} · {t(p.targetId ? mp.existing : mp.newDrafts)}</span></div>
          <p className="k m-0">{t(mp.attribution, { name: p.userName, date: logTime(p.createdAt) })}</p>
          <p className="k m-0">{p.targetId ? <>{t(mp.revision, { revision: p.baseRevision })} · </> : null}{t(mc.expires, { date: logTime(p.expiresAt) })}</p>
          <Note tone="info">{t(mp.noSend)}</Note>
          {blocked ? <Note tone="warn">{t(mp.blocked, { reason: blocked })}</Note> : null}
          {!p.canDecide && !p.decision ? <Note tone="warn">{t(mp.permission)}</Note> : null}
          {blocked !== 'purged' ? <>
            {workflow ? <WorkflowReview proposal={p} /> : <><DraftEmailReview content={p.email} before={p.targetId ? p.before : undefined} html={p.html} afterLabel={p.decision ? t(mp.status, { decision: p.decision }) : undefined} labels={p.kind === 'post' ? { subject: t(mp.postTitle), preheader: t(mp.postExcerpt) } : undefined} /><p className="k m-0">{t(mp.checks)}</p></>}
          </> : null}
          {!p.decision ? <div className="flex flex-wrap gap-2">{(p.targetId ? ['accepted', 'rejected'] as const : ['kept', 'discarded'] as const).map((decision) => <Button key={decision} variant={decision === 'kept' || decision === 'accepted' ? 'primary' : decision === 'discarded' ? 'danger' : 'default'} disabled={!p.canDecide || busy || uncertain || !!q.error || ((decision === 'kept' || decision === 'accepted') && !!blocked)} onClick={() => setConfirm(decision)}>{t(label(decision))}</Button>)}</div> : null}
          {resourceID ? <div>{p.kind === 'post'
            ? <Link className="btn" to="/w/$workspace/forms/website/posts/$post" params={{ workspace: slug, post: resourceID }}>{t(mp.openPost)}</Link>
            : p.kind === 'automation' ? <Link className="btn" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: resourceID }}>{t(mp.openFlow)}</Link>
            : p.kind === 'segment' ? <Link className="btn" to="/w/$workspace/audience/segments/$segmentId" params={{ workspace: slug, segmentId: resourceID }}>{t(mp.openSegment)}</Link>
            : <Link className="btn" to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: resourceID }}>{t(mp.openCampaign)}</Link>}</div> : null}
          {blocked && blocked !== 'purged' ? <div><Button onClick={async () => { try { await navigator.clipboard.writeText(workflow ? JSON.stringify(p.workflow, null, 2) : p.email.mjml); setCopyNotice('copied') } catch { setCopyNotice('copyFailed') } }}>{t(workflow ? mp.copyWorkflow : mp.copy)}</Button><p role="status" className="k">{copyNotice ? t(workflow && copyNotice === 'copied' ? mp.workflowCopied : mp[copyNotice]) : null}</p></div> : null}
        </Card>
        <p className="k m-0">{t(mp.retention)}</p>
      </div><ConnectionDisclosure />
    </div>}
    {confirm && p ? <Dialog open dismissible={!busy && !q.isFetching} onOpenChange={(open) => { if (!open) close() }} title={t(label(confirm))} description={p.name} footer={<>
      <Button disabled={busy || q.isFetching} onClick={close}>{t(mc.cancel)}</Button><Button variant={confirm === 'discarded' ? 'danger-primary' : 'primary'} loading={busy} disabled={uncertain || !!q.error || q.isFetching || !p.canDecide || !!p.decision || ((confirm === 'kept' || confirm === 'accepted') && !!blocked)} onClick={() => void decide()}>{t(label(confirm))}</Button>
    </>}><p className="m-0">{t(description(confirm))}</p>{blocked ? <Note tone="warn" className="mt-4">{t(mp.blocked, { reason: blocked })}</Note> : null}<div className="mt-4">{recovery}</div></Dialog> : null}
  </Page>
}
