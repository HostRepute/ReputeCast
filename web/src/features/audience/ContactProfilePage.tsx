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
import { clock, count, dayMonthYear, money, notificationTime, pct, shortDate, usDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { useInvalidateAudience } from './ContactDialogs'
import { fileName, saveBlob } from './download'
import { consentLabel, consentTone, mailboxColor, providerColor, tierClass, tierLabel, type Consent, type Tier } from './labels'
import { brandOf } from '@/features/languages/LanguagesCard'
import { languageLabel } from '@/features/languages/composerParts'
import { ContactLanguageField, consentVersionText, deliveryLanguage } from '@/features/page-languages/contacts'
import { mc } from '@/features/page-languages/messages'
import { T, t, useI18n, useT } from '@/i18n'
import { common } from '@/i18n/common'
import { m as am } from './messages'
import { m } from './profile.messages'
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
import { channelActivityIcon, useChannelActivity } from '@/features/push/activity'
import { ContactChannelCards } from '@/features/push/ContactCards'

type Profile = Schemas['ContactProfile']
type Item = Schemas['ContactActivityItem']

/** The non-empty parts, for "a · b · c" lines. */
const present = (...xs: (string | null | undefined | false)[]) => xs.filter((x): x is string => Boolean(x))

const fullDate = (iso: string) => dayMonthYear(new Date(iso))
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
  const tr = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contact}', { params: { path: { workspace: slug, contact: contactId } } })
  const p = q.data
  const name = p ? p.name || p.email : tr(m.contact)
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
        <nav aria-label={tr(m.breadcrumb)} className="flex items-center gap-2 text-[13px]">
          <Link to="/w/$workspace/audience" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
            {tr(am.title)}
          </Link>
          <span className="k" aria-hidden="true">
            /
          </span>
          <Link to="/w/$workspace/audience" params={{ workspace: slug }} className="text-ink-3 no-underline hover:text-ink-3">
            {tr(m.contacts)}
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
                      ? { kind: 'success', title: t(m.confirmationSentTo, { email: p.email }), detail: t(m.confirmationSentBody) }
                      : { kind: 'info', title: t(m.noSystemProvider), detail: r.confirmUrl ? t(m.shareLink, { url: r.confirmUrl }) : undefined },
                  )
                } catch (e) {
                  toast({ kind: 'error', title: t(m.resendFailed), detail: errorMessage(e) })
                }
              }}
            >
              {tr(m.resendOptin)}
            </Button>
            <Button disabledReason={exportReason} onClick={() => void exportData(slug, contactId, p.email)}>
              {tr(m.exportData)}
            </Button>
            <Button className="dline" disabledReason={p.suppression ? tr(m.alreadySuppressed) : editReason} onClick={() => setSuppressing(true)}>
              {tr(m.suppress)}
            </Button>
          </>
        ) : null
      }
    >
      {q.isError && !p ? (
        <ErrorPanel title={tr(m.loadFailed)} actions={<Button onClick={() => q.refetch()}>{tr(common.retry)}</Button>}>
          {errorMessage(q.error)} {tr(m.nothingChanged)}
        </ErrorPanel>
      ) : !p ? (
        <ProfileSkeleton />
      ) : (
        <>
          {p.suppression ? (
            <Note tone="bad" label={tr(m.suppressedLabel)}>
              {tr(m.onTheList, {
                email: p.email,
                reason: tr(m.suppressionReason, { reason: p.suppression.reason, label: p.suppression.reason.replace('_', ' ') }),
                date: fullDate(p.suppression.at),
              })}{' '}
              {p.suppression.scope === 'marketing' ? tr(m.scopeMarketing) : tr(m.scopeAll)}
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
              {/* Push subscriptions and WhatsApp with its customer service window (X22). */}
              <ContactChannelCards contactId={p.id} name={[p.firstName, p.lastName].filter(Boolean).join(' ') || p.email} />
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
    toast({ kind: 'success', title: t(m.exportedAll, { email }), detail: t(m.exportInAudit) })
  } catch (e) {
    toast({ kind: 'error', title: t(m.exportFailed), detail: errorMessage(e) })
  }
}

function Summary({ p }: { p: Profile }) {
  const tr = useT()
  const tier = p.tier as Tier
  const consent = p.consent as Consent
  const s = p.stats
  const lastOpen = s.lastOpenAt ? (s.lastOpenMachine ? tr(m.machineOpen, { date: shortDate(s.lastOpenAt) }) : shortDate(s.lastOpenAt)) : tr(m.never)
  return (
    <section className="card flex flex-wrap items-center gap-4.5 px-5.5 py-5" aria-label={tr(m.summary)}>
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
          {p.roleAddress ? <span className="badge warn">{tr(m.roleAddress)}</span> : null}
        </div>
      </div>
      <div className="grid grid-cols-4 gap-4 max-md:grid-cols-[repeat(auto-fit,minmax(140px,1fr))]">
        <Stat label={tr(m.emails90)} value={count(s.emails90d)} />
        <Stat
          label={tr(m.humanOpens)}
          value={s.humanOpenRate !== undefined ? tr(m.humanOpensRate, { count: count(s.humanOpens), rate: pct(Math.round(s.humanOpenRate), 0) }) : count(s.humanOpens)}
        />
        <Stat label={tr(m.clicks)} value={count(s.clicks)} />
        <Stat label={tr(m.lastOpen)} value={lastOpen} />
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
    return <span className="n">{money(n, n % 1 === 0 ? 0 : 2)}</span>
  }
  if (f.type === 'number') return <span className="n">{f.value}</span>
  return <span>{f.value}</span>
}

function FieldsCard({ p }: { p: Profile }) {
  const tr = useT()
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
          {tr(m.fields)}
        </h2>
        <Button size="sm" disabledReason={reason} onClick={() => setEditing(true)}>
          {tr(common.edit)}
        </Button>
      </div>
      <AddressCheckRow contactId={p.id} />
      {p.fields.map((f) =>
        f.key === 'language' ? (
          <ContactLanguageField key={f.key} p={p} />
        ) : (
          <div key={f.key} className="pkv">
            <span className="k">
              {f.type === 'builtin' ? tr(m.builtinField, { key: f.key, label: f.label }) : f.label}
              {f.custom ? (
                <>
                  {' '}
                  <span className="mono">{f.type === 'text' || f.type === 'select' ? tr(m.custom) : tr(m.customType, { type: f.type })}</span>
                </>
              ) : null}
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
                toast({ kind: 'error', title: t(m.tagFailed), detail: errorMessage(e) })
              }
            }}
          />
        ) : (
          <Button size="sm" className="border-dashed" disabledReason={reason} onClick={() => setAdding(true)}>
            {tr(m.addTag)}
          </Button>
        )}
      </div>
      {editing ? <EditFieldsDialog p={p} onClose={() => setEditing(false)} onSave={save} /> : null}
    </section>
  )
}

