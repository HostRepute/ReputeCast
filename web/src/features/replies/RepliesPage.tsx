import { keepPreviousData, useInfiniteQuery, useQueryClient, type InfiniteData } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useMemo, useState } from 'react'
import { $api, ApiError, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Icon } from '@/components/icons/Icon'
import { Avatar, type AvatarColor } from '@/components/ui/Avatar'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, Skeleton } from '@/components/ui/Feedback'
import { Pager } from '@/components/ui/Pager'
import { Check, SelectAll } from '@/components/ui/SelectAll'
import { toast } from '@/components/ui/Toast'
import { failed } from '@/features/connect/parts'
import { AudienceTabs } from '@/features/audience/AudienceTabs'
import { m as am } from '@/features/audience/messages'
import { T, useT } from '@/i18n'
import { ago, count, notificationTime, shortDate } from '@/lib/format'
import { useDebounced } from '@/lib/hooks'
import { isTypingTarget } from '@/lib/shortcuts'
import { useCan, usePermissionReason, useWorkspace } from '@/lib/session'
import { cn } from '@/lib/cn'
import { m } from './messages'
import './replies.css'

type ReplyList = Schemas['ReplyList']
type Item = Schemas['ReplyListItem']
type Detail = Schemas['ReplyDetail']
export type ReplyView = 'open' | 'done' | 'auto' | 'spam' | 'other' | 'unsubscribes'

export interface RepliesSearch {
  view?: ReplyView
  q?: string
  campaign?: string
  automation?: string
  reply?: string
}

const listPath = '/api/v1/workspaces/{workspace}/replies' as const
const PAGE = 50

/** The board's avatar colours, in list order. */
const AVATAR: AvatarColor[] = ['info', 'pink', 'warn', 'ok', 'purple', 'sky']

const VIEWS: { view: ReplyView; chip: keyof typeof m; head: keyof typeof m; name: keyof typeof m; count: keyof Schemas['ReplyCounts'] }[] = [
  { view: 'open', chip: 'chipOpen', head: 'listHeadOpen', name: 'viewOpen', count: 'open' },
  { view: 'done', chip: 'chipDone', head: 'listHeadDone', name: 'viewDone', count: 'done' },
  { view: 'auto', chip: 'chipAuto', head: 'listHeadAuto', name: 'viewAuto', count: 'auto' },
  { view: 'spam', chip: 'chipSpam', head: 'listHeadSpam', name: 'viewSpam', count: 'spam' },
  { view: 'other', chip: 'chipOther', head: 'listHeadOther', name: 'viewOther', count: 'other' },
  { view: 'unsubscribes', chip: 'chipUnsubscribes', head: 'listHeadUnsubscribes', name: 'viewUnsubscribes', count: 'unsubscribes' },
]

