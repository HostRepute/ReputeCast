import { useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useEffect, useId, useState, type FormEvent } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { CardSection } from '@/components/ui/Card'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Note } from '@/components/ui/Note'
import { Table, Td, Th } from '@/components/ui/Table'
import { FilterTabs } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { count, notificationTime, timeOfDay } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { NotificationLink, notificationDotColor, useNotificationActions } from '@/components/shell/NotificationsPopover'
import { m } from './messages'

export type NotificationFilter = 'all' | 'unread' | 'approvals' | 'sending' | 'providers' | 'team' | 'hostrepute'

const tabs: NotificationFilter[] = ['all', 'unread', 'approvals', 'sending', 'providers', 'team', 'hostrepute']

/** The categories the server names; a category it adds later shows the server's own label. */
const categories = new Set(['approvals', 'sending', 'providers', 'team', 'hostrepute', 'other', 'integrations', 'transactional', 'ai', 'earn', 'website', 'grow'])

/** A5 §5.16: the notification center with category tabs and the preferences. */
export function NotificationsPage({ filter }: { filter: NotificationFilter }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/notifications', { params: { path: { workspace: slug }, query: { filter, limit: 100 } } })
  const { markAll, marking } = useNotificationActions()
  const unread = list.data?.unread ?? me.unread
  return (
    <Page
      title={t(m.title)}
      filters={
        <FilterTabs
          className="ml-3"
          label={t(m.filterLabel)}
          value={filter}
          onChange={(v) => navigate({ to: '.', search: v === 'all' ? {} : { filter: v }, replace: true })}
          items={tabs.map((v) => ({ value: v, label: v === 'unread' && unread ? t(m.unreadCount, { count: count(unread) }) : t(m.filter, { filter: v }) }))}
        />
      }
      actions={
        <ButtonLink to="/w/$workspace/account" params={{ workspace: slug }}>
          {t(m.account)}
        </ButtonLink>
      }
    >
      <div className="grid grid-cols-[minmax(0,3fr)_minmax(360px,2fr)] items-start gap-5 max-md:grid-cols-1 [&>*]:min-w-0">
        <section className="card" aria-labelledby="h-notifications">
          <CardSection
            id="h-notifications"
            title={t(m.listTitle)}
            caption={t(m.listCaption)}
            actions={
              <Button size="sm" loading={marking} disabledReason={unread ? undefined : t(m.allRead)} onClick={markAll}>
                {t(m.markAll)}
              </Button>
            }
          />
          {list.isError ? (
            <div className="pad">
              <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => list.refetch()}>{t(common.retry)}</Button>}>
                {errorMessage(list.error)} {t(m.nothingMarked)}
              </ErrorPanel>
            </div>
          ) : list.isPending ? (
            <ListSkeleton />
          ) : list.data.items.length === 0 ? (
            <div className="pad">
              <div className="note n flex-col items-center justify-center gap-1.5 p-[22px] text-center">
                <span className="font-semibold">
                  {filter === 'all' ? t(m.caughtUp) : filter === 'unread' ? t(m.nothingUnread) : t(m.noneInCategory, { filter })}
                </span>
                <span className="k">{t(m.emptyBody)}</span>
              </div>
            </div>
          ) : (
            <ul className="m-0 list-none p-0" aria-label={t(m.listTitle)}>
              {list.data.items.map((n) => (
                <li key={n.id}>
                  <NotificationLink n={n} srUnread={false} className="flex items-start gap-3 border-b border-line-2 px-[18px] py-3 text-ink no-underline hover:text-ink" unreadClassName="bg-ground-2">
                    <span className="dot mt-[7px] h-2.5! w-2.5!" style={{ color: notificationDotColor(n.severity) }} aria-hidden="true" />
                    <span className="col min-w-0 flex-1" style={{ gap: 2 }}>
                      <span className="row justify-between" style={{ gap: 8 }}>
                        <span className="row" style={{ gap: 8 }}>
                          {/* The title and body come from the server, in the member's language. */}
                          <b className="text-[13.5px]">{n.title}</b>
                          <Badge tone="neutral">{categories.has(n.category) ? t(m.category, { category: n.category }) : n.categoryLabel}</Badge>
                        </span>
                        <span className="k whitespace-nowrap">
                          <time dateTime={n.createdAt}>{notificationTime(n.createdAt)}</time>
                        </span>
                      </span>
                      <span className="text-[13px] text-ink-nav">{n.body}</span>
                    </span>
                    {n.read ? null : <Badge tone="info">{t(m.new)}</Badge>}
                  </NotificationLink>
                </li>
              ))}
            </ul>
          )}
          <div className="pad px-[18px] py-3">
            <span className="k">{t(m.retention, { days: list.data?.retentionDays ?? 90 })}</span>
          </div>
        </section>
        <div className="col" style={{ gap: 20 }}>
          <PreferencesCard />
        </div>
      </div>
    </Page>
  )
}

