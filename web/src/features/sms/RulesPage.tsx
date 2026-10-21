import { Link } from '@tanstack/react-router'
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Page } from '@/components/shell/Page'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Table } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { formatList, T, useT } from '@/i18n'
import { ago, clock, count, dayMonthDate, dayMonthYear, monthName, money } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import {
  BudgetMeter,
  CountryChips,
  CountryName,
  countryName,
  price,
  providerColor,
  ReadOnlyNote,
  regionLabel,
  regionSentence,
  registrationBadge,
  SmsTabs,
  throwIfForbidden,
  useKindName,
  useRefreshSms,
} from './common'
import { m as sm } from './messages'
import { m } from './rules.messages'
import { countText } from './segments'
import { parseCountries } from './SenderDialogs'

type Rules = Schemas['SmsRules']
type Form = Schemas['SmsRulesInput']
type Route = Schemas['SmsRouteView']

const retention = [0, 7, 30, 60, 90] as const

function toForm(r: Rules): Form {
  return {
    alertPct: r.alertPct,
    brandPrefix: r.brandPrefix,
    budget: r.budget,
    confirmation: r.confirmation.text,
    confirmationOn: r.confirmationOn,
    helpKeywords: r.helpKeywords,
    helpReply: r.helpReply.text,
    keywordsBy: r.keywordsBy,
    optInKeywords: r.optInKeywords,
    optInReply: r.optInReply.text,
    optOutInSteps: r.optOutInSteps,
    optOutKeywords: r.optOutKeywords,
    optOutReply: r.optOutReply.text,
    optOutText: r.optOutText,
    pushWindow: r.pushWindow,
    quietFrom: r.quietFrom,
    quietTo: r.quietTo,
    requireYes: r.requireYes,
    retentionDays: (retention as readonly number[]).includes(r.retentionDays) ? (r.retentionDays as Form['retentionDays']) : 30,
    thresholds: r.thresholds,
    whatsAppWindow: r.whatsAppWindow,
  }
}

/** "09:00" as minutes since midnight. */
function minutes(hhmm: string): number {
  const [h, mm] = hhmm.split(':').map(Number)
  return (h ?? 0) * 60 + (mm ?? 0)
}

/** The window's problem, if any: before 08:00, after 21:00, or empty. */
function windowProblem(from: string, to: string): 'early' | 'late' | 'empty' | undefined {
  if (minutes(from) < 8 * 60) return 'early'
  if (minutes(to) > 21 * 60) return 'late'
  if (minutes(to) <= minutes(from)) return 'empty'
  return undefined
}

/** Sending providers › SMS › Routing & rules (SmsRouting, SmsStates). */
export function SmsRulesPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/rules', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  if (q.data) return <RulesBody key={q.data.updatedAt ?? 'new'} data={q.data} />
  return (
    <Page title={t(sm.pageTitle)} filters={<SmsTabs />}>
      {q.isError ? (
        <ErrorPanel title={t(m.errorTitle)} reference={problemReference(q.error)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      ) : (
        <div className="card flex flex-col gap-3 p-5" aria-busy="true" aria-label={t(m.loading)}>
          <Skeleton w={160} h={16} />
          <Skeleton h={120} />
          <Skeleton h={120} />
        </div>
      )}
    </Page>
  )
}

function RulesBody({ data }: { data: Rules }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const [form, setForm] = useState<Form>(() => toForm(data))
  const [budgetText, setBudgetText] = useState(String(data.budget))
  const saved = useMemo(() => toForm(data), [data])
  const dirty = JSON.stringify(form) !== JSON.stringify(saved)
  const ro = !data.canManage
  const problem = windowProblem(form.quietFrom, form.quietTo)
  const budget = Number(budgetText.replace(/[$,\s]/g, ''))
  const budgetBad = !Number.isFinite(budget) || budget < 0 || budgetText.trim() === ''
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/rules', {
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const set = <K extends keyof Form>(k: K, v: Form[K]) => setForm((f) => ({ ...f, [k]: v }))
  const submit = () => {
    if (problem || budgetBad) return
    const before = saved
    const body = { ...form, budget }
    save.mutate(
      { params: { path: { workspace: slug } }, body },
      {
        onSuccess: () => {
          void refresh()
          toast({
            kind: 'success',
            title: t(m.saved),
            detail: t(m.savedDetail),
            undo: () => {
              save.mutate({ params: { path: { workspace: slug } }, body: before }, { onSuccess: () => void refresh() })
            },
          })
        },
      },
    )
  }
  const actions = (
    <>
      {dirty ? (
        <Badge tone="warn">{t(m.unsaved)}</Badge>
      ) : data.updatedAt ? (
        <span className="k">{data.updatedBy ? t(m.savedBy, { ago: ago(data.updatedAt), name: data.updatedBy }) : t(m.savedAgo, { ago: ago(data.updatedAt) })}</span>
      ) : null}
      <Button variant="primary" disabled={ro || !dirty || Boolean(problem) || budgetBad} loading={save.isPending} onClick={submit}>
        {t(m.save)}
      </Button>
    </>
  )
  return (
    <Page title={t(sm.pageTitle)} filters={<SmsTabs />} actions={actions}>
      {ro ? <ReadOnlyNote>{t(m.readOnlyText, { helpers: data.helpers.length ? formatList(data.helpers, 'disjunction') : 'none' })}</ReadOnlyNote> : null}
      <div className="flex flex-wrap items-start gap-5">
        <div className="flex min-w-0 flex-col gap-5" style={{ flex: '999 1 620px' }}>
          <Routes data={data} ro={ro} />
          <QuietHours data={data} form={form} set={set} ro={ro} problem={problem} />
          <Keywords data={data} form={form} set={set} ro={ro} />
        </div>
        <div className="flex min-w-0 flex-col gap-5" style={{ flex: '1 1 340px', maxWidth: 440 }}>
          <RouteTester data={data} />
          <Budget data={data} form={form} set={set} ro={ro} budgetText={budgetText} setBudgetText={setBudgetText} budgetBad={budgetBad} />
          <TestNumbers data={data} ro={ro} />
          <Retention form={form} set={set} ro={ro} />
        </div>
      </div>
      <MessageRules data={data} form={form} set={set} ro={ro} />
    </Page>
  )
}

