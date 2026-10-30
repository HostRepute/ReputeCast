import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { toast } from '@/components/ui/Toast'
import { useT, type MessageDescriptor } from '@/i18n'
import { clock, dayMonthDate, dayMonthYear, money } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { categoryBadge, fallbackSentence, FilledBody, statusBadge, testStatusWord, throwIfForbidden, WhatsAppIcon, type Template } from './common'
import { CheckRow } from './Checks'
import { m } from './campaign.messages'
import '@/features/push/push.css'
import './whatsapp.css'

type Campaign = Schemas['WhatsAppCampaign']
type Draft = Omit<Schemas['WhatsAppCampaignUpdate'], 'version'>

const statusWord: Partial<Record<Campaign['status'], MessageDescriptor>> = {
  approval: m.statusApproval,
  approved: m.statusApproved,
  scheduled: m.statusScheduled,
  sending: m.statusSending,
  paused: m.statusPaused,
  sent: m.statusSent,
  cancelled: m.statusCancelled,
  failed: m.statusFailed,
}

const catWord: Record<Template['category'], MessageDescriptor> = { marketing: m.catMarketing, utility: m.catUtility, authentication: m.catAuthentication }

function toDraft(c: Campaign): Draft {
  return {
    name: c.name,
    templateId: c.templateId ?? '',
    variables: c.variables.map((v) => ({ ...v })),
    lists: c.audience.lists.map((l) => l.listId),
    segments: [...c.segments],
    mode: c.schedule.mode,
    sendAt: c.schedule.sendAt,
    localDate: c.schedule.localDate,
    localMinute: c.schedule.localMinute,
    overLimit: c.overLimit,
  }
}

/** "09:00" for a minute of the day. */
function hhmm(minute: number): string {
  return `${String(Math.floor(minute / 60)).padStart(2, '0')}:${String(minute % 60).padStart(2, '0')}`
}

/** A WhatsApp campaign: template, variables, audience and schedule, with its preview and checks. */
export function WhatsAppCampaignPage({ campaignId }: { campaignId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId } } })
  throwIfForbidden(q.error)
  const crumbs = [
    { label: t(m.campaignsCrumb), to: '/w/$workspace/campaigns' as const, params: { workspace: slug } },
    { label: q.data?.name ?? '…' },
  ]
  return (
    <Page title={q.data?.name ?? t(m.loading)} documentTitle={q.data ? t(m.docTitle, { name: q.data.name }) : t(m.loading)} breadcrumb={crumbs}>
      {q.data ? (
        <Composer key={q.data.id} data={q.data} />
      ) : q.isError ? (
        <ErrorPanel
          title={t(m.errorTitle)}
          reference={problemReference(q.error)}
          actions={
            <>
              <Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>
              <ButtonLink to="/w/$workspace/campaigns" params={{ workspace: slug }}>
                {t(m.campaignsCrumb)}
              </ButtonLink>
            </>
          }
        >
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <Loading label={t(m.loading)}>
          <Skeleton w="100%" h={420} />
        </Loading>
      )}
    </Page>
  )
}

