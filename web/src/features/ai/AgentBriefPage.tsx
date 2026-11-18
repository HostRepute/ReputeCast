import { useNavigate } from '@tanstack/react-router'
import { lazy, Suspense, useRef, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Card, CardHeader } from '@/components/ui/Card'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Checkbox, Field, Select, Textarea } from '@/components/ui/Field'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { VoiceSelection } from '@/features/brandkit/VoiceSelection'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count } from '@/lib/format'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { AgentBudget, AgentPrivacy, agentRuns } from './agent'
import { ag } from './agent.messages'
import { AgentPoolSelection } from './AgentPool'
import { AgentReportSelection } from './AgentReport'
import { md } from './draft.messages'
import { dollars, langName } from './format'
import { useSharingReview } from './SharingReview'
import { Sparkle, useAssist } from './suggest'
import './agent.css'

type Kind = 'campaign' | 'automation' | 'post'
type Trigger = Schemas['AIAgentAutomationSelection']['trigger']
type BriefFields = Pick<Schemas['AIAgentBrief'], 'brief' | 'language' | 'templateId'> & { kind: Kind; links: string; trigger: Trigger; resource: string }
/** randomUUID needs a secure context; self-hosted installs may run on plain HTTP. */
function newRequestKey(): string {
  return Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, '0')).join('')
}
const AgentAudience = lazy(() => import('./AgentAudience').then((m) => ({ default: m.AgentAudience })))

/**
 * The brief an assistant's agent-task card hands over (Start an agent run):
 * the page's history state, for this workspace only, at most 4000 characters.
 */
export function handoffBrief(state: unknown, workspace: string): string {
  const b = (state as { agentBrief?: { workspace?: unknown; text?: unknown } } | undefined)?.agentBrief
  return b && b.workspace === workspace && typeof b.text === 'string' ? b.text.slice(0, 4000) : ''
}

