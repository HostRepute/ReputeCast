import { Link } from '@tanstack/react-router'
import { $api, type Schemas } from '@/api/client'
import { Icon, type IconName } from '@/components/icons/Icon'
import { defineMessages, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { mt } from './messages'
import { secs, useStatusLabel } from './shared'

const mc = defineMessages('transactional.contact', {
  transactional: 'Transactional',
  via: 'via',
  afterCall: '{time} after the call',
  contentVisible: 'content visible with transactional:content',
  contentHidden: 'content needs transactional:content',
  notifications: 'Notifications from {product}',
  categoryOn: '{name} on',
  categoryOff: '{name} off',
  notificationsLine: 'Optional categories the contact can turn off in the preference center · {essential} emails always arrive',
  notificationsLineNoEssential: 'Optional categories the contact can turn off in the preference center',
  and: ' and ',
})

const API = '/api/v1/workspaces/{workspace}/transactional' as const

type Activity = Schemas['TransactionalContactActivity']
type Message = Activity['messages'][number]
export type ActivityFilter = 'all' | 'deliveries' | 'engagement' | 'consent'

/** A contact's transactional mail, its Optional categories and the product's name, for people who can see them. */
export function useContactTransactional(contactId: string) {
  const { slug, me } = useWorkspace()
  const can = me.permissions.includes('transactional:view')
  const path = { params: { path: { workspace: slug } } }
  const q = $api.useQuery('get', `${API}/contacts/{contactId}`, { params: { path: { workspace: slug, contactId } } }, { enabled: can })
  const settings = $api.useQuery('get', `${API}/settings`, path, { enabled: can })
  const cats = $api.useQuery('get', `${API}/categories`, path, { enabled: can })
  const essential = (cats.data?.items ?? []).filter((c) => c.kind === 'essential').map((c) => c.name.toLowerCase())
  return { data: q.data, product: settings.data?.productName || me.workspace.name, essential }
}

export type Entry<T> = { kind: 'item'; at: string; it: T } | { kind: 'tx'; at: string; m: Message } | { kind: 'notifications' }

/**
 * The contact's timeline with its transactional mail in it (ContactProfile):
 * messages among the deliveries by date — within the events loaded so far,
 * all of them once the list is complete — and the Optional categories'
 * state after the newest one.
 */
export function withTransactional<T extends { at: string }>(items: T[], tx: Activity | undefined, filter: ActivityFilter, complete: boolean): Entry<T>[] {
  const out: Entry<T>[] = items.map((it) => ({ kind: 'item', at: it.at, it }))
  if (!tx) return out
  const oldest = items.length ? Date.parse(items[items.length - 1]!.at) : -Infinity
  if (filter === 'all' || filter === 'deliveries') {
    for (const m of tx.messages) if (complete || Date.parse(m.at) >= oldest) out.push({ kind: 'tx', at: m.at, m })
  }
  out.sort((a, b) => Date.parse((b as { at: string }).at) - Date.parse((a as { at: string }).at))
  if ((filter === 'all' || filter === 'consent') && tx.categories.length) {
    const first = out.findIndex((e) => e.kind === 'tx')
    out.splice(first >= 0 ? first + 1 : 0, 0, { kind: 'notifications' })
  }
  return out
}

/** How many of the contact's transactional messages the timeline lists. */
export function transactionalCount(tx: Activity | undefined, filter: ActivityFilter): number {
  return tx && (filter === 'all' || filter === 'deliveries') ? tx.messages.length : 0
}

function look(status: Message['status']): { bg: string; fg: string; icon: IconName } {
  switch (status) {
    case 'delivered':
    case 'sent':
      return { bg: '#DCFCE7', fg: '#166534', icon: 'check' }
    case 'deferred':
    case 'queued':
      return { bg: '#FEF3C7', fg: '#92400E', icon: 'clock' }
    case 'bounced':
    case 'complained':
    case 'failed':
      return { bg: '#FEE2E2', fg: '#991B1B', icon: 'alertCircle' }
    case 'test':
      return { bg: '#EDE9FE', fg: '#5B21B6', icon: 'check' }
  }
  return { bg: '#EEF0F3', fg: '#4B5563', icon: 'ban' }
}

/** "Delivered Password reset [Transactional] via Postmark · Today 10:42 · Acme Cloud · 1.3 s after the call". */
export function TransactionalActivityRow({ m, when, product, canViewContent }: { m: Message; when: (iso: string) => string; product: string; canViewContent: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const status = useStatusLabel()
  const l = look(m.status)
  const meta = [when(m.at), product, m.deliveryMs !== undefined ? t(mc.afterCall, { time: secs(m.deliveryMs) }) : '', canViewContent ? t(mc.contentVisible) : t(mc.contentHidden)]
  return (
    <li className="ev">
      <span className="ic" style={{ background: l.bg, color: l.fg }}>
        <Icon name={l.icon} size={13} />
      </span>
      <div className="flex flex-1 flex-col gap-0.5">
        <span>
          <b>{status(m.status)}</b>{' '}
          <Link to="/w/$workspace/transactional/messages/$messageId" params={{ workspace: slug, messageId: m.id }}>
            {m.raw ? t(mt.raw) : (m.template?.name ?? m.subject)}
          </Link>{' '}
          <span className="badge mut h-5">{t(mc.transactional)}</span>
          {m.provider ? (
            <>
              {` ${t(mc.via)} `}
              <span className="badge mut h-5">{m.provider}</span>
            </>
          ) : null}
        </span>
        <span className="k">{meta.filter(Boolean).join(' · ')}</span>
      </div>
    </li>
  )
}

/** "Notifications from Acme Cloud · Reminders on · Product updates on". */
export function NotificationsRow({ tx, product, essential }: { tx: Activity; product: string; essential: string[] }) {
  const t = useT()
  return (
    <li className="ev">
      <span className="ic" style={{ background: '#EEF0F3', color: '#4B5563' }}>
        <svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden="true">
          <path d="M3 8h10" />
        </svg>
      </span>
      <div className="flex flex-1 flex-col gap-0.5">
        <span>
          <b>{t(mc.notifications, { product })}</b> {tx.categories.map((c) => (c.on ? t(mc.categoryOn, { name: c.name }) : t(mc.categoryOff, { name: c.name }))).join(' · ')}
        </span>
        <span className="k">{essential.length ? t(mc.notificationsLine, { essential: essential.join(t(mc.and)) }) : t(mc.notificationsLineNoEssential)}</span>
      </div>
    </li>
  )
}