function Card({ title, lead, action, children }: { title: string; lead?: ReactNode; action?: ReactNode; children: ReactNode }) {
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{title}</h2>
          {lead ? <span className="k">{lead}</span> : null}
        </div>
        {action}
      </div>
      {children}
    </section>
  )
}

/* ---------------- Routes ---------------- */

function Routes({ data, ro }: { data: Rules; ro: boolean }) {
  const t = useT()
  const kindName = useKindName()
  const order = [...new Set(data.senders.map((s) => s.provider))]
  const [editing, setEditing] = useState<Route | 'new' | 'other' | null>(null)
  return (
    <Card
      title={t(m.routesTitle)}
      lead={t(m.routesLead)}
      action={
        <Button size="sm" disabled={ro} onClick={() => setEditing('new')}>
          {t(m.addRoute)}
        </Button>
      }
    >
      <Table caption={t(m.routesTable)}>
        <thead>
          <tr>
            <th scope="col">{t(m.colCountries)}</th>
            <th scope="col">{t(m.colSender)}</th>
            <th scope="col">{t(m.colFallback)}</th>
            <th scope="col" className="r">
              {t(m.colOnLists)}
            </th>
            <th scope="col">
              <span className="vh">{t(m.colActions)}</span>
            </th>
          </tr>
        </thead>
        <tbody>
          {data.routes.map((r) => {
            const b = registrationBadge(r.registration, t)
            return (
              <tr key={r.id} style={r.skipped ? { background: '#FFFBEB' } : undefined}>
                <td>
                  <CountryChips codes={r.countries} named />
                </td>
                <td>
                  <div className="flex flex-col gap-[3px]">
                    <span className="inline-flex items-center gap-[7px] font-semibold">
                      <span className="dot" style={{ color: providerColor(order, r.provider) }} aria-hidden="true" />
                      {r.provider} · <span className={r.sender.startsWith('+') ? 'mono font-semibold' : undefined}>{r.sender}</span>
                    </span>
                    {r.skipped ? (
                      <Badge tone={b.tone} className="self-start">
                        {b.text}
                      </Badge>
                    ) : (
                      <span className="k">{t(m.senderLine, { kind: kindName(r.senderKind), status: b.text.charAt(0).toLowerCase() + b.text.slice(1) })}</span>
                    )}
                  </div>
                </td>
                <td>{r.fallback ? t(m.fallbackVia, { provider: r.fallbackProvider ?? '', sender: r.fallback }) : r.skipped ? t(m.fallbackSkipped) : t(m.fallbackNone)}</td>
                <td className="r num">{count(r.onLists)}</td>
                <td className="r">
                  <Button size="sm" disabled={ro} onClick={() => setEditing(r)}>
                    {t(m.edit)}
                  </Button>
                </td>
              </tr>
            )
          })}
          <tr>
            <td>
              <span className="font-semibold">{t(m.otherCountries)}</span>
            </td>
            <td>{data.other === 'route' && data.otherRouteId ? data.routes.find((r) => r.id === data.otherRouteId)?.sender : t(m.dontSend)}</td>
            <td>
              <span className="k">{t(sm.none)}</span>
            </td>
            <td className="r num">{count(data.otherPeople)}</td>
            <td className="r">
              <Button size="sm" disabled={ro} onClick={() => setEditing('other')}>
                {t(m.edit)}
              </Button>
            </td>
          </tr>
        </tbody>
      </Table>
      <div className="pad flex flex-col gap-2.5 border-t border-line-2">
        {data.gaps.map((g) => (
          <Note key={g.countries.join()} tone="warn" role="note">
            <span className="flex flex-col gap-1">
              <span className="font-bold">{t(m.gapTitle, { region: regionLabel(g.countries) })}</span>
              <span>{t(m.gapText, { sender: g.sender, region: regionLabel(g.countries), held: g.held })}</span>
            </span>
          </Note>
        ))}
        <span className="k">{t(m.routesFoot)}</span>
      </div>
      {editing ? <RouteDialog data={data} editing={editing} onClose={() => setEditing(null)} /> : null}
    </Card>
  )
}

