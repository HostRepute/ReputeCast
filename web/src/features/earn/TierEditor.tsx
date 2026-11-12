import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ConfirmDialog, UnsavedChangesGuard } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, dayMonthDate, orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { EARN, ProviderChip, useEarnStatus } from './EarnPage'
import { money, platformName, pricePer } from './format'
import { me } from './messages'
import { Grip, moved, useReorder } from './Reorder'
import { mt } from './tier.messages'
import './earn.css'
import { SwitchInput } from '@/components/ui/Field'

type Tier = Schemas['EarnTier']
type Input = Schemas['EarnTierInput']
type Plans = Schemas['EarnPlans']
type T = ReturnType<typeof useT>

const trials = [0, 7, 14, 30] as const

/** A tier's group prices as the editor holds them (M65): each period on or off, per seat, and the seats a team can buy. */
type Group = { monthlyOn: boolean; yearlyOn: boolean; monthly: number; yearly: number; seatsMin: number; seatsMax: number }

function toGroup(tier?: Tier): Group {
  const g = tier?.group
  return {
    monthlyOn: Boolean(g?.monthly),
    yearlyOn: Boolean(g?.yearly),
    monthly: g?.monthly?.amountCents ?? 0,
    yearly: g?.yearly?.amountCents ?? 0,
    seatsMin: g?.seatsMin ?? 2,
    seatsMax: g?.seatsMax ?? 1000,
  }
}

/** The tier input's group part: prices when one is on, "off" when the tier had some, nothing otherwise. */
function groupBody(g: Group, had: boolean): Input['group'] {
  const monthly = g.monthlyOn ? g.monthly : 0
  const yearly = g.yearlyOn ? g.yearly : 0
  if (!monthly && !yearly) return had ? { off: true } : undefined
  return { monthly, yearly, seatsMin: g.seatsMin, seatsMax: g.seatsMax }
}

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
  const err = plans.error ?? tier.error
  if (err || !plans.data || (tierId && !tier.data)) {
    return (
      <Frame title={tierId ? (tier.data?.name ?? '…') : t(mt.newTier)}>
        {err ? (
          <ErrorPanel title={errorMessage(err)} actions={<Button onClick={() => void (plans.error ? plans.refetch() : tier.refetch())}>{t(me.retry)}</Button>} />
        ) : (
          <div className="card pad" aria-busy="true">
            <Skeleton w="100%" h={260} />
          </div>
        )}
      </Frame>
    )
  }
  return <TierForm key={tier.data?.id ?? 'new'} tier={tier.data} plans={plans.data} account={status.data?.account ?? ''} />
}

