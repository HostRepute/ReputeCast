import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Select } from '@/components/ui/Field'
import { Banner, Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { count, orList } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { EARN, EarnPage, copyText } from './EarnPage'
import { money, pricePer } from './format'
import { me } from './messages'
import { OffersSection } from './offers/OffersSection'
import { mp } from './plans.messages'

type Plans = Schemas['EarnPlans']
type Tier = Schemas['EarnTier']

/** Earn › Plans & offers (EarnPlans): the upgrade page, the Free card and the tiers. Offers come after them. */
export function PlansPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const path = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', `${EARN}/plans`, path)
  const pay = $api.useQuery('get', `${EARN}/payments`, path)
  return (
    <EarnPage tab="plans" title={t(mp.tiersTitle, { count: q.data?.tiers.length ?? 0 })}>
      {q.error ? (
        <ErrorPanel title={t(mp.errorTitle)} actions={<Button onClick={() => void q.refetch()}>{t(me.retry)}</Button>}>
          {errorMessage(q.error)} {t(mp.errorText)}
        </ErrorPanel>
      ) : !q.data ? (
        <div className="flex flex-col gap-4" aria-busy="true">
          <div className="card pad">
            <Skeleton w="60%" h={40} />
          </div>
          <div className="ea-tiers">
            <div className="card pad">
              <Skeleton w="100%" h={220} />
            </div>
            <div className="card pad">
              <Skeleton w="100%" h={220} />
            </div>
          </div>
        </div>
      ) : (
        <PlansBody plans={q.data} unmapped={pay.data?.unmapped ?? []} />
      )}
    </EarnPage>
  )
}