export function AgentBriefPage({ initialKind = 'campaign', initialBrief = '' }: { initialKind?: Kind; initialBrief?: string }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const assist = useAssist()
  const [reviewSharing, sharingDialog] = useSharingReview(`${assist.provider} · ${assist.model}`)
  const form = useForm<BriefFields>({ defaultValues: { kind: initialKind, brief: initialBrief, language: 'en', templateId: '', links: '', trigger: 'joined_list', resource: '' } })
  const kind = useWatch({ control: form.control, name: 'kind' })
  const trigger = useWatch({ control: form.control, name: 'trigger' })
  const [voiceRevision, setVoiceRevision] = useState<number>()
  const [audience, setAudience] = useState<Schemas['SegmentDefinition']>({ groups: [] })
  const [audienceOpen, setAudienceOpen] = useState(false)
  const [fromAudience, setFromAudience] = useState(false)
  const [reportId, setReportId] = useState('')
  const [reportOpen, setReportOpen] = useState(false)
  const [poolId, setPoolId] = useState('')
  const [poolOpen, setPoolOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [created, setCreated] = useState(false)
  const [error, setError] = useState('')
  const locked = useRef(false)
  // One key per brief: resubmitting the same brief after a lost response
  // returns the run already created instead of paying for another plan.
  const attempt = useRef<{ body: string; key: string }>(undefined)
  const canTemplates = useCan('templates:view')
  const canReport = me.permissions.includes('reports:view') && me.permissions.includes('campaigns:view')
  const canCampaign = me.permissions.includes('campaigns:edit') && me.permissions.includes('campaigns:view')
  const canFlow = me.permissions.includes('automations:edit') && me.permissions.includes('automations:view')
  const canPost = me.permissions.includes('website:edit') && me.permissions.includes('website:view')
  const canAudience = useCan('subscribers:view')
  const canSelectAudience = canAudience && me.permissions.includes('segments:edit') && !me.scope.lists?.length
  const aiReason = usePermissionReason('ai:use')
  const editReason = usePermissionReason(kind === 'campaign' ? 'campaigns:edit' : kind === 'post' ? 'website:edit' : 'automations:edit')
  const canSettings = useCan('integrations:view')
  const templates = $api.useQuery('get', '/api/v1/workspaces/{workspace}/templates', { params: { path: { workspace: slug } } }, { enabled: canTemplates })
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } }, { enabled: kind === 'automation' && trigger === 'joined_list' })
  const segments = $api.useQuery('get', '/api/v1/workspaces/{workspace}/segments', { params: { path: { workspace: slug } } }, { enabled: kind === 'automation' && trigger !== 'joined_list' && canAudience })
  const status = assist.status
  const allowed = kind === 'campaign' ? canCampaign : kind === 'post' ? canPost : canFlow
  const ready = assist.ready && status?.agent.enabled && status.agent.capability?.state !== 'unsupported' && status.prices.known && allowed
  const start = (event: React.FormEvent) => form.handleSubmit(async ({ brief, language, templateId, links, resource }) => {
    if (locked.current || !ready) return
    setError('')
    const addresses = links.split('\n').map((v) => v.trim()).filter(Boolean)
    if (kind === 'automation' && trigger !== 'joined_list' && fromAudience && !audience.groups.length) { setError(t(md.invalid)); return }
    reviewSharing(async () => {
      if (locked.current) return
      locked.current = true
      setBusy(true)
      try {
        const body: Schemas['AIAgentBrief'] = {
          sharingReviewed: true, brief: brief.trim(), kind: kind === 'post' ? 'post' : 'campaign', language, voiceRevision, templateId, links: addresses,
          audience: audience.groups.length ? audience : undefined,
          reportId: canReport && reportId ? reportId : undefined,
          poolId: kind !== 'post' && me.permissions.includes('campaigns:view') && poolId ? poolId : undefined,
          automation: kind === 'automation' ? { trigger, fromAudience: trigger !== 'joined_list' && fromAudience, ...(trigger === 'joined_list' ? { listId: resource } : fromAudience ? {} : { segmentId: resource }) } : undefined,
        }
        const serialized = JSON.stringify(body)
        if (attempt.current?.body !== serialized) attempt.current = { body: serialized, key: newRequestKey() }
        const { data } = await http.POST(agentRuns, { params: { path: { workspace: slug } }, body: { ...body, requestKey: attempt.current.key } })
        setCreated(true)
        await navigate({ to: '/w/$workspace/agent/$runId', params: { workspace: slug, runId: data!.id } })
      } catch (err) { setError(errorMessage(err)) } finally { locked.current = false; setBusy(false) }
    })
  })(event)
  return <Page title={t(ag.newBrief)} subtitle={t(ag.title)} actions={<ButtonLink to="/w/$workspace/agent" params={{ workspace: slug }}>{t(ag.runs)}</ButtonLink>}>
    <div className="agent-columns"><div className="agent-main"><Card className="p-5 max-md:p-4">
      <CardHeader title={t(ag.empty)} caption={t(ag.lead)} />
      <form className="mt-5" noValidate onSubmit={(e) => void start(e)}><fieldset disabled={busy} className="m-0 flex min-w-0 flex-col gap-4 border-0 p-0">
        <Field label={t(ag.brief)} hint={t(md.hint)} error={form.formState.errors.brief?.message}><Textarea required rows={5} maxLength={4000} {...form.register('brief', { validate: (v) => z.string().trim().min(1).max(4000).safeParse(v).success || t(md.invalid) })} /></Field>
        <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
          <Field label={t(ag.goal)}><Select {...form.register('kind', { onChange: () => { form.setValue('resource', ''); setFromAudience(false) } })}>{(['campaign', 'automation', 'post'] as const).map((k) => <option key={k} value={k} disabled={!(k === 'campaign' ? canCampaign : k === 'post' ? canPost : canFlow)}>{t(ag.kind, { kind: k })}</option>)}</Select></Field>
          <Field label={t(ag.language)}><Select {...form.register('language')}>{['en', 'vi', 'de', 'es', 'fr'].map((lang) => <option key={lang} value={lang}>{langName(lang)}</option>)}</Select></Field>
          <VoiceSelection value={voiceRevision} onChange={setVoiceRevision} />
          <Field label={t(md.template)}><Select {...form.register('templateId')} disabled={!canTemplates}><option value="">{t(md.blank)}</option>{[...(templates.data?.starters ?? []), ...(templates.data?.templates ?? [])].map((tpl) => <option key={tpl.id} value={tpl.id}>{tpl.name}</option>)}</Select></Field>
        </div>
        {templates.error ? <ErrorPanel title={errorMessage(templates.error)} actions={<Button onClick={() => void templates.refetch()}>{t(common.retry)}</Button>} /> : null}
        {kind !== 'post' && me.permissions.includes('campaigns:view') ? <details onToggle={(e) => { if (e.currentTarget.open) setPoolOpen(true) }}><summary className="min-h-11 cursor-pointer leading-[44px]">{t(ag.poolSelect)}</summary>{poolOpen ? <AgentPoolSelection value={poolId} onChange={setPoolId} /> : null}{poolId ? <p className="k">{t(ag.poolSelected)}</p> : null}</details> : null}
        {canReport ? <details onToggle={(e) => { if (e.currentTarget.open) setReportOpen(true) }}><summary className="min-h-11 cursor-pointer leading-[44px]">{t(ag.report)}</summary>{reportOpen ? <AgentReportSelection value={reportId} onChange={setReportId} /> : null}{reportId ? <p className="k">{t(ag.reportSelected)}</p> : null}</details> : null}
        {canSelectAudience ? <details onToggle={(e) => setAudienceOpen(e.currentTarget.open)}><summary className="min-h-11 cursor-pointer leading-[44px]">{t(ag.localAudience)}</summary><p className="k">{t(ag.audienceHint)}</p>{audienceOpen ? <Suspense fallback={<Skeleton h={100} />}><AgentAudience value={audience} onChange={setAudience} disabled={busy} /></Suspense> : null}</details> : null}
        {kind === 'automation' ? <>
          <Field label={t(ag.trigger)}><Select {...form.register('trigger', { onChange: () => form.setValue('resource', '') })}>{(['joined_list', 'segment_entered', 'segment_left'] as const).map((value) => <option key={value} value={value} disabled={value !== 'joined_list' && !canAudience}>{t(ag.triggerKind, { kind: value })}</option>)}</Select></Field>
          {trigger !== 'joined_list' && canSelectAudience ? <Checkbox label={t(ag.fromAudience)} checked={fromAudience} onChange={(e) => setFromAudience(e.target.checked)} disabled={busy} /> : null}
          {trigger === 'joined_list' || !fromAudience ? <Field label={t(trigger === 'joined_list' ? ag.list : ag.segment)} error={form.formState.errors.resource?.message}><Select required {...form.register('resource', { validate: (v) => kind !== 'automation' || (trigger !== 'joined_list' && fromAudience) || !!v || t(md.invalid) })}><option value="">{t(ag.choose)}</option>{(trigger === 'joined_list' ? lists.data?.items : segments.data?.items)?.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</Select></Field> : null}
          {(trigger === 'joined_list' ? lists.error : segments.error) ? <ErrorPanel title={errorMessage(trigger === 'joined_list' ? lists.error : segments.error)} actions={<Button onClick={() => void (trigger === 'joined_list' ? lists.refetch() : segments.refetch())}>{t(common.retry)}</Button>} /> : null}
          <p className="note i m-0">{t(ag.enrollment)}</p>
        </> : null}
        <Field label={t(md.links)} hint={t(md.linksHint)} error={form.formState.errors.links?.message}><Textarea rows={2} maxLength={41000} {...form.register('links', { validate: (v) => z.array(z.url({ protocol: /^https?$/ }).max(4096)).max(10).safeParse(v.split('\n').map((line) => line.trim()).filter(Boolean)).success || t(md.invalid) })} /></Field>
        {error ? <p role="alert" className="note b m-0">{error}</p> : null}
        {assist.loading ? <Skeleton h={50} /> : !ready ? <div className="note w"><p className="m-0">{aiReason ?? editReason ?? t(ag.unavailable)}</p>{canSettings ? <ButtonLink to="/w/$workspace/settings/integrations/ai" params={{ workspace: slug }} className="mt-3">{t(md.settings)}</ButtonLink> : null}</div> : <div className="flex flex-col gap-2"><p className="k m-0">{assist.provider} · {assist.model}</p><p className="note i m-0">{t(ag.runDisclosure, { calls: count(status.agent.limits.toolCalls), tokens: count(status.agent.limits.tokens), cost: dollars(status.agent.limits.costMicros), monthly: dollars(status.usage.costMicros) })}</p></div>}
        <p className="k m-0">{t(ag.planningCost)}</p>
        {status?.agent.capability?.state === 'unsupported' ? <p className="note w m-0" role="status">{t(ag.capability, { state: 'unsupported' })} {t(ag.toolsHint)}</p> : null}
        <div><Button type="submit" variant="primary" loading={busy} busyLabel={t(ag.creating)} disabled={!ready} disabledReason={aiReason ?? editReason}><Sparkle />{t(ag.planStart)}</Button></div>
      </fieldset></form>
    </Card></div><aside className="agent-aside"><AgentPrivacy />{status?.agent ? <AgentBudget run={{ limits: status.agent.limits, toolCalls: 0, tokens: 0, costMicros: 0, uncertain: false }} /> : null}</aside></div>
    <UnsavedChangesGuard dirty={!created && !busy && (form.formState.isDirty || voiceRevision !== undefined || !!audience.groups.length || !!reportId || !!poolId)} description={t(ag.leaveBrief)} />
    {sharingDialog}
  </Page>
}
