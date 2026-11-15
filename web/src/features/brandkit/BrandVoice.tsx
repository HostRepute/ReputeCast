import { useInfiniteQuery, useQueryClient } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { Pager } from '@/components/ui/Pager'
import { toast } from '@/components/ui/Toast'
import { aiBase, useRefreshAI } from '@/features/ai/api'
import { costLine, dollars, estimateMicros } from '@/features/ai/format'
import { m as aiMessages, ms } from '@/features/ai/messages'
import { usageLine, useAssist, useSuggest } from '@/features/ai/suggest'
import { useSharingReview } from '@/features/ai/SharingReview'
import { useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, shortDate } from '@/lib/format'
import { useDebounced } from '@/lib/hooks'
import { useCan, useWorkspace } from '@/lib/session'
import { mk } from './messages'
import { mv } from './voice.messages'

const base = '/api/v1/workspaces/{workspace}/brand-kit/voice' as const
type Voice = Schemas['BrandVoice']
type Content = Schemas['BrandVoiceContent']
type Suggestion = Schemas['AILearnedVoice']
const fields = ['tone', 'address', 'sentenceLength', 'preferredWords', 'avoidedWords', 'examples'] as const
const limits: Record<keyof Content, number> = { tone: 1000, address: 500, sentenceLength: 500, preferredWords: 1000, avoidedWords: 1000, examples: 2000 }

