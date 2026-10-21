import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { Dialog as RDialog } from 'radix-ui'
import { useEffect, useRef, useState, type ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonAnchor } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { ErrorPanel, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Textarea } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { andList, clock, count, dayMonthDate, dayMonthYear, orList, pct, shortDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { copyText } from '../EarnPage'
import { day, money } from '../format'
import { me } from '../messages'
import { mb } from './booking.messages'
import { AmountInput, CancelDialog, IconButton, LinkCampaignDialog, PaymentDialog, ShareDialog, SponsorDialog, parseAmount, useRefresh } from './dialogs'
import { CampaignLink, Initials, ends, MoneyIcon, NoteIcon, OpenBadge, PaymentBadge, Two, WS, daysUntil, inputDay, isPastDay, isToday, type SponsorItem } from './parts'

type Detail = Schemas['SponsorBookingDetail']
type IssueDetail = Schemas['SponsorIssueDetail']
type Creative = Schemas['SponsorCreative']
type Proposal = Schemas['SponsorPropose']

/** The panel: a 640 px sheet on the right, with its own link. */
function Sheet({ label, onClose, children }: { label: string; onClose: () => void; children: ReactNode }) {
  return (
    <RDialog.Root open onOpenChange={(o) => !o && onClose()}>
      <RDialog.Portal>
        <RDialog.Overlay className="ea-sheet-ovl" />
        <RDialog.Content className="ea-sheet sp-panel" aria-describedby={undefined} aria-label={label}>
          {children}
        </RDialog.Content>
      </RDialog.Portal>
    </RDialog.Root>
  )
}

function PanelLoading({ onClose }: { onClose: () => void }) {
  const t = useT()
  return (
    <>
      <div className="ea-sheet-head">
        <div className="flex min-w-0 flex-1 flex-col gap-2">
          <RDialog.Title className="vh">{t(mb.loading)}</RDialog.Title>
          <Skeleton w={140} h={12} />
          <Skeleton w="60%" h={22} />
          <Skeleton w={220} h={20} r={999} />
        </div>
        <IconButton icon="close" label={t(mb.close)} onClick={onClose} />
      </div>
      <div className="ea-sheet-body" aria-busy="true">
        <Skeleton w="100%" h={120} />
        <Skeleton w="100%" h={240} />
      </div>
    </>
  )
}

function PanelError({ error, onRetry, onClose }: { error: unknown; onRetry: () => void; onClose: () => void }) {
  const t = useT()
  return (
    <>
      <div className="ea-sheet-head">
        <RDialog.Title className="m-0 flex-1 text-[18px] font-bold">{t(mb.loadError)}</RDialog.Title>
        <IconButton icon="close" label={t(mb.close)} onClick={onClose} />
      </div>
      <div className="ea-sheet-body">
        <ErrorPanel title={t(mb.loadError)} actions={<Button onClick={onRetry}>{t(me.retry)}</Button>}>
          {errorMessage(error)}
        </ErrorPanel>
      </div>
    </>
  )
}

/** A booking: its terms, creative, payment, notes and activity; after the send, its report and read-only link. */
export function BookingPanel({ id, onClose }: { id: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/sponsor-bookings/{id}`, { params: { path: { workspace: slug, id } } })
  return (
    <Sheet label={q.data ? t(mb.title, { issue: q.data.issue, date: shortDate(q.data.issueOn), slot: q.data.slotName }) : t(mb.loading)} onClose={onClose}>
      {q.error ? <PanelError error={q.error} onRetry={() => void q.refetch()} onClose={onClose} /> : !q.data ? <PanelLoading onClose={onClose} /> : <BookingBody b={q.data} onClose={onClose} />}
    </Sheet>
  )
}

/** An open issue: book it or hold it, or confirm an agent's proposal. */
export function IssuePanel({ slotId, number, library, onClose, onBooked }: { slotId: string; number: number; library: SponsorItem[]; onClose: () => void; onBooked: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', `${WS}/sponsor-slots/{id}/issues/{number}`, { params: { path: { workspace: slug, id: slotId, number } } })
  // A booked or held issue opens its booking instead.
  const booked = q.data?.bookingId
  useEffect(() => {
    if (booked) onBooked(booked)
  }, [booked, onBooked])
  return (
    <Sheet label={q.data ? t(mb.title, { issue: q.data.issue, date: shortDate(q.data.issueOn), slot: q.data.slotName }) : t(mb.loading)} onClose={onClose}>
      {q.error ? (
        <PanelError error={q.error} onRetry={() => void q.refetch()} onClose={onClose} />
      ) : !q.data || booked ? (
        <PanelLoading onClose={onClose} />
      ) : (
        <IssueBody d={q.data} library={library} onClose={onClose} onBooked={onBooked} />
      )}
    </Sheet>
  )
}

/** The panel's head: eyebrow, "#143 · Tue 21 Oct · Main sponsor", its badges, Copy link and Close. */
function Head({ eyebrow, title, badges, onClose }: { eyebrow: string; title: string; badges: ReactNode; onClose: () => void }) {
  const t = useT()
  return (
    <div className="ea-sheet-head">
      <div className="flex min-w-0 flex-[1_1_300px] flex-col gap-1.5">
        <span className="k">{eyebrow}</span>
        <RDialog.Title className="sp-ptitle">{title}</RDialog.Title>
        <div className="flex flex-wrap items-center gap-2">{badges}</div>
      </div>
      <div className="flex flex-none gap-1.5">
        <IconButton icon="link" label={t(mb.copyLink)} onClick={() => void copyText(window.location.href, t, toast)} />
        <IconButton icon="close" label={t(mb.close)} onClick={onClose} />
      </div>
    </div>
  )
}

function Section({ title, aside, children }: { title: string; aside?: ReactNode; children: ReactNode }) {
  return (
    <section className="sp-dsec">
      <div className="sp-dsec-head">
        <h3 className="sp-h3">{title}</h3>
        {aside}
      </div>
      {children}
    </section>
  )
}

/** "Aisha Bello · Viewer: managing bookings needs sponsors:manage. Ask …" */
function ReadOnly({ helpers }: { helpers: string[] }) {
  const t = useT()
  const { me: wme } = useWorkspace()
  const user = useMe().data?.user
  const role = roleName({ key: wme.workspace.roleKey, name: wme.workspace.roleName, builtin: true })
  return (
    <div className="note n" role="note">
      <NoteIcon tone="neutral" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(mb.readOnlyTitle, { name: user?.name || user?.email || '', role })}</span>
        <div>
          <T msg={helpers.length ? mb.readOnlyText : mb.readOnlyNoHelpers} values={{ permission: <span className="mono">sponsors:manage</span>, helpers: orList(helpers) }} />
        </div>
      </div>
    </div>
  )
}

function statusBadges(t: ReturnType<typeof useT>, b: Detail): ReactNode {
  switch (b.status) {
    case 'proposed':
      return (
        <>
          <OpenBadge />
          <Badge tone="purple">{t(mb.proposedBadge)}</Badge>
        </>
      )
    case 'hold':
      return <Two badge={<Badge tone="info">{t(mb.statusHold)}</Badge>} detail={b.holdUntil ? t(mb.thenUntil, { date: shortDate(b.holdUntil) }) : undefined} />
    case 'canceled':
      return <Badge tone="neutral">{t(mb.statusCanceled)}</Badge>
    case 'sent':
      return (
        <>
          <Badge tone="neutral">{t(mb.statusSent)}</Badge>
          <PaymentBadge p={b.payment} />
        </>
      )
  }
  return (
    <>
      <Badge tone="ok">{t(mb.statusBooked)}</Badge>
      <CreativeBadge b={b} />
      <PaymentBadge p={b.payment} />
    </>
  )
}

function CreativeBadge({ b }: { b: Detail }) {
  const t = useT()
  if (b.creativeStatus === 'ready') return <Badge tone="ok">{t(mb.creativeReady)}</Badge>
  const due = b.creativeDue
  return (
    <Two
      badge={<Badge tone="warn">{t(mb.creativeWaiting)}</Badge>}
      detail={due ? (isToday(due) ? t(mb.dueTodayShort, { date: shortDate(due) }) : isPastDay(due) ? t(mb.wasDueShort, { date: shortDate(due) }) : t(mb.dueShort, { date: shortDate(due) })) : undefined}
    />
  )
}

function BookingBody({ b, onClose }: { b: Detail; onClose: () => void }) {
  const t = useT()
  const [dialog, setDialog] = useState<'payment' | 'share' | 'cancel' | 'link'>()
  const ro = !b.canManage
  const eyebrow = b.status === 'proposed' ? t(mb.eyebrowOpen, { list: b.listName }) : ro ? t(mb.eyebrowReadOnly, { list: b.listName }) : t(mb.eyebrowBooking, { list: b.listName })
  const title = t(mb.title, { issue: b.issue, date: shortDate(b.issueOn), slot: b.slotName })
  const sponsor = b.sponsor?.name ?? ''
  const live = b.status === 'hold' || b.status === 'booked'
  return (
    <>
      <Head eyebrow={eyebrow} title={title} badges={statusBadges(t, b)} onClose={onClose} />
      <div className="ea-sheet-body">
        {ro ? <ReadOnly helpers={b.helpers} /> : null}
        {b.status === 'proposed' && b.proposal ? (
          <ProposalBody p={b.proposal} slotPrice={b.slotPriceCents} currency={b.currency} />
        ) : (
          <>
            {b.status === 'sent' && b.report ? <ReportSection b={b} /> : null}
            {b.status === 'sent' ? null : <BookingSection b={b} onLink={() => setDialog('link')} onCancel={() => setDialog('cancel')} />}
            {live && !ro ? <CreativeEditor key={`creative-${b.id}`} b={b} /> : live || b.status === 'canceled' ? <CreativeView b={b} /> : null}
            {b.status === 'booked' || b.status === 'sent' ? <PaymentSection b={b} onRecord={() => setDialog('payment')} /> : null}
            {b.status === 'sent' && b.canShare ? <ShareSection b={b} onShare={() => setDialog('share')} /> : null}
            <NotesSection key={`notes-${b.id}`} b={b} />
            <ActivitySection b={b} />
          </>
        )}
      </div>
      {b.status === 'proposed' && b.proposal && !ro ? (
        <ProposalFoot b={b} onClose={onClose} />
      ) : !ro && b.status !== 'canceled' ? (
        <div className="ea-sheet-foot">
          {live ? (
            <Button size="sm" variant="danger" onClick={() => setDialog('cancel')}>
              {t(mb.cancelBooking)}
            </Button>
          ) : null}
          <span className="k ml-auto">{t(mb.saveNote)}</span>
        </div>
      ) : null}
      {dialog === 'payment' ? (
        <PaymentDialog target={{ bookingId: b.id, issue: b.issue, sponsor, amountCents: b.priceCents, currency: b.currency, payment: b.payment }} onClose={() => setDialog(undefined)} />
      ) : null}
      {dialog === 'share' ? <ShareDialog bookingId={b.id} issue={b.issue} sponsor={sponsor} onClose={() => setDialog(undefined)} /> : null}
      {dialog === 'cancel' && live ? (
        <CancelDialog
          target={{ bookingId: b.id, issue: b.issue, sponsor, status: b.status as 'hold' | 'booked', holdUntil: b.holdUntil, amountCents: b.priceCents, currency: b.currency }}
          onClose={() => setDialog(undefined)}
          onDone={onClose}
        />
      ) : null}
      {dialog === 'link' ? <LinkCampaignDialog bookingId={b.id} issue={b.issue} current={b.campaign?.byHand ? b.campaign.id : undefined} onClose={() => setDialog(undefined)} /> : null}
    </>
  )
}

/** Booking: the sponsor, the price, the status (and hold date) and the campaign. */
function BookingSection({ b, onLink, onCancel }: { b: Detail; onLink: () => void; onCancel: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const qc = useQueryClient()
  const update = $api.useMutation('put', `${WS}/sponsor-bookings/{id}`)
  const ro = !b.canManage
  const live = b.status === 'hold' || b.status === 'booked'
  const [hold, setHold] = useState(b.holdUntil ? inputDay(b.holdUntil) : '')
  const [now] = useState(() => Date.now())
  const save = (body: Schemas['BookingInput'], done?: string) =>
    update.mutate(
      { params: { path: { workspace: slug, id: b.id } }, body },
      {
        onSuccess: (x) => {
          qc.setQueryData(['get', `${WS}/sponsor-bookings/{id}`, { params: { path: { workspace: slug, id: b.id } } }], x)
          refresh()
          if (done) toast({ kind: 'success', title: done })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const onStatus = (v: string) => {
    if (v === 'open' || v === 'canceled') return onCancel()
    if (v === 'booked') return save({ status: 'booked' }, t(mb.bookedToast, { issue: b.issue, sponsor: b.sponsor?.name ?? '' }))
    if (v === 'hold') {
      const until = hold || inputDay(new Date(Math.min(Date.now() + 5 * 86_400_000, new Date(b.issueOn).getTime() - 86_400_000)))
      setHold(until)
      save({ status: 'hold', holdUntil: until })
    }
  }
  const sp = b.sponsor
  const camp = b.campaign
  return (
    <Section title={t(mb.secBooking)}>
      <dl className="sp-kv">
        <dt>{t(mb.sponsor)}</dt>
        <dd>
          {sp ? (
            <span className="inline-flex flex-wrap items-center gap-2">
              <Initials initials={sp.initials} logo={sp.logoUrl} size="sm" />
              <Link to="/w/$workspace/earn/sponsors" params={{ workspace: slug }} search={{ tab: 'sponsors' }} className="font-semibold">
                {sp.name}
              </Link>
              {sp.contactName || sp.contactEmail ? (
                <span className="k">{sp.contactName && sp.contactEmail ? t(mb.contactLine, { name: sp.contactName, email: sp.contactEmail }) : sp.contactName || sp.contactEmail}</span>
              ) : null}
            </span>
          ) : (
            '–'
          )}
        </dd>
        <dt>{t(mb.price)}</dt>
        <dd>
          <span className="ea-num">{money(b.priceCents, b.currency)}</span>
          {b.priceCents === b.slotPriceCents ? <span className="k"> {t(mb.slotDefault)}</span> : null}
        </dd>
        <dt>{t(mb.status)}</dt>
        <dd>
          {live && !ro ? (
            <span className="inline-flex flex-wrap items-center gap-2">
              <select className="in h-8 w-[150px]" aria-label={t(mb.statusSelect)} value={b.status} disabled={update.isPending} onChange={(e) => onStatus(e.target.value)}>
                <option value="open">{t(mb.statusOpen)}</option>
                <option value="hold">{t(mb.statusHold)}</option>
                <option value="booked">{t(mb.statusBooked)}</option>
                <option value="canceled">{t(mb.statusCanceled)}</option>
              </select>
              {b.status === 'hold' ? (
                <label className="inline-flex items-center gap-2">
                  <span className="k">{t(mb.holdUntilLabel)}</span>
                  <input
                    className="in h-8 w-[150px]"
                    type="date"
                    value={hold}
                    min={inputDay(new Date(now + 86_400_000))}
                    max={inputDay(new Date(new Date(b.issueOn).getTime() - 86_400_000))}
                    onChange={(e) => setHold(e.target.value)}
                    onBlur={() => hold && b.holdUntil && hold !== inputDay(b.holdUntil) && save({ status: 'hold', holdUntil: hold })}
                  />
                </label>
              ) : null}
            </span>
          ) : (
            <Badge tone={b.status === 'booked' ? 'ok' : b.status === 'hold' ? 'info' : 'neutral'}>
              {b.status === 'booked' ? t(mb.statusBooked) : b.status === 'hold' ? t(mb.statusHold) : b.status === 'canceled' ? t(mb.statusCanceled) : t(mb.statusSent)}
            </Badge>
          )}
        </dd>
        <dt>{t(mb.campaign)}</dt>
        <dd>
          {camp ? (
            <>
              <CampaignLink id={camp.id} name={camp.name} status={camp.status} />
              <span className="k"> {campaignWords(t, camp)}</span>
              {camp.byHand ? <span className="k"> · {t(mb.campaignByHand)}</span> : null}
            </>
          ) : (
            <span className="k">{t(mb.campaignNone)}</span>
          )}
          {!ro && live && (!camp || camp.byHand) ? (
            <>
              {' '}
              <button type="button" className="sp-lnk align-baseline" onClick={onLink}>
                {t(mb.linkByHand)}
              </button>
            </>
          ) : null}
        </dd>
      </dl>
    </Section>
  )
}

/** "· draft by Sofia Rossi · Tue 21 Oct, 09:00" */
function campaignWords(t: ReturnType<typeof useT>, c: Schemas['SponsorCampaignRef']): string {
  const when = c.at ? `${shortDate(c.at)}, ${clock(c.at)}` : ''
  if (c.author && when) return t(mb.campaignBy, { status: c.status, author: c.author, when })
  if (when) return t(mb.campaignNoAuthor, { status: c.status, when })
  if (c.author) return t(mb.campaignNoWhen, { status: c.status, author: c.author })
  return t(mb.campaignNoAuthor, { status: c.status, when: '' }).replace(/ · $/, '')
}

function complete(c: Creative): boolean {
  return Boolean(c.headline?.trim() && c.text?.trim() && c.link?.trim())
}

/** The Sponsor block as the issue shows it: a placeholder while the creative is waiting. */
export function BlockPreview({ creative, sponsor, initials, logo, label, waiting, due }: { creative: Creative; sponsor: string; initials: string; logo?: string; label: string; waiting?: boolean; due?: string }) {
  const t = useT()
  if (waiting) {
    return (
      <div className="sp-prev">
        <div className="sp-block wait">
          <Initials initials={initials} logo={logo} size="xl" />
          <div className="tx">
            <span className="lab">{t(mb.placeholderLabel, { sponsor })}</span>
            <p>{t(mb.waitingText, { date: due ? shortDate(due) : '' })}</p>
          </div>
        </div>
      </div>
    )
  }
  const img = creative.image
  return (
    <div className="sp-prev">
      <div className={`sp-block${img ? ' top' : ''}`}>
        {img ? <img className="img" src={img} alt={sponsor} /> : <Initials initials={initials} logo={creative.logo || logo} size="xl" />}
        <div className="tx">
          <span className="lab">{label}</span>
          {creative.headline ? <span className="hl">{creative.headline}</span> : null}
          {creative.text ? <p>{creative.text}</p> : null}
          {creative.button ? <span className="cta">{creative.button}</span> : null}
        </div>
      </div>
    </div>
  )
}

/** Creative: its fields save as you go; Mark as ready once it has a headline, text and link. */
function CreativeEditor({ b }: { b: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const refresh = useRefresh()
  const [c, setC] = useState<Creative>(b.creative)
  const [utmEdit, setUtmEdit] = useState(false)
  const [uploading, setUploading] = useState(false)
  const file = useRef<HTMLInputElement>(null)
  const saved = useRef<string>(JSON.stringify(b.creative))
  const save = $api.useMutation('put', `${WS}/sponsor-bookings/{id}/creative`)
  const ready = $api.useMutation('post', `${WS}/sponsor-bookings/{id}/creative/ready`)
  const key = ['get', `${WS}/sponsor-bookings/{id}`, { params: { path: { workspace: slug, id: b.id } } }]
  const persist = (next: Creative) => {
    const raw = JSON.stringify(next)
    if (raw === saved.current) return
    saved.current = raw
    save.mutate(
      { params: { path: { workspace: slug, id: b.id } }, body: next },
      {
        onSuccess: (x) => {
          qc.setQueryData(key, x)
          refresh()
        },
        onError: (e) => {
          saved.current = ''
          toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  }
  const set = (patch: Partial<Creative>) => setC((x) => ({ ...x, ...patch }))
  const blur = () => persist(c)
  const upload = async (f: File | undefined) => {
    if (!f) return
    setUploading(true)
    try {
      const wide = await isWide(f)
      const fd = new FormData()
      fd.append('file', f)
      const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/sponsor-bookings/${b.id}/creative/image`, {
        method: 'POST',
        body: fd,
        credentials: 'same-origin',
        headers: { Accept: 'application/json' },
      })
      const body = (await r.json().catch(() => ({}))) as { url?: string; detail?: string; title?: string; errors?: { message?: string }[] }
      if (!r.ok || !body.url) throw new Error(body.errors?.[0]?.message || body.detail || body.title || `HTTP ${r.status}`)
      const next = wide ? { ...c, image: body.url } : { ...c, logo: body.url }
      setC(next)
      persist(next)
      toast({ kind: 'success', title: t(mb.uploaded) })
    } catch (e) {
      toast({ kind: 'error', title: t(mb.uploadFailed), detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setUploading(false)
      if (file.current) file.current.value = ''
    }
  }
  const markReady = () =>
    ready.mutate(
      { params: { path: { workspace: slug, id: b.id } } },
      {
        onSuccess: (x) => {
          qc.setQueryData(key, x)
          refresh()
          toast({ kind: 'success', title: t(mb.creativeMarkedReady, { issue: b.issue }) })
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const days = Math.max(0, b.creativeDue ? Math.round((new Date(b.issueOn).getTime() - new Date(b.creativeDue).getTime()) / 86_400_000) : 0)
  const isReady = b.creativeStatus === 'ready'
  const ok = complete(c)
  const sp = b.sponsor
  const image = c.image || c.logo
  const utm = c.utm ?? {}
  const textLen = [...(c.text ?? '')].length
  return (
    <Section title={t(mb.secCreative)} aside={isReady ? <Badge tone="ok">{t(mb.creativeReady)}</Badge> : <Badge tone="warn">{t(mb.creativeWaiting)}</Badge>}>
      <span className="k" style={{ color: 'var(--ink-nav)' }}>
        {isReady
          ? t(mb.creativeReadyText, { when: b.readyAt ? `${dayMonthDate(b.readyAt)}, ${clock(b.readyAt)}` : '', issue: b.issue })
          : b.creativeDue && isPastDay(b.creativeDue)
            ? t(mb.creativeLateText, { date: shortDate(b.creativeDue), issue: b.issue })
            : t(mb.creativeDueText, { date: b.creativeDue ? shortDate(b.creativeDue) : '', days, issue: b.issue })}
      </span>
      <div className="flex flex-col gap-3">
        <div className={`sp-drop${uploading ? ' on' : ''}`}>
          {image ? (
            <img src={image} alt="" />
          ) : (
            <span style={{ color: 'var(--ink-3)', display: 'inline-flex' }}>
              <Icon name="image" size={18} />
            </span>
          )}
          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="text-[13px] font-semibold">{t(mb.logoOrImage)}</span>
            <span className="k">{t(mb.logoHint)}</span>
          </span>
          <input ref={file} type="file" accept="image/png,image/jpeg,image/gif" className="vh" aria-label={t(mb.logoOrImage)} onChange={(e) => void upload(e.target.files?.[0])} />
          <Button size="sm" loading={uploading} onClick={() => file.current?.click()}>
            {image ? t(mb.replace) : t(mb.upload)}
          </Button>
          {image ? (
            <Button
              size="sm"
              onClick={() => {
                const next = { ...c, logo: '', image: '' }
                setC(next)
                persist(next)
              }}
            >
              {t(mb.removeImage)}
            </Button>
          ) : null}
        </div>
        <label className="lbl">
          {t(mb.headline)}
          <Input value={c.headline ?? ''} maxLength={120} onChange={(e) => set({ headline: e.target.value })} onBlur={blur} />
        </label>
        <Field label={t(mb.text)} hint={t(mb.textCount, { count: textLen })}>
          <Textarea value={c.text ?? ''} maxLength={280} placeholder={t(mb.textPlaceholder)} rows={3} onChange={(e) => set({ text: e.target.value })} onBlur={blur} />
        </Field>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(200px,100%),1fr))] gap-3">
          <label className="lbl">
            {t(mb.buttonText)}
            <Input value={c.button ?? ''} maxLength={40} placeholder={t(mb.buttonPlaceholder)} onChange={(e) => set({ button: e.target.value })} onBlur={blur} />
          </label>
          <label className="lbl">
            {t(mb.link)}
            <Input value={c.link ?? ''} maxLength={1000} placeholder={t(mb.linkPlaceholder)} onChange={(e) => set({ link: e.target.value })} onBlur={blur} />
          </label>
        </div>
        <div className="flex flex-col gap-1.5">
          <span className="k sp-sub">{t(mb.utmTitle)}</span>
          {utmEdit ? (
            <div className="grid grid-cols-[repeat(auto-fit,minmax(min(150px,100%),1fr))] items-end gap-2">
              {(['source', 'medium', 'campaign'] as const).map((k) => (
                <label key={k} className="lbl">
                  <span className="mono">{k === 'source' ? t(mb.utmSource) : k === 'medium' ? t(mb.utmMedium) : t(mb.utmCampaign)}</span>
                  <Input value={utm[k] ?? ''} maxLength={100} onChange={(e) => set({ utm: { ...utm, [k]: e.target.value } })} />
                </label>
              ))}
              <Button
                size="sm"
                onClick={() => {
                  setUtmEdit(false)
                  persist(c)
                }}
              >
                {t(mb.utmDone)}
              </Button>
            </div>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              {utm.source ? <span className="sp-utm">utm_source={utm.source}</span> : null}
              {utm.medium ? <span className="sp-utm">utm_medium={utm.medium}</span> : null}
              {utm.campaign ? <span className="sp-utm">utm_campaign={utm.campaign}</span> : null}
              <Button size="xs" onClick={() => setUtmEdit(true)}>
                {t(mb.utmEdit)}
              </Button>
            </div>
          )}
          <span className="k">{t(mb.utmNote)}</span>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        <span className="k sp-sub">{t(mb.previewTitle)}</span>
        <BlockPreview creative={c} sponsor={sp?.name ?? ''} initials={sp?.initials ?? ''} logo={sp?.logoUrl} label={b.label} waiting={!ok} due={b.creativeDue} />
      </div>
      {isReady ? null : (
        <div className="flex flex-wrap items-center gap-2">
          <Button size="sm" variant="primary" loading={ready.isPending} disabled={!ok} aria-describedby={ok ? undefined : `sp-why-${b.id}`} onClick={markReady}>
            {t(mb.markReady)}
          </Button>
          {sp?.contactEmail ? (
            <ButtonAnchor size="sm" href={`mailto:${sp.contactEmail}`} newTab={false}>
              {t(mb.emailContact, { name: sp.contactName || sp.contactEmail })}
            </ButtonAnchor>
          ) : null}
          {ok ? null : (
            <span className="k" id={`sp-why-${b.id}`}>
              {t(mb.markReadyWhy)}
            </span>
          )}
        </div>
      )}
    </Section>
  )
}

/** Whether an image is a banner (twice as wide as tall) rather than a logo. */
function isWide(f: File): Promise<boolean> {
  return new Promise((resolve) => {
    const url = URL.createObjectURL(f)
    const img = new Image()
    img.onload = () => {
      resolve(img.naturalWidth >= img.naturalHeight * 2)
      URL.revokeObjectURL(url)
    }
    img.onerror = () => {
      resolve(false)
      URL.revokeObjectURL(url)
    }
    img.src = url
  })
}

/** The creative, read-only. */
function CreativeView({ b }: { b: Detail }) {
  const t = useT()
  const sp = b.sponsor
  const ok = complete(b.creative)
  return (
    <Section title={t(mb.secCreative)} aside={b.creativeStatus === 'ready' ? <Badge tone="ok">{t(mb.creativeReady)}</Badge> : <Badge tone="warn">{t(mb.creativeWaiting)}</Badge>}>
      <BlockPreview creative={b.creative} sponsor={sp?.name ?? ''} initials={sp?.initials ?? ''} logo={sp?.logoUrl} label={b.label} waiting={!ok} due={b.creativeDue} />
    </Section>
  )
}

/** Payment: Unpaid · due …, Overdue, or Paid with its note and who recorded it. */
function PaymentSection({ b, onRecord }: { b: Detail; onRecord: () => void }) {
  const t = useT()
  const p = b.payment
  const ro = !b.canManage
  const sp = b.sponsor
  if (p.status === 'paid') {
    return (
      <Section
        title={t(mb.secPayment)}
        aside={
          ro ? (
            <Badge tone="ok">{t(mb.paymentPaid)}</Badge>
          ) : (
            <Button size="sm" onClick={onRecord}>
              {t(mb.edit)}
            </Button>
          )
        }
      >
        <dl className="sp-kv">
          <dt>{t(mb.paidLabel)}</dt>
          <dd>
            <T msg={mb.paidValue} values={{ amount: <span className="ea-num">{money(p.paidCents ?? b.priceCents, b.currency, 'always')}</span>, date: p.paidOn ? day(p.paidOn) : '' }} />
          </dd>
          {p.note ? (
            <>
              <dt>{t(mb.noteLabel)}</dt>
              <dd>{p.note}</dd>
            </>
          ) : null}
          {p.recordedBy ? (
            <>
              <dt>{t(mb.recordedBy)}</dt>
              <dd>{t(mb.recordedByValue, { name: p.recordedBy, when: p.recordedAt ? `${dayMonthDate(p.recordedAt)}, ${clock(p.recordedAt)}` : '' })}</dd>
            </>
          ) : null}
        </dl>
      </Section>
    )
  }
  const terms = sp?.termsDays ?? 14
  return (
    <Section
      title={t(mb.secPayment)}
      aside={p.status === 'overdue' ? <Badge tone="warn">{t(mb.paymentOverdue)}</Badge> : <Badge tone="neutral">{t(mb.paymentUnpaid)}</Badge>}
    >
      <div className="flex flex-wrap items-center justify-between gap-2.5">
        <span className="flex flex-col gap-0.5">
          <span>
            <T
              msg={p.due ? (p.status === 'overdue' ? mb.paymentWasDue : mb.paymentDue) : mb.paymentNoDue}
              values={{ amount: money(b.priceCents, b.currency), date: p.due ? day(p.due) : '', b: (c) => <b className="ea-num font-semibold">{c}</b> }}
            />
          </span>
          {sp ? <span className="k">{t(mb.paymentTerms, { days: terms, sponsor: sp.name, ends: ends(sp.name) })}</span> : null}
        </span>
        {ro ? null : (
          <Button size="sm" onClick={onRecord}>
            <MoneyIcon />
            {t(mb.recordPayment)}
          </Button>
        )}
      </div>
    </Section>
  )
}

/** Notes: saved when the field loses focus. */
function NotesSection({ b }: { b: Detail }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [notes, setNotes] = useState(b.notes)
  const update = $api.useMutation('put', `${WS}/sponsor-bookings/{id}`)
  if (!b.canManage) {
    if (!b.notes) return null
    return (
      <Section title={t(mb.secNotes)}>
        <p className="m-0 whitespace-pre-wrap">{b.notes}</p>
      </Section>
    )
  }
  return (
    <Section title={t(mb.secNotes)} aside={update.isPending ? <span className="k">{t(mb.saving)}</span> : null}>
      <Textarea
        aria-label={t(mb.secNotes)}
        value={notes}
        maxLength={2000}
        rows={3}
        className="min-h-16"
        onChange={(e) => setNotes(e.target.value)}
        onBlur={() => {
          if (notes === b.notes) return
          update.mutate(
            { params: { path: { workspace: slug, id: b.id } }, body: { notes } },
            {
              onSuccess: (x) => qc.setQueryData(['get', `${WS}/sponsor-bookings/{id}`, { params: { path: { workspace: slug, id: b.id } } }], x),
              onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
            },
          )
        }}
      />
    </Section>
  )
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : typeof v === 'number' ? String(v) : ''
}

/** Activity: what happened to the booking, oldest first. */
function ActivitySection({ b }: { b: Detail }) {
  const t = useT()
  if (!b.activity.length) return null
  return (
    <Section title={t(mb.secActivity)}>
      <ol className="ea-tl">
        {[...b.activity].sort((x, y) => new Date(x.at).getTime() - new Date(y.at).getTime()).map((e, i) => {
          const line = eventLine(t, e, b.currency)
          return (
            <li key={i}>
              <span className={`ic ${line.tone}`} aria-hidden="true">
                <Icon name={line.icon} size={14} />
              </span>
              <span className="flex min-w-0 flex-col gap-0.5">
                <span className="t">{line.title}</span>
                {line.detail ? <span className="d">{line.detail}</span> : null}
              </span>
              <span className="w">{dayMonthDate(e.at)}</span>
            </li>
          )
        })}
      </ol>
    </Section>
  )
}

function eventLine(t: ReturnType<typeof useT>, e: Schemas['SponsorBookingEvent'], currency: string): { title: string; detail?: string; icon: 'check' | 'more' | 'close' | 'clock' | 'link' | 'edit'; tone: string } {
  const actor = e.actor || t(mb.someone)
  const when = `${dayMonthDate(e.at)}, ${clock(e.at)}`
  const d = e.data ?? {}
  const amount = typeof d.priceCents === 'number' ? money(d.priceCents, currency) : typeof d.amountCents === 'number' ? money(d.amountCents, currency, 'always') : ''
  switch (e.kind) {
    case 'booked': {
      const title = d.fromHold ? t(mb.evBookedFromHold, { actor }) : d.fromProposal ? t(mb.evBookedFromProposal, { actor, agent: str(d.fromProposal) }) : t(mb.evBooked, { actor })
      const withIssues = Array.isArray(d.with) ? (d.with as number[]).map((n) => `#${n}`) : []
      const detail =
        withIssues.length > 1 && typeof d.after === 'number'
          ? t(mb.evBookedWith, { when, issues: andList(withIssues), after: d.after })
          : t(mb.evBookedWithout, { when, sponsor: str(d.sponsor), amount })
      return { title, detail, icon: 'check', tone: '' }
    }
    case 'held':
      return { title: t(mb.evHeld, { actor }), detail: t(mb.evHeldDetail, { when, date: d.holdUntil ? shortDate(`${str(d.holdUntil)}T12:00:00`) : '' }), icon: 'clock', tone: 'info' }
    case 'hold_changed':
      return { title: t(mb.evHoldChanged, { actor }), detail: t(mb.evHeldDetail, { when, date: d.holdUntil ? shortDate(`${str(d.holdUntil)}T12:00:00`) : '' }), icon: 'clock', tone: 'info' }
    case 'changed':
      return { title: t(mb.evChanged, { actor }), detail: t(mb.evBookedWithout, { when, sponsor: str(d.sponsor), amount }), icon: 'edit', tone: 'mut' }
    case 'campaign_linked':
      return { title: t(mb.evCampaignLinked, { actor }), detail: d.byHand ? t(mb.evCampaignByHand, { when, campaign: str(d.campaign) }) : t(mb.evCampaignByDate, { when }), icon: 'link', tone: 'mut' }
    case 'creative_ready':
      return { title: t(mb.evCreativeReady, { actor }), detail: when, icon: 'check', tone: '' }
    case 'creative_requested':
      return { title: t(mb.evCreativeRequested), detail: t(mb.evCreativeRequestedDetail, { date: dayMonthDate(e.at), actor, contact: str(d.contact) }), icon: 'more', tone: 'mut' }
    case 'paid':
      return { title: t(mb.evPaid, { actor }), detail: t(mb.evPaidDetail, { when, amount, date: d.paidOn ? day(`${str(d.paidOn)}T12:00:00`) : '' }), icon: 'check', tone: '' }
    case 'payment_cleared':
      return { title: t(mb.evPaymentCleared, { actor }), detail: when, icon: 'edit', tone: 'mut' }
    case 'canceled':
      return { title: t(mb.evCanceled, { actor }), detail: d.reason ? t(mb.evReason, { when, reason: str(d.reason) }) : when, icon: 'close', tone: 'bad' }
    case 'released':
      return { title: t(mb.evReleased, { actor }), detail: d.reason ? t(mb.evReason, { when, reason: str(d.reason) }) : when, icon: 'close', tone: 'mut' }
    case 'proposed':
      return { title: t(mb.evProposed, { actor }), detail: d.note ? t(mb.evReason, { when, reason: str(d.note) }) : when, icon: 'more', tone: 'mut' }
    case 'hold_lapsed':
      return { title: t(mb.evHoldLapsed), detail: t(mb.evHoldLapsedDetail, { date: d.holdUntil ? shortDate(`${str(d.holdUntil)}T12:00:00`) : dayMonthDate(e.at) }), icon: 'clock', tone: 'mut' }
    case 'report_shared':
      return {
        title: t(mb.evReportShared, { actor }),
        detail: d.expiresAt ? t(mb.evReportSharedDetail, { when, date: dayMonthYear(str(d.expiresAt)) }) : t(mb.evReportSharedNever, { when }),
        icon: 'link',
        tone: 'info',
      }
    case 'report_link_revoked':
      return { title: t(mb.evLinkRevoked, { actor }), detail: when, icon: 'close', tone: 'mut' }
  }
  return { title: t(mb.evOther, { kind: e.kind.replace(/_/g, ' '), actor }), detail: when, icon: 'more', tone: 'mut' }
}

/** After the send: the sponsor report's numbers and the creative as sent. */
function ReportSection({ b }: { b: Detail }) {
  const t = useT()
  const r = b.report!
  const link = b.shares[0]?.url
  const finalAt = r.finalAt
  const clicks = r.links.map((l) => t(mb.linkKind, { kind: l.kind, count: count(l.clicks) })).join(' · ')
  return (
    <Section
      title={t(mb.secReport)}
      aside={
        link ? (
          <a href={link} target="_blank" rel="noopener noreferrer" className="sp-lnk">
            {t(mb.openReport)}
            <Icon name="external" size={13} />
          </a>
        ) : null
      }
    >
      <span className="k" style={{ color: 'var(--ink-nav)' }}>
        {finalAt ? (r.final ? t(mb.finalSince, { when: `${shortDate(finalAt)}, ${clock(finalAt)}` }) : t(mb.comingIn, { when: `${shortDate(finalAt)}, ${clock(finalAt)}` })) : null}
      </span>
      <div className="sp-mini">
        <Mini label={t(mb.mDelivered)} value={count(r.delivered)} sub={t(mb.mDeliveredSub)} />
        <Mini label={t(mb.mOpens)} value={count(r.humanOpens)} sub={t(mb.mOpensSub, { rate: pct(r.humanOpenRate, 1) })} />
        <Mini label={t(mb.mClickers)} value={count(r.uniqueClickers)} sub={t(mb.mClickersSub)} />
        <Mini label={t(mb.mRate)} value={pct(r.clickRate, 1)} sub={t(mb.mRateSub)} />
        <Mini label={t(mb.mCto)} value={pct(r.clickToOpen, 1)} sub={t(mb.mCtoSub)} />
        <Mini label={t(mb.mClicks)} value={count(r.clicks)} sub={clicks} />
      </div>
      <div className="flex flex-col gap-2">
        <span className="k sp-sub">{t(mb.asSent)}</span>
        <BlockPreview creative={r.creative} sponsor={r.sponsorName} initials={r.initials} label={r.label} />
      </div>
    </Section>
  )
}

function Mini({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div>
      <span className="k">{label}</span>
      <span className="n">{value}</span>
      {sub ? <span className="k x">{sub}</span> : null}
    </div>
  )
}

/** Read-only link: Copy, Revoke, and who shared it. */
function ShareSection({ b, onShare }: { b: Detail; onShare: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const [revoking, setRevoking] = useState<string>()
  const revoke = $api.useMutation('delete', `${WS}/sponsor-report-shares/{id}`)
  const sponsor = b.sponsor?.name ?? ''
  return (
    <Section title={t(mb.secShare)}>
      {b.shares.length ? (
        b.shares.map((s) => (
          <div key={s.id} className="flex flex-col gap-1.5">
            <div className="sp-copyrow">
              <input className="in" value={(s.url ?? '').replace(/^https?:\/\//, '')} readOnly aria-label={t(mb.secShare)} />
              <Button size="sm" icon="copy" onClick={() => void copyText(s.url ?? '', t, toast)}>
                {t(mb.copy)}
              </Button>
              <Button size="sm" variant="danger" onClick={() => setRevoking(s.id)}>
                {t(mb.revoke)}
              </Button>
            </div>
            <span className="k">
              {s.expiresAt
                ? t(mb.shareLine, { name: s.createdBy, date: day(s.createdAt), expires: dayMonthYear(s.expiresAt), views: s.views })
                : t(mb.shareLineNever, { name: s.createdBy, date: day(s.createdAt), views: s.views })}
            </span>
          </div>
        ))
      ) : (
        <div className="flex flex-wrap items-center justify-between gap-2.5">
          <span className="k">{t(mb.shareNone, { sponsor })}</span>
          <Button size="sm" icon="link" onClick={onShare}>
            {t(mb.shareButton)}
          </Button>
        </div>
      )}
      <ConfirmDialog
        open={Boolean(revoking)}
        onOpenChange={(o) => !o && setRevoking(undefined)}
        badge={t(mb.revokeConfirm)}
        title={t(mb.revokeTitle)}
        description={t(mb.revokeText, { sponsor })}
        confirmLabel={t(mb.revokeConfirm)}
        onConfirm={() =>
          revoke.mutateAsync({ params: { path: { workspace: slug, id: revoking! } } }).then(() => {
            refresh()
            setRevoking(undefined)
            toast({ kind: 'success', title: t(mb.revoked) })
          })
        }
      />
    </Section>
  )
}

/** An agent's proposal: the note, then who, what and the status once confirmed. */
function ProposalBody({ p, slotPrice, currency }: { p: Proposal; slotPrice: number; currency: string }) {
  const t = useT()
  const when = `${dayMonthDate(p.at)}, ${clock(p.at)}`
  return (
    <>
      <div className="note n" role="note">
        <NoteIcon tone="neutral" />
        <div className="min-w-0 flex-[1_1_320px]">
          <T msg={p.note ? mb.proposalNote : mb.proposalNoteNoNote} values={{ agent: p.agent, when, note: p.note, permission: <span className="mono">sponsors:manage</span> }} />
        </div>
      </div>
      <dl className="sp-kv">
        <dt>{t(mb.sponsor)}</dt>
        <dd>
          {p.sponsor ? (
            <span className="sp-who">
              <Initials initials={p.sponsor.initials} logo={p.sponsor.logoUrl} />
              <span className="font-semibold">{p.sponsor.name}</span>
            </span>
          ) : (
            '–'
          )}
        </dd>
        <dt>{t(mb.price)}</dt>
        <dd>
          <span className="ea-num">{money(p.priceCents, currency)}</span>
          {p.priceCents === slotPrice ? <span className="k"> {t(mb.slotDefault)}</span> : null}
        </dd>
        <dt>{t(mb.status)}</dt>
        <dd>{t(mb.onceConfirmed)}</dd>
      </dl>
    </>
  )
}

function ProposalFoot({ b, onClose }: { b: { id: string; issue: number; sponsor?: { name: string } }; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const confirm = $api.useMutation('post', `${WS}/sponsor-bookings/{id}/confirm`)
  const dismiss = $api.useMutation('post', `${WS}/sponsor-bookings/{id}/cancel`)
  return (
    <div className="ea-sheet-foot justify-end">
      <Button
        size="sm"
        loading={dismiss.isPending}
        onClick={() =>
          dismiss.mutate(
            { params: { path: { workspace: slug, id: b.id } }, body: {} },
            {
              onSuccess: () => {
                refresh()
                toast({ kind: 'success', title: t(mb.dismissed, { issue: b.issue }) })
                onClose()
              },
              onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
            },
          )
        }
      >
        {t(mb.dismiss)}
      </Button>
      <Button
        size="sm"
        variant="primary"
        loading={confirm.isPending}
        onClick={() =>
          confirm.mutate(
            { params: { path: { workspace: slug, id: b.id } } },
            {
              onSuccess: (x) => {
                refresh()
                toast({ kind: 'success', title: t(mb.bookedToast, { issue: b.issue, sponsor: x.sponsor?.name ?? '' }) })
              },
              onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
            },
          )
        }
      >
        {t(mb.confirm)}
      </Button>
    </div>
  )
}

/** An open issue's panel. */
function IssueBody({ d, library, onClose, onBooked }: { d: IssueDetail; library: SponsorItem[]; onClose: () => void; onBooked: (id: string) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const ro = !d.canManage
  const past = isPastDay(d.issueOn)
  const proposal = d.proposals[0]
  const [sponsor, setSponsor] = useState('')
  const [price, setPrice] = useState((d.priceCents / 100).toFixed(0))
  const [mode, setMode] = useState<'booked' | 'hold'>('booked')
  const lastHold = inputDay(new Date(new Date(d.issueOn).getTime() - 86_400_000))
  const [now] = useState(() => Date.now())
  const firstHold = inputDay(new Date(now + 86_400_000))
  const suggested = inputDay(new Date(Math.min(now + 5 * 86_400_000, new Date(d.issueOn).getTime() - 86_400_000)))
  const [hold, setHold] = useState(suggested < firstHold ? firstHold : suggested)
  const [adding, setAdding] = useState(false)
  const [error, setError] = useState<Record<string, string>>({})
  const book = $api.useMutation('post', `${WS}/sponsor-slots/{id}/issues/{number}/book`)
  const chosen = library.find((s) => s.id === sponsor)
  const terms = chosen?.termsDays ?? d.termsDays
  const paymentDue = new Date(new Date(d.issueOn).getTime() + terms * 86_400_000).toISOString()
  const creativeDays = Math.max(0, daysUntil(d.issueOn) - daysUntil(d.creativeDue))
  const submit = () => {
    const cents = parseAmount(price, d.currency)
    if (!sponsor) return setError({ sponsorId: t(mb.chooseSponsor) })
    if (cents === undefined) return setError({ priceCents: t(mb.amountInvalid) })
    book.mutate(
      { params: { path: { workspace: slug, id: d.slotId, number: d.issue } }, body: { sponsorId: sponsor, priceCents: cents, status: mode, holdUntil: mode === 'hold' ? hold : undefined } },
      {
        onSuccess: (x) => {
          refresh()
          const name = x.sponsor?.name ?? chosen?.name ?? ''
          toast({ kind: 'success', title: mode === 'hold' ? t(mb.heldToast, { issue: d.issue, sponsor: name, date: x.holdUntil ? shortDate(x.holdUntil) : hold }) : t(mb.bookedToast, { issue: d.issue, sponsor: name }) })
          onBooked(x.id)
        },
        onError: (e) => {
          const f = (e as { problem?: { errors?: { location?: string; message?: string }[] } }).problem?.errors
          if (f?.length) {
            const next: Record<string, string> = {}
            for (const x of f) next[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
            setError(next)
          } else toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  }
  const title = t(mb.title, { issue: d.issue, date: shortDate(d.issueOn), slot: d.slotName })
  const badges = (
    <>
      <OpenBadge />
      {proposal ? <Badge tone="purple">{t(mb.proposedBadge)}</Badge> : null}
    </>
  )
  if (proposal) {
    return (
      <>
        <Head eyebrow={t(mb.eyebrowOpen, { list: d.listName })} title={title} badges={badges} onClose={onClose} />
        <div className="ea-sheet-body">
          {ro ? <ReadOnly helpers={d.helpers} /> : null}
          <ProposalBody p={proposal} slotPrice={d.priceCents} currency={d.currency} />
        </div>
        {ro ? null : <ProposalFoot b={{ id: proposal.id, issue: d.issue, sponsor: proposal.sponsor }} onClose={onClose} />}
      </>
    )
  }
  const dates = (
    <Section title={t(mb.secDates)}>
      <dl className="sp-kv">
        <dt>{t(mb.creativeDue)}</dt>
        <dd>{t(mb.creativeDueValue, { date: shortDate(d.creativeDue), days: creativeDays })}</dd>
        <dt>{t(mb.paymentDueLabel)}</dt>
        <dd>{t(mb.paymentDueValue, { date: shortDate(paymentDue), days: terms })}</dd>
        <dt>{t(mb.campaign)}</dt>
        <dd>
          {d.campaign ? (
            <>
              <CampaignLink id={d.campaign.id} name={d.campaign.name} status={d.campaign.status} />
              <span className="k"> {campaignWords(t, d.campaign)}</span>
            </>
          ) : (
            <span className="k">{t(mb.campaignOpen)}</span>
          )}
        </dd>
      </dl>
    </Section>
  )
  return (
    <>
      <Head eyebrow={t(mb.eyebrowOpen, { list: d.listName })} title={title} badges={badges} onClose={onClose} />
      <div className="ea-sheet-body">
        {ro ? <ReadOnly helpers={d.helpers} /> : null}
        {past ? (
          <div className="note n" role="note">
            <NoteIcon tone="neutral" />
            <div className="min-w-0 flex-[1_1_320px]">{t(mb.issuePast)}</div>
          </div>
        ) : ro ? null : (
          <Section title={t(mb.bookTitle, { issue: d.issue })}>
            <div className="grid grid-cols-[repeat(auto-fit,minmax(min(200px,100%),1fr))] gap-3">
              <label className="lbl">
                {t(mb.sponsor)}
                <select
                  className="in"
                  aria-label={t(mb.sponsor)}
                  aria-invalid={error.sponsorId ? true : undefined}
                  value={sponsor}
                  onChange={(e) => {
                    if (e.target.value === '+new') return setAdding(true)
                    setSponsor(e.target.value)
                    setError({})
                  }}
                >
                  <option value="">{t(mb.chooseSponsor)}</option>
                  {library.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                  <option value="+new">{t(mb.newSponsorOption)}</option>
                </select>
                {error.sponsorId ? <span className="err">{error.sponsorId}</span> : null}
              </label>
              <Field label={t(mb.price)} hint={t(mb.priceHint)} error={error.priceCents}>
                <AmountInput currency={d.currency} value={price} onChange={setPrice} />
              </Field>
            </div>
            <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
              <legend className="lbl mb-1.5">{t(mb.bookOrHold)}</legend>
              <label className={`sp-radio${mode === 'booked' ? ' on' : ''}`}>
                <input type="radio" name={`bk-${d.issue}`} checked={mode === 'booked'} onChange={() => setMode('booked')} />
                <span>{t(mb.booked)}</span>
              </label>
              <label className={`sp-radio${mode === 'hold' ? ' on' : ''}`}>
                <input type="radio" name={`bk-${d.issue}`} checked={mode === 'hold'} onChange={() => setMode('hold')} />
                <span>{t(mb.holdUntil)}</span>
                <input
                  className="in"
                  type="date"
                  aria-label={t(mb.holdUntil)}
                  value={hold}
                  min={firstHold}
                  max={lastHold}
                  onChange={(e) => {
                    setHold(e.target.value)
                    setMode('hold')
                  }}
                />
                <span className="k">{t(mb.thenOpens, { issue: d.issue })}</span>
              </label>
              {error.holdUntil ? <span className="err">{error.holdUntil}</span> : null}
            </fieldset>
          </Section>
        )}
        {dates}
      </div>
      {ro || past ? null : (
        <div className="ea-sheet-foot justify-end">
          <Button size="sm" onClick={onClose}>
            {t(mb.cancel)}
          </Button>
          <Button size="sm" variant="primary" loading={book.isPending} onClick={submit}>
            {mode === 'hold' ? t(mb.holdIssue, { issue: d.issue }) : t(mb.bookIssue, { issue: d.issue })}
          </Button>
        </div>
      )}
      {adding ? (
        <SponsorDialog
          onClose={() => setAdding(false)}
          onSaved={(s) => {
            setSponsor(s.id)
            setAdding(false)
          }}
        />
      ) : null}
    </>
  )
}
