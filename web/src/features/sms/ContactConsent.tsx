import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Dialog } from '@/components/ui/Dialog'
import { Field, Input } from '@/components/ui/Field'
import { Note } from '@/components/ui/Note'
import { Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { m as cm } from '@/features/channels/messages'
import { hasWhatsAppColumn, WhatsAppConsentCell } from '@/features/push/ContactCards'
import { m as wm } from '@/features/push/contact.messages'
import { useT, type MessageDescriptor } from '@/i18n'
import { clock, dayMonthDate } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { useRefreshSms } from './common'
import { m } from './contact.messages'
import './sms.css'

type Channels = Schemas['SmsContactChannels']
type Row = Schemas['SmsContactList']

const emailWord: Record<Row['email'], MessageDescriptor> = {
  subscribed: m.subscribed,
  pending: m.pending,
  unsubscribed: m.unsubscribed,
  not_subscribed: m.notSubscribed,
}
const smsWord: Record<Row['sms'], MessageDescriptor> = {
  subscribed: m.subscribed,
  pending: m.pending,
  opted_out: m.optedOut,
  not_subscribed: m.notSubscribed,
}
const tone = (s: string): Tone => (s === 'subscribed' ? 'ok' : s === 'pending' ? 'warn' : s === 'opted_out' || s === 'unsubscribed' ? 'bad' : 'neutral')
const lineWord: Record<string, MessageDescriptor> = { mobile: m.lineMobile, landline: m.lineLandline, toll_free: m.lineTollFree, voip: m.lineVoip, unknown: m.lineUnknown }

/**
 * Consent per list and per channel on the contact profile (SmsStates ›
 * Contact profile): email and SMS are separate, each change keeps its
 * evidence. Shown once texting is set up or the contact has a phone.
 */
export function ChannelConsentCard({ contactId, name }: { contactId: string; name: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contactId}/channels', { params: { path: { workspace: slug, contactId } } })
  const [editingPhone, setEditingPhone] = useState(false)
  const d = q.data
  // WhatsApp (X22) adds its column once it's connected or a list has its consent.
  const wa = Boolean(d && hasWhatsAppColumn(d))
  if (!d || (!d.smsSetUp && !d.phone && !wa)) return null
  return (
    <section className="card flex flex-col" aria-labelledby="channels-h">
      <div className="sec">
        <div className="flex min-w-0 flex-col gap-0.5">
          <h2 className="h2" id="channels-h">
            {t(m.title, { name })}
          </h2>
          <span className="k">{t(wa ? wm.tableLead : m.lead)}</span>
        </div>
      </div>
      <div className="pad flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2 text-[13px]">
          {d.phone ? (
            <>
              <span className="mono font-semibold">{d.display}</span>
              <span className="k">
                {[d.country, d.lineType ? t(lineWord[d.lineType] ?? m.lineUnknown) : '', d.timeZone].filter(Boolean).join(' · ')}
              </span>
            </>
          ) : (
            <span className="k">{t(m.noPhone)}</span>
          )}
          {d.canEdit ? (
            <Button size="sm" className="ml-auto" onClick={() => setEditingPhone(true)}>
              {d.phone ? t(m.changePhone) : t(m.addPhone)}
            </Button>
          ) : null}
        </div>
        {d.suppressed ? (
          <Note tone="bad">{t(m.suppressed, { reason: d.suppressed.reason, reasonText: d.suppressed.reason.replace(/_/g, ' '), date: dayMonthDate(d.suppressed.at) })}</Note>
        ) : null}
        <div className="overflow-x-auto">
          <table className="sms-cons">
            <thead>
              <tr>
                <Th>{t(m.list)}</Th>
                <Th>{t(cm.email)}</Th>
                <Th>{t(cm.sms)}</Th>
                {wa ? <Th>{t(cm.whatsapp)}</Th> : null}
              </tr>
            </thead>
            <tbody>
              {d.lists.map((l) => (
                <tr key={l.listId}>
                  <td className="font-semibold">{l.name}</td>
                  <td>
                    <Badge tone={tone(l.email)}>{l.emailConfirmed ? t(m.confirmed) : t(emailWord[l.email])}</Badge>
                    {l.emailAt ? <span className="k">{[`${dayMonthDate(l.emailAt)} ${clock(l.emailAt)}`, l.emailSourceLabel ?? l.emailSource].filter(Boolean).join(' · ')}</span> : null}
                  </td>
                  <td>
                    <SmsCell contactId={contactId} l={l} d={d} />
                  </td>
                  {wa ? (
                    <td>
                      <WhatsAppConsentCell contactId={contactId} l={l} d={d} />
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <span className="k">{t(m.foot)}</span>
      </div>
      {editingPhone ? <PhoneDialog contactId={contactId} d={d} onClose={() => setEditingPhone(false)} /> : null}
    </section>
  )
}

function SmsCell({ contactId, l, d }: { contactId: string; l: Row; d: Channels }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const set = $api.useMutation('put', '/api/v1/workspaces/{workspace}/contacts/{contactId}/sms-consent', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.consentSaved, { list: l.name }) })
    },
    onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
  })
  const evidence = [
    l.smsAt ? `${dayMonthDate(l.smsAt)} ${clock(l.smsAt)}` : '',
    l.smsKeyword ? t(m.byKeyword, { keyword: l.smsKeyword }) : '',
    l.smsPage ?? '',
    l.smsVersion ? t(m.consentVersion, { version: l.smsVersion }) : '',
    l.smsIp ?? '',
    l.confirmation ? t(m.confirmation, { status: l.confirmation, time: l.confirmationAt ? clock(l.confirmationAt) : '' }).trim() : '',
  ].filter(Boolean)
  return (
    <>
      <Badge tone={tone(l.sms)}>{t(smsWord[l.sms])}</Badge>
      {evidence.length ? <span className="k">{evidence.join(' · ')}</span> : null}
      {d.canEdit && d.phone && l.sms !== 'opted_out' ? (
        <button
          type="button"
          className="sms-lbtn mt-1 text-[12.5px]"
          disabled={set.isPending}
          onClick={() =>
            set.mutate({
              params: { path: { workspace: slug, contactId } },
              body: { listId: l.listId, status: l.sms === 'subscribed' ? 'not_subscribed' : 'subscribed' },
            })
          }
        >
          {l.sms === 'subscribed' ? t(m.withdraw) : t(m.record)}
        </button>
      ) : null}
    </>
  )
}

function PhoneDialog({ contactId, d, onClose }: { contactId: string; d: Channels; onClose: () => void }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshSms()
  const [phone, setPhone] = useState(d.display || '')
  const [error, setError] = useState<string>()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/contacts/{contactId}/phone', {
    onSuccess: () => {
      void refresh()
      toast({ kind: 'success', title: t(m.phoneSaved) })
      onClose()
    },
    onError: (e) => setError(errorMessage(e)),
  })
  return (
    <Dialog
      open
      onOpenChange={(o) => !o && onClose()}
      title={d.phone ? t(m.changePhone) : t(m.addPhone)}
      description={t(m.phoneText)}
      footer={
        <>
          <Button onClick={onClose}>{t(m.cancel)}</Button>
          <Button variant="primary" loading={save.isPending} onClick={() => save.mutate({ params: { path: { workspace: slug, contactId } }, body: { phone: phone.trim() } })}>
            {t(m.save)}
          </Button>
        </>
      }
    >
      <Field label={t(m.phone)} hint={t(m.phoneHint)} error={error}>
        <Input className="mono" value={phone} inputMode="tel" autoComplete="off" onChange={(e) => setPhone(e.target.value)} />
      </Field>
    </Dialog>
  )
}
