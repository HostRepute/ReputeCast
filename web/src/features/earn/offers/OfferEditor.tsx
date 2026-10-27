import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Checkbox, Field, Input, OptionCard, Select, Toggle } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useI18n, useT } from '@/i18n'
import { count, dayMonthDate, dayMonthYear, getDisplayTimeZone, orList, pct } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { copyText, EARN, ProviderChip, useEarnStatus } from '../EarnPage'
import { money } from '../format'
import { EndOfferDialog } from './EndOfferDialog'
import { discounted, discountLine, forLine, howLine, offOf, tiersLine } from './lines'
import { mof } from './messages'
import './offers.css'

type Detail = Schemas['OfferDetail']
type Offer = Schemas['Offer']
type Applies = Schemas['OfferApplies']
type Kind = 'signup' | 'retention' | 'referral'
type TT = ReturnType<typeof useT>

const bold = (c: ReactNode) => <b className="font-semibold">{c}</b>
const key = (a: { tierId: string; period: string }) => `${a.tierId}|${a.period}`

/** The editor's fields, as typed. */
type Form = {
  name: string
  kind: Kind
  discountType: 'percent' | 'amount'
  percentOff: string
  amountOff: string
  duration: 'once' | 'months' | 'forever'
  months: string
  applies: string[]
  byCode: boolean
  code: string
  byLink: boolean
  limit: string
  endsOn: string
  newOnly: boolean
  minPaidMonths: string
  codeDays: string
}

/** "2026-10-31" in the workspace's time zone. */
function isoDay(at: string): string {
  const tz = getDisplayTimeZone()
  const p = new Intl.DateTimeFormat('en-CA', { timeZone: tz || undefined, year: 'numeric', month: '2-digit', day: '2-digit' }).formatToParts(new Date(at))
  const get = (t: string) => p.find((x) => x.type === t)?.value ?? ''
  return `${get('year')}-${get('month')}-${get('day')}`
}

function toForm(d: Detail): Form {
  return {
    name: d.name,
    kind: d.kind,
    discountType: d.discountType,
    percentOff: d.percentOff ? String(d.percentOff) : d.id ? '' : '20',
    amountOff: d.amountOffCents ? String(d.amountOffCents / 100) : '',
    duration: d.duration,
    months: d.months ? String(d.months) : '3',
    applies: d.applies.map(key),
    byCode: d.id ? d.byCode : true,
    code: d.code,
    byLink: d.id ? d.byLink : true,
    limit: d.limit ? String(d.limit) : '',
    endsOn: d.endsAt ? isoDay(d.endsAt) : '',
    newOnly: d.id ? d.newOnly : true,
    minPaidMonths: String(d.minPaidMonths ?? 0),
    codeDays: String(d.codeDays || 60),
  }
}

/** The form as an offer, for the lines and the preview. */
function draft(f: Form, d: Detail): Offer {
  const applies = d.choices.filter((c) => f.applies.includes(key(c)))
  return {
    ...d,
    name: f.name.trim() || d.name,
    kind: f.kind,
    discountType: f.discountType,
    percentOff: Number(f.percentOff) || 0,
    amountOffCents: Math.round(Number(f.amountOff) * 100) || 0,
    duration: f.duration,
    months: Number(f.months) || 0,
    applies,
    byCode: f.byCode,
    code: f.code.trim().toUpperCase(),
    byLink: f.byLink,
    newOnly: f.newOnly,
    minPaidMonths: Number(f.minPaidMonths) || 0,
    codeDays: Number(f.codeDays) || 60,
    linkHost: `${d.upgradeHost}?offer=${f.code.trim().toUpperCase()}`,
  }
}

