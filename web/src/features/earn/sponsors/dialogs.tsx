import { useQueryClient } from '@tanstack/react-query'
import { useRef, useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input, Select, Textarea, Toggle } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { dayMonthYear, shortDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { copyText } from '../EarnPage'
import { money } from '../format'
import { mb } from './booking.messages'
import { md } from './dialogs.messages'
import { Initials, NoteIcon, WS, inputDay, sponsorKeys, type SlotItem, type SponsorItem, type SponsorPayment } from './parts'

// Currencies without minor units (Stripe's zero-decimal currencies).
const zeroDecimal = new Set(['bif', 'clp', 'djf', 'gnf', 'jpy', 'kmf', 'krw', 'mga', 'pyg', 'rwf', 'ugx', 'vnd', 'vuv', 'xaf', 'xof', 'xpf'])

/** "750", "1,500.50" → minor units; undefined when it isn't an amount. */
export function parseAmount(raw: string, currency: string): number | undefined {
  const s = raw.replace(/[\s,]/g, '').replace(/^[^\d.]+/, '')
  if (!/^\d+(\.\d{1,2})?$/.test(s)) return undefined
  const v = Number(s)
  return zeroDecimal.has(currency.toLowerCase()) ? Math.round(v) : Math.round(v * 100)
}

/** Minor units as an input shows them: "750.00". */
export function amountInput(minor: number, currency: string): string {
  return zeroDecimal.has(currency.toLowerCase()) ? String(minor) : (minor / 100).toFixed(2)
}

/** Field errors of a problem, by field ("body.name" → "name"). */
function fieldErrors(e: unknown): Record<string, string> | undefined {
  if (!(e instanceof ApiError) || !e.problem.errors?.length) return undefined
  const out: Record<string, string> = {}
  for (const x of e.problem.errors) out[(x.location ?? '').replace(/^body\./, '')] = x.message ?? ''
  return out
}

/** Every sponsor query is stale after a change. */
export function useRefresh() {
  const qc = useQueryClient()
  return () => {
    for (const k of sponsorKeys) void qc.invalidateQueries({ queryKey: k })
  }
}

function today(): string {
  return inputDay(new Date())
}

function plusDays(days: number): Date {
  return new Date(Date.now() + days * 86_400_000)
}

function Eyebrow({ children }: { children: ReactNode }) {
  return <span className="k">{children}</span>
}

export interface PaymentTarget {
  bookingId: string
  issue: number
  sponsor: string
  amountCents: number
  currency: string
  /** The payment recorded so far, to change it. */
  payment?: SponsorPayment
}

/** Record a payment (EarnBooking): amount, date and note, by hand. */
export function PaymentDialog({ target, onClose }: { target: PaymentTarget; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const paid = target.payment?.status === 'paid'
  const [amount, setAmount] = useState(amountInput(target.payment?.paidCents ?? target.amountCents, target.currency))
  const [on, setOn] = useState(target.payment?.paidOn ? inputDay(target.payment.paidOn) : today())
  const [note, setNote] = useState(target.payment?.note ?? '')
  const [error, setError] = useState<Record<string, string>>({})
  const save = $api.useMutation('put', `${WS}/sponsor-bookings/{id}/payment`)
  const submit = (clear?: boolean) => {
    const cents = parseAmount(amount, target.currency)
    if (!clear && cents === undefined) return setError({ amountCents: t(mb.amountInvalid) })
    save.mutate(
      { params: { path: { workspace: slug, id: target.bookingId } }, body: clear ? { paid: false } : { paid: true, paidOn: on, amountCents: cents, note } },
      {
        onSuccess: () => {
          refresh()
          toast({ kind: 'success', title: clear ? t(mb.paymentCleared, { issue: target.issue }) : t(mb.paymentRecorded, { issue: target.issue }) })
          onClose()
        },
        onError: (e) => {
          const f = fieldErrors(e)
          if (f) setError(f)
          else toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<Eyebrow>{t(mb.payEyebrow, { issue: target.issue })}</Eyebrow>}
      title={paid ? t(mb.payEditTitle) : t(mb.payTitle)}
      hideClose
      className="max-w-[560px]"
      footer={
        <>
          <span className="k mr-auto">{t(mb.auditNote)}</span>
          {paid ? (
            <Button variant="danger" onClick={() => submit(true)} disabled={save.isPending}>
              {t(mb.markUnpaid)}
            </Button>
          ) : null}
          <Button onClick={onClose}>{t(mb.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} onClick={() => submit()}>
            {t(mb.recordPayment)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span style={{ color: 'var(--ink-nav)' }}>{t(mb.payText, { issue: target.issue, sponsor: target.sponsor, amount: money(target.amountCents, target.currency) })}</span>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(min(170px,100%),1fr))] gap-3">
          <Field label={t(mb.amount)} error={error.amountCents}>
            <AmountInput currency={target.currency} value={amount} onChange={setAmount} />
          </Field>
          <Field label={t(mb.paidOn)} error={error.paidOn}>
            <Input type="date" value={on} max={today()} onChange={(e) => setOn(e.target.value)} />
          </Field>
        </div>
        <Field label={t(mb.payNote)} hint={t(mb.payNoteHint)} error={error.note}>
          <Input value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>
    </Dialog>
  )
}

/** "USD [750.00]": an amount with its currency before it. */
export function AmountInput({ currency, value, onChange, id, ...aria }: { currency: string; value: string; onChange: (v: string) => void; id?: string; 'aria-describedby'?: string; 'aria-invalid'?: boolean }) {
  return (
    <div className="flex items-center gap-1.5">
      <span className="k">{currency.toUpperCase()}</span>
      <Input id={id} inputMode="decimal" className="flex-1" value={value} onChange={(e) => onChange(e.target.value)} {...aria} />
    </div>
  )
}

/** Share a read-only link to one booking's report, or to every booking of a sponsor. */
export function ShareDialog({ bookingId, sponsorId, issue, sponsor, onClose }: { bookingId?: string; sponsorId?: string; issue?: number; sponsor: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const [expiry, setExpiry] = useState<'30' | '90' | 'date' | 'never'>('90')
  const [date, setDate] = useState(inputDay(plusDays(90)))
  const [link, setLink] = useState<string>()
  const forBooking = $api.useMutation('post', `${WS}/sponsor-bookings/{id}/shares`)
  const forSponsor = $api.useMutation('post', `${WS}/sponsors/{id}/shares`)
  const busy = forBooking.isPending || forSponsor.isPending
  const days = (): number => {
    if (expiry === 'never') return 0
    if (expiry === 'date') {
      const [y, m, d] = date.split('-').map(Number)
      const target = Date.UTC(y!, m! - 1, d!)
      const now = new Date()
      return Math.max(1, Math.round((target - Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())) / 86_400_000))
    }
    return Number(expiry)
  }
  const create = () => {
    const opts = {
      onSuccess: (x: Schemas['SponsorShareLink']) => {
        refresh()
        setLink(x.url)
        toast({ kind: 'success', title: t(mb.linkCreated) })
      },
      onError: (e: unknown) => toast({ kind: 'error', title: errorMessage(e) }),
    }
    if (bookingId) forBooking.mutate({ params: { path: { workspace: slug, id: bookingId } }, body: { expiresInDays: days() } }, opts)
    else if (sponsorId) forSponsor.mutate({ params: { path: { workspace: slug, id: sponsorId } }, body: { expiresInDays: days() } }, opts)
  }
  const ok = <path d="m3 8 3.5 3.5L13 5" />
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      eyebrow={<Eyebrow>{issue ? t(mb.shareEyebrow, { issue, sponsor }) : sponsor}</Eyebrow>}
      title={t(mb.shareTitle)}
      hideClose
      className="max-w-[600px]"
      footer={
        <>
          <span className="k mr-auto">
            <T msg={mb.shareFoot} values={{ permission: 'reports:share' }} />
          </span>
          <Button onClick={onClose}>{t(mb.done)}</Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span style={{ color: 'var(--ink-nav)' }}>{t(mb.shareText, { sponsor })}</span>
        <div className="flex flex-wrap items-end gap-3">
          <Field label={t(mb.expires)}>
            <Select className="w-[260px] max-w-full" value={expiry} disabled={Boolean(link)} onChange={(e) => setExpiry(e.target.value as typeof expiry)}>
              <option value="30">{t(mb.in30, { date: dayMonthYear(plusDays(30)) })}</option>
              <option value="90">{t(mb.in90, { date: dayMonthYear(plusDays(90)) })}</option>
              <option value="date">{t(mb.onDate)}</option>
              <option value="never">{t(mb.never)}</option>
            </Select>
          </Field>
          {expiry === 'date' ? (
            <Field label={t(mb.expiresOn)}>
              <Input type="date" value={date} min={inputDay(plusDays(1))} disabled={Boolean(link)} onChange={(e) => setDate(e.target.value)} />
            </Field>
          ) : null}
        </div>
        <div className="sp-sees">
          <span className="text-[13px] font-semibold">{t(mb.seesTitle)}</span>
          <div>
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              {ok}
            </svg>
            <span>{t(mb.sees1)}</span>
          </div>
          <div>
            <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="#15803D" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              {ok}
            </svg>
            <span>{t(mb.sees2)}</span>
          </div>
          <div>
            <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="#991B1B" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M4 4l8 8M12 4l-8 8" />
            </svg>
            <span>{t(mb.sees3)}</span>
          </div>
        </div>
        <div className="sp-copyrow">
          <input className="in" value={link ? link.replace(/^https?:\/\//, '') : ''} readOnly aria-label={t(mb.newLink)} placeholder="…/r/…" />
          {link ? (
            <Button variant="primary" icon="copy" onClick={() => void copyText(link, t, toast)}>
              {t(mb.copy)}
            </Button>
          ) : (
            <Button variant="primary" icon="link" loading={busy} onClick={create}>
              {t(mb.createLink)}
            </Button>
          )}
        </div>
      </div>
    </Dialog>
  )
}

export interface CancelTarget {
  bookingId: string
  issue: number
  sponsor: string
  status: 'hold' | 'booked' | 'proposed'
  holdUntil?: string
  amountCents: number
  currency: string
}

/** Cancel the booking for #145? The Sponsor block is hidden and the issue opens again. */
export function CancelDialog({ target, onClose, onDone }: { target: CancelTarget; onClose: () => void; onDone?: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const [reason, setReason] = useState('')
  const cancel = $api.useMutation('post', `${WS}/sponsor-bookings/{id}/cancel`)
  const submit = () =>
    cancel.mutate(
      { params: { path: { workspace: slug, id: target.bookingId } }, body: { reason } },
      {
        onSuccess: () => {
          refresh()
          toast({ kind: 'success', title: t(mb.canceledToast, { issue: target.issue }) })
          onClose()
          onDone?.()
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const amount = money(target.amountCents, target.currency)
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      dismissible={false}
      eyebrow={<Eyebrow>{t(mb.cancelEyebrow, { issue: target.issue })}</Eyebrow>}
      title={t(mb.cancelTitle, { issue: target.issue })}
      hideClose
      className="max-w-[560px]"
      footer={
        <>
          <Button onClick={onClose}>{t(mb.keep)}</Button>
          <Button variant="danger-primary" loading={cancel.isPending} onClick={submit}>
            {t(mb.cancelSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span style={{ color: 'var(--ink-nav)' }}>
          {target.status === 'hold' && target.holdUntil
            ? t(mb.cancelHold, { sponsor: target.sponsor, date: shortDate(target.holdUntil), amount })
            : t(mb.cancelBooked, { sponsor: target.sponsor, amount })}
        </span>
        <div className="note w" role="note">
          <NoteIcon tone="warn" />
          <div className="min-w-0 flex-[1_1_320px]">{t(mb.cancelNote, { issue: target.issue })}</div>
        </div>
        <Field label={t(mb.reason)} hint={t(mb.reasonHint)}>
          <Input value={reason} maxLength={300} onChange={(e) => setReason(e.target.value)} />
        </Field>
      </div>
    </Dialog>
  )
}

/** Extend the hold: a later date, before the issue. */
export function ExtendHoldDialog({ bookingId, issue, sponsor, holdUntil, issueOn, onClose }: { bookingId: string; issue: number; sponsor: string; holdUntil?: string; issueOn: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const last = inputDay(new Date(new Date(issueOn).getTime() - 86_400_000))
  const first = inputDay(plusDays(1))
  const start = holdUntil ? inputDay(new Date(new Date(holdUntil).getTime() + 3 * 86_400_000)) : first
  const [date, setDate] = useState(start > last ? last : start)
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', `${WS}/sponsor-bookings/{id}`)
  const submit = () =>
    save.mutate(
      { params: { path: { workspace: slug, id: bookingId } }, body: { status: 'hold', holdUntil: date } },
      {
        onSuccess: (b) => {
          refresh()
          toast({ kind: 'success', title: t(mb.heldToast, { issue, sponsor, date: b.holdUntil ? shortDate(b.holdUntil) : date }) })
          onClose()
        },
        onError: (e) => {
          const f = fieldErrors(e)
          if (f) setError(Object.values(f)[0])
          else toast({ kind: 'error', title: errorMessage(e) })
        },
      },
    )
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mb.extendTitle, { issue })}
      className="max-w-[520px]"
      footer={
        <>
          <Button onClick={onClose}>{t(mb.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} onClick={submit}>
            {t(mb.extendSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span style={{ color: 'var(--ink-nav)' }}>{t(mb.extendText, { sponsor, issue })}</span>
        <Field label={t(mb.holdUntil)} error={error}>
          <Input type="date" value={date} min={first} max={last} onChange={(e) => setDate(e.target.value)} className="w-[180px]" />
        </Field>
      </div>
    </Dialog>
  )
}

/** Link a campaign by hand when the list and date don't find it. */
export function LinkCampaignDialog({ bookingId, issue, current, onClose }: { bookingId: string; issue: number; current?: string; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const [campaign, setCampaign] = useState(current ?? '')
  const list = $api.useQuery('get', '/api/v1/workspaces/{workspace}/campaigns', { params: { path: { workspace: slug }, query: { status: 'all', pageSize: 100 } } })
  const save = $api.useMutation('put', `${WS}/sponsor-bookings/{id}/campaign`)
  const submit = () =>
    save.mutate(
      { params: { path: { workspace: slug, id: bookingId } }, body: { campaignId: campaign || undefined } },
      {
        onSuccess: (b) => {
          refresh()
          toast({ kind: 'success', title: campaign ? t(mb.linkedToast, { issue, campaign: b.campaign?.name ?? '' }) : t(mb.unlinkedToast, { issue }) })
          onClose()
        },
        onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
      },
    )
  const items = (list.data?.items ?? []).filter((c) => c.status !== 'cancelled' && c.status !== 'failed')
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(mb.linkTitle, { issue })}
      className="max-w-[560px]"
      footer={
        <>
          <Button onClick={onClose}>{t(mb.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} onClick={submit}>
            {t(mb.linkSubmit)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <span style={{ color: 'var(--ink-nav)' }}>{t(mb.linkText)}</span>
        <Field label={t(mb.linkSelect)} error={list.error ? errorMessage(list.error) : undefined}>
          <Select value={campaign} onChange={(e) => setCampaign(e.target.value)}>
            <option value="">{t(mb.linkAuto)}</option>
            {items.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </Field>
      </div>
    </Dialog>
  )
}

/** Upload a logo through the sponsors' upload endpoint; the URL comes back. */
async function uploadLogo(slug: string, file: File): Promise<string> {
  const fd = new FormData()
  fd.append('file', file)
  const r = await fetch(`/api/v1/workspaces/${encodeURIComponent(slug)}/sponsors/logo`, { method: 'POST', body: fd, credentials: 'same-origin', headers: { Accept: 'application/json' } })
  const body = (await r.json().catch(() => ({}))) as { url?: string; detail?: string; title?: string; errors?: { message?: string }[] }
  if (!r.ok || !body.url) throw new Error(body.errors?.[0]?.message || body.detail || body.title || `HTTP ${r.status}`)
  return body.url
}

/** New sponsor, or Edit: name, website, logo, contact, payment terms and notes. */
export function SponsorDialog({ sponsor, onClose, onSaved }: { sponsor?: SponsorItem; onClose: () => void; onSaved?: (s: Schemas['SponsorItem']) => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const [name, setName] = useState(sponsor?.name ?? '')
  const [website, setWebsite] = useState(sponsor?.website ?? '')
  const [logo, setLogo] = useState(sponsor?.logoUrl ?? '')
  const [contact, setContact] = useState(sponsor?.contactName ?? '')
  const [email, setEmail] = useState(sponsor?.contactEmail ?? '')
  const [terms, setTerms] = useState(String(sponsor?.termsDays ?? 14))
  const [notes, setNotes] = useState(sponsor?.notes ?? '')
  const [error, setError] = useState<Record<string, string>>({})
  const [uploading, setUploading] = useState(false)
  const [removing, setRemoving] = useState(false)
  const file = useRef<HTMLInputElement>(null)
  const create = $api.useMutation('post', `${WS}/sponsors`)
  const update = $api.useMutation('put', `${WS}/sponsors/{id}`)
  const remove = $api.useMutation('delete', `${WS}/sponsors/{id}`)
  const body = { name, website, logoUrl: logo, contactName: contact, contactEmail: email, termsDays: Number(terms) || 0, notes }
  const done = (s: Schemas['SponsorItem'], msg: string) => {
    refresh()
    toast({ kind: 'success', title: msg })
    onSaved?.(s)
    onClose()
  }
  const fail = (e: unknown) => {
    const f = fieldErrors(e)
    if (f) setError(f)
    else toast({ kind: 'error', title: errorMessage(e) })
  }
  const submit = () => {
    if (sponsor) update.mutate({ params: { path: { workspace: slug, id: sponsor.id } }, body }, { onSuccess: (s) => done(s, t(md.sponsorSaved, { name: s.name })), onError: fail })
    else create.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: (s) => done(s, t(md.sponsorAdded, { name: s.name })), onError: fail })
  }
  const pick = async (f: File | undefined) => {
    if (!f) return
    setUploading(true)
    try {
      setLogo(await uploadLogo(slug, f))
    } catch (e) {
      toast({ kind: 'error', title: t(md.uploadFailed), detail: e instanceof Error ? e.message : String(e) })
    } finally {
      setUploading(false)
      if (file.current) file.current.value = ''
    }
  }
  return (
    <>
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        title={sponsor ? t(md.sponsorEditTitle, { name: sponsor.name }) : t(md.sponsorNewTitle)}
        description={t(md.sponsorText)}
        className="max-w-[600px]"
        footer={
          <>
            {sponsor ? (
              <Button variant="danger" className="mr-auto" onClick={() => setRemoving(true)}>
                {t(md.removeSponsor)}
              </Button>
            ) : null}
            <Button onClick={onClose}>{t(md.cancel)}</Button>
            <Button variant="primary" loading={create.isPending || update.isPending} onClick={submit} disabled={!name.trim()}>
              {sponsor ? t(md.save) : t(md.addSponsor)}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3.5">
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(220px,100%),1fr))] gap-3">
            <Field label={t(md.name)} error={error.name}>
              <Input value={name} maxLength={80} onChange={(e) => setName(e.target.value)} autoFocus />
            </Field>
            <Field label={t(md.website)} error={error.website} optional>
              <Input value={website} maxLength={300} placeholder={t(md.websitePlaceholder)} onChange={(e) => setWebsite(e.target.value)} />
            </Field>
          </div>
          <div className="lbl">
            <span>{t(md.logo)}</span>
            <div className={`sp-drop${uploading ? ' on' : ''}`}>
              {logo ? <img src={logo} alt="" /> : <Initials initials={initialsOf(name)} size="xl" />}
              <span className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="k font-medium">{t(md.logoHint)}</span>
              </span>
              <input ref={file} type="file" accept="image/png,image/jpeg,image/gif" className="vh" aria-label={t(md.logo)} onChange={(e) => void pick(e.target.files?.[0])} />
              <Button size="sm" icon="upload" loading={uploading} onClick={() => file.current?.click()}>
                {logo ? t(md.replace) : t(md.upload)}
              </Button>
              {logo ? (
                <Button size="sm" onClick={() => setLogo('')}>
                  {t(md.remove)}
                </Button>
              ) : null}
            </div>
            {error.logoUrl ? <span className="err">{error.logoUrl}</span> : null}
          </div>
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(220px,100%),1fr))] gap-3">
            <Field label={t(md.contactName)} error={error.contactName} optional>
              <Input value={contact} maxLength={120} onChange={(e) => setContact(e.target.value)} />
            </Field>
            <Field label={t(md.contactEmail)} error={error.contactEmail} optional>
              <Input type="email" value={email} maxLength={254} onChange={(e) => setEmail(e.target.value)} />
            </Field>
          </div>
          <Field label={t(md.terms)} hint={t(md.termsHint)} error={error.termsDays}>
            <SuffixInput value={terms} onChange={setTerms} suffix={t(md.termsSuffix, { days: Number(terms) || 0 })} max={120} />
          </Field>
          <Field label={t(md.notes)} hint={t(md.notesHint)} error={error.notes} optional>
            <Textarea value={notes} maxLength={2000} rows={3} onChange={(e) => setNotes(e.target.value)} />
          </Field>
        </div>
      </Dialog>
      {sponsor ? (
        <ConfirmDialog
          open={removing}
          onOpenChange={setRemoving}
          badge={t(md.removeSponsorConfirm)}
          title={t(md.removeSponsorTitle, { name: sponsor.name })}
          description={t(md.removeSponsorText)}
          confirmLabel={t(md.removeSponsorConfirm)}
          onConfirm={() =>
            remove.mutateAsync({ params: { path: { workspace: slug, id: sponsor.id } } }).then(() => {
              refresh()
              toast({ kind: 'success', title: t(md.sponsorRemoved, { name: sponsor.name }) })
              onClose()
            })
          }
        />
      ) : null}
    </>
  )
}

function initialsOf(name: string): string {
  return (
    name
      .split(/\s+/)
      .filter(Boolean)
      .slice(0, 2)
      .map((w) => w[0]!.toUpperCase())
      .join('') || '?'
  )
}

/** A number with its unit after it: "[14] days after the issue". */
function SuffixInput({ value, onChange, suffix, max, id, ...aria }: { value: string; onChange: (v: string) => void; suffix: string; max: number; id?: string; 'aria-describedby'?: string; 'aria-invalid'?: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <Input id={id} type="number" min={0} max={max} className="w-[90px]" value={value} onChange={(e) => onChange(e.target.value)} {...aria} />
      <span className="k">{suffix}</span>
    </div>
  )
}

/** New slot, or Edit: the list, where it sits, the price and the issues' schedule. */
export function SlotDialog({ slot, onClose }: { slot?: SlotItem; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefresh()
  const lists = $api.useQuery('get', '/api/v1/workspaces/{workspace}/lists', { params: { path: { workspace: slug } } })
  const currency = slot?.currency ?? 'usd'
  const [list, setList] = useState(slot?.listId ?? '')
  const [name, setName] = useState(slot?.name ?? '')
  const [placement, setPlacement] = useState(slot?.placement ?? '')
  const [price, setPrice] = useState(slot ? amountInput(slot.priceCents, currency).replace(/\.00$/, '') : '')
  const [first, setFirst] = useState(String(slot?.firstNumber ?? 1))
  const [on, setOn] = useState(slot ? inputDay(slot.firstOn) : today())
  const [every, setEvery] = useState(String(slot?.everyDays ?? 7))
  const [creative, setCreative] = useState(String(slot?.creativeDays ?? 4))
  const [selling, setSelling] = useState(slot?.selling ?? true)
  const [error, setError] = useState<Record<string, string>>({})
  const [removing, setRemoving] = useState(false)
  const create = $api.useMutation('post', `${WS}/sponsor-slots`)
  const update = $api.useMutation('put', `${WS}/sponsor-slots/{id}`)
  const remove = $api.useMutation('delete', `${WS}/sponsor-slots/{id}`)
  const fail = (e: unknown) => {
    const f = fieldErrors(e)
    if (f) setError(f)
    else toast({ kind: 'error', title: errorMessage(e) })
  }
  const submit = () => {
    const cents = parseAmount(price, currency)
    if (cents === undefined) return setError({ priceCents: t(md.amountInvalid) })
    const body = {
      listId: list,
      name,
      placement,
      priceCents: cents,
      currency: slot?.currency,
      firstNumber: Number(first) || 1,
      firstOn: on,
      everyDays: Number(every) || 7,
      creativeDays: Number(creative) || 0,
      selling,
    }
    const done = (msg: string) => {
      refresh()
      toast({ kind: 'success', title: msg })
      onClose()
    }
    if (slot) update.mutate({ params: { path: { workspace: slug, id: slot.id } }, body }, { onSuccess: (s) => done(t(md.slotSaved, { name: s.name })), onError: fail })
    else create.mutate({ params: { path: { workspace: slug } }, body }, { onSuccess: (s) => done(t(md.slotAdded, { name: s.name })), onError: fail })
  }
  return (
    <>
      <Dialog
        open
        onOpenChange={(o) => !o && onClose()}
        title={slot ? t(md.slotEditTitle, { name: slot.name }) : t(md.slotNewTitle)}
        description={t(md.slotText)}
        className="max-w-[600px]"
        footer={
          <>
            {slot ? (
              <Button variant="danger" className="mr-auto" onClick={() => setRemoving(true)}>
                {t(md.removeSlot)}
              </Button>
            ) : null}
            <Button onClick={onClose}>{t(md.cancel)}</Button>
            <Button variant="primary" loading={create.isPending || update.isPending} onClick={submit} disabled={!name.trim() || !list}>
              {slot ? t(md.save) : t(md.addSlot)}
            </Button>
          </>
        }
      >
        <div className="flex flex-col gap-3.5">
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(220px,100%),1fr))] gap-3">
            <Field label={t(md.list)} error={error.listId ?? (lists.error ? t(md.listsError) : undefined)}>
              <Select value={list} onChange={(e) => setList(e.target.value)}>
                <option value="">{t(md.chooseList)}</option>
                {(lists.data?.items ?? []).map((l) => (
                  <option key={l.id} value={l.id}>
                    {l.name}
                  </option>
                ))}
                {slot && !lists.data?.items.some((l) => l.id === slot.listId) ? <option value={slot.listId}>{slot.listName}</option> : null}
              </Select>
            </Field>
            <Field label={t(md.slotName)} error={error.name}>
              <Input value={name} maxLength={60} placeholder={t(md.slotNamePlaceholder)} onChange={(e) => setName(e.target.value)} />
            </Field>
          </div>
          <div className="grid grid-cols-[repeat(auto-fit,minmax(min(220px,100%),1fr))] gap-3">
            <Field label={t(md.placement)} error={error.placement} optional>
              <Input value={placement} maxLength={120} placeholder={t(md.placementPlaceholder)} onChange={(e) => setPlacement(e.target.value)} />
            </Field>
            <Field label={t(md.price)} hint={t(md.priceHint)} error={error.priceCents}>
              <AmountInput currency={currency} value={price} onChange={setPrice} />
            </Field>
          </div>
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            <legend className="mb-1 text-[13px] font-semibold">{t(md.scheduleLegend)}</legend>
            <div className="grid grid-cols-[repeat(auto-fit,minmax(min(150px,100%),1fr))] gap-3">
              <Field label={t(md.firstNumber)} error={error.firstNumber}>
                <Input type="number" min={1} value={first} onChange={(e) => setFirst(e.target.value)} />
              </Field>
              <Field label={t(md.firstOn)} error={error.firstOn}>
                <Input type="date" value={on} onChange={(e) => setOn(e.target.value)} />
              </Field>
              <Field label={t(md.every)} error={error.everyDays}>
                <SuffixInput value={every} onChange={setEvery} suffix={t(md.everySuffix, { days: Number(every) || 0 })} max={31} />
              </Field>
            </div>
            <span className="k">{t(md.scheduleHint, { days: Number(every) || 7 })}</span>
          </fieldset>
          <Field label={t(md.creativeDays)} error={error.creativeDays}>
            <SuffixInput value={creative} onChange={setCreative} suffix={t(md.creativeSuffix, { days: Number(creative) || 0 })} max={30} />
          </Field>
          <Toggle label={t(md.selling)} description={t(md.sellingHint)} checked={selling} onChange={(e) => setSelling(e.target.checked)} />
        </div>
      </Dialog>
      {slot ? (
        <ConfirmDialog
          open={removing}
          onOpenChange={setRemoving}
          badge={t(md.removeSlotConfirm)}
          title={t(md.removeSlotTitle, { name: slot.name })}
          description={t(md.removeSlotText)}
          confirmLabel={t(md.removeSlotConfirm)}
          onConfirm={() =>
            remove.mutateAsync({ params: { path: { workspace: slug, id: slot.id } } }).then(() => {
              refresh()
              toast({ kind: 'success', title: t(md.slotRemoved, { name: slot.name }) })
              onClose()
            })
          }
        />
      ) : null}
    </>
  )
}

/** A short icon button with its label hidden ("Copy link to this booking"). */
export function IconButton({ icon, label, onClick }: { icon: 'link' | 'close'; label: string; onClick: () => void }) {
  return (
    <button type="button" className="btn sm ic" aria-label={label} onClick={onClick}>
      <Icon name={icon} size={14} />
    </button>
  )
}
