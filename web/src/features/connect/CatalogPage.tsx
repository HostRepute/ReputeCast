import { useId, useMemo, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { DisabledReason } from '@/components/ui/DisabledReason'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { NoResults } from '@/components/ui/NoResults'
import { cn } from '@/lib/cn'
import { deniedReason, useWorkspace } from '@/lib/session'
import { StepBadge, StepHeading, WizardBody, WizardLayout } from './Wizard'

type Entry = Schemas['ConnectCatalogEntry']
type Community = Schemas['ConnectCommunityEntry']

/** 7.2 step 1 (SetupCatalog): every adapter this build can connect. */
export function CatalogPage() {
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
  const reason = data && !data.canConnect ? deniedReason('providers:manage') : undefined
  const clear = () => {
    setSearch('')
    setBulk(false)
    setWebhooks(false)
    setDedicated(false)
  }
  return (
    <WizardLayout title="Connect a sending provider" badge={<StepBadge n={1} />}>
      <WizardBody current={1}>
        <StepHeading title="Which provider delivers your mail?">
          Every adapter reports what it can do. Badges come from each adapter&apos;s manifest; terms are reviewed quarterly.
        </StepHeading>
        <div className="flex flex-wrap items-center gap-2.5">
          <label className="k flex items-center gap-2" htmlFor={searchId}>
            Search
            <input
              id={searchId}
              className="in w-[220px] max-md:w-auto max-md:min-w-0 max-md:flex-1"
              type="search"
              placeholder="Provider name"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={bulk} onChange={(e) => setBulk(e.target.checked)} />
            Allows bulk marketing
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={webhooks} onChange={(e) => setWebhooks(e.target.checked)} />
            Has bounce webhooks
          </label>
          <label className="flex items-center gap-2 text-[13px]">
            <input type="checkbox" checked={dedicated} onChange={(e) => setDedicated(e.target.checked)} />
            Dedicated IPs
          </label>
          {data ? <span className="k ml-auto">{`${data.builtin.length} built-in · ${data.community.length} community`}</span> : null}
        </div>
        {q.isError ? (
          <ErrorPanel title="Couldn't load the providers" actions={<Button onClick={() => void q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        ) : !data ? (
          <CardsSkeleton />
        ) : shown.length === 0 ? (
          <NoResults title="No provider matches" actions={<Button onClick={clear}>Clear the filters</Button>}>
            {search.trim() ? `Nothing called “${search.trim()}” with the filters you picked.` : 'No provider has every capability you picked.'}
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
            {c.kind === 'no' ? <span className="vh">Not supported: </span> : null}
            {c.label}
          </span>
        ))}
      </div>
      <DisabledReason reason={reason}>
        <ButtonLink to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: e.adapter }} variant={e.recommended ? 'primary' : 'default'}>
          {`Connect ${e.short}`}
        </ButtonLink>
      </DisabledReason>
    </article>
  )
}

function CommunityCard({ items, reason }: { items: Community[]; reason?: string }) {
  const { slug } = useWorkspace()
  return (
    <section className="card flex flex-col" aria-labelledby="community-h">
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-line-2 px-4.5 py-3.5">
        <div className="flex flex-col">
          <h3 className="h2" id="community-h">
            Community adapters
          </h3>
          <span className="k">Built with the adapter SDK. Review the maintainer and version before connecting.</span>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
          Manage adapters
        </ButtonLink>
      </div>
      <div className="grid max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }}>
        {items.map((c, i) => (
          <div key={c.id} className={cn('flex flex-col gap-1.5 px-4.5 py-3.5', i < items.length - 1 && 'border-r border-line-2')}>
            <div className="flex items-center justify-between gap-2">
              <span className="font-semibold">{c.name}</span>
              <span className="mono k">{c.version}</span>
            </div>
            <span className="k">{c.summary}</span>
            {c.configure ? (
              <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/adapters" params={{ workspace: slug }}>
                Configure
              </ButtonLink>
            ) : (
              <DisabledReason reason={reason ?? (c.adapter ? undefined : c.reason)}>
                <ButtonLink size="sm" className="self-start" to="/w/$workspace/providers/connect/$adapter" params={{ workspace: slug, adapter: c.adapter || c.name }}>
                  Connect
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
  return (
    <div className="grid gap-3.5 max-md:grid-cols-1!" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(min(300px, 100%), 1fr))' }} aria-busy="true">
      <span className="vh" role="status">
        Loading the providers…
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
