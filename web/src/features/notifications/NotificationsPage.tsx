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
import { notificationTime } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { NotificationLink, notificationDotColor, useNotificationActions } from '@/components/shell/NotificationsPopover'

export type NotificationFilter = 'all' | 'unread' | 'approvals' | 'sending' | 'providers' | 'team' | 'hostrepute'

const tabs: { value: NotificationFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'unread', label: 'Unread' },
  { value: 'approvals', label: 'Approvals' },
  { value: 'sending', label: 'Sending' },
  { value: 'providers', label: 'Providers' },
  { value: 'team', label: 'Team' },
  { value: 'hostrepute', label: 'HostRepute' },
]

/** A5 §5.16: the notification center with category tabs and the preferences. */
export function NotificationsPage({ filter }: { filter: NotificationFilter }) {
  const { slug, me } = useWorkspace()
  const navigate = useNavigate()
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/notifications', { params: { path: { workspace: slug }, query: { filter, limit: 100 } } })
  const { markAll, marking } = useNotificationActions()
  const unread = list.data?.unread ?? me.unread
  return (
    <Page
      title="Notifications"
      filters={
        <FilterTabs
          className="ml-3"
          label="Filter"
          value={filter}
          onChange={(v) => navigate({ to: '.', search: v === 'all' ? {} : { filter: v }, replace: true })}
          items={tabs.map((t) => ({ value: t.value, label: t.value === 'unread' && unread ? `Unread · ${unread}` : t.label }))}
        />
      }
      actions={
        <ButtonLink to="/w/$workspace/account" params={{ workspace: slug }}>
          Account &amp; security
        </ButtonLink>
      }
    >
      <div className="grid grid-cols-[minmax(0,3fr)_minmax(360px,2fr)] items-start gap-5 max-md:grid-cols-1 [&>*]:min-w-0">
        <section className="card" aria-labelledby="h-notifications">
          <CardSection
            id="h-notifications"
            title="Notifications"
            caption="Each one links to the screen where you can act"
            actions={
              <Button size="sm" loading={marking} disabledReason={unread ? undefined : 'Everything is read'} onClick={markAll}>
                Mark all as read
              </Button>
            }
          />
          {list.isError ? (
            <div className="pad">
              <ErrorPanel title="Couldn't load notifications" actions={<Button onClick={() => list.refetch()}>Retry</Button>}>
                {errorMessage(list.error)} Nothing was marked as read.
              </ErrorPanel>
            </div>
          ) : list.isPending ? (
            <ListSkeleton />
          ) : list.data.items.length === 0 ? (
            <div className="pad">
              <div className="note n flex-col items-center justify-center gap-1.5 p-[22px] text-center">
                <span className="font-semibold">{filter === 'all' ? "You're all caught up" : filter === 'unread' ? 'Nothing unread' : `No ${tabs.find((t) => t.value === filter)?.label} notifications`}</span>
                <span className="k">Approvals, auto-pauses, provider incidents, team changes and HostRepute alerts show up here.</span>
              </div>
            </div>
          ) : (
            <ul className="m-0 list-none p-0" aria-label="Notifications">
              {list.data.items.map((n) => (
                <li key={n.id}>
                  <NotificationLink n={n} srUnread={false} className="flex items-start gap-3 border-b border-line-2 px-[18px] py-3 text-ink no-underline hover:text-ink" unreadClassName="bg-ground-2">
                    <span className="dot mt-[7px] h-2.5! w-2.5!" style={{ color: notificationDotColor(n.severity) }} aria-hidden="true" />
                    <span className="col min-w-0 flex-1" style={{ gap: 2 }}>
                      <span className="row justify-between" style={{ gap: 8 }}>
                        <span className="row" style={{ gap: 8 }}>
                          <b className="text-[13.5px]">{n.title}</b>
                          <Badge tone="neutral">{n.categoryLabel}</Badge>
                        </span>
                        <span className="k whitespace-nowrap">
                          <time dateTime={n.createdAt}>{notificationTime(n.createdAt)}</time>
                        </span>
                      </span>
                      <span className="text-[13px] text-ink-nav">{n.body}</span>
                    </span>
                    {n.read ? null : <Badge tone="info">New</Badge>}
                  </NotificationLink>
                </li>
              ))}
            </ul>
          )}
          <div className="pad px-[18px] py-3">
            <span className="k">Older notifications are kept for {list.data?.retentionDays ?? 90} days.</span>
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
  return (
    <div aria-busy="true">
      <span className="vh" role="status">
        Loading notifications…
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
        <CardSection id="h-prefs" title="My preferences" />
        <div className="pad">
          <ErrorPanel title="Couldn't load your preferences" actions={<Button onClick={() => prefs.refetch()}>Retry</Button>}>
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
      toast({ kind: 'success', title: 'Preferences saved' })
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }
  const caption = p.email.provider
    ? `Per event type · emails go out on the System stream via ${p.email.provider}`
    : 'Per event type · email starts once a sending provider handles the System stream'
  return (
    <section className="card scroll-mt-4" id="preferences" aria-labelledby="h-prefs">
      <CardSection id="h-prefs" title="My preferences" caption={caption} />
      <form onSubmit={save}>
        <Table density="compact" className="[&_td]:whitespace-normal" caption="Notification preferences">
          <thead>
            <tr>
              <Th>Event</Th>
              <Th align="center">In-app</Th>
              <Th align="center">Email</Th>
            </tr>
          </thead>
          <tbody>
            {p.events.map((e) => (
              <tr key={e.type}>
                <Td>
                  <div className="col" style={{ gap: 1 }}>
                    <span className="font-semibold">{e.label}</span>
                    {e.audience ? <span className="k">{e.audience}</span> : null}
                  </div>
                </Td>
                <Td align="center">
                  <input type="checkbox" className="tg" role="switch" checked={e.inApp} onChange={(ev) => set(e.type, 'inApp', ev.target.checked)} aria-label={`In-app for ${e.label}`} />
                </Td>
                <Td align="center">
                  <input type="checkbox" className="tg" role="switch" checked={e.email} onChange={(ev) => set(e.type, 'email', ev.target.checked)} aria-label={`Email for ${e.label}`} />
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
              <span id={ids.quiet}>Quiet hours · email only</span>
              <div className="row flex-nowrap" style={{ gap: 8 }}>
                <select
                  id={ids.start}
                  className="in pr-2.5 max-md:min-w-0 max-md:flex-[1_1_120px]"
                  aria-label="Quiet hours start"
                  value={p.quietHours?.start ?? 'off'}
                  onChange={(ev) =>
                    setDraft({ ...p, quietHours: ev.target.value === 'off' ? undefined : { start: ev.target.value, end: p.quietHours?.end ?? '07:00' } })
                  }
                >
                  {[...new Set([...(p.quietHours ? [p.quietHours.start] : []), ...quietStarts])].sort().map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                  <option value="off">Off</option>
                </select>
                <span className="k">to</span>
                <select
                  id={ids.end}
                  className="in pr-2.5 max-md:min-w-0 max-md:flex-[1_1_120px]"
                  aria-label="Quiet hours end"
                  disabled={quietOff}
                  value={p.quietHours?.end ?? '07:00'}
                  onChange={(ev) => p.quietHours && setDraft({ ...p, quietHours: { start: p.quietHours.start, end: ev.target.value } })}
                >
                  {[...new Set([...(p.quietHours ? [p.quietHours.end] : []), ...quietEnds])].sort().map((t) => (
                    <option key={t} value={t}>
                      {t}
                    </option>
                  ))}
                </select>
              </div>
              <span className="k font-medium [contain:inline-size]">Auto-pauses and provider outages always send immediately.</span>
            </div>
            <div className="lbl">
              <label htmlFor={ids.digest}>Email digest</label>
              <select id={ids.digest} className="in pr-2.5" value={p.digest} onChange={(ev) => setDraft({ ...p, digest: ev.target.value as typeof p.digest })}>
                <option value="immediate">Immediately</option>
                <option value="hourly">Hourly digest</option>
                <option value="daily">Daily digest at 09:00</option>
              </select>
            </div>
          </div>
          {p.email.available ? null : (
            <Note tone="info" label="Email off">
              No sending provider handles the System stream yet, so notifications stay in-app. Your email choices apply once one does.
            </Note>
          )}
          {error ? (
            <Note tone="bad" label="Not saved">
              {error}
            </Note>
          ) : null}
          <div className="row justify-end">
            <Button type="submit" variant="primary" size="sm" loading={busy}>
              Save preferences
            </Button>
          </div>
        </div>
      </form>
    </section>
  )
}