function PlansBody({ plans, unmapped }: { plans: Plans; unmapped: Schemas['EarnUnmappedPrice'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [mapping, setMapping] = useState<Schemas['EarnUnmappedPrice']>()
  const ro = !plans.canEdit
  const { me: who } = useWorkspace()
  const user = useMe().data
  const freeList = plans.free.list?.name ?? ''
  const tierCount = plans.tiers.filter((x) => x.visibility !== 'archived').length + (plans.free.list ? 1 : 0)
  return (
    <div className="flex flex-col gap-5">
      {ro ? (
        // Read-only (EarnPlansStates): who is looking, and who can change plans.
        <div className="note n" role="note">
          <Icon name="info" size={16} />
          <div className="flex min-w-0 flex-col gap-1">
            <span className="font-semibold">{t(mp.readOnlyTitle, { name: user?.user.name ?? '', role: roleName({ key: who.workspace.roleKey, name: who.workspace.roleName }) })}</span>
            <div>
              <T
                msg={plans.helpers.length ? mp.readOnly : mp.readOnlyNoHelpers}
                values={{ permission: <span className="mono">earn:edit</span>, helpers: orList(plans.helpers) }}
              />
            </div>
          </div>
        </div>
      ) : null}
      {unmapped[0] && plans.canEdit ? (
        <Banner
          tone="warn"
          title={t(mp.unmappedTitle, { count: unmapped.length })}
          actions={
            <Button size="sm" onClick={() => setMapping(unmapped[0])}>
              {t(mp.mapPrice)}
            </Button>
          }
        >
          {t(mp.unmappedText, { price: pricePer(t, unmapped[0].amountCents, unmapped[0].currency, unmapped[0].period), id: unmapped[0].priceId })}
        </Banner>
      ) : null}
      <UpgradePageCard plans={plans} />
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(mp.tiersTitle, { count: tierCount })}</h2>
          <span className="k">{t(mp.tiersCaption)}</span>
        </div>
        {plans.canEdit ? (
          <ButtonLink to="/w/$workspace/earn/plans/tiers/new" params={{ workspace: slug }} variant="primary" icon="plus">
            {t(mp.newTier)}
          </ButtonLink>
        ) : null}
      </div>
      {plans.free.list ? <FreeCard plans={plans} /> : null}
      {plans.tiers.length === 0 ? (
        <section className="card pad flex flex-col items-start gap-3">
          <h3 className="h2">{t(mp.noTiersTitle)}</h3>
          <p className="m-0 max-w-[640px] text-ink-2">{t(mp.noTiersText, { list: freeList || 'newsletter' })}</p>
          <div className="flex flex-wrap gap-2">
            {plans.canEdit ? (
              <ButtonLink to="/w/$workspace/earn/plans/tiers/new" params={{ workspace: slug }} variant="primary" icon="plus">
                {t(mp.newTier)}
              </ButtonLink>
            ) : null}
            {!plans.connected ? (
              <ButtonLink to="/w/$workspace/earn/connect" params={{ workspace: slug }}>
                {t(mp.connectStripe)}
              </ButtonLink>
            ) : null}
          </div>
          {!plans.connected ? <span className="k">{t(mp.noTiersConnect)}</span> : null}
        </section>
      ) : (
        <div className="ea-tiers">
          {plans.tiers.map((tier) => (
            <TierCard key={tier.id} tier={tier} plans={plans} />
          ))}
        </div>
      )}
      {/* Offers (X17) render after the tiers. */}
      <OffersSection />
      {mapping ? <MapPriceDialog price={mapping} tiers={plans.tiers} onClose={() => setMapping(undefined)} /> : null}
    </div>
  )
}

function UpgradePageCard({ plans }: { plans: Plans }) {
  const t = useT()
  const { slug } = useWorkspace()
  const u = plans.upgradePage
  return (
    <section className="card ea-upg" aria-labelledby="ea-upg-title">
      <div className="ea-upg-main">
        <span className="ea-upg-ico" aria-hidden="true">
          <Icon name="globe" size={20} />
        </span>
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="h2 m-0" id="ea-upg-title">
              {t(mp.upgradeTitle)}
            </h2>
            {u.live ? (
              <Badge tone={u.testMode ? 'purple' : 'ok'} dot>
                {u.testMode ? t(mp.upgradeTest) : t(mp.upgradeLive)}
              </Badge>
            ) : (
              <Badge tone="neutral">{t(mp.upgradeNotLive)}</Badge>
            )}
          </div>
          <span className="text-[13px] text-ink-2">
            <span className="mono">{u.host}</span> · {u.live ? t(mp.upgradeLine, { trials: u.trials, paid: u.paidAtOnce }) : t(mp.upgradeNotLiveLine)}
          </span>
        </div>
      </div>
      <dl className="ea-upg-stats m-0">
        <div>
          <dt className="k">{t(mp.visits)}</dt>
          <dd className="v m-0">{count(u.visits)}</dd>
        </div>
        <div>
          <dt className="k">{t(mp.checkoutsStarted)}</dt>
          <dd className="v m-0">{count(u.checkoutsStarted)}</dd>
        </div>
        <div>
          <dt className="k">{t(mp.completed)}</dt>
          <dd className="v m-0">{count(u.completed)}</dd>
        </div>
      </dl>
      <div className="flex flex-wrap gap-2">
        <ButtonLink to="/w/$workspace/earn/plans/upgrade-page" params={{ workspace: slug }} size="sm">
          {plans.canEdit ? t(mp.editPage) : t(mp.view)}
        </ButtonLink>
        <Button size="sm" icon="copy" onClick={() => void copyText(u.url, t, toast)}>
          {t(mp.copyLink)}
        </Button>
        <ButtonAnchor size="sm" href={`${u.url}${u.url.includes('?') ? '&' : '?'}preview=test`}>
          {t(mp.previewTest)}
        </ButtonAnchor>
      </div>
    </section>
  )
}

function FreeCard({ plans }: { plans: Plans }) {
  const t = useT()
  const { slug } = useWorkspace()
  const f = plans.free
  const list = f.list?.name ?? ''
  return (
    <section className="card ea-free" aria-label={t(mp.freeName)}>
      <div className="flex min-w-0 flex-col items-start gap-1.5">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="h2 m-0">{t(mp.freeName)}</h3>
          <Badge tone="neutral">{t(mp.freeFirst)}</Badge>
        </div>
        <span className="k">{t(mp.freeList, { list })}</span>
        <ButtonLink to="/w/$workspace/earn/plans/upgrade-page" params={{ workspace: slug }} size="sm" className="mt-2">
          {plans.canEdit ? t(mp.editCardText) : t(mp.viewCard)}
        </ButtonLink>
      </div>
      <dl className="ea-kv">
        <dt>{t(mp.cardText)}</dt>
        <dd>{f.text || '—'}</dd>
        <dt>{t(mp.button)}</dt>
        <dd>{t(mp.buttonValue)}</dd>
        <dt>{t(mp.readers)}</dt>
        <dd>{t(mp.readersValue, { list })}</dd>
      </dl>
    </section>
  )
}

function yearlyMonthsFree(m?: Schemas['EarnPrice'], y?: Schemas['EarnPrice']): number {
  if (!m || !y || m.amountCents <= 0) return 0
  const n = Math.round((m.amountCents * 12 - y.amountCents) / m.amountCents)
  return n > 0 ? n : 0
}

function TierCard({ tier, plans }: { tier: Tier; plans: Plans }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const sync = $api.useMutation('post', `${EARN}/tiers/{id}/sync`)
  const prices = [tier.monthly, tier.yearly].filter(Boolean) as Schemas['EarnPrice'][]
  const free = yearlyMonthsFree(tier.monthly, tier.yearly)
  const product = tier.syncStatus === 'synced' ? t(mp.productInStripe) : tier.syncStatus === 'failed' ? t(mp.productFailed) : t(mp.productDraft)
  const link = tier.visibility === 'hidden' ? tier.hiddenLink : plans.upgradePage.url
  const retry = () =>
    sync.mutate(
      { params: { path: { workspace: slug, id: tier.id } } },
      {
        onSuccess: (r) => {
          void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
          toast(r.syncError ? { kind: 'error', title: t(mp.savedSyncFailed, { tier: tier.name }), detail: r.syncError } : { kind: 'success', title: t(mp.savedStripe, { tier: tier.name }) })
        },
        onError: (e) => toast({ kind: 'error', title: t(mp.savedSyncFailed, { tier: tier.name }), detail: errorMessage(e) }),
      },
    )
  return (
    <article className="card ea-tier" aria-labelledby={`tier-${tier.id}`}>
      <div className="ea-tier-head">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h3 className="h2 m-0" id={`tier-${tier.id}`}>
            {tier.name}
          </h3>
          <span className="k">
            {t(mp.benefits, { count: tier.benefits.length })} · {product}
          </span>
        </div>
        <div className="flex flex-wrap justify-end gap-1.5">
          <Badge tone={tier.visibility === 'public' ? 'ok' : 'neutral'}>
            {tier.visibility === 'public' ? t(mp.visPublic) : tier.visibility === 'hidden' ? t(mp.visHidden) : t(mp.visArchived)}
          </Badge>
          {tier.highlight ? <Badge tone="purple">{tier.highlight}</Badge> : null}
        </div>
      </div>
      <div className="ea-tier-body">
        <dl className="ea-kv">
          <dt>{t(mp.prices)}</dt>
          <dd>
            {prices.length ? prices.map((p) => pricePer(t, p.amountCents, p.currency, p.period)).join(' · ') : <span className="k">{t(mp.noPrice)}</span>}
            {free > 0 ? <span className="sub">{t(mp.monthsFree, { count: free })}</span> : null}
          </dd>
          <dt>{t(mp.trial)}</dt>
          <dd>{tier.trialDays ? t(mp.trialDays, { days: tier.trialDays }) : t(mp.trialNone)}</dd>
          <dt>{t(mp.lists)}</dt>
          <dd>{tier.lists.length ? tier.lists.map((l) => l.name).join(', ') : t(mp.listsNone)}</dd>
          <dt>{t(mp.access)}</dt>
          <dd>{tier.includes.length ? t(mp.rankIncludes, { rank: tier.accessRank, tiers: tier.includes.join(', ') }) : t(mp.rank, { rank: tier.accessRank })}</dd>
          <dt>{t(mp.paidMembers)}</dt>
          <dd>
            {count(tier.paying)}
            {tier.monthly && tier.yearly ? <span className="sub">{t(mp.byPeriod, { monthly: tier.payingMonthly, yearly: tier.payingYearly })}</span> : null}
          </dd>
          <dt>{t(mp.mrr)}</dt>
          <dd className="font-bold">{money(tier.mrrCents, plans.currency, 'never')}</dd>
        </dl>
        {tier.syncStatus === 'failed' ? (
          <Note tone="bad" className="mt-3">
            {tier.syncSteps.find((s) => s.status === 'failed')?.error ?? t(mp.productFailed)}
          </Note>
        ) : null}
      </div>
      <div className="ea-tier-foot">
        <ButtonLink to="/w/$workspace/earn/plans/tiers/$tierId" params={{ workspace: slug, tierId: tier.id }} size="sm">
          {plans.canEdit ? t(mp.edit) : t(mp.view)}
        </ButtonLink>
        {plans.canEdit ? (
          <Button size="sm" icon="copy" onClick={() => void copyText(link, t, toast)}>
            {t(mp.copyLink)}
          </Button>
        ) : null}
        {plans.canEdit && tier.syncStatus === 'failed' && plans.connected ? (
          <Button size="sm" icon="refresh" onClick={retry} disabled={sync.isPending}>
            {t(mp.syncRetry)}
          </Button>
        ) : null}
      </div>
    </article>
  )
}

/** Maps a Stripe price nobody set up here to a tier, as its price or a legacy one. */
function MapPriceDialog({ price, tiers, onClose }: { price: Schemas['EarnUnmappedPrice']; tiers: Tier[]; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [tier, setTier] = useState(tiers[0]?.id ?? '')
  const [legacy, setLegacy] = useState(true)
  const map = $api.useMutation('post', `${EARN}/unmapped-prices/{id}/map`)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mp.mapPrice)}
      description={t(mp.unmappedText, { price: pricePer(t, price.amountCents, price.currency, price.period), id: price.priceId })}
      footer={
        <>
          <Button onClick={onClose}>{t(mp.cancel)}</Button>
          <Button
            variant="primary"
            disabled={!tier || map.isPending}
            onClick={() =>
              map.mutate(
                { params: { path: { workspace: slug, id: price.id } }, body: { tierId: tier, legacy } },
                {
                  onSuccess: () => {
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/payments`] })
                    void qc.invalidateQueries({ queryKey: ['get', `${EARN}/plans`] })
                    onClose()
                  },
                  onError: (e) => toast({ kind: 'error', title: t(mp.mapPrice), detail: errorMessage(e) }),
                },
              )
            }
          >
            {t(mp.mapPrice)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <Field label={t(mp.access)}>
          <Select value={tier} onChange={(e) => setTier(e.target.value)}>
            {tiers.map((x) => (
              <option key={x.id} value={x.id}>
                {x.name}
              </option>
            ))}
          </Select>
        </Field>
        <label className="flex items-center gap-2 text-[13px]">
          <input type="checkbox" checked={legacy} onChange={(e) => setLegacy(e.target.checked)} />
          {t(mp.legacyNote, { count: price.subscriptions, price: pricePer(t, price.amountCents, price.currency, price.period) })}
        </label>
      </div>
    </Dialog>
  )
}