function TagInput({ onSave, onCancel }: { onSave: (tag: string) => Promise<void>; onCancel: () => void }) {
  const tr = useT()
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
        {tr(m.newTag)}
      </label>
      <input id="new-tag" className="in h-[30px] w-[140px]" value={tag} onChange={(e) => setTag(e.target.value)} placeholder={tr(m.tagPlaceholder)} maxLength={60} autoFocus />
      <Button size="sm" type="submit" variant="primary">
        {tr(m.add)}
      </Button>
      <Button size="sm" onClick={onCancel}>
        {tr(common.cancel)}
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
      toast({ kind: 'success', title: t(m.fieldsSaved) })
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
      title={t(m.editFields)}
      description={t(m.editFieldsBody)}
      footer={
        <>
          <Button onClick={onClose}>{t(common.cancel)}</Button>
          <Button variant="primary" loading={busy} onClick={() => void submit()}>
            {t(m.saveFields)}
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
          <Field label={t(m.firstName)}>
            <Input value={vals.firstName} onChange={(e) => set('firstName', e.target.value)} />
          </Field>
          <Field label={t(m.lastName)}>
            <Input value={vals.lastName} onChange={(e) => set('lastName', e.target.value)} />
          </Field>
        </div>
        <Field label={t(m.company)}>
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
            <Field key={f.key} label={f.key} hint={f.type === 'date' ? t(m.dateHint, { example: '2026-10-30' }) : f.display === 'currency' ? t(m.currencyHint) : undefined}>
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
        return t(m.consentNever)
      case 'pending':
        return t(m.consentWaiting)
      case 'unsubscribed':
        return t(m.consentUnsubscribed)
    }
    return t(m.consentSubscribed)
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
        parts.push(t(m.partSubscribedVia, { source: src, date: fullDate(e.at) }))
        break
      case 'subscribed':
        parts.push(t(e.source === 'preference_center' ? m.partAddedIn : m.partSubscribedVia, { source: src, date: fullDateTime(e.at) }))
        if (e.detail) parts.push(e.detail)
        break
      case 'confirmed': {
        const how = e.source === 'repermission' ? 'repermission' : parts.length ? 'again' : 'first'
        parts.push(t(m.partConfirmed, { how, source: src, date: fullDateTime(e.at), fromIp: e.ip ? 'yes' : 'no', ip: e.ip ?? '' }))
        if (e.consentVersion) parts.push(e.language && consentLanguage ? consentLanguage(e.consentVersion, e.language) : t(m.partConsentText, { version: e.consentVersion }))
        break
      }
      case 'unsubscribed':
        parts.push(t(m.partUnsubscribed, { inSource: src ? 'yes' : 'no', date: fullDate(e.at), source: src ?? '' }))
        break
      case 'resubscribed':
        parts.push(t(m.partResubscribed, { date: fullDate(e.at) }))
        break
      case 'optin_sent':
        parts.push(t(m.partOptinSent, { date: fullDate(e.at) }))
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
      return <Badge tone="ok">{confirmed ? t(m.badgeConfirmed) : t(m.badgeSubscribed)}</Badge>
    }
    case 'pending':
      return <Badge tone="warn">{t(m.badgePending)}</Badge>
    case 'unsubscribed':
      return <Badge tone="neutral">{t(m.badgeUnsubscribed)}</Badge>
  }
  return <Badge tone="neutral">{t(m.badgeNotSubscribed)}</Badge>
}

