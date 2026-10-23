import { useQueryClient } from '@tanstack/react-query'
import { Fragment, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Avatar } from '@/components/ui/Avatar'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Dialog } from '@/components/ui/Dialog'
import { toast } from '@/components/ui/Toast'
import { useI18n, useT } from '@/i18n'
import { ago, clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { browserName } from './common'
import { m } from './contact.messages'
import './push.css'
import '@/features/whatsapp/whatsapp.css'

/**
 * The contact profile's channel cards (design expansion phase 8, X22): the
 * browsers subscribed to push, and WhatsApp with its customer service
 * window, consent and thread. Each hides itself where the channel isn't set
 * up and the contact has nothing on it.
 */
export function ContactChannelCards({ contactId, name }: { contactId: string; name: string }) {
  return (
    <>
      <PushCard contactId={contactId} name={name} />
      <WhatsAppCard contactId={contactId} />
    </>
  )
}

function PushCard({ contactId, name }: { contactId: string; name: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/push', { params: { path: { workspace: slug, contactId } } }, { retry: false })
  const remove = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/contacts/{contactId}/push/{subscriptionId}')
  const [removing, setRemoving] = useState<Schemas['PushContactSubscription'] | null>(null)
  const d = q.data
  if (!d || (!d.pushOn && !d.subscriptions.length)) return null
  const how = (s: Schemas['PushContactSubscription']) =>
    s.linkedBy === 'member' ? t(m.linkedMember) : s.linkedBy === 'account' ? t(m.linkedAccount) : s.linkedBy === 'preferences' ? t(m.linkedPreferences) : s.linkedBy ? t(m.linkedLink) : ''
  return (
    <section className="card flex flex-col px-4.5 py-4" aria-labelledby="push-card-h">
      <div className="flex flex-col gap-0.5">
        <h2 className="h2" id="push-card-h">
          {t(m.pushTitle)}
        </h2>
        <span className="k">{t(m.pushIntro, { name })}</span>
      </div>
      {d.subscriptions.length ? (
        <ul className="m-0 mt-2 flex list-none flex-col p-0">
          {d.subscriptions.map((s) => (
            <li key={s.id} className="px-mrow">
              <span className="tx">
                <b>{t(browserName(s.browser))}</b>
                <span className="k">
                  {[s.site, t(m.pushSince, { date: dayMonthDate(s.since) }), how(s), s.lastSeen ? t(m.pushLastSeen, { ago: ago(s.lastSeen) }) : '', s.timeZone].filter(Boolean).join(' · ')}
                </span>
              </span>
              {d.canEdit ? (
                <Button size="sm" variant="danger" onClick={() => setRemoving(s)}>
                  {t(m.remove)}
                </Button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : (
        <span className="k mt-2">{t(m.pushNone)}</span>
      )}
      <ConfirmDialog
        open={Boolean(removing)}
        onOpenChange={(o) => !o && setRemoving(null)}
        badge={t(m.remove)}
        title={t(m.removeTitle)}
        description={removing ? t(m.removeText, { browser: t(browserName(removing.browser)), site: removing.site }) : ''}
        confirmLabel={t(m.remove)}
        onConfirm={async () => {
          if (!removing) return
          await remove.mutateAsync({ params: { path: { workspace: slug, contactId, subscriptionId: removing.id } } })
          toast({ kind: 'deleted', title: t(m.removed) })
          void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && x.queryKey[1].includes('/contacts/') })
        }}
      />
    </section>
  )
}

function WhatsAppCard({ contactId }: { contactId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/whatsapp', { params: { path: { workspace: slug, contactId } } }, { retry: false })
  const reply = $api.useMutation('post', '/api/v1/workspaces/{workspace}/contacts/{contactId}/whatsapp/reply')
  const [text, setText] = useState('')
  const { locale } = useI18n()
  const src = useSource()
  const d = q.data
  if (!d) return null
  const subscribed = d.lists.filter((l) => l.status !== 'not_subscribed')
  const country = regionName(locale, d.country)
  if (!d.connected && !subscribed.length && !d.thread.length) return null
  const first = d.name.split(' ')[0] || d.name
  const w = d.window
  const untilText = w.until ? `${dayMonthDate(w.until)}, ${clock(w.until)}` : ''
  // A day chip before the first message of each day.
  const newDay = d.thread.map((it, i) => i === 0 || dayMonthDate(it.at) !== dayMonthDate(d.thread[i - 1]!.at))
  return (
    <section className="card flex flex-col gap-3.5 px-4.5 py-4" aria-labelledby="wa-card-h">
      <h2 className="h2" id="wa-card-h">
        {t(m.waTitle)}
      </h2>
      {!d.connected ? (
        <span className="k">{t(m.notConnected)}</span>
      ) : (
        <div className={w.open ? 'note i' : 'note n'} role="note">
          <div className="px-note-body">
            <b>{w.open && w.until ? t(m.waOpen, { until: untilText }) : w.lastAt ? t(m.waClosed, { name: first }) : t(m.waNever, { name: first })}</b>
            <div>
              {w.open && w.until ? t(m.waOpenText, { name: first, time: clock(w.until) }) : w.lastAt ? t(m.waClosedText, { name: first }) : t(m.waNeverText)}
            </div>
          </div>
        </div>
      )}
      <dl className="kv">
        <dt>{t(m.phone)}</dt>
        <dd>
          {d.phone ? (
            <>
              <span className="mono">{d.display}</span> ·{' '}
              {d.lineType && d.lineType !== 'unknown'
                ? t(m.phoneLine, { country, type: d.lineType, typeText: d.lineType.replace(/_/g, ' '), zone: d.timeZone })
                : t(m.phoneLineNoType, { country, zone: d.timeZone })}
            </>
          ) : (
            t(m.phoneNone)
          )}
        </dd>
        <dt>{t(m.consent)}</dt>
        <dd className="flex flex-col gap-1">
          {subscribed.length ? (
            subscribed.map((l) => (
              <span key={l.listId} className="flex flex-wrap items-center gap-1.5">
                <Badge tone={l.status === 'subscribed' ? 'ok' : l.status === 'opted_out' ? 'bad' : 'warn'}>
                  {t(l.status === 'subscribed' ? m.subscribed : l.status === 'opted_out' ? m.optedOut : m.pending)}
                </Badge>
                <span className="k">
                  {l.status === 'opted_out'
                    ? t(m.optedOutLine, { list: l.name, how: l.keyword && l.at ? t(m.optedOutKeyword, { keyword: l.keyword, date: dayMonthDate(l.at) }) : src(l.source) })
                    : l.at
                      ? t(m.consentLine, { list: l.name, date: dayMonthDate(l.at), source: src(l.source) })
                      : t(m.consentLineNoDate, { list: l.name, source: src(l.source) })}
                </span>
              </span>
            ))
          ) : (
            <span className="k">{t(m.consentNone)}</span>
          )}
        </dd>
        {w.lastAt ? (
          <>
            <dt>{t(m.lastMessage)}</dt>
            <dd>{t(m.lastWrote, { name: first, date: dayMonthDate(w.lastAt), time: clock(w.lastAt) })}</dd>
          </>
        ) : null}
      </dl>
      {d.suppressed ? (
        <span className="err">
          {d.suppressed.keyword
            ? t(m.suppressedKeyword, { reason: d.suppressed.reason, reasonText: d.suppressed.reason.replace(/_/g, ' '), keyword: d.suppressed.keyword })
            : t(m.suppressed, { reason: d.suppressed.reason, reasonText: d.suppressed.reason.replace(/_/g, ' ') })}
        </span>
      ) : null}
      {d.thread.length ? (
        <div className="wa-thread">
          <div className="hd">
            <Avatar name={d.name} size={30} />
            <span className="flex flex-col leading-[1.3]">
              <span className="font-semibold">{d.name}</span>
              <span className="k mono">{d.display}</span>
            </span>
            <span className="ml-auto">
              {w.open && w.until ? (
                <Badge tone="info" dot>
                  {t(m.windowBadge, { until: untilText })}
                </Badge>
              ) : (
                <Badge tone="neutral">{t(m.windowClosedBadge)}</Badge>
              )}
            </span>
          </div>
          <div className="bd wa-chat" style={{ border: 0, borderRadius: 0, maxWidth: 'none' }}>
            {d.thread.map((it, i) => {
              return (
                <Fragment key={it.id}>
                  {newDay[i] ? <span className="day">{dayMonthDate(it.at)}</span> : null}
                  <div className={it.direction === 'in' ? 'bub me' : 'bub'}>
                    {it.text}
                    <span className="tm">{it.kind === 'template' ? `${clock(it.at)} · ${t(m.template)}` : clock(it.at)}</span>
                  </div>
                </Fragment>
              )
            })}
          </div>
          {d.canReply ? (
            <form
              className="reply"
              onSubmit={(e) => {
                e.preventDefault()
                reply.mutate(
                  { params: { path: { workspace: slug, contactId } }, body: { text: text.trim() } },
                  {
                    onSuccess: () => {
                      setText('')
                      toast({ kind: 'success', title: t(m.replied) })
                      void q.refetch()
                      void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && x.queryKey[1].includes('/activity') })
                    },
                  },
                )
              }}
            >
              <label className="text-[12px] font-semibold text-ink-2" htmlFor="wa-reply">
                {t(m.replyLabel)}
              </label>
              <div className="flex items-end gap-2">
                <textarea id="wa-reply" className="in min-w-0 flex-1" rows={2} value={text} maxLength={4096} aria-label={t(m.replyTo, { name: d.name })} onChange={(e) => setText(e.target.value)} />
                <Button variant="primary" type="submit" loading={reply.isPending} busyLabel={t(m.sending)} disabled={!text.trim()}>
                  {t(m.sendReply)}
                </Button>
              </div>
              {reply.isError ? (
                <span className="err" role="alert">
                  {errorMessage(reply.error)}
                </span>
              ) : null}
            </form>
          ) : w.lastAt && !w.open ? (
            <div className="reply">
              <span className="k">{t(m.replyClosed)}</span>
            </div>
          ) : null}
        </div>
      ) : null}
      <span className="k">{t(m.timelineNote)}</span>
    </section>
  )
}

/** Where consent came from, in words: "from the summit form", "added by the team". */
function useSource(): (s: string) => string {
  const t = useT()
  return (s) =>
    s.startsWith('form:')
      ? t(m.srcForm, { name: s.slice(5).replace(/[-_]/g, ' ') })
      : s === 'team'
        ? t(m.srcTeam)
        : s === 'keyword'
          ? t(m.srcKeyword)
          : s.startsWith('import')
            ? t(m.srcImport)
            : s === 'api'
              ? t(m.srcApi)
              : s
}

const waWord = { subscribed: m.subscribed, pending: m.pending, opted_out: m.optedOut, not_subscribed: m.notSubscribed } as const
const waTone = { subscribed: 'ok', pending: 'warn', opted_out: 'bad', not_subscribed: 'neutral' } as const

/** The consent per list table shows a WhatsApp column once WhatsApp is connected, or while a list still has WhatsApp consent. */
export function hasWhatsAppColumn(d: Schemas['SmsContactChannels']): boolean {
  return d.whatsAppSetUp || d.lists.some((l) => l.whatsApp !== 'not_subscribed')
}

/**
 * A list's WhatsApp consent in the profile's consent per list (SmsStates ›
 * Contact profile): "Subscribed · 2 Oct 14:06 · same page · WhatsApp
 * consent text v1", or an opt-out with its keyword.
 */
export function WhatsAppConsentCell({ contactId, l, d }: { contactId: string; l: Schemas['SmsContactList']; d: Schemas['SmsContactChannels'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const src = useSource()
  const refresh = useRefreshContact()
  const [recording, setRecording] = useState(false)
  const withdraw = $api.useMutation('put', '/api/v1/workspaces/{workspace}/contacts/{contactId}/whatsapp/consent', {
    onSuccess: () => {
      refresh()
      toast({ kind: 'success', title: t(m.consentSavedOn, { list: l.name }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const evidence = [
    l.whatsAppAt ? `${dayMonthDate(l.whatsAppAt)} ${clock(l.whatsAppAt)}` : '',
    l.whatsAppKeyword ? t(m.bySending, { keyword: l.whatsAppKeyword }) : '',
    l.whatsAppPage ? (l.whatsAppPage === l.smsPage ? t(m.samePage) : l.whatsAppPage) : '',
    l.whatsAppVersion ? t(m.waConsentVersion, { version: l.whatsAppVersion }) : '',
  ].filter(Boolean)
  if (l.whatsApp !== 'not_subscribed' && !l.whatsAppPage && !l.whatsAppKeyword && l.whatsAppSource) evidence.push(src(l.whatsAppSource))
  // A keyword opt-out can only be undone by the contact; there's nothing to record without a number.
  const editable = d.canEdit && Boolean(d.phone) && l.whatsApp !== 'opted_out'
  return (
    <>
      <Badge tone={waTone[l.whatsApp]}>{t(waWord[l.whatsApp])}</Badge>
      {evidence.length ? <span className="k">{evidence.join(' · ')}</span> : null}
      {editable ? (
        <button
          type="button"
          className="sms-lbtn mt-1 text-[12.5px]"
          disabled={withdraw.isPending}
          onClick={() =>
            l.whatsApp === 'not_subscribed'
              ? setRecording(true)
              : withdraw.mutate({ params: { path: { workspace: slug, contactId } }, body: { listId: l.listId, status: 'not_subscribed' } })
          }
        >
          {l.whatsApp === 'not_subscribed' ? t(m.recordConsent) : t(m.withdrawConsent)}
        </button>
      ) : null}
      {recording ? <RecordDialog contactId={contactId} list={l} onClose={() => setRecording(false)} /> : null}
    </>
  )
}

/** "United States" for "US", in the app's language. */
function regionName(locale: string, code: string): string {
  try {
    return new Intl.DisplayNames([locale], { type: 'region' }).of(code) ?? code
  } catch {
    return code
  }
}

/** Records WhatsApp consent on one list with its evidence: where it was given, the exact text and when. */
function RecordDialog({ contactId, list, onClose }: { contactId: string; list: Schemas['SmsContactList']; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshContact()
  const [source, setSource] = useState('')
  const [wording, setWording] = useState('')
  const [at, setAt] = useState(() => {
    const d = new Date()
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
    return d.toISOString().slice(0, 16)
  })
  const [note, setNote] = useState('')
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/contacts/{contactId}/whatsapp/consent')
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={t(m.recordTitle, { list: list.name })}
      description={t(m.consentText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button
            variant="primary"
            loading={save.isPending}
            disabled={!source.trim() || !wording.trim() || !at}
            onClick={() =>
              save.mutate(
                {
                  params: { path: { workspace: slug, contactId } },
                  body: {
                    listId: list.listId,
                    status: 'subscribed',
                    note: note.trim() || undefined,
                    evidence: { source: source.trim(), consentText: wording.trim(), at: new Date(at).toISOString() },
                  },
                },
                {
                  onSuccess: () => {
                    toast({ kind: 'success', title: t(m.consentSavedOn, { list: list.name }) })
                    refresh()
                    onClose()
                  },
                },
              )
            }
          >
            {t(m.consentSave)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="lbl">
          <label htmlFor="wa-c-source">{t(m.consentSource)}</label>
          <input id="wa-c-source" className="in w-full" value={source} placeholder={t(m.consentSourcePh)} maxLength={200} onChange={(e) => setSource(e.target.value)} />
        </div>
        <div className="lbl">
          <label htmlFor="wa-c-wording">{t(m.consentWording)}</label>
          <textarea id="wa-c-wording" className="in w-full" rows={2} value={wording} maxLength={1000} onChange={(e) => setWording(e.target.value)} />
        </div>
        <div className="lbl">
          <label htmlFor="wa-c-at">{t(m.consentAt)}</label>
          <input id="wa-c-at" className="in w-full" type="datetime-local" value={at} onChange={(e) => setAt(e.target.value)} />
        </div>
        <div className="lbl">
          <label htmlFor="wa-c-note">{t(m.consentNote)}</label>
          <input id="wa-c-note" className="in w-full" value={note} maxLength={300} onChange={(e) => setNote(e.target.value)} />
        </div>
        {save.isError ? (
          <span className="err" role="alert">
            {errorMessage(save.error)}
          </span>
        ) : null}
      </div>
    </Dialog>
  )
}

/** Refetches what the profile shows of a contact: its cards and timeline. */
function useRefreshContact(): () => void {
  const qc = useQueryClient()
  return () => void qc.invalidateQueries({ predicate: (x) => typeof x.queryKey[1] === 'string' && x.queryKey[1].includes('/contacts/') })
}
