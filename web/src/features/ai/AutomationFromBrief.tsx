import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type FormEvent } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { toast } from '@/components/ui/Toast'
import { conditionPhrase, emptyRefs, nodeTitle, pathLabel, typeName, type FlowGraph, type Refs } from '@/features/automations/graph'
import { mg } from '@/features/automations/graph.messages'
import { VoiceSelection } from '@/features/brandkit/VoiceSelection'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { aiBase, useRefreshAI } from './api'
import { ma } from './automation-draft.messages'
import { md } from './draft.messages'
import { DraftEmailReview, DraftUnavailable } from './DraftFromBrief'
import { costLine, estimateMicros, langName } from './format'
import { PendingDrafts } from './PendingDrafts'
import { useSharingReview } from './SharingReview'
import { Sparkle, useAssist, useSuggest, usageLine } from './suggest'

type Brief = Schemas['AIAutomationBriefInput']
type Proposal = Schemas['AIAutomationProposal']
const endpoint = `${aiBase}/drafts/automations` as const

export function supportsAutomationBrief(graph: FlowGraph): boolean {
  const triggers = graph.nodes.filter((n) => n.type === 'trigger')
  const trigger = triggers[0]?.data.trigger
  if (triggers.length !== 1 || !trigger || trigger.kind !== 'joined_list' || !trigger.listId || !trigger.confirmedOnly || trigger.includeImports || trigger.reentry !== 'once' || !trigger.exitOnLeave) return false
  const allowed = new Set(['kind', 'listId', 'confirmedOnly', 'includeImports', 'reentry', 'exitOnLeave'])
  return Object.entries(trigger).every(([key, value]) => allowed.has(key) || !value || (Array.isArray(value) && !value.length))
}