/** The top bar: Earn / Plans & offers / the tier, the provider chip and the actions. */
function Frame({ title, actions, children }: { title: string; actions?: ReactNode; children: ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = useEarnStatus()
  const crumbs = [
    { label: t(me.title), to: '/w/$workspace/earn' as const, params: { workspace: slug } },
    { label: t(mt.crumb), to: '/w/$workspace/earn/plans' as const, params: { workspace: slug } },
    { label: title },
  ]
  return (
    <Page title={title} breadcrumb={crumbs} badges={<ProviderChip status={status.data} />} actions={actions}>
      {children}
    </Page>
  )
}

function TierForm({ tier, plans, account }: { tier?: Tier; plans: Plans; account: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const saved = useMemo(() => toInput(tier), [tier])
  const [v, setV] = useState<Input>(saved)
  const [monthlyOn, setMonthlyOn] = useState(tier ? Boolean(saved.monthly) : true)
  const [yearlyOn, setYearlyOn] = useState(tier ? Boolean(saved.yearly) : true)
  const savedGroup = useMemo(() => toGroup(tier), [tier])
  const [group, setGroup] = useState<Group>(savedGroup)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [deleting, setDeleting] = useState(false)
  const [archiving, setArchiving] = useState(false)
  // A save whose prices didn't all reach Stripe: the tier as saved, with its steps.
  const [failed, setFailed] = useState<Tier>()
  // Leaving with unsaved changes asks first (review 2026-11 EA-1); a save
  // marks the tier saved before it navigates.
  const [leaving, setLeaving] = useState<'plans' | string>()
  const create = $api.useMutation('post', `${EARN}/tiers`)
  const update = $api.useMutation('put', `${EARN}/tiers/{id}`)
  const del = $api.useMutation('delete', `${EARN}/tiers/{id}`)
  const sync = $api.useMutation('post', `${EARN}/tiers/{id}/sync`)
  const ro = !plans.canEdit
  const cur = plans.currency || 'usd'
  const set = <K extends keyof Input>(k: K, x: Input[K]) => setV((p) => ({ ...p, [k]: x }))
  const major = (minor: number | undefined) => (minor ? String(minor / 100) : '')
  const minor = (s: string) => Math.round(Number(s.replace(/[^0-9.]/g, '')) * 100) || 0
  const back = () => void navigate({ to: '/w/$workspace/earn/plans', params: { workspace: slug } })
  const hadGroup = Boolean(tier?.group)
  const body: Input = {
    ...v,
    benefits: v.benefits.map((b) => b.trim()).filter(Boolean),
    monthly: monthlyOn ? v.monthly : 0,
    yearly: yearlyOn ? v.yearly : 0,
    group: groupBody(group, hadGroup),
  }
  const benefits = useReorder((from, to) => set('benefits', moved(v.benefits, from, to)), v.benefits.length)
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/tiers/{id}`] })
    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/checklist`] })
  }
  const onSaved = (r: Schemas['EarnTierSaved']) => {
    refresh()
    if (r.syncError) {
      setFailed(r.tier)
      toast({ kind: 'error', title: t(mt.savedSyncFailed, { tier: r.tier.name }), detail: r.syncError })
      if (!tier) setLeaving(r.tier.id)
      return
    }
    toast({ kind: 'success', title: plans.connected ? t(mt.savedStripe, { tier: r.tier.name }) : t(mt.saved, { tier: r.tier.name }) })
    setLeaving('plans')
  }
  useEffect(() => {
    if (leaving === 'plans') back()
    else if (leaving) void navigate({ to: '/w/$workspace/earn/plans/tiers/$tierId', params: { workspace: slug, tierId: leaving }, replace: true })
  }, [leaving]) // eslint-disable-line react-hooks/exhaustive-deps
  const onError = (e: unknown) => {
    if (e instanceof ApiError && e.problem.errors?.length) {
      const next: Record<string, string> = {}
      for (const x of e.problem.errors) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
      setErrors(next)
    }
    toast({ kind: 'error', title: errorMessage(e) })
  }
  const submit = () => {
    if (tier) update.mutate({ params: { path: { workspace: slug, id: tier.id } }, body }, { onSuccess: onSaved, onError })
    else create.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: onSaved, onError })
  }
  const save = () => {
    const next: Record<string, string> = {}
    if (!body.name.trim()) next.name = t(mt.errName)
    if (!body.monthly && !body.yearly) next.monthly = t(mt.errPrice)
    if (body.group && !body.group.off) {
      const { seatsMin = 0, seatsMax = 0 } = body.group
      if (seatsMin < 2 || seatsMax > 1000) next['group.seatsMax'] = t(mt.groupSeatsError)
      else if (seatsMin > seatsMax) next['group.seatsMax'] = t(mt.groupSeatsOrder)
    }
    setErrors(next)
    if (Object.keys(next).length) return
    // Archiving a tier people pay for is confirmed first.
    if (tier && tier.visibility !== 'archived' && body.visibility === 'archived') {
      setArchiving(true)
      return
    }
    submit()
  }
  const busy = create.isPending || update.isPending
  const pristine = useMemo(
    () =>
      JSON.stringify({
        ...saved,
        benefits: saved.benefits.map((b) => b.trim()).filter(Boolean),
        monthly: tier && !saved.monthly ? 0 : saved.monthly,
        yearly: tier && !saved.yearly ? 0 : saved.yearly,
        group: groupBody(savedGroup, Boolean(tier?.group)),
      }),
    [saved, savedGroup, tier],
  )
  const dirty = !ro && !leaving && JSON.stringify(body) !== pristine
  const title = tier ? tier.name : t(mt.newTier)
  const named = Boolean(tier?.shortName || body.name.trim())
  const short = tier?.shortName || body.name.trim() || t(mt.untitled)
  const others = plans.tiers.filter((x) => x.id !== tier?.id)
  const twelve = (body.monthly ?? 0) * 12
  const yearlyMore = Boolean(body.monthly && body.yearly && body.yearly > twelve)
  const free = body.monthly && body.yearly && !yearlyMore ? Math.round((twelve - body.yearly) / body.monthly) : 0
  const changed = (period: 'month' | 'year') => {
    const p = period === 'month' ? tier?.monthly : tier?.yearly
    const now = period === 'month' ? body.monthly : body.yearly
    return p && now && now !== p.amountCents ? p : undefined
  }
  const changedMonth = changed('month')
  const changedYear = changed('year')
  const membersLine = (p?: Schemas['EarnPrice']) => {
    if (!p) return null
    const parts = [t(mt.paidMembers, { count: p.paying })]
    for (const o of p.offers ?? []) parts.push(t(mt.withOffer, { count: count(o.count), offer: o.label }))
    return parts.join(' · ')
  }
  const actions = (
    <>
      <Button onClick={() => (JSON.stringify(v) !== JSON.stringify(saved) ? setV(saved) : back())}>{t(mt.discard)}</Button>
      {ro ? null : (
        <Button variant="primary" onClick={save} loading={busy} busyLabel={t(mt.saving)}>
          {t(mt.save)}
        </Button>
      )}
    </>
  )
  return (
    <Frame title={title} actions={actions}>
      <UnsavedChangesGuard dirty={dirty} description={t(mt.leaveTier)} />
      {ro ? (
        <Note tone="neutral">
          <T msg={plans.helpers.length ? mt.readOnly : mt.readOnlyNoHelpers} values={{ helpers: orList(plans.helpers) }} />
        </Note>
      ) : null}
      {busy && !tier && plans.connected ? <SavingCard /> : null}
      {failed ? (
        <SyncFailedCard
          tier={failed}
          currency={cur}
          busy={sync.isPending}
          onRetry={() =>
            sync.mutate(
              { params: { path: { workspace: slug, id: failed.id } } },
              {
                onSuccess: (r) => {
                  refresh()
                  if (r.syncError) {
                    setFailed(r.tier)
                    toast({ kind: 'error', title: t(mt.savedSyncFailed, { tier: r.tier.name }), detail: r.syncError })
                  } else {
                    setFailed(undefined)
                    toast({ kind: 'success', title: t(mt.retried, { tier: r.tier.name }) })
                  }
                },
                onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
              },
            )
          }
        />
      ) : null}
      <div className="ea-tier-grid">
        <div className="flex min-w-0 flex-col gap-4">
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mt.nameTitle)}</h2>
            </div>
            <div className="pad flex flex-col gap-3">
              <label className="lbl">
                {t(mt.name)}
                <input className="in" aria-label={t(mt.nameLabel)} value={v.name} maxLength={80} disabled={ro} aria-invalid={errors.name ? true : undefined} onChange={(e) => set('name', e.target.value)} />
                {errors.name ? (
                  <span className="err" role="alert">
                    {errors.name}
                  </span>
                ) : null}
              </label>
              <label className="lbl">
                {t(mt.description)}
                <textarea className="in ea-ta" style={{ height: 64 }} aria-label={t(mt.description)} value={v.description} maxLength={500} disabled={ro} onChange={(e) => set('description', e.target.value)} />
                <span className="k">{t(mt.descriptionHint)}</span>
              </label>
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2">{t(mt.benefits, { count: body.benefits.length })}</h2>
                <span className="k">{t(mt.benefitsHint)}</span>
              </div>
              {ro ? null : (
                <Button size="sm" icon="plus" onClick={() => set('benefits', [...v.benefits, ''])} disabled={v.benefits.length >= 12}>
                  {t(mt.addBenefit)}
                </Button>
              )}
            </div>
            <div className="pad flex flex-col gap-2" data-rows>
              {v.benefits.map((b, i) => {
                const name = b || t(mt.benefitN, { n: i + 1 })
                return (
                  <div key={i} className="ea-orow" data-row {...benefits.row(i)}>
                    <Grip label={t(mt.moveBenefit, { benefit: name })} disabled={ro || v.benefits.length < 2} {...benefits.grip(i)} />
                    <input
                      className="in min-w-0 flex-1"
                      aria-label={t(mt.benefitN, { n: i + 1 })}
                      value={b}
                      maxLength={120}
                      disabled={ro}
                      onChange={(e) => set('benefits', v.benefits.map((x, j) => (j === i ? e.target.value : x)))}
                    />
                    {ro ? null : (
                      <Button size="sm" iconOnly icon="close" aria-label={t(mt.removeBenefit, { benefit: name })} onClick={() => set('benefits', v.benefits.filter((_, j) => j !== i))} />
                    )}
                  </div>
                )
              })}
            </div>
          </section>
          <PricesCard
            v={body}
            raw={v}
            currency={cur}
            ro={ro}
            monthlyOn={monthlyOn}
            yearlyOn={yearlyOn}
            setMonthlyOn={setMonthlyOn}
            setYearlyOn={setYearlyOn}
            setAmount={(k, s) => set(k, minor(s))}
            major={major}
            monthlyLine={membersLine(tier?.monthly)}
            yearlyLine={membersLine(tier?.yearly)}
            error={errors.monthly}
            changedMonth={changedMonth}
            changedYear={changedYear}
            yearlyMore={yearlyMore}
            free={free}
          />
          <GroupPricesCard
            g={group}
            set={(patch) => setGroup((p) => ({ ...p, ...patch }))}
            tier={tier}
            currency={cur}
            ro={ro}
            major={major}
            minor={minor}
            host={plans.upgradePage.host}
            slug={body.slug || v.slug || ''}
            error={errors['group.seatsMax'] ?? errors['group.monthly']}
          />
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mt.trialTitle)}</h2>
            </div>
            <div className="pad flex flex-col gap-2">
              <div>
                <Segmented
                  label={t(mt.trialLength)}
                  value={String(v.trialDays)}
                  onChange={(x) => !ro && set('trialDays', Number(x) as Input['trialDays'])}
                  items={trials.map((d) => ({ value: String(d), label: d ? t(mt.trialDays, { days: d }) : t(mt.trialNone) }))}
                />
              </div>
              <span className="k">{t(mt.trialHint)}</span>
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mt.listsTitle)}</h2>
            </div>
            <div className="pad flex flex-col gap-4">
              <div className="lbl">
                {t(mt.listsJoin)}
                <div className="row flex-wrap gap-2">
                  {v.lists.map((id) => {
                    const l = plans.lists.find((x) => x.id === id)
                    if (!l) return null
                    return (
                      <span key={id} className="ea-lchip">
                        {l.name}
                        {ro ? null : (
                          <button type="button" aria-label={t(mt.removeList, { list: l.name })} onClick={() => set('lists', v.lists.filter((x) => x !== id))}>
                            <Icon name="close" size={12} strokeWidth={2.2} />
                          </button>
                        )}
                      </span>
                    )
                  })}
                  {ro || plans.lists.every((l) => v.lists.includes(l.id)) ? null : (
                    <Menu
                      align="start"
                      label={t(mt.addList)}
                      trigger={
                        <Button size="sm" icon="plus">
                          {t(mt.addList)}
                        </Button>
                      }
                    >
                      {plans.lists
                        .filter((l) => !v.lists.includes(l.id))
                        .map((l) => (
                          <MenuItem key={l.id} onSelect={() => set('lists', [...v.lists, l.id])}>
                            {l.name}
                          </MenuItem>
                        ))}
                    </Menu>
                  )}
                </div>
                <span className="k font-medium">{t(mt.listsHint)}</span>
              </div>
              <label className="lbl">
                {t(mt.rank)}
                <select className="in w-[160px]" aria-label={t(mt.rank)} value={String(v.accessRank)} disabled={ro} onChange={(e) => set('accessRank', Number(e.target.value))}>
                  {Array.from({ length: Math.max(3, others.length + 2) }, (_, i) => i + 1).map((r) => (
                    <option key={r} value={r}>
                      {t(mt.rankN, { rank: r })}
                    </option>
                  ))}
                </select>
                <span className="k font-medium">{named ? rankHint(t, short, v.accessRank, others) : t(mt.rankHint)}</span>
              </label>
            </div>
          </section>
          <section className="card">
            <div className="sec">
              <h2 className="h2">{t(mt.visibility)}</h2>
            </div>
            <div className="pad flex flex-col gap-2">
              {(['public', 'hidden', 'archived'] as const).map((x) => (
                <label key={x} className={`ea-radio${v.visibility === x ? ' on' : ''}`}>
                  <input type="radio" name="tier-visibility" checked={v.visibility === x} disabled={ro} onChange={() => set('visibility', x)} />
                  <span className="flex min-w-0 flex-col gap-0.5 font-semibold">
                    {x === 'public' ? t(mt.visPublic) : x === 'hidden' ? t(mt.visHidden) : t(mt.visArchived)}
                    <span className="k">
                      {x === 'public' ? (
                        t(mt.visPublicHint)
                      ) : x === 'hidden' ? (
                        <>
                          {t(mt.visHiddenHint)} <span className="mono">{tier?.hiddenLink ?? `${plans.upgradePage.host}/${v.slug || '…'}`}</span>
                        </>
                      ) : (
                        t(mt.visArchivedHint)
                      )}
                    </span>
                  </span>
                </label>
              ))}
            </div>
            <div className="ea-srow">
              <div className="d">
                <span className="font-semibold">{t(mt.highlight)}</span>
                <span className="k">{t(mt.highlightHint)}</span>
              </div>
              <div className="c">
                <input
                  className="in w-[150px]"
                  aria-label={t(mt.highlightLabel)}
                  value={v.highlight}
                  maxLength={30}
                  disabled={ro || !v.highlight}
                  placeholder={t(mt.highlightDefault)}
                  onChange={(e) => set('highlight', e.target.value)}
                />
                <SwitchInput
                  aria-label={t(mt.highlightThis)}
                  checked={Boolean(v.highlight)}
                  disabled={ro}
                  onChange={(e) => set('highlight', e.target.checked ? tier?.highlight || t(mt.highlightDefault) : '')}
                />
              </div>
            </div>
          </section>
          {tier && !ro && tier.canDelete ? (
            <div>
              <Button variant="danger" onClick={() => setDeleting(true)}>
                {t(mt.delete)}
              </Button>
            </div>
          ) : null}
        </div>
        <aside className="flex min-w-0 flex-col gap-4">
          <TierPreview v={body} short={named ? short : ''} currency={cur} host={plans.upgradePage.host} brand={plans.brand} yearlyMore={yearlyMore} free={free} />
          <StripeCard tier={tier} v={body} currency={cur} account={account} connected={plans.connected} changedMonth={changedMonth} changedYear={changedYear} membersLine={membersLine} />
        </aside>
      </div>
      <ConfirmDialog
        open={deleting}
        onOpenChange={setDeleting}
        tone="bad"
        badge={t(mt.delete)}
        title={t(mt.deleteTitle, { tier: tier?.name ?? '' })}
        description={t(mt.deleteText)}
        confirmLabel={t(mt.deleteConfirm)}
        onConfirm={async () => {
          if (!tier) return
          await del.mutateAsync({ params: { path: { workspace: slug, id: tier.id } } })
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
          toast({ kind: 'deleted', title: t(mt.deleted, { tier: tier.name }) })
          back()
        }}
      />
      {tier ? (
        <ConfirmDialog
          open={archiving}
          onOpenChange={setArchiving}
          tone="info"
          badge={t(mt.archiveBadge, { tier: tier.name, rank: tier.accessRank })}
          title={t(mt.archiveTitle, { tier: tier.name })}
          description={t(mt.archiveText, { count: tier.paying })}
          confirmLabel={t(mt.archiveConfirm)}
          confirmVariant="primary"
          onConfirm={async () => {
            submit()
          }}
        >
          <dl className="ea-kv">
            <dt>{t(mt.archiveMembers)}</dt>
            <dd>{t(mt.archiveMembersText, { count: tier.paying, price: tier.yearly ? pricePer(t, tier.yearly.amountCents, cur, 'year') : tier.monthly ? pricePer(t, tier.monthly.amountCents, cur, 'month') : '' })}</dd>
            <dt>{t(mt.archivePage)}</dt>
            <dd>{t(mt.archivePageText)}</dd>
            <dt>{t(mt.archiveStripe)}</dt>
            <dd>{t(mt.archiveStripeText)}</dd>
          </dl>
          <span className="k">{t(mt.archiveNote)}</span>
        </ConfirmDialog>
      ) : null}
    </Frame>
  )
}

