import { Link, useNavigate } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Bar, KeyValues } from '@/components/ui/Data'
import { Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { Th } from '@/components/ui/Table'
import { AttentionIcon, Tile } from '@/features/integrations/shared'
import '@/features/integrations/integrations.css'
import { T, formatList, useT, type MessageDescriptor } from '@/i18n'
import { cn } from '@/lib/cn'
import { ago, clock, count, dayMonthDate, dayMonthYear, pct } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useMe, useWorkspace } from '@/lib/session'
import { ActivityEntry, ActivityLogDialog, type ActivityContext } from './Activity'
import { money, storefront, storesBase, useRefreshStores, useStore, type StorePage as Page_, type SyncState } from './api'
import { ConnectStoreDialog } from './ConnectStoreDialog'
import { DisconnectStoreDialog } from './DisconnectStoreDialog'
import { cm } from './messages'
import './commerce.css'

type T_ = ReturnType<typeof useT>
type Dialog = 'connect' | 'reconnect' | 'settings' | 'disconnect' | 'log' | null

const mono = (c: ReactNode[]) => <span className="mono text-[12.5px]">{c}</span>
const bold = (c: ReactNode[]) => <b>{c}</b>

/** The status badge of the store sync row. */
export function syncStatus(t: T_, sync: SyncState, platform: string): { tone: Tone; label: string } {
  const p = platform === 'woocommerce' ? 'WooCommerce' : 'Shopify'
  switch (sync.state) {
    case 'connecting':
      return { tone: 'neutral', label: t(cm.statusConnecting) }
    case 'importing': {
      const im = sync.import
      if (!im) return { tone: 'info', label: t(cm.statusImportingOrders, { done: '0', total: '0' }) }
      switch (im.phase) {
        case 'products':
          return { tone: 'info', label: t(cm.statusImportingProducts, { done: count(im.products.done), total: count(im.products.total) }) }
        case 'customers':
          return { tone: 'info', label: t(cm.statusImportingCustomers, { done: count(im.customers.done), total: count(im.customers.total) }) }
        case 'matching':
          return { tone: 'info', label: t(cm.statusMatching) }
      }
      return { tone: 'info', label: t(cm.statusImportingOrders, { done: count(im.orders.done), total: count(im.orders.total) }) }
    }
    case 'webhook_missing':
      return { tone: 'warn', label: t(cm.statusWebhookMissing) }
    case 'rate_limited':
      return { tone: 'warn', label: t(cm.statusRateLimited, { platform: p, time: sync.rateLimitedUntil ? clock(sync.rateLimitedUntil) : '' }) }
    case 'no_plugin':
      return { tone: 'warn', label: t(cm.statusNoPlugin) }
    case 'disconnected':
      return { tone: 'bad', label: t(cm.statusDisconnected) }
  }
  return { tone: 'ok', label: sync.lastEventAt ? t(cm.statusConnectedEvent, { ago: ago(sync.lastEventAt) }) : t(cm.statusConnected) }
}

const capLabel: Record<string, MessageDescriptor> = {
  orders: cm.capOrders,
  products: cm.capProducts,
  customers: cm.capCustomers,
  checkouts: cm.capCheckouts,
  write_back: cm.capWriteBack,
  attribution: cm.capAttribution,
}

/** Settings › Integrations › Shopify · Acme Merch (IntegrationsShopify, CommerceStates). */
export function StorePage({ id, connect, error }: { id: string; connect?: boolean; error?: string }) {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useStore(id)
  const p = q.data
  const navigate = useNavigate()
  const [dialog, setDialog] = useState<Dialog>(connect ? 'connect' : null)
  const platform = p?.store.platform === 'woocommerce' ? t(cm.woocommerce) : t(cm.shopify)
  const name = p?.store.name ?? ''
  const close = () => {
    setDialog(null)
    // ?connect and ?error are Shopify's way back: a reload doesn't reopen the dialog.
    if (connect || error) void navigate({ to: '/w/$workspace/settings/integrations/stores/$storeId', params: { workspace: slug, storeId: id }, search: {}, replace: true })
  }
  // A store connected already has nothing to resume when Shopify sends the browser back.
  const connecting = dialog === 'reconnect' || dialog === 'settings' || (dialog === 'connect' && (p?.store.step !== 'done' || !!error))
  return (
    <Page
      title={p ? t(cm.crumb, { platform, name }) : t(cm.integrations)}
      documentTitle={p ? t(cm.documentTitle, { name }) : t(cm.integrations)}
      breadcrumb={[
        { label: t(cm.settings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(cm.integrations), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: p ? t(cm.crumb, { platform, name }) : '…' },
      ]}
    >
      {q.isError ? (
        <section className="card flex flex-wrap items-center justify-between gap-3 px-5 py-4" role="alert">
          <span>{errorMessage(q.error) || t(cm.loadError)}</span>
          <Button size="sm" onClick={() => void q.refetch()}>
            {t(cm.retry)}
          </Button>
        </section>
      ) : !p ? (
        <StoreSkeleton />
      ) : (
        <Store page={p} onDialog={setDialog} />
      )}
      {p && connecting && (dialog === 'connect' || dialog === 'reconnect' || dialog === 'settings') ? (
        <ConnectStoreDialog platform={p.store.platform} storeId={p.store.id} error={error} mode={dialog} onOpenChange={(o) => !o && close()} />
      ) : null}
      {p && dialog === 'disconnect' ? <DisconnectStoreDialog page={p} onOpenChange={(o) => !o && close()} /> : null}
      {p && dialog === 'log' ? <ActivityLogDialog page={p} c={contextOf(p, platform)} open onOpenChange={(o) => !o && close()} /> : null}
    </Page>
  )
}

function contextOf(p: Page_, platform: string): ActivityContext {
  return { store: p.store.name, platform, list: p.consent.listName, window: p.attribution.days }
}

function StoreSkeleton() {
  return (
    <>
      <section className="card ih-dhead" aria-busy="true">
        <div className="ih-dmain">
          <Skeleton w={40} h={40} r={10} />
          <div className="ih-dtext">
            <Skeleton w={220} h={22} />
            <Skeleton w="70%" h={13} />
            <Skeleton w="50%" h={13} />
          </div>
        </div>
      </section>
      <div className="ih-stats">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="card ih-stat">
            <Skeleton w={90} h={12} />
            <Skeleton w={60} h={22} />
          </div>
        ))}
      </div>
    </>
  )
}

