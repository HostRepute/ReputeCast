import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { $api } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import '@/features/integrations/integrations.css'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { clock, count, dayMonthDate, dayMonthYear, shortDate, trimPct } from '@/lib/format'
import { deniedReason, useCan, useMe, useWorkspace } from '@/lib/session'
import { bareAddress, exampleCode, money, perCall, platformName, redeemedShare, sampleCode, usePool, useRefreshCoupons, type Batch, type Pool, type PoolUse } from './api'
import { m } from './messages'
import { AddCodesDialog, ArchiveDialog, CreateMoreDialog, EditPoolDialog, RemoveUnusedDialog } from './PoolDialogs'
import { discountText } from './PoolsPage'
import './coupons.css'

type TFunc = ReturnType<typeof useT>
type Dialog = 'add' | 'more' | 'remove' | 'archive' | 'edit' | null

const mono = (c: ReactNode[]) => <span className="mono text-[12.5px]">{c}</span>
const bold = (c: ReactNode[]) => <b>{c}</b>

/** Whether a moment falls on today, in the display timezone. */
function today(d: string): boolean {
  return shortDate(d) === shortDate(new Date())
}

/** "Today 10:12", else "Mon 24 Nov 11:00". */
function startedAt(t: TFunc, d: string): string {
  return today(d) ? t(m.todayAt, { time: clock(d) }) : `${shortDate(d)} ${clock(d)}`
}

/** "15% off · all products · no minimum · no end date". */
function discountLine(t: TFunc, p: Pool): string {
  const s = p.settings
  const parts = [discountText(t, p)]
  if (s.kind !== 'free_shipping') {
    const n = s.collectionIds?.length ?? 0
    parts.push(n > 0 ? t(m.collectionsLower, { count: n }) : t(m.allProductsLower))
  }
  parts.push(s.minimumCents ? t(m.minimumOf, { amount: money(s.minimumCents, p.currency) }) : t(m.noMinimum))
  parts.push(s.endsDays ? t(m.endsIn, { count: s.endsDays }) : t(m.noEnd))
  return parts.join(' · ')
}

/** "15% off everything": the header's sentence. */
function discountPhrase(t: TFunc, p: Pool): string {
  const d = discountText(t, p)
  const n = p.settings.collectionIds?.length ?? 0
  if (p.settings.kind === 'free_shipping') return d
  return n > 0 ? t(m.inCollections, { discount: d, count: n }) : t(m.everything, { discount: d })
}

/** What stopped a batch, in the reader's words. */
function failureText(t: TFunc, b: Batch, store: string, retrying: boolean): string {
  const args = { store, created: count(b.created) }
  switch (b.error) {
    case 'unreachable':
      return retrying && b.waitingUntil ? t(m.errRetrying, { ...args, time: clock(b.waitingUntil) }) : t(m.errUnreachable, args)
    case 'not_granted':
      return t(m.errNotGranted, args)
    case 'read_only':
      return t(m.errReadOnly, args)
    case 'not_connected':
      return t(m.errNotConnected, args)
    case 'store_gone':
      return t(m.errStoreGone)
    case 'archived':
      return t(m.errArchived)
    case 'ended':
      return t(m.errEnded)
  }
  return t(m.errUnsupported, args)
}

function useStatusLabel(): (s: string) => [string, Tone] {
  const t = useT()
  return (s) => {
    switch (s) {
      case 'live':
        return [t(m.stLive), 'ok']
      case 'paused':
        return [t(m.stPaused), 'warn']
      case 'approval':
        return [t(m.stApproval), 'info']
      case 'approved':
        return [t(m.stApproved), 'info']
      case 'scheduled':
        return [t(m.stScheduled), 'info']
      case 'sending':
        return [t(m.stSending), 'info']
    }
    return [t(m.stDraft), 'neutral']
  }
}

/** A Liquid tag, read-only, with Copy. */
function LiquidTag({ label, value, copyLabel, children }: { label: string; value: string; copyLabel: string; children?: ReactNode }) {
  const t = useT()
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      toast({ kind: 'success', title: t(common.copied) })
    } catch {
      toast({ kind: 'error', title: value })
    }
  }
  return (
    <label className="lbl">
      {label}
      <span className="ih-inl">
        <input className="in mono text-[12px]" readOnly value={value} aria-label={label} />
        <Button size="sm" aria-label={copyLabel} onClick={() => void copy()}>
          {t(common.copy)}
        </Button>
      </span>
      {children}
    </label>
  )
}

