import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { ForbiddenState } from '@/components/shell/SystemScreens'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { TableScroll, Th } from '@/components/ui/Table'
import { T, useT } from '@/i18n'
import { andList, count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { mt } from './messages'
import { mg } from './settings.messages'
import { TxTabs } from './shared'

type Settings = Schemas['TransactionalSettings']
type Input = Schemas['TransactionalSettingsInput']
type Category = Schemas['TransactionalCategory']

const API = '/api/v1/workspaces/{workspace}/transactional' as const

function toInput(s: Settings): Input {
  return {
    linkContacts: s.linkContacts,
    contentRetentionDays: s.contentRetentionDays,
    alertBounceRate: s.alertBounceRate,
    alertFailed: s.alertFailed,
    alertMedianSeconds: s.alertMedianSeconds,
    defaultSenderId: s.defaultSender?.id ?? '',
    defaultReplyTo: s.defaultReplyTo,
    defaultTrackOpens: s.defaultTrackOpens,
    defaultTrackClicks: s.defaultTrackClicks,
  }
}

/** Transactional › Settings (TransactionalSettings): categories, retention, delivery alerts and template defaults. */
export function TransactionalSettingsPage() {
  const t = useT()
  const { slug, me } = useWorkspace()
  const canView = me.permissions.includes('transactional:view')
  const path = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', `${API}/settings`, path, { enabled: canView })
  const cats = $api.useQuery('get', `${API}/categories`, path, { enabled: canView })
  return (
    <Page title={t(mt.title)} filters={<TxTabs />}>
      {!canView ? (
        <ForbiddenState permission="transactional:view" title={t(mt.forbiddenTitle)} />
      ) : q.error || cats.error ? (
        <ErrorPanel
          title={errorMessage(q.error ?? cats.error)}
          actions={
            <Button
              onClick={() => {
                void q.refetch()
                void cats.refetch()
              }}
            >
              {t(mt.retry)}
            </Button>
          }
        />
      ) : !q.data || !cats.data ? (
        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(460px, 100%), 1fr))' }} aria-busy="true">
          {Array.from({ length: 4 }, (_, i) => (
            <div key={i} className="card pad flex flex-col gap-3">
              <Skeleton w="40%" />
              <Skeleton w="100%" h={100} />
            </div>
          ))}
        </div>
      ) : (
        <SettingsForm key={q.data.updatedAt ?? 'new'} s={q.data} cats={cats.data} />
      )}
    </Page>
  )
}