/** Add or change a route, or the rule for every other country; saves the routes. */
function RouteDialog({ data, editing, onClose }: { data: Rules; editing: Route | 'new' | 'other'; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const route = typeof editing === 'object' ? editing : undefined
  const [countries, setCountries] = useState(route ? route.countries.join(', ') : '')
  const [senderId, setSenderId] = useState(route?.senderId ?? data.senders[0]?.id ?? '')
  const [fallbackId, setFallbackId] = useState(route?.fallbackId ?? '')
  const [other, setOther] = useState<string>(data.other === 'route' && data.otherRouteId ? data.otherRouteId : '')
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/routes', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.routesSaved) })
      onClose()
    },
    onError: (e) => setError(errorMessage(e)),
  })
  const routesBody = (replace?: { countries: string[]; senderId: string; fallbackId?: string } | null) => {
    const list: { countries: string[]; senderId: string; fallbackId?: string }[] = data.routes
      .filter((r) => !route || r.id !== route.id)
      .map((r) => ({ countries: r.countries, senderId: r.senderId, fallbackId: r.fallbackId || undefined }))
    if (replace) list.push(replace)
    return list
  }
  const apply = (remove?: boolean) => {
    setError(undefined)
    if (editing === 'other') {
      const idx = data.routes.findIndex((r) => r.id === other)
      save.mutate({
        params: { path: { workspace: slug } },
        body: {
          routes: data.routes.map((r) => ({ countries: r.countries, senderId: r.senderId, fallbackId: r.fallbackId || undefined })),
          other: idx >= 0 ? 'route' : 'dont_send',
          otherRoute: idx >= 0 ? idx : undefined,
        },
      })
      return
    }
    let next: { countries: string[]; senderId: string; fallbackId?: string } | null = null
    if (!remove) {
      const cs = parseCountries(countries)
      if (!cs || cs.length === 0) {
        setError(t(m.countriesInvalid))
        return
      }
      next = { countries: cs, senderId, fallbackId: fallbackId || undefined }
    }
    const routes = routesBody(next)
    const otherIdx = data.other === 'route' && data.otherRouteId ? data.routes.filter((r) => !route || r.id !== route.id).findIndex((r) => r.id === data.otherRouteId) : -1
    save.mutate({
      params: { path: { workspace: slug } },
      body: { routes, other: otherIdx >= 0 ? 'route' : 'dont_send', otherRoute: otherIdx >= 0 ? otherIdx : undefined },
    })
  }
  const title = editing === 'new' ? t(m.routeDialogAdd) : editing === 'other' ? t(m.otherDialog) : t(m.routeDialogEdit, { region: regionLabel(route!.countries) })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={title}
      footer={
        <>
          {route ? (
            <Button variant="danger" className="mr-auto max-md:mr-0" loading={save.isPending} onClick={() => apply(true)}>
              {t(m.routeRemove)}
            </Button>
          ) : null}
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} onClick={() => apply()}>
            {t(m.routeApply)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {error ? <Note tone="bad">{error}</Note> : null}
        {editing === 'other' ? (
          <Field label={t(m.otherCountries)}>
            <Select value={other} onChange={(e) => setOther(e.target.value)}>
              <option value="">{t(m.dontSend)}</option>
              {data.routes.map((r) => (
                <option key={r.id} value={r.id}>
                  {t(m.otherRoute, { sender: `${r.provider} · ${r.sender}` })}
                </option>
              ))}
            </Select>
          </Field>
        ) : (
          <>
            <Field label={t(m.routeCountries)} hint={t(m.routeCountriesHint)}>
              <Input value={countries} onChange={(e) => setCountries(e.target.value)} autoComplete="off" />
            </Field>
            <Field label={t(m.routeSender)}>
              <Select value={senderId} onChange={(e) => setSenderId(e.target.value)}>
                {data.senders.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.provider} · {s.label}
                  </option>
                ))}
              </Select>
            </Field>
            <Field label={t(m.routeFallback)}>
              <Select value={fallbackId} onChange={(e) => setFallbackId(e.target.value)}>
                <option value="">{t(m.routeFallbackNone)}</option>
                {data.senders
                  .filter((s) => s.id !== senderId)
                  .map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.provider} · {s.label}
                    </option>
                  ))}
              </Select>
            </Field>
          </>
        )}
      </div>
    </Dialog>
  )
}

/* ---------------- Quiet hours ---------------- */

const fromOptions = ['06:00', '07:00', '08:00', '09:00', '10:00', '11:00', '12:00']
const toOptions = ['17:00', '18:00', '19:00', '20:00', '21:00', '22:00', '23:00']

/** "America/Chicago" → "Chicago". */
export function placeOf(zone: string): string {
  const last = zone.split('/').pop() ?? zone
  return last.replace(/_/g, ' ')
}

