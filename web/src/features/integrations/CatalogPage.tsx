import { useNavigate } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { errorMessage } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { settingsTabs } from '@/features/settings/SettingsTabs'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { ago } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { ConnectDialog } from './ConnectDialog'
import { m } from './messages'
import {
  API_REFERENCE,
  AttentionIcon,
  capabilities,
  entryLine,
  entryName,
  helpersText,
  isPlatform,
  maintainer,
  reasonWhy,
  ReasonSentence,
  stamp,
  StatusBadge,
  tags,
  Tile,
  useHub,
  type Entry,
  type GroupKey,
  type Hub,
  type PlatformKind,
} from './shared'

type Filter = 'all' | 'connected' | 'attention' | GroupKey

const groupName: Record<GroupKey, MessageDescriptor> = {
  automation: m.groupAutomation,
  websites: m.groupWebsites,
  deliverability: m.groupDeliverability,
  developer: m.groupDeveloper,
  ai: m.groupAi,
  verification: m.groupVerification,
  payments: m.groupPayments,
  commerce: m.groupCommerce,
}

const groupLine: Record<GroupKey, MessageDescriptor> = {
  automation: m.groupAutomationLine,
  websites: m.groupWebsitesLine,
  deliverability: m.groupDeliverabilityLine,
  developer: m.groupDeveloperLine,
  ai: m.groupAiLine,
  verification: m.groupVerificationLine,
  payments: m.groupPaymentsLine,
  commerce: m.groupCommerceLine,
}

/** Groups of one card sit side by side (Websites, Deliverability); the others take the full width. */
const narrowGroups: GroupKey[] = ['websites', 'deliverability']

/** At most three banners; beyond that, one line says "and 2 more". */
const MAX_BANNERS = 3

/** Settings › Integrations: the catalog (IntegrationsCatalog, IntegrationsStates). */
export function CatalogPage() {
  const { slug } = useWorkspace()
  const t = useT()
  const q = useHub()
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<Filter>('all')
  const [connecting, setConnecting] = useState<PlatformKind | null>(null)
  const navigate = useNavigate()

  return (
    <Page title={t(m.settings)} documentTitle={t(m.documentTitle)} tabs={settingsTabs(slug)}>
      <div className="row items-end justify-between gap-x-5 gap-y-3">
        <div className="flex min-w-0 flex-[1_1_420px] flex-col gap-1">
          <h2 className="ih-ttl">{t(m.title)}</h2>
          <p className="ih-lead">{t(m.lead)}</p>
        </div>
        <label className="ih-search">
          <Icon name="search" size={15} />
          <input value={search} onChange={(e) => setSearch(e.target.value)} aria-label={t(m.searchLabel)} placeholder={t(m.searchLabel)} type="search" />
        </label>
      </div>
      {q.isError ? (
        <LoadError detail={errorMessage(q.error)} onRetry={() => void q.refetch()} />
      ) : !q.data ? (
        <CatalogSkeleton />
      ) : (
        <Catalog
          hub={q.data}
          search={search}
          setSearch={setSearch}
          filter={filter}
          setFilter={setFilter}
          onConnect={(k) => setConnecting(k)}
        />
      )}
      <div className="row justify-center gap-x-2.5 gap-y-2 py-1 pb-1.5 text-center">
        <Icon name="code" size={15} className="text-ink-3" />
        <span className="text-[13px] text-ink-2">{t(m.footer)}</span>
        <a href={API_REFERENCE} className="ih-lnk" target="_blank" rel="noopener noreferrer">
          {t(m.apiReference)} <Icon name="external" size={13} />
          <span className="vh">{t(m.newTab)}</span>
        </a>
      </div>
      {connecting ? (
        <ConnectDialog
          kind={connecting}
          open
          onOpenChange={(o) => !o && setConnecting(null)}
          onDone={() => {
            const kind = connecting
            setConnecting(null)
            void navigate({ to: '/w/$workspace/settings/integrations/$kind', params: { workspace: slug, kind } })
          }}
        />
      ) : null}
    </Page>
  )
}