function SettingsForm({ s, cats }: { s: Settings; cats: Schemas['TransactionalCategoryList'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const saved = useMemo(() => toInput(s), [s])
  // The form remounts when the saved settings change (key above), so this starts from them.
  const [v, setV] = useState<Input>(saved)
  const save = $api.useMutation('put', `${API}/settings`)
  const [editing, setEditing] = useState<Category | 'new'>()
  const changes = changeLines(t, saved, v, s)
  const set = <K extends keyof Input>(k: K, x: Input[K]) => setV((p) => ({ ...p, [k]: x }))
  const ro = !s.canEdit
  const optional = cats.items.filter((c) => c.kind === 'optional')
  const essential = cats.items.filter((c) => c.kind === 'essential')
  return (
    <>
      {ro ? (
        <Note tone="neutral">
          <T msg={mg.readOnly} values={{ permission: <span className="mono">{'transactional:publish'}</span> }} />
        </Note>
      ) : null}
      <section className="card">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h2 className="h2">{t(mg.categoriesTitle, { count: cats.items.length })}</h2>
            <span className="k">{t(mg.categoriesCaption)}</span>
          </div>
        </div>
        <TableScroll label={t(mg.categoriesLabel)}>
          <table className="tbl" aria-label={t(mg.categoriesLabel)}>
            <thead>
              <tr>
                <Th>{t(mg.colCategory)}</Th>
                <Th>{t(mg.colKind)}</Th>
                <Th>{t(mg.colSees)}</Th>
                <Th align="right">{t(mg.colTemplates)}</Th>
                <Th align="right">{t(mg.colOptedOut)}</Th>
                <Th>
                  <span className="vh">{t(mg.edit)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              {cats.items.map((c) => (
                <tr key={c.id}>
                  <td>
                    <span className="font-semibold">{c.name}</span>
                  </td>
                  <td>
                    <Badge tone={c.kind === 'optional' ? 'info' : 'neutral'}>{c.kind === 'optional' ? t(mg.optional) : t(mg.essential)}</Badge>
                  </td>
                  <td>{c.description || <span className="k">—</span>}</td>
                  <td className="r tx-num">{count(c.templates)}</td>
                  <td className={`r ${c.kind === 'optional' && c.optedOut ? 'tx-num' : 'k'}`}>{c.kind === 'optional' && c.optedOut ? count(c.optedOut) : '—'}</td>
                  <td className="r">
                    {cats.canEdit ? (
                      <Button size="sm" onClick={() => setEditing(c)}>
                        {t(mg.edit)}
                      </Button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </TableScroll>
        <div className="tx-foot">
          <span className="k">{t(mg.kindFixed)}</span>
          {cats.canEdit ? (
            <Button size="sm" icon="plus" onClick={() => setEditing('new')}>
              {t(mg.addCategory)}
            </Button>
          ) : null}
        </div>
      </section>
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(460px, 100%), 1fr))' }}>
        <div className="flex min-w-0 flex-col gap-4">
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mg.retentionTitle)}</h2>
            </div>
            <SettingRow title={t(mg.linkContacts)} body={t(mg.linkContactsBody)}>
              <input type="checkbox" role="switch" className="tg" aria-label={t(mg.linkContacts)} checked={v.linkContacts} disabled={ro} onChange={(e) => set('linkContacts', e.target.checked)} />
            </SettingRow>
            <SettingRow title={t(mg.keepContent)} body={t(mg.keepContentBody)}>
              <select className="in" style={{ width: 180 }} aria-label={t(mg.keepContent)} value={v.contentRetentionDays} disabled={ro} onChange={(e) => set('contentRetentionDays', Number(e.target.value))}>
                <option value={0}>{t(mg.keepNone)}</option>
                {[7, 30, 60, 90].map((d) => (
                  <option key={d} value={d}>
                    {t(mg.keepDays, { days: d })}
                  </option>
                ))}
              </select>
            </SettingRow>
          </section>
          <section className="card" id="alerts">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2">{t(mg.alertsTitle)}</h2>
                <span className="k">{t(mg.alertsCaption)}</span>
              </div>
            </div>
            <div className="pad flex flex-col gap-3.5">
              <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(180px, 100%), 1fr))' }}>
                <Limit label={t(mg.bounceRate)} unit={t(mg.bounceRateUnit)} aria={t(mg.bounceRateLabel)} value={v.alertBounceRate} step={0.1} disabled={ro} onChange={(x) => set('alertBounceRate', x)} />
                <Limit label={t(mg.failed)} unit={t(mg.failedUnit)} aria={t(mg.failedLabel)} value={v.alertFailed} step={1} disabled={ro} onChange={(x) => set('alertFailed', Math.round(x))} />
                <Limit label={t(mg.median)} unit={t(mg.medianUnit)} aria={t(mg.medianLabel)} value={v.alertMedianSeconds} step={1} disabled={ro} onChange={(x) => set('alertMedianSeconds', Math.round(x))} />
              </div>
              <Note tone="neutral" role="note">
                <T
                  msg={mg.noAutoPause}
                  values={{
                    permission: <span className="mono">{'transactional:view'}</span>,
                    link: (chunks) => (
                      <Link to="/w/$workspace/account" params={{ workspace: slug }} hash="notifications">
                        {chunks}
                      </Link>
                    ),
                  }}
                />
              </Note>
            </div>
          </section>
        </div>
        <div className="flex min-w-0 flex-col gap-4">
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mg.defaultsTitle)}</h2>
            </div>
            <div className="pad grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(240px, 100%), 1fr))' }}>
              <label className="lbl">
                {t(mg.sender)}
                <select className="in" aria-label={t(mg.defaultSender)} value={v.defaultSenderId ?? ''} disabled={ro} onChange={(e) => set('defaultSenderId', e.target.value)}>
                  {s.senders.length === 0 ? <option value="">{t(mg.noSender)}</option> : null}
                  {s.senders
                    .filter((sn) => sn.verified)
                    .map((sn) => (
                      <option key={sn.id} value={sn.id}>
                        {`${sn.name} <${sn.email}>`}
                      </option>
                    ))}
                </select>
              </label>
              <label className="lbl">
                {t(mg.replyTo)}
                <input className="in" type="email" aria-label={t(mg.defaultReplyTo)} value={v.defaultReplyTo ?? ''} disabled={ro} onChange={(e) => set('defaultReplyTo', e.target.value)} />
              </label>
            </div>
            <SettingRow title={t(mg.trackOpens)} body={t(mg.trackOpensBody)}>
              <input type="checkbox" role="switch" className="tg" aria-label={t(mg.trackOpens)} checked={v.defaultTrackOpens} disabled={ro} onChange={(e) => set('defaultTrackOpens', e.target.checked)} />
            </SettingRow>
            <SettingRow title={t(mg.trackClicks)} body={t(mg.trackClicksBody)}>
              <input type="checkbox" role="switch" className="tg" aria-label={t(mg.trackClicks)} checked={v.defaultTrackClicks} disabled={ro} onChange={(e) => set('defaultTrackClicks', e.target.checked)} />
            </SettingRow>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mg.previewTitle)}</h2>
            </div>
            <div className="pad flex flex-col gap-2.5">
              <span className="k" style={{ color: 'var(--ink-nav)' }}>
                {t(mg.previewCaption)}
              </span>
              <div className="flex flex-col gap-3 rounded-[10px] border border-line px-4 py-3.5" style={{ background: '#FBFBFC' }} aria-hidden="true">
                <span className="font-bold">{t(mg.previewHeading, { product: s.productName })}</span>
                {optional.length === 0 ? <span className="k">{t(mg.previewNoOptional)}</span> : null}
                {optional.map((c) => (
                  <div key={c.id} className="row" style={{ justifyContent: 'space-between', flexWrap: 'nowrap' }}>
                    <span className="flex flex-col">
                      <span className="font-semibold">{c.name}</span>
                      <span className="k">{c.description}</span>
                    </span>
                    <input type="checkbox" className="tg" checked readOnly tabIndex={-1} />
                  </div>
                ))}
                {essential.length ? <span className="k">{t(mg.previewEssential, { names: andList(essential.map((c) => c.name.toLowerCase())).replace(/^./, (x) => x.toUpperCase()) })}</span> : null}
              </div>
              <a className="tx-lnk" href={`/p/${encodeURIComponent(slug)}/preferences`} target="_blank" rel="noopener noreferrer">
                {t(mg.openPreferences)}
              </a>
            </div>
          </section>
        </div>
      </div>
      {changes.length && !ro ? (
        <div className="card sticky bottom-3 z-10 flex flex-wrap items-center gap-3" style={{ padding: '12px 18px', boxShadow: '0 8px 24px rgba(11,18,32,.12)' }} role="status">
          <Badge tone="warn" dot>
            {t(mg.unsaved)}
          </Badge>
          <span className="k">{changes.length === 1 ? changes[0] : t(mg.changeMany, { count: changes.length })}</span>
          <span className="ml-auto flex gap-2">
            <Button onClick={() => setV(saved)}>{t(mg.discard)}</Button>
            <Button
              variant="primary"
              loading={save.isPending}
              onClick={() =>
                save.mutate(
                  { params: { path: { workspace: slug } }, body: { ...v, defaultSenderId: v.defaultSenderId || undefined } },
                  {
                    onSuccess: () => {
                      toast({ kind: 'success', title: t(mg.saved) })
                      void qc.invalidateQueries({ queryKey: ['get', `${API}/settings`] })
                    },
                    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
                  },
                )
              }
            >
              {t(mg.save)}
            </Button>
          </span>
        </div>
      ) : null}
      {editing ? <CategoryDialog c={editing === 'new' ? undefined : editing} onClose={() => setEditing(undefined)} /> : null}
    </>
  )
}