function Store({ page: p, onDialog }: { page: Page_; onDialog: (d: Dialog) => void }) {
  const t = useT()
  const platform = p.store.platform === 'woocommerce' ? t(cm.woocommerce) : t(cm.shopify)
  const state = p.store.sync.state
  const c = contextOf(p, platform)
  return (
    <>
      {state === 'webhook_missing' ? <MissingBanner page={p} platform={platform} /> : null}
      {state === 'rate_limited' ? (
        <div className="ih-banner" role="status">
          <div className="ih-bi">
            <AttentionIcon size={16} />
            <span className="ih-bt">
              <T msg={cm.rateBanner} values={{ platform, time: p.store.sync.rateLimitedUntil ? clock(p.store.sync.rateLimitedUntil) : '', b: bold }} />
            </span>
          </div>
        </div>
      ) : null}
      <Header page={p} onDialog={onDialog} />
      {!p.canManage ? <ReadOnly helpers={p.helpers} /> : null}
      {state === 'disconnected' ? <Disconnected page={p} platform={platform} onReconnect={() => onDialog('reconnect')} /> : null}
      {state === 'no_plugin' ? <NoPlugin page={p} /> : null}
      {state === 'importing' && p.store.sync.import ? (
        <ImportCard page={p} />
      ) : state === 'connecting' ? null : (
        <>
          <Stats page={p} />
          <WhatItDoes page={p} />
          <div className="ih-two">
            <SyncCard page={p} />
            <ConsentCard page={p} platform={platform} />
          </div>
          <AttributionCard page={p} />
          {/* A missing webhook (CommerceStates) gets the whole width, for its Register again. */}
          <div className={cn('ih-two', state === 'webhook_missing' && 'cm-stack')}>
            {state !== 'disconnected' ? <WebhooksCard page={p} platform={platform} /> : null}
            <section className="card" aria-labelledby="cm-act">
              <div className="sec">
                <div className="flex min-w-0 flex-col gap-0.5">
                  <h2 className="h2" id="cm-act">
                    {t(cm.activityTitle)}
                  </h2>
                  <span className="k">{t(cm.activityCaption)}</span>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <button type="button" className="ih-lnk cursor-pointer border-0 bg-transparent p-0 text-accent" onClick={() => onDialog('log')}>
                    {t(cm.activityFull)}
                  </button>
                </div>
              </div>
              <div className="pad pt-0.5 pb-1.5">
                {p.activity.length === 0 ? (
                  <div className="py-3">
                    <span className="k">{t(cm.activityEmpty)}</span>
                  </div>
                ) : (
                  p.activity.map((it) => <ActivityEntry key={it.id} item={it} c={c} />)
                )}
              </div>
            </section>
          </div>
          {p.canManage && state !== 'disconnected' ? <Danger page={p} onDisconnect={() => onDialog('disconnect')} /> : null}
        </>
      )}
    </>
  )
}

