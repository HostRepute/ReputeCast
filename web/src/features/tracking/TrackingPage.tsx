import { useState, type ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { Table, Th } from '@/components/ui/Table'
import '@/features/commerce/commerce.css'
import '@/features/integrations/integrations.css'
import { T, formatList, useT } from '@/i18n'
import { ago, clock, count, dayMonthYear } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { deniedReason, useMe, useWorkspace } from '@/lib/session'
import { useRecentEvents, useSites, type RecentEvent, type Site, type SiteList } from './api'
import { tm } from './messages'
import { DeleteSiteDialog, RotateSecretDialog, SiteDialog } from './SiteDialog'
import './tracking.css'
import { TrackingIcon } from './icon'

type T_ = ReturnType<typeof useT>
export type TrackingDialog =
  | { kind: 'add'; origin?: string }
  | { kind: 'setup' | 'edit' | 'rotate' | 'delete'; id: string }
  | null

const mono = (c: ReactNode[]) => <span className="mono">{c}</span>
const code = (v: string) => <span className="mono text-[12.5px]">{v}</span>

/** Settings › Integrations › Website tracking (TrackingSites, TrackingStates). */
export function TrackingPage({ add }: { add?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useSites()
  const [dialog, setDialog] = useState<TrackingDialog>(add ? { kind: 'add' } : null)
  const [picked, setPicked] = useState<string>()
  const list = q.data
  const site = list?.items.find((s) => s.id === picked) ?? list?.items[0]
  const byId = (id: string) => list?.items.find((s) => s.id === id)
  const open = dialog && dialog.kind !== 'add' ? byId(dialog.id) : undefined
  return (
    <Page
      title={t(tm.title)}
      documentTitle={t(tm.documentTitle)}
      breadcrumb={[
        { label: t(tm.crumbSettings), to: '/w/$workspace/settings', params: { workspace: slug } },
        { label: t(tm.crumbIntegrations), to: '/w/$workspace/settings/integrations', params: { workspace: slug } },
        { label: t(tm.title) },
      ]}
    >
      {q.isError ? (
        <ErrorPanel
          title={t(tm.errorTitle)}
          actions={
            <Button variant="primary" onClick={() => void q.refetch()}>
              {t(tm.retry)}
            </Button>
          }
        >
          {errorMessage(q.error)} {t(tm.errorBody)}
        </ErrorPanel>
      ) : !list ? (
        <PageSkeleton />
      ) : (
        <>
          {!list.canManage ? <ReadOnly helpers={list.helpers} /> : null}
          <Header list={list} onAdd={() => setDialog({ kind: 'add' })} />
          {list.items.length === 0 ? (
            <EmptyState
              title={t(tm.emptyTitle)}
              className="border-dashed"
              action={
                <Button variant="primary" icon="plus" onClick={() => setDialog({ kind: 'add' })} disabledReason={manageReason(list)}>
                  {t(tm.addSite)}
                </Button>
              }
            >
              {t(tm.emptyBody)}
            </EmptyState>
          ) : (
            <>
              <SitesCard list={list} selected={site?.id} onSelect={setPicked} onDialog={setDialog} />
              <div className="grid items-start gap-4 [grid-template-columns:repeat(auto-fit,minmax(min(460px,100%),1fr))]">
                {site ? <EventsCard site={site} list={list} onDialog={setDialog} /> : null}
                <div className="flex min-w-0 flex-col gap-4">
                  <Recognised />
                  <Kept />
                </div>
              </div>
            </>
          )}
        </>
      )}
      {list && dialog?.kind === 'add' ? (
        <SiteDialog list={list} mode="add" origin={dialog.origin} onClose={() => setDialog(null)} onAdded={(id) => setPicked(id)} />
      ) : null}
      {list && open && (dialog?.kind === 'setup' || dialog?.kind === 'edit') ? <SiteDialog list={list} mode={dialog.kind} site={open} onClose={() => setDialog(null)} /> : null}
      {open && dialog?.kind === 'rotate' ? <RotateSecretDialog site={open} onClose={() => setDialog(null)} /> : null}
      {open && dialog?.kind === 'delete' ? <DeleteSiteDialog site={open} onClose={() => setDialog(null)} /> : null}
    </Page>
  )
}

/** "Adding a site needs integrations:manage. Ask Marcus Chen or Linh Nguyen." */
function manageReason(list: SiteList): string | undefined {
  return list.canManage ? undefined : deniedReason('integrations:manage', list.helpers)
}

function ReadOnly({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data
  return (
    <div className="note n" role="note">
      <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(tm.readOnlyTitle, { name: user?.user.name ?? '', role: me.workspace.roleName })}</span>
        <div>{helpers.length ? t(tm.readOnlyBody, { helpers: formatList(helpers, 'disjunction') }) : t(tm.readOnlyAlone)}</div>
      </div>
    </div>
  )
}

function PageSkeleton() {
  const t = useT()
  return (
    <div className="flex flex-col gap-5" aria-busy="true" aria-label={t(tm.loading)}>
      <section className="card ih-dhead">
        <div className="ih-dmain">
          <Skeleton w={52} h={52} r={12} />
          <div className="ih-dtext gap-2">
            <Skeleton w={220} h={22} />
            <Skeleton w="70%" h={12} />
            <Skeleton w="90%" h={12} />
          </div>
        </div>
      </section>
      <section className="card">
        <div className="sec">
          <Skeleton w={120} h={14} />
        </div>
        <div className="flex flex-col gap-3.5 px-[18px] py-4">
          {[0, 1, 2].map((i) => (
            <div key={i} className="flex items-center gap-4">
              <Skeleton w="22%" h={12} />
              <Skeleton w="16%" h={12} />
              <Skeleton w={110} h={20} r={999} />
              <Skeleton w="10%" h={12} />
              <Skeleton w="10%" h={12} />
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}

/** The header: the tile, the status, the facts, the lead, the capabilities and Add site. */
function Header({ list, onAdd }: { list: SiteList; onAdd: () => void }) {
  const t = useT()
  const now = useNow()
  const sites = list.items
  const last = sites.reduce<string | undefined>((a, s) => (s.lastEventAt && (!a || s.lastEventAt > a) ? s.lastEventAt : a), undefined)
  const first = sites[0]
  const facts = !first
    ? t(tm.factsNone)
    : first.createdBy
      ? t(tm.facts, { count: sites.length, name: first.createdBy, date: dayMonthYear(first.createdAt) })
      : t(tm.factsNoName, { count: sites.length, date: dayMonthYear(first.createdAt) })
  const caps = [tm.capPages, tm.capProducts, tm.capCart, tm.capCustom, tm.capConsent, tm.capIdentify]
  return (
    <section className="card ih-dhead" aria-label={t(tm.title)}>
      <div className="ih-dmain basis-[460px]">
        <span className="ih-tile lg" aria-hidden="true">
          <TrackingIcon size={22} />
        </span>
        <div className="ih-dtext gap-1.5">
          <div className="row gap-2.5">
            <h2 className="ih-ttl">{t(tm.title)}</h2>
            {sites.length ? (
              <Badge tone="ok" dot>
                {last ? t(tm.statusLastEvent, { ago: ago(last, now) }) : t(tm.statusConnected)}
              </Badge>
            ) : (
              <Badge tone="neutral">{t(tm.statusNone)}</Badge>
            )}
          </div>
          <span className="text-[13px] text-ink-2">{facts}</span>
          <p className="max-w-[760px]">{t(tm.lead)}</p>
          <div className="ih-caps">
            {caps.map((c) => (
              <span key={c.id} className="ih-cap">
                {t(c)}
              </span>
            ))}
          </div>
        </div>
      </div>
      <div className="ih-dact">
        <Button variant="primary" icon="plus" onClick={onAdd} disabledReason={manageReason(list)}>
          {t(tm.addSite)}
        </Button>
      </div>
    </section>
  )
}

function ConsentBadge({ wait }: { wait: boolean }) {
  const t = useT()
  return wait ? (
    <Badge tone="ok" dot>
      {t(tm.waits)}
    </Badge>
  ) : (
    <Badge tone="neutral">{t(tm.noWait)}</Badge>
  )
}

/** Set up, Edit and the menu with Rotate identify secret and Delete site…. */
function SiteActions({ site, list, onDialog }: { site: Site; list: SiteList; onDialog: (d: TrackingDialog) => void }) {
  const t = useT()
  const reason = list.canManage ? undefined : deniedReason('integrations:manage', list.helpers)
  return (
    <span className="inline-flex gap-1.5">
      <Button size="sm" aria-label={t(tm.setUpLabel, { name: site.name })} onClick={() => onDialog({ kind: 'setup', id: site.id })} disabledReason={reason}>
        {t(tm.setUp)}
      </Button>
      <Button size="sm" aria-label={t(tm.editLabel, { name: site.name })} onClick={() => onDialog({ kind: 'edit', id: site.id })} disabledReason={reason}>
        {t(tm.edit)}
      </Button>
      {reason ? (
        <Button size="sm" className="w-[30px] px-0" aria-haspopup="menu" aria-label={t(tm.moreLabel, { name: site.name })} disabledReason={reason}>
          <Icon name="more" size={16} />
        </Button>
      ) : (
        <Menu
          trigger={
            <button type="button" className="btn sm w-[30px] px-0" aria-haspopup="menu" aria-label={t(tm.moreLabel, { name: site.name })}>
              <Icon name="more" size={16} />
            </button>
          }
        >
          <MenuItem onSelect={() => onDialog({ kind: 'rotate', id: site.id })}>{t(tm.rotate)}</MenuItem>
          <MenuSeparator />
          <MenuItem onSelect={() => onDialog({ kind: 'delete', id: site.id })}>{t(tm.delete)}</MenuItem>
        </Menu>
      )}
    </span>
  )
}

function SitesCard({ list, selected, onSelect, onDialog }: { list: SiteList; selected?: string; onSelect: (id: string) => void; onDialog: (d: TrackingDialog) => void }) {
  const t = useT()
  const now = useNow()
  const platform = (p?: string) => (p === 'woocommerce' ? 'WooCommerce' : 'Shopify')
  const name = (s: Site) => (
    <button type="button" className="cursor-pointer border-0 bg-transparent p-0 text-left font-semibold text-ink underline-offset-2 hover:underline" aria-pressed={s.id === selected} onClick={() => onSelect(s.id)}>
      {s.name}
    </button>
  )
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(tm.sitesTitle, { count: count(list.items.length) })}</h2>
          <span className="k">{t(tm.sitesSub)}</span>
        </div>
      </div>
      <div className="ih-no-narrow">
        <Table caption={t(tm.sitesTable)}>
          <thead>
            <tr>
              <Th>{t(tm.colSite)}</Th>
              <Th>{t(tm.colStore)}</Th>
              <Th>{t(tm.colConsent)}</Th>
              <Th align="right">{t(tm.colEvents)}</Th>
              <Th>{t(tm.colLast)}</Th>
              <Th>
                <span className="vh">{t(tm.colActions)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {list.items.map((s) => (
              <tr key={s.id} className={s.id === selected ? 'tk-sel' : undefined}>
                <td>
                  <div className="tk-stk">
                    {name(s)}
                    <span className="tk-addr">{s.origin}</span>
                  </div>
                </td>
                <td>
                  {s.store ? (
                    <div className="tk-stk">
                      <span>{s.store.name}</span>
                      <span className="k">{platform(s.store.platform)}</span>
                    </div>
                  ) : (
                    <span className="k">{t(tm.notLinked)}</span>
                  )}
                </td>
                <td>
                  <ConsentBadge wait={s.waitConsent} />
                </td>
                <td className="r">
                  <span className="num">{s.lastEventAt || s.eventsToday ? count(s.eventsToday) : '–'}</span>
                </td>
                <td className="nw">
                  <span className="k">{s.lastEventAt ? ago(s.lastEventAt, now) : t(tm.noEvents)}</span>
                </td>
                <td className="r">
                  <SiteActions site={s} list={list} onDialog={onDialog} />
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      </div>
      <div className="ih-only-narrow">
        {list.items.map((s) => (
          <div key={s.id} className={s.id === selected ? 'tk-site sel' : 'tk-site'}>
            <div className="flex flex-wrap items-start justify-between gap-x-2.5 gap-y-2">
              <div className="tk-stk">
                {name(s)}
                <span className="tk-addr [overflow-wrap:anywhere]">{s.origin}</span>
              </div>
              <ConsentBadge wait={s.waitConsent} />
            </div>
            <span className="k text-ink-2">
              {t(tm.narrowLine, {
                store: s.store?.name ?? 'none',
                platform: platform(s.store?.platform),
                events: s.eventsToday,
                last: s.lastEventAt ? ago(s.lastEventAt, now) : 'none',
              })}
            </span>
            <SiteActions site={s} list={list} onDialog={onDialog} />
          </div>
        ))}
      </div>
    </section>
  )
}

function kindLabel(t: T_, e: RecentEvent): ReactNode {
  switch (e.kind) {
    case 'page':
      return t(tm.kindPage)
    case 'product':
      return t(tm.kindProduct)
    case 'cart':
      return t(tm.kindCart)
  }
  return code(e.name)
}

/** What the event was about: the product's title, or the page's path. */
function eventWhat(t: T_, e: RecentEvent, site: Site): { node: ReactNode; text: string } {
  if (e.kind === 'product' || e.kind === 'cart') {
    const text = e.productTitle || (site.store ? t(tm.notFound) : t(tm.notNamed))
    return { node: text, text }
  }
  return { node: <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{e.path || '/'}</span>, text: e.path || '/' }
}

function WhoBadge({ known }: { known: boolean }) {
  const t = useT()
  return known ? <Badge tone="info">{t(tm.known)}</Badge> : <Badge tone="neutral">{t(tm.visitor)}</Badge>
}

/** Last events · {site}: the live list, or waiting for the first event, with the refused-origin and no-store notes. */
function EventsCard({ site, list, onDialog }: { site: Site; list: SiteList; onDialog: (d: TrackingDialog) => void }) {
  const t = useT()
  const now = useNow()
  const q = useRecentEvents(site.id, true)
  const events = q.data?.items ?? []
  const refused = site.refused[0]
  const unnamed = !site.store && events.some((e) => e.kind === 'product' || e.kind === 'cart')
  const reason = manageReason(list)
  const silentFor = now.getTime() - new Date(site.createdAt).getTime()
  return (
    <section className="card min-w-0">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(tm.eventsTitle, { name: site.name })}</h2>
          <span className="k">{t(tm.eventsSub)}</span>
        </div>
        {events.length ? (
          <div className="flex flex-wrap items-center gap-2">
            <span className="tk-live">
              <span className="tk-dot" aria-hidden="true" />
              {t(tm.live)}
            </span>
          </div>
        ) : null}
      </div>
      {refused ? (
        <div className="px-[18px] pt-3.5">
          <div className="note w" role="status">
            <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
              <b className="font-semibold">{t(tm.refusedTitle, { origin: refused.origin })}</b>
              <span>{t(tm.refusedBody, { count: refused.count, origin: refused.origin, time: clock(refused.lastAt) })}</span>
              <span>
                <Button size="sm" onClick={() => onDialog({ kind: 'add', origin: refused.origin })} disabledReason={reason}>
                  {t(tm.addAsSite)}
                </Button>
              </span>
            </div>
          </div>
        </div>
      ) : null}
      {unnamed ? (
        <div className="px-[18px] pt-3.5">
          <div className="note i" role="note">
            <Icon name="info" size={16} className="mt-px flex-none text-info-strong" />
            <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
              <b className="font-semibold">{t(tm.noStoreTitle)}</b>
              <span>{t(tm.noStoreBody, { origin: site.origin })}</span>
              {list.stores.length ? (
                <span>
                  <Button size="sm" variant="primary" onClick={() => onDialog({ kind: 'edit', id: site.id })} disabledReason={reason}>
                    {t(tm.linkStore)}
                  </Button>
                </span>
              ) : null}
            </div>
          </div>
        </div>
      ) : null}
      {q.isError ? (
        <div className="pad" role="alert">
          {t(tm.eventsError)} {errorMessage(q.error)}
        </div>
      ) : q.isPending ? (
        <div className="flex flex-col gap-3 px-[18px] py-4" aria-busy="true">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} w="100%" h={14} />
          ))}
        </div>
      ) : events.length === 0 ? (
        <div className="flex flex-col gap-3 px-[18px] py-4">
          {site.waitConsent && silentFor > 30 * 60_000 ? (
            <div className="note i" role="note">
              <Icon name="info" size={16} className="mt-px flex-none text-info-strong" />
              <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
                <b className="font-semibold">{t(tm.consentTitle)}</b>
                <span><T msg={tm.consentBody} values={{ call: code("reputecast('consent', true)") }} /></span>
              </div>
            </div>
          ) : null}
          <div className="flex items-start gap-3">
            <span className="tk-spin" aria-hidden="true" />
            <div className="flex min-w-0 flex-col gap-1">
              <b className="font-semibold" role="status">
                {t(tm.waitingFirst)}
              </b>
              <span className="k">
                {site.waitConsent && silentFor > 30 * 60_000
                  ? t(tm.waitingNothing, { origin: site.origin, when: ago(site.createdAt, now) })
                  : site.createdBy
                    ? t(tm.waitingBy, { origin: site.origin, when: clock(site.createdAt), name: site.createdBy })
                    : t(tm.waitingAdded, { origin: site.origin, when: clock(site.createdAt) })}
              </span>
              {site.waitConsent && silentFor > 30 * 60_000 ? null : <span>{t(tm.waitingHelp, { origin: site.origin })}</span>}
              <span className="mt-1">
                <Button size="sm" onClick={() => onDialog({ kind: 'setup', id: site.id })} disabledReason={reason}>
                  {t(tm.setUp)}
                </Button>
              </span>
            </div>
          </div>
        </div>
      ) : (
        <EventsList site={site} events={events} />
      )}
    </section>
  )
}

/** The events as a table, and as a list on phones. */
export function EventsList({ site, events }: { site: Site; events: RecentEvent[] }) {
  const t = useT()
  return (
    <>
      <div className="tk-no-phone">
        <Table caption={t(tm.eventsTable, { name: site.name })}>
          <thead>
            <tr>
              <Th>{t(tm.colEvent)}</Th>
              <Th>{t(tm.colWhat)}</Th>
              <Th>{t(tm.colWho)}</Th>
              <Th>{t(tm.colWhen)}</Th>
            </tr>
          </thead>
          <tbody>
            {events.map((e, i) => (
              <tr key={i}>
                <td className="nw">{kindLabel(t, e)}</td>
                <td>{eventWhat(t, e, site).node}</td>
                <td>
                  <WhoBadge known={e.known} />
                </td>
                <td className="nw">
                  <span className="k mono">{clock(e.at)}</span>
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      </div>
      <div className="tk-only-phone">
        {events.map((e, i) => (
          <div key={i} className="tk-evrow">
            <span className="flex min-w-0 flex-col gap-0.5">
              <b className="font-semibold">{kindLabel(t, e)}</b>
              <span className="k [overflow-wrap:anywhere]">{t(tm.eventLine, { what: eventWhat(t, e, site).text, time: clock(e.at) })}</span>
            </span>
            <WhoBadge known={e.known} />
          </div>
        ))}
      </div>
    </>
  )
}

function Recognised() {
  const t = useT()
  const icon = (d: ReactNode) => (
    <svg width="15" height="15" viewBox="0 0 16 16" fill="none" stroke="var(--accent)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      {d}
    </svg>
  )
  const rows: [ReactNode, string, ReactNode][] = [
    [icon(<path d="M3 2.5 13 7l-5 1-2.5 5z" />), t(tm.clickTitle), <T key="c" msg={tm.clickBody} values={{ mono }} />],
    [
      icon(
        <>
          <rect x="2" y="2" width="12" height="12" rx="2" />
          <path d="M5 6h6M5 9h6M5 12h3" />
        </>,
      ),
      t(tm.signupTitle),
      t(tm.signupBody),
    ],
    [
      icon(
        <>
          <circle cx="5" cy="11" r="2.5" />
          <path d="m6.8 9.2 6.7-6.7M11.2 4.8l1.6 1.6" />
        </>,
      ),
      t(tm.identifyTitle),
      t(tm.identifyBody),
    ],
  ]
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(tm.recognisedTitle)}</h2>
          <span className="k">{t(tm.recognisedSub)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <div>
          {rows.map(([ic, title, body]) => (
            <div key={title} className="tk-ckl">
              {ic}
              <div className="tk-tx">
                <span className="font-semibold">{title}</span>
                <span className="k text-ink-2">{body}</span>
              </div>
            </div>
          ))}
        </div>
        <span className="k text-ink-2">{t(tm.recognisedFoot)}</span>
      </div>
    </section>
  )
}

function Kept() {
  const t = useT()
  const tick = (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="var(--ok-bar)" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="m3 8 3.5 3.5L13 5" />
    </svg>
  )
  const cross = (
    <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="var(--bad-strong)" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 4l8 8M12 4l-8 8" />
    </svg>
  )
  return (
    <section className="card">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2">{t(tm.keptTitle)}</h2>
          <span className="k">{t(tm.keptSub)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3.5">
        <dl className="kv">
          <dt>{t(tm.keptEvents)}</dt>
          <dd>{t(tm.keptEventsValue)}</dd>
          <dt>{t(tm.keptVisitors)}</dt>
          <dd>{t(tm.keptVisitorsValue)}</dd>
          <dt>{t(tm.keptInterest)}</dt>
          <dd>{t(tm.keptInterestValue)}</dd>
          <dt>{t(tm.keptErase)}</dt>
          <dd>{t(tm.keptEraseValue)}</dd>
        </dl>
        <div className="cm-two tk-two">
          <div className="cm-sbox">
            <h4>{t(tm.stored)}</h4>
            <ul>
              {[tm.storedPaths, tm.storedReferrer, tm.storedProducts, tm.storedCustom].map((x) => (
                <li key={x.id}>
                  {tick}
                  <span>{t(x)}</span>
                </li>
              ))}
            </ul>
          </div>
          <div className="cm-sbox">
            <h4>{t(tm.neverStored)}</h4>
            <ul>
              {[tm.neverIp, tm.neverAgents, tm.neverQueries, tm.neverForms].map((x) => (
                <li key={x.id}>
                  {cross}
                  <span>{t(x)}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>
        <div className="note n" role="note">
          <Icon name="info" size={16} className="mt-px flex-none text-ink-3" />
          <div className="min-w-0 flex-[1_1_320px]">{t(tm.gpc)}</div>
        </div>
      </div>
    </section>
  )
}
