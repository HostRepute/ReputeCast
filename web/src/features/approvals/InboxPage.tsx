import { keepPreviousData } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Td, Th } from '@/components/ui/Table'
import { FilterTabs } from '@/components/ui/Tabs'
import { ago, clock, count, monthDay, orList } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { waitShort } from '@/features/review/logic'
import { ApprovalStatusBadge, decidedText } from './parts'

export type InboxStatus = 'pending' | 'approved' | 'changes_requested' | 'rejected' | 'invalidated' | 'all'

export interface InboxSearch {
  status?: InboxStatus
  author?: string
  approver?: string
  days?: number
}

type Item = Schemas['ApprovalListItem']

/**
 * A5 §5.12 Approval inbox. It opens on every request of the period, as the
 * ApprovalInbox board lists them (pending first: they're the newest).
 */
export function InboxPage({ search }: { search: InboxSearch }) {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const status = search.status ?? 'all'
  const days = search.days ?? 30
  const list = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/approvals',
    { params: { path: { workspace: slug }, query: { status, author: search.author, approver: search.approver, days } } },
    { placeholderData: keepPreviousData },
  )
  const set = (patch: Partial<InboxSearch>) =>
    void navigate({ to: '/w/$workspace/approvals', params: { workspace: slug }, search: { ...search, ...patch }, replace: true })
  const counts = list.data?.counts
  const n = (v?: number) => (v !== undefined ? ` · ${count(v)}` : '')
  return (
    <Page
      title="Approvals"
      filters={
        <FilterTabs
          label="Approval status"
          className="ml-3 max-md:ml-0"
          value={status}
          onChange={(next) => set({ status: next === 'all' ? undefined : next })}
          items={[
            { value: 'pending', label: `Pending${n(counts?.pending)}` },
            { value: 'approved', label: `Approved${n(counts?.approved)}` },
            { value: 'changes_requested', label: `Changes requested${n(counts?.changesRequested)}` },
            { value: 'rejected', label: `Rejected${n(counts?.rejected)}` },
            { value: 'invalidated', label: `Invalidated${n(counts?.invalidated)}` },
            { value: 'all', label: 'All' },
          ]}
        />
      }
    >
      {list.isError ? (
        <ErrorPanel title="Couldn't load the approval requests" actions={<Button onClick={() => list.refetch()}>Retry</Button>}>
          {errorMessage(list.error)} Nothing was changed.
        </ErrorPanel>
      ) : list.isPending ? (
        <InboxSkeleton />
      ) : (
        <>
          <RequestsCard data={list.data} status={status} search={search} days={days} onFilter={set} />
          <div className="grid gap-5 [grid-template-columns:repeat(auto-fill,minmax(min(380px,100%),1fr))]">
            <PolicyCard policy={list.data.policy} canManage={list.data.canManage} />
          </div>
        </>
      )}
    </Page>
  )
}