/** The offer editor (EarnOfferEditor): /earn/plans/offers/new and /earn/plans/offers/{id}. */
export function OfferEditorPage({ offerId }: { offerId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const path = { params: { path: { workspace: slug, id: offerId ?? '' } } }
  const existing = $api.useQuery('get', `${EARN}/offers/{id}`, path, { enabled: Boolean(offerId) })
  const fresh = $api.useQuery('get', `${EARN}/offers/new`, { params: { path: { workspace: slug } } }, { enabled: !offerId })
  const q = offerId ? existing : fresh
  const status = useEarnStatus()
  if (q.error || !q.data) {
    const crumbs = [
      { label: t(mof.crumbEarn), to: '/w/$workspace/earn' as const, params: { workspace: slug } },
      { label: t(mof.crumbPlans), to: '/w/$workspace/earn/plans' as const, params: { workspace: slug } },
      { label: '…' },
    ]
    return (
      <Page title={offerId ? '…' : t(mof.newTitle)} breadcrumb={crumbs} badges={<ProviderChip status={status.data} />}>
        {q.error ? (
          <ErrorPanel title={t(mof.errorLoad)} actions={<Button onClick={() => void q.refetch()}>{t(mof.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : (
          <div className="card pad" aria-busy="true">
            <Skeleton w="100%" h={300} />
          </div>
        )}
      </Page>
    )
  }
  return <OfferForm key={q.data.id || 'new'} d={q.data} />
}

function OfferForm({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const status = useEarnStatus()
  const plans = $api.useQuery('get', `${EARN}/plans`, { params: { path: { workspace: slug } } })
  const [f, setF] = useState<Form>(() => toForm(d))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [ending, setEnding] = useState(false)
  const create = $api.useMutation('post', `${EARN}/offers`)
  const update = $api.useMutation('put', `${EARN}/offers/{id}`)
  const recreate = $api.useMutation('post', `${EARN}/offers/{id}/recreate`)
  const isNew = !d.id
  const ro = !d.canEdit
  const provider = d.provider || 'Stripe'
  const o = draft(f, d)
  const set = <K extends keyof Form>(k: K, v: Form[K]) => setF((p) => ({ ...p, [k]: v }))
  const title = isNew ? t(mof.newTitle) : d.name
  const crumbs = [
    { label: t(mof.crumbEarn), to: '/w/$workspace/earn' as const, params: { workspace: slug } },
    { label: t(mof.crumbPlans), to: '/w/$workspace/earn/plans' as const, params: { workspace: slug } },
    { label: title },
  ]
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/offers`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/offers/{id}`] })
  }
  const body = () => ({
    name: f.name.trim(),
    kind: f.kind,
    discountType: f.discountType,
    percentOff: f.discountType === 'percent' ? Number(f.percentOff) || 0 : undefined,
    amountOffCents: f.discountType === 'amount' ? Math.round(Number(f.amountOff) * 100) || 0 : undefined,
    duration: f.duration,
    months: f.duration === 'months' ? Number(f.months) || 0 : undefined,
    applies: d.choices.filter((c) => f.applies.includes(key(c))).map((c) => ({ tierId: c.tierId, period: c.period })),
    byCode: f.byCode,
    code: f.code.trim() || undefined,
    byLink: f.byLink,
    limit: f.limit.trim() ? Number(f.limit) : undefined,
    endsOn: f.endsOn || undefined,
    newOnly: f.newOnly,
    minPaidMonths: Number(f.minPaidMonths) || 0,
    codeDays: Number(f.codeDays) || undefined,
  })
  const onError = (e: unknown) => {
    if (e instanceof ApiError && e.problem.errors?.length) {
      const next: Record<string, string> = {}
      for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '').replace(/\[\d+\]$/, '')] = x.message ?? ''
      setErrors(next)
      return
    }
    toast({ kind: 'error', title: t(mof.save), detail: errorMessage(e) })
  }
  const onSaved = (r: Detail) => {
    refresh()
    setErrors({})
    if (r.sync === 'failed') toast({ kind: 'error', title: t(mof.saved, { name: r.name }), detail: t(mof.syncFailed, { provider, error: r.syncError ?? '' }) })
    else toast({ kind: 'success', title: r.sync === 'synced' ? t(mof.savedStripe, { name: r.name, provider }) : t(mof.saved, { name: r.name }) })
    if (isNew) void navigate({ to: '/w/$workspace/earn/plans/offers/$offerId', params: { workspace: slug, offerId: r.id }, replace: true })
  }
  const save = () => {
    setErrors({})
    if (isNew) create.mutate({ params: { path: { workspace: slug } }, body: body() }, { onSuccess: onSaved, onError })
    else update.mutate({ params: { path: { workspace: slug, id: d.id } }, body: body() }, { onSuccess: onSaved, onError })
  }
  const actions = ro ? null : (
    <>
      {!isNew && d.status === 'live' ? (
        <Button variant="danger" onClick={() => setEnding(true)}>
          {t(mof.end)}
        </Button>
      ) : null}
      <Button variant="primary" loading={create.isPending || update.isPending} onClick={save}>
        {isNew ? t(mof.create) : t(mof.save)}
      </Button>
    </>
  )
  const benefits = plans.data?.tiers.find((x) => x.id === o.applies[0]?.tierId)?.benefits ?? []
  return (
    <Page title={title} breadcrumb={crumbs} badges={<ProviderChip status={status.data} />} actions={actions}>
      <div className="flex flex-col gap-4">
        {ro ? <ReadOnlyNote helpers={d.helpers} /> : null}
        {d.state === 'missing' && !ro ? (
          <div className="note b" role="alert">
            <Icon name="alertCircle" size={16} />
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
              <span className="font-bold">{t(mof.missingTitle, { name: d.name, provider })}</span>
              <span>{t(mof.missingText, { name: d.name, provider })}</span>
              <div className="mt-1 flex flex-wrap gap-2">
                <Button
                  size="sm"
                  variant="primary"
                  loading={recreate.isPending}
                  onClick={() => recreate.mutate({ params: { path: { workspace: slug, id: d.id } } }, { onSuccess: onSaved, onError })}
                >
                  {t(mof.recreate)}
                </Button>
                <Button size="sm" onClick={() => setEnding(true)}>
                  {t(mof.endOffer)}
                </Button>
              </div>
            </div>
          </div>
        ) : null}
        {d.sync === 'failed' && !ro && d.state !== 'missing' ? <Note tone="bad">{t(mof.syncFailed, { provider, error: d.syncError ?? '' })}</Note> : null}
        <div className="of-ed">
          <div className="col">
            {ro ? (
              <ReadOnlyCard d={d} />
            ) : (
              <Fields f={f} d={d} o={o} set={set} errors={errors} isNew={isNew} provider={provider} t={t} />
            )}
          </div>
          <div className="col">
            {o.kind === 'signup' ? <UpgradePreview o={o} d={d} benefits={benefits} /> : null}
            {o.kind === 'retention' ? <CancelPreview o={o} host={d.upgradeHost.replace(/\/upgrade$/, '/cancel')} /> : null}
            {isNew ? null : <Results d={d} />}
          </div>
        </div>
        {!isNew && d.kind === 'referral' ? <Codes d={d} /> : null}
      </div>
      {ending ? <EndOfferDialog offer={d} provider={provider} onClose={() => setEnding(false)} /> : null}
    </Page>
  )
}