/** "Founding member (rank 2) includes Insider's paid content…": how ranks reach paid content. */
function rankHint(t: T, short: string, rank: number, others: Plans['tiers']): string {
  const above = others.filter((x) => x.accessRank > rank && x.visibility !== 'archived').sort((a, b) => a.accessRank - b.accessRank)[0]
  if (above) return t(mt.rankHintAbove, { tier: above.name, rank: above.accessRank, short })
  const below = others.filter((x) => x.accessRank < rank && x.visibility !== 'archived')
  if (below.length) return t(mt.rankHintBelow, { tiers: orList(below.map((x) => x.shortName)), short })
  return t(mt.rankHint)
}

function PricesCard({
  v,
  raw,
  currency,
  ro,
  monthlyOn,
  yearlyOn,
  setMonthlyOn,
  setYearlyOn,
  setAmount,
  major,
  monthlyLine,
  yearlyLine,
  error,
  changedMonth,
  changedYear,
  yearlyMore,
  free,
}: {
  v: Input
  raw: Input
  currency: string
  ro: boolean
  monthlyOn: boolean
  yearlyOn: boolean
  setMonthlyOn: (on: boolean) => void
  setYearlyOn: (on: boolean) => void
  setAmount: (k: 'monthly' | 'yearly', s: string) => void
  major: (minor: number | undefined) => string
  monthlyLine: string | null
  yearlyLine: string | null
  error?: string
  changedMonth?: Schemas['EarnPrice']
  changedYear?: Schemas['EarnPrice']
  yearlyMore: boolean
  free: number
}) {
  const t = useT()
  const cur = currency.toUpperCase()
  const m = (x: number) => money(x, currency)
  const rows = [
    { period: 'month' as const, on: monthlyOn, setOn: setMonthlyOn, key: 'monthly' as const, line: monthlyLine },
    { period: 'year' as const, on: yearlyOn, setOn: setYearlyOn, key: 'yearly' as const, line: yearlyLine },
  ]
  const changed = changedMonth ?? changedYear
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mt.pricesTitle)}</h2>
          <span className="k">{t(mt.pricesHint, { currency: cur })}</span>
        </div>
      </div>
      {rows.map((r) => (
        <div key={r.period} className="ea-srow">
          <div className="d">
            <span className="font-semibold">{r.period === 'month' ? t(mt.monthly) : t(mt.yearly)}</span>
            {r.line ? <span className="k">{r.line}</span> : null}
          </div>
          <div className="c">
            <SwitchInput
              aria-label={r.period === 'month' ? t(mt.offerMonthly) : t(mt.offerYearly)}
              checked={r.on}
              disabled={ro}
              onChange={(e) => r.setOn(e.target.checked)}
            />
            <div className="row gap-1.5">
              <span className="k">$</span>
              <input
                className="in num w-[90px]"
                inputMode="decimal"
                aria-label={r.period === 'month' ? t(mt.monthlyIn, { currency: cur }) : t(mt.yearlyIn, { currency: cur })}
                disabled={ro || !r.on}
                value={major(raw[r.key])}
                onChange={(e) => setAmount(r.key, e.target.value)}
              />
              <span className="k">{r.period === 'month' ? t(mt.perMonth) : t(mt.perYear)}</span>
            </div>
          </div>
        </div>
      ))}
      <div className="pad flex flex-col gap-3 border-t border-line-2">
        {error ? (
          <span className="err" role="alert">
            {error}
          </span>
        ) : null}
        {v.monthly || v.yearly ? (
          <span>
            <span className="k">{t(mt.readersSee)}</span>{' '}
            {changedMonth && v.monthly ? (
              <>
                <b>{t(mt.seeMonth, { price: m(v.monthly) })}</b> {t(mt.forNew)}
                {v.yearly ? ` ${yearlyMore ? t(mt.nowMoreThanMonthly, { price: m(v.yearly) }) : t(mt.nowMonthsFree, { price: m(v.yearly), free: t(mt.monthsFree, { count: free }) })}` : ''}
              </>
            ) : v.yearly ? (
              <>
                <b>{t(mt.seeYear, { price: m(v.yearly) })}</b>{' '}
                {yearlyMore ? (
                  t(mt.moreThanMonthly, { price: money(Math.round(v.yearly / 12), currency) })
                ) : v.monthly && free > 0 ? (
                  <>
                    {t(mt.seeMonthly, { price: money(Math.round(v.yearly / 12), currency) })} <b>{t(mt.monthsFree, { count: free })}</b> {t(mt.comparedMonthly)}
                  </>
                ) : (
                  `· ${t(mt.perMonthYearly, { price: money(Math.round(v.yearly / 12), currency) })}`
                )}
              </>
            ) : (
              <b>{t(mt.seeMonth, { price: m(v.monthly ?? 0) })}</b>
            )}
          </span>
        ) : null}
        {yearlyMore ? (
          <Note tone="warn">
            <span className="font-semibold">{t(mt.yearlyMoreTitle)}</span>
            <div>{t(mt.yearlyMoreText, { yearly: m(v.yearly ?? 0), twelve: m((v.monthly ?? 0) * 12) })}</div>
          </Note>
        ) : changed ? (
          <Note tone="warn">
            <span className="font-semibold">{t(mt.newPriceTitle)}</span>
            <div>
              {t(mt.newPriceText, {
                price: pricePer(t, changedMonth ? (v.monthly ?? 0) : (v.yearly ?? 0), currency, changed.period),
                count: changed.paying,
                period: changed.period,
                old: m(changed.amountCents),
              })}
            </div>
          </Note>
        ) : (
          <Note tone="neutral">{t(mt.pricesNote)}</Note>
        )}
      </div>
    </section>
  )
}