function Catalog({
  hub,
  search,
  setSearch,
  filter,
  setFilter,
  onConnect,
}: {
  hub: Hub
  search: string
  setSearch: (s: string) => void
  filter: Filter
  setFilter: (f: Filter) => void
  onConnect: (k: PlatformKind) => void
}) {
  const t = useT()
  const query = search.trim().toLocaleLowerCase()

  // Every word a card shows, so search finds "postmaster" or "shortcode" too.
  const words = (e: Entry) =>
    [entryName[e.key], maintainer[e.key], entryLine[e.key], ...capabilities[e.key]]
      .map((d) => t(d))
      .join(' ')
      .toLocaleLowerCase()

  const groups = useMemo(
    () =>
      hub.groups
        .filter((g) => filter === 'all' || filter === 'connected' || filter === 'attention' || g.key === filter)
        .map((g) => ({
          ...g,
          entries: g.entries.filter((e) => {
            if (filter === 'connected' && !(e.status === 'connected' || e.status === 'waiting' || e.status === 'attention')) return false
            if (filter === 'attention' && e.status !== 'attention') return false
            if (query && !words(e).includes(query) && !t(groupName[g.key]).toLocaleLowerCase().includes(query)) return false
            return true
          }),
        }))
        .filter((g) => g.entries.length > 0),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [hub, filter, query, t],
  )

  const entryOf = (key: string) => hub.groups.flatMap((g) => g.entries).find((e) => e.key === key)
  const banners = hub.alerts.slice(0, MAX_BANNERS)
  const more = hub.alerts.length - banners.length

  return (
    <>
      {banners.length ? (
        <div className="flex flex-col gap-2">
          {banners.map((a, i) => (
            <AlertBanner key={i} alert={a} entry={entryOf(a.entry)} />
          ))}
          {more > 0 ? (
            <div className="row gap-2">
              <button type="button" className="ih-lnk cursor-pointer border-0 bg-transparent p-0 text-accent" onClick={() => setFilter('attention')}>
                {t(m.moreProblems, { count: more })}
              </button>
            </div>
          ) : null}
        </div>
      ) : null}
      {!hub.canManage ? (
        <div className="note n" role="status">
          <Badge tone="neutral">{t(m.readOnly)}</Badge>
          <span>{hub.helpers.length ? t(m.readOnlyCatalog, { helpers: helpersText(hub.helpers) }) : t(m.readOnlyCatalogAlone)}</span>
        </div>
      ) : null}
      <div className="ih-chips" role="group" aria-label={t(m.filterLabel)}>
        <Chip on={filter === 'all'} onClick={() => setFilter('all')}>
          {t(m.filterAll)}
        </Chip>
        <Chip on={filter === 'connected'} onClick={() => setFilter('connected')}>
          {t(m.filterConnected)}
        </Chip>
        {hub.groups.map((g) => (
          <Chip key={g.key} on={filter === g.key} onClick={() => setFilter(g.key)}>
            {t(groupName[g.key])}
          </Chip>
        ))}
      </div>
      {filter === 'attention' ? (
        <div className="row gap-2 text-[13px] text-ink-2" role="status">
          <span>{t(m.attentionOnly)}</span>
          <button type="button" className="ih-lnk cursor-pointer border-0 bg-transparent p-0 text-accent" onClick={() => setFilter('all')}>
            {t(m.showAll)}
          </button>
        </div>
      ) : null}
      {groups.length === 0 ? (
        <NoResults query={search.trim()} onClear={() => setSearch('')} />
      ) : (
        <div className="ih-page">
          {groups.map((g) => {
            const full = !narrowGroups.includes(g.key)
            const cards = g.entries.map((e) => <EntryCard key={e.key} entry={e} canManage={hub.canManage} onConnect={onConnect} />)
            return (
              <section key={g.key} className={`ih-group${full ? ' ih-full' : ''}`} aria-labelledby={`g-${g.key}`}>
                <div className="ih-gh">
                  <div className="ih-gt">
                    <h2 id={`g-${g.key}`}>{t(groupName[g.key])}</h2>
                    <span className="k">{t(groupLine[g.key])}</span>
                  </div>
                </div>
                {full ? <div className="ih-grid">{cards}</div> : cards}
              </section>
            )
          })}
        </div>
      )}
    </>
  )
}

function Chip({ on, onClick, children }: { on: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" className="ih-chip" aria-pressed={on} onClick={onClick}>
      {children}
    </button>
  )
}

/** One problem, its sentence and the way to it (IntegrationsCatalog's banner). */
function AlertBanner({ alert, entry }: { alert: Hub['alerts'][number]; entry?: Entry }) {
  const { slug } = useWorkspace()
  const t = useT()
  const name = t(entryName[alert.entry as Entry['key']] ?? m.title)
  const params = { workspace: slug }
  const action =
    alert.entry === 'wordpress' ? (
      <ButtonLink size="sm" to="/w/$workspace/settings/integrations/wordpress" params={params}>
        {alert.reason.code === 'site_silent' ? t(m.checkTheSite) : t(m.openEntry, { name })}
      </ButtonLink>
    ) : isPlatform(alert.entry) ? (
      <ButtonLink size="sm" to="/w/$workspace/settings/integrations/$kind" params={{ workspace: slug, kind: alert.entry }}>
        {t(m.openEntry, { name })}
      </ButtonLink>
    ) : null
  const bad = alert.reason.code === 'key_revoked' || alert.reason.code === 'key_expired' || alert.reason.code === 'owner_left'
  void entry
  return (
    <div className={`ih-banner${bad ? ' b' : ''}`} role="status">
      <div className="ih-bi">
        <AttentionIcon color={bad ? 'var(--bad-fg)' : undefined} />
        <span className="ih-bt">
          <ReasonSentence r={alert.reason} name={name} />
        </span>
      </div>
      {action}
    </div>
  )
}