/** A discount code pool (DiscountCodePool, DiscountCodesStates). */
export function PoolPage({ id, adding, create }: { id: string; adding?: boolean; create?: number }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = usePool(id)
  const p = q.data
  const can = useCan('coupons:manage')
  const [dialog, setDialog] = useState<Dialog>(null)
  const [opened, setOpened] = useState(false)
  // A batch that finishes while the page is open says so (DiscountCodesStates · generating).
  const watching = useRef<string | null>(null)
  useEffect(() => {
    if (!p) return
    if (p.running) {
      watching.current = p.running.id
      return
    }
    const id = watching.current
    watching.current = null
    const done = id ? p.batches.find((b) => b.id === id) : undefined
    if (done?.status === 'done')
      toast({ kind: 'success', title: <T msg={m.createdToast} values={{ count: done.created, store: p.store?.name ?? '', name: p.name, free: count(p.free), b: bold }} /> })
  }, [p])
  // The fixes of Send Readiness open a dialog once the pool has loaded.
  if (p && !opened) {
    setOpened(true)
    if (can && adding && p.source === 'upload') setDialog('add')
    else if (can && create && p.source !== 'upload' && !p.running) setDialog('more')
  }
  const crumbs = [
    { label: t(m.crumbCampaigns), to: '/w/$workspace/campaigns' as const, params: { workspace: slug } },
    { label: t(m.title), to: '/w/$workspace/campaigns/codes' as const, params: { workspace: slug } },
    { label: p?.name ?? '…' },
  ]
  return (
    <Page title={p?.name ?? t(m.title)} breadcrumb={crumbs}>
      {q.isPending ? (
        <PoolSkeleton />
      ) : q.isError || !p ? (
        <ErrorPanel title={t(m.poolLoadFailed)} actions={<Button onClick={() => q.refetch()}>{t(common.retry)}</Button>}>
          {t(m.loadFailedDetail)}
        </ErrorPanel>
      ) : (
        <PoolBody p={p} can={can} open={setDialog} />
      )}
      {p && dialog === 'add' ? <AddCodesDialog pool={p} onClose={() => setDialog(null)} /> : null}
      {p && dialog === 'more' ? <CreateMoreDialog pool={p} initial={create} onClose={() => setDialog(null)} /> : null}
      {p && dialog === 'remove' ? <RemoveUnusedDialog pool={p} onClose={() => setDialog(null)} /> : null}
      {p && dialog === 'archive' ? <ArchiveDialog pool={p} onClose={() => setDialog(null)} /> : null}
      {p && dialog === 'edit' ? <EditPoolDialog pool={p} onClose={() => setDialog(null)} /> : null}
    </Page>
  )
}