function ListSkeleton() {
  const t = useT()
  return (
    <div aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
      </span>
      {[0, 1, 2, 3].map((i) => (
        <div key={i} className="flex items-start gap-3 border-b border-line-2 px-[18px] py-3">
          <Skeleton w={10} h={10} r={5} className="mt-[7px]" />
          <div className="flex flex-1 flex-col gap-2">
            <Skeleton w="35%" h={14} />
            <Skeleton w="85%" h={12} />
          </div>
        </div>
      ))}
    </div>
  )
}

const quietStarts = ['21:00', '22:00', '23:00']
const quietEnds = ['06:00', '07:00', '08:00']

/** My preferences: in-app and email per event type, quiet hours, email digest. */
function PreferencesCard() {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const prefs = $api.useQuery('get', '/api/v1/me/notification-preferences', { params: { query: { workspace: slug } } })
  const [draft, setDraft] = useState<Schemas['NotificationPreferences']>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const ids = { start: useId(), end: useId(), digest: useId(), quiet: useId() }
  const p = draft ?? prefs.data

  // Arriving from "Edit preferences" (#preferences) scrolls the card into view.
  useEffect(() => {
    if (prefs.data && window.location.hash === '#preferences') document.getElementById('preferences')?.scrollIntoView({ block: 'start' })
  }, [prefs.data])

  if (prefs.isError) {
    return (
      <section className="card" id="preferences" aria-labelledby="h-prefs">
        <CardSection id="h-prefs" title={t(m.prefsTitle)} />
        <div className="pad">
          <ErrorPanel title={t(m.prefsLoadFailed)} actions={<Button onClick={() => prefs.refetch()}>{t(common.retry)}</Button>}>
            {errorMessage(prefs.error)}
          </ErrorPanel>
        </div>
      </section>
    )
  }
  if (!p) {
    return (
      <section className="card" id="preferences" aria-busy="true">
        <div className="sec">
          <div className="flex flex-col gap-1.5">
            <Skeleton w={120} h={15} />
            <Skeleton w={260} h={12} />
          </div>
        </div>
        <div className="pad col" style={{ gap: 14 }}>
          {[0, 1, 2, 3, 4].map((i) => (
            <Skeleton key={i} w="100%" h={24} />
          ))}
        </div>
      </section>
    )
  }

  const set = (type: string, field: 'inApp' | 'email', v: boolean) =>
    setDraft({ ...p, events: p.events.map((e) => (e.type === type ? { ...e, [field]: v } : e)) })
  const quietOff = !p.quietHours
  const save = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT('/api/v1/me/notification-preferences', {
        body: { events: p.events.map(({ type, inApp, email }) => ({ type, inApp, email })), quietHours: p.quietHours ?? undefined, digest: p.digest },
      })
      setDraft(undefined)
      await Promise.all([qc.invalidateQueries({ queryKey: ['get', '/api/v1/me/notification-preferences'] }), qc.invalidateQueries({ queryKey: ['get', '/api/v1/me/account'] })])
      toast({ kind: 'success', title: t(m.saved) })
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }
  const caption = p.email.provider ? t(m.prefsCaptionVia, { provider: p.email.provider }) : t(m.prefsCaption)
  return (
    <section className="card scroll-mt-4" id="preferences" aria-labelledby="h-prefs">
      <CardSection id="h-prefs" title={t(m.prefsTitle)} caption={caption} />
      <form onSubmit={save}>
        <Table density="compact" className="[&_td]:whitespace-normal" caption={t(m.prefsTable)}>
          <thead>
            <tr>
              <Th>{t(m.colEvent)}</Th>
              <Th align="center">{t(m.colInApp)}</Th>
              <Th align="center">{t(m.colEmail)}</Th>
            </tr>
          </thead>
          <tbody>
            {p.events.map((e) => (
              <tr key={e.type}>
                <Td>
                  {/* The event types' names and audiences come from the server. */}
                  <div className="col" style={{ gap: 1 }}>
                    <span className="font-semibold">{e.label}</span>
                    {e.audience ? <span className="k">{e.audience}</span> : null}
                  </div>
                </Td>
                <Td align="center">
                  <input
                    type="checkbox"
                    className="tg"
                    role="switch"
                    checked={e.inApp}
                    onChange={(ev) => set(e.type, 'inApp', ev.target.checked)}
                    aria-label={t(m.inAppFor, { event: e.label })}
                  />
                </Td>
                <Td align="center">
                  <input
                    type="checkbox"
                    className="tg"
                    role="switch"
                    checked={e.email}
                    onChange={(ev) => set(e.type, 'email', ev.target.checked)}
                    aria-label={t(m.emailFor, { event: e.label })}
                  />
                </Td>
              </tr>
            ))}
          </tbody>
        </Table>
        <div className="pad col border-t border-line-2" style={{ gap: 10 }}>
          {/* The board's 1fr 1fr, except that the quiet-hours times never get
              cut off when the card is narrow (834): that column keeps its
              content width and the digest select takes the rest. */}
          <div className="grid grid-cols-[minmax(max-content,1fr)_minmax(0,1fr)] gap-3.5 max-md:grid-cols-1">
            <div className="lbl">
              <span id={ids.quiet}>{t(m.quietHours)}</span>
              <div className="row flex-nowrap" style={{ gap: 8 }}>
                {/* "22:00 to 07:00": the words around the two selects come from one message. */}
                <T
                  msg={m.quietRange}
                  values={{
                    start: (
                      <select
                        id={ids.start}
                        className="in pr-2.5 max-md:min-w-0 max-md:flex-[1_1_120px]"
                        aria-label={t(m.quietStart)}
                        value={p.quietHours?.start ?? 'off'}
                        onChange={(ev) =>
                          setDraft({ ...p, quietHours: ev.target.value === 'off' ? undefined : { start: ev.target.value, end: p.quietHours?.end ?? '07:00' } })
                        }
                      >
                        {[...new Set([...(p.quietHours ? [p.quietHours.start] : []), ...quietStarts])].sort().map((h) => (
                          <option key={h} value={h}>
                            {timeOfDay(h)}
                          </option>
                        ))}
                        <option value="off">{t(common.off)}</option>
                      </select>
                    ),
                    k: (c) => <span className="k">{c}</span>,
                    end: (
                      <select
                        id={ids.end}
                        className="in pr-2.5 max-md:min-w-0 max-md:flex-[1_1_120px]"
                        aria-label={t(m.quietEnd)}
                        disabled={quietOff}
                        value={p.quietHours?.end ?? '07:00'}
                        onChange={(ev) => p.quietHours && setDraft({ ...p, quietHours: { start: p.quietHours.start, end: ev.target.value } })}
                      >
                        {[...new Set([...(p.quietHours ? [p.quietHours.end] : []), ...quietEnds])].sort().map((h) => (
                          <option key={h} value={h}>
                            {timeOfDay(h)}
                          </option>
                        ))}
                      </select>
                    ),
                  }}
                />
              </div>
              <span className="k font-medium [contain:inline-size]">{t(m.quietUrgent)}</span>
            </div>
            <div className="lbl">
              <label htmlFor={ids.digest}>{t(m.digest)}</label>
              <select id={ids.digest} className="in pr-2.5" value={p.digest} onChange={(ev) => setDraft({ ...p, digest: ev.target.value as typeof p.digest })}>
                <option value="immediate">{t(m.digestImmediate)}</option>
                <option value="hourly">{t(m.digestHourly)}</option>
                <option value="daily">{t(m.digestDaily, { time: '09:00' })}</option>
              </select>
            </div>
          </div>
          {p.email.available ? null : (
            <Note tone="info" label={t(m.emailOff)}>
              {t(m.emailOffBody)}
            </Note>
          )}
          {error ? (
            <Note tone="bad" label={t(m.notSaved)}>
              {error}
            </Note>
          ) : null}
          <div className="row justify-end">
            <Button type="submit" variant="primary" size="sm" loading={busy}>
              {t(m.save)}
            </Button>
          </div>
        </div>
      </form>
    </section>
  )
}