function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  return (
    <div className="note n" role="note">
      <span style={{ color: 'var(--ink-2)', display: 'inline-flex', flex: 'none', marginTop: 1 }}>
        <Icon name="info" size={16} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(mof.readOnlyTitle, { name: user?.name || user?.email || '', role })}</span>
        <div>
          <T msg={helpers.length ? mof.readOnly : mof.readOnlyNoHelpers} values={{ helpers: orList(helpers) }} />
        </div>
      </div>
    </div>
  )
}

function Card({ title, aside, children }: { title: ReactNode; aside?: ReactNode; children: ReactNode }) {
  return (
    <section className="card">
      <div className="of-sec flex flex-wrap items-center justify-between gap-2">
        <h2 className="h2">{title}</h2>
        {aside}
      </div>
      <div className="of-body">{children}</div>
    </section>
  )
}

function Fields({
  f,
  d,
  o,
  set,
  errors,
  isNew,
  provider,
  t,
}: {
  f: Form
  d: Detail
  o: Offer
  set: <K extends keyof Form>(k: K, v: Form[K]) => void
  errors: Record<string, string>
  isNew: boolean
  provider: string
  t: TT
}) {
  const { locale } = useI18n()
  const zone = (getDisplayTimeZone() ?? 'UTC').split('/').pop()?.replaceAll('_', ' ') ?? 'UTC'
  const choices = d.choices
  const toggle = (k: string, on: boolean) => set('applies', on ? [...f.applies, k] : f.applies.filter((x) => x !== k))
  return (
    <>
      <Card title={t(mof.cardOffer)}>
        <Field label={t(mof.name)} error={errors.name}>
          <Input value={f.name} maxLength={80} onChange={(e) => set('name', e.target.value)} />
        </Field>
        <div className="flex flex-col gap-1.5">
          <span className="text-[12px] font-semibold text-ink-2">{t(mof.kindLabel)}</span>
          {isNew ? (
            <Segmented<Kind>
              value={f.kind}
              onChange={(k) => set('kind', k)}
              label={t(mof.kindLabel)}
              className="self-start"
              items={[
                { value: 'signup', label: t(mof.kSignup) },
                { value: 'retention', label: t(mof.kRetention) },
                { value: 'referral', label: t(mof.kReferral) },
              ]}
            />
          ) : (
            <span>
              <Badge tone="neutral">{t(mof.kind, { kind: f.kind })}</Badge>
            </span>
          )}
          <span className="k">{t(mof.kindHint)}</span>
        </div>
      </Card>
      <Card title={t(mof.cardDiscount)}>
        <div className="of-inline">
          <Segmented<'percent' | 'amount'>
            value={f.discountType}
            onChange={(v) => set('discountType', v)}
            label={t(mof.cardDiscount)}
            items={[
              { value: 'percent', label: t(mof.percentage) },
              { value: 'amount', label: t(mof.fixed) },
            ]}
          />
          {f.discountType === 'percent' ? (
            <>
              <Input
                className="w-[80px]"
                inputMode="numeric"
                aria-label={t(mof.percentage)}
                invalid={Boolean(errors.percentOff)}
                value={f.percentOff}
                onChange={(e) => set('percentOff', e.target.value)}
              />
              <span className="k">{t(mof.pctOff)}</span>
            </>
          ) : (
            <>
              <Input
                className="w-[100px]"
                inputMode="decimal"
                aria-label={t(mof.fixed)}
                invalid={Boolean(errors.amountOffCents)}
                value={f.amountOff}
                onChange={(e) => set('amountOff', e.target.value)}
              />
              <span className="k">{t(mof.amountOff)}</span>
            </>
          )}
        </div>
        {errors.percentOff || errors.amountOffCents ? (
          <span className="err" role="alert">
            {errors.percentOff || errors.amountOffCents}
          </span>
        ) : null}
        {locale === 'en' ? (
          <Field label={t(mof.durationLabel)} error={errors.months}>
            <Select value={f.duration} onChange={(e) => set('duration', e.target.value as Form['duration'])}>
              <option value="once">{t(mof.durOnce)}</option>
              <option value="months">{t(mof.durMonths)}</option>
              <option value="forever">{t(mof.durForever)}</option>
            </Select>
          </Field>
        ) : (
          // Outside English a choice that can pass about 32 characters is a
          // set of radio cards, not a select (CONVENTIONS › Text expansion, rule 1).
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            <legend className="mb-1.5 p-0 text-[12px] font-semibold text-ink-2">{t(mof.durationLabel)}</legend>
            {(['once', 'months', 'forever'] as const).map((v) => (
              <OptionCard
                key={v}
                name="of-duration"
                value={v}
                checked={f.duration === v}
                onChange={() => set('duration', v)}
                title={t(v === 'once' ? mof.durOnce : v === 'months' ? mof.durMonths : mof.durForever)}
              />
            ))}
            {errors.months ? (
              <span className="err" role="alert">
                {errors.months}
              </span>
            ) : null}
          </fieldset>
        )}
        {f.duration === 'months' ? (
          <div className="of-inline">
            <Input className="w-[80px]" inputMode="numeric" aria-label={t(mof.monthsLabel)} value={f.months} onChange={(e) => set('months', e.target.value)} />
            <span className="k">{t(mof.months)}</span>
          </div>
        ) : null}
        {o.applies.length ? (
          <span className="k">
            <T msg={mof.readersSee} values={{ line: discountLine(t, o), b: bold }} />
          </span>
        ) : null}
      </Card>
      <Card title={t(mof.cardWho)}>
        <div className="of-checks" role="group" aria-label={t(mof.cardWho)}>
          {choices.map((c) => (
            <Checkbox
              key={key(c)}
              label={t(mof.choice, { tier: c.tier, period: c.period })}
              checked={f.applies.includes(key(c))}
              onChange={(e) => toggle(key(c), e.target.checked)}
            />
          ))}
          {errors.applies ? (
            <span className="err" role="alert">
              {errors.applies}
            </span>
          ) : null}
        </div>
        {f.kind === 'signup' ? (
          <div className="of-sep">
            <Toggle label={t(mof.newOnly)} description={t(mof.newOnlyHint)} checked={f.newOnly} onChange={(e) => set('newOnly', e.target.checked)} />
          </div>
        ) : null}
      </Card>
      {f.kind === 'signup' ? (
        <Card title={t(mof.cardHow)}>
          <div className="of-how">
            <label>
              <input type="checkbox" checked={f.byCode} onChange={(e) => set('byCode', e.target.checked)} />
              {t(mof.code)}
            </label>
            <Input
              className="w-[160px] uppercase"
              aria-label={t(mof.code)}
              invalid={Boolean(errors.code)}
              value={f.code}
              onChange={(e) => set('code', e.target.value.toUpperCase())}
            />
            <span className="k">{t(mof.codeHint)}</span>
          </div>
          {errors.code || errors.byCode ? (
            <span className="err" role="alert">
              {errors.code || errors.byCode}
            </span>
          ) : null}
          <div className="of-how">
            <label>
              <input type="checkbox" checked={f.byLink} onChange={(e) => set('byLink', e.target.checked)} />
              {t(mof.link)}
            </label>
            <span className="mono text-[13px]">{o.linkHost}</span>
            {d.link ? (
              <Button size="sm" icon="copy" onClick={() => void copyText(d.link ?? '', t, toast)}>
                {t(mof.copy)}
              </Button>
            ) : null}
          </div>
        </Card>
      ) : null}
      {f.kind === 'retention' ? (
        <Card title={t(mof.cardEligibility)}>
          <div className="of-inline">
            <span className="text-[13.5px] font-semibold">{t(mof.paidAtLeast)}</span>
            <Input
              className="w-[80px]"
              inputMode="numeric"
              aria-label={t(mof.paidAtLeast)}
              invalid={Boolean(errors.minPaidMonths)}
              value={f.minPaidMonths}
              onChange={(e) => set('minPaidMonths', e.target.value)}
            />
            <span className="k">{t(mof.months)}</span>
          </div>
          <ul className="m-0 flex list-none flex-col gap-2 p-0 text-[13.5px]">
            {[t(mof.ruleGap), t(mof.rulePastDue), t(mof.ruleOne)].map((r) => (
              <li key={r} className="flex items-start gap-2">
                <span style={{ color: 'var(--ok-fg)', display: 'inline-flex', marginTop: 2 }}>
                  <Icon name="check" size={14} />
                </span>
                {r}
              </li>
            ))}
          </ul>
        </Card>
      ) : null}
      {f.kind === 'referral' ? (
        <Card title={t(mof.cardLimits)}>
          <div className="of-inline">
            <span className="text-[13.5px] font-semibold">{t(mof.codeDays)}</span>
            <Input
              className="w-[80px]"
              inputMode="numeric"
              aria-label={t(mof.codeDays)}
              invalid={Boolean(errors.codeDays)}
              value={f.codeDays}
              onChange={(e) => set('codeDays', e.target.value)}
            />
            <span className="k">{t(mof.days)}</span>
          </div>
        </Card>
      ) : null}
      {f.kind === 'signup' ? (
        <Card title={t(mof.cardLimits)}>
          <div className="of-limits">
            <Field label={t(mof.limit)} hint={t(mof.optional)} error={errors.limit}>
              <Input inputMode="numeric" value={f.limit} onChange={(e) => set('limit', e.target.value)} />
            </Field>
            <Field label={t(mof.ends)} hint={t(mof.endsHint, { zone })} error={errors.endsOn}>
              <Input type="date" value={f.endsOn} onChange={(e) => set('endsOn', e.target.value)} />
            </Field>
          </div>
          <span className="k">{t(mof.limitsNote)}</span>
        </Card>
      ) : null}
      <Card
        title={t(mof.cardStripe, { provider })}
        aside={
          d.stripe.couponUrl ? (
            <ButtonAnchor size="sm" href={d.stripe.couponUrl} iconRight="external">
              {t(mof.openInStripe, { provider })}
            </ButtonAnchor>
          ) : null
        }
      >
        {d.stripe.coupon ? (
          <dl className="kv">
            <dt>{t(mof.coupon)}</dt>
            <dd>
              {t(mof.couponLine, { off: offOf(t, d), duration: d.duration, months: d.months ?? 0 })} <span className="mono k">{d.stripe.coupon}</span>
            </dd>
            {d.stripe.promotionCode ? (
              <>
                <dt>{t(mof.promo)}</dt>
                <dd>
                  {d.code} <span className="mono k">{d.stripe.promotionCode}</span>
                </dd>
              </>
            ) : null}
          </dl>
        ) : null}
        <span className="k">{!d.connected ? t(mof.stripeDraft) : d.stripe.coupon ? t(mof.stripeNote) : t(mof.stripeNotYet, { provider })}</span>
      </Card>
    </>
  )
}

