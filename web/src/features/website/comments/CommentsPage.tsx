import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useEffect, useState, type ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { problemReference } from '@/components/shell/SystemScreens'
import { Icon } from '@/components/icons/Icon'
import { Avatar, type AvatarColor } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { EmptyState, ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Menu, MenuItem, MenuSeparator } from '@/components/ui/Menu'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { clock, count, dayMonthDate, orList } from '@/lib/format'
import { useMediaQuery } from '@/lib/hooks'
import { useCan, useMe, useWorkspace } from '@/lib/session'
import { bare, siteURL, WS, type Overview } from '../api'
import { WebsitePage } from '../WebsiteFrame'
import { mc } from './messages'
import './comments.css'

type Comment = Schemas['WebsiteComment']
type List = Schemas['WebsiteCommentList']
export type CommentsSearch = { status?: 'pending' | 'published' | 'hidden'; post?: string; q?: string }

const LIST = `${WS}/comments` as const
const PAGE = 20
const tones: AvatarColor[] = ['purple', 'ok', 'sky', 'pink', 'info', 'warn']

/** A steady avatar colour per name. */
function toneOf(name: string): AvatarColor {
  let h = 0
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return tones[h % tones.length]!
}

/** "15 Oct, 10:02". */
const when = (d: string) => `${dayMonthDate(d)}, ${clock(d)}`

/** Splits a comment's text into words and links, the links clickable (rel ugc). */
function Body({ text }: { text: string }) {
  const parts = text.split(/(https?:\/\/[^\s<>"]+[^\s<>".,;:!?)])/g)
  return (
    <p className="body">
      {parts.map((p, i) =>
        i % 2 === 1 ? (
          <a key={i} href={p} rel="nofollow ugc noopener noreferrer" target="_blank">
            {p}
          </a>
        ) : (
          <span key={i}>{p}</span>
        ),
      )}
    </p>
  )
}

/**
 * Website › Comments: the moderation queue. Waiting for review, Published
 * and Hidden; publish, hide or delete several at once with Undo for 10
 * seconds, pin, reply as the publication and block commenters.
 */