function Composer({ data }: { data: Campaign }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const key = ['get', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}', { params: { path: { workspace: slug, campaignId: data.id } } }] as const
  const editable = data.access.canEdit && (data.status === 'draft' || data.status === 'approval' || data.status === 'approved')
  const [draft, setDraft] = useState<Draft>(() => toDraft(data))
  const [saved, setSaved] = useState(() => JSON.stringify(toDraft(data)))
  const [saving, setSaving] = useState<'idle' | 'saving' | 'error'>('idle')
  const [testing, setTesting] = useState(false)
  const [rates, setRates] = useState(false)
  const version = useRef(data.version)
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}')
  const review = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/review', { params: { path: { workspace: slug, campaignId: data.id } } })
  const set = <K extends keyof Draft>(k: K, v: Draft[K]) => setDraft((d) => ({ ...d, [k]: v }))
  const current = JSON.stringify(draft)

  useEffect(() => {
    if (!editable || current === saved) return
    const id = window.setTimeout(() => {
      setSaving('saving')
      const body = JSON.parse(current) as Draft
      update.mutate(
        { params: { path: { workspace: slug, campaignId: data.id } }, body: { ...body, version: version.current } },
        {
          onSuccess: (next) => {
            version.current = next.version
            setSaved(current)
            qc.setQueryData(key, next)
            void review.refetch()
            setSaving('idle')
          },
          onError: (e) => {
            if (e instanceof ApiError && e.status === 409) {
              toast({ kind: 'info', title: t(m.conflict) })
              void qc.invalidateQueries({ queryKey: key })
              return
            }
            setSaving('error')
            toast({ kind: 'error', title: t(m.saveFailed), detail: errorMessage(e) })
          },
        },
      )
    }, 800)
    return () => window.clearTimeout(id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current, saved, editable])

  const tpl = data.templates.find((x) => x.id === draft.templateId)
  const sub =
    data.status === 'draft'
      ? saving === 'saving'
        ? t(m.saving)
        : data.autosavedAt
          ? t(m.subDraftSaved, { name: data.createdBy, time: clock(data.autosavedAt) })
          : t(m.subDraft, { name: data.createdBy })
      : t(m.subOther, { status: t(statusWord[data.status] ?? m.statusSent) })
  const sent = data.status === 'sent' || data.status === 'sending' || data.status === 'paused' || data.status === 'cancelled'
  return (
    <>
      {!data.connected ? (
        <div className="note w" role="note">
          <span>{data.accountStatus === 'token_rejected' ? t(m.tokenRejected) : t(m.notConnected)}</span>
          <Link to="/w/$workspace/providers/whatsapp" params={{ workspace: slug }} className="px-lnk">
            {t(m.openWhatsApp)}
          </Link>
        </div>
      ) : null}
      <section className="card">
        <div className="px-head">
          <div>
            <span className="flex items-center gap-2">
              <span className="text-accent">
                <WhatsAppIcon />
              </span>
              <label className="vh" htmlFor="wa-name">
                {t(m.nameLabel)}
              </label>
              <input
                id="wa-name"
                className="min-w-0 flex-1 border-0 bg-transparent p-0 text-[15px] font-bold text-ink outline-offset-2"
                value={draft.name}
                readOnly={!editable}
                maxLength={200}
                onChange={(e) => set('name', e.target.value)}
              />
            </span>
            <span className="k">{saving === 'error' ? t(m.saveFailed) : sub}</span>
          </div>
          <div>
            {sent ? (
              <ButtonLink variant="primary" to="/w/$workspace/campaigns/$campaignId/whatsapp-report" params={{ workspace: slug, campaignId: data.id }}>
                {t(m.report)}
              </ButtonLink>
            ) : (
              <>
                <Button size="sm" icon="send" onClick={() => setTesting(true)} disabled={!data.connected || !tpl || current !== saved}>
                  {t(m.sendTest)}
                </Button>
                <ButtonLink size="sm" variant="primary" to="/w/$workspace/campaigns/$campaignId/whatsapp-review" params={{ workspace: slug, campaignId: data.id }}>
                  {data.access.canSend ? t(m.review) : t(m.reviewApproval)}
                </ButtonLink>
              </>
            )}
          </div>
        </div>
        <div className="wa-cols">
          <Settings data={data} draft={draft} set={set} patch={(p) => setDraft((d) => ({ ...d, ...p }))} editable={editable} />
          <Preview data={data} tpl={tpl} />
          <div className="wa-col">
            <span className="t">{t(m.beforeSending)}</span>
            <div className="wa-checks">
              {review.data ? (
                review.data.checks
                  .filter((c) => !((c.key === 'approval' || c.key === 'account') && c.status === 'pass'))
                  .map((c) => <CheckRow key={c.key} c={c} business={data.business} split={review.data.checks.find((x) => x.key === 'limit')?.split} />)
              ) : (
                <div className="flex flex-col gap-3 p-4">
                  <Skeleton w="80%" h={12} />
                  <Skeleton w="70%" h={12} />
                  <Skeleton w="60%" h={12} />
                </div>
              )}
            </div>
            <div className="wa-cost">
              <div className="flex items-baseline justify-between gap-2">
                <b>{t(m.estimatedCost)}</b>
                <span className="v">{money(data.cost.amount)}</span>
              </div>
              <span className="k">
                {t(m.costNote, { n: data.cost.messages, category: t(catWord[(data.cost.category || 'utility') as Template['category']] ?? m.catUtility), date: dayMonthYear(`${data.cost.ratesAsOf.slice(0, 10)}T12:00:00Z`) })}
                {' · '}
                <button type="button" className="px-lnk text-[12px] underline" onClick={() => setRates(true)}>
                  {t(m.editRates)}
                </button>
              </span>
            </div>
          </div>
        </div>
      </section>
      {testing ? <TestDialog data={data} onClose={() => setTesting(false)} /> : null}
      {rates ? <RatesDialog onClose={() => setRates(false)} /> : null}
    </>
  )
}

