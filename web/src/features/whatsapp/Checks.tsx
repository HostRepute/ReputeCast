import type { ReactNode } from 'react'
import type { Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { ago, andList, clock, count, dayMonthDate, money, orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { categoryBadge, qualityLabel, statusBadge } from './common'
import { m } from './campaign.messages'

type Check = Schemas['WhatsAppCheck']

const name: Record<Check['key'], MessageDescriptor> = {
  account: m.kAccount,
  consent: m.kConsent,
  template: m.kTemplate,
  quality: m.kQuality,
  limit: m.kLimit,
  cost: m.kCost,
  approval: m.kApproval,
}

const badge: Record<Check['status'], { tone: Tone; msg: MessageDescriptor }> = {
  pass: { tone: 'ok', msg: m.pass },
  warn: { tone: 'warn', msg: m.warn },
  block: { tone: 'bad', msg: m.block },
}

/** "tomorrow" when a moment is on the next day, otherwise "later". */
export function dayWord(t: ReturnType<typeof useT>, at?: string): string {
  if (!at) return t(m.later)
  const d = new Date(at)
  const now = new Date()
  const tomorrow = new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1)
  return d.getFullYear() === tomorrow.getFullYear() && d.getMonth() === tomorrow.getMonth() && d.getDate() === tomorrow.getDate() ? t(m.tomorrow) : t(m.later)
}

/** One pre-send check of a WhatsApp campaign, worded from what it found. */
export function CheckRow({ c, business, split }: { c: Check; business?: { displayName: string; number: string }; split?: Schemas['WhatsAppLimitSplit'] }) {
  const t = useT()
  const { me } = useWorkspace()
  let title: ReactNode = t(name[c.key])
  let detail: ReactNode
  switch (c.key) {
    case 'account':
      detail =
        c.status === 'pass'
          ? business
            ? t(m.accountOk, { name: business.displayName, number: business.number })
            : t(m.accountConnected)
          : c.accountStatus === 'token_rejected'
            ? t(m.accountToken)
            : c.accountStatus === 'waiting'
              ? t(m.accountWaiting)
              : t(m.accountNone)
      break
    case 'consent':
      detail = c.count ? t(m.consentLine, { count: c.count, lists: andList(c.lists) }) : t(m.consentNone)
      break
    case 'template':
      detail =
        c.template && c.templateStatus
          ? t(m.templateLine, {
              name: c.template,
              status: t(statusBadge[c.templateStatus as keyof typeof statusBadge]?.msg ?? m.kTemplate),
              category: t(categoryBadge[c.category as keyof typeof categoryBadge]?.msg ?? m.kTemplate),
            })
          : t(m.templateNone)
      break
    case 'quality':
      if (c.quality === 'medium' || c.quality === 'low') {
        title = t(m.kQualityMedium, { quality: qualityLabel(t, c.quality) })
        detail = t(m.qualityDrop)
      } else {
        detail = c.quality ? t(m.qualityLine, { quality: qualityLabel(t, c.quality), ago: ago(c.checkedAt ?? null) }) : '–'
      }
      break
    case 'limit': {
      const s = c.split
      if (!s) {
        detail = '–'
        break
      }
      if (s.later > 0) {
        const laterAt = s.laterAt ? clock(s.laterAt) : ''
        detail =
          s.usedBy && s.usedByAt
            ? t(m.limitSplit, {
                left: count(s.left),
                limit: count(s.limit),
                used: count(s.usedByN ?? s.used),
                name: s.usedBy,
                at: clock(s.usedByAt),
                now: count(s.now),
                later: count(s.later),
                laterDay: dayWord(t, s.laterAt),
                laterAt,
              })
            : t(m.limitSplitPlain, { left: count(s.left), limit: count(s.limit), now: count(s.now), later: count(s.later), laterDay: dayWord(t, s.laterAt), laterAt })
      } else {
        detail = t(m.limitOk, { limit: s.limit, n: count(s.now) })
      }
      break
    }
    case 'cost':
      detail = c.cost ? t(split?.later ? m.costLineSplit : m.costLine, { amount: money(c.cost.amount) }) : '–'
      break
    case 'approval':
      detail =
        c.status === 'pass' ? (
          <T msg={m.approvalOk} values={{ name: c.author ?? '', perm: <span className="mono">whatsapp:send</span> }} />
        ) : (
          <T
            msg={m.approvalNeeds}
            values={{ name: c.author ?? '', role: me.workspace.roleName, perms: <span className="mono">{andList(c.missing)}</span>, helpers: orList(c.helpers) }}
          />
        )
      break
  }
  const b = badge[c.status]
  return (
    <div className={c.status === 'warn' ? 'px-chk w' : c.status === 'block' ? 'px-chk b' : 'px-chk'}>
      <Badge tone={b.tone}>{t(b.msg)}</Badge>
      <div className="tx">
        <span className="font-semibold">{title}</span>
        <span className="k text-ink-2!">{detail}</span>
      </div>
    </div>
  )
}

/** "21 Oct, 09:00" for the split's days. */
export function dayLabel(d: string): string {
  return dayMonthDate(d)
}