function changeLines(t: ReturnType<typeof useT>, a: Input, b: Input, s: Settings): string[] {
  const out: string[] = []
  const onOff = (x: boolean) => (x ? 'on' : 'off')
  if (a.alertBounceRate !== b.alertBounceRate) out.push(t(mg.changeBounce, { from: a.alertBounceRate, to: b.alertBounceRate }))
  if (a.alertFailed !== b.alertFailed) out.push(t(mg.changeFailed, { from: a.alertFailed, to: b.alertFailed }))
  if (a.alertMedianSeconds !== b.alertMedianSeconds) out.push(t(mg.changeMedian, { from: a.alertMedianSeconds, to: b.alertMedianSeconds }))
  if (a.contentRetentionDays !== b.contentRetentionDays) out.push(t(mg.changeRetention, { from: a.contentRetentionDays, to: b.contentRetentionDays }))
  if (a.linkContacts !== b.linkContacts) out.push(t(mg.changeLink, { to: onOff(b.linkContacts) }))
  if ((a.defaultSenderId ?? '') !== (b.defaultSenderId ?? '')) out.push(t(mg.changeSender))
  if ((a.defaultReplyTo ?? '') !== (b.defaultReplyTo ?? '')) out.push(t(mg.changeReplyTo))
  if (a.defaultTrackOpens !== b.defaultTrackOpens) out.push(t(mg.changeOpens, { to: onOff(b.defaultTrackOpens) }))
  if (a.defaultTrackClicks !== b.defaultTrackClicks) out.push(t(mg.changeClicks, { to: onOff(b.defaultTrackClicks) }))
  void s
  return out
}

