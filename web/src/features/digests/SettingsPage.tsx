import { Link } from '@tanstack/react-router'
import { useMemo, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { LegendItem, StackBar } from '@/components/ui/Data'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, OptionCard, Select, Toggle } from '@/components/ui/Field'
import { Icon, type IconName } from '@/components/icons/Icon'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { clock, count, dayMonthDate, rate, shortDate, zonedParts } from '@/lib/format'
import { cn } from '@/lib/cn'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { m as cm } from '@/features/campaigns/messages'
import { PauseDialog, PreviewDialog, TurnOnDialog } from './Dialogs'
import { FeedItemRow } from './FeedItemRow'
import { FindButton, FoundBadge, FoundNote, SiteBadge, SiteStatusNote, useFeedFinder } from './FeedFinder'
import { m } from './messages'
import { s } from './settingsMessages'
import { ruleClauses } from './NextIssueCard'
import { DigestStateBadge, IssueStatusBadge, scheduleText, useDigest, useDigestFeed, useRefreshDigest, type Digest, type DigestSchedule } from './shared'

type Form = {
  name: string
  archive: boolean
  mode: 'auto' | 'draft'
  source: 'url' | 'wordpress'
  siteId: string
  feedUrl: string
  categories: string[]
  tag: string
  ageOn: boolean
  age: number
  startMode: 'after_on' | 'latest'
  startLatest: number
  schedule: DigestSchedule
  prepareHours: number
  lateHours: 6 | 24 | 48
  audience: string
  senderId: string
}

function formOf(d: Digest): Form {
  return {
    name: d.name,
    archive: d.archive,
    mode: d.mode,
    source: d.source,
    siteId: d.site?.id ?? '',
    feedUrl: d.feedUrl,
    categories: d.filters.categories,
    tag: d.filters.tags[0] ?? '',
    ageOn: d.filters.maxAgeDays != null,
    age: d.filters.maxAgeDays ?? 14,
    startMode: d.filters.startMode,
    startLatest: d.filters.startLatest || 3,
    schedule: d.schedule,
    prepareHours: d.prepareHours,
    lateHours: (d.lateHours === 6 || d.lateHours === 48 ? d.lateHours : 24) as 6 | 24 | 48,
    audience: d.audience.listIds[0] ? `l:${d.audience.listIds[0]}` : d.audience.segmentIds[0] ? `s:${d.audience.segmentIds[0]}` : '',
    senderId: d.sender?.id ?? '',
  }
}

const TIMEZONES = ['America/New_York', 'America/Chicago', 'America/Denver', 'America/Los_Angeles', 'Europe/London', 'Europe/Berlin', 'Asia/Ho_Chi_Minh', 'Asia/Tokyo', 'Australia/Sydney', 'UTC']
const PREPARE = [1, 2, 3, 6, 12, 24]