export function BrandVoiceSection() {
  const { slug } = useWorkspace()
  const t = useT()
  const q = $api.useQuery('get', base, { params: { path: { workspace: slug } } })
  return (
    <section aria-label={t(mv.title)} className="mt-5 min-w-0">
      {q.data ? <VoiceEditor key={slug} voice={q.data} /> : q.isError ? (
        <ErrorPanel title={t(mv.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>{errorMessage(q.error)}</ErrorPanel>
      ) : <Skeleton h={320} />}
    </section>
  )
}

function VoiceEditor({ voice }: { voice: Voice }) {
  const t = useT()
  const { slug } = useWorkspace()
  const canEdit = useCan('templates:edit')
  const qc = useQueryClient()
  const refreshAI = useRefreshAI()
  const [saved, setSaved] = useState(voice)
  const form = useForm<Content>({ defaultValues: voice.content, mode: 'onChange' })
  const draft = { ...saved.content, ...useWatch({ control: form.control }) }
  const [suggestion, setSuggestion] = useState<Suggestion | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [learning, setLearning] = useState(false)
  const dirty = fields.some((key) => draft[key] !== saved.content[key])
  const id = useId()
  // Refetches must not replace local edits. A server-side revision check
  // rejects a stale save, and Reload explicitly adopts the latest version.
  const changed = saved.revision !== voice.revision
  const reset = () => { setSaved(voice); form.reset(voice.content); setSuggestion(null); setError('') }
  const save = async () => {
    if (!(await form.trigger())) throw new Error(t(mv.invalid))
    setBusy(true)
    setError('')
    try {
      const { data } = await http.PUT(base, { params: { path: { workspace: slug } }, body: { baseRevision: suggestion?.baseRevision ?? saved.revision, content: draft, suggestionId: suggestion?.requestId } })
      if (data) { setSaved(data); form.reset(data.content); setSuggestion(null) }
      await qc.invalidateQueries({ queryKey: ['get', base] })
      await refreshAI()
      toast({ kind: 'success', title: t(mv.saved, { revision: data?.revision ?? 0 }) })
    } catch (e) {
      setError(errorMessage(e))
      await qc.invalidateQueries({ queryKey: ['get', base] })
      throw e
    } finally {
      setBusy(false)
    }
  }
  const reject = async () => {
    if (!suggestion) return
    setBusy(true)
    try {
      await http.DELETE(`${base}/suggestions/{suggestion}`, { params: { path: { workspace: slug, suggestion: suggestion.requestId } } })
      reset()
      await refreshAI()
      toast({ kind: 'success', title: t(mv.rejected) })
    } catch (e) { setError(errorMessage(e)) } finally { setBusy(false) }
  }
  return (
    <div className="flex min-w-0 flex-col gap-5">
      <UnsavedChangesGuard dirty={dirty || !!suggestion || learning} description={t(mv.leave)} onSave={canEdit && !learning ? save : undefined} saveLabel={t(mv.save)} />
      <div className="flex flex-wrap items-center gap-2.5">
        <h2 className="ttl" id={`${id}-h`}>{t(mv.title)}</h2>
        <Badge tone={saved.revision ? 'ok' : 'neutral'}>{saved.revision ? t(mv.revision, { revision: saved.revision }) : t(mv.empty)}</Badge>
      </div>
      <div className="grid min-w-0 items-start grid-cols-[minmax(0,1fr)_minmax(280px,360px)] gap-5 max-[1100px]:grid-cols-1">
        <section className="card min-w-0" aria-labelledby={`${id}-edit`}>
          <CardSection title={<span id={`${id}-edit`}>{suggestion ? t(mv.suggested) : t(mv.editorTitle)}</span>} caption={t(mv.lead)} />
          <form className="pad flex min-w-0 flex-col gap-4" onSubmit={(e) => { e.preventDefault(); if (canEdit && !busy && !learning) void save().catch(() => {}) }}>
            {suggestion ? <p className="note i m-0" role="status">{t(mv.suggested)}</p> : null}
            <div className={suggestion ? 'flex flex-col gap-4' : 'grid grid-cols-2 gap-4 max-md:grid-cols-1'}>
            {fields.map((key) => (
              <div key={key} className={suggestion ? 'grid min-w-0 grid-cols-2 gap-4 max-md:grid-cols-1' : ''}>
                {suggestion ? (
                  <div className="min-w-0 rounded-md border border-line-2 bg-ground p-3">
                    <span className="k">{t(mv.before)} · {t(mv[key])}</span>
                    <p className="m-0 mt-1 whitespace-pre-wrap break-words">{saved.content[key] || t(mv.notSet)}</p>
                  </div>
                ) : null}
                <Field label={t(mv[key])} error={form.formState.errors[key]?.message} hint={key === 'examples' ? t(mv.examplesHint) : suggestion ? t(mv.proposed) : undefined}>
                  <Textarea rows={key === 'examples' ? 3 : 2} required={key === 'tone'} readOnly={!canEdit || busy || learning}
                    {...form.register(key, { validate: (value) => z.string().trim().refine((v) => (key !== 'tone' || !!v) && [...v].length <= limits[key]).safeParse(value).success || t(mv.invalid) })} />
                </Field>
              </div>
            ))}
            </div>
            {error ? <div className="note b" role="alert">{error}</div> : null}
            {changed ? <p className="note w" role="status">{t(mv.changed)}</p> : null}
            <div className="flex flex-wrap items-center gap-2">
              {canEdit ? <Button type="submit" variant="primary" disabled={busy || learning || !draft.tone.trim() || (!dirty && !suggestion)} loading={busy}>{suggestion ? t(mv.saveSuggested) : t(mv.save)}</Button> : null}
              {suggestion ? <Button disabled={busy} onClick={() => void reject()}>{t(aiMessages.reject)}</Button> : dirty || changed ? <Button disabled={busy || learning} onClick={reset}>{t(mk.discardChanges)}</Button> : null}
            </div>
            <p className="k m-0">{t(canEdit ? mv.manual : mv.readOnly)}</p>
            {suggestion ? <div className="flex flex-col gap-1 text-ink-3">
              <p className="k m-0">{t(mv.sent, { provider: suggestion.sent.providerName, model: suggestion.sent.modelName, count: count(suggestion.sent.characters), cost: dollars(suggestion.sent.costMicros) })}</p>
              <p className="k m-0">{t(mv.expires, { date: shortDate(suggestion.expiresAt) })}</p>
            </div> : null}
          </form>
        </section>
        <div className="flex min-w-0 flex-col gap-5">
          <VoiceLearning blocked={dirty || !!suggestion || busy} onBusy={setLearning} onSuggestion={(next) => { setSuggestion(next); form.reset(next.content); setError('') }} />
          <VoiceHistory revision={saved.revision} />
        </div>
      </div>
    </div>
  )
}

function VoiceLearning({ blocked, onBusy, onSuggestion }: { blocked: boolean; onBusy: (busy: boolean) => void; onSuggestion: (s: Suggestion) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const assist = useAssist()
  const [reviewSharing, sharingDialog] = useSharingReview(`${assist.provider} · ${assist.model}`)
  const [query, setQuery] = useState('')
  const search = useDebounced(query, 250)
  const [page, setPage] = useState(1)
  const [selected, setSelected] = useState<Record<string, string>>({})
  const [error, setError] = useState('')
  const ids = Object.keys(selected)
  const enabled = assist.can && assist.configured && assist.featureOn('brand_voice')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns', { params: { path: { workspace: slug }, query: { status: 'sent', channel: 'email', q: search, page, pageSize: 10 } } }, { enabled })
  const suggest = useSuggest<Schemas['AILearnVoiceInput'], Suggestion>(async (body, signal) => (await http.POST(`${aiBase}/brand-voice`, { params: { path: { workspace: slug } }, body, signal })).data!)
  const busy = suggest.state.phase === 'working'
  const learn = async () => {
    onBusy(true)
    setError('')
    try {
      const result = await suggest.start({ campaignIds: ids, sharingReviewed: true })
      if (result && !result.refusal) onSuggestion(result)
      if (result?.refusal) setError(result.refusal)
    } finally { onBusy(false) }
  }
  const reason = !assist.can ? mv.noAI : !assist.configured ? mv.noModel : !enabled ? mv.off : blocked ? mv.pendingEdits : undefined
  const unavailable = assist.status?.state === 'key_rejected' ? t(ms.assistKeyRejected) : assist.status?.state === 'cap_reached' ? t(ms.assistCapReached) : undefined
  return (
    <section className="card min-w-0" aria-label={t(mv.learnTitle)}>
      <CardSection title={t(mv.learnTitle)} caption={t(mv.learnLead)} />
      <div className="pad flex min-w-0 flex-col gap-3">
        {assist.loading ? <Skeleton h={80} /> : reason ? <p className="note n m-0">{t(reason)}</p> : null}
        {enabled ? <>
          <p className="k m-0">{assist.provider} · {assist.model}</p>
          {assist.status ? <>
            <p className="k m-0">{t(mv.estimate, { cost: costLine(estimateMicros(60_000, 1600, assist.status.prices)) })}</p>
            <p className="k m-0">{usageLine(t, assist.status)}</p>
          </> : null}
          {unavailable ? <p className="note w m-0" role="status">{unavailable}</p> : null}
          <Field label={t(mv.search)}><Input type="search" value={query} maxLength={200} onChange={(e) => { setQuery(e.target.value); setPage(1) }} disabled={busy} /></Field>
          {q.isPending ? <Skeleton h={140} /> : q.isError ? <ErrorPanel title={t(mv.sourcesFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>{errorMessage(q.error)}</ErrorPanel> : (
            <>
              <div className="flex max-h-72 flex-col gap-2 overflow-y-auto overscroll-contain" role="group" aria-label={t(mv.search)}>
                {!q.data.items.length ? <p className="k m-0">{t(mv.noCampaigns)}</p> : q.data.items.map((campaign) => (
                  <label key={campaign.id} className="flex min-h-11 shrink-0 cursor-pointer items-center gap-2 rounded-md border border-line-2 px-3 py-1.5">
                    <input type="checkbox" checked={!!selected[campaign.id]} disabled={busy || blocked || (!selected[campaign.id] && ids.length === 10)} onChange={(e) => setSelected((old) => {
                      const next = { ...old }
                      if (e.target.checked) next[campaign.id] = campaign.name
                      else delete next[campaign.id]
                      return next
                    })} />
                    <span className="min-w-0 break-words">{campaign.name}</span>
                  </label>
                ))}
              </div>
              <div className="flex flex-wrap gap-2">
                <Button size="sm" disabled={page <= 1 || busy} onClick={() => setPage((p) => p - 1)}>{t(common.back)}</Button>
                <Button size="sm" disabled={page * 10 >= q.data.total || busy} onClick={() => setPage((p) => p + 1)}>{t(common.next)}</Button>
              </div>
            </>
          )}
          <p className="k m-0" role="status">{t(mv.selected, { count: ids.length })}</p>
          {ids.length ? <ul className="m-0 flex list-none flex-col gap-1 p-0">{ids.map((id) => <li key={id} className="flex items-center justify-between gap-2"><span className="min-w-0 break-words">{selected[id]}</span><Button size="sm" disabled={busy || blocked} onClick={() => setSelected((old) => { const next = { ...old }; delete next[id]; return next })}>{t(mk.remove)}</Button></li>)}</ul> : null}
          {suggest.state.phase === 'error' ? <div className="note b" role="alert">{suggest.state.error.message}</div> : error ? <div className="note w" role="alert">{error}</div> : null}
          <Button disabled={blocked || !ids.length || busy || q.isError} disabledReason={unavailable} loading={busy} onClick={() => reviewSharing(learn)}>{t(busy ? mv.learning : mv.learn)}</Button>
        </> : null}
        {sharingDialog}
      </div>
    </section>
  )
}

function VoiceHistory({ revision }: { revision: number }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useInfiniteQuery({
    queryKey: ['get', `${base}/history`, slug, revision],
    initialPageParam: 0,
    queryFn: async ({ pageParam, signal }) => (await http.GET(`${base}/history`, { params: { path: { workspace: slug }, query: { before: pageParam } }, signal })).data!,
    getNextPageParam: (last) => last.beforeRevision || undefined,
  })
  return (
    <section className="card min-w-0" aria-label={t(mv.history)}>
      <CardSection title={t(mv.history)} caption={t(mv.historyLead)} />
      <div className="pad flex flex-col gap-3">
        {q.data && q.data.pages.every((p) => !p.items.length) ? <p className="k m-0">{t(mv.empty)}</p> : null}
        {q.isPending ? <Skeleton h={60} /> : q.isError ? <ErrorPanel title={t(mv.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>{errorMessage(q.error)}</ErrorPanel> : q.data.pages.flatMap((p) => p.items).map((item) => (
          <details key={item.revision} className="min-w-0 border-b border-line-2 pb-2">
            <summary className="min-h-11 cursor-pointer py-2">{t(mv.revision, { revision: item.revision })} · {item.createdBy}{item.createdAt ? ` · ${shortDate(item.createdAt)}` : ''}</summary>
            <dl className="m-0 flex flex-col gap-2">{fields.map((key) => <div key={key}><dt className="k">{t(mv[key])}</dt><dd className="m-0 whitespace-pre-wrap break-words">{item.content[key] || t(mv.notSet)}</dd></div>)}</dl>
          </details>
        ))}
        <Pager query={q} label={t(mv.loadMore)} />
      </div>
    </section>
  )
}