function Settings({
  data,
  draft,
  set,
  patch,
  editable,
}: {
  data: Campaign
  draft: Draft
  set: <K extends keyof Draft>(k: K, v: Draft[K]) => void
  patch: (p: Partial<Draft>) => void
  editable: boolean
}) {
  const t = useT()
  const approved = data.templates.filter((x) => x.status === 'approved')
  const others = data.templates.filter((x) => x.status !== 'approved')
  const tpl = data.templates.find((x) => x.id === draft.templateId)
  const othersText = others
    .map((x) =>
      x.status === 'pending'
        ? t(m.othersPending, { name: x.name })
        : x.status === 'rejected'
          ? t(m.othersRejected, { name: x.name })
          : t(m.othersOther, { name: x.name, status: t(statusBadge[x.status].msg).toLowerCase() }),
    )
    .join('; ')
  const varOf = (key: string) => draft.variables.find((v) => v.key === key) ?? { key, source: 'field' as const, field: '', fallback: '' }
  const setVar = (key: string, patch: Partial<Schemas['WhatsAppVariable']>) => {
    const cur = varOf(key)
    const rest = draft.variables.filter((v) => v.key !== key)
    set('variables', [...rest, { ...cur, ...patch }].sort((a, b) => Number(a.key) - Number(b.key)))
  }
  const listOptions = data.listOptions
  const audience = data.audience
  const zone = (Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC').replace(/_/g, ' ')
  return (
    <div className="wa-col">
      <label className="t" htmlFor="wa-template">
        {t(m.template)}
      </label>
      <select
        id="wa-template"
        className="in w-full"
        value={draft.templateId}
        disabled={!editable}
        onChange={(e) => {
          const next = data.templates.find((x) => x.id === e.target.value)
          set('templateId', e.target.value)
          if (next) patch({ templateId: e.target.value, variables: next.variables.map((v) => varOf(v.key)) })
        }}
      >
        {draft.templateId ? null : <option value="">{t(m.templatePick)}</option>}
        {approved.map((x) => (
          <option key={x.id} value={x.id}>
            {t(m.templateOption, { name: x.name, category: t(categoryBadge[x.category].msg), status: t(statusBadge[x.status].msg) })}
          </option>
        ))}
        {others.map((x) => (
          <option key={x.id} value={x.id} disabled>
            {t(m.templateOption, { name: x.name, category: t(categoryBadge[x.category].msg), status: t(statusBadge[x.status].msg) })}
          </option>
        ))}
      </select>
      <span className="k">{others.length ? t(m.templateNoteOthers, { others: othersText }) : t(m.templateNote)}</span>

      <span className="t mt-1.5">{t(m.variables)}</span>
      {tpl && tpl.variables.length ? (
        tpl.variables.map((tv) => {
          const v = varOf(tv.key)
          return (
            <div key={tv.key} className="wa-var">
              <span />
              <label className="text-[12px] font-semibold text-ink-2" htmlFor={`wa-var-${tv.key}`}>
                {t(m.varFrom, { label: tv.label.charAt(0).toUpperCase() + tv.label.slice(1) })}
              </label>
              <label className="text-[12px] font-semibold text-ink-2" htmlFor={`wa-var-${tv.key}-v`}>
                {v.source === 'text' ? t(m.text) : t(m.fallback)}
              </label>
              <span className="tok-cell">
                <span className="wa-tok">{`{{${tv.key}}}`}</span>
              </span>
              <select
                id={`wa-var-${tv.key}`}
                className="in w-full"
                disabled={!editable}
                value={v.source === 'text' ? '__text' : (v.field ?? '')}
                onChange={(e) => (e.target.value === '__text' ? setVar(tv.key, { source: 'text', field: undefined }) : setVar(tv.key, { source: 'field', field: e.target.value }))}
              >
                {v.source === 'field' && !v.field ? <option value="">–</option> : null}
                {data.fields.map((f) => (
                  <option key={f.key} value={f.key}>
                    {t(m.fieldOption, { label: f.key })}
                  </option>
                ))}
                <option value="__text">{t(m.fixedText)}</option>
              </select>
              <input
                id={`wa-var-${tv.key}-v`}
                className="in w-full"
                disabled={!editable}
                value={(v.source === 'text' ? v.text : v.fallback) ?? ''}
                maxLength={200}
                onChange={(e) => setVar(tv.key, v.source === 'text' ? { text: e.target.value } : { fallback: e.target.value })}
              />
            </div>
          )
        })
      ) : (
        <span className="k">{t(m.noVariables)}</span>
      )}

      <span className="t mt-1.5">{t(m.audience)}</span>
      <div className="flex flex-col gap-1.5" role="group" aria-label={t(m.pickLists)}>
        {listOptions
          .filter((l) => l.consent > 0 || draft.lists.includes(l.listId))
          .map((l) => (
          <label key={l.listId} className="wa-box cursor-pointer">
            <span className="flex min-w-0 items-center gap-2">
              <input
                type="checkbox"
                checked={draft.lists.includes(l.listId)}
                disabled={!editable}
                onChange={(e) => set('lists', e.target.checked ? [...draft.lists, l.listId] : draft.lists.filter((x) => x !== l.listId))}
              />
              <span className="truncate">{l.name}</span>
            </span>
            <span className="k flex-none font-medium">{t(m.audienceConsent, { count: l.consent })}</span>
          </label>
        ))}
      </div>
      <span className="k">
        {!draft.lists.length
          ? t(m.audienceNone)
          : audience.without
            ? t(m.audienceWithout, { n: audience.without })
            : t(m.audienceAll)}
        {audience.suppressed ? ` ${t(m.audienceSuppressed, { n: audience.suppressed })}` : null}
      </span>

      <span className="t mt-1.5">{t(m.schedule)}</span>
      <select
        className="in w-full"
        aria-label={t(m.schedule)}
        disabled={!editable}
        value={draft.mode}
        onChange={(e) => {
          const mode = e.target.value as Draft['mode']
          const tomorrow = new Date(Date.now() + 86_400_000)
          if (mode === 'local') patch({ mode, localDate: draft.localDate ?? tomorrow.toISOString().slice(0, 10), localMinute: draft.localMinute ?? 540, sendAt: undefined })
          else if (mode === 'at') patch({ mode, sendAt: draft.sendAt ?? new Date(Date.now() + 3_600_000).toISOString(), localDate: undefined, localMinute: undefined })
          else patch({ mode, sendAt: undefined, localDate: undefined, localMinute: undefined })
        }}
      >
        <option value="now">{t(m.schedNow)}</option>
        <option value="at">{t(m.schedAt, { zone })}</option>
        <option value="local">{t(m.schedLocal)}</option>
      </select>
      {draft.mode === 'local' ? (
        <div className="grid grid-cols-2 gap-2">
          <div className="lbl">
            <label htmlFor="wa-local-date">{t(m.schedDate)}</label>
            <input id="wa-local-date" className="in w-full" type="date" disabled={!editable} value={draft.localDate ?? ''} onChange={(e) => set('localDate', e.target.value)} />
          </div>
          <div className="lbl">
            <label htmlFor="wa-local-time">{t(m.schedTime)}</label>
            <input
              id="wa-local-time"
              className="in w-full"
              type="time"
              disabled={!editable}
              value={hhmm(draft.localMinute ?? 540)}
              onChange={(e) => {
                const [h = 0, mi = 0] = e.target.value.split(':').map(Number)
                set('localMinute', h * 60 + mi)
              }}
            />
          </div>
        </div>
      ) : draft.mode === 'at' ? (
        <div className="lbl">
          <label htmlFor="wa-send-at">{t(m.schedWhen)}</label>
          <input
            id="wa-send-at"
            className="in w-full"
            type="datetime-local"
            disabled={!editable}
            value={draft.sendAt ? new Date(new Date(draft.sendAt).getTime() - new Date().getTimezoneOffset() * 60000).toISOString().slice(0, 16) : ''}
            onChange={(e) => e.target.value && set('sendAt', new Date(e.target.value).toISOString())}
          />
        </div>
      ) : null}
      <div className="wa-box font-semibold">
        <span className="flex items-center gap-2">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
            <rect x="2" y="3" width="12" height="11" rx="1.5" />
            <path d="M2 6.5h12M5.5 1.5v3M10.5 1.5v3" />
          </svg>
          {draft.mode === 'local' && draft.localDate
            ? t(m.schedLabelLocal, { date: dayMonthDate(`${draft.localDate}T12:00:00Z`), time: hhmm(draft.localMinute ?? 540) })
            : draft.mode === 'at' && draft.sendAt
              ? t(m.schedLabelAt, { date: dayMonthDate(draft.sendAt), time: clock(draft.sendAt), zone })
              : t(m.schedLabelNow)}
        </span>
      </div>
      <span className="k">{data.quiet ? t(m.quietNote, { window: data.window }) : t(m.quietOff)}</span>
    </div>
  )
}

function Preview({ data, tpl }: { data: Campaign; tpl?: Template }) {
  const t = useT()
  const p = data.preview
  const b = data.business
  return (
    <div className="wa-col">
      <span className="t">{p ? t(m.previewAs, { name: p.contact }) : t(m.previewNoContact)}</span>
      <div className="wa-chat">
        <div className="hd">
          <span className="wa-mark" aria-hidden="true">
            {(b?.displayName ?? 'A').charAt(0).toUpperCase()}
          </span>
          <span className="flex min-w-0 flex-col">
            <b className="text-[14px]">{b?.displayName ?? '–'}</b>
            <span className="k">{b ? t(m.businessLine, { number: b.number }) : ''}</span>
          </span>
        </div>
        <div className="bd">
          <span className="day">{t(m.today)}</span>
          {p && tpl ? (
            <>
              <div className="bub">
                <FilledBody body={tpl.body} text={p.text} />
                <span className="tm">{data.schedule.localMinute !== undefined ? hhmm(data.schedule.localMinute) : clock(new Date())}</span>
              </div>
              {p.buttons.map((btn, i) => (
                <span key={i} className="qr">
                  {btn.type === 'url' ? (
                    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                      <path d="M6.5 9.5a3 3 0 0 0 4.2 0l2-2a3 3 0 0 0-4.2-4.2l-.6.6M9.5 6.5a3 3 0 0 0-4.2 0l-2 2a3 3 0 0 0 4.2 4.2l.6-.6" />
                    </svg>
                  ) : null}
                  {btn.text}
                </span>
              ))}
            </>
          ) : (
            <span className="k">{t(m.previewEmpty)}</span>
          )}
        </div>
      </div>
      {p ? (
        <span className="k">
          {t(m.previewFilled, { name: p.firstName || p.contact })}{' '}
          {p.fallbacks.map((f) => t(m.previewFallback, { field: f.field.replace(/_/g, ' '), fallback: fallbackSentence(tpl?.body ?? '', f.key, f.fallback) })).join(' ')}
        </span>
      ) : null}
    </div>
  )
}

function TestDialog({ data, onClose }: { data: Campaign; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [phone, setPhone] = useState('')
  const send = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/test')
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.testTitle)}
      description={t(m.testText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            icon="send"
            loading={send.isPending}
            busyLabel={t(m.testSending)}
            disabled={!phone.trim()}
            onClick={() =>
              send.mutate(
                { params: { path: { workspace: slug, campaignId: data.id } }, body: { phone: phone.trim() } },
                {
                  onSuccess: (r) => {
                    toast({ kind: 'success', title: t(m.testQueued, { to: r.to }) })
                    onClose()
                  },
                },
              )
            }
          >
            {t(m.testSend)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="lbl">
          <label htmlFor="wa-c-test-phone">{t(m.testPhone)}</label>
          <input id="wa-c-test-phone" className="in w-full" type="tel" value={phone} placeholder="+1 312 555 0123" onChange={(e) => setPhone(e.target.value)} />
        </div>
        {data.test ? <span className="k">{t(m.lastTest, { time: clock(data.test.at), to: data.test.to, status: t(testStatusWord[data.test.status]) })}</span> : null}
        {send.isError ? (
          <span className="err" role="alert">
            {errorMessage(send.error)}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

/** Meta's rate card, per category and country, as the workspace keeps it. */
export function RatesDialog({ onClose }: { onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const canManage = useCan('providers:manage')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp/rates', { params: { path: { workspace: slug } } })
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/whatsapp/rates')
  const [rows, setRows] = useState<Schemas['WhatsAppRate'][] | null>(null)
  const [asOf, setAsOf] = useState<string | null>(null)
  const data = q.data
  const list = rows ?? data?.rates ?? []
  const date = asOf ?? data?.asOf.slice(0, 10) ?? ''
  return (
    <Dialog
      open
      size="lg"
      onOpenChange={(o) => !o && onClose()}
      title={t(m.ratesTitle)}
      description={t(m.ratesText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          {canManage && data?.canManage ? (
            <Button
              variant="primary"
              loading={save.isPending}
              onClick={() =>
                save.mutate(
                  { params: { path: { workspace: slug } }, body: { asOf: date, rates: list } },
                  {
                    onSuccess: () => {
                      toast({ kind: 'success', title: t(m.ratesSaved) })
                      void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && x.queryKey[1].includes('whatsapp') })
                      onClose()
                    },
                  },
                )
              }
            >
              {t(m.ratesSave)}
            </Button>
          ) : null}
        </>
      }
    >
      {data ? (
        <div className="flex flex-col gap-3">
          <div className="lbl max-w-[220px]">
            <label htmlFor="wa-rates-asof">{t(m.ratesAsOf)}</label>
            <input id="wa-rates-asof" className="in" type="date" value={date} disabled={!data.canManage} onChange={(e) => setAsOf(e.target.value)} />
          </div>
          <div className="tscroll max-h-[360px] overflow-y-auto">
            <table className="tbl compact">
              <thead>
                <tr>
                  <th scope="col">{t(m.colCountry)}</th>
                  <th scope="col">{t(m.colCategory)}</th>
                  <th scope="col" className="r">
                    {t(m.colPrice)}
                  </th>
                </tr>
              </thead>
              <tbody>
                {list.map((r, i) => (
                  <tr key={`${r.country}-${r.category}`}>
                    <td className="mono">{r.country}</td>
                    <td>{t(catWord[r.category])}</td>
                    <td className="r">
                      <input
                        className="in w-28 text-right"
                        inputMode="decimal"
                        aria-label={`${r.country} ${t(catWord[r.category])}`}
                        disabled={!data.canManage}
                        value={String(r.price)}
                        onChange={(e) => {
                          const v = Number(e.target.value.replace(',', '.'))
                          if (Number.isNaN(v)) return
                          setRows(list.map((x, j) => (j === i ? { ...x, price: v } : x)))
                        }}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {data.editedAt && data.editedBy ? <span className="k">{t(m.ratesEdited, { date: dayMonthYear(data.editedAt), name: data.editedBy })}</span> : null}
          {!data.canManage ? <span className="k">{t(m.ratesReadOnly)}</span> : null}
          {save.isError ? (
            <span className="err" role="alert">
              {errorMessage(save.error)}
            </span>
          ) : null}
        </div>
      ) : (
        <Skeleton w="100%" h={200} />
      )}
    </Dialog>
  )
}

