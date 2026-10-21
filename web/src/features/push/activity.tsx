import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { browserName } from './common'
import { m } from './activity.messages'

type Item = Schemas['ContactActivityItem']

/** The timeline icon of a notification or WhatsApp event: an icon of the profile's set and its tone. */
export function channelActivityIcon(it: Item): ['check' | 'eye' | 'cursor' | 'x' | 'minus' | 'bang' | 'mail' | 'flow', 'blue' | 'green' | 'purple' | 'amber' | 'red' | 'grey'] | null {
  if (it.type.startsWith('push_')) {
    switch (it.type) {
      case 'push_shown':
        return ['eye', 'blue']
      case 'push_clicked':
        return ['cursor', 'blue']
      case 'push_failed':
        return ['x', 'red']
      case 'push_skipped':
      case 'push_unsubscribed':
        return ['minus', 'grey']
      case 'push_held':
        return ['bang', 'amber']
      case 'push_subscribed':
        return ['check', 'green']
    }
    return ['check', 'green']
  }
  if (it.type.startsWith('whatsapp_consent_')) {
    return /opted_out|unsubscribed|withdrawn/.test(it.type) ? ['x', 'red'] : ['check', 'green']
  }
  if (it.type.startsWith('whatsapp_')) {
    switch (it.type) {
      case 'whatsapp_reply':
      case 'whatsapp_button':
        return ['mail', 'purple']
      case 'whatsapp_opt_out':
        return ['x', 'red']
      case 'whatsapp_failed':
      case 'whatsapp_undelivered':
        return ['x', 'red']
      case 'whatsapp_skipped':
      case 'whatsapp_cancelled':
        return ['minus', 'grey']
      case 'whatsapp_held':
        return ['bang', 'amber']
    }
    return it.source === 'automation' ? ['flow', 'purple'] : ['check', 'green']
  }
  return null
}

const pushTitle: Record<string, MessageDescriptor> = {
  push_sent: m.pushSent,
  push_shown: m.pushShown,
  push_clicked: m.pushClicked,
  push_failed: m.pushFailed,
  push_skipped: m.pushSkipped,
  push_held: m.pushHeld,
}

const waTitle: Record<string, MessageDescriptor> = {
  whatsapp_sent: m.waSent,
  whatsapp_delivered: m.waDelivered,
  whatsapp_undelivered: m.waUndelivered,
  whatsapp_failed: m.waFailed,
  whatsapp_skipped: m.waSkipped,
  whatsapp_held: m.waHeld,
  whatsapp_cancelled: m.waCancelled,
}

/**
 * Notifications and WhatsApp messages on the contact's timeline: the
 * title and the meta line, or null for other events.
 */
export function useChannelActivity(): (it: Item) => { title: ReactNode; meta: string[] } | null {
  const t = useT()
  const b = (c: ReactNode[]) => <b>{c}</b>
  const reason = (r: string) =>
    r === 'frequency_cap' ? t(m.reasonCap) : r === 'quiet_hours' ? t(m.reasonQuiet) : r === 'worker_missing' ? t(m.reasonWorker) : r ? t(m.reasonOther, { reason: r.replace(/_/g, ' ') }) : ''
  return (it) => {
    if (it.type.startsWith('push_')) {
      const browser = it.code ? t(browserName(it.code)) : ''
      if (it.type === 'push_subscribed' || it.type === 'push_unsubscribed') {
        const how =
          it.type === 'push_subscribed'
            ? it.source === 'member'
              ? t(m.linkedMember)
              : it.source === 'account'
                ? t(m.linkedAccount)
                : it.source === 'preferences'
                  ? t(m.linkedPreferences)
                  : it.source
                    ? t(m.linkedLink)
                    : ''
            : it.source === 'unsubscribed'
              ? t(m.removedUnsubscribed)
              : it.source === 'turned_off'
                ? t(m.removedTurnedOff)
                : it.source === 'gone'
                  ? t(m.removedGone)
                  : it.source === 'rotated'
                    ? t(m.removedRotated)
                    : (it.source ?? '').replace(/_/g, ' ')
        return {
          title: <T msg={it.type === 'push_subscribed' ? m.pushSubscribed : m.pushUnsubscribed} values={{ site: it.detail ?? '', b }} />,
          meta: [browser, how, it.answer ? t(m.prompt, { text: it.answer }) : ''].filter(Boolean),
        }
      }
      const d = pushTitle[it.type] ?? m.pushSent
      return {
        title: <T msg={d} values={{ title: it.answer || it.subject, b }} />,
        meta: [it.subject ? t(m.from, { name: it.subject }) : '', it.detail ?? '', browser, it.type === 'push_skipped' || it.type === 'push_held' || it.type === 'push_failed' ? reason(it.note ?? '') : ''].filter(
          Boolean,
        ),
      }
    }
    if (it.type.startsWith('whatsapp_consent_')) {
      const action = it.type.slice('whatsapp_consent_'.length)
      const d = /opted_out|unsubscribed|withdrawn/.test(action) ? m.waConsentWithdrawn : /subscribed|imported|confirmed|opted_in/.test(action) ? m.waConsentSubscribed : m.waConsentOther
      return {
        title: <T msg={d} values={{ list: it.list || '–', b }} />,
        meta: [
          it.source ?? '',
          it.detail ?? '',
          it.code ?? '',
          it.consentVersion ? t(m.consentVersion, { version: it.consentVersion }) : '',
          it.ip ? t(m.ip, { ip: it.ip }) : '',
          it.actor ? t(m.byActor, { name: it.actor }) : '',
          it.note ?? '',
        ].filter(Boolean),
      }
    }
    if (it.type.startsWith('whatsapp_')) {
      const kind = it.type.slice('whatsapp_'.length)
      if (['reply', 'button', 'opt_out', 'opt_in', 'help', 'keyword'].includes(kind)) {
        const keyword = it.detail || it.answer || ''
        const d =
          kind === 'reply'
            ? m.waReply
            : kind === 'button'
              ? m.waButton
              : kind === 'opt_out'
                ? m.waOptOut
                : kind === 'opt_in'
                  ? m.waOptIn
                  : kind === 'help'
                    ? m.waHelp
                    : m.waKeyword
        return {
          title: <T msg={d} values={{ text: it.answer ?? '', keyword, b }} />,
          meta: [it.subject ? t(m.from, { name: it.subject }) : '', it.list ?? ''].filter(Boolean),
        }
      }
      const d = kind === 'delivered' && it.note === 'read' ? m.waDeliveredRead : (waTitle[it.type] ?? m.waSent)
      return {
        title: <T msg={d} values={{ template: it.detail || t(m.waText), b }} />,
        meta: [it.subject ? t(m.from, { name: it.subject }) : '', it.answer ? `“${it.answer}”` : '', it.code ?? '', it.note && it.note !== 'read' ? it.note : ''].filter(Boolean),
      }
    }
    return null
  }
}