function QuietHours({
  data,
  form,
  set,
  ro,
  problem,
}: {
  data: Rules
  form: Form
  set: <K extends keyof Form>(k: K, v: Form[K]) => void
  ro: boolean
  problem: 'early' | 'late' | 'empty' | undefined
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const zones = data.zones
  const [zone, setZone] = useState(zones[0]?.zone ?? '')
  const z = zones.find((x) => x.zone === zone)
  const from = minutes(form.quietFrom)
  const to = minutes(form.quietTo)
  const day = 24 * 60
  const now = z ? minutes(z.local) : undefined
  const inside = now !== undefined && !problem && now >= from && now < to
  const pctOf = (v: number) => `${((v / day) * 100).toFixed(2)}%`
  const lim = { left: pctOf(8 * 60), width: pctOf(13 * 60) }
  const ok = { left: pctOf(Math.max(from, 8 * 60)), width: pctOf(Math.max(0, Math.min(to, 21 * 60) - Math.max(from, 8 * 60))) }
  return (
    <Card title={t(m.quietTitle)} lead={t(m.quietLead)}>
      <div className="pad flex flex-col gap-3.5">
        <div className="flex flex-wrap gap-2.5">
          <Field label={t(m.from)}>
            <Select aria-label={t(m.windowStarts)} style={{ width: 110 }} disabled={ro} value={form.quietFrom} onChange={(e) => set('quietFrom', e.target.value)}>
              {fromOptions.map((o) => (
                <option key={o}>{o}</option>
              ))}
            </Select>
          </Field>
          <Field label={t(m.to)}>
            <Select aria-label={t(m.windowEnds)} style={{ width: 110 }} disabled={ro} value={form.quietTo} onChange={(e) => set('quietTo', e.target.value)}>
              {toOptions.map((o) => (
                <option key={o}>{o}</option>
              ))}
            </Select>
          </Field>
          {zones.length ? (
            <Field label={t(m.showNowIn)} className="flex-[1_1_220px]">
              <Select value={zone} onChange={(e) => setZone(e.target.value)}>
                {zones.map((x) => (
                  <option key={x.zone} value={x.zone}>
                    {t(m.zoneNow, { zone: x.zone, time: x.local })}
                  </option>
                ))}
              </Select>
            </Field>
          ) : null}
        </div>
        <div className="sms-qh">
          <div className="sms-qh-t" role="img" aria-label={t(m.trackAria, { from: form.quietFrom, to: form.quietTo })}>
            <span className="lim" style={lim} />
            <span className="win" style={ok}>
              {problem ? null : `${form.quietFrom}–${form.quietTo}`}
            </span>
            {problem === 'late' ? <span className="win x" style={{ left: pctOf(21 * 60), width: pctOf(to - 21 * 60) }} /> : null}
            {problem === 'early' ? <span className="win x" style={{ left: pctOf(from), width: pctOf(8 * 60 - from) }} /> : null}
            {now !== undefined ? <span className="now" style={{ left: pctOf(now) }} /> : null}
          </div>
          <div className="sms-qh-x" aria-hidden="true">
            {['00:00', '06:00', '12:00', '18:00', '24:00'].map((l, i) => (
              <span key={l} style={{ left: `${i * 25}%` }}>
                {l}
              </span>
            ))}
          </div>
          <div className="sms-bm-l">
            <span>
              <span className="sms-bsw" style={{ background: 'var(--accent)' }} />
              {t(m.legendWindow, { from: form.quietFrom, to: form.quietTo })}
            </span>
            <span>
              <span className="sms-bsw" style={{ background: '#E3F1EE', border: '1px solid #99F6E4', boxSizing: 'border-box' }} />
              {t(m.legendLimit)}
            </span>
            {z ? (
              <span>
                <span aria-hidden="true" style={{ width: 3, height: 12, background: 'var(--ink)', borderRadius: 1, display: 'inline-block' }} />
                {inside
                  ? t(m.legendNowInside, { time: z.local, place: placeOf(z.zone) })
                  : t(m.legendNowOutside, { time: z.local, place: placeOf(z.zone), opens: form.quietFrom })}
              </span>
            ) : null}
          </div>
        </div>
        {problem ? (
          <Note tone="bad">
            <span className="flex flex-col gap-1">
              <span className="font-bold">
                {problem === 'empty' ? t(m.emptyWindow) : t(m.askedFor, { from: form.quietFrom, to: form.quietTo })}
              </span>
              {problem !== 'empty' ? (
                <span>
                  {problem === 'late' ? t(m.tooLateTitle) : t(m.tooEarlyTitle)} {t(m.limitText)}
                </span>
              ) : null}
            </span>
          </Note>
        ) : null}
        <div className="sms-cbox">
          <span className="text-[13px] font-semibold">{t(m.whoseZone)}</span>
          <span className="k text-[#374151]">{t(m.whoseZoneText)}</span>
          {data.assumed ? (
            <span className="text-[13px]">
              <T msg={m.assumed} values={{ count: data.assumed, b: (c) => <b className="n">{c}</b> }} />{' '}
              <Link to="/w/$workspace/audience" params={{ workspace: slug }} search={{ q: 'timezone:assumed' } as never}>
                {t(m.reviewThem)}
              </Link>
            </span>
          ) : null}
        </div>
      </div>
      <ToggleRow title={t(m.pushWindow)} text={t(m.pushWindowText)} checked={form.pushWindow} disabled={ro} onChange={(v) => set('pushWindow', v)} />
      <ToggleRow title={t(m.whatsAppWindow)} text={t(m.whatsAppWindowText)} checked={form.whatsAppWindow} disabled={ro} onChange={(v) => set('whatsAppWindow', v)} />
    </Card>
  )
}

function ToggleRow({
  title,
  text,
  checked,
  disabled,
  onChange,
  children,
}: {
  title: ReactNode
  text: ReactNode
  checked: boolean
  disabled?: boolean
  onChange: (v: boolean) => void
  children?: ReactNode
}) {
  return (
    <div className="sms-srow">
      <div className="tx">
        <span className="font-semibold">{title}</span>
        <span className="k text-[#4B5563]">{text}</span>
        {children}
      </div>
      <div className="ct">
        <input type="checkbox" className="tg" checked={checked} disabled={disabled} aria-label={typeof title === 'string' ? title : undefined} onChange={(e) => onChange(e.target.checked)} />
      </div>
    </div>
  )
}

/* ---------------- Keywords and replies ---------------- */

type KwGroup = 'optOutKeywords' | 'optInKeywords' | 'helpKeywords'

function Keywords({ data, form, set, ro }: { data: Rules; form: Form; set: <K extends keyof Form>(k: K, v: Form[K]) => void; ro: boolean }) {
  const t = useT()
  const all = [...form.optOutKeywords, ...form.optInKeywords, ...form.helpKeywords].map((k) => k.toUpperCase())
  const groups: { key: KwGroup; label: string; hint: string }[] = [
    { key: 'optOutKeywords', label: t(m.optOut), hint: t(m.optOutHint) },
    { key: 'optInKeywords', label: t(m.optIn), hint: t(m.optInHint) },
    { key: 'helpKeywords', label: t(m.help), hint: t(m.helpHint) },
  ]
  return (
    <Card title={t(m.keywordsTitle)} lead={t(m.keywordsLead)}>
      <div className="pad flex flex-col gap-4">
        {groups.map((g) => (
          <KeywordGroup key={g.key} label={g.label} hint={g.hint} words={form[g.key]} all={all} ro={ro} onChange={(w) => set(g.key, w)} />
        ))}
        <span className="k">{t(m.keywordsNote)}</span>
        <div className="grid gap-3.5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(300px, 100%), 1fr))' }}>
          <ReplyField label={t(m.helpReply)} keyword="HELP" value={form.helpReply} ro={ro} onChange={(v) => set('helpReply', v)} />
          <ReplyField label={t(m.optOutReply)} keyword="STOP" value={form.optOutReply} ro={ro} onChange={(v) => set('optOutReply', v)} />
        </div>
        <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
          <legend className="lbl mb-1.5">{t(m.whoAnswers)}</legend>
          <OptionCard
            name="who"
            value="reputecast"
            checked={form.keywordsBy === 'reputecast'}
            disabled={ro}
            onChange={() => set('keywordsBy', 'reputecast')}
            title={t(m.reputecastAnswers)}
            description={t(m.reputecastAnswersText)}
          />
          <OptionCard
            name="who"
            value="provider"
            checked={form.keywordsBy === 'provider'}
            disabled={ro || data.providerAnswers.length === 0}
            onChange={() => set('keywordsBy', 'provider')}
            title={t(m.providerAnswers)}
            description={data.providerAnswers.length ? t(m.providerAnswersText, { names: formatList(data.providerAnswers) }) : t(m.providerAnswersNone)}
          />
        </fieldset>
      </div>
    </Card>
  )
}

function KeywordGroup({ label, hint, words, all, ro, onChange }: { label: string; hint: string; words: string[]; all: string[]; ro: boolean; onChange: (w: string[]) => void }) {
  const t = useT()
  const [adding, setAdding] = useState(false)
  const [value, setValue] = useState('')
  const [error, setError] = useState<string>()
  const input = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (adding) input.current?.focus()
  }, [adding])
  const add = () => {
    const w = value.trim().toUpperCase()
    if (!/^[A-Z0-9]+$/.test(w)) {
      setError(t(m.keywordInvalid))
      return
    }
    if (all.includes(w)) {
      setError(t(m.keywordTaken, { keyword: w }))
      return
    }
    onChange([...words, w])
    setValue('')
    setError(undefined)
    setAdding(false)
  }
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[13px] font-semibold">{label}</span>
      <div className="flex flex-wrap items-center gap-1.5">
        {words.map((w) => (
          <span key={w} className="sms-kwd">
            {w}
            {!ro ? (
              <button type="button" aria-label={t(m.removeKeyword, { keyword: w })} onClick={() => onChange(words.filter((x) => x !== w))}>
                <svg width="10" height="10" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                  <path d="M4 4l8 8M12 4l-8 8" />
                </svg>
              </button>
            ) : null}
          </span>
        ))}
        {adding ? (
          <span className="inline-flex items-center gap-1.5">
            <input
              ref={input}
              className="in mono h-6 w-[110px] px-2 text-[12px]"
              aria-label={t(m.addKeywordLabel, { group: label.toLowerCase() })}
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  add()
                }
                if (e.key === 'Escape') setAdding(false)
              }}
              onBlur={() => (value.trim() ? add() : setAdding(false))}
            />
          </span>
        ) : (
          <button type="button" className="btn sm" style={{ minHeight: 24, padding: '0 8px' }} disabled={ro} onClick={() => setAdding(true)}>
            <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden="true">
              <path d="M8 3v10M3 8h10" />
            </svg>
            {t(m.addKeyword)}
          </button>
        )}
      </div>
      {error ? <span className="text-[12px] text-bad-strong">{error}</span> : null}
      <span className="k">{hint}</span>
    </div>
  )
}