/** "How readers see it": the tier's card on the upgrade page, in the publisher's colours. */
function TierPreview({
  v,
  short,
  currency,
  host,
  brand,
  yearlyMore,
  free,
}: {
  v: Input
  short: string
  currency: string
  host: string
  brand: Schemas['Brand']
  yearlyMore: boolean
  free: number
}) {
  const t = useT()
  const both = Boolean(v.monthly && v.yearly)
  const [picked, setPeriod] = useState<'month' | 'year'>('year')
  const period = both ? picked : v.yearly ? 'year' : 'month'
  const amount = (period === 'year' ? v.yearly : v.monthly) ?? 0
  const style = { '--ea-accent': brand.accent, '--ea-ground': brand.ground, '--ea-soft': brand.soft, '--ea-text': brand.text, '--ea-on': brand.onAccent } as CSSProperties
  const article = /^[aeiou]/i.test(short) ? 'an' : 'a'
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mt.previewTitle)}</h2>
          <span className="k">{yearlyMore ? t(mt.previewHintWarn) : short ? t(mt.previewHint, { tier: short, period }) : t(mt.previewHintNew)}</span>
        </div>
      </div>
      <div className="ea-tprev keep-light" style={style}>
        <span className="ea-tprev-on">{t(mt.previewOn, { host })}</span>
        {both ? (
          <div className="flex justify-center">
            <div className="ea-tprev-toggle" role="group" aria-label={t(mt.billingPeriod)}>
              <button type="button" aria-pressed={period === 'month'} onClick={() => setPeriod('month')}>
                {t(me.periodMonthly)}
              </button>
              <button type="button" aria-pressed={period === 'year'} onClick={() => setPeriod('year')}>
                {t(me.periodYearly)} {!yearlyMore && free > 0 ? <span className="sv">· {t(mt.monthsFree, { count: free })}</span> : null}
              </button>
            </div>
          </div>
        ) : null}
        <article className={`ea-tcard${v.highlight ? ' hl' : ''}`} aria-label={v.name || t(mt.untitled)}>
          {v.highlight ? <span className="ea-tcard-hl">{v.highlight}</span> : null}
          <h3>{v.name || t(mt.untitled)}</h3>
          <div className="flex flex-col gap-0.5">
            <span className="flex flex-wrap items-baseline gap-1">
              <span className="ea-tcard-amt">{amount ? money(amount, currency) : '—'}</span>
              <span className="ea-tcard-unit">{period === 'year' ? t(mt.aYear) : t(mt.aMonth)}</span>
            </span>
            {period === 'year' && v.yearly ? <span className="ea-tcard-sub">{t(mt.perMonthYearly, { price: money(Math.round(v.yearly / 12), currency) })}</span> : null}
          </div>
          {v.trialDays ? <span className="ea-tcard-trial">{t(mt.trialFree, { days: v.trialDays })}</span> : null}
          {v.benefits.length ? (
            <ul>
              {v.benefits.map((b, i) => (
                <li key={i}>
                  <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                    <path d="m3 8.5 3 3 7-7" />
                  </svg>
                  <span>{b}</span>
                </li>
              ))}
            </ul>
          ) : null}
          {v.group && !v.group.off && (v.group.yearly || v.group.monthly) ? (
            <span className="ea-tcard-team">
              <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                <circle cx="6" cy="5" r="2.5" />
                <path d="M1.5 14a4.5 4.5 0 0 1 9 0M11 3a2.5 2.5 0 0 1 0 5M14.5 14a4 4 0 0 0-3-3.8" />
              </svg>
              <span>
                {t(mt.teamLink, {
                  price: v.group.yearly ? `${money(v.group.yearly, currency)} ${t(mt.aYear)}` : `${money(v.group.monthly ?? 0, currency)} ${t(mt.aMonth)}`,
                })}
              </span>
            </span>
          ) : null}
          <div className="ea-tcard-buy">
            <span className={v.highlight ? 'ea-tcard-btn pri' : 'ea-tcard-btn'}>{v.trialDays ? t(mt.startTrial) : short ? t(mt.become, { article, tier: short }) : t(mt.becomeMember)}</span>
            {amount ? (
              <span className="ea-tcard-fine">
                {v.trialDays
                  ? t(mt.fineTrial, { price: period === 'year' ? `${money(amount, currency)} ${t(mt.aYear)}` : `${money(amount, currency)} ${t(mt.aMonth)}` })
                  : t(mt.fineNoTrial, { amount: money(amount, currency, 'always'), period })}
              </span>
            ) : null}
          </div>
        </article>
        {yearlyMore ? <span className="k">{t(mt.switchNote)}</span> : null}
      </div>
    </section>
  )
}