function RequestsCard({
  data,
  status,
  search,
  days,
  onFilter,
}: {
  data: Schemas['ApprovalList']
  status: InboxStatus
  search: InboxSearch
  days: number
  onFilter: (p: Partial<InboxSearch>) => void
}) {
  const now = useNow()
  const filtered = Boolean(search.author || search.approver) || days !== 30
  return (
    <section className="card" aria-labelledby="ap-requests">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-requests">
            Approval requests
          </h2>
          <span className="k">
            {data.allRequests ? 'Everything that needed a second person · approvers see all requests, authors see their own' : 'Your requests · approvers see every request'}
          </span>
        </div>
        <div className="row gap-2!">
          {data.allRequests ? (
            <select className="in w-[150px] max-md:min-w-full" aria-label="Filter by author" value={search.author ?? ''} onChange={(e) => onFilter({ author: e.target.value || undefined })}>
              <option value="">Any author</option>
              {data.authors.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          ) : null}
          <select className="in w-[150px] max-md:min-w-full" aria-label="Filter by approver" value={search.approver ?? ''} onChange={(e) => onFilter({ approver: e.target.value || undefined })}>
            <option value="">Any approver</option>
            {data.approvers.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
          <select className="in w-[140px] max-md:min-w-full" aria-label="Date range" value={days} onChange={(e) => onFilter({ days: Number(e.target.value) === 30 ? undefined : Number(e.target.value) })}>
            <option value={30}>Last 30 days</option>
            <option value={7}>Last 7 days</option>
            <option value={90}>Last 90 days</option>
            <option value={0}>All time</option>
          </select>
        </div>
      </div>
      {data.items.length === 0 ? (
        <div className="pad">
          <div className="note n flex-col items-center justify-center gap-1.5! p-[22px]! text-center" role="status">
            {status === 'pending' && !filtered ? (
              <>
                <span className="font-semibold">Nothing waiting for you</span>
                <span className="k">Requests appear here and in your notifications. Authors can still see their own requests.</span>
              </>
            ) : (
              <>
                <span className="font-semibold">No {status === 'all' ? '' : `${statusWords[status]} `}requests {days ? `in the last ${days} days` : 'yet'}</span>
                <span className="k">{filtered ? 'Clear the filters or pick a longer date range to see more.' : 'Requests appear here once someone asks for an approval.'}</span>
              </>
            )}
          </div>
        </div>
      ) : (
        <AccessTable caption="Approval requests">
          <thead>
            <tr>
              <Th>Campaign</Th>
              <Th>Author</Th>
              <Th>Audience</Th>
              <Th>Route</Th>
              <Th>Scheduled</Th>
              <Th>Requested</Th>
              <Th>Status</Th>
              <Th>
                <span className="vh">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {data.items.map((it) => (
              <tr key={it.id} className={it.status === 'pending' ? 'bg-warn-tint' : undefined}>
                <Td>
                  <div className="col gap-0.5!">
                    <span className="font-semibold">{it.name}</span>
                    <span className="k">{it.rules}</span>
                  </div>
                </Td>
                <Td>
                  <PersonCell name={it.author.name} sub={it.author.email} color={it.author.color} />
                </Td>
                <Td>{it.audienceNow != null && it.audienceNow !== it.audience ? `${count(it.audience)} → ${count(it.audienceNow)}` : count(it.audience)}</Td>
                <Td>{it.route}</Td>
                <Td>{scheduledText(it)}</Td>
                <Td>{requestedText(it.requestedAt, now)}</Td>
                <Td>
                  {it.status === 'pending' ? (
                    <ApprovalStatusBadge status={it.status} />
                  ) : (
                    <span className="col gap-0.5!">
                      <ApprovalStatusBadge status={it.status} />
                      <span className="k">{statusNote(it, now)}</span>
                    </span>
                  )}
                </Td>
                <Td>
                  <RowAction it={it} canDecide={data.allRequests} />
                </Td>
              </tr>
            ))}
          </tbody>
        </AccessTable>
      )}
    </section>
  )
}

const statusWords: Record<Exclude<InboxStatus, 'all'>, string> = {
  pending: 'pending',
  approved: 'approved',
  changes_requested: 'changes-requested',
  rejected: 'rejected',
  invalidated: 'invalidated',
}

/** "Nov 4, 09:00", "Sent Oct 9" or "Unscheduled". */
function scheduledText(it: Item): string {
  if (it.sentAt) return `Sent ${monthDay(it.sentAt)}`
  if (it.scheduleMode !== 'now' && it.scheduleAt) return `${monthDay(it.scheduleAt)}, ${clock(it.scheduleAt)}`
  return 'Unscheduled'
}

/** "2 h ago" today, "Oct 10" before. */
function requestedText(d: string, now: Date): string {
  return now.getTime() - new Date(d).getTime() < 86_400_000 ? ago(d, now) : monthDay(d)
}

/** The line under a decided request's status. */
function statusNote(it: Item, now: Date): string {
  if (it.note) return it.note
  if (it.status === 'approved' || it.status === 'rejected') return it.decidedBy ? `${it.decidedBy}${it.decidedAt ? ` · ${decidedText(it.decidedAt, now)}` : ''}` : ''
  if (it.status === 'cancelled') return 'withdrawn by the author'
  if (it.status === 'bypassed') return 'sent without approval by an Owner'
  return ''
}

function RowAction({ it, canDecide }: { it: Item; canDecide: boolean }) {
  const { slug } = useWorkspace()
  if (it.sentAt && it.campaignId) {
    return (
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: it.campaignId }}>
        Report
      </ButtonLink>
    )
  }
  const params = { workspace: slug, approvalId: it.id }
  if (it.status === 'pending' && canDecide) {
    return (
      <ButtonLink size="sm" variant="primary" to="/w/$workspace/approvals/$approvalId" params={params}>
        Review
      </ButtonLink>
    )
  }
  return (
    <ButtonLink size="sm" to="/w/$workspace/approvals/$approvalId" params={params}>
      {it.status === 'invalidated' ? 'View diff' : 'View'}
    </ButtonLink>
  )
}

function PolicyCard({ policy, canManage }: { policy: Schemas['ApprovalPolicySummary']; canManage: boolean }) {
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="ap-policy">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-policy">
            Policy in force
          </h2>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/settings/approvals" params={{ workspace: slug }}>
          {canManage ? 'Edit policy' : 'View policy'}
        </ButtonLink>
      </div>
      <div className="pad">
        <dl className="kv">
          <dt>Rules</dt>
          <dd>{policy.rules.length ? policy.rules.join(' · ') : 'No rule is on: sends never need approval'}</dd>
          <dt>Approvals</dt>
          <dd>
            {policy.approvalsRequired} from {policy.approvers.length ? orList(policy.approvers) : 'nobody yet (no member holds campaigns:approve)'}
          </dd>
          <dt>Median wait</dt>
          <dd>{policy.medianWaitMinutes != null ? `${waitShort(policy.medianWaitMinutes)} over the last 30 days` : 'No decisions in the last 30 days'}</dd>
        </dl>
      </div>
    </section>
  )
}

function InboxSkeleton() {
  return (
    <Loading label="Loading approval requests" className="card">
      <div className="sec">
        <div className="flex flex-col gap-2">
          <Skeleton w={160} h={14} />
          <Skeleton w={320} h={10} />
        </div>
        <div className="flex gap-2">
          <Skeleton w={150} h={36} r={8} />
          <Skeleton w={150} h={36} r={8} />
          <Skeleton w={140} h={36} r={8} />
        </div>
      </div>
      {Array.from({ length: 5 }, (_, i) => (
        <div key={i} className="flex items-center gap-6 border-b border-line-2 px-3 py-3.5 last:border-b-0">
          <div className="flex min-w-[180px] flex-col gap-2">
            <Skeleton w={170} h={12} />
            <Skeleton w={220} h={10} />
          </div>
          <Skeleton w={28} h={28} r={14} />
          <Skeleton w={90} h={12} />
          <Skeleton w={60} h={12} />
          <Skeleton w={140} h={12} />
          <Skeleton w={80} h={12} />
          <Skeleton w={70} h={22} r={999} />
          <Skeleton w={60} h={30} r={8} className="ml-auto" />
        </div>
      ))}
    </Loading>
  )
}
