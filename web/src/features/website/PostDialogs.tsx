import { useQuery } from '@tanstack/react-query'
import { useId, useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Avatar } from '@/components/ui/Avatar'
import { Button, ButtonLink } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Segmented } from '@/components/ui/Tabs'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, clock, dayMonthDate, orList } from '@/lib/format'
import { useCan, useWorkspace } from '@/lib/session'
import { useInvalidateWebsite, WS, type Overview, type PostRow } from './api'
import { mp } from './posts.messages'
import { isoToZonedLocal, scheduledLine, zonedToISO } from './postFormat'
import { StatusBadge, VisibilityLabel } from './PostsPage'
import { SiteFrame } from './SiteFrame'

/** "/posts/feature-flags-preview": a post's path on the site. */
export function postPath(p: { slug: string }): string {
  return `/posts/${p.slug}`
}

/**
 * A post's card (WebsiteStates · posts): waiting for approval with who can
 * approve it, unpublished with its reason, or scheduled with when it goes
 * live, and the actions that move it on.
 */
export function PostStateDialog({ post: p, overview, onClose, onDialog }: { post: PostRow; overview: Overview; onClose: () => void; onDialog: (d: { kind: 'unpublish' | 'delete' | 'redirect' | 'time' | 'preview'; post: PostRow }) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const canPublish = useCan('website:publish')
  const canApprove = useCan('campaigns:approve')
  const [busy, setBusy] = useState(false)
  const code = (x: string) => <code className="mono">{x}</code>
  const publishNow = async () => {
    setBusy(true)
    try {
      await http.POST(`${WS}/posts/{post}/publish`, { params: { path: { workspace: slug, post: p.id } }, body: { when: 'now' } })
      toast({ kind: 'success', title: t(mp.publishedToast, { title: p.title, host: overview.address.host }) })
      void invalidate()
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(mp.publishFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  let sub = ''
  let rows: [string, React.ReactNode][] = []
  let note: React.ReactNode = null
  let footer: React.ReactNode = null
  if (p.status === 'draft' && p.approval) {
    const a = p.approval
    sub = t(mp.waitingSince, { date: `${dayMonthDate(a.requestedAt)}, ${clock(a.requestedAt)}` })
    rows = [
      [
        t(mp.requestedBy),
        <span className="inline-flex items-center gap-2" key="by">
          <Avatar name={a.requestedBy} size={22} color="warn" />
          {a.requestedBy}
        </span>,
      ],
      [t(mp.rule), a.rule === 'post_publish' ? t(mp.rulePostPublish) : <T msg={mp.ruleAuthorLacksSend} values={{ perm: code('website:publish') }} />],
      [t(mp.canApprove), <T key="ca" msg={mp.canApproveWho} values={{ approvers: orList(a.approvers), perm: <span className="k">campaigns:approve</span> }} />],
      [t(mp.covers), t(mp.coversText, { path: postPath(p), visibility: visName(t, p.visibility) })],
    ]
    if (canPublish && !canApprove) {
      note = (
        <Note tone="neutral">
          <T msg={mp.cantApprove} values={{ perm: code('campaigns:approve') }} />
        </Note>
      )
    }
    footer = (
      <>
        <Button onClick={() => onDialog({ kind: 'preview', post: p })}>{t(mp.preview)}</Button>
        <ButtonLink variant="primary" to="/w/$workspace/approvals/$approvalId" params={{ workspace: slug, approvalId: a.id }}>
          {t(mp.openInbox)}
        </ButtonLink>
      </>
    )
  } else if (p.status === 'unpublished') {
    sub = p.unpublishedAt ? `${dayMonthDate(p.unpublishedAt)}, ${clock(p.unpublishedAt)}${p.unpublishedBy ? ` · ${p.unpublishedBy}` : ''}` : ''
    rows = [
      ...(p.unpublishReason ? ([[t(mp.reason), `“${p.unpublishReason}”`]] as [string, React.ReactNode][]) : []),
      [
        t(mp.before),
        p.campaign?.sentAt
          ? t(mp.beforeText, { views: p.viewsAllTime, campaign: p.campaign.name, date: dayMonthDate(p.campaign.sentAt) })
          : t(mp.beforeTextWeb, { views: p.viewsAllTime }),
      ],
      [
        t(mp.itsAddress),
        p.redirectTo ? (
          <T key="ad" msg={mp.answersRedirect} values={{ path: code(postPath(p)), to: code(p.redirectTo) }} />
        ) : (
          <T key="ad" msg={mp.answers410} values={{ path: code(postPath(p)) }} />
        ),
      ],
      [t(mp.leftOutOf), t(mp.leftOutText)],
    ]
    footer = canPublish ? (
      <>
        <Button onClick={() => onDialog({ kind: 'redirect', post: p })}>{t(mp.redirect)}</Button>
        <Button variant="primary" loading={busy} onClick={() => void publishNow()}>
          {t(mp.publishAgain)}
        </Button>
      </>
    ) : null
  } else if (p.status === 'scheduled') {
    sub = p.publishAt ? scheduledLine(p.publishAt, p.publishTz) : ''
    const tierNames = p.tiers.length ? p.tiers.map((x) => x.label || x.name).join(', ') : ''
    rows = [
      [
        t(mp.by),
        <span className="inline-flex items-center gap-2" key="by">
          <Avatar name={p.byline.name || p.createdBy} size={22} />
          {p.byline.name || p.createdBy}
        </span>,
      ],
      [
        t(mp.stateVisibility),
        <span className="inline-flex flex-wrap items-center gap-1" key="vis">
          {p.visibility === 'paid' && tierNames ? (
            <span className="ws-vis">
              <Icon name="star" size={14} />
              {tierNames}
            </span>
          ) : (
            <VisibilityLabel visibility={p.visibility} />
          )}
          {p.visibility !== 'public' ? <span className="k">· {p.neverFree ? t(mp.neverFreeMeter) : t(mp.meterCounts)}</span> : null}
        </span>,
      ],
      [t(mp.statePreview), p.visibility === 'public' ? t(mp.wholePost) : p.hasMarker ? t(mp.markerAfter, { count: p.markerAfter }) : t(mp.noMarker)],
      [t(mp.whenLive), p.tags.length ? t(mp.whenLiveText, { tags: andList(p.tags.map((x) => x.name)) }) : t(mp.whenLiveNoTags)],
    ]
    footer = canPublish ? (
      <>
        {p.publishMode === 'with_send' ? null : <Button onClick={() => onDialog({ kind: 'time', post: p })}>{t(mp.changeTime)}</Button>}
        <Button variant="primary" loading={busy} onClick={() => void publishNow()}>
          {t(mp.publishNow)}
        </Button>
      </>
    ) : null
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={p.title}
      eyebrow={<StatusBadge status={p.status} />}
      description={sub || undefined}
      footer={footer ?? undefined}
    >
      <dl className="ws-kv">
        {rows.map(([k, v]) => (
          <div key={k} className="contents">
            <dt>{k}</dt>
            <dd>{v}</dd>
          </div>
        ))}
      </dl>
      {note}
    </Dialog>
  )
}

function visName(t: ReturnType<typeof useT>, v: string): string {
  return v === 'paid' ? t(mp.paid) : v === 'members' ? t(mp.members) : t(mp.public)
}

/** Unpublish: a reason and where its address goes, kept with the post and audited. */
export function UnpublishDialog({ post: p, onClose }: { post: PostRow; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const [reason, setReason] = useState('')
  const [to, setTo] = useState('')
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="warn"
      badge={t(mp.unpublishBadge)}
      title={t(mp.unpublishTitle, { title: p.title })}
      description={
        <>
          {t(mp.unpublishText)} {p.kind === 'campaign' ? t(mp.unpublishCampaign) : null}
        </>
      }
      confirmLabel={t(mp.unpublishBadge)}
      confirmVariant="danger"
      onConfirm={async () => {
        await http.POST(`${WS}/posts/{post}/unpublish`, { params: { path: { workspace: slug, post: p.id } }, body: { reason: reason.trim() || undefined, redirectTo: to.trim() || undefined } })
        toast({ kind: 'success', title: t(mp.unpublishedToast, { title: p.title }) })
        void invalidate()
      }}
    >
      <Field label={t(mp.reasonLabel)} hint={t(mp.reasonHint)}>
        <Textarea value={reason} maxLength={300} rows={2} onChange={(e) => setReason(e.target.value)} />
      </Field>
      <Field label={t(mp.redirectLabel)} hint={t(mp.redirectHint)}>
        <Input className="mono" value={to} maxLength={300} placeholder="/posts/…" onChange={(e) => setTo(e.target.value)} />
      </Field>
    </ConfirmDialog>
  )
}

/** Delete a web-only post: its versions and numbers go too; audited. */
export function DeletePostDialog({ post: p, onClose }: { post: PostRow; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  return (
    <ConfirmDialog
      open
      onOpenChange={(o) => !o && onClose()}
      tone="bad"
      badge={t(mp.deleteBadge)}
      title={t(mp.deleteTitle, { title: p.title })}
      description={p.status === 'draft' ? t(mp.deleteDraftText) : t(mp.deleteText)}
      confirmLabel={t(mp.deleteConfirm)}
      onConfirm={async () => {
        await http.DELETE(`${WS}/posts/{post}`, { params: { path: { workspace: slug, post: p.id } } })
        toast({ kind: 'deleted', title: t(mp.deleted), detail: p.title })
        void invalidate()
      }}
    />
  )
}

/** "Redirect to a post or page": where an unpublished post's address sends readers. */
export function RedirectDialog({ post: p, onClose }: { post: PostRow; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const [to, setTo] = useState(p.redirectTo ?? '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const save = async (value: string) => {
    setBusy(true)
    setError(undefined)
    try {
      await http.PUT(`${WS}/posts/{post}/redirect`, { params: { path: { workspace: slug, post: p.id } }, body: { redirectTo: value.trim() || undefined } })
      toast({ kind: 'success', title: t(mp.redirectSaved) })
      void invalidate()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mp.redirectTitle, { title: p.title })}
      description={<T msg={mp.redirectText} values={{ path: <code className="mono">{postPath(p)}</code> }} />}
      footer={
        <>
          {p.redirectTo ? (
            <Button disabled={busy} onClick={() => void save('')}>
              {t(mp.redirectRemove)}
            </Button>
          ) : null}
          <Button variant="primary" loading={busy} disabledReason={to.trim() ? undefined : t(mp.redirectHint)} onClick={() => void save(to)}>
            {t(mp.redirectSave)}
          </Button>
        </>
      }
    >
      <Field label={t(mp.redirectLabel)} hint={t(mp.redirectHint)} error={error}>
        <Input className="mono" value={to} maxLength={300} placeholder="/posts/…" onChange={(e) => setTo(e.target.value)} />
      </Field>
    </Dialog>
  )
}

/** "Change the time": a scheduled post's time, in its timezone. */
export function PublishTimeDialog({ post: p, host, onClose }: { post: PostRow; host: string; onClose: () => void }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const invalidate = useInvalidateWebsite()
  const tz = p.publishTz || me.timezone || 'UTC'
  const [local, setLocal] = useState(p.publishAt ? isoToZonedLocal(p.publishAt, tz) : '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const id = useId()
  void host
  const save = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const at = zonedToISO(local, tz)
      await http.POST(`${WS}/posts/{post}/publish`, { params: { path: { workspace: slug, post: p.id } }, body: { when: 'at', at, tz } })
      toast({ kind: 'success', title: t(mp.scheduledToast, { title: p.title, date: scheduledLine(at, tz) }) })
      void invalidate()
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mp.timeTitle)}
      description={t(mp.timeText, { tz })}
      size="sm"
      footer={
        <Button variant="primary" loading={busy} disabledReason={local ? undefined : t(mp.timeLabel)} onClick={() => void save()}>
          {t(mp.timeSave)}
        </Button>
      }
    >
      <Field label={t(mp.timeLabel)} error={error}>
        <Input id={id} type="datetime-local" value={local} onChange={(e) => setLocal(e.target.value)} />
      </Field>
    </Dialog>
  )
}

type As = 'visitor' | 'free' | 'paid'

/** A post's page as a reader sees it (inert), for a draft or an approval's review. */
export function PostPreviewDialog({ postId, title, onClose }: { postId: string; title: string; onClose: () => void }) {
  const t = useT()
  const [as, setAs] = useState<As>('paid')
  const preview = usePostPreview(postId, as)
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()} title={title} size="lg" description={t(mp.preview)}>
      <div className="flex flex-col gap-3">
        <Segmented
          value={as}
          onChange={setAs}
          label={t(mp.preview)}
          items={[
            { value: 'visitor', label: t(mp.previewVisitor) },
            { value: 'free', label: t(mp.previewFree) },
            { value: 'paid', label: t(mp.previewPaid) },
          ]}
        />
        {preview.data ? (
          <div className="max-h-[70vh] overflow-auto rounded-[8px] border border-line">
            <SiteFrame html={preview.data.html} title={t(mp.previewFrame, { title })} minHeight={560} />
          </div>
        ) : preview.isError ? (
          <Note tone="bad">{errorMessage(preview.error)}</Note>
        ) : (
          <Loading label={t(mp.preview)}>
            <Skeleton w="100%" h={560} r={8} />
          </Loading>
        )}
      </div>
    </Dialog>
  )
}

/** A post's page rendered by the server as one kind of reader (inert). */
export function usePostPreview(postId: string, as: 'visitor' | 'visitor_used' | 'free' | 'free_used' | 'paid', enabled = true) {
  const { slug } = useWorkspace()
  return useQuery({
    queryKey: ['website-post-preview', slug, postId, as],
    enabled,
    queryFn: async () => {
      const { data } = await http.POST(`${WS}/posts/{post}/preview`, { params: { path: { workspace: slug, post: postId } }, body: { as } })
      return data!
    },
  })
}
