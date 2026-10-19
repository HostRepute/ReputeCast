import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useMemo, useState } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select, Textarea, Toggle } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN, ProviderChip, useEarnStatus } from './EarnPage'
import { money, pricePer } from './format'
import { me } from './messages'
import { mp } from './plans.messages'
import './earn.css'

type Tier = Schemas['EarnTier']
type Input = Schemas['EarnTierInput']

const trials = [0, 7, 14, 30] as const

function toInput(tier?: Tier): Input {
  if (!tier) return { name: '', slug: '', description: '', benefits: [''], monthly: 0, yearly: 0, trialDays: 0, lists: [], accessRank: 1, visibility: 'public', highlight: '' }
  return {
    name: tier.name,
    slug: tier.slug,
    description: tier.description,
    benefits: tier.benefits.length ? tier.benefits : [''],
    monthly: tier.monthly?.amountCents ?? 0,
    yearly: tier.yearly?.amountCents ?? 0,
    trialDays: tier.trialDays,
    lists: tier.lists.map((l) => l.id),
    accessRank: tier.accessRank,
    visibility: tier.visibility,
    highlight: tier.highlight ?? '',
  }
}

/** The tier editor (EarnPlanEditors): /earn/plans/tiers/new and /earn/plans/tiers/{id}. */
export function TierEditorPage({ tierId }: { tierId?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const plans = $api.useQuery('get', `${EARN}/plans`, { params: { path: { workspace: slug } } })
  const tier = $api.useQuery('get', `${EARN}/tiers/{id}`, { params: { path: { workspace: slug, id: tierId ?? '' } } }, { enabled: Boolean(tierId) })
  const status = useEarnStatus()
  const title = tierId ? (tier.data?.name ?? '…') : t(mp.editorNew)
  const crumbs = [
    { label: t(me.title), to: '/w/$workspace/earn' as const, params: { workspace: slug } },
    { label: t(mp.editorCrumb), to: '/w/$workspace/earn/plans' as const, params: { workspace: slug } },
    { label: title },
  ]
  const err = plans.error ?? tier.error
  return (
    <Page title={title} breadcrumb={crumbs} badges={<ProviderChip status={status.data} />}>
      {err ? (
        <ErrorPanel title={errorMessage(err)} actions={<Button onClick={() => void plans.refetch()}>{t(me.retry)}</Button>} />
      ) : !plans.data || (tierId && !tier.data) ? (
        <div className="card pad" aria-busy="true">
          <Skeleton w="100%" h={260} />
        </div>
      ) : (
        <TierForm key={tier.data?.id ?? 'new'} tier={tier.data} plans={plans.data} account={status.data?.account ?? ''} />
      )}
    </Page>
  )
}

function TierForm({ tier, plans, account }: { tier?: Tier; plans: Schemas['EarnPlans']; account: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const saved = useMemo(() => toInput(tier), [tier])
  const [v, setV] = useState<Input>(saved)
  const [monthlyOn, setMonthlyOn] = useState(Boolean(saved.monthly))
  const [yearlyOn, setYearlyOn] = useState(Boolean(saved.yearly))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [deleting, setDeleting] = useState(false)
  const create = $api.useMutation('post', `${EARN}/tiers`)
  const update = $api.useMutation('put', `${EARN}/tiers/{id}`)
  const del = $api.useMutation('delete', `${EARN}/tiers/{id}`)
  const ro = !plans.canEdit
  const cur = plans.currency || 'usd'
  const set = <K extends keyof Input>(k: K, x: Input[K]) => setV((p) => ({ ...p, [k]: x }))
  const major = (minor: number | undefined) => (minor ? String(minor / 100) : '')
  const minor = (s: string) => Math.round(Number(s.replace(/[^0-9.]/g, '')) * 100) || 0
  const back = () => void navigate({ to: '/w/$workspace/earn/plans', params: { workspace: slug } })
  const body: Input = { ...v, benefits: v.benefits.map((b) => b.trim()).filter(Boolean), monthly: monthlyOn ? v.monthly : 0, yearly: yearlyOn ? v.yearly : 0 }
  const others = plans.tiers.filter((x) => x.id !== tier?.id)
  const onSaved = (r: Schemas['EarnTierSaved']) => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/tiers/{id}`] })
    if (r.syncError) toast({ kind: 'error', title: t(mp.savedSyncFailed, { tier: r.tier.name }), detail: t(mp.syncFailedText, { error: r.syncError }) })
    else toast({ kind: 'success', title: plans.connected ? t(mp.savedStripe, { tier: r.tier.name }) : t(mp.saved, { tier: r.tier.name }) })
    back()
  }
  const onError = (e: unknown) => {
    if (e instanceof ApiError && e.problem.errors?.length) {
      const next: Record<string, string> = {}
      for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
      setErrors(next)
    }
    toast({ kind: 'error', title: errorMessage(e) })
  }
  const save = () => {
    const next: Record<string, string> = {}
    if (!body.name.trim()) next.name = t(mp.errName)
    if (!body.monthly && !body.yearly) next.monthly = t(mp.errPrice)
    setErrors(next)
    if (Object.keys(next).length) return
    if (tier) update.mutate({ params: { path: { workspace: slug, id: tier.id } }, body }, { onSuccess: onSaved, onError })
    else create.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: onSaved, onError })
  }
  const busy = create.isPending || update.isPending
  const yearlyFree = body.monthly && body.yearly ? Math.round((body.monthly * 12 - body.yearly) / body.monthly) : 0
  const changedMonthly = tier?.monthly && body.monthly && body.monthly !== tier.monthly.amountCents ? tier.monthly : undefined
  const changedYearly = tier?.yearly && body.yearly && body.yearly !== tier.yearly.amountCents ? tier.yearly : undefined
  return (
    <>
      <div className="mb-4 flex flex-wrap justify-end gap-2">
        <Button onClick={back}>{t(mp.discard)}</Button>
        {ro ? null : (
          <Button variant="primary" onClick={save} loading={busy} busyLabel={t(mp.saving)}>
            {tier ? t(mp.save) : t(mp.create)}
          </Button>
        )}
      </div>
      <div className="grid items-start gap-4" style={{ gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 360px)' }}>
        <div className="flex min-w-0 flex-col gap-4">
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mp.name)}</h2>
            </div>
            <div className="pad flex flex-col gap-3">
              <Field label={t(mp.name)} hint={t(mp.nameHint)} error={errors.name}>
                <Input value={v.name} onChange={(e) => set('name', e.target.value)} maxLength={80} disabled={ro} />
              </Field>
              <Field label={t(mp.description)} hint={t(mp.descriptionHint)} optional>
                <Textarea value={v.description} onChange={(e) => set('description', e.target.value)} maxLength={500} rows={3} disabled={ro} />
              </Field>
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2">
                  {t(mp.benefitsLabel)} · {body.benefits.length}
                </h2>
                <span className="k">{t(mp.benefitsHint)}</span>
              </div>
              {ro ? null : (
                <Button size="sm" icon="plus" onClick={() => set('benefits', [...v.benefits, ''])} disabled={v.benefits.length >= 12}>
                  {t(mp.addBenefit)}
                </Button>
              )}
            </div>
            <div className="pad flex flex-col gap-2">
              {v.benefits.map((b, i) => (
                <div key={i} className="flex items-center gap-2">
                  <Input
                    aria-label={t(mp.benefitN, { n: i + 1 })}
                    value={b}
                    maxLength={120}
                    disabled={ro}
                    onChange={(e) => set('benefits', v.benefits.map((x, j) => (j === i ? e.target.value : x)))}
                  />
                  {ro ? null : (
                    <Button
                      size="sm"
                      iconOnly
                      icon="close"
                      aria-label={t(mp.removeBenefit, { benefit: b || t(mp.benefitN, { n: i + 1 }) })}
                      onClick={() => set('benefits', v.benefits.filter((_, j) => j !== i))}
                    />
                  )}
                </div>
              ))}
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2">{t(mp.pricesTitle)}</h2>
                <span className="k">{t(mp.pricesCaption, { currency: cur.toUpperCase() })}</span>
              </div>
            </div>
            <div className="pad flex flex-col gap-4">
              {(['month', 'year'] as const).map((period) => {
                const on = period === 'month' ? monthlyOn : yearlyOn
                const setOn = period === 'month' ? setMonthlyOn : setYearlyOn
                const key = period === 'month' ? 'monthly' : 'yearly'
                const cur0 = period === 'month' ? tier?.monthly : tier?.yearly
                return (
                  <div key={period} className="flex flex-wrap items-center justify-between gap-3">
                    <div className="flex min-w-0 flex-col">
                      <span className="font-semibold">{period === 'month' ? t(me.periodMonthly) : t(me.periodYearly)}</span>
                      {cur0 ? <span className="k">{t(mp.paidMembers)}: {count(cur0.members)}</span> : null}
                    </div>
                    <div className="flex items-center gap-2">
                      <Toggle label={period === 'month' ? t(mp.monthlyPrice) : t(mp.yearlyPrice)} hideLabel checked={on} disabled={ro} onChange={(e) => setOn(e.target.checked)} />
                      <Input
                        aria-label={period === 'month' ? t(mp.monthlyPrice) : t(mp.yearlyPrice)}
                        inputMode="decimal"
                        className="w-[120px]"
                        disabled={ro || !on}
                        value={major(v[key])}
                        onChange={(e) => set(key, minor(e.target.value))}
                      />
                      <span className="k">{period === 'month' ? t(mp.unitMonth) : t(mp.unitYear)}</span>
                    </div>
                  </div>
                )
              })}
              {errors.monthly ? (
                <span className="err" role="alert">
                  {errors.monthly}
                </span>
              ) : null}
              {yearlyFree > 0 ? <span className="k">{t(mp.yearlySaving, { count: yearlyFree })}</span> : null}
              {changedMonthly ? <Note tone="info">{t(mp.legacyNote, { count: changedMonthly.members, price: pricePer(t, changedMonthly.amountCents, cur, 'month') })}</Note> : null}
              {changedYearly ? <Note tone="info">{t(mp.legacyNote, { count: changedYearly.members, price: pricePer(t, changedYearly.amountCents, cur, 'year') })}</Note> : null}
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mp.trialLabel)}</h2>
            </div>
            <div className="pad flex flex-col gap-2">
              <Segmented
                label={t(mp.trialLabel)}
                value={String(v.trialDays)}
                onChange={(x) => !ro && set('trialDays', Number(x) as Input['trialDays'])}
                items={trials.map((d) => ({ value: String(d), label: d ? t(mp.trialOption, { days: d }) : t(mp.trialOff) }))}
              />
              <span className="k">{t(mp.trialHint)}</span>
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">
                {t(mp.listsLabel)} · {t(mp.access)}
              </h2>
            </div>
            <div className="pad flex flex-col gap-4">
              <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
                <legend className="mb-1 text-[13px] font-semibold">{t(mp.listsLabel)}</legend>
                {plans.lists.map((l) => (
                  <label key={l.id} className="flex items-center gap-2 text-[13px]">
                    <input
                      type="checkbox"
                      disabled={ro}
                      checked={v.lists.includes(l.id)}
                      onChange={(e) => set('lists', e.target.checked ? [...v.lists, l.id] : v.lists.filter((x) => x !== l.id))}
                    />
                    {l.name}
                  </label>
                ))}
                <span className="k">{t(mp.listsHint)}</span>
              </fieldset>
              <Field label={t(mp.rankLabel)} hint={t(mp.rankHint)}>
                <Select value={String(v.accessRank)} disabled={ro} onChange={(e) => set('accessRank', Number(e.target.value))} className="max-w-[200px]">
                  {Array.from({ length: Math.max(3, others.length + 2) }, (_, i) => i + 1).map((r) => (
                    <option key={r} value={r}>
                      {t(mp.rank, { rank: r })}
                    </option>
                  ))}
                </Select>
              </Field>
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mp.visibility)}</h2>
            </div>
            <div className="pad flex flex-col gap-2">
              <OptionCard name="visibility" value="public" checked={v.visibility === 'public'} onChange={(x) => set('visibility', x as Input['visibility'])} title={t(mp.visPublic)} description={t(mp.visPublicHint)} disabled={ro} />
              <OptionCard
                name="visibility"
                value="hidden"
                checked={v.visibility === 'hidden'}
                onChange={(x) => set('visibility', x as Input['visibility'])}
                title={t(mp.visHidden)}
                description={t(mp.visHiddenHint, { link: tier?.hiddenLink ?? `${plans.upgradePage.host}/${v.slug || '…'}` })}
                disabled={ro}
              />
              <OptionCard name="visibility" value="archived" checked={v.visibility === 'archived'} onChange={(x) => set('visibility', x as Input['visibility'])} title={t(mp.visArchived)} description={t(mp.visArchivedHint)} disabled={ro} />
              <div className="mt-2 flex flex-wrap items-center justify-between gap-3">
                <Toggle label={t(mp.highlight)} checked={Boolean(v.highlight)} disabled={ro} onChange={(e) => set('highlight', e.target.checked ? 'Most popular' : '')} />
                {v.highlight ? (
                  <Input aria-label={t(mp.highlightLabel)} value={v.highlight} maxLength={30} className="max-w-[200px]" disabled={ro} onChange={(e) => set('highlight', e.target.value)} />
                ) : null}
              </div>
            </div>
          </section>
          {tier && !ro && tier.canDelete ? (
            <div>
              <Button variant="danger" onClick={() => setDeleting(true)}>
                {t(mp.delete)}
              </Button>
            </div>
          ) : null}
        </div>
        <aside className="flex min-w-0 flex-col gap-4">
          <TierPreview v={body} currency={cur} host={plans.upgradePage.host} />
          <StripeCard tier={tier} account={account} connected={plans.connected} />
        </aside>
      </div>
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        tone="bad"
        badge={t(mp.delete)}
        title={t(mp.deleteTitle, { tier: tier?.name ?? '' })}
        description={t(mp.deleteText)}
        confirmLabel={t(mp.deleteConfirm)}
        onConfirm={async () => {
          if (!tier) return
          await del.mutateAsync({ params: { path: { workspace: slug, id: tier.id } } })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
          toast({ kind: 'deleted', title: t(mp.deleted, { tier: tier.name }) })
          back()
        }}
      />
    </>
  )
}

/** "How readers see it": the tier's card on the upgrade page, yearly first. */
function TierPreview({ v, currency, host }: { v: Input; currency: string; host: string }) {
  const t = useT()
  const [period, setPeriod] = useState<'month' | 'year'>(v.yearly ? 'year' : 'month')
  const amount = period === 'year' ? v.yearly : v.monthly
  const free = v.monthly && v.yearly ? Math.round((v.monthly * 12 - v.yearly) / v.monthly) : 0
  return (
    <section className="card" aria-label={t(mp.previewLabel)}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mp.previewLabel)}</h2>
          <span className="k">{host}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3" style={{ background: '#f5f3ff' }}>
        {v.monthly && v.yearly ? (
          <Segmented
            label={t(mp.defaultPeriod)}
            value={period}
            onChange={setPeriod}
            items={[
              { value: 'month', label: t(me.periodMonthly) },
              { value: 'year', label: free > 0 ? `${t(me.periodYearly)} · ${t(mp.monthsFree, { count: free })}` : t(me.periodYearly) },
            ]}
          />
        ) : null}
        <div className="ea-pcard" data-highlight={v.highlight ? true : undefined}>
          {v.highlight ? <span className="ea-pcard-hl">{v.highlight}</span> : null}
          <h3 className="m-0 text-[16px] font-bold">{v.name || t(mp.editorNew)}</h3>
          <div className="flex items-baseline gap-1">
            <span className="text-[28px] font-bold">{amount ? money(amount, currency) : '—'}</span>
            <span className="k">{period === 'year' ? t(mp.aYearUnit) : t(mp.aMonthUnit)}</span>
          </div>
          {v.trialDays ? <Badge tone="info">{t(mp.trialFree, { days: v.trialDays })}</Badge> : null}
          <ul className="m-0 flex list-none flex-col gap-1.5 p-0 text-[13px]">
            {v.benefits.map((b, i) => (
              <li key={i} className="flex gap-2">
                <Icon name="check" size={14} />
                {b}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}

/** "In Stripe": the product, the prices and their ids, the legacy prices. */
function StripeCard({ tier, account, connected }: { tier?: Tier; account: string; connected: boolean }) {
  const t = useT()
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mp.stripeTitle)}</h2>
          <span className="k">{connected ? t(mp.stripeCaption, { account }) : t(mp.stripeDraft)}</span>
        </div>
        {tier?.productUrl ? (
          <ButtonAnchor href={tier.productUrl} size="sm" iconRight="external">
            {t(me.openInStripe)}
          </ButtonAnchor>
        ) : null}
      </div>
      {tier ? (
        <div className="pad flex flex-col gap-3">
          <dl className="ea-kv">
            <dt>{t(mp.stepProduct)}</dt>
            <dd>
              {tier.name} {tier.productId ? <span className="ea-mono k">{tier.productId}</span> : null}
            </dd>
          </dl>
          {tier.prices.map((p) => (
            <div key={p.id} className="flex flex-wrap items-baseline justify-between gap-2 text-[13px]">
              <span>
                <span className="font-semibold">{pricePer(t, p.amountCents, p.currency, p.period)}</span> {p.stripeId ? <span className="ea-mono k">{p.stripeId}</span> : null}
                {p.status === 'legacy' ? <Badge tone="neutral" className="ml-2">{t(me.legacyPrice)}</Badge> : null}
              </span>
              <span className="k">
                {t(mp.paidMembers)}: {count(p.members)}
              </span>
            </div>
          ))}
          {tier.syncSteps.some((s) => s.status === 'failed') ? (
            <Note tone="bad">{tier.syncSteps.find((s) => s.status === 'failed')?.error}</Note>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}