/** "In Stripe": the product, its prices with their ids and who pays them, legacy prices and where it came from. */
function StripeCard({
  tier,
  v,
  currency,
  account,
  connected,
  changedMonth,
  changedYear,
  membersLine,
}: {
  tier?: Tier
  v: Input
  currency: string
  account: string
  connected: boolean
  changedMonth?: Schemas['EarnPrice']
  changedYear?: Schemas['EarnPrice']
  membersLine: (p?: Schemas['EarnPrice']) => string | null
}) {
  const t = useT()
  const legacy = tier?.prices.filter((p) => p.status === 'legacy') ?? []
  const current = tier?.prices.filter((p) => p.status === 'current') ?? []
  const pending = [
    changedMonth && v.monthly ? { period: 'month' as const, amount: v.monthly } : undefined,
    changedYear && v.yearly ? { period: 'year' as const, amount: v.yearly } : undefined,
  ].filter((x) => x !== undefined)
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mt.stripeTitle)}</h2>
          <span className="k">{!connected ? t(mt.stripeDraft) : tier?.productId ? t(mt.stripeUpdates) : t(mt.stripeCreates)}</span>
        </div>
        {tier?.productUrl ? (
          <ButtonAnchor href={tier.productUrl} size="sm" iconRight="external">
            {t(mt.openInStripe)}
          </ButtonAnchor>
        ) : null}
      </div>
      {tier ? (
        <div className="pad flex flex-col gap-3">
          <dl className="ea-kv">
            <dt>{t(mt.product)}</dt>
            <dd>
              {tier.name} {tier.productId ? <span className="mono k">{tier.productId}</span> : null}
            </dd>
          </dl>
          <div>
            {pending.map((p) => (
              <div key={`new-${p.period}`} className="ea-vline">
                <span className="flex min-w-0 flex-1 flex-wrap items-baseline gap-2">
                  <b className="num">{pricePer(t, p.amount, currency, p.period)}</b>
                </span>
                <span className="flex items-center gap-2">
                  <Badge tone="info" size="sm">
                    {t(mt.badgeNew)}
                  </Badge>
                  <span className="k">{t(mt.forNewMembers)}</span>
                </span>
              </div>
            ))}
            {[...current, ...legacy].map((p) => {
              const replaced = (p.period === 'month' && changedMonth?.id === p.id) || (p.period === 'year' && changedYear?.id === p.id)
              return (
                <div key={p.id} className="ea-vline">
                  <span className="flex min-w-0 flex-1 flex-wrap items-baseline gap-2">
                    <b className="num">{pricePer(t, p.amountCents, p.currency, p.period)}</b>
                    {p.stripeId ? <span className="mono k">{p.stripeId}</span> : null}
                  </span>
                  <span className="flex items-center gap-2">
                    {p.status === 'legacy' || replaced ? (
                      <Badge tone="neutral" size="sm">
                        {t(mt.badgeLegacy)}
                      </Badge>
                    ) : null}
                    <span className="k">{membersLine(p)}</span>
                  </span>
                </div>
              )
            })}
            {[tier.group?.monthly, tier.group?.yearly]
              .filter((p): p is Schemas['EarnPrice'] => p !== undefined)
              .map((p) => (
                <div key={p.id} className="ea-vline">
                  <span className="flex min-w-0 flex-1 flex-wrap items-baseline gap-2">
                    <b className="num">{t(mt.perSeatSlash, { price: money(p.amountCents, p.currency), period: p.period })}</b>
                    {p.stripeId ? <span className="mono k">{p.stripeId}</span> : null}
                  </span>
                  <span className="k">{t(mt.groupPriceRow, { teams: p.paying ? t(mt.groupTeams, { count: p.paying }) : t(mt.groupNoTeams) })}</span>
                </div>
              ))}
          </div>
          <dl className="ea-kv">
            <dt>{t(mt.legacyPrices)}</dt>
            <dd>
              {legacy.length || pending.length
                ? [...legacy, ...[changedMonth, changedYear].filter((x) => x !== undefined)]
                    .map((p) => t(mt.legacyLine, { price: pricePer(t, p.amountCents, p.currency, p.period), count: p.paying }))
                    .join(', ')
                : t(mt.legacyNone)}
            </dd>
            <dt>{t(mt.created)}</dt>
            <dd>
              {tier.origin === 'import' && tier.importedFrom && tier.importedAt
                ? t(mt.createdImport, { platform: platformName(tier.importedFrom), account, date: dayMonthDate(tier.importedAt) })
                : t(mt.createdHere, { date: dayMonthDate(tier.createdAt) })}
            </dd>
          </dl>
        </div>
      ) : null}
    </section>
  )
}