/** One catalog card: tile, name, status, line, capabilities, the reason it needs attention, a fact and an action. */
function EntryCard({ entry: e, canManage, onConnect }: { entry: Entry; canManage: boolean; onConnect: (k: PlatformKind) => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const canHostRepute = useCan('integrations:hostrepute')
  const params = { workspace: slug }
  const name = t(entryName[e.key])
  const invite = e.key === 'hostrepute' && e.status === 'not_connected'
  const builtIn = e.key === 'api' || e.key === 'webhooks' || e.key === 'mcp'
  const reason = e.status === 'attention' ? e.reasons[0] : undefined

  let action: ReactNode = null
  if (isPlatform(e.key)) {
    const kind = e.key
    action =
      e.status === 'not_connected' && canManage ? (
        <Button size="sm" onClick={() => onConnect(kind)}>
          {t(m.connect)}
        </Button>
      ) : (
        <ButtonLink size="sm" to="/w/$workspace/settings/integrations/$kind" params={{ workspace: slug, kind }}>
          {canManage ? (e.status === 'not_connected' ? t(m.connect) : t(m.manage)) : t(m.view)}
        </ButtonLink>
      )
  } else if (e.key === 'wordpress') {
    action =
      e.status === 'not_connected' && canManage ? (
        <ButtonLink size="sm" to="/w/$workspace/settings/integrations/wordpress" params={params} search={{ connect: true }}>
          {t(m.connect)}
        </ButtonLink>
      ) : (
        <ButtonLink size="sm" to="/w/$workspace/settings/integrations/wordpress" params={params}>
          {canManage ? t(m.manage) : t(m.view)}
        </ButtonLink>
      )
  } else if (e.key === 'hostrepute') {
    action = invite ? (
      canHostRepute ? (
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/deliverability/connect" params={params}>
          {t(m.connectHostRepute)}
        </ButtonLink>
      ) : (
        <ButtonLink size="sm" to="/w/$workspace/settings/integrations/hostrepute" params={params}>
          {t(m.view)}
        </ButtonLink>
      )
    ) : (
      <ButtonLink size="sm" to="/w/$workspace/settings/integrations/hostrepute" params={params}>
        {canHostRepute ? t(m.manage) : t(m.view)}
      </ButtonLink>
    )
  } else if (e.key === 'api') {
    action = (
      <ButtonAnchor size="sm" href={API_REFERENCE} iconRight="external">
        {t(m.open)}
        <span className="vh">{t(m.newTab)}</span>
      </ButtonAnchor>
    )
  } else if (e.key === 'webhooks') {
    action = (
      <ButtonLink size="sm" to="/w/$workspace/settings/webhooks" params={params}>
        {e.facts.endpoints ? t(m.manage) : t(m.addEndpoint)}
      </ButtonLink>
    )
  } else if (e.key === 'mcp') {
    action = (
      <ButtonLink size="sm" to="/w/$workspace/settings/mcp" params={params}>
        {e.facts.mcpEnabled || e.facts.agents ? t(m.manage) : t(m.setUp)}
      </ButtonLink>
    )
  }

  return (
    <article className={`card ih-card${invite ? ' inv' : ''}`} aria-label={name}>
      <div className="ih-head">
        <Tile k={e.key} />
        <div className="ih-name">
          <h3>{name}</h3>
          <span className="k">{t(invite ? m.byHostReputeOptional : maintainer[e.key])}</span>
        </div>
        {builtIn ? null : <StatusBadge status={e.status} />}
      </div>
      <p className="ih-desc">{t(invite ? m.hostreputeInvite : entryLine[e.key])}</p>
      <div className="ih-caps">
        {capabilities[e.key].map((c) => (
          <span key={c.id} className="ih-cap">
            {t(c)}
          </span>
        ))}
      </div>
      {reason ? (
        <div className="ih-why">
          <AttentionIcon size={14} />
          <span>{reasonWhy(t, reason)}</span>
        </div>
      ) : null}
      <div className="ih-ft">
        <span className="k">
          <Fact entry={e} />
        </span>
        {action}
      </div>
    </article>
  )
}

/** The card's one fact line: "3 Zaps · last call 2 min ago", "2 sites · 1 not heard from". */
function Fact({ entry: e }: { entry: Entry }) {
  const t = useT()
  const f = e.facts
  switch (e.key) {
    case 'zapier':
    case 'make':
    case 'n8n':
      if (e.status === 'not_connected') return <>{t(m.factCounts, { triggers: f.triggers, actions: f.actions })}</>
      if (e.status === 'waiting' || !f.lastCallAt) return <>{t(m.factNoCallYet, { time: f.keyCreatedAt ? stamp(f.keyCreatedAt) : '' })}</>
      return <>{t(m.factWorkflows, { kind: e.key, count: f.workflows, ago: ago(f.lastCallAt) })}</>
    case 'wordpress': {
      if (!f.sites) return <>{t(m.factInstallPlugin)}</>
      const parts = [t(m.factSites, { count: f.sites })]
      if (f.silentSites) parts.push(t(m.factSilent, { count: f.silentSites }))
      if (f.updates) parts.push(t(m.factUpdates, { count: f.updates }))
      return <>{parts.join(' · ')}</>
    }
    case 'hostrepute':
      return e.status === 'not_connected' ? <>{t(m.factNoPaywall)}</> : <>{t(m.factSynced, { team: f.team ?? '', ago: ago(f.lastSyncAt) })}</>
    case 'api':
      return <>{t(m.factApi)}</>
    case 'webhooks':
      if (!f.endpoints) return <>{t(m.factNoEndpoints)}</>
      return f.failing ? <T msg={m.factEndpointsFailing} values={{ ...tags, count: f.endpoints, failing: f.failing }} /> : <>{t(m.factEndpoints, { count: f.endpoints })}</>
    case 'mcp':
      if (!f.mcpEnabled) return <>{t(m.factMcpOff)}</>
      return f.agents ? <>{t(m.factMcpOn, { count: f.agents })}</> : <>{t(m.factMcpOnNone)}</>
  }
  return null
}

function NoResults({ query, onClear }: { query: string; onClear: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  const params = (kind: PlatformKind) => ({ workspace: slug, kind })
  return (
    <section className="card ih-empty" aria-live="polite">
      <Icon name="searchLarge" size={28} className="text-ink-3" />
      <span className="ih-et">{t(m.noResultsTitle, { query })}</span>
      <span className="ih-ed">{t(m.noResultsBody)}</span>
      <div className="row justify-center gap-2">
        <Button onClick={onClear}>{t(m.clearSearch)}</Button>
        <ButtonLink to="/w/$workspace/settings/integrations/$kind" params={params('zapier')}>
          {t(m.zapier)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/settings/integrations/$kind" params={params('make')}>
          {t(m.make)}
        </ButtonLink>
        <ButtonLink to="/w/$workspace/settings/integrations/$kind" params={params('n8n')}>
          {t(m.n8n)}
        </ButtonLink>
      </div>
      <span className="k">
        <T
          msg={m.buildYourOwn}
          values={{
            a: (c) => (
              <a href={API_REFERENCE} target="_blank" rel="noopener noreferrer">
                {c}
              </a>
            ),
          }}
        />
      </span>
    </section>
  )
}

/** Loading: two placeholder cards (IntegrationsStates). */
function CatalogSkeleton() {
  const t = useT()
  const card = (
    <div className="card ih-card" aria-hidden="true">
      <div className="ih-head">
        <Skeleton w={40} h={40} r={10} />
        <div className="ih-name gap-1.5">
          <Skeleton w={110} h={14} />
          <Skeleton w={90} />
        </div>
        <Skeleton w={84} h={22} r={999} />
      </div>
      <Skeleton w="100%" />
      <Skeleton w="70%" />
      <div className="ih-caps">
        <Skeleton w={96} h={22} r={5} />
        <Skeleton w={60} h={22} r={5} />
      </div>
      <div className="ih-ft">
        <Skeleton w={150} />
        <Skeleton w={70} h={30} r={8} />
      </div>
    </div>
  )
  return (
    <div className="flex flex-col gap-4" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loadingCatalog)}
      </span>
      <div className="row gap-1.5">
        <Skeleton w={60} h={30} r={999} />
        <Skeleton w={96} h={30} r={999} />
        <Skeleton w={150} h={30} r={999} />
      </div>
      <div className="ih-grid">
        {card}
        {card}
        {card}
      </div>
    </div>
  )
}

function LoadError({ detail, onRetry }: { detail: string; onRetry: () => void }) {
  const { slug } = useWorkspace()
  const t = useT()
  return (
    <section className="card flex flex-col items-start gap-2.5 px-5 py-4.5" role="alert">
      <span className="row gap-2 font-semibold">
        <Icon name="alertCircle" size={18} className="text-bad-fg" />
        {t(m.loadError)}
      </span>
      <span className="k">{t(m.loadErrorBody, { detail })}</span>
      <div className="row gap-2">
        <Button variant="primary" onClick={onRetry}>
          {t(m.retry)}
        </Button>
        <ButtonLink to="/w/$workspace" params={{ workspace: slug }}>
          {t(m.statusOnOverview)}
        </ButtonLink>
      </div>
    </section>
  )
}