function PoolBody({ p, can, open }: { p: Pool; can: boolean; open: (d: Dialog) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const me = useMe().data
  const myRole = me?.workspaces.find((w) => w.slug === slug)?.roleName ?? ''
  const reason = can ? undefined : deniedReason('coupons:manage', p.helpers, t(m.manageAction))
  const isStore = p.source !== 'upload'
  const store = p.store?.name ?? ''
  const archived = p.status === 'archived'
  const exportHref = `/api/v1/workspaces/${slug}/coupons/pools/${p.id}/export.csv`
  const action = (label: string, d: Dialog, variant?: 'primary', icon?: 'plus') =>
    reason ? (
      <Button variant={variant} icon={icon} disabledReason={reason}>
        {label}
      </Button>
    ) : (
      <Button variant={variant} icon={icon} disabled={archived && d !== 'edit'} onClick={() => open(d)}>
        {label}
      </Button>
    )
  const token = `coupons.${p.slug}`
  return (
    <>
      <section className="card cp-dhead" aria-label={p.name}>
        <div className="cp-dl">
          <span className="ih-tile lg" aria-hidden="true">
            {isStore ? (p.source === 'woocommerce' ? 'Woo' : 'Sh') : <Icon name="upload" size={20} strokeWidth={1.8} />}
          </span>
          <div className="cp-dt">
            <div className="row" style={{ gap: 10 }}>
              <h2 className="cp-ttl">{p.name}</h2>
              {archived ? <Badge>{t(m.statusArchived)}</Badge> : <Badge tone="ok" dot>{t(m.statusActive)}</Badge>}
            </div>
            <span className="text-[13px] text-ink-2">
              {isStore ? (
                <T
                  msg={p.createdBy ? m.headStore : m.headStoreNoOne}
                  values={{ platform: platformName(p.source), store: store || t(m.storeGone), name: p.name, token, who: p.createdBy, date: dayMonthYear(p.createdAt), mono }}
                />
              ) : p.settings.applyUrl ? (
                <T msg={m.headUploadApply} values={{ token, address: p.settings.applyUrl, mono }} />
              ) : (
                <T msg={m.headUpload} values={{ token, mono }} />
              )}
            </span>
            <p className="m-0 max-w-[760px] text-[13.5px] text-ink-nav">
              {isStore
                ? t(p.onePerContact ? m.descStoreOne : m.descStoreMany, { discount: discountPhrase(t, p), example: exampleCode(p.prefix) })
                : uploadLine(t, p)}
            </p>
          </div>
        </div>
        <div className="cp-dact">
          {isStore ? action(t(m.createMore), 'more', 'primary', 'plus') : action(t(m.addCodes), 'add', 'primary', 'plus')}
          {reason ? (
            <Button icon="download" disabledReason={reason}>
              {t(m.export)}
            </Button>
          ) : (
            <ButtonAnchor icon="download" href={exportHref} download="">
              {t(m.export)}
            </ButtonAnchor>
          )}
          {action(t(m.removeUnused), 'remove')}
          {action(t(m.archive), 'archive')}
        </div>
      </section>

      {!can ? (
        <Note tone="info" className="items-start">
          <b>{t(m.readOnlyTitle, { name: me?.user.name ?? '', role: myRole })}</b>
          <div>{p.helpers.length ? t(m.readOnlyAsk, { helpers: listOr(p.helpers) }) : t(m.readOnly)}</div>
        </Note>
      ) : null}

      {!archived && p.free < p.lowAt ? (
        <Note tone="warn" className="items-center">
          <span>
            <T msg={p.free <= 0 ? m.outBanner : m.lowBanner} values={{ name: p.name, free: count(p.free), low: count(p.lowAt), b: bold }} />
          </span>
          {can ? (
            <Button size="sm" onClick={() => open(isStore ? 'more' : 'add')} disabled={!!p.running}>
              {isStore ? t(m.createMore) : t(m.addCodes)}
            </Button>
          ) : null}
        </Note>
      ) : null}
      {p.waiting.map((w) => (
        <Note key={w.id} tone="warn">
          <T msg={m.waitingFlow} values={{ flow: w.name, count: w.emails, b: bold }} />
        </Note>
      ))}

      <div className="cp-stats">
        <div className="card cp-stat">
          <span className="k">{t(m.statFree)}</span>
          <span className="v">{count(p.free)}</span>
          <span className="s">{t(isStore ? m.statFreeStore : m.statFreeUpload, { total: count(p.total), low: count(p.lowAt) })}</span>
        </div>
        <div className="card cp-stat">
          <span className="k">{t(m.statSent)}</span>
          <span className="v">{count(p.sent)}</span>
          <span className="s">{p.firstSentAt ? t(m.statSentSince, { date: dayMonthDate(p.firstSentAt) }) : t(m.statSentNone)}</span>
        </div>
        <div className="card cp-stat">
          <span className="k">{t(m.statRedeemed)}</span>
          <span className="v">{count(p.redeemed)}</span>
          <span className="s">{t(isStore ? m.statRedeemedStore : m.statRedeemedUpload, { pct: trimPct(redeemedShare(p)), store })}</span>
        </div>
        <div className="card cp-stat">
          <span className="k">{t(m.statRevenue)}</span>
          <span className="v">{money(p.revenueCents, p.currency)}</span>
          <span className="s">{t(m.statRevenueSub, { orders: p.orders })}</span>
        </div>
      </div>

      <BatchState p={p} can={can} />

      <div className="cp-grid">
        <UsedIn p={p} />
        <section className="card" aria-labelledby="cp-lq-h">
          <div className="sec">
            <div className="flex min-w-0 flex-col gap-0.5">
              <h2 className="h2" id="cp-lq-h">
                {t(m.liquidTitle)}
              </h2>
              <span className="k">{t(m.liquidNote)}</span>
            </div>
          </div>
          <div className="pad flex flex-col gap-3.5">
            <LiquidTag label={t(m.liquidCode)} value={p.tokens.code} copyLabel={t(m.copyCode)} />
            <LiquidTag label={t(m.liquidUrl)} value={p.tokens.url} copyLabel={t(m.copyUrl)}>
              <span className="k font-medium">
                {p.sampleUrl ? <T msg={m.liquidSampleUrl} values={{ url: p.sampleUrl, mono: (c) => <span className="mono">{c}</span> }} /> : t(m.liquidNoUrl)}
              </span>
            </LiquidTag>
            <span className="k">{t(m.liquidSample, { sample: sampleCode(p.prefix) })}</span>
          </div>
        </section>
      </div>

      <div className="cp-grid">
        <section className="card" aria-labelledby="cp-ps-h">
          <div className="sec">
            <div className="flex min-w-0 flex-col gap-0.5">
              <h2 className="h2" id="cp-ps-h">
                {t(m.settingsTitle)}
              </h2>
              <span className="k">{isStore ? t(m.settingsStore, { store }) : t(m.settingsUpload)}</span>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {reason ? (
                <Button size="sm" disabledReason={reason}>
                  {t(m.edit)}
                </Button>
              ) : (
                <Button size="sm" disabled={archived} onClick={() => open('edit')}>
                  {t(m.edit)}
                </Button>
              )}
            </div>
          </div>
          <div className="pad">
            <dl className="kv">
              {isStore ? (
                <>
                  <dt>{t(m.setDiscount)}</dt>
                  <dd>{discountLine(t, p)}</dd>
                  <dt>{t(m.setEach)}</dt>
                  <dd>{t(m.setEachValue)}</dd>
                  <dt>{t(m.setPrefix)}</dt>
                  <dd>{p.prefix ? <span className="mono">{p.prefix}</span> : t(m.setPrefixNone)}</dd>
                </>
              ) : (
                <>
                  <dt>{t(m.setApply)}</dt>
                  <dd>{p.settings.applyUrl ? <span className="mono break-all">{bareAddress(p.settings.applyUrl)}</span> : t(m.setApplyNone)}</dd>
                </>
              )}
              <dt>{t(m.setOnePer)}</dt>
              <dd>{p.onePerContact ? t(m.on) : t(m.off)}</dd>
              <dt>{t(m.setLow)}</dt>
              <dd>{t(m.setLowValue, { low: count(p.lowAt) })}</dd>
              {isStore ? (
                <>
                  <dt>{t(m.setRefill)}</dt>
                  <dd>
                    {p.refill
                      ? p.refillCheckedAt
                        ? t(m.setRefillLive, { low: count(p.lowAt), time: clock(p.refillCheckedAt) })
                        : t(m.setRefillLiveNew, { low: count(p.lowAt) })
                      : t(m.setRefillIdle, { low: count(p.lowAt) })}
                  </dd>
                </>
              ) : null}
            </dl>
          </div>
        </section>
        <Batches p={p} />
      </div>
    </>
  )
}

/** "2,000 codes, uploaded by Linh Nguyen on 3 Nov 2026 · 1,880 free, …". */
function uploadLine(t: TFunc, p: Pool): string {
  const first = [...p.batches].reverse().find((b) => b.kind === 'upload')
  if (!first || p.total === 0) return t(m.descUploadNone)
  return t(m.descUpload, { total: p.total, who: first.by || '—', date: dayMonthYear(first.createdAt), free: count(p.free), sent: count(p.sent), redeemed: count(p.redeemed) })
}

function listOr(names: string[]): string {
  if (names.length < 2) return names.join('')
  return new Intl.ListFormat(document.documentElement.lang || 'en', { type: 'disjunction' }).format(names)
}

/** The batch under way, a store slowing it down, or why the last one failed. */
function BatchState({ p, can }: { p: Pool; can: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshCoupons()
  const store = p.store?.name ?? ''
  const retry = $api.useMutation('post', '/api/v1/workspaces/{workspace}/coupons/pools/{pool}/generate')
  const b = p.running ?? (p.batch?.status === 'failed' ? p.batch : undefined)
  if (!b) return null
  const failing = b.status === 'failed' || (b.status === 'running' && !!b.error)
  if (failing) {
    const missing = b.requested - b.created - b.failed
    return (
      <section className="card" aria-labelledby="cp-err-h">
        <div className="pad">
          <div className="note b flex-wrap" role="alert">
            <Icon name="alertCircle" size={16} className="mt-0.5 flex-none text-bad-strong" />
            <div className="min-w-0 flex-[1_1_360px]">
              <b id="cp-err-h">{t(m.errTitle, { missing: count(missing), requested: count(b.requested), store })}</b> {failureText(t, b, store, b.status === 'running')}
            </div>
            <div className="flex flex-wrap gap-2">
              {b.status === 'failed' && can && b.error !== 'store_gone' && b.error !== 'ended' && b.error !== 'archived' ? (
                <Button
                  size="sm"
                  loading={retry.isPending}
                  onClick={() =>
                    void retry
                      .mutateAsync({ params: { path: { workspace: slug, pool: p.id } }, body: { count: Math.min(Math.max(missing, 1), 50000) } })
                      .then(() => refresh())
                      .catch((e: unknown) => toast({ kind: 'error', title: String((e as Error).message) }))
                  }
                >
                  {t(m.retryNow)}
                </Button>
              ) : null}
              {p.store ? (
                <ButtonLink size="sm" to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: slug, storeId: p.store.id }}>
                  {t(m.openStore, { store })}
                </ButtonLink>
              ) : null}
            </div>
          </div>
        </div>
      </section>
    )
  }
  const slow = b.waitingUntil && new Date(b.waitingUntil).getTime() > Date.now()
  const share = b.requested > 0 ? Math.min(100, Math.round((b.created / b.requested) * 100)) : 0
  return (
    <section className="card" aria-labelledby="cp-bt-h">
      <div className="pad cp-prog" role="status">
        <div className="cp-row1">
          <span className="inline-flex items-center gap-2">
            <span className="spin sm" aria-hidden="true" />
            <b id="cp-bt-h" className="font-semibold">
              {t(m.progTitle, { requested: count(b.requested), store, done: count(b.created) })}
            </b>
          </span>
          <span className="k">{b.reason === 'refill' ? t(m.progRefill, { time: clock(b.createdAt) }) : t(m.progBy, { time: clock(b.createdAt), who: b.by ?? '' })}</span>
        </div>
        <div className="bar h-2.5" aria-hidden="true">
          <i style={{ width: `${share}%`, background: 'var(--accent)' }} />
        </div>
        <span className="k">{slow ? t(m.progSlow, { store }) : t(m.progNote, { store, per: perCall(p.store?.platform) })}</span>
      </div>
    </section>
  )
}