function SettingRow({ title, body, children }: { title: string; body: string; children: ReactNode }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-4 border-t border-line-2" style={{ padding: '14px 18px' }}>
      <div className="flex min-w-0 flex-col gap-[3px]" style={{ flex: '1 1 380px' }}>
        <span className="font-semibold">{title}</span>
        <span className="k" style={{ color: '#4B5563' }}>
          {body}
        </span>
      </div>
      <div className="flex flex-none items-center gap-2">{children}</div>
    </div>
  )
}

function Limit({ label, unit, aria, value, step, disabled, onChange }: { label: string; unit: string; aria: string; value: number; step: number; disabled?: boolean; onChange: (v: number) => void }) {
  return (
    <label className="lbl">
      {label}
      <div className="row" style={{ flexWrap: 'nowrap', gap: 8 }}>
        <input
          className="in"
          type="number"
          min={step < 1 ? 0.1 : 1}
          step={step}
          value={value}
          disabled={disabled}
          style={{ width: 72 }}
          aria-label={aria}
          onChange={(e) => {
            const n = Number(e.target.value)
            if (!Number.isNaN(n) && n > 0) onChange(n)
          }}
        />
        <span className="k">{unit}</span>
      </div>
    </label>
  )
}

function CategoryDialog({ c, onClose }: { c?: Category; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [name, setName] = useState(c?.name ?? '')
  const [kind, setKind] = useState<'essential' | 'optional'>(c?.kind ?? 'optional')
  const [description, setDescription] = useState(c?.description ?? '')
  const [error, setError] = useState<string>()
  const create = $api.useMutation('post', `${API}/categories`)
  const update = $api.useMutation('put', `${API}/categories/{categoryId}`)
  const del = $api.useMutation('delete', `${API}/categories/{categoryId}`)
  const locked = Boolean(c && c.templates > 0)
  const done = (msg: string) => {
    toast({ kind: 'success', title: msg })
    void qc.invalidateQueries()
    onClose()
  }
  const fail = (e: unknown) => setError(errorMessage(e))
  const submit = () => {
    if (!name.trim()) return setError(t(mg.nameRequired))
    const body = { name: name.trim(), kind, description: description.trim() }
    if (c) update.mutate({ params: { path: { workspace: slug, categoryId: c.id } }, body }, { onSuccess: () => done(t(mg.categorySaved)), onError: fail })
    else create.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: () => done(t(mg.categorySaved)), onError: fail })
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={c ? t(mg.editTitle, { name: c.name }) : t(mg.addTitle)}
      footer={
        <>
          {c ? (
            <Button
              variant="danger"
              className="mr-auto"
              disabledReason={c.templates > 0 ? t(mg.deleteBlocked) : undefined}
              loading={del.isPending}
              onClick={() => del.mutate({ params: { path: { workspace: slug, categoryId: c.id } } }, { onSuccess: () => done(t(mg.categoryDeleted)), onError: fail })}
            >
              {t(mg.deleteCategory)}
            </Button>
          ) : null}
          <Button onClick={onClose}>{t(mg.cancel)}</Button>
          <Button variant="primary" loading={create.isPending || update.isPending} onClick={submit}>
            {c ? t(mg.saveCategory) : t(mg.createCategory)}
          </Button>
        </>
      }
    >
      {error ? <Note tone="bad">{error}</Note> : null}
      <Field label={t(mg.name)}>
        <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={60} />
      </Field>
      <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
        <legend className="mb-1 text-[12px] font-semibold text-ink-2">{t(mg.kind)}</legend>
        <OptionCard name="kind" value="essential" checked={kind === 'essential'} onChange={() => setKind('essential')} title={t(mg.essential)} description={t(mg.essentialBody)} disabled={locked} />
        <OptionCard name="kind" value="optional" checked={kind === 'optional'} onChange={() => setKind('optional')} title={t(mg.optional)} description={t(mg.optionalBody)} disabled={locked} />
        {locked && c ? <span className="k">{t(mg.kindLocked, { count: c.templates })}</span> : null}
      </fieldset>
      <Field label={t(mg.description)} hint={t(mg.descriptionHint)}>
        <Input value={description} onChange={(e) => setDescription(e.target.value)} maxLength={160} />
      </Field>
    </Dialog>
  )
}
