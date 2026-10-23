import { useRef } from 'react'
import { $api, errorMessage } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Button } from '@/components/ui/Button'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, monthName, orList, pct } from '@/lib/format'
import { useMe, usePermissionReason, useWorkspace } from '@/lib/session'
import { ChannelSwitch } from '@/features/channels/ChannelSwitch'
import { PushTabs, throwIfForbidden, type PushSetup } from './common'
import { m } from './messages'
import { NotSetUp } from './NotSetUp'
import { GuardsCard, InfoIcon, KeysCard, PromptsCard, SitesCard, SubscribersCard, WorkerAlerts } from './SetupCards'

/** "15 Oct, 08:00". */
export function useWhen() {
  const t = useT()
  return (d: string) => t(m.when, { date: dayMonthDate(d), time: clock(d) })
}

/**
 * Sending providers › Push › Setup (brief §5.9, boards PushSetup and
 * PushStates): the keys, the guards, the sites with their service worker,
 * the subscribers and the prompts.
 */
export function PushSetupPage() {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/setup', { params: { path: { workspace: slug } } })
  throwIfForbidden(q.error)
  const addRef = useRef<HTMLInputElement>(null)
  const data = q.data
  return (
    <Page
      title={t(m.screenTitle)}
      documentTitle={t(m.docTitle)}
      filters={
        <>
          <ChannelSwitch current="push" />
          <PushTabs />
        </>
      }
      actions={data?.on ? <HeaderActions data={data} onAdd={() => addRef.current?.focus()} /> : null}
    >
      {q.isError ? (
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)} {t(m.loadFailedText)}
        </ErrorPanel>
      ) : !data ? (
        <SetupSkeleton />
      ) : !data.on ? (
        <NotSetUp data={data} />
      ) : (
        <>
          {data.canManage ? null : <ReadOnlyNote helpers={data.helpers} />}
          <WorkerAlerts data={data} />
          <Tiles data={data} />
          <div className="px-grid">
            <KeysCard data={data} />
            <GuardsCard data={data} />
          </div>
          <SitesCard data={data} addRef={addRef} />
          <div className="px-grid">
            <SubscribersCard data={data} />
            <PromptsCard data={data} />
          </div>
        </>
      )}
    </Page>
  )
}

function HeaderActions({ data, onAdd }: { data: PushSetup; onAdd: () => void }) {
  const t = useT()
  const when = useWhen()
  const reason = usePermissionReason('providers:manage', data.helpers)
  return (
    <>
      {data.checkedAt && data.sites.length ? <span className="k max-md:hidden">{t(m.checkedAll, { count: data.sites.length, date: when(data.checkedAt) })}</span> : null}
      <Button variant="primary" onClick={onAdd} disabledReason={reason}>
        {t(m.addSite)}
      </Button>
    </>
  )
}

/** "Read-only · Aisha Bello, Viewer": what the role can see, and who can change it. */
function ReadOnlyNote({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me } = useWorkspace()
  const user = useMe().data?.user
  return (
    <div className="note n" role="note">
      <InfoIcon />
      <div className="flex min-w-0 flex-col gap-1">
        <span className="font-bold">{t(m.readOnlyTitle, { name: user?.name ?? '', role: me.workspace.roleName })}</span>
        <div>
          <T msg={m.readOnlyText} values={{ perm: <span className="mono">{'providers:manage'}</span> }} />
          {helpers.length ? ` ${t(m.askHelpers, { helpers: orList(helpers) })}` : null}
        </div>
      </div>
    </div>
  )
}

function Tiles({ data }: { data: PushSetup }) {
  const t = useT()
  const s = data.subscribers
  const month = monthName(data.month.month, 'long')
  return (
    <div className="px-stats">
      <div className="card px-stat">
        <span className="k">{t(m.tileSubscribers)}</span>
        <span className="v">{count(s.total)}</span>
        <span className="s">{s.bySite.map((b) => t(m.siteCount, { host: b.key, count: count(b.count) })).join(' · ')}</span>
      </div>
      <div className="card px-stat">
        <span className="k">{t(m.tileLinked)}</span>
        <span className="v">{count(s.linked)}</span>
        <span className="s">{t(m.tileLinkedSub, { pct: pct(s.total ? (s.linked / s.total) * 100 : 0) })}</span>
      </div>
      <div className="card px-stat">
        <span className="k">{t(m.tileSentIn, { month })}</span>
        <span className="v">{count(data.month.sent)}</span>
        <span className="s">
          {data.month.sent ? t(m.tileSentSub, { campaigns: data.month.campaigns, pct: pct(data.month.shownPct) }) : t(m.tileSentNone)}
        </span>
      </div>
      <div className="card px-stat">
        <span className="k">{t(m.tileCap)}</span>
        <span className="v">{t(m.tileCapValue, { n: data.cap })}</span>
        <span className="s">{data.quiet ? t(m.tileCapQuiet) : t(m.tileCapNoQuiet)}</span>
      </div>
    </div>
  )
}

function SetupSkeleton() {
  const t = useT()
  return (
    <Loading label={t(m.loading)} className="flex flex-col gap-4">
      <div className="px-stats">
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="card px-stat">
            <Skeleton w={90} h={11} />
            <Skeleton w={70} h={22} className="mt-1" />
            <Skeleton w={150} h={11} className="mt-1" />
          </div>
        ))}
      </div>
      <div className="px-grid">
        {[0, 1].map((i) => (
          <div key={i} className="card px-pad flex flex-col gap-3">
            <Skeleton w={140} h={14} />
            <Skeleton w="80%" h={11} />
            <Skeleton w="100%" h={60} />
          </div>
        ))}
      </div>
    </Loading>
  )
}
