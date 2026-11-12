import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Th } from '@/components/ui/Table'
import '@/features/integrations/integrations.css'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, trimPct } from '@/lib/format'
import { deniedReason, useCan, useWorkspace } from '@/lib/session'
import { bareAddress, money, platformName, redeemedShare, usePools, useSources, type PoolItem } from './api'
import { m } from './messages'
import { NewPoolDialog } from './NewPoolDialog'
import './coupons.css'

type TFunc = ReturnType<typeof useT>

/** "15% off", "$10.00 off" or "Free shipping": what a store pool's code takes off. */
export function discountText(t: TFunc, p: Pick<PoolItem, 'settings' | 'currency'>): string {
  const s = p.settings
  if (s.kind === 'percentage') return t(m.percentOff, { value: String(s.value ?? 0) })
  if (s.kind === 'fixed_amount') return t(m.amountOff, { amount: money(s.value ?? 0, p.currency) })
  if (s.kind === 'free_shipping') return t(m.freeShipping)
  return ''
}

/** The facts under a pool's name: its Liquid, what it takes off, where it applies. */
function poolFacts(t: TFunc, p: PoolItem): ReactNode {
  const facts: string[] = []
  if (p.source === 'upload') {
    if (p.settings.applyUrl) facts.push(t(m.appliesAt, { address: bareAddress(p.settings.applyUrl) }))
  } else {
    facts.push(discountText(t, p))
  }
  if (p.onePerContact && p.source !== 'upload') facts.push(t(m.onePerContactShort))
  return (
    <>
      <span className="mono">{`coupons.${p.slug}`}</span>
      {facts.map((f) => ` · ${f}`).join('')}
    </>
  )
}

/** The Source badge: the store's platform and name, or Uploaded. */
export function SourceChip({ p }: { p: Pick<PoolItem, 'source' | 'store'> }) {
  const t = useT()
  if (p.source === 'upload') {
    return (
      <span className="ih-mby">
        <span className="ih-mt" aria-hidden="true">
          <Icon name="upload" size={10} strokeWidth={2} />
        </span>
        {t(m.sourceUpload)}
      </span>
    )
  }
  return (
    <span className="ih-mby">
      <span className="ih-mt" aria-hidden="true">
        {p.source === 'woocommerce' ? 'Wo' : 'Sh'}
      </span>
      {t(m.sourceStore, { platform: platformName(p.source), store: p.store?.name ?? '' })}
    </span>
  )
}

function StatusCell({ p }: { p: PoolItem }) {
  const t = useT()
  return p.status === 'archived' ? <Badge>{t(m.statusArchived)}</Badge> : <Badge tone="ok" dot>{t(m.statusActive)}</Badge>
}