/** Logo, name, status, the store's facts, its capabilities and the actions (`.dhead7`). */
function Header({ page: p, onDialog }: { page: Page_; onDialog: (d: Dialog) => void }) {
  const t = useT()
  const s = p.store
  const woo = s.platform === 'woocommerce'
  const platform = woo ? t(cm.woocommerce) : t(cm.shopify)
  const st = syncStatus(t, s.sync, s.platform)
  const host = woo ? s.address : storefront(p)
  const date = p.connectedAt ? dayMonthYear(p.connectedAt) : ''
  const facts = woo
    ? s.sync.state === 'no_plugin'
      ? s.demo
        ? cm.factsWooSampleNoPlugin
        : cm.factsWooNoPlugin
      : cm.factsWoo
    : p.connectedAt
      ? p.connectedBy
        ? cm.factsShopify
        : cm.factsShopifyNoName
      : cm.factsShopifyPlain
  const { slug } = useWorkspace()
  const sync = useSyncNow(p)
  const check = usePluginCheck(p)
  const title = woo && s.sync.state === 'no_plugin' ? s.address : s.name
  const caps = p.capabilities.filter((c) => capLabel[c])
  return (
    <section className="card ih-dhead" aria-label={title}>
      <div className="ih-dmain">
        <Tile k={s.platform} size="lg" />
        <div className="ih-dtext gap-1.5">
          <div className="row gap-2.5">
            <h2 className="ih-ttl">{title}</h2>
            <Badge tone={st.tone} dot>
              {st.label}
            </Badge>
          </div>
          <span className="text-[13px] text-ink-2">
            <T
              msg={facts}
              values={{ platform, host, domain: s.domain, currency: s.currency, app: p.appName, version: p.apiVersion, name: p.connectedBy, date, mono }}
            />
          </span>
          <p className="max-w-[760px]">{t(woo ? (s.sync.state === 'no_plugin' ? cm.lineWooNoPlugin : cm.lineWoo) : cm.lineShopify)}</p>
          <div className="ih-caps">
            {caps.map((c) => (
              <span key={c} className="ih-cap">
                {t(capLabel[c]!)}
              </span>
            ))}
          </div>
        </div>
      </div>
      {p.canManage ? (
        <div className="ih-dact">
          {s.sync.state === 'disconnected' ? (
            <Button variant="primary" onClick={() => onDialog('reconnect')}>
              {t(cm.reconnect)}
            </Button>
          ) : s.sync.state === 'no_plugin' ? (
            <>
              <ButtonLink variant="primary" to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} search={{ connect: true }}>
                {t(cm.installPlugin)}
              </ButtonLink>
              <Button onClick={() => check.run()} disabled={check.busy}>
                {t(cm.checkAgain)}
              </Button>
            </>
          ) : s.sync.state === 'connecting' ? (
            <Button variant="primary" onClick={() => onDialog('connect')}>
              {t(cm.continueConnect)}
            </Button>
          ) : (
            <>
              <ButtonAnchor href={`https://${host}`} iconRight="external">
                {t(cm.openStore)}
                <span className="vh">{t(cm.newTab)}</span>
              </ButtonAnchor>
              {s.sync.state !== 'importing' ? (
                <>
                  <Button icon="refresh" onClick={() => sync.run()} disabled={sync.busy}>
                    {t(cm.syncNow)}
                  </Button>
                  <Menu
                    trigger={
                      <button type="button" className="btn ic" aria-haspopup="menu" aria-label={t(cm.moreActions, { name: s.name })}>
                        <Icon name="more" size={16} />
                      </button>
                    }
                  >
                    <MenuItem onSelect={() => onDialog('settings')}>{t(cm.changeSyncs)}</MenuItem>
                    <MenuItem onSelect={() => onDialog('reconnect')}>{t(cm.reconnect)}</MenuItem>
                    <MenuItem onSelect={() => onDialog('disconnect')}>{t(cm.disconnect)}</MenuItem>
                  </Menu>
                </>
              ) : null}
            </>
          )}
        </div>
      ) : null}
    </section>
  )
}