/** Audience › Replies (M58 G-17): what readers wrote back, by view. */
export function RepliesPage({ search }: { search: RepliesSearch }) {
  const t = useT()
  const { slug } = useWorkspace()
  const can = useCan('replies:view')
  const navigate = useNavigate()
  const qc = useQueryClient()
  const view: ReplyView = search.view ?? 'open'
  const [text, setText] = useState(search.q ?? '')
  const q = useDebounced(text.trim(), 300)
  const go = (next: RepliesSearch, replace = false) =>
    void navigate({ to: '/w/$workspace/audience/replies', params: { workspace: slug }, search: next, replace })
  // The search field writes the URL once typing stops.
  useEffect(() => {
    if ((search.q ?? '') !== q) go({ ...search, q: q || undefined, reply: undefined }, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q])
  const key = ['get', listPath, slug, view, search.q ?? '', search.campaign ?? '', search.automation ?? ''] as const
  const list = useInfiniteQuery({
    queryKey: key,
    initialPageParam: '',
    enabled: can,
    placeholderData: keepPreviousData,
    queryFn: async ({ pageParam }) => {
      const { data } = await http.GET(listPath, {
        params: {
          path: { workspace: slug },
          query: { view, q: search.q || undefined, campaign: search.campaign, automation: search.automation, cursor: pageParam || undefined, limit: PAGE },
        },
      })
      return data!
    },
    getNextPageParam: (last) => last.nextCursor || undefined,
  })
  const first = list.data?.pages[0]
  const items = useMemo(() => list.data?.pages.flatMap((p) => p.items) ?? [], [list.data])
  const selectedID = search.reply ?? items[0]?.id
  // The selection belongs to one view and search: changing either clears it.
  const selKey = `${view}|${search.q ?? ''}`
  const [sel, setSel] = useState<{ key: string; ids: string[] }>({ key: selKey, ids: [] })
  const selected = sel.key === selKey ? sel.ids : []
  const setSelected = (next: string[] | ((cur: string[]) => string[])) =>
    setSel((s) => ({ key: selKey, ids: typeof next === 'function' ? next(s.key === selKey ? s.ids : []) : next }))

  // Mark done / open: the rows leave the view at once and come back if the server refuses.
  const status = $api.useMutation('post', '/api/v1/workspaces/{workspace}/replies/status')
  const mark = (ids: string[], to: 'open' | 'done', undo = true) => {
    if (!ids.length) return
    const prev = qc.getQueryData<InfiniteData<ReplyList>>(key)
    qc.setQueryData<InfiniteData<ReplyList>>(key, (d) =>
      d
        ? {
            ...d,
            pages: d.pages.map((p) => ({
              ...p,
              items: p.items.filter((x) => !ids.includes(x.id)),
              counts: moveCounts(p.counts, view, to, p.items.filter((x) => ids.includes(x.id)).length),
            })),
          }
        : d,
    )
    setSelected([])
    if (search.reply && ids.includes(search.reply)) {
      const idx = items.findIndex((x) => x.id === search.reply)
      const next = items.filter((x) => !ids.includes(x.id))[Math.min(idx, items.length - ids.length - 1)]
      go({ ...search, reply: next?.id }, true)
    }
    const run = () =>
      status.mutate(
        { params: { path: { workspace: slug } }, body: { ids, status: to } },
        {
          onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ['get', listPath] })
            void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/replies/{id}'] })
            if (undo)
              toast({
                title: to === 'done' ? t(m.markedDone, { count: ids.length }) : t(m.markedOpen, { count: ids.length }),
                action: { label: t(m.undo), onClick: () => mark(ids, to === 'done' ? 'open' : 'done', false) },
              })
          },
          onError: (e) => {
            qc.setQueryData(key, prev)
            failed(t(m.markFailed), run)(e)
          },
        },
      )
    run()
  }

  // j / k move through the list, e marks the open reply done.
  useEffect(() => {
    if (!can) return
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey || isTypingTarget(e.target)) return
      if (document.querySelector('[role="dialog"][aria-modal="true"]')) return
      const k = e.key.toLowerCase()
      if (k !== 'j' && k !== 'k' && k !== 'e') return
      const idx = items.findIndex((x) => x.id === selectedID)
      if (k === 'e') {
        const cur = items[idx]
        if (cur && cur.status === 'open' && first?.canManage) {
          e.preventDefault()
          mark([cur.id], 'done')
        }
        return
      }
      const next = items[k === 'j' ? Math.min(idx + 1, items.length - 1) : Math.max(idx - 1, 0)]
      if (next && next.id !== selectedID) {
        e.preventDefault()
        go({ ...search, reply: next.id }, true)
        window.requestAnimationFrame(() => document.getElementById(`reply-${next.id}`)?.focus())
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  })

  const denied = !can || (list.error instanceof ApiError && list.error.status === 403)
  const viewName = t(m[VIEWS.find((v) => v.view === view)?.name ?? 'viewOpen'])
  let body
  if (denied) {
    const helpers = list.error instanceof ApiError ? (list.error.problem.helpers ?? []) : []
    body = (
      <EmptyState
        title={t(m.deniedTitle)}
        action={
          helpers.length ? (
            <>
              {helpers.map((h) => (
                <span key={h} className="flex items-center gap-2.5 rounded-full border border-line py-2 pr-3.5 pl-2 text-left">
                  <Avatar name={h} size={32} />
                  <span className="font-semibold">{h}</span>
                </span>
              ))}
            </>
          ) : undefined
        }
      >
        {t(m.deniedBody)}
      </EmptyState>
    )
  } else if (list.isError && !first) {
    body = (
      <EmptyState title={t(m.errorTitle)} action={<Button onClick={() => void list.refetch()}>{t(m.tryAgain)}</Button>}>
        {t(m.errorBody)}
      </EmptyState>
    )
  } else if (!first) {
    body = <RepliesSkeleton />
  } else if (!first.receiving && totalOf(first.counts) === 0) {
    body = (
      <EmptyState
        title={t(m.noRouteTitle)}
        action={
          <ButtonLink variant="primary" to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
            {t(m.setUp)}
          </ButtonLink>
        }
      >
        {t(m.noRouteBody)}
      </EmptyState>
    )
  } else if (totalOf(first.counts) === 0 && !search.q) {
    body = (
      <EmptyState
        title={t(m.emptyTitle)}
        action={
          <ButtonLink to="/w/$workspace/settings/senders" params={{ workspace: slug }}>
            {t(m.openReceiving)}
          </ButtonLink>
        }
      >
        {t(m.emptyBodyNoDomain)}
      </EmptyState>
    )
  } else {
    body = (
      <>
        <div className="rp-tools">
          <div className="rp-chips" role="group" aria-label={t(m.chipsLabel)} data-scroll-row>
            {VIEWS.map((v) => (
              <button
                key={v.view}
                type="button"
                className="rp-chip"
                aria-pressed={view === v.view}
                onClick={() => go({ ...search, view: v.view === 'open' ? undefined : v.view, reply: undefined })}
              >
                {t(m[v.chip], { count: count(first.counts[v.count]) })}
              </button>
            ))}
          </div>
          <label className="rp-search">
            <Icon name="search" size={14} />
            <input type="search" placeholder={t(m.search)} aria-label={t(m.search)} value={text} onChange={(e) => setText(e.target.value)} />
          </label>
        </div>
        {items.length === 0 ? (
          search.q ? (
            <EmptyState
              title={t(m.noResultsTitle, { q: search.q })}
              action={
                <Button
                  onClick={() => {
                    setText('')
                    go({ ...search, q: undefined })
                  }}
                >
                  {t(m.clearSearch)}
                </Button>
              }
            >
              {t(m.noResultsBody, { view: viewName })}
            </EmptyState>
          ) : (
            <EmptyState title={t(m.emptyView)}>{t(m.emptyViewBody)}</EmptyState>
          )
        ) : (
          <div className={cn('rpanes', search.reply && 'rp-has-detail')}>
            <section className="card rp-list" aria-label={t(m.listLabel)}>
              {selected.length ? (
                <div className="rp-bulk">
                  <span className="text-[13px] font-semibold">{t(m.selected, { count: count(selected.length) })}</span>
                  {first.canManage ? (
                    <>
                      <Button size="sm" onClick={() => mark(selected, 'done')}>
                        {t(m.markDone)}
                      </Button>
                      <Button size="sm" onClick={() => mark(selected, 'open')}>
                        {t(m.markOpen)}
                      </Button>
                    </>
                  ) : null}
                  <Button size="sm" className="ml-auto" onClick={() => setSelected([])}>
                    {t(m.clearSelection)}
                  </Button>
                </div>
              ) : null}
              <div className="rp-head">
                <SelectAll
                  className="rck"
                  label={t(m.selectAll, { count: items.length })}
                  all={selected.length > 0 && selected.length === items.length}
                  some={selected.length > 0}
                  onChange={(on) => setSelected(on ? items.map((x) => x.id) : [])}
                />
                <span className="k">{t(m[VIEWS.find((v) => v.view === view)?.head ?? 'listHeadOpen'], { count: first.counts[VIEWS.find((v) => v.view === view)?.count ?? 'open'] })}</span>
              </div>
              {items.map((it, i) => (
                <ReplyRow
                  key={it.id}
                  item={it}
                  color={AVATAR[i % AVATAR.length] ?? 'info'}
                  on={it.id === selectedID}
                  checked={selected.includes(it.id)}
                  onCheck={(c) => setSelected((s) => (c ? [...s, it.id] : s.filter((x) => x !== it.id)))}
                  search={search}
                />
              ))}
              {list.hasNextPage ? (
                <div className="border-t border-line-2 px-3.5 py-2.5">
                  <Pager query={list} label={t(m.loadMore)} />
                </div>
              ) : null}
            </section>
            {selectedID ? (
              <ReplyDetail
                id={selectedID}
                color={AVATAR[Math.max(0, items.findIndex((x) => x.id === selectedID)) % AVATAR.length] ?? 'info'}
                onBack={() => go({ ...search, reply: undefined })}
                onMark={(id, to) => mark([id], to)}
              />
            ) : null}
          </div>
        )}
      </>
    )
  }
  return (
    <Page title={t(am.title)} documentTitle={t(m.documentTitle)} filters={<AudienceTabs />} contentClassName="rp-main">
      {body}
    </Page>
  )
}

