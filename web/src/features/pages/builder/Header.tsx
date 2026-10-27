import { Link } from '@tanstack/react-router'
import { useLayoutEffect, useRef } from 'react'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { useMe, useWorkspace } from '@/lib/session'
import { useBuilder } from './context'
import { DEVICE_WIDTH, type PageDevice } from './editor/pageEditor'
import { UiIcon } from './icons'
import { m } from './messages'
import { dayTimeText, whenText } from './wording'

/** The signed-in person's name. */
export function useMyName(): string {
  const { data } = useMe()
  return data?.user?.name || data?.user?.email || ''
}

export function DeviceSeg() {
  const t = useT()
  const b = useBuilder()
  const items: { d: PageDevice; label: string }[] = [
    { d: 'desktop', label: t(m.deviceDesktop) },
    { d: 'tablet', label: t(m.deviceTablet) },
    { d: 'phone', label: t(m.devicePhone) },
  ]
  return (
    <div className="seg pb-devseg" role="group" aria-label={t(m.deviceLabel)}>
      {items.map((it) => (
        <button
          key={it.d}
          type="button"
          aria-pressed={b.device === it.d}
          aria-label={it.label}
          title={t(m.deviceTitle, { device: it.label, width: DEVICE_WIDTH[it.d] })}
          onClick={() => b.setDevice(it.d)}
        >
          <UiIcon name={it.d} />
          <span className="pb-devlabel">{it.label}</span>
        </button>
      ))}
    </div>
  )
}

function StatusBadge() {
  const t = useT()
  const { page: p, status } = useBuilder()
  switch (p.status) {
    case 'published':
      return p.changes || status.kind === 'dirty' ? (
        <Badge tone="warn" dot>
          {t(m.statusChanges)}
        </Badge>
      ) : (
        <Badge tone="ok" dot>
          {t(m.statusPublished)}
        </Badge>
      )
    case 'scheduled':
      return (
        <Badge tone="info" dot>
          {t(m.statusScheduled, { when: p.publishAt ? dayTimeText(t, p.publishAt) : '' })}
        </Badge>
      )
    case 'unpublished':
      return <Badge tone="neutral">{t(m.statusUnpublished)}</Badge>
  }
  return <Badge tone="neutral">{t(m.statusDraft)}</Badge>
}

/** Autosave's state, as the composer's header says it, and who saved last when it wasn't you. */
function SaveState() {
  const t = useT()
  const { status, canEdit } = useBuilder()
  const me = useMyName()
  let text = ''
  let tone = ''
  let by = ''
  let full = ''
  switch (status.kind) {
    case 'saving':
      text = t(m.saving)
      break
    case 'dirty':
      text = t(m.unsaved)
      break
    case 'offline':
      text = t(m.offline)
      tone = ' pb-bad'
      break
    case 'error':
      text = t(m.notSaved)
      tone = ' pb-bad'
      break
    case 'conflict':
      text = t(m.conflictState)
      tone = ' pb-warn'
      break
    case 'saved':
      text = status.auto && canEdit ? t(m.autosavedAt, { time: whenText(t, status.at, undefined, true) }) : t(m.savedAt, { time: whenText(t, status.at, undefined, true) })
      full = t(status.auto && canEdit ? m.autosavedAt : m.savedAt, { time: dayTimeText(t, status.at) })
      if (status.by && status.by !== me) by = status.by
      break
  }
  return (
    <>
      <span className={`k${tone}`} role="status" aria-live="polite" title={full ? (by ? t(m.savedByAt, { when: full, name: by }) : full) : undefined}>
        {text}
      </span>
      {by ? <span className="k pb-savedby">{by}</span> : null}
    </>
  )
}

/** Publish, with why it can't happen yet. */
function PublishButton() {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  if (!b.canEdit && !b.canPublishPerm) return null
  const blocking = p.checks.filter((c) => c.status === 'block').length
  const live = p.status === 'published'
  const label = live ? t(m.publishChanges) : t(m.publish)
  if (!b.canPublishPerm) {
    const helpers = p.publish.publishers
    const reason = helpers.length ? t(m.publishWhy, { helpers: formatList(helpers, 'disjunction') }) : t(m.publishWhyNoHelpers)
    return (
      <Button variant="primary" disabledReason={reason}>
        {label}
      </Button>
    )
  }
  if (p.publish.pending)
    return (
      <Button variant="primary" disabled>
        {t(m.approvalPending)}
      </Button>
    )
  if (blocking)
    return (
      <>
        <span className="k">{t(m.fixChecks, { count: blocking })}</span>
        <Button variant="primary" disabled onClick={() => b.setRightTab('checks')}>
          {label}
        </Button>
      </>
    )
  if (p.publish.needsApproval && !p.publish.canPublish)
    return (
      <Button variant="primary" onClick={b.startPublish}>
        {t(m.requestApproval)}
      </Button>
    )
  const unchanged = live && !p.changes && b.status.kind === 'saved'
  if (unchanged)
    return (
      <Button variant="primary" disabled>
        {t(m.noChanges)}
      </Button>
    )
  return (
    <Button variant="primary" onClick={b.startPublish}>
      {label}
    </Button>
  )
}

/** The preview route, for the people of the workspace: the draft as readers will see it. */
export function previewUrl(slug: string, pageId: string): string {
  return `${window.location.origin}/w/${encodeURIComponent(slug)}/forms/pages/${encodeURIComponent(pageId)}/preview`
}

