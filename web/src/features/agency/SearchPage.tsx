import { Link, useSearch } from '@tanstack/react-router'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { formatList, useT } from '@/i18n'
import { useMe, useWorkspace } from '@/lib/session'
import { useAgencyMe } from './api'
import { AgencyPage, AgencySearchBox } from './AgencyShell'
import { kindLabel } from './kinds'
import { m } from './messages'
import { ClientTile, dayMonth } from './parts'
import { m as sm } from './search.messages'

type Hit = Schemas['AgencySearchHit']

const statusBadge: Record<string, { tone: Tone; label: typeof sm.stApproval }> = {
  approval: { tone: 'warn', label: sm.stApproval },
  draft: { tone: 'neutral', label: sm.stDraft },
  scheduled: { tone: 'info', label: sm.stScheduled },
  approved: { tone: 'ok', label: sm.stApproved },
  sending: { tone: 'info', label: sm.stSending },
  sent: { tone: 'ok', label: sm.stSent },
  paused: { tone: 'warn', label: sm.stPaused },
  live: { tone: 'ok', label: sm.stLive },
  published: { tone: 'ok', label: sm.stLive },
}

/** Search across clients (AgencyStates): names everywhere, contacts by exact address, audited. */
export function SearchPage() {
  const t = useT()
  const { q } = useSearch({ strict: false }) as { q?: string }
  const query = (q ?? '').trim()
  const res = $api.useQuery('get', '/api/v1/agency/search', { params: { query: { q: query } } }, { enabled: query.length >= 2 })
  const { data: me } = useMe()
  const { data: agency } = useAgencyMe()
  const name = me?.user.name || me?.user.email || ''
  const d = res.data
  return (
    <AgencyPage title={t(sm.title)} search={false}>
      <section className="card max-w-[760px]" aria-label={t(sm.title)}>
        <div className="flex flex-col gap-2 px-4 pt-4 pb-3">
          <AgencySearchBox key={query} initial={query} width={9999} />
          {d ? (
            <span className="k">
              {d.contact
                ? t(sm.contactCaption, { name })
                : agency?.staff?.level === 'agency_manager'
                  ? t(sm.nameCaptionAll, { name, n: d.total })
                  : t(sm.nameCaption, { name, clients: agency?.clientCount ?? 0, n: d.total })}
            </span>
          ) : null}
        </div>
        {query.length < 2 ? (
          <p className="k m-0 border-t border-line-2 px-4 py-4">{t(sm.typeMore)}</p>
        ) : res.isError ? (
          <div className="border-t border-line-2 p-4">
            <ErrorPanel title={t(sm.failed)} actions={<Button onClick={() => res.refetch()}>{t(m.retry)}</Button>}>
              {errorMessage(res.error)}
            </ErrorPanel>
          </div>
        ) : !d ? (
          <div className="flex flex-col gap-2 border-t border-line-2 p-4" aria-busy="true">
            <Skeleton w="50%" h={14} />
            <Skeleton w="80%" h={14} />
            <Skeleton w="65%" h={14} />
          </div>
        ) : d.total === 0 ? (
          <p className="m-0 border-t border-line-2 px-4 py-4">{d.contact ? t(sm.noContact, { q: query, n: d.searched }) : t(sm.noMatch, { q: query })}</p>
        ) : (
          <>
            {d.groups.map((g) => (
              <div key={g.workspace.id} className="flex flex-col border-t border-line-2 px-4 py-3">
                <span className="mb-1.5 flex items-center gap-2">
                  <ClientTile ws={g.workspace} size={22} />
                  <b>{g.workspace.name}</b>
                </span>
                {g.hits.map((h) => (
                  <HitRow key={h.kind + h.id} hit={h} slug={g.workspace.slug} />
                ))}
              </div>
            ))}
            {d.library.length ? (
              <div className="flex flex-col border-t border-line-2 px-4 py-3">
                <span className="mb-1.5 flex items-center gap-2">
                  <span className="inline-flex h-[22px] w-[22px] items-center justify-center rounded-md text-accent" style={{ background: '#E3F1EE' }} aria-hidden="true">
                    <Icon name="layers" size={13} />
                  </span>
                  <b>{t(sm.libraryGroup, { agency: agency?.name ?? '' })}</b>
                </span>
                {d.library.map((h) => (
                  <LibraryHit key={h.id} hit={h} />
                ))}
              </div>
            ) : null}
            {d.skipped ? <p className="k m-0 border-t border-line-2 px-4 py-3">{t(sm.skipped, { n: d.skipped })}</p> : null}
          </>
        )}
        {d?.contact ? (
          <div className="border-t border-line-2 p-4">
            <Note tone="info">{t(sm.audited)}</Note>
          </div>
        ) : null}
      </section>
    </AgencyPage>
  )
}