function totalOf(c: Schemas['ReplyCounts']) {
  return c.open + c.done + c.auto + c.spam + c.other + c.unsubscribes
}

/** The chip counts after n replies moved from one status to the other. */
function moveCounts(c: Schemas['ReplyCounts'], view: ReplyView, to: 'open' | 'done', n: number): Schemas['ReplyCounts'] {
  if (view === 'open' && to === 'done') return { ...c, open: Math.max(0, c.open - n), done: c.done + n }
  if (view === 'done' && to === 'open') return { ...c, done: Math.max(0, c.done - n), open: c.open + n }
  return c
}

function personName(it: Item) {
  return it.from.name || it.from.email
}

function ReplyRow({
  item,
  color,
  on,
  checked,
  onCheck,
  search,
}: {
  item: Item
  color: AvatarColor
  on: boolean
  checked: boolean
  onCheck: (c: boolean) => void
  search: RepliesSearch
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const name = personName(item)
  return (
    <div className={cn('rrow', on && 'on')}>
      <Check className="rck" checked={checked} onChange={(e) => onCheck(e.target.checked)} aria-label={t(m.selectOne, { name })} />
      <Avatar name={name} size={32} fontSize={12} color={color} />
      <Link
        id={`reply-${item.id}`}
        className="rl"
        to="/w/$workspace/audience/replies"
        params={{ workspace: slug }}
        search={{ ...search, reply: item.id }}
        aria-current={on || undefined}
      >
        <span className="r1">
          <span className="tr rn">{name}</span>
          {item.from.name ? <span className="tr re k mono">{item.from.email}</span> : <span className="re" />}
          <span className="ra">{ago(item.receivedAt)}</span>
        </span>
        <span className="tr rs">{item.subject || t(m.noSubject)}</span>
        <span className="tr rv">{item.preview}</span>
        {item.source || item.autoReply || item.spam || item.asksUnsubscribe ? (
          <span className="rm">
            {item.source ? (
              <span className="k text-[12px]">{item.source.name}</span>
            ) : null}
            <Flags item={item} />
          </span>
        ) : null}
      </Link>
    </div>
  )
}

function Flags({ item }: { item: Item }) {
  const t = useT()
  return (
    <>
      {item.autoReply ? <Badge tone="neutral">{t(m.badgeAuto)}</Badge> : null}
      {item.spam ? <Badge tone="bad">{t(m.badgeSpam)}</Badge> : null}
      {item.asksUnsubscribe ? <Badge tone="warn">{t(m.badgeAsks)}</Badge> : null}
    </>
  )
}

const CONTACT_TONE: Record<string, [Tone, keyof typeof m]> = {
  active: ['ok', 'subscribedBadge'],
  unsubscribed: ['neutral', 'unsubscribedBadge'],
  bounced: ['bad', 'bouncedBadge'],
  complained: ['bad', 'complainedBadge'],
  pending: ['warn', 'pendingBadge'],
}

function size(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

/** One reply, opened beside the list (or as its own view with Back at ≤1100 px). */
function ReplyDetail({ id, color, onBack, onMark }: { id: string; color: AvatarColor; onBack: () => void; onMark: (id: string, to: 'open' | 'done') => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const workspace = me.workspace.name
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/replies/{id}', { params: { path: { workspace: slug, id } } })
  const [unsub, setUnsub] = useState(false)
  const d = q.data
  // Who can help, in the app's own words (never the permission key).
  const manageReason = usePermissionReason('replies:manage', d?.helpers)
  const editReason = usePermissionReason('subscribers:edit', d?.helpers)
  if (!d) {
    return (
      <section className="card rp-detail" aria-label={t(m.detailLoading)} aria-busy="true">
        <div className="flex flex-col gap-2.5 border-b border-line-2 px-5 py-4" aria-hidden="true">
          <Skeleton w="55%" h={18} />
          <Skeleton w="40%" h={12} />
          <Skeleton w="30%" h={12} />
          <Skeleton w="45%" h={12} />
        </div>
        <div className="flex flex-col gap-3 px-5 py-4" aria-hidden="true">
          <Skeleton w="100%" h={56} />
          <Skeleton w="90%" h={12} />
          <Skeleton w="80%" h={12} />
          <Skeleton w="60%" h={12} />
        </div>
      </section>
    )
  }
  const name = d.from.name || d.from.email
  const first = (d.contact?.name || name).split(' ')[0] ?? name
  const contactTone = d.contact ? CONTACT_TONE[d.contact.status] : undefined
  const subject = d.subject.toLowerCase().startsWith('re:') ? d.subject : `Re: ${d.subject}`
  const mailto = `mailto:${d.from.email}?subject=${encodeURIComponent(subject)}`
  return (
    <section className="card rp-detail" aria-labelledby="rd-h">
      <div className="hidden border-b border-line-2 px-3 py-2 max-[1100px]:block">
        <Button onClick={onBack}>{t(m.back)}</Button>
      </div>
      <div className="flex flex-col gap-2.5 border-b border-line-2 px-5 py-4 max-md:px-3.5">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="h2 text-[16px]" id="rd-h">
            {d.subject || t(m.noSubject)}
          </h2>
          <Flags item={d} />
        </div>
        <dl className="kv">
          <dt>{t(m.from)}</dt>
          <dd>
            {d.from.name ? (
              <>
                {d.from.name} <span className="mono text-[12.5px] text-ink-2">{`<${d.from.email}>`}</span>
              </>
            ) : (
              <span className="mono text-[12.5px]">{d.from.email}</span>
            )}
          </dd>
          <dt>{t(m.received)}</dt>
          <dd>{notificationTime(d.receivedAt)}</dd>
          {d.kind === 'other' && d.to ? (
            <>
              <dt>{t(m.to)}</dt>
              <dd className="mono text-[12.5px]">{d.to}</dd>
            </>
          ) : null}
        </dl>
        {d.source ? (
          <div>
            {d.source.type === 'campaign' ? (
              <Link className="lnk text-[13px] font-semibold no-underline" to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: d.source.id }}>
                {d.sentAt ? t(m.replyTo, { source: d.source.name, date: shortDate(d.sentAt) }) : t(m.replyToUnsent, { source: d.source.name })}
              </Link>
            ) : (
              <Link className="lnk text-[13px] font-semibold no-underline" to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: d.source.id }}>
                {d.sentAt ? t(m.replyTo, { source: d.source.name, date: shortDate(d.sentAt) }) : t(m.replyToUnsent, { source: d.source.name })}
              </Link>
            )}
          </div>
        ) : null}
      </div>
      <div className="flex flex-col gap-4 px-5 py-4 max-md:px-3.5">
        {d.contact ? (
          <div className="rcontact">
            <Avatar name={d.contact.name || d.contact.email} size={32} fontSize={12} color={color} />
            <div className="flex min-w-0 flex-[1_1_180px] flex-col gap-0.5">
              <span className="font-semibold">{d.contact.name || d.contact.email}</span>
              <span className="k mono [overflow-wrap:anywhere]">{d.contact.email}</span>
            </div>
            {contactTone ? <Badge tone={contactTone[0]}>{t(m[contactTone[1]])}</Badge> : null}
            <ButtonLink size="sm" to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: d.contact.id }}>
              {t(m.openContact)}
            </ButtonLink>
          </div>
        ) : null}
        <div className="rtext">{d.text}</div>
        {d.attachments.length ? (
          <div className="flex flex-col gap-1.5 rounded-lg border border-line px-3.5 py-3">
            <span className="font-semibold">{t(m.attachments)}</span>
            <ul className="m-0 flex list-none flex-col gap-1 p-0">
              {d.attachments.map((a) => (
                <li key={a.name} className="flex min-w-0 items-center gap-2">
                  <span className="mono text-[12.5px] [overflow-wrap:anywhere]">{t(m.attachmentLine, { name: a.name, size: size(a.size) })}</span>
                </li>
              ))}
            </ul>
            <span className="k">{t(m.attachmentsNotKept, { name: first })}</span>
          </div>
        ) : null}
      </div>
      <div className="rp-actions flex flex-wrap items-center gap-2 border-t border-line-2 px-5 pt-3 pb-3.5 max-md:px-3.5">
        <a href={mailto} className="btn pri">
          {t(m.replyByEmail)}
        </a>
        {d.kind !== 'unsubscribe' ? (
          <Button onClick={() => onMark(d.id, d.status === 'done' ? 'open' : 'done')} disabledReason={manageReason}>
            {d.status === 'done' ? t(m.markOpen) : t(m.markDone)}
          </Button>
        ) : null}
        {d.contact && d.contact.status === 'active' ? (
          <Button variant="danger" onClick={() => setUnsub(true)} disabledReason={manageReason ?? editReason}>
            {t(m.unsubscribeContact)}
          </Button>
        ) : null}
        {d.contact ? (
          <ButtonLink to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: d.contact.id }}>
            {t(m.openContact)}
          </ButtonLink>
        ) : null}
        <span className="k ml-auto max-md:hidden">
          <T msg={m.keyboardHint} values={{ kbd: (c) => <kbd>{c}</kbd> }} />
        </span>
      </div>
      {d.contact ? (
        <UnsubscribeDialog open={unsub} onOpenChange={setUnsub} reply={d} name={d.contact.name || d.contact.email} first={first} workspace={workspace} />
      ) : null}
    </section>
  )
}