/** Read-only · Sofia Rossi: the offer as lines. */
function ReadOnlyCard({ d }: { d: Detail }) {
  const t = useT()
  const limits =
    d.limit && d.endsAt
      ? t(mof.limitsLine, { limit: count(d.limit), date: dayMonthYear(d.endsAt) })
      : d.limit
        ? t(mof.limitsLimit, { limit: count(d.limit) })
        : d.endsAt
          ? t(mof.limitsEnds, { date: dayMonthYear(d.endsAt) })
          : t(mof.limitsNone)
  return (
    <Card title={t(mof.offerTitle, { name: d.name })} aside={<Badge tone="neutral">{t(mof.readOnlyBadge)}</Badge>}>
      <dl className="kv">
        <dt>{t(mof.kvKind)}</dt>
        <dd>{t(mof.kind, { kind: d.kind })}</dd>
        <dt>{t(mof.kvDiscount)}</dt>
        <dd>{discountLine(t, d)}</dd>
        <dt>{t(mof.kvFor)}</dt>
        <dd>{forLine(t, d)}</dd>
        <dt>{t(mof.kvHow)}</dt>
        <dd>{howLine(t, d)}</dd>
        {d.kind === 'signup' ? (
          <>
            <dt>{t(mof.kvLimits)}</dt>
            <dd>{limits}</dd>
          </>
        ) : null}
      </dl>
    </Card>
  )
}