function hitLink(hit: Hit, slug: string): { to: string; params: Record<string, string> } {
  switch (hit.kind) {
    case 'campaign':
      return hit.status === 'draft'
        ? { to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: hit.id } }
        : hit.status === 'sent'
          ? { to: '/w/$workspace/campaigns/$campaignId/report', params: { workspace: slug, campaignId: hit.id } }
          : { to: '/w/$workspace/campaigns/$campaignId/review', params: { workspace: slug, campaignId: hit.id } }
    case 'automation':
      return { to: '/w/$workspace/automations/$automationId', params: { workspace: slug, automationId: hit.id } }
    case 'contact':
      return { to: '/w/$workspace/audience/contacts/$contactId', params: { workspace: slug, contactId: hit.id } }
    case 'form':
      return { to: '/w/$workspace/forms', params: { workspace: slug } }
    default:
      return { to: '/w/$workspace/campaigns', params: { workspace: slug } }
  }
}

function HitRow({ hit, slug }: { hit: Hit; slug: string }) {
  const t = useT()
  const link = hitLink(hit, slug)
  const kind =
    hit.kind === 'campaign'
      ? t(sm.kindCampaign)
      : hit.kind === 'automation'
        ? t(sm.kindAutomation)
        : hit.kind === 'section'
          ? t(sm.kindSection)
          : hit.kind === 'template'
            ? t(sm.kindTemplate)
            : hit.kind === 'form'
              ? t(sm.kindForm)
              : t(sm.kindContact)
  let detail = kind
  if (hit.kind === 'contact') {
    const lists = hit.lists?.length ? formatList(hit.lists) : hit.leftLists?.length ? formatList(hit.leftLists) : ''
    if (hit.lists?.length && hit.subscribedAt) detail = t(sm.contactSubscribed, { lists, date: longDay(hit.subscribedAt) })
    else if (hit.unsubscribedAt) detail = t(sm.contactLeft, { lists, date: longDay(hit.unsubscribedAt) })
    else if (lists) detail = t(sm.contactLists, { lists })
  }
  const st = hit.status ? statusBadge[hit.status] : undefined
  return (
    <Link to={link.to as never} params={link.params as never} className="flex items-center gap-3 rounded-lg px-1 py-1.5 text-ink! no-underline hover:bg-ground-2">
      <span className="flex min-w-0 flex-1 flex-col">
        <b className="font-semibold">{hit.name}</b>
        <span className="k">{detail}</span>
      </span>
      {st ? <Badge tone={st.tone}>{t(st.label)}</Badge> : null}
    </Link>
  )
}

function LibraryHit({ hit }: { hit: Hit }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Link
      to="/w/$workspace/agency/library"
      params={{ workspace: slug }}
      search={{ item: hit.id }}
      className="flex items-center gap-3 rounded-lg px-1 py-1.5 text-ink! no-underline hover:bg-ground-2"
    >
      <span className="flex min-w-0 flex-1 flex-col">
        <b className="font-semibold">{hit.name}</b>
        <span className="k">{t(sm.libraryHit, { kind: kindLabel(t, hit.libraryKind ?? 'section'), version: hit.version ?? 1 })}</span>
      </span>
    </Link>
  )
}

function longDay(iso: string) {
  return `${dayMonth(iso)} ${new Date(iso).getFullYear()}`
}
