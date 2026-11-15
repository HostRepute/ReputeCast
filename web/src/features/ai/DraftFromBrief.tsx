import { useQueryClient } from '@tanstack/react-query'
import { useMemo, useRef, useState, type FormEvent } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { ScrollPre } from '@/components/ui/ScrollPre'
import { toast } from '@/components/ui/Toast'
import { VoiceSelection } from '@/features/brandkit/VoiceSelection'
import { stripActive } from '@/features/composer/inert'
import { diffWords } from '@/features/languages/compare'
import { mf } from '@/features/automations/flow.messages'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { aiBase, useRefreshAI } from './api'
import { md } from './draft.messages'
import { costLine, estimateMicros, langName } from './format'
import { ms } from './messages'
import { PendingDrafts } from './PendingDrafts'
import { useSharingReview } from './SharingReview'
import { Sparkle, useAssist, useSuggest, usageLine, type Assist } from './suggest'

type Brief = Schemas['AICampaignBriefInput']
type Proposal = Schemas['AICampaignProposal']
const endpoint = `${aiBase}/drafts/campaigns` as const

/** A one-shot proposal. Generation never creates a product resource. */
export function DraftFromBriefDialog({ onClose, target, onAccepted, initialVoiceRevision, initialProposal }: {
  onClose: () => void
  target?: { id: string; name: string; baseRevision: number; hasUnsaved: () => boolean }
  onAccepted: (id: string) => Promise<void> | void
  initialVoiceRevision?: number
  initialProposal?: Proposal
}) {
  const { slug } = useWorkspace()
  const t = useT()
  const assist = useAssist()
  const [reviewSharing, sharingDialog] = useSharingReview(`${assist.provider} · ${assist.model}`)
  const canEdit = useCan('campaigns:edit')
  const editReason = usePermissionReason('campaigns:edit')
  const canTemplates = useCan('templates:view')
  const qc = useQueryClient()
  const refresh = useRefreshAI()
  const [proposal, setProposal] = useState<Proposal | undefined>(initialProposal)
  const [voiceRevision, setVoiceRevision] = useState<number | undefined>(initialVoiceRevision)
  const [links, setLinks] = useState('')
  const [error, setError] = useState('')
  const [deciding, setDeciding] = useState(false)
  const [leaving, setLeaving] = useState(false)
  const submitting = useRef(false)
  const form = useForm<Brief>({ defaultValues: { brief: '', name: target?.name ?? '', language: 'en', templateId: '' } })
  const templates = $api.useQuery('get', '/api/v1/workspaces/{workspace}/templates', { params: { path: { workspace: slug } } }, { enabled: !target && canTemplates })
  const suggest = useSuggest<Brief, Proposal>(async (body, signal) => (await http.POST(endpoint, { params: { path: { workspace: slug } }, body, signal })).data!)
  const working = suggest.state.phase === 'working'
  const ready = canEdit && assist.ready && assist.featureOn('draft')
  const generate = (event?: FormEvent) => form.handleSubmit(async (values) => {
    if (submitting.current || !ready) return
    setError('')
    if (target?.hasUnsaved()) { setError(t(md.unsaved)); return }
    const addresses = links.split('\n').map((v) => v.trim()).filter(Boolean)
    if (addresses.length > 10 || addresses.some((v) => !z.url({ protocol: /^https?$/ }).safeParse(v).success)) { setError(t(md.invalid)); return }
    reviewSharing(async () => {
      if (submitting.current) return
      if (target?.hasUnsaved()) { setError(t(md.unsaved)); return }
      submitting.current = true
      try {
        const result = await suggest.start({ ...values, sharingReviewed: true, links: addresses, voiceRevision, campaignId: target?.id, baseRevision: target?.baseRevision ?? 0 })
        if (result) setProposal(result)
        await refresh()
      } finally { submitting.current = false }
    })
  })(event)
  const decide = async (decision: 'accepted' | 'rejected') => {
    if (!proposal || submitting.current) return
    setError('')
    if (decision === 'accepted' && target?.hasUnsaved()) { setError(t(md.unsaved)); return }
    submitting.current = true
    setDeciding(true)
    try {
      const { data } = await http.POST(`${endpoint}/{proposal}/decision`, { params: { path: { workspace: slug, proposal: proposal.id } }, body: { decision } })
      await Promise.all([refresh(), qc.invalidateQueries({ queryKey: ['get', endpoint] }), qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns'] })])
      toast({ kind: 'success', title: t(decision === 'accepted' ? md.saved : md.rejected) })
      if (data?.id) await onAccepted(data.id)
      onClose()
    } catch (e) { setError(errorMessage(e)) } finally { submitting.current = false; setDeciding(false) }
  }
  const close = () => {
    if (working || deciding) return
    if (form.formState.isDirty || links || proposal) setLeaving(true)
    else onClose()
  }
  return <>
    <Dialog open onOpenChange={(open) => { if (!open) close() }} title={t(md.title)} description={t(md.lead)} size="lg" dismissible={!working && !deciding}
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
        {working ? <div role="status" className="flex flex-col gap-3"><h3 className="m-0">{t(md.generating)}</h3><Skeleton h={180} /><p className="k m-0">{t(md.waiting)}</p></div> : proposal ? <DraftReview proposal={proposal} /> : <>
          <PendingDrafts kind="campaigns" target={target?.id} onOpen={async (id) => {
            const { data } = await http.GET(`${endpoint}/{proposal}`, { params: { path: { workspace: slug, proposal: id } } })
            if (data?.decision) throw new Error(t(md.alreadyDecided))
            setProposal(data)
          }} />
          <form className="flex min-w-0 flex-col gap-4" onSubmit={(e) => { void generate(e) }}>
            <Field label={t(md.brief)} hint={t(md.hint)} error={form.formState.errors.brief?.message}>
              <Textarea rows={4} maxLength={4000} required {...form.register('brief', { validate: (v) => !!v.trim() || t(md.invalid) })} />
            </Field>
            <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
              <Field label={t(md.name)} error={form.formState.errors.name?.message}><Input maxLength={200} readOnly={!!target} required {...form.register('name', { validate: (v) => !!v.trim() || t(md.invalid) })} /></Field>
              <Field label={t(md.language)}><Select {...form.register('language')}>{(['en', 'vi', 'de', 'es', 'fr'] as const).map((code) => <option key={code} value={code}>{langName(code)}</option>)}</Select></Field>
              {!target && canTemplates ? <Field label={t(md.template)}><Select {...form.register('templateId')}><option value="">{t(md.blank)}</option>{[...(templates.data?.starters ?? []), ...(templates.data?.templates ?? [])].map((tpl) => <option key={tpl.id} value={tpl.id}>{tpl.name}</option>)}</Select></Field> : null}
              <VoiceSelection value={voiceRevision} onChange={setVoiceRevision} />
            </div>
            <Field label={t(md.links)} hint={t(md.linksHint)}><Textarea rows={2} maxLength={41000} value={links} onChange={(e) => setLinks(e.target.value)} /></Field>
          </form>
          <p className="note i m-0">{t(md.privacy)}</p>
          {assist.loading ? <Skeleton h={50} /> : !assist.ready || !assist.featureOn('draft') ? <DraftUnavailable assist={assist} /> : <div className="flex flex-col gap-1">
            <p className="k m-0">{assist.provider} · {assist.model}</p>
            <p className="k m-0">{t(md.estimate, { cost: costLine(estimateMicros(70000, 8000, assist.status!.prices)).replace(/\.$/, '') })}</p>
            <p className="k m-0">{usageLine(t, assist.status)}</p>
          </div>}
        </>}
      </div>
    </Dialog>
    {leaving ? <ConfirmDialog open onOpenChange={setLeaving} tone="warn" badge={t(md.title)} title={t(md.leaveTitle)} description={t(md.leaveBody)} confirmLabel={t(md.leave)} onConfirm={onClose} /> : null}
    {sharingDialog}
  </>
}

export function DraftUnavailable({ assist }: { assist: Assist }) {
  const t = useT()
  const { slug } = useWorkspace()
  const denied = usePermissionReason('ai:use')
  const canSettings = useCan('integrations:view')
  const state = assist.status?.state
  const message = denied ?? (state === 'cap_reached'
    ? t(ms.capMenuAlone, { date: shortDate(assist.status!.usage.resetsAt) })
    : state === 'key_rejected' ? t(ms.keyRejectedMenuAlone, { provider: assist.provider }) : t(md.unavailable))
  return <div role="status" className="note w m-0 flex flex-col gap-2"><span>{message}</span>
    {canSettings ? <ButtonLink to="/w/$workspace/settings/integrations/ai" params={{ workspace: slug }}>{t(md.settings)}</ButtonLink> : null}
  </div>
}

function DraftReview({ proposal }: { proposal: Proposal }) {
  const t = useT()
  return <div className="flex min-w-0 flex-col gap-4">
    <div className="flex flex-wrap items-center justify-between gap-2" role="status"><strong>{proposal.name}</strong><Badge tone="info">{t(md.after)}</Badge></div>
    <div className="flex flex-col gap-1">
      <p className="k m-0">{t(md.sent, { model: proposal.sent.modelName, tokens: count(proposal.sent.inputTokens + proposal.sent.outputTokens), cost: costLine(proposal.sent.costMicros).replace(/\.$/, '') })}</p>
      {proposal.sent.voiceRevision ? <p className="k m-0">{t(md.voice, { revision: proposal.sent.voiceRevision })}</p> : null}
      <p className="k m-0">{t(md.expires, { date: shortDate(proposal.expiresAt) })}</p>
    </div>
    <DraftEmailReview content={proposal.content} before={proposal.before} html={proposal.html} />
    <section aria-label={t(md.checks)} className="flex flex-wrap items-center gap-2"><span className="k">{t(md.checks)}</span>{proposal.subjectCheck.heuristics.length ? proposal.subjectCheck.heuristics.map((issue) => <Badge key={issue} tone="warn">{issue}</Badge>) : <Badge tone="ok">{t(md.checked)}</Badge>}</section>
    <p className="k m-0">{t(md.previewHint)}</p>
  </div>
}

/** The same exact, text-labelled diff for campaign and flow email decisions. */
export function DraftEmailReview({ content, before, html, labels, afterLabel }: { content: Schemas['AIDraftEmail']; before?: Schemas['AIDraftEmail']; html: string; labels?: { subject: string; preheader: string }; afterLabel?: string }) {
  const t = useT()
  const changes = useMemo(() => before ? diffWords(before.mjml, content.mjml) : [], [before, content.mjml])
  return <div className="flex min-w-0 flex-col gap-3">
    <div className={`grid min-w-0 gap-4 ${before ? 'grid-cols-2 max-md:grid-cols-1' : ''}`}>
      {[...(before ? [{ value: before, saved: true }] : []), { value: content, saved: false }].map(({ value, saved }) => <section key={String(saved)} aria-label={saved ? t(md.before) : afterLabel ?? t(md.after)} className={`min-w-0 rounded-md border p-3 ${saved ? 'border-line-2 bg-ground' : 'border-ok-line bg-ok-tint'}`}>
        <h3 className="m-0 mb-3 text-[13px]">{saved ? t(md.before) : afterLabel ?? t(md.after)}</h3>
        <dl className="m-0 flex flex-col gap-3">{(['subject', 'preheader'] as const).map((key) => <div key={key}><dt className="k">{labels?.[key] ?? t(md[key])}{before && before[key] !== content[key] ? <> · {t(mf.kindChanged)}</> : null}</dt><dd className="m-0 break-words">{value[key] || '—'}</dd></div>)}</dl>
        <details className="mt-3"><summary className="min-h-11 cursor-pointer leading-[44px]">{t(md.source)}</summary><ScrollPre vertical label={t(md.source)} className="m-0 max-h-64 overflow-auto whitespace-pre-wrap break-all text-xs">{value.mjml}</ScrollPre></details>
      </section>)}
    </div>
    {before ? <details><summary className="min-h-11 cursor-pointer leading-[44px]">{t(md.changes)}</summary>
      <div className="mb-2 flex gap-2"><Badge tone="bad">{t(mf.kindRemoved)}</Badge><Badge tone="ok">{t(mf.kindAdded)}</Badge></div>
      <ScrollPre vertical label={t(md.changes)} className="m-0 max-h-80 overflow-auto whitespace-pre-wrap break-all text-xs">{changes.map((piece, i) => piece.kind === 'same' ? <span key={i}>{piece.text}</span> : piece.kind === 'add' ? <ins key={i} className="bg-ok-tint text-ok"><span className="sr-only">{t(mf.kindAdded)}: </span>{piece.text}</ins> : <del key={i} className="bg-bad-tint text-bad"><span className="sr-only">{t(mf.kindRemoved)}: </span>{piece.text}</del>)}</ScrollPre>
    </details> : null}
    <DraftEmailPreview html={html} />
  </div>
}

/** Content under review has no network, script, form or navigation capability. */
export function DraftEmailPreview({ html }: { html: string }) {
  const t = useT()
  const preview = useMemo(() => {
    const doc = new DOMParser().parseFromString(html, 'text/html')
    stripActive(doc.documentElement)
    const policy = doc.createElement('meta')
    policy.httpEquiv = 'Content-Security-Policy'
    policy.content = "default-src 'none'; style-src 'unsafe-inline'; img-src data:; form-action 'none'; base-uri 'none'"
    doc.head.prepend(policy)
    doc.querySelectorAll('a,area').forEach((el) => el.removeAttribute('href'))
    return `<!doctype html>${doc.documentElement.outerHTML}`
  }, [html])
  return <iframe title={t(md.preview)} sandbox="" srcDoc={preview} className="keep-light box-border h-[380px] w-full rounded-md border border-line-2" />
}