export function AutomationFromBriefDialog({ onClose, target, onAccepted, initialProposal }: {
  onClose: () => void
  target?: { id: string; name: string; listId: string; baseRevision: number; hasUnsaved: () => boolean }
  onAccepted: (id: string) => Promise<void> | void
  initialProposal?: Proposal
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const assist = useAssist('draft')
  const [reviewSharing, sharingDialog] = useSharingReview(`${assist.provider} · ${assist.model}`)
  const canEdit = useCan('automations:edit')
  const editReason = usePermissionReason('automations:edit')
  const canTemplates = useCan('templates:view')
  const qc = useQueryClient()
  const refresh = useRefreshAI()
  const [proposal, setProposal] = useState<Proposal | undefined>(initialProposal)
  const [voiceRevision, setVoiceRevision] = useState<number>()
  const [links, setLinks] = useState('')
  const [error, setError] = useState('')
  const [deciding, setDeciding] = useState(false)
  const [leaving, setLeaving] = useState(false)
  const submitting = useRef(false)
  const form = useForm<Brief>({ defaultValues: { brief: '', name: target?.name ?? '', language: 'en', listId: target?.listId ?? '' } })
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } })
  const templates = $api.useQuery('get', '/api/v1/workspaces/{workspace}/templates', { params: { path: { workspace: slug } } }, { enabled: !target && canTemplates })
  const suggest = useSuggest<Brief, Proposal>(async (body, signal) => (await http.POST(endpoint, { params: { path: { workspace: slug } }, body, signal })).data!)
  const working = suggest.state.phase === 'working'
  const ready = canEdit && assist.ready && assist.featureOn('draft') && lists.isSuccess
  const generate = (event?: FormEvent) => form.handleSubmit(async (values) => {
    if (submitting.current || !ready) return
    setError('')
    if (target?.hasUnsaved()) { setError(t(ma.unsaved)); return }
    const addresses = links.split('\n').map((v) => v.trim()).filter(Boolean)
    if (addresses.length > 10 || addresses.some((v) => !z.url({ protocol: /^https?$/ }).safeParse(v).success)) { setError(t(md.invalid)); return }
    reviewSharing(async () => {
      if (submitting.current) return
      if (target?.hasUnsaved()) { setError(t(ma.unsaved)); return }
      submitting.current = true
      try {
        const result = await suggest.start({ ...values, sharingReviewed: true, links: addresses, voiceRevision, automationId: target?.id, baseRevision: target?.baseRevision ?? 0 })
        if (result) setProposal(result)
        await refresh()
      } finally { submitting.current = false }
    })
  })(event)
  const decide = async (decision: 'accepted' | 'rejected') => {
    if (!proposal || submitting.current) return
    setError('')
    if (decision === 'accepted' && target?.hasUnsaved()) { setError(t(ma.unsaved)); return }
    submitting.current = true
    setDeciding(true)
    try {
      const { data } = await http.POST(`${endpoint}/{proposal}/decision`, { params: { path: { workspace: slug, proposal: proposal.id } }, body: { decision } })
      await Promise.all([refresh(), qc.invalidateQueries({ queryKey: ['get', endpoint] }), qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations'] }), qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/automations/options'] })])
      toast({ kind: 'success', title: t(decision === 'accepted' ? ma.saved : ma.rejected) })
      if (data?.id) await onAccepted(data.id)
      onClose()
    } catch (e) { setError(errorMessage(e)) } finally { submitting.current = false; setDeciding(false) }
  }
  const close = () => {
    if (working || deciding) return
    if (form.formState.isDirty || links || proposal) setLeaving(true)
    else onClose()
  }
  const refs: Refs = {
    ...emptyRefs,
    lists: new Map((lists.data?.items ?? []).map((l) => [l.id, l.name])),
    emails: new Map((proposal?.content.emails ?? []).map((e) => [e.key, { name: e.name, latestVersion: 1, subject: e.subject }])),
  }
  return <>
    <Dialog open onOpenChange={(open) => { if (!open) close() }} title={t(ma.title)} description={t(ma.lead)} size="lg" dismissible={!working && !deciding}
      className="ai-draft-dialog max-h-[calc(100dvh-144px)] max-md:max-h-[calc(100dvh-48px)] [&>.dh]:shrink-0 [&>.db]:min-h-0 [&>.df]:shrink-0"
      footer={working ? <Button onClick={suggest.cancel}>{t(md.stop)}</Button> : proposal ? <>
        <Button disabled={deciding} onClick={close}>{t(md.later)}</Button>
        <Button disabled={!canEdit || deciding} onClick={() => void decide('rejected')}>{t(md.reject)}</Button>
        <Button variant="primary" disabled={!canEdit || deciding} loading={deciding} onClick={() => void decide('accepted')}>{t(md.accept)}</Button>
      </> : <>
        <Button onClick={close}>{t(common.cancel)}</Button>
        <Button variant="primary" disabled={!ready} disabledReason={editReason} onClick={() => void generate()}><Sparkle />{t(md.generate)}</Button>
      </>}>
      <div className="flex min-w-0 flex-col gap-4">
        {error ? <p role="alert" className="note b m-0">{error}</p> : null}
        {suggest.state.phase === 'error' ? <p role="alert" className="note b m-0">{suggest.state.error.message}</p> : null}
        {working ? <div role="status" className="flex flex-col gap-3"><h3 className="m-0">{t(md.generating)}</h3><Skeleton h={180} /><p className="k m-0">{t(ma.waiting)}</p></div> : proposal ? <>
          <div role="status" className="flex flex-wrap items-center justify-between gap-2"><strong>{proposal.name}</strong><Badge tone="info">{t(md.after)}</Badge></div>
          <div className="flex flex-col gap-1">
            <p className="k m-0">{t(md.sent, { model: proposal.sent.modelName, tokens: count(proposal.sent.inputTokens + proposal.sent.outputTokens), cost: costLine(proposal.sent.costMicros).replace(/\.$/, '') })}</p>
            {proposal.sent.voiceRevision ? <p className="k m-0">{t(md.voice, { revision: proposal.sent.voiceRevision })}</p> : null}
            <p className="k m-0">{t(md.expires, { date: shortDate(proposal.expiresAt) })}</p>
          </div>
          <p className="note i m-0">{t(ma.audience)}</p>
          <div className={`grid min-w-0 gap-4 ${proposal.before ? 'grid-cols-2 max-md:grid-cols-1' : ''}`}>
            {proposal.before ? <GraphReview graph={proposal.before} refs={{ ...refs, emails: new Map(proposal.beforeEmails.map((e) => [e.id, { name: e.name, latestVersion: e.version, subject: e.content.subject }])) }} before /> : null}
            <GraphReview graph={proposal.content.graph} refs={refs} />
          </div>
          <section aria-label={t(ma.emails)} className="flex min-w-0 flex-col gap-3">
            <h3 className="m-0 text-[14px]">{t(ma.emails)}</h3>
            {proposal.content.emails.map((email, index) => <details key={email.key} open={index === 0} className="min-w-0 rounded-md border border-line-2 p-3">
              <summary className="min-h-11 cursor-pointer break-words leading-[1.5]">{email.name}</summary>
              <DraftEmailReview content={email} html={proposal.previews[index] ?? ''} before={proposal.beforeEmails.find((e) => e.key === email.key)?.content} />
            </details>)}
          </section>
          <p className="note g m-0">{t(ma.checked)}</p>
          <p className="k m-0">{t(md.previewHint)}</p>
        </> : <>
          <PendingDrafts kind="automations" target={target?.id} onOpen={async (id) => {
            const { data } = await http.GET(`${endpoint}/{proposal}`, { params: { path: { workspace: slug, proposal: id } } })
            if (data?.decision) throw new Error(t(md.alreadyDecided))
            setProposal(data)
          }} />
          <form className="flex min-w-0 flex-col gap-4" onSubmit={(e) => { void generate(e) }}>
            <Field label={t(ma.brief)} hint={t(ma.hint)} error={form.formState.errors.brief?.message}><Textarea rows={4} maxLength={4000} required {...form.register('brief', { validate: (v) => !!v.trim() || t(common.required) })} /></Field>
            <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
              <Field label={t(ma.name)} error={form.formState.errors.name?.message}><Input maxLength={200} readOnly={!!target} required {...form.register('name', { validate: (v) => !!v.trim() || t(common.required) })} /></Field>
              <Field label={t(md.language)}><Select {...form.register('language')}>{(['en', 'vi', 'de', 'es', 'fr'] as const).map((code) => <option key={code} value={code}>{langName(code)}</option>)}</Select></Field>
              <Field label={t(ma.list)} error={form.formState.errors.listId?.message}><Select required {...form.register('listId', { required: t(common.required) })} disabled={!!target || !lists.isSuccess}><option value="">{t(ma.chooseList)}</option>{(lists.data?.items ?? []).map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}</Select></Field>
              <VoiceSelection value={voiceRevision} onChange={setVoiceRevision} />
              {!target && canTemplates ? <Field label={t(md.template)}><Select {...form.register('templateId')}><option value="">{t(md.blank)}</option>{[...(templates.data?.starters ?? []), ...(templates.data?.templates ?? [])].map((tpl) => <option key={tpl.id} value={tpl.id}>{tpl.name}</option>)}</Select></Field> : null}
            </div>
            <Field label={t(md.links)} hint={t(md.linksHint)}><Textarea rows={2} maxLength={41000} value={links} onChange={(e) => setLinks(e.target.value)} /></Field>
          </form>
          {lists.isError ? <ErrorPanel title={errorMessage(lists.error)} actions={<Button onClick={() => void lists.refetch()}>{t(common.retry)}</Button>} /> : null}
          <p className="note i m-0">{t(ma.audience)}</p>
          {target ? <p className="note w m-0">{t(ma.replace)}</p> : null}
          <details><summary className="min-h-11 cursor-pointer leading-[44px]">{t(ma.graph)}</summary><p className="k m-0">{t(ma.scope)}</p></details>
          <p className="k m-0">{t(ma.privacy)}</p>
          {assist.loading ? <Skeleton h={50} /> : !assist.ready || !assist.featureOn('draft') ? <DraftUnavailable assist={assist} /> : <div className="flex flex-col gap-1">
            <p className="k m-0">{assist.provider} · {assist.model}</p>
            <p className="k m-0">{t(md.estimate, { cost: costLine(estimateMicros(80000, 8000, assist.status!.prices)).replace(/\.$/, '') })}</p>
            <p className="k m-0">{usageLine(t, assist.status)}</p>
          </div>}
        </>}
      </div>
    </Dialog>
    {leaving ? <ConfirmDialog open onOpenChange={setLeaving} tone="warn" badge={t(ma.title)} title={t(md.leaveTitle)} description={t(ma.leaveBody)} confirmLabel={t(md.leave)} onConfirm={onClose} /> : null}
    {sharingDialog}
  </>
}