/** How it reads on the upgrade page, with the offer link. */
function UpgradePreview({ o, d, benefits }: { o: Offer; d: Detail; benefits: string[] }) {
  const t = useT()
  const a: Applies | undefined = o.applies[0]
  return (
    <section className="card">
      <div className="of-sec flex flex-col gap-0.5">
        <h2 className="h2">{t(mof.previewTitle)}</h2>
        {a ? <span className="k">{t(mof.previewSub, { period: a.period })}</span> : null}
      </div>
      {a ? (
        <div className="of-pv">
          <span className="of-pv-url">{t(mof.previewUrl, { link: o.linkHost ?? d.upgradeHost, period: a.period })}</span>
          <PreviewBody o={o} a={a} benefits={benefits} />
        </div>
      ) : (
        <p className="of-body k m-0">{t(mof.previewEmpty)}</p>
      )}
    </section>
  )
}

function PreviewBody({ o, a, benefits }: { o: Offer; a: Applies; benefits: string[] }) {
  const t = useT()
  const cur = a.currency || o.currency || 'usd'
  const price = money(discounted(a.priceCents, o), cur)
  const full = money(a.priceCents, cur)
  const v = { price, full, period: a.period, months: o.months ?? 0 }
  const now = new Date()
  const renews = new Date(now)
  if (o.duration === 'months') renews.setMonth(renews.getMonth() + (o.months ?? 0))
  else if (a.period === 'year') renews.setFullYear(renews.getFullYear() + 1)
  else renews.setMonth(renews.getMonth() + 1)
  const kind = o.duration === 'months' && a.period === 'month' ? 'months' : o.duration === 'forever' ? 'forever' : 'once'
  const line = kind === 'months' ? t(mof.previewLineMonths, v) : kind === 'forever' ? t(mof.previewLineForever, v) : t(mof.previewLineOnce, v)
  const unit = kind === 'months' ? t(mof.previewUnitMonths, v) : kind === 'forever' ? t(mof.previewUnitForever, v) : t(mof.previewUnitOnce, v)
  const sub = kind === 'months' ? t(mof.previewSubMonths, v) : kind === 'forever' ? t(mof.previewSubForever, v) : t(mof.previewSubOnce, v)
  const note =
    kind === 'months'
      ? t(mof.previewNoteMonths, { ...v, date: dayMonthYear(renews) })
      : kind === 'forever'
        ? t(mof.previewNoteForever, v)
        : t(mof.previewNote, { ...v, date: dayMonthYear(renews) })
  return (
    <>
      <div className="of-pv-banner">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
          <path d="M8.6 1.8H13a1.2 1.2 0 0 1 1.2 1.2v4.4L7.6 14a1.2 1.2 0 0 1-1.7 0L2 10.1a1.2 1.2 0 0 1 0-1.7z" />
          <circle cx="10.8" cy="5.2" r="1" />
        </svg>
        <span>
          <T msg={mof.previewBanner} values={{ name: o.code || o.name, line, b: bold }} />
        </span>
      </div>
      <div className="of-pv-card">
        <span className="of-pv-pill">{t(mof.previewApplied, { name: o.code || o.name })}</span>
        <b className="text-[16px]">{a.tier}</b>
        <div className="flex flex-col gap-0.5">
          <span className="flex flex-wrap items-baseline gap-1">
            <span className="of-pv-amt">{price}</span>
            <span className="k">{unit}</span>
          </span>
          <span className="k">{sub}</span>
        </div>
        {benefits.length ? (
          <ul className="of-pv-ben">
            {benefits.map((b) => (
              <li key={b}>
                <Icon name="check" size={14} />
                {b}
              </li>
            ))}
          </ul>
        ) : null}
        <span className="of-pv-btn">{t(mof.previewButton, { tier: a.short, price })}</span>
        <span className="k text-[12.5px]">{note}</span>
      </div>
    </>
  )
}