function ConsentCard({ p }: { p: Profile }) {
  const { slug } = useWorkspace()
  const t = useT()
  const { locale } = useI18n()
  const multilingual = p.language.options.filter((o) => o.offered).length > 1
  const consentLanguage = multilingual ? (version: string, language: string) => consentVersionText(version, language, t, locale) : undefined
  const link = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/{contact}/preference-link')
  const allDouble = p.lists.length > 0 && p.lists.every((l) => l.doubleOptIn)
  const first = p.firstName || p.name
  return (
    <section className="card flex flex-col" aria-labelledby="consent-h">
      <div className="flex items-center justify-between px-4.5 py-3.5">
        <h2 className="h2" id="consent-h">
          {t(m.consentTitle)}
        </h2>
        <span className="k">{allDouble ? t(m.allDouble) : t(m.someSingle)}</span>
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
          {t(m.preferences, {
            frequency: p.preferences.frequency,
            paused: p.preferences.pausedUntil ? 'yes' : 'no',
            date: p.preferences.pausedUntil ? shortDate(p.preferences.pausedUntil) : '',
            fewer: p.preferences.fewerEmails ? 'yes' : 'no',
          })}
        </span>
        <Button
          size="sm"
          loading={link.isPending}
          onClick={async () => {
            try {
              const r = await link.mutateAsync({ params: { path: { workspace: slug, contact: p.id } } })
              window.open(r.url, '_blank', 'noopener,noreferrer')
            } catch (e) {
              toast({ kind: 'error', title: t(m.prefsFailed), detail: errorMessage(e) })
            }
          }}
        >
          {t(m.openPrefs, { named: first ? 'yes' : 'no', name: first })}
        </Button>
      </div>
    </section>
  )
}