function UsedIn({ p }: { p: Pool }) {
  const t = useT()
  const { slug } = useWorkspace()
  const label = useStatusLabel()
  const campaigns = p.usedIn.filter((u) => u.kind === 'campaign')
  const flows = p.usedIn.filter((u) => u.kind === 'automation')
  const templates = p.usedIn.filter((u) => u.kind === 'template')
  const group = (title: string, uses: PoolUse[], none: string, row: (u: PoolUse) => ReactNode) => (
    <>
      <div className="ih-phd">
        {title}
        <span className="k">{count(uses.length)}</span>
      </div>
      {uses.length === 0 ? (
        <div className="ih-prow">
          <div className="ih-pl">
            <span className="ih-pd">{none}</span>
          </div>
        </div>
      ) : (
        uses.map(row)
      )}
    </>
  )
  const how = (u: PoolUse) => (u.block ? t(m.useBlock) : t(m.useLiquid))
  return (
    <section className="card" aria-labelledby="cp-ui-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cp-ui-h">
            {t(m.usedTitle)}
          </h2>
          <span className="k">
            <T msg={m.usedNote} values={{ token: `coupons.${p.slug}`, mono: (c) => <span className="mono">{c}</span> }} />
          </span>
        </div>
      </div>
      <div className="pad ih-prl pt-1">
        {group(t(m.usedCampaigns), campaigns, t(m.noCampaigns), (u) => {
          const [st, tone] = label(u.status ?? 'draft')
          return (
            <div className="ih-prow" key={`c-${u.id}`}>
              <div className="ih-pl">
                <span className="ih-pn">
                  <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: u.id }} className="text-ink no-underline">
                    {u.name}
                  </Link>
                </span>
                <span className="ih-pd">{[u.lists, t(m.useRecipients, { count: u.recipients }), how(u)].filter(Boolean).join(' · ')}</span>
              </div>
              <div className="ih-st">
                <Badge tone={tone} dot={tone === 'ok'}>
                  {st}
                </Badge>
              </div>
            </div>
          )
        })}
        {group(t(m.usedAutomations), flows, t(m.noAutomations), (u) => {
          const [st, tone] = label(u.live ? 'live' : (u.status ?? 'draft'))
          return (
            <div className="ih-prow" key={`a-${u.emailId}`}>
              <div className="ih-pl">
                <span className="ih-pn">
                  <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: u.id }} className="text-ink no-underline">
                    {t(m.flowEmail, { flow: u.name, email: u.emailName ?? '' })}
                  </Link>
                </span>
                <span className="ih-pd">{u.live ? (p.source === 'upload' ? t(m.flowLive) : t(m.flowKeptUp)) : t(m.flowNotLive)}</span>
              </div>
              <div className="ih-st">
                <Badge tone={tone} dot={tone === 'ok'}>
                  {st}
                </Badge>
              </div>
            </div>
          )
        })}
        {group(t(m.usedTemplates), templates, t(m.noTemplates), (u) => (
          <div className="ih-prow" key={`t-${u.id}`}>
            <div className="ih-pl">
              <span className="ih-pn">{u.name}</span>
            </div>
          </div>
        ))}
      </div>
    </section>
  )
}