/** The cancel page with the retention offer. */
function CancelPreview({ o, host }: { o: Offer; host: string }) {
  const t = useT()
  const a: Applies | undefined = o.applies[0]
  if (!a) return null
  const cur = a.currency || o.currency || 'usd'
  const v = { price: money(discounted(a.priceCents, o), cur), full: money(a.priceCents, cur), period: a.period, months: o.months ?? 0, tier: a.short }
  const offer = o.duration === 'months' ? t(mof.cancelOffer, v) : o.duration === 'forever' ? t(mof.cancelOfferForever, v) : t(mof.cancelOfferOnce, v)
  return (
    <section className="card">
      <div className="of-sec flex flex-col gap-0.5">
        <h2 className="h2">{t(mof.cancelPreviewTitle)}</h2>
        <span className="k">{t(mof.cancelPreviewSub)}</span>
      </div>
      <div className="of-pv">
        <span className="of-pv-url">{t(mof.cancelPreviewUrl, { host })}</span>
        <b className="text-[19px]">{t(mof.cancelHeading, v)}</b>
        <span className="k">{t(mof.cancelLead, v)}</span>
        <div className="of-pv-banner">
          <span>{offer}</span>
        </div>
        <div className="of-pv-btns">
          <span className="of-pv-btn">{t(mof.cancelKeep, v)}</span>
          <span className="of-pv-out">{t(mof.cancelContinue)}</span>
        </div>
      </div>
    </section>
  )
}