export function CopyPreviewLink({ small }: { small?: boolean }) {
  const t = useT()
  const b = useBuilder()
  const copy = async () => {
    const url = previewUrl(b.slug, b.page.id)
    try {
      await navigator.clipboard.writeText(url)
      toast({ kind: 'success', title: t(m.previewLinkCopied), detail: t(m.previewLinkCopiedDetail) })
    } catch {
      toast({ kind: 'info', title: t(m.previewLinkFailed, { url }) })
    }
  }
  return (
    <Button size={small ? 'sm' : 'md'} onClick={() => void copy()}>
      <UiIcon name="link" size={13} />
      {t(m.copyPreviewLink)}
    </Button>
  )
}

const desktop = '(min-width: 1101px)'

/**
 * Marks the header .tight when its one line can't hold everything with the
 * device switch's labels: the width each part needs (the name's in full),
 * against the room there is.
 */
function fitHeader(head: HTMLElement, full: { seg: number }) {
  const seg = head.querySelector<HTMLElement>('.pb-devseg')
  if (!seg) return
  if (!window.matchMedia(desktop).matches) {
    head.classList.remove('tight')
    return
  }
  const tight = head.classList.contains('tight')
  if (!tight) full.seg = seg.offsetWidth
  const cs = getComputedStyle(head)
  const kids = Array.from(head.children).filter((k) => getComputedStyle(k).display !== 'none')
  let need = parseFloat(cs.paddingLeft) + parseFloat(cs.paddingRight) + (parseFloat(cs.columnGap) || 0) * (kids.length - 1)
  for (const k of kids) need += Math.max(k.scrollWidth, k.getBoundingClientRect().width)
  if (tight) need += full.seg - seg.offsetWidth
  head.classList.toggle('tight', need > head.clientWidth + 0.5)
}

/**
 * The builder's top bar (PageBuilder): back, Pages › the name, the address,
 * the status, autosave; then the device, Preview, Versions and Publish.
 */
export function Header() {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const ws = useWorkspace()
  const me = useMyName()
  const viewer = !b.canEdit || !b.canPublishPerm
  const ref = useRef<HTMLElement>(null)
  const full = useRef({ seg: 0 })
  // After every render (what the header holds changes with the page's state) and on resizes.
  useLayoutEffect(() => {
    if (ref.current) fitHeader(ref.current, full.current)
  })
  useLayoutEffect(() => {
    const head = ref.current
    if (!head) return
    const ro = new ResizeObserver(() => fitHeader(head, full.current))
    ro.observe(head)
    return () => ro.disconnect()
  }, [])
  return (
    <header ref={ref} className="cmp-head pb-head">
      <Link to="/w/$workspace/forms/pages" params={{ workspace: b.slug }} className="btn ic" aria-label={t(m.back)}>
        <UiIcon name="back" size={16} />
      </Link>
      <nav className="pb-crumb" aria-label={t(m.breadcrumb)}>
        <Link to="/w/$workspace/forms/pages" params={{ workspace: b.slug }}>
          {t(m.crumbPages)}
        </Link>
        <span className="k" aria-hidden="true">
          ›
        </span>
        <b title={p.name}>{p.name}</b>
      </nav>
      <span className="pb-idchip" title={t(m.addressTitle)}>
        {p.host}
        {p.path}
      </span>
      <StatusBadge />
      {b.canEdit ? null : <Badge tone="neutral">{t(m.readOnly)}</Badge>}
      <SaveState />
      {viewer && me ? <span className="k">{t(m.viewerLine, { name: me, role: ws.me.workspace.roleName })}</span> : null}
      <div className="pb-acts">
        <DeviceSeg />
        <Button onClick={() => b.openDialog({ kind: 'preview' })}>
          <UiIcon name="eye" />
          {t(m.preview)}
        </Button>
        {b.canEdit && !b.canPublishPerm ? <CopyPreviewLink /> : null}
        <Button onClick={() => b.openDialog({ kind: 'versions' })}>{t(m.versions)}</Button>
        <PublishButton />
      </div>
    </header>
  )
}

/**
 * Who can do what here, under the header: a Content Writer edits and
 * saves drafts but doesn't publish (PagesStates · Daniel Park's draft); a
 * Viewer only looks.
 */
export function PermissionNote() {
  const t = useT()
  const b = useBuilder()
  const ws = useWorkspace()
  const me = useMyName()
  const role = ws.me.workspace.roleName
  if (!b.canEdit) {
    return (
      <div className="pb-band">
        <div className="note n" role="note">
          <UiIcon name="info" size={16} color="#4B5563" />
          <div className="flex min-w-0 flex-col gap-1">
            <span className="font-bold">{t(m.readOnlyTitle, { name: me, role })}</span>
            <div>
              {t(m.readOnlyBody)}
            </div>
          </div>
        </div>
      </div>
    )
  }
  if (b.canPublishPerm) return null
  const helper = b.page.publish.publishers[0]
  return (
    <div className="pb-band">
      <div className="note n" role="note">
        <UiIcon name="info" size={16} color="#4B5563" />
        <div className="flex min-w-0 flex-col gap-1">
          <span className="font-bold">{t(m.cantPublishTitle, { name: me })}</span>
          <div>
            {helper ? t(m.cantPublishBody, { role, helper }) : t(m.cantPublishBodyNoHelper, { role })}
          </div>
        </div>
      </div>
    </div>
  )
}