/** 5.2 Digest settings (DigestSetup): feed, filters, schedule, audience, sending mode, and the feed preview beside them. */
export function DigestSettingsPage({ digestId }: { digestId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = useDigest(digestId)
  const d = q.data
  const crumbs = [{ label: t(cm.title), to: '/w/$workspace/campaigns' as const, params: { workspace: slug } }]
  if (q.isError) {
    return (
      <Page title={t(m.loadFailed)} breadcrumb={[...crumbs, { label: t(m.loadFailed) }]}>
        <ErrorPanel title={t(m.loadFailed)} actions={<Button onClick={() => void q.refetch()}>{t(m.retry)}</Button>}>
          {errorMessage(q.error)}
        </ErrorPanel>
      </Page>
    )
  }
  if (!d) {
    return (
      <Page title={t(m.loading)} breadcrumb={[...crumbs, { label: '…' }]}>
        <div className="flex flex-col gap-4" aria-busy="true">
          {Array.from({ length: 3 }, (_, i) => (
            <section key={i} className="card pad flex flex-col gap-3">
              <Skeleton w={160} h={14} />
              <Skeleton w="100%" h={36} />
              <Skeleton w="70%" h={12} />
            </section>
          ))}
        </div>
      </Page>
    )
  }
  return <SettingsForm key={d.id + d.updatedAt} d={d} />
}

function SettingsForm({ d }: { d: Digest }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const canEdit = useCan('campaigns:edit')
  const editReason = usePermissionReason('campaigns:edit', d.editHelpers)
  const pauseReason = usePermissionReason('campaigns:pause', d.pauseHelpers)
  const ctx = $api.useQuery('get', '/api/v1/workspaces/{workspace}/composer', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const sites = $api.useQuery('get', '/api/v1/workspaces/{workspace}/digests/sites', { params: { path: { workspace: slug } } })
  const [f, setF] = useState<Form>(() => formOf(d))
  const [saving, setSaving] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [dialog, setDialog] = useState<'pause' | 'turn-on' | 'preview' | null>(null)
  const dirty = JSON.stringify(f) !== JSON.stringify(formOf(d))
  const set = <K extends keyof Form>(k: K, v: Form[K]) => setF((x) => ({ ...x, [k]: v }))
  const setSchedule = (patch: Partial<DigestSchedule>) => setF((x) => ({ ...x, schedule: { ...x.schedule, ...patch } }))

  const save = async () => {
    setSaving(true)
    setErrors({})
    const [kind, id] = f.audience.split(':')
    try {
      await http.PUT('/api/v1/workspaces/{workspace}/digests/{digestId}', {
        params: { path: { workspace: slug, digestId: d.id } },
        body: {
          name: f.name.trim(),
          archive: f.archive,
          mode: f.mode,
          source: f.source,
          siteId: f.source === 'wordpress' ? f.siteId : undefined,
          feedUrl: f.source === 'url' ? f.feedUrl : undefined,
          filters: { categories: f.categories, tags: f.tag ? [f.tag] : [], maxAgeDays: f.ageOn ? f.age : undefined, startMode: f.startMode, startLatest: f.startLatest },
          categoryFilter: f.categories.length > 0 || (d.categoryOptions.length > 0 && f.categories.length === 0 && d.filters.categories.length > 0),
          schedule: f.schedule,
          prepareHours: f.prepareHours,
          lateHours: f.lateHours,
          listIds: kind === 'l' && id ? [id] : [],
          segmentIds: kind === 's' && id ? [id] : [],
          senderId: f.senderId || undefined,
        },
      })
      refresh()
      toast({ kind: 'success', title: t(s.saved) })
    } catch (e) {
      if (e instanceof ApiError && e.problem.errors?.length) {
        const out: Record<string, string> = {}
        for (const err of e.problem.errors) out[(err.location ?? '').replace(/^body\./, '')] = err.message ?? ''
        setErrors(out)
      }
      toast({ kind: 'error', title: t(s.saveFailed), detail: errorMessage(e) })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Page
      title={t(s.title)}
      documentTitle={`${d.name} · ${t(s.title)}`}
      breadcrumb={[
        { label: t(cm.title), to: '/w/$workspace/campaigns', params: { workspace: slug } },
        { label: d.name, to: '/w/$workspace/campaigns/digests/$digestId', params: { workspace: slug, digestId: d.id } },
        { label: t(m.settings) },
      ]}
      badges={<DigestStateBadge state={d.state} mode={d.mode} />}
      actions={
        <>
          {d.templateId ? (
            <ButtonLink to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: d.templateId }} className="max-md:hidden">
              {t(s.designEmail)}
            </ButtonLink>
          ) : null}
          {d.state === 'on' ? (
            <Button icon="pause" disabledReason={pauseReason} onClick={() => setDialog('pause')}>
              {t(m.pause)}
            </Button>
          ) : (
            <Button icon="play" disabledReason={editReason} disabled={dirty} onClick={() => setDialog('turn-on')}>
              {t(m.turnOn)}
            </Button>
          )}
          <Button variant="primary" disabledReason={editReason} disabled={!dirty} loading={saving} busyLabel={t(s.saving)} onClick={() => void save()}>
            {t(s.save)}
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-[minmax(0,1fr)_340px] items-start gap-4 max-lg:grid-cols-1">
        <fieldset className="m-0 flex min-w-0 flex-col gap-4 border-0 p-0" disabled={!canEdit}>
          <DigestCard f={f} set={set} errors={errors} d={d} />
          <FeedCard f={f} set={set} errors={errors} d={d} sites={sites.data?.items ?? []} />
          <ScheduleCard f={f} set={set} setSchedule={setSchedule} errors={errors} />
          <AudienceCard f={f} set={set} errors={errors} d={d} ctx={ctx.data} />
          <ModeCard f={f} set={set} d={d} />
        </fieldset>
        <aside className="flex min-w-0 flex-col gap-4">
          <PreviewCard d={d} />
          <NextCard d={d} onPreview={() => setDialog('preview')} />
          <ChecksCard d={d} />
        </aside>
      </div>
      {dialog === 'pause' ? <PauseDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog === 'turn-on' ? <TurnOnDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
      {dialog === 'preview' ? <PreviewDialog digest={d} onOpenChange={() => setDialog(null)} /> : null}
    </Page>
  )
}

type CardProps = { f: Form; set: <K extends keyof Form>(k: K, v: Form[K]) => void; errors: Record<string, string>; d: Digest }

function Card({ title, sub, id, badge, children }: { title: string; sub?: string; id: string; badge?: ReactNode; children: ReactNode }) {
  return (
    <section className="card" aria-labelledby={id} id={id === 'set-feed' ? 'feed' : undefined}>
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id={id}>
            {title}
          </h2>
          {sub ? <span className="k">{sub}</span> : null}
        </div>
        {badge}
      </div>
      <div className="pad flex flex-col gap-4">{children}</div>
    </section>
  )
}

function DigestCard({ f, set, errors, d }: CardProps) {
  const t = useT()
  const { slug } = useWorkspace()
  const next = d.next?.number ?? d.sentIssues + 1
  return (
    <Card title={t(s.digest)} id="set-digest">
      <Field label={t(s.name)} hint={t(s.nameHint, { name: `${f.name || d.name} #${next}` })} error={errors.name}>
        <Input value={f.name} maxLength={200} onChange={(e) => set('name', e.target.value)} />
      </Field>
      <div className="flex items-start justify-between gap-4 border-t border-line-2 pt-4">
        <div className="flex flex-col gap-0.5">
          <span className="font-semibold">{t(s.archive)}</span>
          <span className="k">
            {t(s.archiveHint)}{' '}
            <Link to="/w/$workspace/forms/archive" params={{ workspace: slug }} className="lnk">
              {t(s.viewArchive)}
            </Link>
          </span>
        </div>
        <Toggle label={t(s.archive)} hideLabel checked={f.archive} onChange={(e) => set('archive', e.target.checked)} />
      </div>
    </Card>
  )
}

function FeedCard({ f, set, errors, d, sites }: CardProps & { sites: Schemas['DigestSite'][] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshDigest()
  const [reading, setReading] = useState(false)
  const site = sites.find((x) => x.id === f.siteId)
  const read = async () => {
    setReading(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/read', { params: { path: { workspace: slug, digestId: d.id } } })
      refresh()
      if (data?.ok) toast({ kind: 'success', title: data.notModified ? t(m.readNotModified) : t(m.readDone, { count: data.new }) })
      else toast({ kind: 'error', title: data?.error?.detail ?? t(m.feedBroken, { status: 'unreachable' }) })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setReading(false)
    }
  }
  const options = d.categoryOptions
  const draftCats = site?.draftCategories ?? []
  const toggleCat = (c: string) => set('categories', f.categories.includes(c) ? f.categories.filter((x) => x !== c) : [...f.categories, c])
  const fd = d.feed
  // "Find feed" reads the address as typed; what it finds can replace it.
  const finder = useFeedFinder()
  return (
    <Card title={t(s.feed)} sub={t(s.feedSub)} id="set-feed" badge={finder.found ? <FoundBadge found={finder.found} /> : null}>
      <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1" role="radiogroup" aria-label={t(s.feed)}>
        <OptionCard name="feed-source" value="wordpress" checked={f.source === 'wordpress'} onChange={() => set('source', 'wordpress')} disabled={!sites.length} title={t(s.sourceWp)} description={t(s.sourceWpHint)}>
          {f.source === 'wordpress' && sites.length ? (
            <div className="mt-2 flex flex-col gap-1.5">
              <Select aria-label={t(s.sourceWp)} value={f.siteId} onChange={(e) => set('siteId', e.target.value)}>
                {sites.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name ? `${x.host} · ${x.name}` : x.host}
                  </option>
                ))}
              </Select>
              {site ? (
                <span className="flex flex-wrap items-center gap-2">
                  <span className="k">{t(s.siteFeed, { feed: site.feedUrl.replace(/^https?:\/\//, ''), name: site.connectedBy, date: dayMonthDate(site.connectedAt) })}</span>
                  {site.status !== 'ok' ? <SiteBadge site={site} /> : null}
                </span>
              ) : null}
            </div>
          ) : null}
        </OptionCard>
        <OptionCard name="feed-source" value="url" checked={f.source === 'url'} onChange={() => set('source', 'url')} title={t(s.sourceUrl)} description={t(s.sourceUrlHint)}>
          {f.source === 'url' ? (
            <div className="mt-2">
              <Field label={t(m.feedOrPage)} error={errors.feedUrl}>
                <span className="flex flex-wrap gap-2">
                  <Input
                    type="url"
                    className="min-w-0 flex-[1_1_160px]"
                    value={f.feedUrl}
                    onChange={(e) => {
                      set('feedUrl', e.target.value)
                      finder.clear()
                    }}
                    placeholder={'https://blog.example.com/feed/'}
                  />
                  <FindButton busy={Boolean(finder.finding)} disabled={!f.feedUrl.trim()} onFind={() => void finder.find(f.feedUrl)} />
                </span>
              </Field>
            </div>
          ) : null}
        </OptionCard>
      </div>
      {f.source === 'url' ? (
        <FoundNote
          address={f.feedUrl}
          finding={finder.finding}
          found={finder.found}
          sites={sites}
          onUse={(url) => {
            set('feedUrl', url)
            finder.clear()
          }}
          onKeep={finder.clear}
          onRetry={() => void finder.find(f.feedUrl)}
          onUseSite={() => {
            set('source', 'wordpress')
            finder.clear()
          }}
        />
      ) : site ? (
        <SiteStatusNote site={site} />
      ) : null}
      <div className={cn('flex flex-wrap items-center gap-3 rounded-lg border px-3.5 py-2.5', fd.status === 'ok' ? 'border-line bg-ground' : 'border-bad-line bg-bad-tint')} role="status">
        {fd.status === 'ok' && fd.lastReadAt ? (
          <Badge tone="ok" dot>
            {t(s.lastReadBadge, { time: clock(fd.lastReadAt) })}
          </Badge>
        ) : (
          <Badge tone={fd.status === 'unread' ? 'neutral' : 'bad'} dot>
            {t(m.feedBroken, { status: fd.status })}
          </Badge>
        )}
        <span className="min-w-0 flex-1 text-[13px]">
          {fd.status === 'ok' ? t(s.feedLine, { kind: fd.kind || 'rss', count: fd.items, via: fd.lastReadVia || 'setup' }) : (fd.error?.detail ?? '')}
        </span>
        <Button size="sm" icon="refresh" loading={reading} onClick={() => void read()}>
          {t(m.readNow)}
        </Button>
      </div>
      {fd.error?.lines?.length ? <pre className="mono m-0 overflow-x-auto rounded-md border border-line bg-ground p-3 text-[12px]">{fd.error.lines.join('\n')}</pre> : null}
      <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
        <div className="flex flex-col gap-2">
          <span className="text-[13px] font-semibold">{t(m.categories)}</span>
          <div className="flex flex-wrap gap-2">
            {options.map((c) => (
              <label
                key={c}
                className={cn(
                  'inline-flex cursor-pointer items-center gap-2 rounded-full border px-3 py-1.5 text-[13px] font-medium',
                  f.categories.includes(c) ? 'border-accent-line bg-accent-tint' : 'border-line bg-surface',
                )}
              >
                <input type="checkbox" className="m-0" checked={f.categories.includes(c)} onChange={() => toggleCat(c)} />
                {c}
              </label>
            ))}
          </div>
          {errors['filters.categories'] ? (
            <span className="err" role="alert">
              {errors['filters.categories']}
            </span>
          ) : draftCats.length && site?.draftList ? (
            <span className="k">{t(s.draftCategories, { categories: formatList(draftCats), list: site.draftList })}</span>
          ) : null}
        </div>
        <Field label={t(s.tags)} hint={t(s.tagsHint)}>
          <Select value={f.tag} onChange={(e) => set('tag', e.target.value)}>
            <option value="">{t(s.anyTag)}</option>
            {d.tagOptions.map((x) => (
              <option key={x} value={x}>
                {x}
              </option>
            ))}
          </Select>
        </Field>
      </div>
      <div className="chk items-center">
        <input id="age-on" type="checkbox" checked={f.ageOn} onChange={(e) => set('ageOn', e.target.checked)} />
        <label htmlFor="age-on" className="flex flex-wrap items-center gap-2">
          {t(s.ageBefore)}
          <input
            className="in w-[60px] text-center"
            type="number"
            min={1}
            max={365}
            aria-label={t(s.ageDays)}
            value={f.age}
            disabled={!f.ageOn}
            onChange={(e) => set('age', Math.max(1, Math.min(365, Number(e.target.value) || 1)))}
          />
          {t(s.ageAfter)}
        </label>
      </div>
      <div className="flex flex-col gap-2">
        <span className="text-[13px] font-semibold">{t(m.startingPoint)}</span>
        <OptionCard
          name="start"
          value="after_on"
          checked={f.startMode === 'after_on'}
          onChange={() => set('startMode', 'after_on')}
          title={t(s.startAfter)}
          description={d.firstOnAt ? t(s.startAfterHintOn, { date: shortDate(d.firstOnAt), time: clock(d.firstOnAt) }) : t(s.startAfterHint)}
        />
        <OptionCard
          name="start"
          value="latest"
          checked={f.startMode === 'latest'}
          onChange={() => set('startMode', 'latest')}
          title={
            <span className="flex flex-wrap items-center gap-2">
              {t(s.startLatestBefore)}
              <input
                className="in w-[56px] text-center"
                type="number"
                min={1}
                max={20}
                aria-label={t(s.startLatestCount)}
                value={f.startLatest}
                onChange={(e) => set('startLatest', Math.max(1, Math.min(20, Number(e.target.value) || 1)))}
              />
              {t(s.startLatestAfter)}
            </span>
          }
          description={t(s.startLatestHint)}
        />
        {d.firstSentAt ? <span className="k">{t(s.firstIssueOnly, { date: shortDate(d.firstSentAt) })}</span> : null}
      </div>
    </Card>
  )
}

function Segmented<T extends string | number>({ label, value, options, onChange }: { label: string; value: T; options: { value: T; label: string }[]; onChange: (v: T) => void }) {
  return (
    <div className="seg w-fit max-w-full" role="radiogroup" aria-label={label}>
      {options.map((o) => (
        <button key={String(o.value)} type="button" role="radio" aria-checked={value === o.value} className={cn(value === o.value && 'on')} onClick={() => onChange(o.value)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}

function ScheduleCard({ f, set, setSchedule, errors }: Omit<CardProps, 'd'> & { setSchedule: (p: Partial<DigestSchedule>) => void }) {
  const t = useT()
  const sc = f.schedule
  return (
    <Card title={t(m.schedule)} id="set-schedule">
      <div className="flex flex-col gap-1.5">
        <span className="text-[13px] font-semibold">{t(s.send)}</span>
        <Segmented
          label={t(s.send)}
          value={sc.frequency}
          onChange={(v) => setSchedule({ frequency: v })}
          options={[
            { value: 'daily', label: t(s.daily) },
            { value: 'weekly', label: t(s.weekly) },
            { value: 'monthly', label: t(s.monthly) },
            { value: 'items', label: t(s.items) },
          ]}
        />
      </div>
      <div className="grid grid-cols-3 gap-4 max-md:grid-cols-1">
        {sc.frequency === 'weekly' ? (
          <Field label={t(s.day)}>
            <Select value={sc.weekday} onChange={(e) => setSchedule({ weekday: Number(e.target.value) })}>
              {[1, 2, 3, 4, 5, 6, 0].map((day) => (
                <option key={day} value={day}>
                  {t(m.weekday, { day: String(day) })}
                </option>
              ))}
            </Select>
          </Field>
        ) : sc.frequency === 'daily' ? (
          <Field label={t(s.days)}>
            <Select value={sc.weekdaysOnly ? 'weekdays' : 'every'} onChange={(e) => setSchedule({ weekdaysOnly: e.target.value === 'weekdays' })}>
              <option value="every">{t(s.everyDay)}</option>
              <option value="weekdays">{t(s.weekdaysOnly)}</option>
            </Select>
          </Field>
        ) : sc.frequency === 'monthly' ? (
          <Field label={t(s.day)}>
            <Select
              value={sc.monthRule === 'first' ? `first:${sc.weekday}` : `day:${sc.monthDay ?? 1}`}
              onChange={(e) => {
                const [rule, n] = e.target.value.split(':')
                setSchedule(rule === 'first' ? { monthRule: 'first', weekday: Number(n) } : { monthRule: 'day', monthDay: Number(n) })
              }}
            >
              {Array.from({ length: 28 }, (_, i) => i + 1).map((day) => (
                <option key={`d${day}`} value={`day:${day}`}>
                  {t(s.monthDay, { day })}
                </option>
              ))}
              {[1, 2, 3, 4, 5, 6, 0].map((day) => (
                <option key={`f${day}`} value={`first:${day}`}>
                  {t(s.firstWeekday, { day: t(m.weekday, { day: String(day) }) })}
                </option>
              ))}
            </Select>
          </Field>
        ) : (
          <Field label={t(s.itemCount)}>
            <Input type="number" min={1} max={50} value={sc.count ?? 3} onChange={(e) => setSchedule({ count: Math.max(1, Math.min(50, Number(e.target.value) || 1)) })} />
          </Field>
        )}
        {sc.frequency === 'items' ? (
          <Field label={t(s.window)} hint={t(s.windowHint)}>
            <span className="flex items-center gap-2">
              <Input type="time" aria-label={t(s.windowFrom)} value={sc.windowFrom || '08:00'} onChange={(e) => setSchedule({ windowFrom: e.target.value })} />
              <span aria-hidden="true">–</span>
              <Input type="time" aria-label={t(s.windowUntil)} value={sc.windowUntil || '18:00'} onChange={(e) => setSchedule({ windowUntil: e.target.value })} />
            </span>
          </Field>
        ) : (
          <Field label={t(s.time)} error={errors['schedule.time']}>
            <Input type="time" value={sc.time} onChange={(e) => setSchedule({ time: e.target.value })} />
          </Field>
        )}
        <Field label={t(s.timezone)} hint={t(s.timezoneHint)}>
          <Select value={sc.timezone} onChange={(e) => setSchedule({ timezone: e.target.value })}>
            {(TIMEZONES.includes(sc.timezone) ? TIMEZONES : [sc.timezone, ...TIMEZONES]).map((tz) => (
              <option key={tz} value={tz}>
                {tz}
              </option>
            ))}
          </Select>
        </Field>
      </div>
      <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
        <Field label={t(s.preparation)} hint={t(s.preparationHint)} error={errors.prepareHours}>
          <Select value={f.prepareHours} onChange={(e) => set('prepareHours', Number(e.target.value))}>
            {PREPARE.map((h) => (
              <option key={h} value={h}>
                {t(s.hoursBefore, { count: h })}
              </option>
            ))}
          </Select>
        </Field>
        <div className="lbl">
          <span className="text-[13px] font-semibold">{t(s.lateLimit)}</span>
          <Segmented
            label={t(s.lateLimit)}
            value={f.lateHours}
            onChange={(v) => set('lateHours', v)}
            options={[
              { value: 6 as const, label: t(s.hours, { count: 6 }) },
              { value: 24 as const, label: t(s.hours, { count: 24 }) },
              { value: 48 as const, label: t(s.hours, { count: 48 }) },
            ]}
          />
          <span className="k">{t(s.lateHint)}</span>
        </div>
      </div>
      <div className="flex items-center gap-2.5 rounded-lg border border-accent-line bg-accent-tint px-3.5 py-2.5 text-[13px] font-semibold text-accent" role="status">
        <Icon name="calendar" size={16} />
        {sc.frequency === 'items' ? t(s.summaryItems, { schedule: scheduleText(sc) }) : t(s.summary, { schedule: scheduleText(sc), time: prepareClock(sc, f.prepareHours) })}
      </div>
    </Card>
  )
}

function prepareClock(sc: DigestSchedule, hours: number): string {
  const [h, min] = sc.time.split(':').map(Number)
  const total = ((((h ?? 8) * 60 + (min ?? 0) - hours * 60) % 1440) + 1440) % 1440
  return `${String(Math.floor(total / 60)).padStart(2, '0')}:${String(total % 60).padStart(2, '0')}`
}

const routeColors: Record<string, string> = { ses: 'var(--chart-ses)', postal: 'var(--chart-postal)', postmark: 'var(--chart-postmark)' }

function AudienceCard({ f, set, errors, d, ctx }: CardProps & { ctx?: Schemas['ComposerContext'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const a = d.audience
  const rows: [string, number][] = [
    [t(s.suppressed), a.suppressed],
    [t(s.role), a.role],
    [t(s.duplicates), a.duplicates],
    ...(a.stopped ? ([[t(s.stopped), a.stopped]] as [string, number][]) : []),
  ]
  return (
    <Card title={t(s.audienceTitle)} id="set-audience">
      <div className="grid grid-cols-2 gap-4 max-md:grid-cols-1">
        <Field label={t(m.audience)} error={errors.listIds ?? errors.segmentIds}>
          <Select value={f.audience} onChange={(e) => set('audience', e.target.value)}>
            <option value="">{t(s.pickAudience)}</option>
            {(ctx?.lists ?? []).map((l) => (
              <option key={l.id} value={`l:${l.id}`}>
                {t(s.listOption, { name: l.name })}
              </option>
            ))}
            {(ctx?.segments ?? []).map((x) => (
              <option key={x.id} value={`s:${x.id}`}>
                {t(s.segmentOption, { name: x.name })}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t(m.from)} error={errors.senderId}>
          <Select value={f.senderId} onChange={(e) => set('senderId', e.target.value)}>
            {(ctx?.senders ?? []).map((x) => (
              <option key={x.id} value={x.id}>
                {`${x.name} <${x.email}>`}
              </option>
            ))}
          </Select>
        </Field>
      </div>
      {!a.empty ? (
        <div className="rounded-lg border border-line px-3.5 py-3">
          <dl className="m-0 grid grid-cols-[1fr_auto] gap-x-4 gap-y-1 text-[13px]">
            <dt>{a.label}</dt>
            <dd className="n m-0 text-right font-semibold">{count(a.base)}</dd>
            {rows.map(([label, v]) => (
              <div key={label} className="contents text-ink-3">
                <dt>{label}</dt>
                <dd className="n m-0 text-right">−{count(v)}</dd>
              </div>
            ))}
            <dt className="mt-1 border-t border-line-2 pt-2 font-semibold">{t(s.recipientsToday)}</dt>
            <dd className="n m-0 mt-1 border-t border-line-2 pt-2 text-right text-[18px] font-bold">{count(a.final)}</dd>
          </dl>
          <span className="k mt-1.5 block">{t(s.recountedEach)}</span>
        </div>
      ) : null}
      <div className="flex flex-col gap-2">
        <div className="flex items-center justify-between gap-2">
          <span className="text-[13px] font-semibold">{t(s.routeTitle)}</span>
          <Link to="/w/$workspace/providers/routing" params={{ workspace: slug }} className="lnk text-[13px] font-semibold">
            {t(s.editRoute)}
          </Link>
        </div>
        {d.route.shares.length ? (
          <>
            <StackBar label={d.route.shares.map((x) => `${x.name} ${Math.round(x.percent)}%`).join(', ')} height={8} segments={d.route.shares.map((x) => ({ value: x.percent, color: routeColors[x.adapter] ?? 'var(--chart-other)' }))} />
            <div className="flex flex-wrap gap-x-5 gap-y-1">
              {d.route.shares.map((x) => (
                <LegendItem
                  key={x.providerId}
                  color={routeColors[x.adapter] ?? 'var(--chart-other)'}
                  title={`${x.name} · ${Math.round(x.percent)}%`}
                  detail={x.status === 'warming_up' && x.warmupDay ? t(s.warming, { day: x.warmupDay, days: x.warmupDays ?? 0 }) : t(s.healthy)}
                />
              ))}
            </div>
          </>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center gap-2 text-[13px]">
        <span className="font-semibold">{t(s.autoPause)}</span>
        <span>{t(s.autoPauseValue, { bounce: rate(d.thresholds.bounce), complaint: rate(d.thresholds.complaint) })}</span>
      </div>
    </Card>
  )
}

function ModeCard({ f, set, d }: Omit<CardProps, 'errors'>) {
  const t = useT()
  const canSend = useCan('campaigns:send')
  return (
    <Card title={t(s.modeTitle)} id="set-mode">
      <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1" role="radiogroup" aria-label={t(s.modeTitle)}>
        <OptionCard name="mode" value="auto" checked={f.mode === 'auto'} onChange={() => set('mode', 'auto')} disabled={!canSend && d.mode !== 'auto'} title={t(m.modeAuto)} description={t(s.autoHint)}>
          <span className="k">{t(s.autoNeeds)}</span>
        </OptionCard>
        <OptionCard name="mode" value="draft" checked={f.mode === 'draft'} onChange={() => set('mode', 'draft')} title={t(m.modeDraft)} description={t(s.draftHint)}>
          <span className="k">{t(s.draftNeeds)}</span>
        </OptionCard>
      </div>
      {!canSend ? <Note tone="info">{t(s.draftOnly, { helpers: formatList(d.sendHelpers, 'disjunction') })}</Note> : null}
      {d.owner ? (
        <div className="flex flex-wrap items-center gap-2 text-[13px]">
          <span className="font-semibold">{d.owner.name}</span>
          <span className="k">{d.ownerSince ? t(s.ownerSince, { date: shortDate(d.ownerSince) }) : t(s.ownerNow)}</span>
        </div>
      ) : (
        <span className="k">{t(s.noOwner)}</span>
      )}
    </Card>
  )
}

/** The feed's newest items with their badges. */
function PreviewCard({ d }: { d: Digest }) {
  const t = useT()
  const q = useDigestFeed(d.id, 6)
  const p = q.data
  const parts = useMemo(() => {
    if (!p) return [] as string[]
    const out: string[] = []
    for (const x of p.olderSent) out.push(t(m.olderSent, { count: x.count, number: x.number }))
    if (p.olderLeftOut) out.push(t(m.olderLeftOut, { count: p.olderLeftOut }))
    if (p.olderBefore) out.push(t(m.olderBefore, { count: p.olderBefore }))
    if (p.olderNew) out.push(t(m.olderNew, { count: p.olderNew }))
    return out
  }, [p, t])
  const older = p ? p.total - p.items.length : 0
  return (
    <section className="card" aria-labelledby="feed-preview">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="feed-preview">
            {t(s.previewTitle)}
          </h2>
          {p ? <span className="k">{t(s.previewSub, { shown: p.items.length, total: p.total, url: d.feed.url.replace(/^https?:\/\//, '') })}</span> : null}
        </div>
      </div>
      {!p ? (
        <div className="pad flex flex-col gap-3">
          {Array.from({ length: 4 }, (_, i) => (
            <Skeleton key={i} w="100%" h={36} />
          ))}
        </div>
      ) : p.items.length === 0 ? (
        <p className="pad k m-0">{t(s.previewEmpty)}</p>
      ) : (
        <>
          <div className="flex flex-col px-[18px]">
            {p.items.map((it, i) => (
              <FeedItemRow key={it.id} item={it} withTime={i < 2} showCategory={it.status === 'left_out'} className="border-b border-line-2 last:border-b-0" />
            ))}
          </div>
          {older > 0 ? <p className="k m-0 border-t border-line-2 px-[18px] py-2.5 text-[12px]">{t(m.olderItems, { count: older, parts: parts.join(', ') })}</p> : null}
        </>
      )}
    </section>
  )
}

function NextCard({ d, onPreview }: { d: Digest; onPreview: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const n = d.next
  if (!n) return null
  return (
    <section className="card" aria-labelledby="next-issue">
      <div className="sec">
        <h2 className="h2" id="next-issue">
          {t(m.nextIssueNumber, { number: n.number })}
        </h2>
        <IssueStatusBadge status={n.status} />
      </div>
      <div className="pad flex flex-col gap-3">
        <dl className="kv">
          <dt>{t(m.sendTime)}</dt>
          <dd>{n.sendAt ? `${shortDate(n.sendAt)}, ${clock(n.sendAt)}` : '–'}</dd>
          <dt>{t(m.preparedLabel)}</dt>
          <dd>{n.prepareAt ? t(s.preparedChecks, { time: clock(n.prepareAt) }) : '–'}</dd>
          <dt>{t(m.postsLabel)}</dt>
          <dd>{count(n.posts)}</dd>
          <dt>{t(m.recipientsLabel)}</dt>
          <dd>{t(m.about, { count: count(n.recipients) })}</dd>
          <dt>{t(s.approval)}</dt>
          <dd>{d.approvals.needed ? t(s.approvalNeeded, { count: d.approvals.rules.length }) : t(s.approvalNone)}</dd>
        </dl>
        <div className="flex flex-col gap-2">
          <span className="text-[12px] font-semibold text-ink-2">{t(s.subjectPreview)}</span>
          <div className="flex flex-col">
            <div className="border-b border-line-2 pb-2">
              <span className="k block text-[12px]">{t(s.subjectNow, { count: n.posts })}</span>
              <span className="font-semibold">{n.posts ? n.subject : t(m.subjectNoPosts)}</span>
            </div>
            {n.subjects.map((x) => (
              <div key={x.items} className="border-b border-line-2 py-2 last:border-b-0">
                <span className="k block text-[12px]">{t(s.subjectWith, { count: x.items })}</span>
                <span className={cn('font-semibold', x.zeroMore && 'text-bad-fg')}>{x.text}</span>
                {x.zeroMore ? <span className="err block text-[12px]">{t(s.zeroMore)}</span> : null}
              </div>
            ))}
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button size="sm" onClick={onPreview}>
            {t(m.previewNext)}
          </Button>
          {d.templateId ? (
            <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: d.templateId }}>
              {t(s.designEmail)}
            </ButtonLink>
          ) : null}
        </div>
      </div>
    </section>
  )
}

/** What happens to every issue: prepared, approval requested, sent once approved, the late limit. */
function ChecksCard({ d }: { d: Digest }) {
  const t = useT()
  const { slug } = useWorkspace()
  const n = d.next
  const day = (iso?: string) => (iso ? t(m.sendTimeValue, { day: String(zonedParts(iso).weekday), time: clock(iso) }) : '–')
  const clauses = ruleClauses(t, d)
  const owner = d.owner?.name ?? ''
  const steps: { icon: IconName; tone: string; title: string; body: string }[] = [
    { icon: 'arrowRight', tone: 'bg-info-bg text-info-fg', title: t(s.stepPrepared, { when: day(n?.prepareAt) }), body: t(s.stepPreparedBody, { owner }) },
  ]
  if (d.approvals.needed) {
    steps.push({ icon: 'clock', tone: 'bg-warn-bg text-warn-fg', title: t(s.stepApproval, { when: day(n?.prepareAt) }), body: t(s.stepApprovalBody, { owner, approvers: formatList(d.approvals.approvers, 'disjunction') }) })
  }
  steps.push({ icon: 'check', tone: 'bg-ok-bg text-ok-fg', title: d.approvals.needed ? t(s.stepSentApproved, { when: day(n?.sendAt) }) : t(s.stepSent, { when: day(n?.sendAt) }), body: t(s.stepSentBody) })
  steps.push({ icon: 'more', tone: 'bg-neutral-bg text-neutral-fg', title: t(s.stepLate, { when: day(n?.deadline) }), body: t(s.stepLateBody) })
  return (
    <section className="card" aria-labelledby="checks-approvals">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="checks-approvals">
            {t(s.checksTitle)}
          </h2>
          <span className="k">{t(s.checksSub)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        {d.approvals.needed ? <Note tone="info">{t(s.policyNote, { rules: formatList(clauses) })}</Note> : null}
        <ol className="m-0 flex list-none flex-col p-0">
          {steps.map((x) => (
            <li key={x.title} className="flex items-start gap-3 border-b border-line-2 py-2.5 last:border-b-0">
              <span className={cn('grid h-6 w-6 shrink-0 place-items-center rounded-full', x.tone)} aria-hidden="true">
                <Icon name={x.icon} size={12} strokeWidth={2.2} />
              </span>
              <div className="flex min-w-0 flex-col gap-0.5">
                <span className="font-semibold">{x.title}</span>
                <span className="k">{x.body}</span>
              </div>
            </li>
          ))}
        </ol>
        <Link to="/w/$workspace/settings/approvals" params={{ workspace: slug }} className="lnk text-[13px] font-semibold">
          {t(s.approvalPolicy)}
        </Link>
      </div>
    </section>
  )
}