/** Results · since 1 Oct. */
function Results({ d }: { d: Detail }) {
  const t = useT()
  const r = d.results
  const stat = (value: string, label: string) => (
    <div className="flex min-w-0 flex-col gap-0.5">
      <span className="text-[22px] font-bold tabular-nums">{value}</span>
      <span className="k">{label}</span>
    </div>
  )
  const a = d.applies[0]
  const started = d.startedBy
    ? d.endsAt
      ? t(mof.startedByEnds, { who: d.startedBy, date: dayMonthDate(d.startedAt), ends: dayMonthDate(d.endsAt) })
      : t(mof.startedBy, { who: d.startedBy, date: dayMonthDate(d.startedAt) })
    : ''
  return (
    <section className="card">
      <div className="of-sec">
        <h2 className="h2">{t(mof.resultsTitle, { date: dayMonthDate(d.startedAt) })}</h2>
      </div>
      <div className="of-body">
        {d.kind === 'signup' ? (
          <>
            <div className="flex flex-wrap gap-x-8 gap-y-3">
              {stat(count(r.redemptions), t(mof.rRedemptions))}
              {stat(count(r.newMembers), t(mof.rNew))}
            </div>
            {stat(money(r.mrrCents, d.currency, 'never'), t(mof.rMrr))}
            {d.limit ? (
              <div className="flex flex-col gap-1">
                <div className="flex justify-between text-[13px]">
                  <span>{t(mof.used)}</span>
                  <b className="tabular-nums">{t(mof.useOf, { used: count(r.redemptions), limit: count(d.limit) })}</b>
                </div>
                <div className="bar" aria-hidden="true">
                  <i style={{ width: `${Math.min(100, (r.redemptions / d.limit) * 100)}%`, background: 'var(--accent)' }} />
                </div>
              </div>
            ) : null}
            <span className="k">
              {started}
              {a && r.redemptions
                ? ` · ${t(mof.mrrMath, { count: r.redemptions, price: money(discounted(a.priceCents, d), a.currency), div: a.period === 'year' ? 12 : 1 })}`
                : ''}
            </span>
          </>
        ) : d.kind === 'retention' ? (
          <>
            <div className="flex flex-wrap gap-x-8 gap-y-3">
              {stat(count(r.shown), t(mof.rShown))}
              {stat(count(r.accepted), t(mof.rAccepted, { rate: pct(r.acceptRate, 0) }))}
            </div>
            <span className="k">{t(mof.reasonsNote)}</span>
          </>
        ) : (
          <div className="flex flex-wrap gap-x-8 gap-y-3">
            {stat(count(r.codesIssued), t(mof.rIssued))}
            {stat(count(r.codesRedeemed), t(mof.rRedeemed))}
          </div>
        )}
      </div>
    </section>
  )
}