export function CommentsPage({ overview, search, onSearch }: { overview: Overview; search: CommentsSearch; onSearch: (s: CommentsSearch, replace?: boolean) => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const user = useMe().data?.user
  const qc = useQueryClient()
  const canModerate = useCan('comments:moderate')
  const phone = useMediaQuery('(max-width: 767px)')
  const status = search.status ?? 'pending'
  const [cursors, setCursors] = useState<string[]>([])
  const [q, setQ] = useState(search.q ?? '')
  const [selected, setSelected] = useState<string[]>([])
  const [replying, setReplying] = useState<string | null>(null)
  const [dialog, setDialog] = useState<{ kind: 'delete'; ids: string[] } | { kind: 'block'; comment: Comment } | null>(null)
  const filterKey = `${status}|${search.post ?? ''}|${search.q ?? ''}`
  const [shownKey, setShownKey] = useState(filterKey)
  if (shownKey !== filterKey) {
    setShownKey(filterKey)
    setCursors([])
    setSelected([])
    setReplying(null)
  }
  useEffect(() => {
    const h = window.setTimeout(() => {
      if ((search.q ?? '') !== q) onSearch({ ...search, q: q || undefined }, true)
    }, 300)
    return () => window.clearTimeout(h)
  }, [q, search, onSearch])
  const cursor = cursors[cursors.length - 1]
  const list = $api.useQuery(
    'get',
    LIST,
    { params: { path: { workspace: slug }, query: { status, post: search.post, q: search.q, cursor, limit: PAGE } } },
    { placeholderData: (p) => p },
  )
  const d = list.data
  const refresh = () => qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && (x.queryKey[1].includes('/website/comments') || x.queryKey[1].endsWith('/website/posts')) })
  const host = bare(overview.address.url)

  const decide = async (ids: string[], decision: 'publish' | 'hide' | 'delete') => {
    try {
      const { data } = await http.POST(`${LIST}/decide`, { params: { path: { workspace: slug } }, body: { ids, decision } })
      setSelected([])
      await refresh()
      if (!data) return
      const n = data.count
      const title = decision === 'publish' ? t(mc.published, { count: n }) : decision === 'hide' ? t(mc.hidden, { count: n }) : t(mc.deleted, { count: n })
      toast({
        kind: decision === 'delete' ? 'deleted' : 'success',
        title,
        detail: decision === 'hide' ? t(mc.hiddenDetail) : undefined,
        undo: () => void undo(data.decisionId),
      })
    } catch (e) {
      toast({ kind: 'error', title: t(mc.decideFailed), detail: errorMessage(e) })
    }
  }
  const undo = async (id: string) => {
    try {
      await http.POST(`${LIST}/decisions/{decision}/undo`, { params: { path: { workspace: slug, decision: id } } })
      await refresh()
      toast({ title: t(mc.undone) })
    } catch (e) {
      toast({ kind: 'error', title: t(mc.undoFailed), detail: errorMessage(e) })
    }
  }
  const pin = async (c: Comment) => {
    try {
      if (c.pinned) await http.POST(`${LIST}/{comment}/unpin`, { params: { path: { workspace: slug, comment: c.id } } })
      else await http.POST(`${LIST}/{comment}/pin`, { params: { path: { workspace: slug, comment: c.id } } })
      await refresh()
      toast({ title: c.pinned ? t(mc.unpinnedDone) : t(mc.pinnedDone) })
    } catch (e) {
      toast({ kind: 'error', title: t(mc.pinFailed), detail: errorMessage(e) })
    }
  }
  const unblock = async (c: Comment) => {
    if (!c.contactId) return
    try {
      await http.POST(`${WS}/members/{contact}/comments-unblock`, { params: { path: { workspace: slug, contact: c.contactId } } })
      await refresh()
      toast({ title: t(mc.unblockedDone, { name: c.name }) })
    } catch (e) {
      toast({ kind: 'error', title: t(mc.blockFailed), detail: errorMessage(e) })
    }
  }

  const tabs = (
    <nav className="wc-tabs" aria-label={t(mc.statusLabel)} data-scroll-row={phone ? '' : undefined}>
      {(
        [
          ['pending', t(mc.tabPending), d?.counts.pending],
          ['published', t(mc.tabPublished), d?.counts.published],
          ['hidden', t(mc.tabHidden), d?.counts.hidden],
        ] as const
      ).map(([key, label, n]) => (
        <Link
          key={key}
          to="/w/$workspace/forms/website/comments"
          params={{ workspace: slug }}
          search={{ ...search, status: key === 'pending' ? undefined : key }}
          className={status === key ? 'ws-fchip on' : 'ws-fchip'}
          aria-current={status === key ? 'page' : undefined}
        >
          {label} {n !== undefined ? <b>{count(n)}</b> : null}
        </Link>
      ))}
    </nav>
  )

  let content: ReactNode
  if (list.isError && !d) {
    content = (
      <div className="pad">
        <ErrorPanel
          title={t(mc.loadFailed)}
          actions={
            <>
              <Button variant="primary" onClick={() => void list.refetch()}>
                {t(mc.retry)}
              </Button>
              <ButtonAnchor href={overview.address.url}>{t(mc.viewSite)}</ButtonAnchor>
            </>
          }
        >
          {errorMessage(list.error)} {t(mc.loadFailedText, { host })}
          {problemReference(list.error) ? <span className="wc-mono mt-2 block">{problemReference(list.error)}</span> : null}
        </ErrorPanel>
      </div>
    )
  } else if (!d) {
    content = (
      <Loading label={t(mc.loading)} className="wc-list">
        {[0, 1, 2].map((i) => (
          <div key={i} className="wc-row">
            <Skeleton w={32} h={32} r={999} />
            <div className="tx">
              <Skeleton w={220} />
              <Skeleton w={300} />
              <Skeleton w="90%" h={14} />
            </div>
          </div>
        ))}
      </Loading>
    )
  } else if (d.items.length === 0) {
    content = <Empty list={d} search={search} status={status} host={host} overview={overview} onClear={() => { setQ(''); onSearch({ status: search.status }) }} />
  } else {
    const items = d.items
    const allSel = canModerate && items.every((c) => selected.includes(c.id))
    content = (
      <>
        {canModerate && selected.length ? (
          <div className="wc-dbar" role="region" aria-label={t(mc.decisionsLabel)}>
            <div className="dtop">
              <label className="flex items-center gap-2 font-semibold">
                <input type="checkbox" checked={allSel} aria-label={t(mc.selectAll)} onChange={(e) => setSelected(e.target.checked ? items.map((c) => c.id) : [])} />
                {t(mc.selected, { count: selected.length })}
              </label>
              <div className="grp flex flex-wrap gap-2">
                {status !== 'published' ? (
                  <Button variant="primary" icon="check" onClick={() => void decide(selected, 'publish')}>
                    {t(mc.publish)}
                  </Button>
                ) : null}
                {status !== 'hidden' ? (
                  <Button icon="eye" onClick={() => void decide(selected, 'hide')}>
                    {t(mc.hide)}
                  </Button>
                ) : null}
                <Button variant="danger" icon="trash" onClick={() => setDialog({ kind: 'delete', ids: selected })}>
                  {t(mc.delete)}
                </Button>
              </div>
              <Button className="ml-auto" size="sm" onClick={() => setSelected([])}>
                {t(mc.clearSelection)}
              </Button>
            </div>
          </div>
        ) : null}
        <div className="wc-list" role="list" aria-label={t(mc.listLabel)}>
          {items.map((c) => (
            <Row
              key={c.id}
              c={c}
              list={d}
              items={items}
              selected={selected.includes(c.id)}
              onSelect={(on) => setSelected((s) => (on ? [...s, c.id] : s.filter((x) => x !== c.id)))}
              replying={replying === c.id}
              onReply={(on) => setReplying(on ? c.id : null)}
              onReplied={() => {
                setReplying(null)
                void refresh()
              }}
              onPin={() => void pin(c)}
              onBlock={() => setDialog({ kind: 'block', comment: c })}
              onUnblock={() => void unblock(c)}
            />
          ))}
        </div>
        <div className="wc-foot">
          <span className="k">
            {!canModerate
              ? t(mc.footRead)
              : status === 'pending'
                ? t(mc.footPending, { count: d.counts.pending })
                : status === 'published'
                  ? t(mc.footPublished, { shown: items.length, count: d.counts.published })
                  : t(mc.footHidden, { count: d.counts.hidden })}
          </span>
          <div className="flex gap-2">
            <Button size="sm" disabled={cursors.length === 0} onClick={() => setCursors((c) => c.slice(0, -1))}>
              {t(mc.previous)}
            </Button>
            <Button size="sm" disabled={!d.nextCursor} onClick={() => d.nextCursor && setCursors((c) => [...c, d.nextCursor!])}>
              {t(mc.next)}
            </Button>
          </div>
        </div>
      </>
    )
  }

  const post = search.post ? d?.posts.find((p) => p.id === search.post) : undefined
  const total = d ? d.counts.pending + d.counts.published + d.counts.hidden : 0
  const off = d?.comments === 'off'
  return (
    <WebsitePage view="comments" overview={overview}>
      {!canModerate ? (
        <section className="ws-ro" role="status">
          <div className="hd">
            <Icon name="info" size={15} />
            {t(mc.view)} · {user?.name ?? ''}, {me.workspace.roleName}
          </div>
          <div className="bd">
            {t(mc.readOnly)}
            {overview.helpers.manage.length ? ` ${orList(overview.helpers.manage)}` : null}
          </div>
        </section>
      ) : null}
      {off ? (
        <section className="wc-off" aria-labelledby="wc-off-h">
          <h3 id="wc-off-h">
            <Icon name="mail" size={18} />
            {t(mc.offTitle)}
          </h3>
          <span>{t(mc.offText)}</span>
          <div>
            <ButtonLink variant="primary" to="/w/$workspace/forms/website/access" params={{ workspace: slug }} hash="comments">
              {t(mc.offAction)}
            </ButtonLink>
          </div>
        </section>
      ) : null}
      <section className="card" aria-labelledby="wc-h">
        <div className="sec">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h3 className="h2" id="wc-h">
              {post ? t(mc.titlePost, { title: post.title }) : d && total ? t(mc.title, { count: total }) : t(mc.titleNone)}
            </h3>
            {d ? <span className="k">{post ? t(mc.summaryPost) : t(mc.summary, { posts: d.posts.length, who: d.comments, review: d.review })}</span> : <Skeleton w={360} />}
          </div>
          <ButtonLink size="sm" to="/w/$workspace/forms/website/access" params={{ workspace: slug }} hash="comments">
            {t(mc.settings)}
          </ButtonLink>
        </div>
        {tabs}
        <div className="wc-filters">
          <label className="ws-search">
            <Icon name="search" size={14} />
            <input type="search" aria-label={t(mc.searchLabel)} placeholder={t(mc.searchPlaceholder)} value={q} onChange={(e) => setQ(e.target.value)} />
          </label>
          <select className="in" aria-label={t(mc.postLabel)} value={search.post ?? ''} onChange={(e) => onSearch({ ...search, post: e.target.value || undefined })}>
            <option value="">{t(mc.allPosts)}</option>
            {(d?.posts ?? []).map((p) => (
              <option key={p.id} value={p.id}>
                {p.title}
              </option>
            ))}
          </select>
        </div>
        {content}
      </section>
      {dialog?.kind === 'delete' ? (
        <DeleteDialog
          ids={dialog.ids}
          items={d?.items ?? []}
          onClose={() => setDialog(null)}
          onDelete={() => {
            setDialog(null)
            void decide(dialog.ids, 'delete')
          }}
          onHide={() => {
            setDialog(null)
            void decide(dialog.ids, 'hide')
          }}
        />
      ) : null}
      {dialog?.kind === 'block' ? <BlockDialog c={dialog.comment} host={host} onClose={() => setDialog(null)} onDone={() => void refresh()} /> : null}
    </WebsitePage>
  )
}

