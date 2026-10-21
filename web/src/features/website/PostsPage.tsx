import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Select } from '@/components/ui/Field'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { NoResults } from '@/components/ui/NoResults'
import { TableScroll } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { count, dayMonthDate, orList } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { copyText } from '@/features/forms/common'
import { bare, useInvalidateWebsite, WS, type Overview, type PostRow } from './api'
import { m } from './messages'
import { PastCampaignsDialog } from './PastCampaigns'
import { PostStateDialog, PublishTimeDialog, RedirectDialog, DeletePostDialog, UnpublishDialog, PostPreviewDialog } from './PostDialogs'
import { mp } from './posts.messages'
import { dateTime, scheduledLine, visibilityIcon } from './postFormat'
import { ReadOnlyNote, WebsitePage } from './WebsiteFrame'

export type PostsSearch = {
  status?: 'published' | 'scheduled' | 'draft' | 'unpublished'
  q?: string
  visibility?: 'public' | 'members' | 'paid'
  tag?: string
  source?: 'campaign' | 'web'
  page?: number
  /** Publish past campaigns is open. */
  past?: boolean
  /** A post's state card is open (approval, unpublished, scheduled). */
  post?: string
}

const PAGE = 8

type Dialog = { kind: 'unpublish' | 'delete' | 'redirect' | 'time' | 'preview'; post: PostRow } | null

