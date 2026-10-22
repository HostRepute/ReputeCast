import { useId, useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { DisabledReason } from '@/components/ui/DisabledReason'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { NoResults } from '@/components/ui/NoResults'
import { ConnectChannels } from '@/features/channels/ConnectChannels'
import { ReadOnlyNote } from '@/features/senders/ReadOnlyNote'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { cn } from '@/lib/cn'
import { count } from '@/lib/format'
import { deniedReason, useWorkspace } from '@/lib/session'
import { mc } from './catalog.messages'
import { m } from './messages'
import { StepBadge, StepHeading, WizardBody, WizardLayout } from './Wizard'

type Entry = Schemas['ConnectCatalogEntry']
type Community = Schemas['ConnectCommunityEntry']

/** 7.2 step 1 (SetupCatalog): every adapter this build can connect. */
export function CatalogPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/connect/catalog', { params: { path: { workspace: slug } } })
  const [search, setSearch] = useState('')
  const [bulk, setBulk] = useState(true)
  const [webhooks, setWebhooks] = useState(false)
  const [dedicated, setDedicated] = useState(false)
  const searchId = useId()
  const data = q.data
  const shown = useMemo(() => {
    const needle = search.trim().toLowerCase()
    return (data?.builtin ?? []).filter(
      (e) =>
        (!needle || e.name.toLowerCase().includes(needle) || e.short.toLowerCase().includes(needle)) &&
        (!bulk || e.bulkAllowed) &&
        (!webhooks || e.webhooks) &&
        (!dedicated || e.dedicatedIps),
    )
  }, [data, search, bulk, webhooks, dedicated])
  const reason = data && !data.canConnect ? deniedReason('providers:manage', data.helpers) : undefined
  const clear = () => {
    setSearch('')
    setBulk(false)
    setWebhooks(false)
    setDedicated(false)
  }
  return (
    <WizardLayout title={t(m.title)} badge={<StepBadge n={1} />}>
      <WizardBody current={1}>
        {data && !data.canConnect ? (
          <ReadOnlyNote msg={m.readOnlyCatalog} perm="providers:manage" helpers={data.helpers} />
        ) : null}
        <StepHeading title={t(mc.heading)}>{t(mc.lead)}</StepHeading>
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Email · SMS · …: the other channels' catalogs (X21). */}
          <ConnectChannels current="email" />
          <label className="k flex items-center gap-2" htmlFor={searchId}>
            {t(common.search)}
            <input
              id={searchId}
              className="in w-[220px] max-md:w-auto max-md:min-w-0 max-md:flex-1"
              type="search"
              placeholder={t(mc.searchPlaceholder)}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={bulk} onChange={(e) => setBulk(e.target.checked)} />
            {t(mc.filterBulk)}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={webhooks} onChange={(e) => setWebhooks(e.target.checked)} />
            {t(mc.filterWebhooks)}
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={dedicated} onChange={(e) => setDedicated(e.target.checked)} />
            {t(mc.filterDedicated)}
          </label>
          {data ? <span className="k ml-auto">{t(mc.counts, { builtin: count(data.builtin.length), community: count(data.community.length) })}</span> : null}
        </div>
        {q.isError ? (
          <ErrorPanel title={t(mc.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : !data ? (
          <CardsSkeleton />
        ) : shown.length === 0 ? (
          <NoResults title={t(mc.noMatch)} actions={<Button onClick={clear}>{t(mc.clearFilters)}</Button>}>
            {search.trim() ? t(mc.noMatchSearch, { query: search.trim() }) : t(mc.noMatchFilters)}
          </NoResults>
        ) : (
          <div className="cn-grid grid gap-3.5 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }}>
            {shown.map((e) => (
              <ProviderCard key={e.adapter} entry={e} reason={reason} />
            ))}
          </div>
        )}
        {data && data.community.length > 0 ? <CommunityCard items={data.community} reason={reason} /> : null}
      </WizardBody>
    </WizardLayout>
  )
}

function ProviderCard({ entry: e, reason }: { entry: Entry; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const tone = e.termsTone === 'neutral' ? 'neutral' : e.termsTone
  return (
    <article className={cn('card cn-prov', e.recommended && 'rec')} aria-labelledby={`prov-${e.adapter}`}>
      <div className="t">
        <span className="nm" id={`prov-${e.adapter}`}>
          {e.name}
        </span>
        <Badge tone={tone}>{e.terms}</Badge>
      </div>
      <span className="k">{e.pricing}</span>
      <div className="flex flex-wrap gap-[5px]">
        {e.chips.map((c) => (
          <span key={c.label} className={cn('cn-cap', c.kind === 'yes' && 'y', c.kind === 'no' && 'n', c.kind === 'partial' && 'p')}>
            {c.kind === 'no' ? <T msg={mc.notSupported} values={{ capability: c.label, vh: (chunks) => <span className="vh">{chunks}</span> }} /> : c.label}
          </span>
        ))}
      </div>
      <DisabledReason reason={reason}>
        <ButtonLink to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: e.adapter }} variant={e.recommended ? 'primary' : 'default'}>
          {t(mc.connectProvider, { provider: e.short })}
        </ButtonLink>
      </DisabledReason>
    </article>
  )
}

function CommunityCard({ items, reason }: { items: Community[]; reason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col" aria-labelledby="community-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h3 className="h2" id="community-h">
            {t(mc.communityTitle)}
          </h3>
          <span className="k">{t(mc.communityLine)}</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
          {t(mc.manageAdapters)}
        </ButtonLink>
      </div>
      {/* As on the phone board: two 140 px columns (the provider cards above stay one per row there, where the board breaks names letter by letter). */}
      <div className="grid max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }}>
        {items.map((c, i) => (
          <div key={c.id} className={cn('flex flex-col gap-1.5 px-4.5 py-3.5', i < items.length - 1 && 'border-r border-line-2')}>
            <div className="flex items-center justify-between gap-2">
              <span className="font-semibold">{c.name}</span>
              <span className="mono k">{c.version}</span>
            </div>
            <span className="k">{c.summary}</span>
            {c.configure ? (
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
                {t(mc.configure)}
              </ButtonLink>
            ) : (
              <DisabledReason reason={reason ?? (c.adapter ? undefined : c.reason)}>
                <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: c.adapter || c.name }}>
                  {t(mc.connect)}
                </ButtonLink>
              </DisabledReason>
            )}
          </div>
        ))}
      </div>
    </section>
  )
}

function CardsSkeleton() {
  const t = useT()
  return (
    <div className="grid gap-3.5 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }} aria-busy="true">
      <span className="vh" role="status">
        {t(mc.loading)}
      </span>
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="card cn-prov" aria-hidden="true">
          <Skeleton w="45%" h={16} />
          <Skeleton w="90%" h={12} />
          <Skeleton w="80%" h={44} />
          <Skeleton w={120} h={36} r={8} />
        </div>
      ))}
    </div>
  )
}
