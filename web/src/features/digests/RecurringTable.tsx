import { Link } from '@tanstack/react-router'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Button, ButtonLink } from '@/components/ui/Button'
import { EmptyState, ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Table, Td, Th } from '@/components/ui/Table'
import { useT } from '@/i18n'
import { pct, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { DigestIcon, DigestStateBadge, scheduleText, signed } from './shared'

/** The digests' count for the Campaigns tabs ("Recurring · 1"). */
export function useDigestCount(): number | undefined {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests', { params: { path: { workspace: slug } } })
  return q.data?.items.length
}

/** Campaigns › Recurring: each digest with its status, schedule, next issue, last issue's human opens and owner. */
export function RecurringTable({ onNew, newReason }: { onNew: () => void; newReason?: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests', { params: { path: { workspace: slug } } })
  if (q.isError) {
    return (
      <ErrorPanel title={t(m.recurringLoadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
        {errorMessage(q.error)}
      </ErrorPanel>
    )
  }
  if (q.isPending) {
    return (
      <section className="card" aria-busy="true">
        <span className="vh" role="status">
          {t(m.loading)}
        </span>
        {Array.from({ length: 3 }, (_, i) => (
          <div key={i} className="flex items-center gap-6 border-b border-line-2 px-5 py-4 last:border-b-0">
            <Skeleton w={180} h={12} />
            <Skeleton w={110} h={20} r={999} />
            <Skeleton w={120} h={12} />
            <Skeleton w={90} h={12} className="ml-auto" />
          </div>
        ))}
      </section>
    )
  }
  const items = q.data.items
  if (items.length === 0) {
    return (
      <EmptyState
        title={t(m.recurringEmptyTitle)}
        action={
          newReason ? null : (
            <Button variant="primary" onClick={onNew}>
              {t(m.newDigest)}
            </Button>
          )
        }
      >
        {t(m.recurringEmptyBody)}
      </EmptyState>
    )
  }
  return (
    <section className="card" aria-label={t(m.recurringTab)}>
      <Table phoneWrap className="[&_td]:align-top [&_td:first-child]:min-w-[200px] [&_td:first-child]:whitespace-normal">
        <thead>
          <tr>
            <Th>{t(m.colDigest)}</Th>
            <Th>{t(m.colStatus)}</Th>
            <Th>{t(m.colSchedule)}</Th>
            <Th>{t(m.colNextIssue)}</Th>
            <Th align="right">{t(m.colLastOpens)}</Th>
            <Th>{t(m.colOwner)}</Th>
            <Th>
              <span className="vh">{t(m.open)}</span>
            </Th>
          </tr>
        </thead>
        <tbody>
          {items.map((d) => (
            <RecurringRow key={d.id} d={d} />
          ))}
        </tbody>
      </Table>
    </section>
  )
}

function RecurringRow({ d }: { d: Schemas['DigestListItem'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug, digestId: d.id }
  const source = d.source.categories.length ? `${d.source.label} · ${d.source.categories.join(', ')}` : d.source.label
  const next = d.next
  return (
    <tr>
      <Td>
        <div className="flex flex-col gap-0.5">
          <Link to="/w/$workspace/campaigns/digests/$digestId" params={params} className="font-semibold text-ink no-underline">
            {d.name}
          </Link>
          <span className="k">{source}</span>
        </div>
      </Td>
      <Td>
        <DigestStateBadge state={d.state} mode={d.mode} />
      </Td>
      <Td>
        <div className="flex flex-col gap-0.5">
          <span>{scheduleText(d.schedule, true)}</span>
          <span className="k">{d.schedule.timezone}</span>
        </div>
      </Td>
      <Td>
        {next ? (
          <div className="flex flex-col gap-0.5">
            <span>{next.sendAt ? t(m.nextIssueCell, { number: next.number, date: shortDate(next.sendAt) }) : t(m.nextIssueCellNoDate, { number: next.number, count: d.schedule.count ?? 3 })}</span>
            <span className="k">{next.needsApproval ? t(m.postsSoFarApproval, { count: next.posts }) : t(m.postsSoFar, { count: next.posts })}</span>
          </div>
        ) : (
          <span className="k">–</span>
        )}
      </Td>
      <Td align="right">
        {d.last?.humanOpens != null ? (
          <div className="flex flex-col items-end gap-0.5">
            <span className="n font-semibold">{pct(d.last.humanOpens)}</span>
            {d.last.humanOpensDelta != null ? (
              <span className="k">{t(m.lastIssueCell, { number: d.last.number, delta: t(m.pts, { delta: signed(d.last.humanOpensDelta, 1) }) })}</span>
            ) : null}
          </div>
        ) : (
          <span className="k">{t(m.noIssuesYet)}</span>
        )}
      </Td>
      <Td>
        {d.owner ? (
          <span className="flex items-center gap-2 whitespace-nowrap">
            <Avatar name={d.owner.name} color={d.owner.color} size={22} />
            {d.owner.name}
          </span>
        ) : (
          <span className="k">–</span>
        )}
      </Td>
      <Td>
        <div className="row-actions flex flex-wrap justify-end gap-1.5">
          <ButtonLink size="sm" to="/w/$workspace/campaigns/digests/$digestId" params={params}>
            {t(m.open)}
          </ButtonLink>
          <ButtonLink size="sm" to="/w/$workspace/campaigns/digests/$digestId/settings" params={params}>
            {t(m.settings)}
          </ButtonLink>
        </div>
      </Td>
    </tr>
  )
}

/** The "Digest" badge on an issue's row in the campaigns list: it links to its digest. */
export function DigestBadgeLink({ digest }: { digest: { id: string; name: string } }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Link
      to="/w/$workspace/campaigns/digests/$digestId"
      params={{ workspace: slug, digestId: digest.id }}
      className="badge mut no-underline"
      aria-label={t(m.digestBadgeLabel, { name: digest.name })}
    >
      <DigestIcon />
      {t(m.digestBadge)}
    </Link>
  )
}