function useSyncNow(p: Page_) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshStores()
  const m = $api.useMutation('post', `${storesBase}/{id}/sync`)
  return {
    busy: m.isPending,
    run: () =>
      m.mutate(
        { params: { path: { workspace: slug, id: p.store.id } } },
        {
          onSuccess: () => {
            toast({ kind: 'success', title: t(cm.syncQueued, { name: p.store.name }) })
            void refresh()
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      ),
  }
}

function usePluginCheck(p: Page_) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshStores()
  const m = $api.useMutation('post', `${storesBase}/{id}/plugin-check`)
  const host = p.plugin?.host ?? p.store.address
  return {
    busy: m.isPending,
    run: () =>
      m.mutate(
        { params: { path: { workspace: slug, id: p.store.id } } },
        {
          onSuccess: (st) => {
            if (st.plugin?.found) toast({ kind: 'success', title: t(cm.pluginFound, { host }) })
            else toast({ kind: 'info', title: t(cm.pluginStillMissing, { host }) })
            void refresh()
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      ),
  }
}

function ReadOnly({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data
  return (
    <div className="note n" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(cm.readOnlyTitle, { name: user?.user.name ?? '', role: me.workspace.roleName })}</span>
        <div>
          <T msg={helpers.length ? cm.readOnlyBody : cm.readOnlyAlone} values={{ helpers: formatList(helpers, 'disjunction') }} />
        </div>
      </div>
    </div>
  )
}

/** "Webhook checkouts/update is missing." with Register again (CommerceStates, 13 Oct). */
function MissingBanner({ page: p, platform }: { page: Page_; platform: string }) {
  const t = useT()
  const register = useRegister(p)
  const topic = p.store.sync.missing[0] ?? ''
  const since = p.store.sync.missingSince
  // The message words each resource whole: checkouts, customers, products, refunds or orders.
  const resource = topic.split('/')[0] ?? ''
  return (
    <div className="ih-banner" role="status">
      <div className="ih-bi">
        <AttentionIcon size={16} />
        <span className="ih-bt">
          <T msg={cm.missingBanner} values={{ topic, platform, date: since ? dayMonthDate(since) : '', time: since ? clock(since) : '', resource, b: bold }} />
        </span>
      </div>
      {p.canManage ? (
        <Button size="sm" onClick={() => register.run(false)} disabled={register.busy}>
          {t(cm.hookAgain)}
        </Button>
      ) : null}
    </div>
  )
}

function useRegister(p: Page_) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshStores()
  const m = $api.useMutation('post', `${storesBase}/{id}/webhooks`)
  return {
    busy: m.isPending,
    run: (all: boolean) =>
      m.mutate(
        { params: { path: { workspace: slug, id: p.store.id }, query: { all } } },
        {
          onSuccess: () => {
            toast({ kind: 'success', title: t(cm.hooksRegistered) })
            void refresh()
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      ),
  }
}

/** "Acme Merch is disconnected" with Reconnect and the paused flow. */
function Disconnected({ page: p, platform, onReconnect }: { page: Page_; platform: string; onReconnect: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const s = p.store.sync
  const at = s.revokedAt ?? s.disconnectedAt
  const date = at ? dayMonthDate(at) : ''
  const time = at ? clock(at) : ''
  const paused = p.flows.filter((f) => f.status === 'paused')
  const flow = paused[0]?.name ?? ''
  const values = { platform, date, time, flows: paused.length, flow, name: s.disconnectedBy ?? '' }
  const body =
    s.revokedReason === 'uninstalled'
      ? t(cm.disconnectedUninstalled, values)
      : s.revokedReason === 'revoked'
        ? t(cm.disconnectedRevoked, values)
        : s.disconnectedBy
          ? t(cm.disconnectedByPerson, values)
          : t(cm.disconnectedNoName, values)
  return (
    <div className="note b" role="alert">
      <AttentionIcon size={16} color="var(--bad-fg)" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(cm.disconnectedTitle, { name: p.store.name })}</span>
        <div>
          {body}
          {p.canManage || paused.length ? (
            <div className="mt-2 flex flex-wrap gap-2">
              {p.canManage ? (
                <Button size="sm" variant="primary" onClick={onReconnect}>
                  {t(cm.reconnect)}
                </Button>
              ) : null}
              {paused[0] ? (
                <ButtonLink size="sm" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: paused[0].id }}>
                  {t(cm.openFlow, { name: paused[0].name })}
                </ButtonLink>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
    </div>
  )
}

/** WooCommerce without the plugin: what works now, what needs it. */
function NoPlugin({ page: p }: { page: Page_ }) {
  const t = useT()
  const { slug } = useWorkspace()
  const host = p.plugin?.host ?? p.store.address
  return (
    <>
      <div className="ih-banner" role="status">
        <div className="ih-bi">
          <AttentionIcon size={16} />
          <span className="ih-bt">{t(cm.noPluginBanner, { host })}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/settings/integrations/wordpress" params={{ workspace: slug }} search={{ connect: true }}>
          {t(cm.installPlugin)}
        </ButtonLink>
      </div>
      <div className="cm-two">
        <div className="cm-sbox">
          <h4>{t(cm.worksNow)}</h4>
          <ul>
            {[cm.worksProducts, cm.worksOrders, cm.worksTriggers].map((d) => (
              <li key={d.id}>
                <CheckSvg />
                <span>{t(d)}</span>
              </li>
            ))}
          </ul>
        </div>
        <div className="cm-sbox">
          <h4>{t(cm.needsPlugin)}</h4>
          <ul>
            {[cm.needsConsent, cm.needsCheckouts, cm.needsToken].map((d) => (
              <li key={d.id}>
                <CrossSvg />
                <span>{t(d)}</span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </>
  )
}

function CheckSvg() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m3 8 3.5 3.5L13 5" />
    </svg>
  )
}

function CrossSvg() {
  return (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="#B91C1C" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 4l8 8M12 4l-8 8" />
    </svg>
  )
}

/** The four tiles: orders, from email, the list, abandoned cart. */
function Stats({ page: p }: { page: Page_ }) {
  const t = useT()
  const n = p.numbers
  const cur = n.currency || p.store.currency
  const cart = p.recovered
  return (
    <div className="ih-stats">
      <div className="card ih-stat">
        <span className="k">{t(cm.statOrders, { days: n.days })}</span>
        <span className="ih-v">{count(n.orders)}</span>
        <span className="ih-s">{t(cm.statOrdersSub, { amount: money(n.revenue, cur) })}</span>
      </div>
      <div className="card ih-stat">
        <span className="k">{t(cm.statEmail)}</span>
        <span className="ih-v">{t(cm.statEmailValue, { orders: count(n.credited), amount: money(n.creditedRevenue, cur) })}</span>
        <span className="ih-s">{t(cm.statEmailSub, { share: pct(n.share) })}</span>
      </div>
      <div className="card ih-stat">
        <span className="k">{p.consent.listName ? t(cm.statList, { list: p.consent.listName }) : t(cm.statListNone)}</span>
        <span className="ih-v">{count(n.listMembers)}</span>
        <span className="ih-s">{t(cm.statListSub)}</span>
      </div>
      <div className="card ih-stat">
        <span className="k">{t(cm.statCart)}</span>
        <span className="ih-v">{t(cm.statCartValue, { orders: count(cart.orders), amount: money(cart.revenue, cur) })}</span>
        <span className="ih-s">{cart.since ? t(cm.statCartSub, { date: dayMonthDate(cart.since) }) : t(cm.statCartNone)}</span>
      </div>
    </div>
  )
}

function Bolt() {
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M9 1.5 3.5 9H8l-1 5.5L12.5 7H8z" />
    </svg>
  )
}

function Bag() {
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 5.5h10l-.8 8.5H3.8z" />
      <path d="M5.8 5.5V4.6a2.2 2.2 0 0 1 4.4 0v.9" />
    </svg>
  )
}

function Check() {
  return (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="#0F766E" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m3 8 3.5 3.5L13 5" />
    </svg>
  )
}

/** What the store does: its triggers, its blocks and the data it syncs, and who uses it. */
function WhatItDoes({ page: p }: { page: Page_ }) {
  const t = useT()
  const woo = p.store.platform === 'woocommerce'
  const used = p.usedBy
    .map((u) =>
      u.kind === 'automation'
        ? t(cm.usedFlow, { name: u.name ?? '', status: u.status ?? 'draft' })
        : u.kind === 'campaign'
          ? t(cm.usedCampaign, { name: u.name ?? '' })
          : t(cm.usedReports),
    )
    .join(' · ')
  const col = (title: MessageDescriptor, icon: ReactNode, items: MessageDescriptor[]) => (
    <div>
      <h3>{t(title)}</h3>
      <ul>
        {items.map((d) => (
          <li key={d.id}>
            {icon}
            <span>
              <T msg={d} values={{ b: bold }} />
            </span>
          </li>
        ))}
      </ul>
    </div>
  )
  const triggers = woo && !p.plugin?.found ? [cm.triggerOrder, cm.triggerFirst] : [cm.triggerOrder, cm.triggerFirst, cm.triggerAbandoned]
  const data = woo && !p.plugin?.found ? [cm.dataProducts, cm.dataOrders] : [cm.dataProducts, cm.dataCustomers, cm.dataOrders, cm.dataCheckouts]
  return (
    <section className="card" aria-labelledby="cm-what">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cm-what">
            {t(cm.whatTitle, { name: p.store.name })}
          </h2>
          <span className="k">{used ? t(cm.usedBy, { items: used }) : t(cm.usedByNothing)}</span>
        </div>
      </div>
      <div className="cm-wd">
        {col(cm.triggersTitle, <Bolt />, triggers)}
        {col(cm.blocksTitle, <Bag />, [cm.blockCard, cm.blockGrid, cm.blockCheckout])}
        {col(cm.dataTitle, <Check />, data)}
      </div>
    </section>
  )
}

/** The sync row's colour. */
function syncColor(state: SyncState['state']): string {
  switch (state) {
    case 'connected':
      return '#15803D'
    case 'importing':
    case 'connecting':
      return '#1D4ED8'
    case 'disconnected':
      return '#B91C1C'
  }
  return '#B45309'
}

function SyncCard({ page: p }: { page: Page_ }) {
  const t = useT()
  const f = p.sync
  const s = p.store.sync
  const sync = useSyncNow(p)
  const st = syncStatus(t, s, p.store.platform)
  const today = f.reconciledAt && dayMonthDate(f.reconciledAt) === dayMonthDate(new Date())
  const rows: [ReactNode, ReactNode][] = [
    [
      t(cm.kvProducts),
      f.collections.length ? (
        <T msg={cm.kvProductsValue} values={{ count: count(f.products), collections: f.collections.length, names: f.collections.join(', '), b: bold }} />
      ) : (
        <T msg={cm.kvProductsNone} values={{ count: count(f.products), b: bold }} />
      ),
    ],
  ]
  if (f.customersFromOrders) {
    rows.push([t(cm.kvAccepted), <T key="a" msg={cm.kvAcceptedOrders} values={{ count: count(f.accepted), list: p.consent.listName ?? '', b: bold }} />])
  } else {
    rows.push([
      t(cm.kvAccepted),
      p.consent.listName ? (
        <T msg={cm.kvAcceptedValue} values={{ count: count(f.accepted), list: p.consent.listName, known: count(f.known), fresh: count(f.new), b: bold }} />
      ) : (
        <T msg={cm.kvAcceptedNoList} values={{ count: count(f.accepted), b: bold }} />
      ),
    ])
    rows.push([t(cm.kvNotAdded), <T key="n" msg={cm.kvNotAddedValue} values={{ count: count(f.notAdded), b: bold }} />])
  }
  rows.push([t(cm.kvOrders), <T key="o" msg={cm.kvOrdersValue} values={{ count: count(f.ordersImported), months: f.months, b: bold }} />])
  if (f.importFinished) {
    rows.push([t(cm.kvImport), t(cm.kvImportValue, { date: dayMonthDate(f.importFinished), time: clock(f.importFinished), minutes: f.importMinutes })])
  }
  rows.push([
    t(cm.kvWebhooks),
    s.lastEventAt ? t(cm.kvWebhooksValue, { ok: f.webhooks, total: f.webhooksTotal, ago: ago(s.lastEventAt) }) : t(cm.kvWebhooksNoEvent, { ok: f.webhooks, total: f.webhooksTotal }),
  ])
  rows.push([
    t(cm.kvReconcile),
    f.reconciledAt
      ? today
        ? t(cm.kvReconcileToday, { time: clock(f.reconciledAt), missed: f.missed })
        : t(cm.kvReconcileDay, { date: dayMonthDate(f.reconciledAt), time: clock(f.reconciledAt), missed: f.missed })
      : t(cm.kvReconcileNever),
  ])
  return (
    <section className="card" aria-labelledby="cm-sync">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cm-sync">
            {t(cm.syncTitle)}
          </h2>
          <span className="k">{t(cm.syncCaption)}</span>
        </div>
        {p.canManage && s.state !== 'disconnected' ? (
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" icon="refresh" onClick={() => sync.run()} disabled={sync.busy}>
              {t(cm.syncNow)}
            </Button>
          </div>
        ) : null}
      </div>
      <div className="pad flex flex-col gap-3.5">
        <KeyValues items={rows} className="cm-kv" />
        <span className="cm-syncr">
          <span className="dot" style={{ color: syncColor(s.state) }} aria-hidden="true" />
          {st.label}
        </span>
      </div>
    </section>
  )
}

function useSettings(p: Page_) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshStores()
  const m = $api.useMutation('patch', `${storesBase}/{id}/settings`)
  return {
    busy: m.isPending,
    save: (body: Schemas['CommerceSettingsInput']) =>
      m.mutate(
        { params: { path: { workspace: slug, id: p.store.id } }, body },
        {
          onSuccess: () => {
            toast({ kind: 'success', title: t(cm.saved) })
            void refresh()
          },
          onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
        },
      ),
  }
}

function ConsentCard({ page: p, platform }: { page: Page_; platform: string }) {
  const t = useT()
  const settings = useSettings(p)
  const shopify = p.store.platform === 'shopify'
  const off = !p.canManage || settings.busy
  return (
    <section className="card" aria-labelledby="cm-consent">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cm-consent">
            {t(cm.consentTitle)}
          </h2>
          <span className="k">{t(cm.consentCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3.5">
        <label className="lbl">
          {t(cm.consentJoin)}
          <select
            className="in"
            aria-label={t(cm.consentListLabel)}
            value={p.consent.listId ?? ''}
            disabled={off}
            onChange={(e) => settings.save({ listId: e.target.value })}
          >
            {p.consent.listId ? null : <option value="">{t(cm.consentNoList)}</option>}
            {p.lists.map((l) => (
              <option key={l.id} value={l.id}>
                {l.store ? t(cm.consentStoreList, { name: l.name, count: count(l.members) }) : l.name}
              </option>
            ))}
          </select>
        </label>
        <div className="note w" role="note">
          <AttentionIcon size={16} color="#B45309" />
          <div className="min-w-0 flex-[1_1_320px]">{t(cm.consentReminder, { store: p.store.name })}</div>
        </div>
        {p.consent.writeBackSupported ? (
          <div className="cm-srow">
            <span>
              <b className="font-semibold">{t(cm.writeBack)}</b>
              <span className="k">{p.consent.writeGranted ? t(cm.writeBackGranted) : t(cm.writeBackNotGranted)}</span>
            </span>
            <input
              type="checkbox"
              role="switch"
              className="tg"
              aria-label={t(cm.writeBack)}
              checked={p.consent.writeBack && p.consent.writeGranted}
              disabled={off || !p.consent.writeGranted}
              onChange={(e) => settings.save({ writeBack: e.target.checked })}
            />
          </div>
        ) : null}
        <span className="k text-ink-nav">{shopify ? t(cm.consentExplain, { platform }) : t(cm.consentExplainWoo)}</span>
        <div className="cm-two">
          <div className="cm-sbox">
            <h4>{t(cm.stored)}</h4>
            <ul>
              {[cm.storedContact, cm.storedConsent, cm.storedOrders, cm.storedLanding].map((d) => (
                <li key={d.id}>
                  <CheckSvg />
                  <span>{t(d)}</span>
                </li>
              ))}
            </ul>
          </div>
          <div className="cm-sbox">
            <h4>{t(cm.neverStored)}</h4>
            <ul>
              {[cm.neverAddresses, cm.neverPhones, cm.neverPayment, cm.neverNotes].map((d) => (
                <li key={d.id}>
                  <CrossSvg />
                  <span>{t(d)}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </section>
  )
}

const windows = [1, 3, 5, 7, 14, 30]

function AttributionCard({ page: p }: { page: Page_ }) {
  const t = useT()
  const a = p.attribution
  const settings = useSettings(p)
  const off = !p.canManage || settings.busy
  const cur = p.numbers.currency || p.store.currency
  const other = Math.max(0, a.revenue - a.creditedRevenue)
  const rows = [...a.rows]
  const maxOrders = Math.max(1, ...rows.map((r) => r.orders), a.others?.orders ?? 0)
  const label = (r: Schemas['CommerceRevenueRow']) => (r.kind === 'automation' ? t(cm.attrFlow, { name: r.name }) : r.name)
  return (
    <section className="card" aria-labelledby="cm-attr">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cm-attr">
            {t(cm.attrTitle)}
          </h2>
          <span className="k">{t(cm.attrCaption)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-4">
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(260px,100%),1fr))] gap-3.5">
          <label className="lbl">
            {t(cm.attrWindow)}
            <select className="in" aria-label={t(cm.attrWindowLabel)} value={a.days} disabled={off} onChange={(e) => settings.save({ attributionDays: Number(e.target.value) })}>
              {windows.map((d) => (
                <option key={d} value={d}>
                  {t(cm.attrDays, { days: d })}
                </option>
              ))}
            </select>
            {a.setBy ? <span className="k font-medium">{t(cm.attrSetBy, { name: a.setBy })}</span> : null}
          </label>
          <div className="lbl">
            {t(cm.attrMatching)}
            <span className="text-[13px] leading-[1.45] font-medium text-ink-nav">
              <T msg={cm.attrMatchingText} values={{ param: a.param, mono: (c) => <span className="mono">{c}</span> }} />
            </span>
          </div>
        </div>
        <div className="cm-srow">
          <span className="gap-1">
            <b className="font-semibold">{t(cm.attrTag, { host: a.tagHost || storefront(p) })}</b>
            <span className="k">{t(cm.attrTagSub)}</span>
            <span className="mono k [overflow-wrap:anywhere]">{a.tagExample}</span>
          </span>
          <input type="checkbox" role="switch" className="tg" aria-label={t(cm.attrTagLabel)} checked={a.tagLinks} disabled={off} onChange={(e) => settings.save({ tagLinks: e.target.checked })} />
        </div>
        <div className="flex flex-col gap-2">
          <span className="font-semibold">{t(cm.attrLast, { days: p.numbers.days, orders: a.orders, amount: money(a.revenue, cur) })}</span>
          <div className="flex flex-col gap-2.5">
            <div
              className={cn('flex h-2.5 gap-0.5 overflow-hidden rounded-[5px]', a.creditedRevenue + other === 0 && 'bg-line-2')}
              role="img"
              aria-label={t(cm.attrBar, { credited: count(a.creditedRevenue / 100), other: count(other / 100) })}
            >
              {/* No orders yet: the empty track alone. */}
              {a.creditedRevenue > 0 ? <span style={{ flex: a.creditedRevenue, background: '#0F766E' }} /> : null}
              {other > 0 ? <span style={{ flex: other, background: '#C9CED6' }} /> : null}
            </div>
            <div className="flex flex-wrap gap-3.5 text-[12px] text-ink-nav">
              <span className="inline-flex items-center gap-1.5">
                <span className="size-2.5 rounded-[3px] bg-[#0F766E]" />
                {t(cm.attrCredited, { amount: money(a.creditedRevenue, cur) })}
              </span>
              <span className="inline-flex items-center gap-1.5">
                <span className="size-2.5 rounded-[3px] bg-[#C9CED6]" />
                {t(cm.attrOther, { amount: money(other, cur) })}
              </span>
            </div>
          </div>
          <span className="k">{t(p.sync.ordersImported ? cm.attrMatched : cm.attrMatchedNoImport, { token: a.byToken, email: a.byEmail })}</span>
        </div>
      </div>
      {rows.length || a.others ? (
        <>
          <div className="ih-only-wide hidden sm:block">
            <div className="overflow-x-auto">
              <table className="tbl compact" aria-label={t(cm.attrTable)}>
                {/* The board's columns, at every width. */}
                <colgroup>
                  <col style={{ width: '36.8%' }} />
                  <col style={{ width: '13.7%' }} />
                  <col style={{ width: '15%' }} />
                  <col style={{ width: '34.5%' }} />
                </colgroup>
                <thead>
                  <tr>
                    <Th>{t(cm.attrTo)}</Th>
                    <Th align="right">{t(cm.attrOrders)}</Th>
                    <Th align="right">{t(cm.attrRevenue)}</Th>
                    <Th>
                      <span className="vh">{t(cm.attrOrders)}</span>
                    </Th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.kind + (r.id ?? r.name)}>
                      <td>{label(r)}</td>
                      <td className="r">
                        <span className="num">{count(r.orders)}</span>
                      </td>
                      <td className="r">
                        <span className="num">{money(r.revenue, cur)}</span>
                      </td>
                      <td>
                        <Bar value={(r.orders / maxOrders) * 100} color="#0F766E" height={6} className="w-[160px]" />
                      </td>
                    </tr>
                  ))}
                  {a.others ? (
                    <tr>
                      <td>{t(cm.attrOthers, { count: a.others.count })}</td>
                      <td className="r">
                        <span className="num">{count(a.others.orders)}</span>
                      </td>
                      <td className="r">
                        <span className="num">{money(a.others.revenue, cur)}</span>
                      </td>
                      <td>
                        <Bar value={(a.others.orders / maxOrders) * 100} color="#0F766E" height={6} className="w-[160px]" />
                      </td>
                    </tr>
                  ) : null}
                  <tr className="font-bold">
                    <td>
                      <b>{t(cm.attrTotal)}</b>
                    </td>
                    <td className="r">
                      <b className="num">{count(a.credited)}</b>
                    </td>
                    <td className="r">
                      <b className="num">{money(a.creditedRevenue, cur)}</b>
                    </td>
                    <td>
                      <span className="k">{t(cm.attrShare, { share: pct(p.numbers.share), amount: money(a.revenue, cur) })}</span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <div className="sm:hidden">
            {rows.map((r) => (
              <div key={r.kind + (r.id ?? r.name)} className="cm-mrow">
                <span>{label(r)}</span>
                <span className="num">
                  <b>{count(r.orders)}</b> · {money(r.revenue, cur)}
                </span>
              </div>
            ))}
            {a.others ? (
              <div className="cm-mrow">
                <span>{t(cm.attrOthers, { count: a.others.count })}</span>
                <span className="num">
                  <b>{count(a.others.orders)}</b> · {money(a.others.revenue, cur)}
                </span>
              </div>
            ) : null}
          </div>
        </>
      ) : (
        <div className="pad border-t border-line-2">
          <span className="k">{t(cm.attrNone, { days: p.numbers.days })}</span>
        </div>
      )}
    </section>
  )
}

function WebhooksCard({ page: p, platform }: { page: Page_; platform: string }) {
  const t = useT()
  const register = useRegister(p)
  const now = useNow()
  const missing = (s: string) => s === 'missing'
  // A row with Register again sizes its own columns, as on the board.
  const action = p.canManage && p.webhooks.some((w) => missing(w.status))
  const badge = (w: Schemas['CommerceWebhook'], short?: boolean) =>
    w.status === 'registered' ? (
      <Badge tone="ok">{t(cm.hookRegistered)}</Badge>
    ) : missing(w.status) ? (
      <Badge tone="warn" dot>
        {short ? t(cm.hookMissing) : t(cm.hookMissingPolling)}
      </Badge>
    ) : w.status === 'failed' ? (
      <Badge tone="bad">{t(cm.hookFailed)}</Badge>
    ) : (
      <Badge tone="neutral">{t(cm.hookPending)}</Badge>
    )
  const last = (w: Schemas['CommerceWebhook']) => {
    if (!w.lastEventAt) return t(cm.hookNever)
    const d = new Date(w.lastEventAt)
    return now.getTime() - d.getTime() < 60 * 60 * 1000 ? ago(d, now) : `${dayMonthDate(d)} ${clock(d)}`
  }
  return (
    <section className="card" aria-labelledby="cm-hooks">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cm-hooks">
            {t(cm.hooksTitle, { count: p.webhooks.length })}
          </h2>
          <span className="k">{t(cm.hooksCaption, { platform })}</span>
        </div>
        {p.canManage ? (
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" onClick={() => register.run(true)} disabled={register.busy}>
              {t(cm.hooksAgain)}
            </Button>
          </div>
        ) : null}
      </div>
      <div className="hidden sm:block">
        <div className="overflow-x-auto">
          <table className="tbl compact" aria-label={t(cm.hooksTitle, { count: p.webhooks.length })}>
            {action ? null : (
              <colgroup>
                <col style={{ width: '39.6%' }} />
                <col style={{ width: '27.5%' }} />
                <col style={{ width: '26.4%' }} />
                <col style={{ width: '6.5%' }} />
              </colgroup>
            )}
            <thead>
              <tr>
                <Th>{t(cm.hookTopic)}</Th>
                <Th>{t(cm.hookStatus)}</Th>
                <Th>{t(cm.hookLast)}</Th>
                <Th>
                  <span className="vh">{t(cm.hookAgain)}</span>
                </Th>
              </tr>
            </thead>
            <tbody>
              {p.webhooks.map((w) => (
                <tr key={w.topic}>
                  <td>
                    <span className="mono text-[12.5px]">{w.topic}</span>
                  </td>
                  <td>{badge(w)}</td>
                  <td className="nw">
                    <span className="k">{last(w)}</span>
                  </td>
                  <td className="r">
                    {missing(w.status) && p.canManage ? (
                      <Button size="sm" variant="primary" onClick={() => register.run(false)} disabled={register.busy}>
                        {t(cm.hookAgain)}
                      </Button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <div className="sm:hidden">
        {p.webhooks.map((w) => (
          <div key={w.topic} className="cm-mrow">
            <span className="mono text-[12.5px]">{w.topic}</span>
            {badge(w, true)}
          </div>
        ))}
      </div>
    </section>
  )
}

/** The first import's progress (CommerceStates, "Importing orders · 5,120 of 7,880"). */
function ImportCard({ page: p }: { page: Page_ }) {
  const t = useT()
  const im = p.store.sync.import!
  const prog = (label: string, done: number, total: number, sub: string, waiting?: boolean) => {
    const v = total > 0 ? (done / total) * 100 : 0
    return (
      <div className="cm-prog">
        <div className="cm-row1">
          <b className="font-semibold">{label}</b>
          <span className="num">{waiting ? t(cm.importWaiting) : t(cm.progressOf, { done: count(done), total: count(total) })}</span>
        </div>
        <Bar value={waiting ? 0 : v} color={waiting ? '#C9CED6' : '#0F766E'} height={6} label={label} />
        <span className="k">{sub}</span>
      </div>
    )
  }
  const ordersActive = im.phase === 'orders'
  return (
    <section className="card" aria-labelledby="cm-import">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="cm-import">
            {t(cm.importTitle)}
          </h2>
          <span className="k">
            {im.startedBy
              ? t(cm.importStarted, { date: dayMonthDate(im.startedAt), time: clock(im.startedAt), name: im.startedBy })
              : t(cm.importStartedNoName, { date: dayMonthDate(im.startedAt), time: clock(im.startedAt) })}
          </span>
        </div>
      </div>
      <div className="pad flex flex-col gap-4">
        {prog(t(cm.importProducts), im.products.done, im.products.total, t(cm.importProductsSub, { count: im.collections }), im.phase === 'products' && im.products.total === 0)}
        {im.customersSkipped
          ? null
          : prog(
              t(cm.importCustomers),
              im.customers.done,
              im.customers.total,
              p.consent.listName
                ? t(cm.importCustomersSub, { accepted: count(im.accepted), list: p.consent.listName, notAdded: count(im.notAdded) })
                : t(cm.importCustomersSubNoList, { accepted: count(im.accepted), notAdded: count(im.notAdded) }),
              im.phase === 'products',
            )}
        {prog(
          ordersActive ? t(cm.importOrders) : t(cm.importOrdersDone),
          im.orders.done,
          im.orders.total,
          ordersActive ? t(cm.importOrdersSub, { months: im.months, minutes: Math.max(1, im.minutesLeft) }) : t(cm.importOrdersSubDone, { months: im.months }),
          im.phase === 'products' || im.phase === 'customers',
        )}
        {prog(t(cm.importMatching), im.matching.done, im.matching.total, t(cm.importAfterOrders), im.phase !== 'matching' && im.phase !== 'done')}
        <div className="note i" role="note">
          <Icon name="info" size={16} className="mt-px flex-none text-info-fg" />
          <div className="min-w-0 flex-[1_1_320px]">{t(cm.importNote)}</div>
        </div>
      </div>
    </section>
  )
}

function Danger({ page: p, onDisconnect }: { page: Page_; onDisconnect: () => void }) {
  const t = useT()
  const live = p.flows.filter((f) => f.status === 'live')
  return (
    <section className="card ih-dz" aria-label={t(cm.dangerTitle, { name: p.store.name })}>
      <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-0.5">
        <h2 className="h2">{t(cm.dangerTitle, { name: p.store.name })}</h2>
        <span className="k text-ink-nav">{t(cm.dangerBody, { flows: live.length, flow: live[0]?.name ?? '' })}</span>
      </div>
      <Button variant="danger" onClick={onDisconnect}>
        {t(cm.dangerButton, { name: p.store.name })}
      </Button>
    </section>
  )
}

/** A link to a store's page. */
export function StoreLink({ id, children }: { id: string; children: ReactNode }) {
  const { slug } = useWorkspace()
  return (
    <Link to="/w/$workspace/settings/integrations/stores/$storeId" params={{ workspace: slug, storeId: id }}>
      {children}
    </Link>
  )
}
