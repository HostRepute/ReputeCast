import { useEffect, useMemo, useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, monthName } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { useInvalidateWebsite, WS, type Access, type AccessInput, type Overview } from './api'
import { ma } from './access.messages'
import { m } from './messages'
import { visibilityIcon } from './postFormat'
import { ReadOnlyNote, SavedLine, WebsitePage } from './WebsiteFrame'

type Tier = Access['tiers'][number]

function inputOf(a: Access): AccessInput {
  return {
    signinOn: a.signinOn,
    linkMinutes: a.linkMinutes as AccessInput['linkMinutes'],
    sessionDays: a.sessionDays,
    signupOn: a.signupOn,
    lists: a.lists.filter((l) => l.offered).map((l) => ({ listId: l.id, checked: l.checked, consent: l.consent })),
    defaultVisibility: a.defaultVisibility,
    campaignVisibility: a.campaignVisibility,
    meterVisitorsOn: a.meterVisitorsOn,
    meterVisitors: a.meterVisitors,
    meterMembersOn: a.meterMembersOn,
    meterMembers: a.meterMembers,
    neverFree: a.neverFree.map((p) => p.id),
    paywall: { ...a.paywall, tiers: [...a.paywall.tiers] },
  }
}

/** Website › Access (WebsiteAccess, WebsiteStates · access). */
export function AccessPage({ overview }: { overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/access`, { params: { path: { workspace: slug } } })
  if (q.isError && !q.data) {
    return (
      <WebsitePage view="access" overview={overview}>
        <ErrorPanel title={t(ma.saveFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      </WebsitePage>
    )
  }
  if (!q.data) {
    return (
      <WebsitePage view="access" overview={overview}>
        <Loading label={t(m.loading)} className="grid gap-5 lg:grid-cols-2">
          <Skeleton w="100%" h={520} r={10} />
          <Skeleton w="100%" h={520} r={10} />
        </Loading>
      </WebsitePage>
    )
  }
  return <AccessEditor key={q.data.updatedAt} data={q.data} overview={overview} />
}

function AccessEditor({ data, overview }: { data: Access; overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const canManage = useCan('website:manage')
  const ro = !canManage
  const initial = useMemo(() => inputOf(data), [data])
  const [draft, setDraft] = useState<AccessInput>(initial)
  const [neverFree, setNeverFree] = useState(data.neverFree)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const dirty = JSON.stringify(draft) !== JSON.stringify(initial)
  const set = (patch: Partial<AccessInput>) => setDraft((d) => ({ ...d, ...patch }))
  const host = overview.address.domain || overview.address.host
  const visitorsBad = draft.meterVisitorsOn && (draft.meterVisitors < 1 || draft.meterVisitors > 20)
  const membersBad = draft.meterMembersOn && (draft.meterMembers < 1 || draft.meterMembers > 20)
  const sessionBad = draft.sessionDays < 7 || draft.sessionDays > 365
  const invalid = visitorsBad || membersBad || sessionBad
  const save = async (): Promise<boolean> => {
    if (invalid) return false
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT(`${WS}/access`, { params: { path: { workspace: slug } }, body: draft })
      toast({ kind: 'success', title: t(ma.saved), detail: t(ma.savedDetail, { host }) })
      await invalidate()
      return true
    } catch (e) {
      setError(errorMessage(e))
      toast({ kind: 'error', title: t(ma.saveFailed), detail: errorMessage(e) })
      return false
    } finally {
      setBusy(false)
    }
  }
  const actions = canManage ? (
    <>
      {dirty ? null : <SavedLine at={data.updatedAt} by={data.updatedBy} />}
      <Button variant="primary" loading={busy} busyLabel={t(ma.saving)} disabled={!dirty} disabledReason={dirty && invalid ? t(ma.meterRange) : undefined} onClick={() => void save()}>
        {t(ma.save)}
      </Button>
    </>
  ) : undefined
  const paidBlocked = !data.payments.connected ? t(ma.connectToUsePaid) : data.tiers.length === 0 ? t(ma.tierFirst) : undefined
  return (
    <WebsitePage view="access" overview={overview} actions={actions}>
      {ro ? <ReadOnlyNote msg={m.readOnlyAccess} helpers={overview.helpers.manage} perms={{ perm: 'website:manage' }} /> : null}
      {!data.payments.connected ? (
        <Note tone="neutral">
          <div className="flex flex-col gap-2">
            <b>{t(ma.stripeTitle)}</b>
            <span>{t(ma.stripeText)}</span>
            <span>
              <ButtonLink size="sm" to="/w/$workspace/earn/settings" params={{ workspace: slug }} search={{ tab: 'payments' } as never}>
                {t(ma.stripeLink)}
              </ButtonLink>
            </span>
          </div>
        </Note>
      ) : data.tiers.length === 0 ? (
        <Note tone="info">
          <div className="flex flex-col gap-2">
            <b>{t(ma.noTiersTitle)}</b>
            <span>{t(ma.noTiersText, { account: data.payments.account, mode: data.payments.mode })}</span>
            <span>
              <ButtonLink size="sm" to="/w/$workspace/earn/plans" params={{ workspace: slug }}>
                {t(ma.noTiersLink)}
              </ButtonLink>
            </span>
          </div>
        </Note>
      ) : null}
      {error ? <Note tone="bad">{error}</Note> : null}
      <div className="grid items-start gap-5 min-[1101px]:grid-cols-2">
        <div className="flex min-w-0 flex-col gap-5">
          <SigninCard data={data} draft={draft} set={set} ro={ro} sessionBad={sessionBad} />
          <SignupCard data={data} draft={draft} set={set} ro={ro} />
          <VisibilityCard draft={draft} set={set} ro={ro} paidBlocked={paidBlocked} />
          <MeterCard
            draft={draft}
            set={set}
            ro={ro}
            visitorsBad={visitorsBad}
            membersBad={membersBad}
            neverFree={neverFree}
            onNeverFree={(list) => {
              setNeverFree(list)
              set({ neverFree: list.map((p) => p.id) })
            }}
          />
        </div>
        <div className="flex min-w-0 flex-col gap-5">
          <PaywallCard data={data} draft={draft} set={set} ro={ro} />
          <MembersCard data={data} overview={overview} />
        </div>
      </div>
      {dirty && canManage ? (
        <div className="card sticky bottom-3 z-10 flex flex-wrap items-center gap-3 px-[18px] py-3 shadow-[0_8px_24px_rgba(11,18,32,.12)]" role="region" aria-label={t(ma.unsaved)}>
          <Badge tone="warn" dot>
            {t(ma.unsaved)}
          </Badge>
          <span className="flex-1" />
          <Button
            onClick={() => {
              setDraft(initial)
              setNeverFree(data.neverFree)
            }}
            disabled={busy}
          >
            {t(ma.discard)}
          </Button>
          <Button variant="primary" loading={busy} disabledReason={invalid ? t(ma.meterRange) : undefined} onClick={() => void save()}>
            {t(ma.save)}
          </Button>
        </div>
      ) : null}
      <UnsavedChangesGuard
        dirty={dirty && canManage}
        title={t(ma.leaveTitle)}
        description={t(ma.leaveText)}
        saveLabel={t(ma.leaveSave)}
        onSave={async () => {
          if (!(await save())) throw new Error(t(ma.saveFailed))
        }}
      />
    </WebsitePage>
  )
}

type CardProps = { draft: AccessInput; set: (p: Partial<AccessInput>) => void; ro: boolean }

function SigninCard({ data, draft, set, ro, sessionBad }: CardProps & { data: Access; sessionBad: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="ac-signin">
      <div className="ws-sec">
        <h2 className="ws-h2" id="ac-signin">
          {t(ma.signinTitle)}
        </h2>
        <ButtonLink variant="ghost" size="sm" to="/w/$workspace/settings/senders" params={{ workspace: slug }} className="text-accent">
          {t(ma.senders)}
        </ButtonLink>
      </div>
      <div className="ws-setr" style={{ borderTop: 0 }}>
        <div className="tx">
          <b id="ac-signin-on">{t(ma.signinOn)}</b>
          <span className="k">{draft.signinOn ? t(ma.signinOnText) : t(ma.signinOffText)}</span>
        </div>
        <div className="ct">
          <input type="checkbox" role="switch" className="tg" aria-labelledby="ac-signin-on" checked={draft.signinOn} disabled={ro} onChange={(e) => set({ signinOn: e.target.checked })} />
        </div>
      </div>
      {!draft.signinOn && data.signinOn ? (
        <div className="px-[18px] pb-4">
          <Note tone="warn">
            <div className="flex flex-col gap-1">
              <b>{t(ma.signinOffTitle)}</b>
              <span>{t(ma.signinOffImpact, { members: data.membersOnlyPosts, paid: data.paidPosts, paidMembers: data.members.paid })}</span>
            </div>
          </Note>
        </div>
      ) : null}
      <div className="ws-sub">
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t(ma.linkMinutes)} hint={t(ma.linkHint)}>
            <Select value={String(draft.linkMinutes)} disabled={ro} onChange={(e) => set({ linkMinutes: Number(e.target.value) as AccessInput['linkMinutes'] })}>
              {[10, 15, 30, 60].map((n) => (
                <option key={n} value={n}>
                  {t(ma.minutes, { n })}
                </option>
              ))}
            </Select>
          </Field>
          <Field label={t(ma.sessionDays)} hint={t(ma.sessionHint)} error={sessionBad ? t(ma.sessionRange) : undefined}>
            <SessionInput value={draft.sessionDays} disabled={ro} onChange={(n) => set({ sessionDays: n })} />
          </Field>
        </div>
      </div>
      <dl className="ws-kv pad" style={{ borderTop: '1px solid var(--line-2)' }}>
        <dt>{t(ma.sender)}</dt>
        <dd>
          {data.senderName} &lt;{data.senderEmail}&gt; · <span className="k">{t(ma.senderDefault)}</span>
        </dd>
        <dt>{t(ma.stream)}</dt>
        <dd>
          {t(ma.streamText)}
          {data.systemProvider ? <span className="k"> · {data.systemProvider}</span> : null}
        </dd>
        <dt>{t(ma.limits)}</dt>
        <dd>{t(ma.limitsText)}</dd>
      </dl>
    </section>
  )
}

/** The days field with "days" after it; any whole number may be typed (the range is checked on save). */
function SessionInput({ value, onChange, disabled, id, ...rest }: { value: number; onChange: (n: number) => void; disabled?: boolean; id?: string; 'aria-describedby'?: string; 'aria-invalid'?: boolean }) {
  const t = useT()
  return (
    <span className="flex items-center gap-2">
      <Input id={id} className="ws-numin" style={{ width: 80 }} type="number" min={7} max={365} value={String(value)} disabled={disabled} onChange={(e) => onChange(Math.round(Number(e.target.value) || 0))} {...rest} />
      <span className="k">{t(ma.days)}</span>
    </span>
  )
}

function SignupCard({ data, draft, set, ro }: CardProps & { data: Access }) {
  const t = useT()
  const offered = (id: string) => draft.lists.find((l) => l.listId === id)
  const toggle = (id: string, consent: string, on: boolean) =>
    set({ lists: on ? [...draft.lists, { listId: id, checked: false, consent }] : draft.lists.filter((l) => l.listId !== id) })
  const check = (id: string, on: boolean) => set({ lists: draft.lists.map((l) => (l.listId === id ? { ...l, checked: on } : l)) })
  return (
    <section className="card" aria-labelledby="ac-signup">
      <div className="ws-sec">
        <h2 className="ws-h2" id="ac-signup">
          {t(ma.signupTitle)}
        </h2>
      </div>
      <div className="ws-setr" style={{ borderTop: 0 }}>
        <div className="tx">
          <b id="ac-signup-on">{t(ma.signupOn)}</b>
          <span className="k">{t(ma.signupOnText)}</span>
        </div>
        <div className="ct">
          <input type="checkbox" role="switch" className="tg" aria-labelledby="ac-signup-on" checked={draft.signupOn} disabled={ro || !draft.signinOn} onChange={(e) => set({ signupOn: e.target.checked })} />
        </div>
      </div>
      <div className="ws-sub">
        <b>{t(ma.listsOffered)}</b>
        <div className="ws-dlist">
          {data.lists.map((l) => {
            const o = offered(l.id)
            return (
              <div key={l.id} className="ws-drag" style={{ alignItems: 'flex-start' }}>
                <span className="tx">
                  <b>{l.name}</b>
                  <span className="k" style={{ whiteSpace: 'normal' }}>
                    {o ? t(ma.consentText, { text: o.consent || l.consent }) : t(ma.notOffered)}
                  </span>
                </span>
                <span className="ct" style={{ gap: 14 }}>
                  <label className="flex items-center gap-2 font-medium">
                    <input type="checkbox" aria-label={t(ma.offeredFor, { name: l.name })} checked={Boolean(o)} disabled={ro || !draft.signupOn} onChange={(e) => toggle(l.id, l.consent, e.target.checked)} />
                    <span aria-hidden="true">{t(ma.offered)}</span>
                  </label>
                  <label className="flex items-center gap-2 font-medium">
                    <input type="checkbox" aria-label={t(ma.checkedFor, { name: l.name })} checked={Boolean(o?.checked)} disabled={ro || !o || !draft.signupOn} onChange={(e) => check(l.id, e.target.checked)} />
                    <span aria-hidden="true">{t(ma.checked)}</span>
                  </label>
                </span>
              </div>
            )
          })}
        </div>
        <span className="k">{t(ma.signupNote)}</span>
      </div>
    </section>
  )
}

function VisibilityCard({ draft, set, ro, paidBlocked }: CardProps & { paidBlocked?: string }) {
  const t = useT()
  const items: { value: 'public' | 'members' | 'paid'; label: string }[] = [
    { value: 'public', label: t(ma.public) },
    { value: 'members', label: t(ma.members) },
    { value: 'paid', label: t(ma.paid) },
  ]
  return (
    <section className="card" aria-labelledby="ac-vis">
      <div className="ws-sec">
        <h2 className="ws-h2" id="ac-vis">
          {t(ma.visibilityTitle)}
        </h2>
      </div>
      <div className="ws-sub" style={{ borderTop: 0 }}>
        <div className="flex flex-col gap-0.5">
          <b>{t(ma.webOnlyTitle)}</b>
          <span className="k">{t(ma.webOnlyText)}</span>
        </div>
        <div className="seg self-start" role="group" aria-label={t(ma.webOnlyLabel)}>
          {items.map((it) => (
            <button
              key={it.value}
              type="button"
              aria-pressed={draft.defaultVisibility === it.value}
              disabled={ro || (it.value === 'paid' && Boolean(paidBlocked))}
              onClick={() => set({ defaultVisibility: it.value })}
            >
              <Icon name={visibilityIcon(it.value)} size={13} />
              {it.label}
            </button>
          ))}
        </div>
        {paidBlocked ? <span className="k">{paidBlocked}</span> : null}
      </div>
      <div className="ws-setr">
        <div className="tx">
          <b>{t(ma.fromCampaigns)}</b>
          <span className="k">{t(ma.fromCampaignsText)}</span>
        </div>
        <div className="ct" style={{ minWidth: 210 }}>
          <Select aria-label={t(ma.fromCampaignsLabel)} value={draft.campaignVisibility} disabled={ro} onChange={(e) => set({ campaignVisibility: e.target.value as AccessInput['campaignVisibility'] })}>
            <option value="audience">{t(ma.followAudience)}</option>
            <option value="public">{t(ma.public)}</option>
            <option value="members">{t(ma.members)}</option>
            {paidBlocked ? null : <option value="paid">{t(ma.paid)}</option>}
          </Select>
        </div>
      </div>
    </section>
  )
}

type NeverFree = Access['neverFree'][number]

function MeterCard({
  draft,
  set,
  ro,
  visitorsBad,
  membersBad,
  neverFree,
  onNeverFree,
}: CardProps & { visitorsBad: boolean; membersBad: boolean; neverFree: NeverFree[]; onNeverFree: (l: NeverFree[]) => void }) {
  const t = useT()
  const [picking, setPicking] = useState(false)
  const rule = (on: boolean, n: number, bad: boolean, before: string, after: string, hint: string, setOn: (v: boolean) => void, setN: (v: number) => void, id: string) => (
    <div className="ws-setr">
      <div className="tx">
        <span className="flex flex-wrap items-center gap-x-2 gap-y-1 font-semibold">
          {before}
          <input
            className={`in ws-numin${bad ? ' err' : ''}`}
            type="number"
            min={1}
            max={20}
            aria-label={t(ma.meterNumber)}
            aria-invalid={bad || undefined}
            aria-describedby={bad ? `${id}-err` : undefined}
            value={String(n)}
            disabled={ro || !on}
            onChange={(e) => setN(Math.round(Number(e.target.value) || 0))}
          />
          {after}
        </span>
        {bad ? (
          <span className="err" id={`${id}-err`} role="alert">
            {t(ma.meterRange)}
          </span>
        ) : null}
        <span className="k">{hint}</span>
      </div>
      <div className="ct">
        <input type="checkbox" role="switch" className="tg" aria-label={`${before} ${t(ma.meterRule)}`} checked={on} disabled={ro} onChange={(e) => setOn(e.target.checked)} />
      </div>
    </div>
  )
  return (
    <section className="card" aria-labelledby="ac-meter">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="ac-meter">
            {t(ma.meterTitle)}
          </h2>
          <span className="k">{t(ma.meterSub)}</span>
        </div>
      </div>
      {rule(
        draft.meterVisitorsOn,
        draft.meterVisitors,
        visitorsBad,
        t(ma.meterVisitorsBefore),
        t(ma.meterVisitorsAfter, { count: draft.meterVisitors }),
        t(ma.meterVisitorsText),
        (v) => set({ meterVisitorsOn: v, meterVisitors: v && draft.meterVisitors < 1 ? 2 : draft.meterVisitors }),
        (n) => set({ meterVisitors: n }),
        'ac-mv',
      )}
      {rule(
        draft.meterMembersOn,
        draft.meterMembers,
        membersBad,
        t(ma.meterMembersBefore),
        t(ma.meterMembersAfter, { count: draft.meterMembers }),
        t(ma.meterMembersText),
        (v) => set({ meterMembersOn: v, meterMembers: v && draft.meterMembers < 1 ? 1 : draft.meterMembers }),
        (n) => set({ meterMembers: n }),
        'ac-mm',
      )}
      <div className="ws-sub">
        <b>{t(ma.neverFree, { count: neverFree.length })}</b>
        {neverFree.length ? (
          <div className="ws-dlist">
            {neverFree.map((p) => (
              <div key={p.id} className="ws-drag">
                <span className="tx">
                  <b>{p.title}</b>
                  <span className="k">{neverLine(t, p)}</span>
                </span>
                <button
                  type="button"
                  className="btn sm ic ws-menu-ic"
                  aria-label={t(ma.removeNever, { title: p.title })}
                  disabled={ro}
                  onClick={() => onNeverFree(neverFree.filter((x) => x.id !== p.id))}
                >
                  <Icon name="close" size={13} />
                </button>
              </div>
            ))}
          </div>
        ) : null}
        {ro ? null : (
          <span>
            <Button size="sm" icon="plus" onClick={() => setPicking(true)}>
              {t(ma.addPost)}
            </Button>
          </span>
        )}
      </div>
      {picking ? (
        <NeverFreePicker
          exclude={neverFree.map((p) => p.id)}
          onClose={() => setPicking(false)}
          onPick={(p) => {
            onNeverFree([...neverFree, p])
            setPicking(false)
          }}
        />
      ) : null}
    </section>
  )
}

function neverLine(t: ReturnType<typeof useT>, p: NeverFree): string {
  const vis = p.visibility === 'paid' ? t(ma.paid) : p.visibility === 'members' ? t(ma.members) : t(ma.public)
  const when = p.status === 'scheduled' && p.publishAt ? t(ma.scheduledOn, { date: dayMonthDate(p.publishAt) }) : p.publishedAt ? t(ma.publishedOn, { date: dayMonthDate(p.publishedAt) }) : t(ma.draftWord)
  return p.tiers.length ? t(ma.neverFreeLine, { visibility: vis, tiers: p.tiers.join(', '), when }) : t(ma.neverFreeLineNoTiers, { visibility: vis, when })
}

function NeverFreePicker({ exclude, onClose, onPick }: { exclude: string[]; onClose: () => void; onPick: (p: NeverFree) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [q, setQ] = useState('')
  const [term, setTerm] = useState('')
  useEffect(() => {
    const h = window.setTimeout(() => setTerm(q), 250)
    return () => window.clearTimeout(h)
  }, [q])
  const paid = $api.useQuery('get', `${WS}/posts`, { params: { path: { workspace: slug }, query: { visibility: 'paid', q: term || undefined, pageSize: 10 } } }, { placeholderData: (p) => p })
  const members = $api.useQuery('get', `${WS}/posts`, { params: { path: { workspace: slug }, query: { visibility: 'members', q: term || undefined, pageSize: 10 } } }, { placeholderData: (p) => p })
  const rows = [...(paid.data?.posts ?? []), ...(members.data?.posts ?? [])].filter((p) => !exclude.includes(p.id) && p.status !== 'unpublished')
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={t(ma.pickTitle)}>
      <div className="flex flex-col gap-3">
        <label className="ws-search" style={{ width: '100%' }}>
          <Icon name="search" size={14} />
          <input type="search" aria-label={t(ma.pickSearch)} placeholder={t(ma.pickSearch)} value={q} onChange={(e) => setQ(e.target.value)} autoFocus />
        </label>
        {paid.isPending || members.isPending ? (
          <Skeleton w="100%" h={160} r={8} />
        ) : rows.length === 0 ? (
          <span className="k">{t(ma.pickNone)}</span>
        ) : (
          <div className="ws-dlist">
            {rows.map((p) => (
              <div key={p.id} className="ws-drag">
                <span className="tx">
                  <b>{p.title}</b>
                  <span className="k">{p.visibility === 'paid' ? t(ma.paid) : t(ma.members)}</span>
                </span>
                <Button
                  size="sm"
                  onClick={() =>
                    onPick({ id: p.id, title: p.title, visibility: p.visibility, tiers: p.tiers.map((x) => x.name), status: p.status, publishAt: p.publishAt, publishedAt: p.publishedAt })
                  }
                >
                  {t(ma.pickAdd)}
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </Dialog>
  )
}

function money(cents: number, currency: string): string {
  const v = cents / 100
  try {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: currency || 'USD', minimumFractionDigits: Number.isInteger(v) ? 0 : 2, maximumFractionDigits: 2 }).format(v)
  } catch {
    return `${v}`
  }
}

function tierPrice(t: ReturnType<typeof useT>, x: Tier): string {
  if (x.monthly && x.yearly) return t(ma.tierLine, { monthly: money(x.monthly, x.currency), yearly: money(x.yearly, x.currency) })
  if (x.yearly) return t(ma.tierLineYearly, { yearly: money(x.yearly, x.currency) })
  return t(ma.tierLineMonthly, { monthly: money(x.monthly, x.currency) })
}

function PaywallCard({ data, draft, set, ro }: CardProps & { data: Access }) {
  const t = useT()
  const [as, setAs] = useState<'visitor' | 'free' | 'used'>('visitor')
  const pw = draft.paywall
  const setPw = (patch: Partial<AccessInput['paywall']>) => set({ paywall: { ...pw, ...patch } })
  const shown = (id: string) => (pw.tiers.length ? pw.tiers.includes(id) : true)
  const toggleTier = (id: string, on: boolean) => {
    const current = data.tiers.filter((x) => shown(x.id)).map((x) => x.id)
    setPw({ tiers: on ? [...current, id] : current.filter((x) => x !== id) })
  }
  const tiersOK = data.payments.connected && data.tiers.length > 0
  const popular = data.tiers.find((x) => x.public)?.id
  return (
    <section className="card" aria-labelledby="ac-pw">
      <div className="ws-sec">
        <div className="tx">
          <h2 className="ws-h2" id="ac-pw">
            {t(ma.paywallTitle)}
          </h2>
          <span className="k">{tiersOK ? t(ma.paywallSub) : t(ma.stillWork)}</span>
        </div>
      </div>
      <div className="ws-sub" style={{ borderTop: 0 }}>
        <Field label={t(ma.heading)}>
          <Input value={pw.heading} maxLength={120} disabled={ro} onChange={(e) => setPw({ heading: e.target.value })} />
        </Field>
        <Field label={t(ma.text)}>
          <Textarea value={pw.text} maxLength={400} rows={2} disabled={ro} onChange={(e) => setPw({ text: e.target.value })} />
        </Field>
        <div className="flex flex-col gap-2">
          <span className="lbl">
            <span className="font-semibold">{t(ma.tiersToShow)}</span>
          </span>
          {tiersOK ? (
            data.tiers.map((x) => (
              <label key={x.id} className="ws-lcheck">
                <input type="checkbox" checked={shown(x.id)} disabled={ro} onChange={(e) => toggleTier(x.id, e.target.checked)} />
                <span className="tx">
                  {x.name}
                  <span className="k">
                    {[tierPrice(t, x), x.trialDays ? t(ma.trial, { days: x.trialDays }) : '', x.includes ? t(ma.includes, { tier: x.includes }) : '', x.id === popular && data.tiers.length > 1 ? t(ma.mostPopular) : '']
                      .filter(Boolean)
                      .join(' · ')}
                  </span>
                </span>
              </label>
            ))
          ) : (
            <Note tone="neutral">
              <div className="flex flex-col gap-1">
                <b>{t(ma.tiersUnavailable)}</b>
                <span>{data.payments.connected ? t(ma.noTiersYet) : t(ma.tiersNeedStripe)}</span>
              </div>
            </Note>
          )}
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label={t(ma.upgradeButton)}>
            <Input value={pw.upgrade} maxLength={40} disabled={ro} onChange={(e) => setPw({ upgrade: e.target.value })} />
          </Field>
          <Field label={t(ma.signupButton)}>
            <Input value={pw.signup} maxLength={40} disabled={ro} onChange={(e) => setPw({ signup: e.target.value })} />
          </Field>
        </div>
        <Field label={t(ma.freeRow)} hint={t(ma.freeRowHint)}>
          <Input value={pw.free} maxLength={200} disabled={ro} onChange={(e) => setPw({ free: e.target.value })} />
        </Field>
      </div>
      <div className="ws-sub">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <b>{t(ma.previewAs)}</b>
          <Segmented
            value={as}
            onChange={setAs}
            label={t(ma.previewAs)}
            items={[
              { value: 'visitor', label: t(ma.asVisitor) },
              { value: 'free', label: t(ma.asFree) },
              { value: 'used', label: t(ma.asFreeUsed) },
            ]}
          />
        </div>
        <div className="ws-pw-pane">
          <PaywallPreview data={data} draft={draft} as={as} tiersOK={tiersOK} />
        </div>
      </div>
    </section>
  )
}

/** The paywall card as the site draws it, from the unsaved settings (inert). */
function PaywallPreview({ data, draft, as, tiersOK }: { data: Access; draft: AccessInput; as: 'visitor' | 'free' | 'used'; tiersOK: boolean }) {
  const t = useT()
  const pw = draft.paywall
  const tiers = tiersOK ? data.tiers.filter((x) => (pw.tiers.length ? pw.tiers.includes(x.id) : x.public)) : []
  const now = new Date()
  const next = new Date(now.getFullYear(), now.getMonth() + 1, 1)
  const meterRow = draft.meterMembersOn && draft.meterMembers > 0
  const signupRow = draft.signupOn && draft.signinOn
  const offered = data.lists.filter((l) => draft.lists.some((x) => x.listId === l.id))
  return (
    <div className="ws-pw" role="img" aria-label={t(ma.previewLabel)}>
      <h3>{pw.heading || t(ma.required)}</h3>
      <p className="tx">{as === 'used' ? t(ma.usedUp, { month: monthName(now, 'long'), next: `${next.getDate()} ${monthName(next, 'long')}` }) : pw.text}</p>
      {tiers.map((x, i) => (
        <div key={x.id} className={`tier${i === 0 ? ' hi' : ''}`}>
          <span className="flex flex-col gap-0.5">
            <b>{x.name}</b>
            <span className="sm">
              {[tierPrice(t, x), x.trialDays ? t(ma.trial, { days: x.trialDays }) : '', x.includes ? t(ma.includes, { tier: x.includes }) : ''].filter(Boolean).join(' · ')}
            </span>
          </span>
          <span className="up">{pw.upgrade || t(ma.required)}</span>
        </div>
      ))}
      {as === 'visitor' && signupRow ? (
        <div className="free">
          <b>{meterRow ? pw.free : pw.heading}</b>
          <div className="frm">
            <span>{t(ma.emailPlaceholder)}</span>
            <span className="up" style={{ flex: 'none', color: '#0B1220' }}>
              {pw.signup}
            </span>
          </div>
          {offered.map((l) => {
            const x = draft.lists.find((y) => y.listId === l.id)
            return (
              <span key={l.id} className="cb">
                <input type="checkbox" checked={Boolean(x?.checked)} readOnly tabIndex={-1} aria-hidden="true" />
                {x?.consent || l.consent}
              </span>
            )
          })}
          <span className="sm k" style={{ fontSize: 11 }}>
            {t(ma.confirmNote)}
          </span>
        </div>
      ) : null}
      <div className="ft">
        {as === 'visitor' ? (
          <span>
            {t(ma.alreadyMember)} <u>{t(ma.signIn)}</u>
          </span>
        ) : (
          <span>{as === 'used' ? t(ma.freeMemberNote) : ''}</span>
        )}
        {as === 'visitor' ? <span>{t(ma.paymentsNote, { publisher: data.payments.account || data.senderName })}</span> : <u>{t(ma.yourAccount)}</u>}
      </div>
    </div>
  )
}

function MembersCard({ data, overview }: { data: Access; overview: Overview }) {
  const t = useT()
  const { slug } = useWorkspace()
  const since = overview.turnedOnAt ?? data.since
  return (
    <section className="card" aria-labelledby="ac-members">
      <div className="ws-sec">
        <h2 className="ws-h2" id="ac-members">
          {t(ma.membersTitle)}
        </h2>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="ws-stats" style={{ gridTemplateColumns: 'repeat(auto-fit,minmax(150px,1fr))' }}>
          <div className="card ws-stat">
            <span className="k">{t(ma.signedInSince, { date: since ? dayMonthDate(since) : '' })}</span>
            <span className="v">{count(data.members.signedIn)}</span>
            <span className="s">{t(ma.signedInSinceText)}</span>
          </div>
          <div className="card ws-stat">
            <span className="k">{t(ma.lastWeek)}</span>
            <span className="v">{count(data.members.lastWeek)}</span>
            <span className="s">{t(ma.lastWeekText)}</span>
          </div>
          <div className="card ws-stat">
            <span className="k">{t(ma.paidMembers)}</span>
            <span className="v">{count(data.members.paid)}</span>
            <span className="s">{t(ma.paidMembersText)}</span>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <ButtonLink size="sm" icon="audience" to="/w/$workspace/audience" params={{ workspace: slug }} search={{ signedIn: 'ever' } as never}>
            {t(ma.showSignedIn)}
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/earn/members" params={{ workspace: slug }}>
            {t(ma.earnMembers)}
          </ButtonLink>
        </div>
      </div>
    </section>
  )
}

/** "Sign-in is turned off" impact, for the Website tab's other views. */
export function SigninOffNote({ data }: { data: Access }) {
  return <T msg={ma.signinOffImpact} values={{ members: data.membersOnlyPosts, paid: data.paidPosts, paidMembers: data.members.paid }} />
}