/** A reply with its live counter; over one segment it says what that costs. */
function ReplyField({ label, keyword, value, ro, onChange }: { label: string; keyword: string; value: string; ro: boolean; onChange: (v: string) => void }) {
  const t = useT()
  const c = countText(value)
  return (
    <div className="flex flex-col gap-1.5">
      <Field label={label}>
        <Textarea
          className="min-h-16 text-[13px] leading-[1.45]"
          style={{ resize: 'vertical' }}
          disabled={ro}
          value={value}
          invalid={c.segments > 1}
          onChange={(e) => onChange(e.target.value)}
        />
      </Field>
      <div className="flex flex-wrap justify-between gap-2">
        <span className="k" style={c.segments > 1 ? { color: '#92400E' } : undefined}>
          {t(m.counter, { characters: c.characters, segments: Math.max(c.segments, 1), encoding: c.encoding === 'gsm7' ? 'GSM-7' : 'UCS-2' })}
        </span>
        <span className="k">{t(m.sentOnce)}</span>
      </div>
      {c.segments > 1 ? (
        <Note tone="warn" role="note">
          {t(m.replyLong, { characters: c.characters, segments: c.segments, keyword, limit: c.encoding === 'gsm7' ? 160 : 70 })}
        </Note>
      ) : null}
    </div>
  )
}

/* ---------------- Route tester ---------------- */

