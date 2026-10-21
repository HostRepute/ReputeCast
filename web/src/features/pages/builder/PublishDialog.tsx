import { Fragment, useState } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { dayMonthDate } from '@/lib/format'
import { useBuilder, type PageCheck, type PageDetail } from './context'
import { timeZones, zonedInstant } from './model'
import { m } from './messages'
import { dayTimeText, zoneLabel, type T } from './wording'

type PageDiff = Schemas['PageDiff']
type DiffSetting = Schemas['PageDiffSetting']

const TEMPLATE: Record<PageDetail['template'], keyof typeof m> = {
  about: 'tplAbout',
  lead_magnet: 'tplLeadMagnet',
  launch: 'tplLaunch',
  webinar: 'tplWebinar',
  sales: 'tplSales',
  blank: 'tplBlank',
}

export function templateName(t: T, tpl: PageDetail['template']): string {
  return t(m[TEMPLATE[tpl]])
}

/** A check as a badge of the publish dialog: "Address /field-guide is yours", "Stripe · Acme Media · Live". */
export function checkBadge(t: T, c: PageCheck): { tone: Tone; label: string } {
  const tone: Tone = c.status === 'pass' ? 'ok' : c.status === 'warn' ? 'warn' : 'bad'
  switch (c.key) {
    case 'address':
      return { tone, label: c.status === 'block' ? t(m.chkAddressTaken, { path: c.path ?? '' }) : c.detail === 'yours' ? t(m.pbAddressYours, { path: c.path ?? '' }) : t(m.pbAddressFree, { path: c.path ?? '' }) }
    case 'heading':
      return { tone, label: c.status === 'pass' ? t(m.pbHeading) : t(m.pbHeadingMissing) }
    case 'moments':
      return { tone, label: c.status === 'pass' ? t(m.pbMoments) : t(m.pbPast) }
    case 'product':
      return { tone, label: c.status === 'pass' ? t(m.pbProduct) : c.problem === 'missing' ? t(m.chkProductMissing, { count: c.count ?? 1 }) : c.problem === 'draft' ? t(m.chkProductDraft, { name: c.name ?? '' }) : t(m.chkProductDeleted) }
    case 'stripe':
      return { tone, label: c.status === 'pass' ? t(m.pbStripe, { account: c.account ?? '', mode: c.mode === 'test' ? t(m.pbStripeTest) : t(m.pbStripeLive) }) : t(m.chkStripeMissing) }
    case 'form':
      return { tone, label: c.status === 'pass' ? t(m.pbForm, { name: c.name ?? '' }) : c.problem === 'missing' ? t(m.chkFormMissing, { count: c.count ?? 1 }) : c.problem === 'paused' ? t(m.chkFormPaused, { name: c.name ?? '' }) : t(m.chkFormDeleted) }
    case 'images':
      return { tone, label: c.status === 'pass' ? t(m.pbImages, { count: c.count ?? 0 }) : t(m.pbImageNoAlt, { name: c.name || c.ref || '' }) }
    case 'links':
      return { tone, label: c.status === 'pass' ? t(m.pbLinks, { count: c.count ?? 0 }) : t(m.pbLinkBroken, { name: c.name || c.ref || '' }) }
    case 'visibility':
      return { tone, label: t(m.chkVisibility) }
    case 'store_blocks':
      return { tone, label: t(m.pbStores) }
  }
  return { tone, label: c.key }
}

/** "Meta description changed", "Address changed from /a to /b". */
function settingLine(t: T, s: DiffSetting): string {
  switch (s.key) {
    case 'name':
      return t(m.setName)
    case 'slug':
      return t(m.setSlug, { before: s.before, after: s.after })
    case 'seoTitle':
      return t(m.setSeoTitle)
    case 'seoDescription':
      return t(m.setSeoDescription)
    case 'socialImage':
      return t(m.setSocialImage)
    case 'visibility':
      return t(m.setVisibility, { after: s.after })
    case 'tiers':
      return t(m.setTiers)
    case 'hideHeader':
      return t(m.setHideHeader, { after: s.after })
    case 'showInNav':
      return t(m.setShowInNav, { after: s.after })
    case 'navLabel':
      return t(m.setNavLabel)
  }
  return t(m.setOther, { key: s.key })
}