/** Text remains navigable without dragging or interpreting a canvas. */
export function GraphReview({ graph, refs, before = false, afterLabel }: { graph: FlowGraph; refs: Refs; before?: boolean; afterLabel?: string }) {
  const t = useT()
  const label = before ? t(md.before) : afterLabel ?? t(md.after)
  return <section aria-label={`${t(ma.graph)} · ${label}`} className={`min-w-0 rounded-md border p-3 ${before ? 'border-line-2 bg-ground' : 'border-ok-line bg-ok-tint'}`}>
    <h3 className="m-0 mb-3 text-[13px]">{label}</h3>
    <ol className="m-0 flex flex-col gap-3 pl-5">
      {graph.nodes.map((node) => <li key={node.id} className="break-words">
        <div className="k">{typeName(node.type)}</div>
        <strong>{nodeTitle(graph, node, refs)}</strong>
        {node.data.condition ? <p className="m-0 text-xs">{conditionPhrase(graph, node.data.condition, refs)}</p> : null}
        <ul className="m-0 pl-4">{graph.edges.filter((e) => e.source === node.id).map((edge) => {
          const next = graph.nodes.find((n) => n.id === edge.target)
          return <li key={edge.id} className="text-xs">{edge.data.path ? `${pathLabel(edge.data.path)} → ` : ''}{t(ma.next, { steps: next ? nodeTitle(graph, next, refs) : t(mg.aRemovedStep) })}</li>
        })}</ul>
        {node.type === 'exit' ? <span className="k">{t(ma.end)}</span> : null}
      </li>)}
    </ol>
    <details className="mt-3"><summary className="min-h-11 cursor-pointer leading-[44px]">{t(ma.exact)}</summary><ScrollPre vertical label={t(ma.exact)} className="m-0 max-h-64 overflow-auto whitespace-pre-wrap break-all text-xs">{JSON.stringify(graph, null, 2)}</ScrollPre></details>
  </section>
}