function RouteTester({ data }: { data: Rules }) {
  const t = useT()
  const { slug } = useWorkspace()
  const order = [...new Set(data.senders.map((s) => s.provider))]
  const options = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sms/message-options', { params: { path: { workspace: slug } } })
  const [phone, setPhone] = useState(data.testNumbers[data.testNumbers.length - 1]?.display ?? '')
  const [message, setMessage] = useState('')
  const test = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/route-test')
  const msgKey = message || options.data?.items[0]?.key || ''
  useEffect(() => {
    if (!phone.trim()) return
    const id = window.setTimeout(() => {
      test.mutate({ params: { path: { workspace: slug } }, body: { phone, message: msgKey || undefined } })
    }, 350)
    return () => window.clearTimeout(id)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [phone, msgKey, slug])
  const r = test.data
  const name = (o: Schemas['SmsMessageOption']) => (o.key === 'help' ? t(m.helpReply) : o.key === 'opt_out' ? t(m.optOutReply) : o.name)
  return (
    <Card title={t(m.testerTitle)} lead={t(m.testerLead)}>
      <div className="pad flex flex-col gap-3.5">
        <Field label={t(m.phone)}>
          <Input className="mono" aria-label={t(m.phoneLabel)} value={phone} onChange={(e) => setPhone(e.target.value)} inputMode="tel" autoComplete="off" />
        </Field>
        <Field label={t(m.message)}>
          <Select aria-label={t(m.messageLabel)} value={msgKey} onChange={(e) => setMessage(e.target.value)}>
            {(options.data?.items ?? []).map((o) => (
              <option key={o.key} value={o.key}>
                {t(m.messageOption, { name: name(o), segments: o.segments })}
              </option>
            ))}
          </Select>
        </Field>
        {test.isError ? <Note tone="bad">{errorMessage(test.error)}</Note> : null}
        {r ? <TesterResult r={r} color={providerColor(order, r.provider)} /> : null}
      </div>
    </Card>
  )
}

function TesterResult({ r, color }: { r: Schemas['SmsRouteTest']; color: string }) {
  const t = useT()
  const code = r.callingCode.replace('+', '')
  const place = placeOf(r.zone)
  const b = registrationBadge(r.registration, t)
  return (
    <div className="flex flex-col gap-3 rounded-[10px] border border-line px-4 py-3.5" style={{ background: '#FBFBFC' }}>
      {r.route === 'none' ? (
        <Note tone="warn" role="note">
          {t(m.noRoute, { country: countryName(r.country) })}
        </Note>
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-2.5">
            <span className="dot" style={{ color, width: 10, height: 10 }} aria-hidden="true" />
            <span className="text-[17px] font-bold">
              {r.provider} · {r.sender}
            </span>
          </div>
          <div className="sms-dl">
            <div>
              <span className="k">{t(m.country)}</span>
              <span>
                <CountryName code={r.country} /> <span className="k">{t(m.fromCode, { code })}</span>
              </span>
            </div>
            <div>
              <span className="k">{t(m.registration)}</span>
              <span>{b.text}</span>
            </div>
            <div>
              <span className="k">{t(m.price)}</span>
              <span>
                <T msg={r.segments > 1 ? m.priceTotal : m.perSegment} values={{ price: price(r.price), total: money(r.cost, 4), b: (c) => <b className="n">{c}</b> }} />
              </span>
            </div>
            <div>
              <span className="k">{t(m.localTime)}</span>
              <span>
                {r.local} · {r.zone}
              </span>
            </div>
          </div>
          <div className="border-t border-line pt-3">
            <div className="flex flex-col gap-2.5">
              <Step n={1}>
                {r.route === 'route'
                  ? t(m.step1, { code, country: countryName(r.country), region: regionSentence(r.routeCountries) })
                  : t(m.step1Other, { code, country: countryName(r.country) })}
              </Step>
              <Step n={2}>
                {r.canReply
                  ? t(m.step2, { provider: r.provider, sender: r.sender })
                  : t(m.step2NoReply, { provider: r.provider, sender: r.sender, country: countryName(r.country) })}
              </Step>
              <Step n={3}>{r.suppressed ? t(m.step3Suppressed, { number: r.number, reason: r.suppressed }) : t(m.step3Ok, { number: r.number })}</Step>
              <Step n={4}>{r.inside ? t(m.step4Inside, { time: r.local, place, window: r.window }) : t(m.step4Outside, { time: r.local, place, window: r.window })}</Step>
            </div>
          </div>
          {!r.canSend ? (
            <Note tone="warn" role="note">
              {t(m.cantSend, { sender: r.sender, status: b.text.toLowerCase() })}
            </Note>
          ) : r.inside ? (
            <Note tone="ok" role="note">
              {t(m.goesNow)}
            </Note>
          ) : (
            <Note tone="warn" role="note">
              {t(m.waits, { opens: r.opens })}
            </Note>
          )}
        </>
      )}
    </div>
  )
}

function Step({ n, children }: { n: number; children: ReactNode }) {
  return (
    <div className="sms-hstep">
      <span className="hn">{n}</span>
      <span>{children}</span>
    </div>
  )
}

/* ---------------- Budget and guards ---------------- */

function NumberRow({ label, sub, prefix, suffix, value, onChange, width, ro, invalid }: {
  label: string
  sub?: string
  prefix?: string
  suffix?: string
  value: string
  onChange: (v: string) => void
  width: number
  ro: boolean
  invalid?: boolean
}) {
  return (
    <div className="sms-nr">
      <span className="flex min-w-0 flex-[1_1_150px] flex-col gap-px">
        <span className="text-[13px] font-medium">{label}</span>
        {sub ? <span className="k">{sub}</span> : null}
      </span>
      <span className="inline-flex flex-none items-center gap-1.5">
        {prefix ? <span className="k">{prefix}</span> : null}
        <input
          className={`in${invalid ? ' bad' : ''}`}
          type="text"
          inputMode="decimal"
          aria-label={label}
          aria-invalid={invalid || undefined}
          style={{ width, textAlign: 'right' }}
          value={value}
          disabled={ro}
          onChange={(e) => onChange(e.target.value)}
        />
        {suffix ? <span className="k">{suffix}</span> : null}
      </span>
    </div>
  )
}

function Budget({
  data,
  form,
  set,
  ro,
  budgetText,
  setBudgetText,
  budgetBad,
}: {
  data: Rules
  form: Form
  set: <K extends keyof Form>(k: K, v: Form[K]) => void
  ro: boolean
  budgetText: string
  setBudgetText: (v: string) => void
  budgetBad: boolean
}) {
  const t = useT()
  const month = monthName(data.month, 'long')
  const budget = budgetBad ? data.budget : Number(budgetText.replace(/[$,\s]/g, ''))
  const pctField = (k: keyof Form['thresholds'], label: string) => (
    <NumberRow
      label={label}
      suffix="%"
      width={60}
      ro={ro}
      value={String(form.thresholds[k])}
      invalid={!(form.thresholds[k] > 0 && form.thresholds[k] <= 100)}
      onChange={(v) => set('thresholds', { ...form.thresholds, [k]: Number(v) || 0 })}
    />
  )
  return (
    <section className="card" id="budget">
      <div className="sec">
        <h2 className="h2">{t(m.budgetTitle)}</h2>
      </div>
      <div className="pad flex flex-col gap-3">
        <div>
          <NumberRow
            label={t(m.monthlyBudget)}
            sub={data.budgetSetBy && data.budgetSetAt ? t(m.setBy, { name: data.budgetSetBy, date: dayMonthDate(data.budgetSetAt) }) : undefined}
            prefix="$"
            width={90}
            ro={ro}
            value={budgetText}
            invalid={budgetBad}
            onChange={setBudgetText}
          />
          <NumberRow
            label={t(m.alertAt)}
            sub={t(m.alertHint)}
            suffix="%"
            width={60}
            ro={ro}
            value={String(form.alertPct)}
            invalid={!(form.alertPct > 0 && form.alertPct <= 100)}
            onChange={(v) => set('alertPct', Number(v) || 0)}
          />
          {budgetBad ? <span className="text-[12px] text-bad-strong">{t(m.budgetInvalid)}</span> : null}
        </div>
        <div className="sms-bm">
          <div className="sms-sum">
            <span className="k">{t(m.budgetLabel, { month })}</span>
            <span className="n">
              {money(data.spent)} <span className="k">{t(m.budgetOf, { budget: money(budget) })}</span>
            </span>
          </div>
          <BudgetMeter spent={data.spent} budget={budget} alertPct={form.alertPct} month={month} legend />
        </div>
        <span className="k">{t(m.budgetNote)}</span>
      </div>
      <div className="pad flex flex-col gap-1.5 border-t border-line-2">
        <span className="font-semibold">{t(m.autoPause)}</span>
        <div>
          {pctField('failed', t(m.failedAbove))}
          {pctField('filtered', t(m.filteredAbove))}
          {pctField('optOuts', t(m.optOutsAbove))}
        </div>
        <span className="k">{t(m.autoPauseNote)}</span>
      </div>
    </section>
  )
}

/* ---------------- Test numbers ---------------- */

function TestNumbers({ data, ro }: { data: Rules; ro: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const [adding, setAdding] = useState(false)
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/sms/test-numbers/{id}', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'deleted', title: t(m.testNumberRemoved) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  return (
    <Card title={t(m.testNumbersTitle)} lead={t(m.testNumbersLead)}>
      {data.testNumbers.length === 0 ? <div className="pad k">{t(m.noTestNumbers)}</div> : null}
      {data.testNumbers.map((n) => (
        <div key={n.id} className="flex flex-wrap items-center justify-between gap-2.5 border-t border-line-2 px-[18px] py-3">
          <div className="flex min-w-0 items-center gap-2.5">
            <Avatar name={n.name} color={n.color} size={28} fontSize={11} />
            <div className="flex min-w-0 flex-col leading-[1.3]">
              <span className="font-semibold">{n.name}</span>
              <span className="k">{n.role}</span>
            </div>
          </div>
          <span className="inline-flex items-center gap-2">
            <span className="mono whitespace-nowrap text-[13px]">{n.display}</span>
            <CountryChips codes={[n.country]} />
            {!ro ? (
              <button
                type="button"
                className="btn sm ic border-transparent"
                aria-label={t(m.removeTestNumber, { name: n.name })}
                onClick={() => remove.mutate({ params: { path: { workspace: slug, id: n.id } } })}
              >
                <svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                  <path d="M4 4l8 8M12 4l-8 8" />
                </svg>
              </button>
            ) : null}
          </span>
        </div>
      ))}
      <div className="sms-foot">
        <span className="k">{t(m.testsFoot, { count: data.testsSent, cost: money(data.testsCost) })}</span>
        <Button size="sm" disabled={ro} onClick={() => setAdding(true)}>
          {t(m.addTestNumber)}
        </Button>
      </div>
      {adding ? <AddTestNumber data={data} onClose={() => setAdding(false)} /> : null}
    </Card>
  )
}

function AddTestNumber({ data, onClose }: { data: Rules; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const [userId, setUserId] = useState(data.members[0]?.userId ?? '')
  const [phone, setPhone] = useState('')
  const [error, setError] = useState<string>()
  const add = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/test-numbers', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.testNumberAdded) })
      onClose()
    },
    onError: (e) => setError(errorMessage(e)),
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.testNumberDialog)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={add.isPending} onClick={() => add.mutate({ params: { path: { workspace: slug } }, body: { phone, userId: userId || undefined } })}>
            {t(m.addTestNumber)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        {error ? <Note tone="bad">{error}</Note> : null}
        <Field label={t(m.testNumberMember)}>
          <Select value={userId} onChange={(e) => setUserId(e.target.value)}>
            {data.members.map((mb) => (
              <option key={mb.userId} value={mb.userId}>
                {mb.name} · {mb.role}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(m.testNumberPhone)} hint={t(m.testNumberPhoneHint)}>
          <Input className="mono" value={phone} inputMode="tel" autoComplete="off" onChange={(e) => setPhone(e.target.value)} />
        </Field>
      </div>
    </Dialog>
  )
}

/* ---------------- Retention ---------------- */

function Retention({ form, set, ro }: { form: Form; set: <K extends keyof Form>(k: K, v: Form[K]) => void; ro: boolean }) {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <h2 className="h2">{t(m.retentionTitle)}</h2>
      </div>
      <div className="pad flex flex-col gap-2.5">
        <Field label={t(m.keepFor)}>
          <Select disabled={ro} value={String(form.retentionDays)} onChange={(e) => set('retentionDays', Number(e.target.value) as Form['retentionDays'])}>
            {retention.map((d) => (
              <option key={d} value={d}>
                {d === 0 ? t(m.keepNone) : t(m.keepDays, { n: d })}
              </option>
            ))}
          </Select>
        </Field>
        <span className="k">{t(m.retentionNote)}</span>
      </div>
    </section>
  )
}

/* ---------------- Message rules ---------------- */

function MessageRules({ data, form, set, ro }: { data: Rules; form: Form; set: <K extends keyof Form>(k: K, v: Form[K]) => void; ro: boolean }) {
  const t = useT()
  const conf = countText(form.confirmation)
  const prefixLen = Array.from(form.brandPrefix.trim()).length + (form.brandPrefix.trim() ? 1 : 0)
  const optOutLen = Array.from(form.optOutText.trim()).length + (form.optOutText.trim() ? 1 : 0)
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(m.messageRulesTitle)}</h2>
          <span className="k">{t(m.messageRulesLead)}</span>
        </div>
      </div>
      <div className="sms-srow" style={{ borderTop: 0 }}>
        <div className="tx">
          <span className="font-semibold">{t(m.brandPrefix)}</span>
          <span className="k text-[#4B5563]">{t(m.brandPrefixText, { count: prefixLen })}</span>
        </div>
        <div className="ct">
          <input className="in" type="text" aria-label={t(m.brandPrefix)} style={{ width: 140 }} disabled={ro} value={form.brandPrefix} onChange={(e) => set('brandPrefix', e.target.value)} />
        </div>
      </div>
      <div className="sms-srow">
        <div className="tx">
          <span className="font-semibold">{t(m.optOutText)}</span>
          <span className="k text-[#4B5563]">{t(m.optOutTextText, { count: optOutLen })}</span>
          <label className="mt-1 flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={form.optOutInSteps} disabled={ro} onChange={(e) => set('optOutInSteps', e.target.checked)} />
            {t(m.optOutInSteps)}
          </label>
        </div>
        <div className="ct">
          <input className="in" type="text" aria-label={t(m.optOutText)} style={{ width: 220 }} disabled={ro} value={form.optOutText} onChange={(e) => set('optOutText', e.target.value)} />
        </div>
      </div>
      <ToggleRow title={t(m.confirmation)} text={t(m.confirmationText)} checked={form.confirmationOn} disabled={ro} onChange={(v) => set('confirmationOn', v)}>
        <div className="sms-cbox mt-1.5">
          <textarea
            className="w-full resize-y border-0 bg-transparent p-0 text-[13px] outline-none"
            aria-label={t(m.confirmation)}
            rows={2}
            disabled={ro}
            value={form.confirmation}
            onChange={(e) => set('confirmation', e.target.value)}
          />
          <span className="k">{t(m.counter, { characters: conf.characters, segments: Math.max(conf.segments, 1), encoding: conf.encoding === 'gsm7' ? 'GSM-7' : 'UCS-2' })}</span>
        </div>
      </ToggleRow>
      <ToggleRow title={t(m.requireYes)} text={form.requireYes ? t(m.requireYesOn) : t(m.requireYesOff)} checked={form.requireYes} disabled={ro} onChange={(v) => set('requireYes', v)} />
      <ShortDomain data={data} ro={ro} />
    </section>
  )
}