/** A referral reward's codes. */
function Codes({ d }: { d: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const redeemed = d.codes.filter((c) => c.status === 'redeemed').length
  return (
    <section className="card">
      <div className="of-sec flex flex-col gap-0.5">
        <h2 className="h2">{t(mof.codesTitle, { name: d.name })}</h2>
        <span className="k">{t(mof.codesSub, { issued: d.codes.length, redeemed })}</span>
      </div>
      <div className="of-body">
        <Note tone="neutral">
          <T msg={mof.codesNote} values={{ tiers: tiersLine(t, d.applies) }} />
        </Note>
      </div>
      {d.codes.length ? (
        <div className="overflow-x-auto">
          <table className="tbl">
            <thead>
              <tr>
                <th scope="col">{t(mof.cCode)}</th>
                <th scope="col">{t(mof.cIssuedTo)}</th>
                <th scope="col">{t(mof.cIssued)}</th>
                <th scope="col">{t(mof.cValid)}</th>
                <th scope="col">{t(mof.cStatus)}</th>
              </tr>
            </thead>
            <tbody>
              {d.codes.map((c) => (
                <tr key={c.code}>
                  <td className="mono">{c.code}</td>
                  <td>{c.issuedTo}</td>
                  <td>{dayMonthDate(c.issuedAt)}</td>
                  <td>{dayMonthDate(c.expiresAt)}</td>
                  <td>
                    <Badge tone={c.status === 'redeemed' ? 'ok' : 'neutral'}>{c.status === 'redeemed' ? t(mof.cRedeemed) : c.status === 'expired' ? t(mof.cExpired) : t(mof.cNot)}</Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <p className="of-body k m-0">{t(mof.noCodes)}</p>
      )}
      <div className="of-body flex flex-row! flex-wrap items-center justify-between gap-2" style={{ borderTop: '1px solid var(--line-2)' }}>
        <span className="k">{t(mof.codesFoot, { discount: discountLine(t, d), days: d.codeDays })}</span>
        <Link to="/w/$workspace/grow" params={{ workspace: slug }} className="text-[13px] font-semibold">
          {t(mof.growLink)}
        </Link>
      </div>
    </section>
  )
}
