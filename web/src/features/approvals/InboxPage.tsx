import { keepPreviousData } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Td, Th } from '@/components/ui/Table'
import { FilterTabs } from '@/components/ui/Tabs'
import { t as tt, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { ago, count, monthDay, monthDayTime, orList } from '@/lib/format'
import { useNow } from '@/lib/hooks'
import { useWorkspace } from '@/lib/session'
import { AccessTable, PersonCell } from '@/features/access/parts'
import { waitShort } from '@/features/review/logic'
import { RecurringDigestBadge } from '@/features/digests/IssueParts'
import { useIssueInfos } from '@/features/digests/shared'
import { PageBadge, PageLine, pageScheduledText } from '@/features/pages/approval'
import { m } from './messages'
import { ApprovalStatusBadge, decidedText } from './parts'
import { PostBadge, postLine, postRoute, postWhen } from '@/features/website/elsewhere'
import { ChannelTag } from '@/features/channels/ChannelFilter'

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
  const t = useT()
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
  const label = (s: InboxStatus, v?: number) => (v !== undefined ? t(m.filterCount, { status: s, count: count(v) }) : t(m.filter, { status: s }))
  return (
    <Page
      title={t(m.title)}
      filters={
        <FilterTabs
          label={t(m.filterLabel)}
          className="ml-3 max-md:ml-0"
          value={status}
          onChange={(next) => set({ status: next === 'all' ? undefined : next })}
          items={[
            { value: 'pending', label: label('pending', counts?.pending) },
            { value: 'approved', label: label('approved', counts?.approved) },
            { value: 'changes_requested', label: label('changes_requested', counts?.changesRequested) },
            { value: 'rejected', label: label('rejected', counts?.rejected) },
            { value: 'invalidated', label: label('invalidated', counts?.invalidated) },
            { value: 'all', label: label('all') },
          ]}
        />
      }
    >
      {list.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => list.refetch()}>{t(common.retry)}</Button>}>
          {errorMessage(list.error)} {t(m.nothingChanged)}
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
  const t = useT()
  const now = useNow()
  const filtered = Boolean(search.author || search.approver) || days !== 30
  // Requests for a recurring digest's issues carry its badge.
  const issues = useIssueInfos(data.items.flatMap((it) => (it.campaignId ? [it.campaignId] : []))).data?.items ?? []
  const issueOf = new Map(issues.map((x) => [x.campaignId, x]))
  return (
    <section className="card" aria-labelledby="ap-requests">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-requests">
            {t(m.requestsTitle)}
          </h2>
          <span className="k">{data.allRequests ? t(m.requestsCaptionAll) : t(m.requestsCaptionOwn)}</span>
        </div>
        <div className="ap-filters row gap-2!">
          {data.allRequests ? (
            <select className="in w-[150px] max-md:min-w-full" aria-label={t(m.filterByAuthor)} value={search.author ?? ''} onChange={(e) => onFilter({ author: e.target.value || undefined })}>
              <option value="">{t(m.anyAuthor)}</option>
              {data.authors.map((a) => (
                <option key={a.id} value={a.id}>
                  {a.name}
                </option>
              ))}
            </select>
          ) : null}
          <select className="in w-[150px] max-md:min-w-full" aria-label={t(m.filterByApprover)} value={search.approver ?? ''} onChange={(e) => onFilter({ approver: e.target.value || undefined })}>
            <option value="">{t(m.anyApprover)}</option>
            {data.approvers.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
          </select>
          <select className="in w-[140px] max-md:min-w-full" aria-label={t(m.dateRange)} value={days} onChange={(e) => onFilter({ days: Number(e.target.value) === 30 ? undefined : Number(e.target.value) })}>
            <option value={30}>{t(m.lastDays, { days: count(30) })}</option>
            <option value={7}>{t(m.lastDays, { days: count(7) })}</option>
            <option value={90}>{t(m.lastDays, { days: count(90) })}</option>
            <option value={0}>{t(m.allTime)}</option>
          </select>
        </div>
      </div>
      {data.items.length === 0 ? (
        <div className="pad">
          <div className="note n flex-col items-center justify-center gap-1.5! p-[22px]! text-center" role="status">
            {status === 'pending' && !filtered ? (
              <>
                <span className="font-semibold">{t(m.emptyPendingTitle)}</span>
                <span className="k">{t(m.emptyPendingBody)}</span>
              </>
            ) : (
              <>
                <span className="font-semibold">{days ? t(m.emptyTitleDays, { status, days: count(days) }) : t(m.emptyTitleYet, { status })}</span>
                <span className="k">{filtered ? t(m.emptyFiltered) : t(m.emptyNone)}</span>
              </>
            )}
          </div>
        </div>
      ) : (
        <AccessTable caption={t(m.tableCaption)}>
          <thead>
            <tr>
              <Th>{t(m.colCampaign)}</Th>
              <Th>{t(m.colAuthor)}</Th>
              <Th>{t(m.colAudience)}</Th>
              <Th>{t(m.colRoute)}</Th>
              <Th>{t(m.colScheduled)}</Th>
              <Th>{t(m.colRequested)}</Th>
              <Th>{t(m.colStatus)}</Th>
              <Th>
                <span className="vh">{t(m.colActions)}</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {data.items.map((it) => (
              <tr key={it.id} className={it.status === 'pending' ? 'bg-warn-tint' : undefined}>
                <Td>
                  <div className="col gap-0.5!">
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="font-semibold">{it.name}</span>
                      {it.channel !== 'email' ? <ChannelTag channel={it.channel} /> : null}
                      {it.campaignId && issueOf.has(it.campaignId) ? <RecurringDigestBadge issue={issueOf.get(it.campaignId)!} /> : null}
                      {it.subjectType === 'website_post' ? <PostBadge /> : null}
                      {it.page ? <PageBadge page={it.page} /> : null}
                    </span>
                    <span className="k">{it.subjectType === 'website_post' && it.website ? postLine(it.website, it.rules) : it.rules}</span>
                    {it.page ? <PageLine page={it.page} /> : null}
                  </div>
                </Td>
                <Td>
                  <PersonCell name={it.author.name} sub={it.author.email} color={it.author.color} />
                </Td>
                <Td>{it.subjectType === 'website_post' || it.page ? '–' : it.audienceNow != null && it.audienceNow !== it.audience ? `${count(it.audience)} → ${count(it.audienceNow)}` : count(it.audience)}</Td>
                <Td>{it.subjectType === 'website_post' || it.page ? postRoute() : it.route}</Td>
                <Td>{it.subjectType === 'website_post' && it.website ? postWhen(it.website) : scheduledText(it)}</Td>
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

/** "Nov 4, 09:00", "Sent Oct 9" or "Unscheduled". */
function scheduledText(it: Item): string {
  if (it.page) return pageScheduledText(it.page)
  if (it.sentAt) return tt(m.scheduledSent, { date: monthDay(it.sentAt) })
  if (it.scheduleMode !== 'now' && it.scheduleAt) return monthDayTime(it.scheduleAt)
  return tt(m.unscheduled)
}

/** "2 h ago" today, "Oct 10" before. */
function requestedText(d: string, now: Date): string {
  return now.getTime() - new Date(d).getTime() < 86_400_000 ? ago(d, now) : monthDay(d)
}

/** The line under a decided request's status. */
function statusNote(it: Item, now: Date): string {
  if (it.note) return it.note
  if (it.status === 'approved' || it.status === 'rejected') return it.decidedBy ? `${it.decidedBy}${it.decidedAt ? ` · ${decidedText(it.decidedAt, now)}` : ''}` : ''
  if (it.status === 'cancelled') return tt(m.noteWithdrawn)
  if (it.status === 'bypassed') return tt(m.noteBypassed)
  return ''
}

function RowAction({ it, canDecide }: { it: Item; canDecide: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (it.sentAt && it.campaignId) {
    return (
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: it.campaignId }}>
        {t(m.report)}
      </ButtonLink>
    )
  }
  const params = { workspace: slug, approvalId: it.id }
  if (it.status === 'pending' && canDecide) {
    return (
      <ButtonLink size="sm" variant="primary" to="/w/$workspace/approvals/$approvalId" params={params}>
        {t(m.review)}
      </ButtonLink>
    )
  }
  return (
    <ButtonLink size="sm" to="/w/$workspace/approvals/$approvalId" params={params}>
      {it.status === 'invalidated' ? t(m.viewDiff) : t(m.view)}
    </ButtonLink>
  )
}

function PolicyCard({ policy, canManage }: { policy: Schemas['ApprovalPolicySummary']; canManage: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <section className="card" aria-labelledby="ap-policy">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-policy">
            {t(m.policyTitle)}
          </h2>
        </div>
        <ButtonLink size="sm" to="/w/$workspace/settings/approvals" params={{ workspace: slug }}>
          {canManage ? t(m.editPolicy) : t(m.viewPolicy)}
        </ButtonLink>
      </div>
      <div className="pad">
        <dl className="kv">
          <dt>{t(m.policyRules)}</dt>
          <dd>{policy.rules.length ? policy.rules.join(' · ') : t(m.noRules)}</dd>
          <dt>{t(m.policyApprovals)}</dt>
          <dd>
            {policy.approvers.length
              ? t(m.approvalsFrom, { count: policy.approvalsRequired, approvers: orList(policy.approvers) })
              : t(m.approvalsFromNobody, { count: policy.approvalsRequired })}
          </dd>
          <dt>{t(m.policyMedianWait)}</dt>
          <dd>{policy.medianWaitMinutes != null ? t(m.medianWait, { wait: waitShort(policy.medianWaitMinutes) }) : t(m.medianWaitNone)}</dd>
        </dl>
      </div>
    </section>
  )
}

function InboxSkeleton() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="card">
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
