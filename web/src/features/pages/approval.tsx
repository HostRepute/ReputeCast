import { Link } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Skeleton } from '@/components/ui/Feedback'
import { t as tr, useT } from '@/i18n'
import { clock, monthDay } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { zonedWhen } from '@/features/review/logic'
import { builderLink, TEMPLATE_KEYS, type TemplateKey } from './format'
import { m } from './approval.messages'
import { templateInfo } from './templates'
import './pages.css'

/**
 * Pages in the approval inbox and review (design expansion phase 7, §6
 * "Approval inbox and review"): a request to publish a page carries a
 * snapshot of it (name, template, address, visibility, when it goes live).
 * Once approved, the page goes live by itself, now or at its time.
 */
type Snapshot = Schemas['ApprovalPageSnapshot']
type Detail = Schemas['ApprovalDetail']

function templateOf(key: string): TemplateKey | undefined {
  return (TEMPLATE_KEYS as readonly string[]).includes(key) ? (key as TemplateKey) : undefined
}

/** The "Page" badge beside a request's name; it opens the page in the builder. */
export function PageBadge({ page }: { page: Snapshot }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <Link {...builderLink(slug, page.pageId)} className="badge mut no-underline" aria-label={t(m.badgeLabel, { name: page.name })}>
      <Icon name="forms" size={12} strokeWidth={2} />
      {t(m.badge)}
    </Link>
  )
}

/** "Page · /field-guide · Public": what a request publishes, under its name. */
export function PageLine({ page }: { page: Snapshot }) {
  const t = useT()
  return <span className="k">{t(m.line, { path: page.path, visibility: page.visibilityLabel })}</span>
}

/** The inbox's Scheduled column for a page: "When approved", "Oct 22, 09:00" or "Published". */
export function pageScheduledText(page: Snapshot): string {
  if (page.published) return tr(m.published)
  if (page.when === 'at' && page.publishAt) return tr(m.scheduled, { date: monthDay(page.publishAt), time: clock(page.publishAt) })
  return tr(m.whenApproved)
}

/**
 * The review's page card: what the page is, its address, visibility and
 * publish time, the version the approval covers, and the page as readers
 * will see it (get-approval-preview draws it in the site's frame).
 */
export function PageApprovalCard({ detail: d }: { detail: Detail }) {
  const t = useT()
  const { slug, me } = useWorkspace()
  const [open, setOpen] = useState(false)
  const preview = $api.useQuery('get', '/api/v1/workspaces/{workspace}/approvals/{approvalId}/preview', { params: { path: { workspace: slug, approvalId: d.id } } })
  const p = d.page
  if (!p) return null
  const template = templateOf(p.template)
  const live = p.published ? t(m.goesLivePublished) : p.when === 'at' && p.publishAt ? zonedWhen(p.publishAt, me.timezone) : t(m.goesLiveNow)
  return (
    <section className="card" aria-labelledby="ap-page">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="ap-page">
            {t(m.cardTitle)}
          </h2>
          <span className="k">{t(m.cardCaption)}</span>
        </div>
        <ButtonLink size="sm" {...builderLink(slug, p.pageId)}>
          {t(m.openBuilder)}
        </ButtonLink>
      </div>
      <div className="pad col gap-3!">
        <dl className="kv">
          <dt>{t(m.page)}</dt>
          <dd className="font-semibold">{template ? t(m.pageValue, { name: p.name, template: t(templateInfo(template).name) }) : p.name}</dd>
          <dt>{t(m.address)}</dt>
          <dd className="mono">{p.path}</dd>
          <dt>{t(m.visibility)}</dt>
          <dd>{p.visibilityLabel}</dd>
          <dt>{t(m.goesLive)}</dt>
          <dd>{live}</dd>
          {d.version > 0 ? (
            <>
              <dt>{t(m.version)}</dt>
              <dd>{t(m.versionValue, { version: d.version })}</dd>
            </>
          ) : null}
        </dl>
        <div className="ap-preview">
          {preview.data ? (
            <PageThumb html={preview.data.html} title={t(m.previewTitle, { name: p.name, version: preview.data.version })} />
          ) : preview.isError ? (
            <span className="k block py-6 text-center">{t(m.previewUnavailable, { error: errorMessage(preview.error) })}</span>
          ) : (
            <div className="pg-thumb flex flex-col gap-2 p-3.5" aria-hidden="true">
              <Skeleton w="40%" h={10} />
              <Skeleton w="100%" h={90} r={4} />
              <Skeleton w="80%" h={12} />
              <Skeleton w="90%" h={10} />
            </div>
          )}
        </div>
        <div className="row gap-2!">
          <Button size="sm" onClick={() => setOpen(true)} disabledReason={preview.data ? undefined : t(m.previewLoading)}>
            {t(m.openPreview)}
          </Button>
        </div>
      </div>
      {open && preview.data ? (
        <Dialog open onOpenChange={setOpen} size="lg" title={p.name} description={t(m.previewCaption)}>
          <iframe title={t(m.previewFullTitle, { name: p.name })} sandbox="allow-same-origin" srcDoc={preview.data.html} className="h-[70vh] w-full rounded-lg border border-line bg-white" />
        </Dialog>
      ) : null}
    </section>
  )
}

/** A page is drawn at desktop width and scaled to the card; a long page is cut at 360 px. */
const PAGE_WIDTH = 1200

function PageThumb({ html, title }: { html: string; title: string }) {
  const box = useRef<HTMLDivElement>(null)
  const frame = useRef<HTMLIFrameElement>(null)
  const [scale, setScale] = useState(0.5)
  const [height, setHeight] = useState(900)
  useEffect(() => {
    const el = box.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(() => setScale(el.clientWidth / PAGE_WIDTH))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  const measure = () => {
    const body = frame.current?.contentDocument?.body
    if (body) setHeight(Math.max(300, body.scrollHeight))
  }
  return (
    <div ref={box} className="pg-thumb" style={{ height: Math.min(360, height * scale) }}>
      <iframe ref={frame} title={title} sandbox="allow-same-origin" srcDoc={html} tabIndex={-1} onLoad={measure} style={{ width: PAGE_WIDTH, height, transform: `scale(${scale})` }} />
    </div>
  )
}

/** The decision card's note for a page (the campaign note talks about audiences and routes). */
export function PageDecisionNote({ author, page }: { author: string; page: Snapshot }) {
  const t = useT()
  return (
    <div className="note n text-[12.5px]">
      <span>{t(m.decisionNote, { name: author.split(' ')[0], when: page.when })}</span>
    </div>
  )
}

/** What rejecting or requesting changes does to a page publish request. */
export function PageDecisionDetail({ kind, author }: { kind: 'reject' | 'request_changes'; author: string }) {
  const t = useT()
  return <>{t(kind === 'reject' ? m.rejectDetail : m.changesDetail, { name: author })}</>
}