type Filter = 'all' | 'deliveries' | 'engagement' | 'consent' | 'paid'

function ActivityCard({ contactId, total }: { contactId: string; total: number }) {
  const tr = useT()
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
          {tr(m.activity)}
        </h2>
        <div className="flex gap-1.5 max-md:flex-wrap" role="group" aria-label={tr(m.show)}>
          {(
            [
              ['all', tr(m.filterAll)],
              ['deliveries', tr(m.filterDeliveries)],
              ['engagement', tr(m.filterEngagement)],
              ['consent', tr(m.filterConsent)],
              ...(hasPaid ? [['paid', tr(m.filterPaid)]] : []),
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
          <ErrorPanel title={tr(m.activityFailed)} actions={<Button size="sm" onClick={() => q.refetch()}>{tr(common.retry)}</Button>}>
            {errorMessage(q.error)}
          </ErrorPanel>
        </div>
      ) : q.isPending ? (
        <Loading label={tr(m.loadingActivity)} className="flex flex-col">
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
          {filter === 'all' ? tr(m.noActivity) : tr(m.noEvents, { filter })}
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
            {tr(m.eventsCount, { listed: count(listed), shown: count(shown) })}
          </span>
          {q.hasNextPage ? (
            <Button size="sm" loading={q.isFetchingNextPage} onClick={() => void q.fetchNextPage()}>
              {tr(m.loadMore)}
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
  // Notifications and WhatsApp messages (X22).
  const channel = channelActivityIcon(it)
  if (channel) return channel
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

function ActivityRow({ it }: { it: Item }) {
  const answerActivity = useAnswerActivity()
  // Store orders and abandoned checkouts (X20).
  const storeActivity = useStoreActivity()
  const smsActivity = useSmsActivity()
  const [icon, toneKey] = eventIcon(it)
  const c = tone[toneKey]
  const tr = useT()
  const b = (chunks: ReactNode[]) => <b>{chunks}</b>
  const { locale } = useI18n()
  const { me } = useWorkspace()
  // The language version a delivery sent (ContactProfile, LanguageStates).
  const dl = deliveryLanguage(it, tr, locale, brandOf(me.workspace.name))
  let title: ReactNode
  let meta: string[] = [dateTime(it.at)]
  // Notifications and WhatsApp messages (X22).
  const channelActivity = useChannelActivity()
  const answer = answerActivity(it) ?? storeActivity(it) ?? channelActivity(it)
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
    const by = it.actor ? tr(m.actBy, { name: it.actor }) : ''
    switch (it.type) {
      case 'confirmed':
        title = (
          <>
            <T msg={m.actConfirmed} values={{ list: it.list ?? '', b }} />
            {it.detail ? ` · ${it.detail}` : ''}
          </>
        )
        meta = [...meta, ...present(it.note, it.ip ? tr(m.actIp, { ip: it.ip }) : '')]
        break
      case 'imported':
        title = (
          <>
            <T msg={m.actImported} values={{ source: (it.sourceLabel ?? '').replace(/ import$/, ''), b }} />
            {it.detail ? ` · ${it.detail}` : ''}
          </>
        )
        meta = [...meta, ...present(by, it.note)]
        break
      case 'subscribed':
      case 'resubscribed':
        title = <T msg={it.type === 'subscribed' ? m.actSubscribed : m.actResubscribed} values={{ list: it.list ?? '', source: it.sourceLabel ?? '', b }} />
        meta = [...meta, ...present(it.detail, it.note)]
        break
      case 'unsubscribed':
        title = it.list ? <T msg={m.actUnsubscribedFrom} values={{ list: it.list, b }} /> : <T msg={m.actUnsubscribedAll} values={{ b }} />
        meta = [...meta, ...present(it.sourceLabel ? tr(m.actIn, { source: it.sourceLabel }) : '', it.detail)]
        break
      case 'optin_sent':
        title = it.list ? <T msg={m.actOptinSentFor} values={{ list: it.list, b }} /> : <T msg={m.actOptinSent} values={{ b }} />
        meta = [...meta, ...present(it.note, by)]
        break
      case 'suppressed':
        title = <b>{tr(m.actSuppressed)}</b>
        meta = [...meta, ...present(it.detail, it.note, by)]
        break
      default:
        title = <b>{tr(m.actConsentOther, { type: it.type, label: it.type.replace('_', ' ') })}</b>
        meta = [...meta, ...present(it.detail, it.note)]
    }
  } else if (it.type === 'open') {
    title = <T msg={it.machine ? m.actOpenedMachine : m.actOpenedHuman} values={{ subject: it.subject, b }} />
    const mins = it.afterDeliveryMin
    meta = [
      ...meta,
      it.machine ? 'Apple Mail Privacy Protection' : it.client ?? '',
      mins !== undefined && mins !== null ? (mins < 60 ? tr(m.minutesAfter, { n: mins }) : tr(m.hoursAfter, { n: Math.round(mins / 60) })) : '',
    ].filter(Boolean)
  } else if (it.type === 'click') {
    title = it.linkLabel ? (
      <T msg={m.actClickedLink} values={{ link: it.linkLabel, subject: it.subject, b }} />
    ) : (
      <T msg={m.actClickedAny} values={{ subject: it.subject, b }} />
    )
    meta = [...meta, ...present(it.url ?? '')]
  } else if (it.type === 'unsubscribe') {
    title = <T msg={m.actUnsubscribedSubject} values={{ subject: it.subject, b }} />
  } else {
    title = (
      <>
        <T msg={m.actDelivery} values={{ type: it.type, label: it.type, subject: it.subject, b }} />
        {it.variant ? ` · ${tr(m.actVariant, { variant: it.variant })}` : ''}
        {dl.label ? ` · ${dl.label}${dl.fallback ? ' ·' : ''}` : ''}
        {it.provider ? (
          <>
            {' '}
            <T msg={m.actVia} values={{ provider: <ProviderBadge p={it.provider} /> }} />
          </>
        ) : it.imported ? (
          ` · ${tr(m.actSentFrom, { origin: it.imported })}`
        ) : null}
        {it.failover ? (
          <>
            <span className="inline-block w-1.5" />
            <span className="badge warn h-5">{tr(m.failover)}</span>
          </>
        ) : null}
      </>
    )
    const code = [it.code, it.diagnostic].filter(Boolean).join(' ')
    meta = [...meta, ...present(code, it.routeNote ?? '', it.retriedDeliveredAt ? tr(m.retried, { time: clock(it.retriedDeliveredAt) }) : '', dl.why)]
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
  const tr = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidateAudience()
  const suppress = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/{contact}/suppress')
  const [reason, setReason] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const submit = async () => {
    if (!reason.trim()) {
      setErr(t(m.sayWhy))
      return
    }
    try {
      await suppress.mutateAsync({ params: { path: { workspace: slug, contact: p.id } }, body: { reason: reason.trim() } })
      await invalidate()
      onClose()
      toast({ kind: 'success', title: t(m.suppressedToast, { email: p.email }), detail: t(m.suppressedBody) })
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      size="sm"
      title={tr(m.suppressTitle, { email: p.email })}
      description={tr(m.suppressBody)}
      footer={
        <>
          <Button onClick={onClose}>{tr(common.cancel)}</Button>
          <Button variant="danger-primary" loading={suppress.isPending} onClick={() => void submit()}>
            {tr(m.suppress)}
          </Button>
        </>
      }
    >
      <Field label={tr(m.reason)} error={err ?? undefined} hint={tr(m.reasonHint)}>
        <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder={tr(m.reasonPlaceholder)} maxLength={300} />
      </Field>
    </Dialog>
  )
}

function ProfileSkeleton() {
  const tr = useT()
  return (
    <Loading label={tr(m.loading)} className="flex flex-col gap-5">
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