function UnsubscribeDialog({
  open,
  onOpenChange,
  reply,
  name,
  first,
  workspace,
}: {
  open: boolean
  onOpenChange: (o: boolean) => void
  reply: Detail
  name: string
  first: string
  workspace: string
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const unsub = $api.useMutation('post', '/api/v1/workspaces/{workspace}/replies/{id}/unsubscribe-contact')
  const run = () =>
    unsub.mutate(
      { params: { path: { workspace: slug, id: reply.id } } },
      {
        onSuccess: (r) => {
          onOpenChange(false)
          void qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/replies/{id}'] })
          toast({ kind: 'success', title: r.already ? t(m.unsubAlready, { name }) : t(m.unsubDone, { name }) })
        },
        onError: failed(t(m.unsubFailed), run),
      },
    )
  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      dismissible={false}
      eyebrow={<span className="k">{t(m.unsubEyebrow, { subject: reply.subject })}</span>}
      title={t(m.unsubTitle, { name })}
      footer={
        <>
          <Button onClick={() => onOpenChange(false)}>{t(m.cancel)}</Button>
          <Button variant="danger-primary" loading={unsub.isPending} onClick={run}>
            {t(m.unsubConfirm)}
          </Button>
        </>
      }
    >
      <p className="m-0 text-[13.5px] text-ink-2">{t(m.unsubBody, { name: first, workspace })}</p>
      <span className="k">{t(m.unsubEvidence)}</span>
    </Dialog>
  )
}

