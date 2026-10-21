import { useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { useState, type ReactNode } from 'react'
import { $api, ApiError, errorMessage, http, type Schemas } from '@/api/client'
import { Page } from '@/components/shell/Page'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { ErrorPanel, Loading, Skeleton } from '@/components/ui/Feedback'
import { Field, Input, Select } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { toast } from '@/components/ui/Toast'
import { clock, count, dayMonthDate, notificationTime, shortDate, usDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { useInvalidateAudience } from './ContactDialogs'
import { fileName, saveBlob } from './download'
import { consentLabel, consentTone, mailboxColor, providerColor, tierClass, tierLabel, type Consent, type Tier } from './labels'
import { brandOf } from '@/features/languages/LanguagesCard'
import { languageLabel } from '@/features/languages/composerParts'
import { ContactLanguageField, consentVersionText, deliveryLanguage } from '@/features/page-languages/contacts'
import { mc } from '@/features/page-languages/messages'
import { useI18n, useT } from '@/i18n'
import { NotificationsRow, TransactionalActivityRow, transactionalCount, useContactTransactional, withTransactional } from '@/features/transactional/ContactActivity'
import { JoinLine } from '@/features/growth/JoinLine'
import { ProfileScoreCard } from '@/features/scoring/ProfileScoreCard'
import { StoreOrdersCard, useStoreActivity } from '@/features/commerce/profile'
import { FieldSourceNote, useAnswerActivity, useFieldSources } from '@/features/polls/contact'
import { ContactPaidCard, ContactTierChip, PaidActivity, useHasPaidActivity } from '@/features/earn/ContactPaidCard'
import { AddressCheckRow } from '@/features/verification/AddressCheck'
// Digital products (X19): the Purchases card, bought and refunded events, the checkout's consent.
import { PurchasesCard, checkoutConsent, purchaseActivity, purchaseIcon, purchaseIcons } from '@/features/products/contact'
import { GrowActivityRow, growEvents, ProfileReferralsCard, useContactGrow, withGrow } from '@/features/grow/contact/ContactGrow'
import { ChannelConsentCard } from '@/features/sms/ContactConsent'
import { smsEventIcon, useSmsActivity } from '@/features/sms/activity'
import './audience.css'
import { ContactWebsiteCard } from '@/features/website/elsewhere'

type Profile = Schemas['ContactProfile']
type Item = Schemas['ContactActivityItem']

/** The non-empty parts, for "a · b · c" lines. */
const present = (...xs: (string | null | undefined | false)[]) => xs.filter((x): x is string => Boolean(x))

const fullDate = (iso: string) => {
  const d = new Date(iso)
  return `${dayMonthDate(d)} ${d.getFullYear()}`
}
/** "Today 10:42", else "Tue 7 Oct 09:41". */
const dateTime = (iso: string) => (usDate(iso) === usDate(new Date()) ? notificationTime(iso) : `${shortDate(iso)} ${clock(iso)}`)
const fullDateTime = (iso: string) => `${fullDate(iso)} ${clock(iso)}`

function initials(p: Profile) {
  const n = p.name || p.email
  const parts = n.split(/\s+/).filter(Boolean)
  return ((parts[0]?.[0] ?? '') + (parts.length > 1 ? (parts[parts.length - 1]?.[0] ?? '') : '')).toUpperCase() || n[0]!.toUpperCase()
}

/** 7.9 Contact profile: summary, fields, consent per list and the activity timeline. */
export function ContactProfilePage({ contactId }: { contactId: string }) {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contact}', { params: { path: { workspace: slug, contact: contactId } } })
  const p = q.data
  const name = p ? p.name || p.email : 'Contact'
  const [suppressing, setSuppressing] = useState(false)
  const editReason = usePermissionReason('subscribers:edit')
  const exportReason = usePermissionReason('subscribers:export')
  const resend = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/{contact}/resend-optin')
  const invalidate = useInvalidateAudience()
  // A contact that doesn't exist (or that the role can't see) gets the
  // shell's 404 or 403 screen.
  if (!p && q.error instanceof ApiError && (q.error.status === 404 || q.error.status === 403)) throw q.error
  return (
    <Page
      title={name}
      contentClassName="aud-crumbs"
      badges={
        <nav aria-label="Breadcrumb" className="flex items-center gap-2 text-[13px]">
          <Link to="/w/$workspace/audience" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
            Audience
          </Link>
          <span className="k" aria-hidden="true">
            /
          </span>
          <Link to="/w/$workspace/audience" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
            Contacts
          </Link>
          <span className="k" aria-hidden="true">
            /
          </span>
          <span className="font-semibold" aria-current="page">
            {name}
          </span>
        </nav>
      }
      actions={
        p ? (
          <>
            <Button
              disabledReason={editReason}
              loading={resend.isPending}
              onClick={async () => {
                try {
                  const r = await resend.mutateAsync({ params: { path: { workspace: slug, contact: contactId } } })
                  await invalidate()
                  toast(
                    r.sent
                      ? { kind: 'success', title: `Sent a confirmation email to ${p.email}`, detail: 'Through the System stream; the profile shows it in the consent history.' }
                      : { kind: 'info', title: 'No System provider could send it', detail: r.confirmUrl ? `Share this link instead: ${r.confirmUrl}` : undefined },
                  )
                } catch (e) {
                  toast({ kind: 'error', title: "Couldn't resend the opt-in", detail: errorMessage(e) })
                }
              }}
            >
              Resend opt-in
            </Button>
            <Button disabledReason={exportReason} onClick={() => void exportData(slug, contactId, p.email)}>
              Export data (GDPR)
            </Button>
            <Button className="dline" disabledReason={p.suppression ? 'This address is already on the suppression list.' : editReason} onClick={() => setSuppressing(true)}>
              Suppress
            </Button>
          </>
        ) : null
      }
    >
      {q.isError && !p ? (
        <ErrorPanel title="Couldn't load this contact" actions={<Button onClick={() => q.refetch()}>Retry</Button>}>
          {errorMessage(q.error)} Nothing was changed.
        </ErrorPanel>
      ) : !p ? (
        <ProfileSkeleton />
      ) : (
        <>
          {p.suppression ? (
            <Note tone="bad" label="Suppressed">
              {p.email} is on the shared suppression list ({p.suppression.reason.replace('_', ' ')}, {fullDate(p.suppression.at)}).{' '}
              {p.suppression.scope === 'marketing' ? 'Broadcasts and Automations are blocked; System mail is allowed.' : 'Every stream is blocked on every provider.'}
            </Note>
          ) : null}
          <Summary p={p} />
          <ProfileScoreCard contactId={p.id} firstName={p.firstName} tier={p.tier as Tier} fields={p.fields} />
          {/* Grow: the Referrals card, or who referred them (design expansion, X15). */}
          <ProfileReferralsCard contactId={p.id} />
          <div className="flex flex-wrap items-start gap-5">
            <div className="flex min-w-0 flex-[1_1_380px] flex-col gap-5">
              <ContactPaidCard contactId={p.id} />
              <FieldsCard p={p} />
              {/* What the contact bought (digital products, X19). */}
              <PurchasesCard contactId={p.id} />
              <ConsentCard p={p} />
              {/* The website's Website card (features/website), while it's on. */}
              <ContactWebsiteCard contactId={p.id} name={[p.firstName, p.lastName].filter(Boolean).join(' ') || p.email} />
              {/* A connected store's orders and consent (X20). */}
              <StoreOrdersCard contactId={p.id} />
              <ChannelConsentCard contactId={p.id} name={p.name || p.email} />
            </div>
            <ActivityCard contactId={contactId} total={p.activityTotal} />
          </div>
        </>
      )}
      {suppressing && p ? <SuppressDialog p={p} onClose={() => setSuppressing(false)} /> : null}
    </Page>
  )
}

async function exportData(slug: string, id: string, email: string) {
  try {
    const { data, response } = await http.GET('/api/v1/workspaces/{workspace}/contacts/{contact}/export', {
      params: { path: { workspace: slug, contact: id } },
      parseAs: 'blob',
    })
    if (data) saveBlob(data as unknown as Blob, fileName(response, `${email}.json`))
    toast({ kind: 'success', title: `Exported everything about ${email}`, detail: 'The export is in the audit log.' })
  } catch (e) {
    toast({ kind: 'error', title: "Couldn't export", detail: errorMessage(e) })
  }
}

function Summary({ p }: { p: Profile }) {
  const tier = p.tier as Tier
  const consent = p.consent as Consent
  const s = p.stats
  const lastOpen = s.lastOpenAt ? `${shortDate(s.lastOpenAt)}${s.lastOpenMachine ? ' · machine' : ''}` : 'never'
  return (
    <section className="card flex flex-wrap items-center gap-4.5 px-5.5 py-5" aria-label="Summary">
      <span className="inline-flex h-[52px] w-[52px] items-center justify-center rounded-full bg-[#DBEAFE] text-[18px] font-bold text-[#1E40AF]" aria-hidden="true">
        {initials(p)}
      </span>
      <div className="flex flex-[1_1_260px] flex-col gap-0.5">
        <span className="text-[20px] font-bold tracking-[-0.01em]">
          {p.name || p.email}
          <ContactTierChip contactId={p.id} />
        </span>
        <span className="mono text-[13px] text-ink-2">{p.email}</span>
        <div className="mt-1 flex flex-wrap gap-1.5">
          <span className={`badge ${tierClass[tier]}`}>{tierLabel[tier]}</span>
          <Badge tone={consentTone[consent]}>{consentLabel[consent]}</Badge>
          <span className="badge mut">
            <span className="dot" style={{ color: mailboxColor[p.mailboxProvider] ?? mailboxColor.other }} aria-hidden="true" />
            {p.mailboxLabel}
          </span>
          <span className="badge mut">
            <JoinLine contactId={p.id} day={fullDate(p.joinedAt)} sourceLabel={p.sourceLabel} />
          </span>
          {p.roleAddress ? <span className="badge warn">Role address · no Broadcasts</span> : null}
        </div>
      </div>
      <div className="grid grid-cols-4 gap-4 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]">
        <Stat label="Emails · 90 d" value={count(s.emails90d)} />
        <Stat label="Human opens" value={s.humanOpenRate !== undefined ? `${count(s.humanOpens)} · ${Math.round(s.humanOpenRate)}%` : count(s.humanOpens)} />
        <Stat label="Clicks" value={count(s.clicks)} />
        <Stat label="Last open" value={lastOpen} />
      </div>
    </section>
  )
}

function Stat({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div>
      <div className="k">{label}</div>
      <div className="n text-[18px]">{value}</div>
    </div>
  )
}

function fieldValue(f: Schemas['ContactProfileField']) {
  if (f.value === '') return <span className="text-ink-4">–</span>
  if (f.display === 'currency' && f.value !== '' && !Number.isNaN(Number(f.value))) {
    const n = Number(f.value)
    return <span className="n">${n % 1 === 0 ? n.toLocaleString('en-US') : n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
  }
  if (f.type === 'number') return <span className="n">{f.value}</span>
  return <span>{f.value}</span>
}

function FieldsCard({ p }: { p: Profile }) {
  // Fields a poll or a survey set say so (design expansion, X11).
  const sources = useFieldSources(p.id)
  const [editing, setEditing] = useState(false)
  const [adding, setAdding] = useState(false)
  const reason = usePermissionReason('subscribers:edit')
  const { slug } = useWorkspace()
  const update = $api.useMutation('patch', '/api/v1/workspaces/{workspace}/contacts/{contact}')
  const qc = useQueryClient()
  const save = async (body: Schemas['ContactUpdate']) => {
    await update.mutateAsync({ params: { path: { workspace: slug, contact: p.id } }, body })
    await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/contacts/{contact}'] })
  }
  return (
    <section className="card flex flex-col px-4.5 py-4" aria-labelledby="fields-h">
      <div className="mb-1.5 flex items-center justify-between">
        <h2 className="h2" id="fields-h">
          Fields
        </h2>
        <Button size="sm" disabledReason={reason} onClick={() => setEditing(true)}>
          Edit
        </Button>
      </div>
      <AddressCheckRow contactId={p.id} />
      {p.fields.map((f) =>
        f.key === 'language' ? (
          <ContactLanguageField key={f.key} p={p} />
        ) : (
          <div key={f.key} className="pkv">
            <span className="k">
              {f.label}
              {f.custom ? <> <span className="mono">{f.type === 'text' || f.type === 'select' ? 'custom' : `custom · ${f.type}`}</span></> : null}
            </span>
            <span>
              {fieldValue(f)}
              <FieldSourceNote source={sources.get(f.key)} />
            </span>
          </div>
        ),
      )}
      <div className="mt-2.5 flex flex-wrap gap-1.5">
        {p.tags.map((t) => (
          <span key={t} className="ptag">
            {t}
          </span>
        ))}
        {adding ? (
          <TagInput
            onCancel={() => setAdding(false)}
            onSave={async (tag) => {
              try {
                await save({ tags: [...p.tags, tag] })
                setAdding(false)
              } catch (e) {
                toast({ kind: 'error', title: "Couldn't add the tag", detail: errorMessage(e) })
              }
            }}
          />
        ) : (
          <Button size="sm" className="border-dashed" disabledReason={reason} onClick={() => setAdding(true)}>
            Add tag
          </Button>
        )}
      </div>
      {editing ? <EditFieldsDialog p={p} onClose={() => setEditing(false)} onSave={save} /> : null}
    </section>
  )
}

function TagInput({ onSave, onCancel }: { onSave: (tag: string) => Promise<void>; onCancel: () => void }) {
  const [tag, setTag] = useState('')
  return (
    <form
      className="flex items-center gap-1.5"
      onSubmit={(e) => {
        e.preventDefault()
        if (tag.trim()) void onSave(tag.trim())
      }}
    >
      <label className="vh" htmlFor="new-tag">
        New tag
      </label>
      <input id="new-tag" className="in h-[30px] w-[140px]" value={tag} onChange={(e) => setTag(e.target.value)} placeholder="tag" maxLength={60} autoFocus />
      <Button size="sm" type="submit" variant="primary">
        Add
      </Button>
      <Button size="sm" onClick={onCancel}>
        Cancel
      </Button>
    </form>
  )
}

function EditFieldsDialog({ p, onClose, onSave }: { p: Profile; onClose: () => void; onSave: (b: Schemas['ContactUpdate']) => Promise<void> }) {
  const t = useT()
  const { locale } = useI18n()
  const builtin = (key: string) => (key === 'language' ? p.language.code : (p.fields.find((f) => f.key === key)?.value ?? ''))
  const [vals, setVals] = useState<Record<string, string>>(() => {
    const out: Record<string, string> = { firstName: p.firstName, lastName: p.lastName, company: builtin('company'), language: builtin('language') }
    for (const f of p.fields) if (f.custom) out[`custom:${f.key}`] = f.value
    return out
  })
  const [err, setErr] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const set = (k: string, v: string) => setVals((s) => ({ ...s, [k]: v }))
  const submit = async () => {
    setBusy(true)
    setErr(null)
    try {
      const fields: Record<string, unknown> = {}
      for (const f of p.fields) {
        if (!f.custom) continue
        const v = vals[`custom:${f.key}`] ?? ''
        if (v !== f.value) fields[f.key] = v === '' ? null : v
      }
      await onSave({ firstName: vals.firstName, lastName: vals.lastName, company: vals.company, ...(vals.language !== p.language.code ? { language: vals.language } : {}), fields })
      onClose()
      toast({ kind: 'success', title: 'Saved the fields' })
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title="Edit fields"
      description="Changes are logged in the audit log with the old and new values."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="primary" loading={busy} onClick={() => void submit()}>
            Save fields
          </Button>
        </>
      }
    >
      <form
        className="flex flex-col gap-3.5"
        onSubmit={(e) => {
          e.preventDefault()
          void submit()
        }}
      >
        {err ? <Note tone="bad">{err}</Note> : null}
        <div className="grid grid-cols-2 gap-3 max-md:grid-cols-1">
          <Field label="First name">
            <Input value={vals.firstName} onChange={(e) => set('firstName', e.target.value)} />
          </Field>
          <Field label="Last name">
            <Input value={vals.lastName} onChange={(e) => set('lastName', e.target.value)} />
          </Field>
        </div>
        <Field label="Company">
          <Input value={vals.company} onChange={(e) => set('company', e.target.value)} />
        </Field>
        <Field label={t(mc.fieldLabel)}>
          <Select value={vals.language} onChange={(e) => set('language', e.target.value)}>
            {p.language.options.map((o) => (
              <option key={o.code} value={o.code}>
                {t(mc.option, { language: languageLabel(o.code, locale), code: o.code })}
              </option>
            ))}
            <option value="">{t(mc.optionNone)}</option>
          </Select>
        </Field>
        {p.fields
          .filter((f) => f.custom)
          .map((f) => (
            <Field key={f.key} label={f.key} hint={f.type === 'date' ? 'A date like 2026-10-30.' : f.display === 'currency' ? 'A number, shown in dollars.' : undefined}>
              <Input
                value={vals[`custom:${f.key}`] ?? ''}
                onChange={(e) => set(`custom:${f.key}`, e.target.value)}
                inputMode={f.type === 'number' ? 'decimal' : undefined}
                type={f.type === 'date' ? 'date' : 'text'}
              />
            </Field>
          ))}
      </form>
    </Dialog>
  )
}

/** One sentence per list from its consent history. consentLanguage names
 * the language a consent text was shown in, when the workspace has several. */
function consentText(l: Schemas['ContactListConsent'], consentLanguage?: (version: string, language: string) => string) {
  if (l.events.length === 0) {
    switch (l.status) {
      case 'none':
        return 'Never subscribed'
      case 'pending':
        return 'Waiting for the confirmation click'
      case 'unsubscribed':
        return 'Unsubscribed'
    }
    return 'Subscribed'
  }
  const parts: string[] = []
  for (const e of l.events) {
    const src = e.sourceLabel
    // A buyer who ticked a product's list offer at checkout (X19): "Product checkout: ticked “…”".
    const checkout = checkoutConsent(e, fullDateTime(e.at))
    if (checkout) {
      parts.push(checkout)
      continue
    }
    switch (e.action) {
      case 'imported':
        parts.push(`Subscribed via ${src} ${fullDate(e.at)}`)
        break
      case 'subscribed':
        parts.push(`${e.source === 'preference_center' ? `Added in ${src}` : `Subscribed via ${src}`} ${fullDateTime(e.at)}${e.detail ? ` · ${e.detail}` : ''}`)
        break
      case 'confirmed': {
        const how = e.source === 'repermission' ? 're-permission confirmed' : parts.length ? 'confirmed' : `Confirmed via ${src}`
        const version = e.consentVersion ? (e.language && consentLanguage ? consentLanguage(e.consentVersion, e.language) : `consent text ${e.consentVersion}`) : ''
        parts.push(`${how} ${fullDateTime(e.at)}${e.ip ? ` from ${e.ip}` : ''}${version ? ` · ${version}` : ''}`)
        break
      }
      case 'unsubscribed':
        parts.push(`unsubscribed ${fullDate(e.at)}${src ? ` in ${src}` : ''}`)
        break
      case 'resubscribed':
        parts.push(`resubscribed ${fullDate(e.at)}`)
        break
      case 'optin_sent':
        parts.push(`opt-in email sent ${fullDate(e.at)}`)
        break
    }
  }
  const s = parts.join(' · ')
  return s.charAt(0).toUpperCase() + s.slice(1)
}

function consentBadge(l: Schemas['ContactListConsent'], p: Profile) {
  switch (l.status) {
    case 'subscribed': {
      const confirmed = p.consent === 'double' || l.events.some((e) => e.action === 'confirmed' || e.source === 'preference_center')
      return <Badge tone="ok">{confirmed ? 'Confirmed' : 'Subscribed'}</Badge>
    }
    case 'pending':
      return <Badge tone="warn">Pending</Badge>
    case 'unsubscribed':
      return <Badge tone="neutral">Unsubscribed</Badge>
  }
  return <Badge tone="neutral">Not subscribed</Badge>
}

function ConsentCard({ p }: { p: Profile }) {
  const { slug } = useWorkspace()
  const t = useT()
  const { locale } = useI18n()
  const multilingual = p.language.options.filter((o) => o.offered).length > 1
  const consentLanguage = multilingual ? (version: string, language: string) => consentVersionText(version, language, t, locale) : undefined
  const link = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/{contact}/preference-link')
  const allDouble = p.lists.length > 0 && p.lists.every((l) => l.doubleOptIn)
  const freq = { all: 'every email', weekly: 'weekly', monthly: 'monthly' }[p.preferences.frequency] ?? p.preferences.frequency
  const first = p.firstName || p.name || 'this contact'
  return (
    <section className="card flex flex-col" aria-labelledby="consent-h">
      <div className="flex items-center justify-between px-4.5 py-3.5">
        <h2 className="h2" id="consent-h">
          Consent per list
        </h2>
        <span className="k">{allDouble ? 'double opt-in on by default' : 'single opt-in on some lists'}</span>
      </div>
      <div className="flex flex-col px-4.5 pb-1.5">
        {p.lists.map((l) => (
          <div key={l.listId} className="pkv items-start">
            <div className="flex flex-col gap-0.5">
              <span className="font-semibold">{l.name}</span>
              <span className="k">{consentText(l, consentLanguage)}</span>
            </div>
            {consentBadge(l, p)}
          </div>
        ))}
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-4.5 py-2.5">
        <span className="k">
          Frequency: {freq} · paused: {p.preferences.pausedUntil ? `until ${shortDate(p.preferences.pausedUntil)}` : 'no'} · fewer-emails: {p.preferences.fewerEmails ? 'yes' : 'no'}
        </span>
        <Button
          size="sm"
          loading={link.isPending}
          onClick={async () => {
            try {
              const r = await link.mutateAsync({ params: { path: { workspace: slug, contact: p.id } } })
              window.open(r.url, '_blank', 'noopener,noreferrer')
            } catch (e) {
              toast({ kind: 'error', title: "Couldn't open the preference center", detail: errorMessage(e) })
            }
          }}
        >
          Open preference center as {first}
        </Button>
      </div>
    </section>
  )
}

type Filter = 'all' | 'deliveries' | 'engagement' | 'consent' | 'paid'

function ActivityCard({ contactId, total }: { contactId: string; total: number }) {
  const { slug } = useWorkspace()
  const [filter, setFilter] = useState<Filter>('all')
  // Paid (Earn, X16): the paid member's timeline, for contacts who had paid access.
  const hasPaid = useHasPaidActivity(contactId)
  const paid = filter === 'paid'
  const q = $api.useInfiniteQuery(
    'get',
    '/api/v1/workspaces/{workspace}/contacts/{contact}/activity',
    { params: { path: { workspace: slug, contact: contactId }, query: { filter: paid ? 'all' : filter, limit: 8 } } },
    { pageParamName: 'cursor', initialPageParam: '', getNextPageParam: (last) => last.nextCursor || undefined, enabled: !paid },
  )
  const items = q.data?.pages.flatMap((pg) => pg.items) ?? []
  // Transactional mail joins the deliveries; the Optional categories, consent.
  const tx = useContactTransactional(contactId)
  const base = paid ? 'all' : filter
  // Grow's events join them the same way (design expansion, X15).
  const grow = useContactGrow(contactId)
  const growList = paid ? [] : growEvents(grow.data, base)
  const entries = paid ? [] : withGrow(withTransactional(items, tx.data, base, !q.hasNextPage), growList, !q.hasNextPage, items.length ? Date.parse(items[items.length - 1]!.at) : -Infinity)
  const txCount = transactionalCount(tx.data, base)
  const shown = (q.data?.pages[0]?.total ?? (filter === 'all' ? total : 0)) + txCount + growList.length
  const listed = entries.filter((e) => e.kind !== 'notifications').length
  return (
    <section className="card flex min-w-0 flex-[1.4_1_480px] flex-col" aria-labelledby="activity-h">
      <div className="flex flex-wrap items-center justify-between gap-2 px-4.5 py-3.5">
        <h2 className="h2" id="activity-h">
          Activity
        </h2>
        <div className="flex gap-1.5 max-md:flex-wrap" role="group" aria-label="Show">
          {(
            [
              ['all', 'All'],
              ['deliveries', 'Deliveries'],
              ['engagement', 'Engagement'],
              ['consent', 'Consent'],
              ...(hasPaid ? [['paid', 'Paid']] : []),
            ] as [Filter, string][]
          ).map(([v, l]) => (
            <button key={v} type="button" className="badge mut afilter" aria-pressed={filter === v} data-all={v === 'all' || undefined} onClick={() => setFilter(v)}>
              {l}
            </button>
          ))}
        </div>
      </div>
      {paid ? (
        <PaidActivity contactId={contactId} />
      ) : q.isError ? (
        <div className="border-t border-line-2 px-4.5 py-4">
          <ErrorPanel title="Couldn't load the activity" actions={<Button size="sm" onClick={() => q.refetch()}>Retry</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <Loading label="Loading activity" className="flex flex-col">
          {Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="ev">
              <Skeleton w={26} h={26} r={13} />
              <div className="flex flex-1 flex-col gap-1.5">
                <Skeleton w="60%" h={13} />
                <Skeleton w="40%" h={11} />
              </div>
            </div>
          ))}
        </Loading>
      ) : entries.length === 0 ? (
        <div className="border-t border-line-2 px-4.5 py-8 text-center text-ink-3">
          {filter === 'all' ? 'No activity yet. Deliveries, opens, clicks and consent changes appear here.' : `No ${filter} events.`}
        </div>
      ) : (
        <ol className="m-0 list-none p-0">
          {entries.map((e) =>
            e.kind === 'item' ? (
              <ActivityRow key={e.it.id} it={e.it} />
            ) : e.kind === 'tx' ? (
              <TransactionalActivityRow key={`tx-${e.m.id}`} m={e.m} when={dateTime} product={tx.product} canViewContent={Boolean(tx.data?.canViewContent)} />
            ) : e.kind === 'grow' ? (
              <GrowActivityRow key={`grow-${e.e.kind}-${e.at}`} e={e.e} />
            ) : tx.data ? (
              <NotificationsRow key="tx-notifications" tx={tx.data} product={tx.product} essential={tx.essential} />
            ) : null,
          )}
        </ol>
      )}
      {entries.length > 0 && !paid ? (
        <div className="flex items-center justify-between border-t border-line-2 px-4.5 py-2.5">
          <span className="k" role="status">
            {count(listed)} of {count(shown)} events
          </span>
          {q.hasNextPage ? (
            <Button size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
              Load more
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  )
}

const icons = {
  eye: (
    <>
      <circle cx="8" cy="8" r="2.5" />
      <path d="M1.5 8s2.5-4.5 6.5-4.5S14.5 8 14.5 8 12 12.5 8 12.5 1.5 8 1.5 8z" />
    </>
  ),
  check: <path d="m3 8 3.5 3.5L13 5" />,
  cursor: <path d="M3 2.5 13 7l-5 1-2.5 5z" strokeLinejoin="round" />,
  flow: (
    <>
      <circle cx="4" cy="3" r="2" />
      <circle cx="4" cy="13" r="2" />
      <circle cx="12" cy="8" r="2" />
      <path d="M4 5v6M6 8h4" />
    </>
  ),
  bang: <path d="M8 2v7M8 12v1" />,
  upload: <path d="M8 11V3M4.5 6.5 8 3l3.5 3.5M2 13h12" />,
  x: <path d="m4 4 8 8M12 4l-8 8" />,
  mail: (
    <>
      <rect x="2" y="3.5" width="12" height="9" rx="1.5" />
      <path d="m2.5 4.5 5.5 4 5.5-4" />
    </>
  ),
  minus: <path d="M4 8h8" />,
  // Purchases and refunds (digital products, X19).
  ...purchaseIcons,
}

const tone = {
  blue: { bg: '#DBEAFE', fg: '#1E40AF' },
  green: { bg: '#DCFCE7', fg: '#166534' },
  purple: { bg: '#EDE9FE', fg: '#5B21B6' },
  amber: { bg: '#FEF3C7', fg: '#92400E' },
  red: { bg: '#FEE2E2', fg: '#991B1B' },
  grey: { bg: '#EEF0F3', fg: '#4B5563' },
}

function eventIcon(it: Item): [keyof typeof icons, keyof typeof tone] {
  // Purchases and refunds (digital products, X19).
  const purchase = purchaseIcon(it)
  if (purchase) return purchase
  // Texts (design expansion, X21).
  const text = smsEventIcon(it)
  if (text) return text
  // Poll answers and survey responses (design expansion, X11).
  if (it.type === 'poll_answer' || it.type === 'survey_response' || it.type === 'survey_started') return ['check', 'green']
  // Store orders and abandoned checkouts (X20).
  if (it.type === 'store_order') return ['check', 'green']
  if (it.type === 'checkout_abandoned') return it.store?.flow ? ['flow', 'purple'] : ['minus', 'grey']
  if (it.category === 'consent') {
    switch (it.type) {
      case 'confirmed':
      case 'subscribed':
      case 'resubscribed':
        return ['check', 'green']
      case 'imported':
        return ['upload', 'grey']
      case 'optin_sent':
        return ['mail', 'blue']
      case 'suppressed':
      case 'erased':
        return ['x', 'red']
    }
    return ['minus', 'grey']
  }
  switch (it.type) {
    case 'open':
      return ['eye', it.machine ? 'grey' : 'blue']
    case 'click':
      return ['cursor', 'blue']
    case 'unsubscribe':
      return ['minus', 'grey']
    case 'delivered':
      return it.subject.includes(' · step') || it.subject.includes('series') ? ['flow', 'purple'] : ['check', 'green']
    case 'soft_bounce':
    case 'deferred':
      return ['bang', 'amber']
    case 'hard_bounce':
    case 'complaint':
    case 'rejected':
      return ['x', 'red']
  }
  return ['check', 'grey']
}

function ProviderBadge({ p }: { p: Schemas['ContactActivityProvider'] }) {
  if (p.adapter === 'ses') {
    return (
      <span className="badge pur h-5">
        {p.name}
      </span>
    )
  }
  return (
    <span className="badge mut h-5">
      <span className="dot" style={{ color: providerColor[p.adapter] ?? 'var(--chart-other)' }} aria-hidden="true" />
      {p.name}
    </span>
  )
}

const verb: Record<string, string> = {
  delivered: 'Delivered',
  deferred: 'Deferred',
  soft_bounce: 'Soft bounce',
  hard_bounce: 'Hard bounce',
  complaint: 'Complaint',
  rejected: 'Rejected',
  open: 'Opened',
  click: 'Clicked',
  unsubscribe: 'Unsubscribed',
}

function ActivityRow({ it }: { it: Item }) {
  const answerActivity = useAnswerActivity()
  // Store orders and abandoned checkouts (X20).
  const storeActivity = useStoreActivity()
  const smsActivity = useSmsActivity()
  const [icon, t] = eventIcon(it)
  const c = tone[t]
  const tr = useT()
  const { locale } = useI18n()
  const { me } = useWorkspace()
  // The language version a delivery sent (ContactProfile, LanguageStates).
  const dl = deliveryLanguage(it, tr, locale, brandOf(me.workspace.name))
  let title: ReactNode
  let meta: string[] = [dateTime(it.at)]
  const answer = answerActivity(it) ?? storeActivity(it)
  // Purchases and refunds (digital products, X19).
  const purchase = purchaseActivity(it)
  const text = smsActivity(it)
  if (answer) {
    title = <b>{answer.title}</b>
    meta = [...meta, ...answer.meta]
  } else if (purchase) {
    title = purchase.title
    meta = [...meta, ...purchase.meta]
  } else if (text) {
    title = text.title
    meta = [...meta, ...text.meta]
  } else if (it.category === 'consent') {
    switch (it.type) {
      case 'confirmed':
        title = (
          <>
            <b>Confirmed subscription</b> {it.list}
            {it.detail ? ` · ${it.detail}` : ''}
          </>
        )
        meta = [...meta, ...present(it.note, it.ip ? `IP ${it.ip}` : '')]
        break
      case 'imported':
        title = (
          <>
            <b>Imported</b> from {(it.sourceLabel ?? "").replace(/ import$/, "")}
            {it.detail ? ` · ${it.detail}` : ''}
          </>
        )
        meta = [...meta, ...present(it.actor ? `by ${it.actor}` : '', it.note)]
        break
      case 'subscribed':
      case 'resubscribed':
        title = (
          <>
            <b>{it.type === 'subscribed' ? 'Subscribed' : 'Resubscribed'}</b> to {it.list} · {it.sourceLabel}
          </>
        )
        meta = [...meta, ...present(it.detail, it.note)]
        break
      case 'unsubscribed':
        title = (
          <>
            <b>Unsubscribed</b> from {it.list || 'every list'}
          </>
        )
        meta = [...meta, ...present(it.sourceLabel ? `in ${it.sourceLabel}` : '', it.detail)]
        break
      case 'optin_sent':
        title = (
          <>
            <b>Opt-in email sent</b>
            {it.list ? ` for ${it.list}` : ''}
          </>
        )
        meta = [...meta, ...present(it.note, it.actor ? `by ${it.actor}` : '')]
        break
      case 'suppressed':
        title = <b>Suppressed</b>
        meta = [...meta, ...present(it.detail, it.note, it.actor ? `by ${it.actor}` : '')]
        break
      default:
        title = <b>{it.type.replace('_', ' ')}</b>
        meta = [...meta, ...present(it.detail, it.note)]
    }
  } else if (it.type === 'open') {
    title = (
      <>
        <b>Opened</b> {it.subject} · {it.machine ? 'machine open' : 'human open'}
      </>
    )
    const mins = it.afterDeliveryMin
    meta = [
      ...meta,
      it.machine ? 'Apple Mail Privacy Protection' : it.client ?? '',
      mins !== undefined && mins !== null ? (mins < 60 ? `${mins} min after delivery` : `${Math.round(mins / 60)} h after delivery`) : '',
    ].filter(Boolean)
  } else if (it.type === 'click') {
    title = (
      <>
        <b>Clicked</b> {it.linkLabel ? `"${it.linkLabel}"` : 'a link'} in {it.subject}
      </>
    )
    meta = [...meta, ...present(it.url ?? '')]
  } else if (it.type === 'unsubscribe') {
    title = (
      <>
        <b>Unsubscribed</b> from {it.subject}
      </>
    )
  } else {
    title = (
      <>
        <b>{verb[it.type] ?? it.type}</b> {it.subject}
        {it.variant ? ` · variant ${it.variant}` : ''}
        {dl.label ? ` · ${dl.label}${dl.fallback ? ' ·' : ''}` : ''}
        {it.provider ? (
          <>
            {' via '}
            <ProviderBadge p={it.provider} />
          </>
        ) : it.imported ? (
          ` · sent from ${it.imported}`
        ) : null}
        {it.failover ? (
          <>
            <span className="inline-block w-1.5" />
            <span className="badge warn h-5">failover</span>
          </>
        ) : null}
      </>
    )
    const code = [it.code, it.diagnostic].filter(Boolean).join(' ')
    meta = [...meta, ...present(code, it.routeNote ?? '', it.retriedDeliveredAt ? `retried and delivered ${clock(it.retriedDeliveredAt)}` : '', dl.why)]
  }
  return (
    <li className="ev">
      <span className="ic" style={{ background: c.bg }}>
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke={c.fg} strokeWidth={icon === 'check' ? 2 : 1.8} aria-hidden="true">
          {icons[icon]}
        </svg>
      </span>
      <div className="flex flex-1 flex-col gap-0.5">
        <span>{title}</span>
        <span className="k">{meta.join(' · ')}</span>
      </div>
    </li>
  )
}

function SuppressDialog({ p, onClose }: { p: Profile; onClose: () => void }) {
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const suppress = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/{contact}/suppress')
  const [reason, setReason] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const submit = async () => {
    if (!reason.trim()) {
      setErr('Say why, for the audit log.')
      return
    }
    try {
      await suppress.mutateAsync({ params: { path: { workspace: slug, contact: p.id } }, body: { reason: reason.trim() } })
      await invalidate()
      onClose()
      toast({ kind: 'success', title: `Suppressed ${p.email}`, detail: 'Every provider stops sending to this address.' })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={`Suppress ${p.email}?`}
      description="The address goes on the shared suppression list and is pushed to every provider. Removing it later needs a reason, and bounced addresses need a new opt-in."
      footer={
        <>
          <Button onClick={onClose}>Cancel</Button>
          <Button variant="danger-primary" loading={suppress.isPending} onClick={() => void submit()}>
            Suppress
          </Button>
        </>
      }
    >
      <Field label="Reason" error={err ?? undefined} hint="Recorded in the audit log with your name.">
        <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Asked to be removed by phone" maxLength={300} />
      </Field>
    </Dialog>
  )
}

function ProfileSkeleton() {
  return (
    <Loading label="Loading the contact" className="flex flex-col gap-5">
      <div className="card flex items-center gap-4.5 px-5.5 py-5">
        <Skeleton w={52} h={52} r={26} />
        <div className="flex flex-1 flex-col gap-2">
          <Skeleton w={180} h={18} />
          <Skeleton w={220} h={12} />
          <Skeleton w={320} h={20} r={999} />
        </div>
      </div>
      <div className="flex flex-wrap items-start gap-5">
        <div className="card flex flex-[1_1_380px] flex-col gap-3 p-4.5">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} w="100%" h={14} />
          ))}
        </div>
        <div className="card flex flex-[1.4_1_480px] flex-col gap-3 p-4.5">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} w="100%" h={14} />
          ))}
        </div>
      </div>
    </Loading>
  )
}
