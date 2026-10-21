import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { useT, type MessageDescriptor } from '@/i18n'
import { m } from './activity.messages'

type Item = Schemas['ContactActivityItem']

/** The icon and its colour on the timeline, in the profile's own sets. */
export type SmsEventIcon = ['check' | 'cursor' | 'x' | 'minus' | 'bang' | 'mail', 'green' | 'blue' | 'red' | 'grey' | 'amber']

/** A text's event's icon, or null for other events. */
export function smsEventIcon(it: Item): SmsEventIcon | null {
  if (!it.type.startsWith('sms_')) return null
  switch (it.type) {
    case 'sms_delivered':
      return ['check', 'green']
    case 'sms_sent':
      return ['check', 'grey']
    case 'sms_undelivered':
    case 'sms_failed':
    case 'sms_opt_out':
    case 'sms_consent_opted_out':
    case 'sms_consent_unsubscribed':
      return ['x', 'red']
    case 'sms_skipped':
    case 'sms_cancelled':
      return ['minus', 'grey']
    case 'sms_held':
    case 'sms_consent_confirmation_failed':
      return ['bang', 'amber']
    case 'sms_click':
      return ['cursor', it.machine ? 'grey' : 'blue']
    case 'sms_help':
    case 'sms_reply':
    case 'sms_yes':
    case 'sms_consent_confirmation_sent':
      return ['mail', 'blue']
  }
  return ['check', 'green']
}

const status: Record<string, MessageDescriptor> = {
  sms_sent: m.sent,
  sms_delivered: m.delivered,
  sms_undelivered: m.undelivered,
  sms_failed: m.failed,
  sms_skipped: m.skipped,
  sms_held: m.held,
  sms_cancelled: m.cancelled,
}

const consent: Record<string, MessageDescriptor> = {
  sms_consent_subscribed: m.consentSubscribed,
  sms_consent_pending: m.consentPending,
  sms_consent_confirmed: m.consentConfirmed,
  sms_consent_opted_out: m.consentOptedOut,
  sms_consent_opted_in: m.consentOptedIn,
  sms_consent_unsubscribed: m.consentUnsubscribed,
  sms_consent_imported: m.consentImported,
  sms_consent_confirmation_sent: m.confirmationSent,
  sms_consent_confirmation_failed: m.confirmationFailed,
}

const reason: Record<string, MessageDescriptor> = {
  no_consent: m.rNo_consent,
  no_phone: m.rNo_phone,
  suppressed: m.rSuppressed,
  recent: m.rRecent,
  quiet_hours: m.rQuiet_hours,
  no_route: m.rNo_route,
  sender_pending: m.rSender_pending,
  pending_sender: m.rPending_sender,
  provider_down: m.rProvider_down,
  budget: m.rBudget,
  cancelled: m.rCancelled,
}

const present = (...parts: (string | undefined | null | false)[]) => parts.filter((p): p is string => Boolean(p))

/**
 * Texts on the contact's timeline: each text with its status, clicks on
 * its links, replies and keywords, and SMS consent with its evidence.
 * Null for events that aren't texts.
 */
export function useSmsActivity(): (it: Item) => { title: ReactNode; meta: string[] } | null {
  const t = useT()
  return (it) => {
    if (!it.type.startsWith('sms_')) return null
    const why = it.note && reason[it.note] ? t(reason[it.note]!) : ''
    if (status[it.type]) {
      return {
        title: (
          <>
            <b>{t(status[it.type]!)}</b> {it.subject}
          </>
        ),
        meta: present(it.provider?.name, it.detail ? t(m.from, { sender: it.detail }) : '', why, it.code),
      }
    }
    if (it.type === 'sms_click') {
      return {
        title: (
          <>
            <b>{t(m.clicked)}</b> {it.subject}
          </>
        ),
        meta: present(it.url, it.machine ? t(m.linkPreview) : ''),
      }
    }
    if (consent[it.type]) {
      return {
        title: <b>{t(consent[it.type]!, { list: it.list || t(m.everyList) })}</b>,
        meta: present(
          it.detail,
          it.code ? t(m.texted, { keyword: it.code }) : '',
          it.consentVersion ? t(m.version, { version: it.consentVersion }) : '',
          it.ip ? t(m.ip, { ip: it.ip }) : '',
          it.actor ? t(m.byActor, { name: it.actor }) : '',
          it.note,
        ),
      }
    }
    // Inbound texts: a keyword or a reply.
    if (it.type === 'sms_reply') {
      return { title: <b>{t(m.replied, { text: it.answer ?? '' })}</b>, meta: present(it.subject, it.provider?.name, t(m.replyNote)) }
    }
    const note = it.type === 'sms_opt_out' ? t(m.optOutNote) : it.type === 'sms_opt_in' ? t(m.optInNote) : it.type === 'sms_help' ? t(m.helpNote) : it.type === 'sms_yes' ? t(m.yesNote) : ''
    return { title: <b>{t(m.texted, { keyword: it.detail || it.answer || '' })}</b>, meta: present(it.subject, note, it.list) }
  }
}
