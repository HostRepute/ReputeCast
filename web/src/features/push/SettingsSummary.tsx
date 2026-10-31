import type { ReactNode } from 'react'
import { $api } from '@/api/client'
import { ButtonLink } from '@/components/ui/Button'
import { channelIcon } from '@/features/channels/channel'
import { useChannelSetUp } from '@/features/channels/hooks'
import { QualityBadge } from '@/features/whatsapp/common'
import { formatList, useT } from '@/i18n'
import { count, dayMonthYear } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './settings.messages'

/**
 * Settings › Providers adds push and WhatsApp to the summary (brief §6), as
 * it does the SMS providers: what each one is, its state, and where the rest
 * lives. Each shows once its channel is set up.
 */
export function ChannelSettingsSummary() {
  const push = useChannelSetUp('push')
  const whatsapp = useChannelSetUp('whatsapp')
  if (!push && !whatsapp) return null
  // Side by side when both are set up; one alone takes the row.
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,380px),1fr))] gap-5">
      <PushSummary />
      <WhatsAppSummary />
    </div>
  )
}

function Summary({ id, icon, title, sub, foot, open, children }: { id: string; icon: ReactNode; title: string; sub: string; foot: string; open: ReactNode; children: ReactNode }) {
  return (
    <section className="card flex flex-col" aria-labelledby={id}>
      <div className="flex flex-wrap items-center gap-2 border-b border-line-2 px-5 py-3">
        <span className="text-ink-3" aria-hidden="true">
          {icon}
        </span>
        <h2 className="h2" id={id}>
          {title}
        </h2>
        <span className="k">{sub}</span>
      </div>
      <dl className="kv px-5 py-3.5">{children}</dl>
      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-2 px-5 py-3">
        <span className="k">{foot}</span>
        {open}
      </div>
    </section>
  )
}

function PushSummary() {
  const t = useT()
  const { slug } = useWorkspace()
  const on = useChannelSetUp('push')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/setup', { params: { path: { workspace: slug } } }, { enabled: on, retry: false })
  const d = q.data
  if (!on || !d?.on || !d.keys) return null
  return (
    <Summary
      id="set-push-h"
      icon={channelIcon.push}
      title={t(m.pushTitle)}
      sub={t(m.pushBuiltIn)}
      foot={t(m.pushFoot)}
      open={
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers/push" params={{ workspace: slug }}>
          {t(m.pushOpen)}
        </ButtonLink>
      }
    >
      <dt>{t(m.keys)}</dt>
      <dd>{t(m.keysCreated, { date: dayMonthYear(d.keys.createdAt), name: d.keys.createdBy })}</dd>
      <dt>{t(m.sites)}</dt>
      <dd>{d.sites.length ? formatList(d.sites.map((s) => s.host)) : t(m.noSites)}</dd>
      <dt>{t(m.subscribers)}</dt>
      <dd>{t(m.subscribersLine, { total: count(d.subscribers.total), linked: count(d.subscribers.linked) })}</dd>
      <dt>{t(m.cap)}</dt>
      <dd>{t(m.capLine, { n: d.cap })}</dd>
    </Summary>
  )
}

function WhatsAppSummary() {
  const t = useT()
  const { slug } = useWorkspace()
  const on = useChannelSetUp('whatsapp')
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/whatsapp', { params: { path: { workspace: slug } } }, { enabled: on, retry: false })
  const d = q.data
  const a = d?.account
  if (!on || !d || !a) return null
  const tc = d.templateCounts
  return (
    <Summary
      id="set-wa-h"
      icon={channelIcon.whatsapp}
      title={t(m.waTitle)}
      sub={t(m.waProvider)}
      foot={t(m.waFoot)}
      open={
        <ButtonLink size="sm" variant="primary" to="/w/$workspace/providers/whatsapp" params={{ workspace: slug }}>
          {t(m.waOpen)}
        </ButtonLink>
      }
    >
      <dt>{t(m.business)}</dt>
      <dd>
        {t(m.businessLine, { name: a.businessName || a.displayName, number: a.number })}
      </dd>
      <dt>{t(m.quality)}</dt>
      <dd>
        <QualityBadge quality={a.quality} />
      </dd>
      <dt>{t(m.limit)}</dt>
      <dd>{t(m.limitLine, { n: a.messagingLimit })}</dd>
      <dt>{t(m.templates)}</dt>
      <dd>{t(m.templatesLine, { approved: tc.approved, pending: tc.pending, rejected: tc.rejected })}</dd>
      <dt>{t(m.consent)}</dt>
      <dd>{t(m.consentLine, { n: d.consent })}</dd>
    </Summary>
  )
}