function ShortDomain({ data, ro }: { data: Rules; ro: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const d = data.shortDomain
  const [domain, setDomain] = useState(d.domain)
  const check = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms/short-domain/check', {
    onSuccess: () => void refresh(),
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const setDom = $api.useMutation('put', '/api/v1/workspaces/{workspace}/sms/short-domain', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.domainSaved) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const sub = d.domain.split('.')[0] ?? ''
  const badge =
    d.status === 'verified' ? (
      <Badge tone="ok" dot>
        {d.verifiedAt ? t(m.verifiedOn, { date: dayMonthYear(d.verifiedAt) }) : t(m.notVerified)}
      </Badge>
    ) : d.status === 'failed' ? (
      <Badge tone="bad">{t(m.checkFailed)}</Badge>
    ) : d.domain ? (
      <Badge tone="warn">{t(m.notVerified)}</Badge>
    ) : null
  return (
    <>
      <div className="sms-srow">
        <div className="tx">
          <span className="inline-flex flex-wrap items-center gap-2 font-semibold">
            {t(m.shortDomain)}
            {d.domain ? <span className="mono font-semibold">{d.domain}</span> : null}
            {badge}
          </span>
          <span className="k text-[#4B5563]">{d.domain ? t(m.shortDomainText, { target: d.target }) : t(m.shortDomainNone)}</span>
        </div>
        <div className="ct">
          {d.domain ? (
            <Button size="sm" disabled={ro} loading={check.isPending} onClick={() => check.mutate({ params: { path: { workspace: slug } } })}>
              {d.status === 'verified' ? t(m.checkDns) : t(m.checkNow)}
            </Button>
          ) : (
            <>
              <input
                className="in mono"
                aria-label={t(m.domainLabel)}
                placeholder={t(m.domainPlaceholder)}
                style={{ width: 180 }}
                disabled={ro}
                value={domain}
                onChange={(e) => setDomain(e.target.value)}
              />
              <Button size="sm" disabled={ro || !domain.trim()} loading={setDom.isPending} onClick={() => setDom.mutate({ params: { path: { workspace: slug } }, body: { domain: domain.trim() } })}>
                {t(m.setDomain)}
              </Button>
            </>
          )}
        </div>
      </div>
      {d.domain && d.status !== 'verified' ? (
        <div className="pad flex flex-col gap-3 border-t border-line-2">
          <Note tone="warn" role="note">
            {t(m.cnameNote, { domain: d.domain, target: d.target })}
          </Note>
          <Table caption={t(m.dnsRecord)}>
            <thead>
              <tr>
                <th scope="col">{t(m.type)}</th>
                <th scope="col">{t(m.name)}</th>
                <th scope="col">{t(m.value)}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td className="mono">{'CNAME'}</td>
                <td className="mono">{sub}</td>
                <td className="mono">{d.target}</td>
              </tr>
            </tbody>
          </Table>
          {d.checkedAt ? <span className="k">{t(m.lastCheck, { time: clock(d.checkedAt), detail: d.detail ?? '' })}</span> : null}
          <span className="k">{t(m.untilVerified)}</span>
        </div>
      ) : null}
      {d.domain ? (
        <div className="sms-srow" style={{ background: '#F9FAFB' }}>
          <div className="tx">
            <span className="inline-flex items-center gap-2 font-semibold">{t(m.monitorTitle, { domain: d.domain })}</span>
            <span className="k text-[#4B5563]">{t(m.monitorText)}</span>
          </div>
          <div className="ct">
            <Link to="/w/$workspace/settings/integrations/hostrepute" params={{ workspace: slug }} className="btn sm">
              {t(m.monitor)}
            </Link>
          </div>
        </div>
      ) : null}
    </>
  )
}