function Row({
  c,
  list,
  items,
  selected,
  onSelect,
  replying,
  onReply,
  onReplied,
  onPin,
  onBlock,
  onUnblock,
}: {
  c: Comment
  list: List
  items: Comment[]
  selected: boolean
  onSelect: (on: boolean) => void
  replying: boolean
  onReply: (on: boolean) => void
  onReplied: () => void
  onPin: () => void
  onBlock: () => void
  onUnblock: () => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const canModerate = useCan('comments:moderate')
  const [menu, setMenu] = useState(false)
  const name = c.team ? list.publication : c.name
  const firstReply = items.find((x) => x.parentId === c.id)
  const id = `wc-${c.id}`
  return (
    <div className={selected || menu ? 'wc-row sel' : 'wc-row'} role="listitem" aria-labelledby={id}>
      {canModerate ? (
        <label className="wc-cbt">
          <input type="checkbox" checked={selected} aria-label={t(mc.selectOne, { name })} onChange={(e) => onSelect(e.target.checked)} />
        </label>
      ) : null}
      {c.team ? (
        <span className="wc-team" style={{ width: 32, height: 32, fontSize: 12 }} aria-hidden="true">
          {list.publication.slice(0, 1).toUpperCase()}
        </span>
      ) : (
        <Avatar name={c.name} color={toneOf(c.name)} size={32} fontSize={12} />
      )}
      <div className="tx">
        <div className="who">
          <b id={id}>{name}</b>
          {c.team ? <Badge size="sm">{t(mc.team)}</Badge> : null}
          {c.email ? <span className="k wc-mono">{c.email}</span> : null}
          {c.team && c.by ? <span className="k">{t(mc.byMember, { name: c.by, date: when(c.createdAt) })}</span> : <span className="k">{when(c.createdAt)}</span>}
          {c.editedAt ? <span className="k">· {t(mc.edited)}</span> : null}
        </div>
        <span className="on-post">
          <T msg={mc.on} values={{ title: c.postTitle, link: (x: ReactNode) => <a href={c.url} target="_blank" rel="noopener noreferrer">{x}</a> }} />
          {c.parentName ? ` · ${t(mc.replyTo, { name: c.parentName })}` : null}
        </span>
        <Body text={c.body} />
        <Meta c={c} />
        {firstReply && c.status === 'published' ? (
          <div className="wc-thread">
            <div className="tr">
              {firstReply.team ? (
                <span className="wc-team" aria-hidden="true">
                  {list.publication.slice(0, 1).toUpperCase()}
                </span>
              ) : (
                <Avatar name={firstReply.name} color={toneOf(firstReply.name)} size={22} />
              )}
              <span>
                <b className="font-semibold">{firstReply.team ? list.publication : firstReply.name}</b> {firstReply.team ? <Badge size="sm">{t(mc.team)}</Badge> : null}{' '}
                <span className="k">{firstReply.team && firstReply.by ? t(mc.byMember, { name: firstReply.by, date: when(firstReply.createdAt) }) : when(firstReply.createdAt)}</span>
                <br />
                {firstReply.body.length > 120 ? `${firstReply.body.slice(0, 120).trimEnd()}…` : firstReply.body}
              </span>
            </div>
            {c.replies > 1 ? (
              <div className="tr">
                <span className="k">{t(mc.moreReplies, { count: c.replies - 1 })}</span>
              </div>
            ) : null}
          </div>
        ) : null}
        {replying ? <ReplyBox c={c} publication={list.publication} onCancel={() => onReply(false)} onDone={onReplied} /> : null}
      </div>
      {canModerate ? (
        <div className="acts">
          <Menu
            label={t(mc.more, { name })}
            open={menu}
            onOpenChange={setMenu}
            className="min-w-[250px]"
            trigger={<Button size="sm" icon="more" iconOnly aria-label={t(mc.more, { name })} />}
          >
            {c.status === 'published' && !c.parentId ? (
              <MenuItem onSelect={onPin}>
                <Icon name="tag" size={14} />
                {c.pinned ? t(mc.unpin) : t(mc.pin)}
              </MenuItem>
            ) : null}
            {c.status === 'published' && !c.team ? (
              <MenuItem onSelect={() => onReply(true)}>
                <Icon name="undo" size={14} />
                {t(mc.replyAs, { publication: list.publication })}
              </MenuItem>
            ) : null}
            {c.status === 'published' && !c.team ? <MenuSeparator /> : null}
            {c.contactId && !c.team ? (
              c.blocked ? (
                <MenuItem onSelect={onUnblock}>
                  <Icon name="ban" size={14} />
                  {t(mc.unblock)}
                </MenuItem>
              ) : (
                <MenuItem onSelect={onBlock}>
                  <Icon name="ban" size={14} />
                  {t(mc.block)}
                </MenuItem>
              )
            ) : null}
            {c.contactId ? (
              <MenuItem asChild>
                <Link to="/w/$workspace/audience/contacts/$contactId" params={{ workspace: slug, contactId: c.contactId }}>
                  <Icon name="audience" size={14} />
                  {t(mc.openContact)}
                </Link>
              </MenuItem>
            ) : null}
          </Menu>
        </div>
      ) : null}
    </div>
  )
}

function Meta({ c }: { c: Comment }) {
  const t = useT()
  const bits: ReactNode[] = []
  if (c.status === 'pending' && c.heldReason) {
    const [label, text, tone] =
      c.heldReason === 'first_comment'
        ? [t(mc.heldFirst), t(mc.heldFirstText), 'info' as const]
        : c.heldReason === 'links'
          ? [t(mc.heldLinks), t(mc.heldLinksText, { count: c.links }), 'warn' as const]
          : c.heldReason === 'blocked'
            ? [t(mc.heldBlocked), t(mc.heldBlockedText), 'bad' as const]
            : [t(mc.heldAll), t(mc.heldAllText), 'info' as const]
    bits.push(
      <Badge key="held" tone={tone}>
        {label}
      </Badge>,
      <span key="held-t" className="k">
        {text}
      </span>,
    )
  }
  if (c.status === 'hidden') {
    bits.push(
      <Badge key="hidden" icon="eye">
        {t(mc.tabHidden)}
      </Badge>,
      c.moderatedAt ? (
        <span key="hidden-t" className="k">
          {c.moderatedBy ? t(mc.hiddenBy, { name: c.moderatedBy, date: when(c.moderatedAt) }) : t(mc.hiddenByNone, { date: when(c.moderatedAt) })}
        </span>
      ) : null,
    )
  }
  if (c.pinned) {
    bits.push(
      <Badge key="pin" icon="tag">
        {t(mc.pinned)}
      </Badge>,
      <span key="pin-t" className="k">
        {t(mc.pinnedLine, { replies: c.replies })}
      </span>,
    )
  } else if (c.status === 'published' && c.replies > 0) {
    bits.push(
      <span key="rep" className="k">
        {t(mc.repliesLine, { replies: c.replies })}
      </span>,
    )
  }
  if (c.blocked && c.heldReason !== 'blocked') {
    bits.push(
      <Badge key="blk" tone="bad">
        {t(mc.blocked)}
      </Badge>,
    )
  }
  return bits.length ? <div className="meta">{bits}</div> : null
}

function ReplyBox({ c, publication, onCancel, onDone }: { c: Comment; publication: string; onCancel: () => void; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const [body, setBody] = useState('')
  const [busy, setBusy] = useState(false)
  const id = `wc-reply-${c.id}`
  const send = async () => {
    setBusy(true)
    try {
      await http.POST(`${LIST}/{comment}/reply`, { params: { path: { workspace: slug, comment: c.id } }, body: { body } })
      toast({ title: t(mc.replied) })
      onDone()
    } catch (e) {
      toast({ kind: 'error', title: t(mc.replyFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="wc-rbox">
      <label className="lbl" htmlFor={id}>
        {t(mc.replyLabel, { publication })}
        <textarea id={id} className="in" maxLength={5000} value={body} onChange={(e) => setBody(e.target.value)} autoFocus />
      </label>
      <div className="rrow">
        <span className="k">{t(mc.replyNote, { name: c.name })}</span>
        <div className="flex gap-2">
          <Button size="sm" onClick={onCancel}>
            {t(mc.cancel)}
          </Button>
          <Button size="sm" variant="primary" loading={busy} disabled={!body.trim()} onClick={() => void send()}>
            {t(mc.postReply)}
          </Button>
        </div>
      </div>
    </div>
  )
}

function Empty({ list, search, status, host, overview, onClear }: { list: List; search: CommentsSearch; status: string; host: string; overview: Overview; onClear: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const post = search.post ? list.posts.find((p) => p.id === search.post) : undefined
  if (search.q) {
    return (
      <div className="pad">
        <EmptyState
          title={post ? t(mc.noMatchTitlePost, { q: search.q, title: post.title }) : t(mc.noMatchTitle, { q: search.q })}
          action={<Button onClick={onClear}>{t(mc.clearSearch)}</Button>}
        >
          {t(mc.noMatchText)}
        </EmptyState>
      </div>
    )
  }
  const to = '/w/$workspace/forms/website/comments' as const
  return (
    <div className="pad">
      {status === 'pending' ? (
        <EmptyState
          title={t(mc.emptyPendingTitle)}
          action={
            <ButtonLink to={to} params={{ workspace: slug }} search={{ ...search, status: 'published' }}>
              {t(mc.emptyPendingAction)}
            </ButtonLink>
          }
        >
          {t(mc.emptyPendingText)}
        </EmptyState>
      ) : status === 'published' ? (
        <EmptyState title={t(mc.emptyPublishedTitle)} action={<ButtonAnchor href={siteURL(overview.address)}>{t(mc.emptyPublishedAction, { host })}</ButtonAnchor>}>
          {t(mc.emptyPublishedText)}
        </EmptyState>
      ) : (
        <EmptyState
          title={t(mc.emptyHiddenTitle)}
          action={
            <ButtonLink to={to} params={{ workspace: slug }} search={{ ...search, status: undefined }}>
              {t(mc.emptyHiddenAction)}
            </ButtonLink>
          }
        >
          {t(mc.emptyHiddenText)}
        </EmptyState>
      )}
    </div>
  )
}

function DeleteDialog({ ids, items, onClose, onDelete, onHide }: { ids: string[]; items: Comment[]; onClose: () => void; onDelete: () => void; onHide: () => void }) {
  const t = useT()
  const one = ids.length === 1 ? items.find((c) => c.id === ids[0]) : undefined
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={one ? t(mc.deleteEyebrow, { count: 1, name: one.name, title: one.postTitle }) : t(mc.deleteEyebrowMany, { count: ids.length })}
      title={t(mc.deleteTitle, { count: ids.length })}
      footer={
        <>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <span className="ml-auto flex gap-2">
            <Button onClick={onHide}>{t(mc.hideInstead)}</Button>
            <Button variant="danger" onClick={onDelete}>
              {ids.length === 1 ? t(mc.deleteConfirm) : t(mc.deleteConfirmMany)}
            </Button>
          </span>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1.5 pl-5">
        <li>{t(mc.deleteGone)}</li>
        <li>{t(mc.deleteReplies)}</li>
        <li>{t(mc.deleteUndo)}</li>
      </ul>
      <div className="note n mt-3" role="note">
        <Icon name="info" size={15} />
        <div>{t(mc.deleteSafer)}</div>
      </div>
    </Dialog>
  )
}

function BlockDialog({ c, host, onClose, onDone }: { c: Comment; host: string; onClose: () => void; onDone: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const [busy, setBusy] = useState(false)
  const block = async () => {
    if (!c.contactId) return
    setBusy(true)
    try {
      await http.POST(`${WS}/members/{contact}/comments-block`, { params: { path: { workspace: slug, contact: c.contactId } } })
      toast({ title: t(mc.blockedDone, { name: c.name }) })
      onDone()
      onClose()
      void navigate({ to: '.', replace: true })
    } catch (e) {
      toast({ kind: 'error', title: t(mc.blockFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={c.email ? t(mc.blockEyebrow, { email: c.email }) : undefined}
      title={t(mc.blockTitle, { name: c.name })}
      footer={
        <>
          <Button onClick={onClose}>{t(mc.cancel)}</Button>
          <Button variant="danger" loading={busy} onClick={() => void block()}>
            {t(mc.blockConfirm)}
          </Button>
        </>
      }
    >
      <ul className="m-0 flex list-disc flex-col gap-1.5 pl-5">
        <li>{t(mc.blockForm, { host })}</li>
        <li>{t(mc.blockHidden)}</li>
        <li>{t(mc.blockElse)}</li>
      </ul>
      <p className="k mt-3 mb-0">{t(mc.blockUndo)}</p>
    </Dialog>
  )
}