/** The diff since the live version: block by block, then the page's settings. */
interface DiffBlock {
  id: string
  name: string
  kind: string
  lines: readonly { op: string; text: string }[]
}

/** Changed blocks as rows of a .diff box: the block's name, then its removed and added lines. */
export function BlockDiffRows({ blocks }: { blocks: readonly DiffBlock[] }) {
  const t = useT()
  // Fragments: the rows stay direct children of .diff, which pads them.
  return blocks.map((blk) => (
    <Fragment key={blk.id}>
      <div className="ctx pb-diffhead">{blk.kind === 'added' ? t(m.blockAdded, { name: blk.name }) : blk.kind === 'removed' ? t(m.blockRemoved, { name: blk.name }) : blk.name}</div>
      {blk.lines.map((l, i) => (
        <div key={i} className={l.op === '+' ? 'add' : l.op === '-' ? 'del' : 'ctx'}>
          <span aria-hidden="true">{l.op === '+' ? '+ ' : l.op === '-' ? '− ' : '  '}</span>
          <span className="vh">{l.op === '+' ? t(m.vAddedLine) : l.op === '-' ? t(m.vRemovedLine) : ''}</span>
          {l.text}
        </div>
      ))}
    </Fragment>
  ))
}

export function DiffBox({ diff }: { diff: PageDiff }) {
  const t = useT()
  const [every, setEvery] = useState(false)
  if (diff.first) return <p className="k m-0">{t(m.firstPublish)}</p>
  const settings = diff.settings
  return (
    <div className="diff pb-diff">
      {diff.blocks.length ? null : <div className="ctx">{t(m.noBlockChanges)}</div>}
      <BlockDiffRows blocks={diff.blocks} />
      {settings.length ? (
        <>
          <div className="ctx pb-diffhead">{t(m.pageSettings)}</div>
          {every ? (
            settings.map((s) => (
              <div key={s.key} className="ctx">
                {settingLine(t, s)} · {t(m.setDetail, { before: s.before || t(m.emptyValue), after: s.after || t(m.emptyValue) })}
              </div>
            ))
          ) : (
            <div className="ctx">
              {settings.map((s) => settingLine(t, s)).join(' · ')} ·{' '}
              <button type="button" className="pb-linkbtn" onClick={() => setEvery(true)}>
                {t(m.showEveryChange)}
              </button>
            </div>
          )}
          {every ? (
            <div className="ctx">
              <button type="button" className="pb-linkbtn" onClick={() => setEvery(false)}>
                {t(m.showFewerChanges)}
              </button>
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  )
}

/**
 * The publish dialog (PageBuilderStates · the publish dialog): what
 * publishing changes since the live version, the checks, and when.
 */
export function PublishDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const [when, setWhen] = useState(b.publishWhen)
  const [busy, setBusy] = useState(false)
  const diffQ = $api.useQuery('get', '/api/v1/workspaces/{workspace}/pages/{page}/diff', { params: { path: { workspace: b.slug, page: p.id } } }, { enabled: open, staleTime: 0, gcTime: 0, retry: false })
  const diff = diffQ.data
  const first = !p.live
  const blocking = p.checks.filter((c) => c.status === 'block')
  const warnings = p.checks.filter((c) => c.status === 'warn')
  const passed = p.checks.filter((c) => c.status === 'pass')
  const at = when.when === 'at' ? zonedInstant(when.date, when.time, when.tz) : null
  // The dialog opens at a moment; a time a minute from it is still the future when it publishes.
  const [opened] = useState(() => Date.now())
  const atPast = when.when === 'at' && (!at || at.getTime() <= opened)
  const zones = timeZones([when.tz, b.tz, 'UTC'])
  const onDay = new Date(`${when.date || '2026-01-01'}T12:00:00Z`)
  const address = `${p.host}${p.path}`
  const title = first ? t(m.publishTitle, { name: p.name }) : t(m.publishChangesTitle, { name: p.name })
  const headLine = blocking.length ? t(m.checksHeadBlock, { count: blocking.length }) : warnings.length ? t(m.checksHeadWarn, { passed: passed.length, count: warnings.length }) : t(m.checksHeadPassed, { count: passed.length })
  const publish = async () => {
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/publish', {
        params: { path: { workspace: b.slug, page: p.id } },
        body: when.when === 'at' && at ? { when: 'at', at: at.toISOString(), tz: when.tz } : { when: 'now' },
      })
      if (!data) return
      b.applyDetail(data)
      b.setPublishWhen(when)
      onClose()
      if (when.when === 'at' && at) {
        toast({ kind: 'success', title: t(m.scheduled, { name: data.name, when: dayTimeText(t, at) }) })
        return
      }
      toast({
        kind: 'success',
        title: t(m.published, { name: data.name }),
        detail: `${data.host}${data.path}`,
        undo: () => {
          void http
            .POST('/api/v1/workspaces/{workspace}/pages/{page}/undo-publish', { params: { path: { workspace: b.slug, page: p.id } } })
            .then(({ data: d }) => {
              if (d) b.applyDetail(d)
              toast({ kind: 'info', title: t(m.undone) })
            })
            .catch((e: unknown) => toast({ kind: 'error', title: t(m.undoFailed), detail: errorMessage(e) }))
        },
      })
    } catch (e) {
      if (e instanceof ApiError && e.code === 'checks_failed') {
        toast({ kind: 'error', title: t(m.checksFailed), detail: errorMessage(e) })
        await b.refreshChecks()
        b.setRightTab('checks')
        onClose()
      } else if (e instanceof ApiError && e.code === 'approval_required') {
        b.openDialog({ kind: 'approval' })
      } else {
        toast({ kind: 'error', title: t(m.publishFailed), detail: errorMessage(e) })
      }
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && !busy && onClose()}
      hideClose
      className="pb-pubdlg"
      eyebrow={<span className="k">{t(m.dlgEyebrow, { address, template: templateName(t, p.template) })}</span>}
      title={title}
      description={first ? t(m.publishLeadFirst, { url: address }) : t(m.publishLead)}
      footer={
        <>
          <span className="k" style={{ marginRight: 'auto' }}>
            {p.live ? t(m.auditNote, { version: p.live.version }) : t(m.auditNoteFirst)}
          </span>
          <Button onClick={onClose} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={busy} disabled={Boolean(blocking.length) || atPast} onClick={() => void publish()}>
            {busy ? t(m.publishing) : when.when === 'at' ? t(m.schedule) : t(m.publish)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-2">
        {p.live && diff && !diff.first ? (
          <span className="font-semibold">
            {diff.publishedBy && diff.publishedAt
              ? t(m.since, { version: diff.since, date: dayMonthDate(diff.publishedAt), name: diff.publishedBy })
              : t(m.sinceNoName, { version: diff.since, date: diff.publishedAt ? dayMonthDate(diff.publishedAt) : '' })}
          </span>
        ) : null}
        {diffQ.isPending ? <span className="k">{t(m.diffLoading)}</span> : diffQ.isError ? <span className="k">{t(m.diffFailed)}</span> : diff ? <DiffBox diff={diff} /> : null}
      </div>
      <div className="flex flex-col gap-2">
        <span className="font-semibold">{headLine}</span>
        <div className="flex flex-wrap gap-1.5">
          {[...blocking, ...warnings, ...passed].map((c, i) => {
            const x = checkBadge(t, c)
            return (
              <Badge key={i} tone={x.tone}>
                {x.label}
              </Badge>
            )
          })}
        </div>
      </div>
      <div className="flex flex-col gap-2">
        <span className="font-semibold">{t(m.when)}</span>
        <div>
          <div className="seg" role="group" aria-label={t(m.when)}>
            <button type="button" aria-pressed={when.when === 'now'} onClick={() => setWhen({ ...when, when: 'now' })}>
              {t(m.now)}
            </button>
            <button type="button" aria-pressed={when.when === 'at'} onClick={() => setWhen({ ...when, when: 'at' })}>
              {t(m.atTime)}
            </button>
          </div>
        </div>
        {when.when === 'at' ? (
          <div className="flex flex-wrap items-end gap-2">
            <label className="lbl">
              {t(m.date)}
              <input type="date" className="in" value={when.date} onChange={(e) => setWhen({ ...when, date: e.target.value })} />
            </label>
            <label className="lbl">
              {t(m.time)}
              <input type="time" className="in" value={when.time} onChange={(e) => setWhen({ ...when, time: e.target.value })} />
            </label>
            <label className="lbl min-w-0 flex-1">
              {t(m.timeZone)}
              <select className="in" value={when.tz} onChange={(e) => setWhen({ ...when, tz: e.target.value })}>
                {zones.map((z) => (
                  <option key={z} value={z}>
                    {zoneLabel(t, z, onDay)}
                  </option>
                ))}
              </select>
            </label>
            {atPast ? <span className="err w-full">{t(m.atPast)}</span> : null}
          </div>
        ) : null}
      </div>
    </Dialog>
  )
}

/** Asks the workspace's approvers to approve publishing the page as it is. */
export function ApprovalDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const [note, setNote] = useState('')
  const [busy, setBusy] = useState(false)
  const send = async () => {
    setBusy(true)
    try {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/approval', { params: { path: { workspace: b.slug, page: p.id } }, body: note.trim() ? { note: note.trim() } : {} })
      if (data) b.applyDetail(data)
      toast({ kind: 'success', title: t(m.approvalSent) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(m.approvalFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const approvers = p.publish.approvers
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && !busy && onClose()}
      title={t(m.approvalTitle, { name: p.name })}
      description={approvers.length ? t(m.approvalLead, { approvers: formatList(approvers) }) : t(m.approvalLeadNoApprovers)}
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="primary" loading={busy} onClick={() => void send()}>
            {t(m.requestApproval)}
          </Button>
        </>
      }
    >
      <label className="lbl">
        {t(m.approvalNote)}
        <textarea className="in" rows={3} value={note} onChange={(e) => setNote(e.target.value)} maxLength={1000} />
      </label>
    </Dialog>
  )
}

/** Takes the page off the site (or takes a schedule back). */
export function UnpublishDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT()
  const b = useBuilder()
  const p = b.page
  const [reason, setReason] = useState('')
  const [redirect, setRedirect] = useState('')
  const [busy, setBusy] = useState(false)
  const scheduled = p.status === 'scheduled'
  const go = async () => {
    setBusy(true)
    try {
      const body = scheduled ? {} : { reason: reason.trim() || undefined, redirectTo: redirect.trim() || undefined }
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/pages/{page}/unpublish', { params: { path: { workspace: b.slug, page: p.id } }, body })
      if (data) b.applyDetail(data)
      toast({ kind: 'success', title: scheduled ? t(m.unscheduled) : t(m.unpublished, { name: p.name }) })
      onClose()
    } catch (e) {
      toast({ kind: 'error', title: t(m.unpublishFailed), detail: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => !o && !busy && onClose()}
      title={scheduled ? t(m.unscheduleTitle) : t(m.unpublishDlgTitle, { name: p.name })}
      description={scheduled ? t(m.unscheduleHint) : t(m.unpublishLead)}
      footer={
        <>
          <Button onClick={onClose} disabled={busy}>
            {t(common.cancel)}
          </Button>
          <Button variant="danger-primary" loading={busy} onClick={() => void go()}>
            {scheduled ? t(m.unschedule) : t(m.unpublish)}
          </Button>
        </>
      }
    >
      {scheduled ? null : (
        <>
          <label className="lbl">
            {t(m.unpublishReason)}
            <input className="in" value={reason} onChange={(e) => setReason(e.target.value)} maxLength={300} />
          </label>
          <label className="lbl">
            {t(m.redirectTo)}
            <input className="in mono" value={redirect} placeholder={'/about'} onChange={(e) => setRedirect(e.target.value)} maxLength={300} />
          </label>
          <span className="k">{t(m.redirectHint)}</span>
        </>
      )}
    </Dialog>
  )
}