function RepliesSkeleton() {
  const t = useT()
  return (
    <div className="rpanes" aria-busy="true">
      <span className="vh" role="status">
        {t(m.loading)}
      </span>
      <section className="card rp-list" aria-label={t(m.listLoading)}>
        <div className="rp-head" aria-hidden="true">
          <Skeleton w={16} h={16} />
          <Skeleton w={150} h={12} />
        </div>
        {[0, 1, 2, 3, 4].map((i) => (
          <div key={i} className="rrow" aria-hidden="true">
            <Skeleton w={16} h={16} className="mt-2 flex-none" />
            <Skeleton w={32} h={32} r={16} className="flex-none" />
            <div className="flex min-w-0 flex-1 flex-col gap-[7px]">
              <Skeleton w="60%" h={13} />
              <Skeleton w="85%" h={12} />
              <Skeleton w="95%" h={11} />
              <Skeleton w="40%" h={11} />
            </div>
          </div>
        ))}
      </section>
      <section className="card rp-detail max-[1100px]:hidden" aria-label={t(m.detailLoading)}>
        <div className="flex flex-col gap-2.5 border-b border-line-2 px-5 py-4" aria-hidden="true">
          <Skeleton w="55%" h={18} />
          <Skeleton w="40%" h={12} />
          <Skeleton w="30%" h={12} />
          <Skeleton w="45%" h={12} />
        </div>
        <div className="flex flex-col gap-3 px-5 py-4" aria-hidden="true">
          <Skeleton w="100%" h={56} />
          <Skeleton w="90%" h={12} />
          <Skeleton w="80%" h={12} />
        </div>
      </section>
    </div>
  )
}