function Batches({ p }: { p: Pool }) {
  const t = useT()
  const store = p.store?.name ?? ''
  const how = (b: Batch): [string, string] => {
    const sub =
      b.reason === 'refill'
        ? (b.automation ?? '')
        : b.kind === 'generate' && b.reason === 'pool' && b.failed > 0
          ? t(m.alreadyIn, { count: count(b.failed), store })
          : (b.by ?? '')
    if (b.kind === 'upload') return [t(m.reasonUpload), b.failed > 0 ? t(m.alreadyInPool, { count: count(b.failed) }) : (b.by ?? '')]
    return [b.reason === 'refill' ? t(m.reasonRefill) : b.reason === 'pool' ? t(m.reasonPool) : t(m.reasonRequest), sub]
  }
  const status = (b: Batch) => {
    if (b.status === 'running') {
      if (b.error && b.waitingUntil) {
        return (
          <>
            <Badge tone="warn" dot>
              {t(m.batchRunning)}
            </Badge>
            <div className="k">{t(m.batchRetryAt, { store, time: clock(b.waitingUntil) })}</div>
          </>
        )
      }
      return (
        <Badge tone="info" dot>
          {t(m.batchRunning)}
        </Badge>
      )
    }
    if (b.status === 'failed') {
      return (
        <Badge tone="bad" dot>
          {t(m.batchFailed)}
        </Badge>
      )
    }
    return (
      <Badge tone="ok" dot>
        {t(m.batchDone)}
      </Badge>
    )
  }
  return (
    <section className="card" aria-labelledby="cp-bh-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cp-bh-h">
            {t(m.batchesTitle)}
          </h2>
          <span className="k">{t(m.batchesNote)}</span>
        </div>
      </div>
      {p.batches.length === 0 ? (
        <p className="k m-0 px-4.5 py-4">{t(m.noBatches)}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="tbl" aria-labelledby="cp-bh-h">
            <thead>
              <tr>
                <Th>{t(m.thStarted)}</Th>
                <Th>{t(m.thHow)}</Th>
                <Th align="right">
                  {t(m.thAsked)}
                </Th>
                <Th align="right">
                  {t(m.thCreated)}
                </Th>
                <Th align="right">
                  {t(m.thRefused)}
                </Th>
                <Th>{t(m.thStatus)}</Th>
              </tr>
            </thead>
            <tbody>
              {p.batches.map((b) => {
                const [what, sub] = how(b)
                return (
                  <tr key={b.id}>
                    <td className="nw">{startedAt(t, b.createdAt)}</td>
                    <td className="nw">
                      {what}
                      {sub ? <div className="k">{sub}</div> : null}
                    </td>
                    <td className="r">
                      <span className="num">{count(b.requested)}</span>
                    </td>
                    <td className="r">
                      <span className="num">{count(b.created)}</span>
                    </td>
                    <td className="r">
                      <span className="num">{count(b.failed)}</span>
                    </td>
                    <td>{status(b)}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

/** Skeletons shaped like the pool. */
function PoolSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col gap-4" aria-busy="true" aria-label={t(m.loading)}>
      <section className="card flex items-start gap-4 p-5">
        <Skeleton w={52} h={52} r={12} />
        <div className="flex flex-1 flex-col gap-2">
          <Skeleton w="40%" h={18} />
          <Skeleton w="70%" h={12} />
          <Skeleton w="55%" h={12} />
        </div>
      </section>
      <div className="cp-stats">
        {[0, 1, 2, 3].map((i) => (
          <div className="card cp-stat" key={i}>
            <Skeleton w={60} h={12} />
            <Skeleton w={90} h={22} />
            <Skeleton w={120} h={10} />
          </div>
        ))}
      </div>
    </div>
  )
}