/** A new tier's first save: the product, then each price, in Stripe. */
function SavingCard() {
  const t = useT()
  return (
    <section className="card" aria-live="polite">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mt.savingTitle)}</h2>
          <span className="k">{t(mt.savingHint)}</span>
        </div>
      </div>
      <div className="pad">
        <div className="ea-ev">
          <span className="ic" style={{ background: 'var(--line-2)' }}>
            <span className="ea-spin" aria-hidden="true" />
          </span>
          <div className="tx">
            <span className="font-semibold">{t(mt.stepProductPending)}</span>
          </div>
        </div>
      </div>
    </section>
  )
}

/** A save that didn't reach Stripe for every price: what was created, what wasn't, and Retry. */
function SyncFailedCard({ tier, currency, busy, onRetry }: { tier: Tier; currency: string; busy: boolean; onRetry: () => void }) {
  const t = useT()
  const failedStep = tier.syncSteps.find((s) => s.status === 'failed')
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mt.errorTitle)}</h2>
          <span className="k">{t(mt.errorHint)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {failedStep ? (
          <Note tone="bad">
            <span className="font-semibold">{t(mt.errorWhat, { step: failedStep.step })}</span>
            <div>
              {t(mt.errorText, { error: failedStep.error ?? '' })}
              <div className="mt-2">
                <Button size="sm" variant="primary" loading={busy} onClick={onRetry}>
                  {t(mt.retry)}
                </Button>
              </div>
            </div>
          </Note>
        ) : null}
        <div>
          {tier.syncSteps.map((s) => {
            const done = s.status === 'done'
            const label =
              s.step === 'product'
                ? done
                  ? t(mt.stepProduct)
                  : t(mt.stepProductFailed)
                : s.step === 'month'
                  ? done
                    ? t(mt.stepMonth)
                    : t(mt.stepMonthFailed)
                  : done
                    ? t(mt.stepYear)
                    : t(mt.stepYearFailed)
            const detail = s.step === 'product' ? (s.id ?? '') : pricePer(t, s.amountCents ?? 0, currency, s.step)
            return (
              <div key={s.step} className="ea-ev">
                <span className="ic" style={{ background: done ? 'var(--ok-bg)' : s.status === 'failed' ? 'var(--bad-tint)' : 'var(--line-2)' }}>
                  <Icon name={done ? 'check' : s.status === 'failed' ? 'close' : 'clock'} size={14} strokeWidth={2.2} />
                </span>
                <div className="tx">
                  <span className="font-semibold">{label}</span>
                  <span className="k">{s.status === 'failed' && s.error ? t(mt.stepTries, { detail, error: s.error, tries: s.tries ?? 1 }) : detail}</span>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}

/** Group prices (EarnPlanEditors, M65): per seat for teams, with the seats a team can buy. */
function GroupPricesCard({
  g,
  set,
  tier,
  currency,
  ro,
  major,
  minor,
  host,
  slug,
  error,
}: {
  g: Group
  set: (patch: Partial<Group>) => void
  tier?: Tier
  currency: string
  ro: boolean
  major: (minor: number | undefined) => string
  minor: (s: string) => number
  host: string
  slug: string
  error?: string
}) {
  const t = useT()
  const cur = currency.toUpperCase()
  const teamsLine = (p: Schemas['EarnPrice'] | undefined) => (p && p.paying > 0 ? t(mt.groupTeams, { count: p.paying }) : t(mt.groupNoTeams))
  const rows = [
    { period: 'month' as const, on: g.monthlyOn, amount: g.monthly, line: teamsLine(tier?.group?.monthly) },
    { period: 'year' as const, on: g.yearlyOn, amount: g.yearly, line: teamsLine(tier?.group?.yearly) },
  ]
  const shown = g.yearlyOn && g.yearly ? { price: g.yearly, period: 'year' as const } : g.monthlyOn && g.monthly ? { price: g.monthly, period: 'month' as const } : undefined
  const pageHost = host.replace(/\/upgrade$/, '')
  return (
    <section className="card" aria-labelledby="ea-group">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ea-group">
            {t(mt.groupTitle)}
          </h2>
          <span className="k">{t(mt.groupHint)}</span>
        </div>
      </div>
      {rows.map((r, i) => (
        <div key={r.period} className={`ea-srow${i === 0 ? ' border-t-0' : ''}`}>
          <div className="d">
            <span className="font-semibold">{r.period === 'month' ? t(mt.groupMonthly) : t(mt.groupYearly)}</span>
            <span className="k">{r.line}</span>
          </div>
          <div className="c">
            <SwitchInput
              aria-label={r.period === 'month' ? t(mt.groupOfferMonthly) : t(mt.groupOfferYearly)}
              checked={r.on}
              disabled={ro}
              onChange={(e) => set(r.period === 'month' ? { monthlyOn: e.target.checked } : { yearlyOn: e.target.checked })}
            />
            <div className="row gap-1.5" style={{ flexWrap: 'nowrap' }}>
              <span className="k">$</span>
              <input
                className="in num w-[90px]"
                inputMode="decimal"
                aria-label={r.period === 'month' ? t(mt.groupMonthlyIn, { currency: cur }) : t(mt.groupYearlyIn, { currency: cur })}
                disabled={ro || !r.on}
                value={major(r.amount)}
                onChange={(e) => set(r.period === 'month' ? { monthly: minor(e.target.value) } : { yearly: minor(e.target.value) })}
              />
              <span className="k whitespace-nowrap">{r.period === 'month' ? t(mt.perSeatMonth) : t(mt.perSeatYear)}</span>
            </div>
          </div>
        </div>
      ))}
      <div className="ea-srow">
        <div className="d">
          <span className="font-semibold">{t(mt.groupSeats)}</span>
          <span className="k">{t(mt.groupSeatsHint)}</span>
        </div>
        <div className="c">
          <div className="flex flex-col items-end gap-1">
            <div className="row gap-1.5" style={{ flexWrap: 'nowrap' }}>
              <span className="k">{t(mt.groupFrom)}</span>
              <input
                className="in num w-[70px]"
                inputMode="numeric"
                aria-label={t(mt.groupFewest)}
                aria-invalid={error ? true : undefined}
                aria-describedby={error ? 'ea-group-err' : undefined}
                disabled={ro}
                value={g.seatsMin || ''}
                onChange={(e) => set({ seatsMin: Number(e.target.value.replace(/[^0-9]/g, '')) || 0 })}
              />
              <span className="k">{t(mt.groupTo)}</span>
              <input
                className="in num w-[70px]"
                inputMode="numeric"
                aria-label={t(mt.groupMost)}
                aria-invalid={error ? true : undefined}
                aria-describedby={error ? 'ea-group-err' : undefined}
                disabled={ro}
                value={g.seatsMax || ''}
                onChange={(e) => set({ seatsMax: Number(e.target.value.replace(/[^0-9]/g, '')) || 0 })}
              />
            </div>
            {error ? (
              <span className="err" id="ea-group-err" role="alert">
                {error}
              </span>
            ) : null}
          </div>
        </div>
      </div>
      <div className="pad flex flex-col gap-2.5 border-t border-line-2">
        {shown ? (
          <span className="text-[13px]">
            <span className="k">{t(mt.readersSee)} </span>
            <b>{t(mt.groupSee, { price: money(shown.price, currency), period: shown.period })}</b> · {t(mt.groupSeeRange, { min: count(g.seatsMin), max: count(g.seatsMax) })} ·{' '}
            {t(mt.groupOn)} <span className="mono text-[12.5px]">{`${pageHost}/upgrade/${slug || '…'}/team`}</span>
          </span>
        ) : null}
        <div className="note n" role="note">
          <Icon name="info" size={16} />
          <div>{t(mt.groupNote)}</div>
        </div>
      </div>
    </section>
  )
}