/** Campaigns › Discount codes (DiscountCodes, DiscountCodesStates). */
export function PoolsPage({ status, creating }: { status: 'active' | 'archived'; creating?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const q = usePools()
  const can = useCan('coupons:manage')
  const [open, setOpen] = useState(!!creating && can)
  const helpers = q.data?.helpers ?? []
  const reason = can ? undefined : deniedReason('coupons:manage', helpers, t(m.newPoolAction))
  const items = q.data?.items ?? []
  const active = items.filter((p) => p.status === 'active')
  const archived = items.filter((p) => p.status === 'archived')
  const shown = status === 'archived' ? archived : active
  const example = `coupons.${active[0]?.slug ?? 'welcome15'}.code`
  const setStatus = (s: 'active' | 'archived') =>
    void navigate({ to: '/w/$workspace/campaigns/codes', params: { workspace: slug }, search: s === 'active' ? {} : { status: s }, replace: true })
  const newPool = reason ? (
    <Button variant="primary" icon="plus" disabledReason={reason}>
      {t(m.newPool)}
    </Button>
  ) : (
    <Button variant="primary" icon="plus" onClick={() => setOpen(true)}>
      {t(m.newPool)}
    </Button>
  )
  const crumbs = [{ label: t(m.crumbCampaigns), to: '/w/$workspace/campaigns' as const, params: { workspace: slug } }, { label: t(m.title) }]
  return (
    <Page title={t(m.title)} breadcrumb={crumbs} actions={items.length > 0 ? newPool : undefined}>
      {q.isPending ? (
        <PoolsSkeleton />
      ) : q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {t(m.loadFailedDetail)}
        </ErrorPanel>
      ) : items.length === 0 ? (
        <EmptyPools action={newPool} />
      ) : (
        <>
          <p className="m-0 max-w-[760px] text-[14px] text-ink-2">
            <T msg={m.lead} values={{ example, mono: (c) => <span className="mono text-[13px]">{c}</span> }} />
          </p>
          <section className="card" aria-labelledby="cp-pools-h">
            <div className="sec">
              <div className="flex min-w-0 flex-col gap-0.5">
                <h2 className="h2" id="cp-pools-h">
                  {t(m.poolsTitle, { count: count(items.length) })}
                </h2>
                <span className="k">{t(m.poolsNote)}</span>
              </div>
            </div>
            <div className="cp-fchips" role="group" aria-label={t(m.filterLabel)}>
              <button type="button" className="cp-fchip" aria-pressed={status === 'active'} onClick={() => setStatus('active')}>
                {t(m.filterActive)} <b>{count(active.length)}</b>
              </button>
              <button type="button" className="cp-fchip" aria-pressed={status === 'archived'} onClick={() => setStatus('archived')}>
                {t(m.filterArchived)} <b>{count(archived.length)}</b>
              </button>
            </div>
            {shown.length === 0 ? (
              <p className="k m-0 px-4.5 py-4">{t(m.archivedNone)}</p>
            ) : (
              <>
                <div className="ih-no-phone">
                  <div className="overflow-x-auto">
                    <table className="tbl" aria-label={t(m.tableLabel)}>
                      <thead>
                        <tr>
                          <Th>{t(m.thName)}</Th>
                          <Th>{t(m.thSource)}</Th>
                          <Th align="right">
                            {t(m.thFree)}
                          </Th>
                          <Th align="right">
                            {t(m.thSent)}
                          </Th>
                          <Th align="right">
                            {t(m.thRedeemed)}
                          </Th>
                          <Th align="right">
                            {t(m.thRevenue)}
                          </Th>
                          <Th>{t(m.thStatus)}</Th>
                        </tr>
                      </thead>
                      <tbody>
                        {shown.map((p) => (
                          <tr key={p.id}>
                            <td className="min-w-[220px] whitespace-normal">
                              <Link to="/w/$workspace/campaigns/codes/$poolId" params={{ workspace: slug, poolId: p.id }} className="font-semibold text-ink no-underline">
                                {p.name}
                              </Link>
                              <div className="k">{poolFacts(t, p)}</div>
                            </td>
                            <td>
                              <SourceChip p={p} />
                            </td>
                            <td className="r">
                              <span className="num">{count(p.free)}</span>
                            </td>
                            <td className="r">
                              <span className="num">{count(p.sent)}</span>
                            </td>
                            <td className="r">
                              <span className="num">{count(p.redeemed)}</span>
                              <div className="k">{t(m.ofSent, { pct: trimPct(redeemedShare(p)) })}</div>
                            </td>
                            <td className="r">
                              <span className="num">{money(p.revenueCents, p.currency)}</span>
                            </td>
                            <td>
                              <StatusCell p={p} />
                              {p.running ? <div className="k">{t(m.creatingShort, { requested: count(p.running.requested), done: count(p.running.created) })}</div> : null}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
                <div className="ih-only-phone">
                  {shown.map((p) => (
                    <div className="cp-mrow" key={p.id}>
                      <div className="flex items-start justify-between gap-2.5">
                        <Link to="/w/$workspace/campaigns/codes/$poolId" params={{ workspace: slug, poolId: p.id }} className="font-semibold text-ink no-underline">
                          {p.name}
                        </Link>
                        <StatusCell p={p} />
                      </div>
                      <div>
                        <SourceChip p={p} />
                      </div>
                      <span className="num text-[13px]">
                        <T
                          msg={m.phoneNumbers}
                          values={{ free: count(p.free), sent: count(p.sent), redeemed: count(p.redeemed), revenue: money(p.revenueCents, p.currency), b: (c) => <b>{c}</b> }}
                        />
                      </span>
                      {p.running ? <span className="k">{t(m.creatingShort, { requested: count(p.running.requested), done: count(p.running.created) })}</span> : null}
                    </div>
                  ))}
                </div>
              </>
            )}
            <div className="cp-foot">
              <span className="k">
                {t(m.footSummary, {
                  count: items.length,
                  free: count(active.reduce((n, p) => n + p.free, 0)),
                  sent: count(items.reduce((n, p) => n + p.sent, 0)),
                })}
              </span>
              <span className="k">{t(m.footAlerts)}</span>
            </div>
          </section>
        </>
      )}
      {open ? <NewPoolDialog open onOpenChange={setOpen} /> : null}
    </Page>
  )
}

/** "No discount code pools yet": a sentence and New pool. */
function EmptyPools({ action }: { action: ReactNode }) {
  const t = useT()
  const sources = useSources()
  const store = sources.data?.items.find((s) => s.can)?.name ?? sources.data?.items[0]?.name
  return (
    <EmptyState
      className="border-dashed"
      title={t(m.emptyTitle)}
      icon={
        <span className="ih-tile" aria-hidden="true">
          <Icon name="tag" size={18} />
        </span>
      }
      action={action}
    >
      {store ? t(m.emptyStore, { store }) : t(m.emptyNoStore)}
    </EmptyState>
  )
}

/** Skeletons shaped like the list. */
function PoolsSkeleton() {
  const t = useT()
  return (
    <section className="card" aria-busy="true" aria-label={t(m.loading)}>
      <div className="sec">
        <Skeleton w={120} h={14} />
      </div>
      {[0, 1, 2].map((i) => (
        <div key={i} className="flex items-center gap-3 border-b border-line-2 px-4.5 py-4 last:border-b-0">
          <Skeleton w="40%" h={12} />
          <Skeleton w={90} h={20} r={999} />
          <Skeleton w={48} h={12} />
          <Skeleton w={48} h={12} />
          <Skeleton w={48} h={12} />
          <Skeleton w={60} h={20} r={999} />
        </div>
      ))}
    </section>
  )
}