/** Website › Posts (WebsitePosts and its tablet and phone boards, WebsiteStates · posts). */
export function PostsPage({ overview, search, onSearch }: { overview: Overview; search: PostsSearch; onSearch: (s: PostsSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const canEdit = useCan('website:edit')
  const canPublish = useCan('website:publish')
  const [q, setQ] = useState(search.q ?? '')
  const [shown, setShown] = useState(search.q)
  if (shown !== search.q) {
    setShown(search.q)
    setQ(search.q ?? '')
  }
  useEffect(() => {
    const h = window.setTimeout(() => {
      if ((search.q ?? '') !== q) onSearch({ ...search, q: q || undefined, page: undefined }, true)
    }, 300)
    return () => window.clearTimeout(h)
  }, [q, search, onSearch])
  const query = { status: (search.status ?? 'all') as NonNullable<PostsSearch['status']> | 'all', visibility: search.visibility, tag: search.tag, source: search.source, q: search.q, page: search.page ?? 1, pageSize: PAGE }
  const list = $api.useQuery('get', `${WS}/posts`, { params: { path: { workspace: slug }, query } }, { placeholderData: (p) => p })
  const tags = $api.useQuery('get', `${WS}/tags`, { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const [dialog, setDialog] = useState<Dialog>(null)
  const [creating, setCreating] = useState(false)
  const d = list.data
  const set = (patch: Partial<PostsSearch>) => onSearch({ ...search, ...patch, page: patch.page })
  const n = overview.numbers
  const newPost = async () => {
    setCreating(true)
    try {
      const { data } = await http.POST(`${WS}/posts`, { params: { path: { workspace: slug } }, body: {} })
      if (data) void navigate({ to: '/w/$workspace/forms/website/posts/$post', params: { workspace: slug, post: data.id } })
    } catch (e) {
      toast({ kind: 'error', title: t(m.createFailed), detail: errorMessage(e) })
    } finally {
      setCreating(false)
    }
  }
  const actions =
    canEdit || canPublish ? (
      <>
        {canPublish ? <Button onClick={() => onSearch({ ...search, past: true })}>{t(m.publishPast)}</Button> : null}
        {canEdit ? (
          <Button variant="primary" loading={creating} onClick={() => void newPost()}>
            {t(m.newPost)}
          </Button>
        ) : null}
      </>
    ) : undefined
  const filtered = Boolean(search.status || search.q || search.visibility || search.tag || search.source)
  const open = search.post ? d?.posts.find((p) => p.id === search.post) : undefined
  return (
    <WebsitePage view="posts" overview={overview} actions={actions}>
      {!canEdit && !canPublish ? <ReadOnlyNote msg={m.readOnlyPosts} helpers={overview.helpers.publish} perms={{ edit: 'website:edit', publish: 'website:publish' }} /> : null}
      <div className="ws-stats">
        <Stat label={t(mp.statViews)} value={count(n.views)} sub={t(mp.statViewsSub)} />
        <Stat label={t(mp.statVisitors)} value={count(n.visitors)} sub={t(mp.statVisitorsSub)} />
        <Stat label={t(mp.statSignups)} value={count(n.signups)} sub={t(mp.statSignupsSub)} />
        <Stat label={t(mp.statUpgrades)} value={count(n.upgrades)} sub={t(mp.statUpgradesSub)} />
      </div>
      {list.error && !d ? (
        <section className="card flex flex-col items-center gap-3 px-6 py-10 text-center" role="alert">
          <Icon name="alertCircle" size={28} className="text-bad-fg" />
          <h2 className="m-0 text-[17px] font-semibold">{t(mp.errorTitle)}</h2>
          <span className="k max-w-[420px]">
            {errorMessage(list.error)} {t(mp.errorText)}
          </span>
          <div className="flex gap-2">
            <Button variant="primary" onClick={() => void list.refetch()}>
              {t(m.retry)}
            </Button>
            <a className="btn" href={overview.address.url} target="_blank" rel="noreferrer">
              {t(m.viewSite)}
            </a>
          </div>
          {problemReference(list.error) ? <span className="mono k">{t(m.requestRef, { ref: problemReference(list.error) })}</span> : null}
        </section>
      ) : !d ? (
        <Loading label={t(mp.tableLabel)} className="card pad flex flex-col gap-5">
          <Skeleton w={120} h={14} />
          {[0, 1, 2, 3, 4].map((i) => (
            <div key={i} className="flex items-center gap-4">
              <Skeleton w="62%" h={12} />
              <Skeleton w={84} h={20} r={999} />
              <Skeleton w={46} h={12} />
            </div>
          ))}
        </Loading>
      ) : (
        <section className="card ws-postscard" aria-labelledby="ws-posts-h">
          <div className="ws-sec">
            <div className="tx">
              <h2 className="ws-h2" id="ws-posts-h">
                {t(mp.title, { count: count(d.counts.all) })}
              </h2>
              <span className="k">
                {t(mp.summary, {
                  published: count(d.counts.published),
                  fromCampaigns: count(d.counts.fromCampaigns),
                  webOnly: count(d.counts.webOnly),
                  scheduled: count(d.counts.scheduled),
                  draft: d.counts.draft,
                  unpublished: count(d.counts.unpublished),
                })}
              </span>
            </div>
            <label className="ws-search">
              <Icon name="search" size={14} />
              <input type="search" aria-label={t(mp.searchLabel)} placeholder={t(mp.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
            </label>
          </div>
          <div className="ws-fchips">
            <div className="flex flex-wrap gap-1.5" role="group" aria-label={t(mp.filterLabel)}>
              {(
                [
                  [undefined, t(mp.all), d.counts.all],
                  ['published', t(mp.published), d.counts.published],
                  ['scheduled', t(mp.scheduled), d.counts.scheduled],
                  ['draft', t(mp.draft), d.counts.draft],
                  ['unpublished', t(mp.unpublished), d.counts.unpublished],
                ] as [PostsSearch['status'], string, number][]
              ).map(([s, label, c]) => (
                <button key={label} type="button" className="ws-fchip" aria-pressed={search.status === s} onClick={() => set({ status: s })}>
                  {label} <b>{count(c)}</b>
                </button>
              ))}
            </div>
            <div className="ws-filters">
              <Select aria-label={t(mp.visibilityLabel)} value={search.visibility ?? ''} onChange={(e) => set({ visibility: (e.target.value || undefined) as PostsSearch['visibility'] })}>
                <option value="">{t(mp.anyVisibility)}</option>
                <option value="public">{t(mp.public)}</option>
                <option value="members">{t(mp.members)}</option>
                <option value="paid">{t(mp.paid)}</option>
              </Select>
              <Select aria-label={t(mp.tagLabel)} value={search.tag ?? ''} onChange={(e) => set({ tag: e.target.value || undefined })}>
                <option value="">{t(mp.anyTag)}</option>
                {(tags.data ?? []).map((x) => (
                  <option key={x.id} value={x.slug}>
                    {x.name}
                  </option>
                ))}
              </Select>
              <Select aria-label={t(mp.sourceLabel)} value={search.source ?? ''} onChange={(e) => set({ source: (e.target.value || undefined) as PostsSearch['source'] })}>
                <option value="">{t(mp.anySource)}</option>
                <option value="campaign">{t(mp.fromCampaigns)}</option>
                <option value="web">{t(mp.webOnly)}</option>
              </Select>
            </div>
          </div>
          {d.posts.length === 0 ? (
            <div className="px-[18px] py-6">
              <NoResults
                className="border-0"
                title={filtered ? (search.q ? t(mp.noResults, { q: search.q }) : t(mp.noResultsPlain)) : t(mp.noPosts)}
                actions={
                  <>
                    {filtered ? <Button onClick={() => onSearch({})}>{t(mp.clearFilters)}</Button> : null}
                    {canEdit ? (
                      <Button variant="primary" loading={creating} onClick={() => void newPost()}>
                        {t(m.newPost)}
                      </Button>
                    ) : null}
                  </>
                }
              >
                {filtered ? t(mp.noResultsText) : t(mp.noPostsText)}
              </NoResults>
            </div>
          ) : (
            <>
              <PostsTable posts={d.posts} overview={overview} onDialog={setDialog} onState={(id) => onSearch({ ...search, post: id })} />
              <PostsList posts={d.posts} />
            </>
          )}
          <div className="ws-foot">
            <span className="k">
              {t(mp.showing, { shown: d.posts.length, total: count(d.total), from: dayMonthDate(d.from), until: dayMonthDate(d.until) })}
            </span>
            <div className="flex gap-2">
              <Button size="sm" disabled={(search.page ?? 1) <= 1} onClick={() => set({ page: (search.page ?? 1) - 1 })}>
                {t(mp.previous)}
              </Button>
              <Button size="sm" disabled={(search.page ?? 1) * PAGE >= d.total} onClick={() => set({ page: (search.page ?? 1) + 1 })}>
                {t(mp.next)}
              </Button>
            </div>
          </div>
        </section>
      )}
      {search.past ? <PastCampaignsDialog onClose={() => onSearch({ ...search, past: undefined })} /> : null}
      {open ? <PostStateDialog post={open} overview={overview} onClose={() => onSearch({ ...search, post: undefined })} onDialog={setDialog} /> : null}
      {dialog?.kind === 'unpublish' ? <UnpublishDialog post={dialog.post} onClose={() => setDialog(null)} /> : null}
      {dialog?.kind === 'delete' ? <DeletePostDialog post={dialog.post} onClose={() => setDialog(null)} /> : null}
      {dialog?.kind === 'redirect' ? <RedirectDialog post={dialog.post} onClose={() => setDialog(null)} /> : null}
      {dialog?.kind === 'time' ? <PublishTimeDialog post={dialog.post} host={overview.address.host} onClose={() => setDialog(null)} /> : null}
      {dialog?.kind === 'preview' ? <PostPreviewDialog postId={dialog.post.id} title={dialog.post.title} onClose={() => setDialog(null)} /> : null}
    </WebsitePage>
  )
}

function Stat({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="card ws-stat">
      <span className="k">{label}</span>
      <span className="v">{value}</span>
      <span className="s">{sub}</span>
    </div>
  )
}

/** A link to a post's editor: the post composer, or its campaign's composer on the Web tab. */
export function EditLink({ post: p, className, children }: { post: PostRow; className?: string; children: ReactNode }) {
  const { slug } = useWorkspace()
  if (p.kind === 'campaign' && p.campaign) {
    return (
      <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: p.campaign.id }} search={{ panel: 'web' } as never} className={className}>
        {children}
      </Link>
    )
  }
  return (
    <Link to="/w/$workspace/forms/website/posts/$post" params={{ workspace: slug, post: p.id }} className={className}>
      {children}
    </Link>
  )
}

/** Where a post's Edit goes: the post composer, or its campaign's composer on the Web tab. */
export function useEditPost() {
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  return (p: PostRow) => {
    if (p.kind === 'campaign' && p.campaign) {
      void navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: p.campaign.id }, search: { panel: 'web' } as never })
    } else {
      void navigate({ to: '/w/$workspace/forms/website/posts/$post', params: { workspace: slug, post: p.id } })
    }
  }
}

function PostsTable({ posts, overview, onDialog, onState }: { posts: PostRow[]; overview: Overview; onDialog: (d: Dialog) => void; onState: (id: string) => void }) {
  const t = useT()
  const canEdit = useCan('website:edit')
  const canPublish = useCan('website:publish')
  const showMenu = canEdit || canPublish
  const [menuFor, setMenuFor] = useState<string | null>(null)
  return (
    <div className="ws-ptable">
      <TableScroll label={t(mp.tableLabel)}>
        <table className="tbl ws-ptbl" aria-label={t(mp.tableLabel)}>
          <thead>
            <tr>
              <th scope="col">{t(mp.colPost)}</th>
              <th scope="col">{t(mp.colSource)}</th>
              <th scope="col">{t(mp.colVisibility)}</th>
              <th scope="col">{t(mp.colStatus)}</th>
              <th scope="col" className="r">
                {t(mp.colViews)}
              </th>
              <th scope="col" className="r">
                {t(mp.colSignups)}
              </th>
              <th scope="col" className="r">
                {t(mp.colUpgrades)}
              </th>
              {showMenu ? (
                <th scope="col">
                  <span className="vh">{t(mp.colActions)}</span>
                </th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {posts.map((p) => (
              <tr key={p.id} className={menuFor === p.id ? 'sel' : undefined}>
                <td>
                  <TitleCell post={p} />
                </td>
                <td>
                  <SourceCell post={p} />
                </td>
                <td>
                  <VisibilityCell post={p} />
                </td>
                <td>
                  <StatusCell post={p} onState={onState} />
                </td>
                <Num post={p} v={p.views} />
                <Num post={p} v={p.signups} />
                <Num post={p} v={p.upgrades} />
                {showMenu ? (
                  <td className="r" style={{ paddingTop: 10 }}>
                    <RowMenu post={p} overview={overview} onDialog={onDialog} open={menuFor === p.id} onOpenChange={(o) => setMenuFor(o ? p.id : null)} />
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      </TableScroll>
    </div>
  )
}

function Num({ post, v }: { post: PostRow; v: number }) {
  const t = useT()
  return <td className="r num">{post.status === 'published' ? count(v) : t(mp.dash)}</td>
}

function TitleCell({ post: p }: { post: PostRow }) {
  return (
    <div className="ws-ptitle">
      <EditLink post={p}>{p.title}</EditLink>
      {p.excerpt ? <span className="ex">{p.excerpt}</span> : null}
      {p.tags.length ? (
        <span className="ws-tags">
          {p.tags.map((x) => (
            <span key={x.id} className="ws-tagc">
              {x.name}
            </span>
          ))}
        </span>
      ) : null}
    </div>
  )
}

function SourceCell({ post: p }: { post: PostRow }) {
  const t = useT()
  const { slug } = useWorkspace()
  if (p.kind === 'campaign' && p.campaign) {
    const c = p.campaign
    return (
      <span className="ws-src">
        <Link to="/w/$workspace/campaigns/$campaignId/report" params={{ workspace: slug, campaignId: c.id }}>
          {p.status === 'unpublished' ? `${c.name} campaign` : c.name}
        </Link>
        <span className="k">{p.status === 'unpublished' && c.sentAt ? t(mp.sentOn, { date: dayMonthDate(c.sentAt) }) : t(mp.campaign)}</span>
      </span>
    )
  }
  return (
    <span className="ws-src">
      <b className="font-semibold">{t(mp.webOnly)}</b>
      <span className="k">{t(mp.byline, { name: p.byline.name || p.createdBy })}</span>
    </span>
  )
}

/** "☆ Paid · all tiers · preview of 3 paragraphs · Featured". */
export function VisibilityCell({ post: p }: { post: PostRow }) {
  const t = useT()
  if (p.status === 'unpublished') return <span className="k">{t(mp.dash)}</span>
  const tiers = p.tiers.length ? p.tiers.map((x) => x.label || x.name).join(', ') : t(mp.paidAllTiers)
  let detail: string | undefined
  if (p.visibility === 'paid') {
    detail = p.neverFree ? t(mp.paidNever, { tiers }) : p.hasMarker ? t(mp.paidPreview, { tiers, count: p.markerAfter }) : tiers
  } else if (p.visibility === 'members' && p.hasMarker) {
    detail = t(mp.membersPreview, { count: p.markerAfter })
  }
  return (
    <span className="ws-stk">
      <VisibilityLabel visibility={p.visibility} />
      {detail ? <span className="k">{detail}</span> : null}
      {p.featured ? <span className="badge sm mut">{t(mp.featured)}</span> : null}
    </span>
  )
}

export function VisibilityLabel({ visibility }: { visibility: string }) {
  const t = useT()
  const label = visibility === 'paid' ? t(mp.paid) : visibility === 'members' ? t(mp.members) : t(mp.public)
  return (
    <span className="ws-vis">
      <Icon name={visibilityIcon(visibility)} size={14} />
      {label}
    </span>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const t = useT()
  switch (status) {
    case 'published':
      return <Badge tone="ok">{t(mp.published)}</Badge>
    case 'scheduled':
      return <Badge tone="info">{t(mp.scheduled)}</Badge>
    case 'unpublished':
      return <Badge tone="neutral">{t(mp.unpublished)}</Badge>
    default:
      return <Badge tone="neutral">{t(mp.draft)}</Badge>
  }
}

/** The status line under the badge: when, by whom, waiting for whom. */
export function statusLine(t: ReturnType<typeof useT>, p: PostRow, short = false): string {
  switch (p.status) {
    case 'published':
      return p.publishMode === 'with_send' && p.kind === 'campaign' && p.publishedAt ? t(mp.withSend, { date: dateTime(p.publishedAt) }) : p.publishedAt ? dateTime(p.publishedAt) : ''
    case 'scheduled':
      if (p.publishMode === 'with_send' && p.kind === 'campaign') return p.campaign?.scheduledAt ? t(mp.withSendScheduled, { date: dateTime(p.campaign.scheduledAt) }) : t(mp.waitingShort)
      return p.publishAt ? scheduledLine(p.publishAt, p.publishTz) : ''
    case 'unpublished':
      if (short) return t(mp.unpublishedShort, { name: p.unpublishedBy ?? '', date: p.unpublishedAt ? dayMonthDate(p.unpublishedAt) : '' })
      return p.unpublishReason
        ? t(mp.unpublishedLine, { date: p.unpublishedAt ? dayMonthDate(p.unpublishedAt) : '', name: p.unpublishedBy ?? '', reason: p.unpublishReason, views: p.viewsAllTime })
        : t(mp.unpublishedNoReason, { date: p.unpublishedAt ? dayMonthDate(p.unpublishedAt) : '', name: p.unpublishedBy ?? '', views: p.viewsAllTime })
    default:
      if (p.approval?.status === 'pending') return short ? t(mp.waitingShort) : t(mp.waitingFor, { approvers: orList(p.approval.approvers) })
      if (p.approval?.status === 'changes_requested') return t(mp.changesRequested)
      return t(mp.draftLine)
  }
}

/** A post with more to say (waiting for approval, unpublished, scheduled) opens its card. */
function hasState(p: PostRow): boolean {
  return p.status === 'unpublished' || p.status === 'scheduled' || (p.status === 'draft' && p.approval?.status === 'pending')
}

function StatusCell({ post: p, onState }: { post: PostRow; onState: (id: string) => void }) {
  const t = useT()
  const line = statusLine(t, p)
  const body = (
    <span className="ws-stk">
      <StatusBadge status={p.status} />
      {line ? <span className="k">{line}</span> : null}
    </span>
  )
  if (!hasState(p)) return body
  return (
    <button type="button" className="ws-stbtn" onClick={() => onState(p.id)}>
      <span className="vh">{t(mp.detailsFor, { title: p.title })}</span>
      {body}
    </button>
  )
}

function RowMenu({ post: p, overview, onDialog, open, onOpenChange }: { post: PostRow; overview: Overview; onDialog: (d: Dialog) => void; open: boolean; onOpenChange: (o: boolean) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const invalidate = useInvalidateWebsite()
  const edit = useEditPost()
  const canEdit = useCan('website:edit')
  const canPublish = useCan('website:publish')
  const host = overview.address.host
  const copy = async () => {
    if (await copyText(p.url)) toast({ kind: 'success', title: t(mp.linkCopied), detail: bare(p.url) })
    else toast({ kind: 'error', title: t(mp.copyFailed), detail: p.url })
  }
  const publishNow = async () => {
    try {
      await http.POST(`${WS}/posts/{post}/publish`, { params: { path: { workspace: slug, post: p.id } }, body: { when: 'now' } })
      toast({ kind: 'success', title: t(mp.publishedToast, { title: p.title, host }) })
      void invalidate()
    } catch (e) {
      toast({ kind: 'error', title: t(mp.publishFailed), detail: errorMessage(e) })
    }
  }
  const sendAsCampaign = async () => {
    try {
      const { data } = await http.POST(`${WS}/posts/{post}/send-as-campaign`, { params: { path: { workspace: slug, post: p.id } } })
      if (data) void navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: data.campaignId } })
    } catch (e) {
      toast({ kind: 'error', title: t(mp.sendAsFailed), detail: errorMessage(e) })
    }
  }
  const items: ReactNode[] = []
  const sep = (k: string) => items.push(<MenuSeparator key={k} />)
  if (canEdit) items.push(item('edit', 'edit', t(mp.edit), () => edit(p)))
  if (p.status === 'published') {
    items.push(
      <MenuItem key="view" asChild>
        <a href={p.url} target="_blank" rel="noreferrer">
          <Icon name="eye" size={14} />
          {t(mp.viewOn, { host })}
        </a>
      </MenuItem>,
    )
    items.push(item('copy', 'link', t(mp.copyLink), () => void copy()))
  }
  if (p.status === 'draft') items.push(item('preview', 'eye', t(mp.preview), () => onDialog({ kind: 'preview', post: p })))
  if (canPublish) {
    if (p.status === 'scheduled') {
      items.push(item('now', 'send', t(mp.publishNow), () => void publishNow()))
      if (p.publishMode !== 'with_send') items.push(item('time', 'clock', t(mp.changeTime), () => onDialog({ kind: 'time', post: p })))
    }
    if (p.status === 'unpublished') {
      items.push(item('again', 'send', t(mp.publishAgain), () => void publishNow()))
      items.push(item('redirect', 'arrowRight', t(mp.redirect), () => onDialog({ kind: 'redirect', post: p })))
    }
    if (p.status === 'published') {
      sep('s1')
      items.push(item('unpublish', <EyeOff />, t(mp.unpublish), () => onDialog({ kind: 'unpublish', post: p })))
    }
  }
  if (canEdit && p.kind === 'web' && p.status === 'published') items.push(item('send', 'send', t(mp.sendAsCampaign), () => void sendAsCampaign()))
  if (canEdit && p.kind === 'web') {
    sep('s2')
    items.push(item('delete', 'trash', t(mp.delete), () => onDialog({ kind: 'delete', post: p }), 'bad'))
  }
  if (!items.length) return null
  return (
    <Menu
      open={open}
      onOpenChange={onOpenChange}
      label={t(mp.actionsFor, { title: p.title })}
      className="ws-menu"
      trigger={
        <button type="button" className="btn sm ic ws-menu-ic" aria-label={t(mp.actionsFor, { title: p.title })}>
          <Icon name="more" size={15} />
        </button>
      }
    >
      {items}
    </Menu>
  )
}

function item(key: string, icon: Parameters<typeof Icon>[0]['name'] | ReactNode, label: string, onSelect: () => void, tone?: 'bad') {
  return (
    <MenuItem key={key} onSelect={onSelect} className={tone}>
      {typeof icon === 'string' ? <Icon name={icon as Parameters<typeof Icon>[0]['name']} size={14} /> : icon}
      {label}
    </MenuItem>
  )
}

/** The unpublish icon: an eye struck through. */
export function EyeOff() {
  return (
    <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 2l12 12M6.6 6.7a2 2 0 0 0 2.7 2.7M4.4 4.6C2.9 5.6 2 7 1.5 8c1.2 2.6 3.6 4.5 6.5 4.5 1.2 0 2.3-.3 3.3-.9M7 3.6c.3 0 .7-.1 1-.1 2.9 0 5.3 1.9 6.5 4.5-.4.8-.9 1.6-1.6 2.3" />
    </svg>
  )
}

/** Under 1100 px the table becomes a list (WebsitePostsTablet, WebsitePostsPhone). */
function PostsList({ posts }: { posts: PostRow[] }) {
  const t = useT()
  return (
    <div className="ws-plist">
      {posts.map((p) => (
        <EditLink key={p.id} className="pi" post={p}>
          <span className="pt">
            <b>{p.title}</b>
            <StatusBadge status={p.status} />
          </span>
          <span className="pd">{statusLine(t, p, true)}</span>
          <span className="pm">
            {p.status === 'unpublished' ? null : <VisibilityLabel visibility={p.visibility} />}
            <span>{p.kind === 'campaign' && p.campaign ? (p.status === 'unpublished' ? `${p.campaign.name} campaign` : p.campaign.name) : t(mp.webOnlyBy, { name: p.byline.name || p.createdBy })}</span>
            {p.status === 'published' ? (
              <span>
                <span className="num">{count(p.views)}</span> {t(mp.numViews)} · <span className="num">{count(p.signups)}</span> {t(mp.numSignups)} · <span className="num">{count(p.upgrades)}</span> {t(mp.numUpgrades)}
              </span>
            ) : null}
          </span>
        </EditLink>
      ))}
    </div>
  )
}
