import { Link, useNavigate } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, errorMessage, http, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { OptionCard } from '@/components/ui/Field'
import { Td } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { Truncate } from '@/components/ui/Truncate'
import { m as cm } from '@/features/campaigns/messages'
import { CampaignStatusBadge } from '@/features/campaigns/StatusBadge'
import { ChannelTag } from '@/features/channels/ChannelFilter'
import { useChannelSetUp } from '@/features/channels/hooks'
import { useT } from '@/i18n'
import { clock, count, money, pct, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './campaigns.messages'

type Item = Schemas['CampaignListItem']
type Channel = 'push' | 'whatsapp'

const editable = (s: Item['status']) => s === 'draft' || s === 'approval' || s === 'approved'

/** The composer, Review & send and report of a push or WhatsApp campaign. */
function paths(channel: Channel) {
  return channel === 'push'
    ? ({ edit: '/w/$workspace/campaigns/$campaignId/push', review: '/w/$workspace/campaigns/$campaignId/push-review', report: '/w/$workspace/campaigns/$campaignId/push-report' } as const)
    : ({
        edit: '/w/$workspace/campaigns/$campaignId/whatsapp',
        review: '/w/$workspace/campaigns/$campaignId/whatsapp-review',
        report: '/w/$workspace/campaigns/$campaignId/whatsapp-report',
      } as const)
}

/**
 * A push or WhatsApp campaign in the campaigns list (Campaigns, phase 8):
 * its channel tag, the sites (push) or Meta (WhatsApp) it goes through,
 * shown or delivered, and clicks; no opens.
 */
export function ChannelCampaignRow({ c, web }: { c: Item; web?: ReactNode }) {
  const t = useT()
  const { slug } = useWorkspace()
  const channel = c.channel as Channel
  const p = paths(channel)
  const params = { workspace: slug, campaignId: c.id }
  const cls = 'font-semibold text-ink no-underline'
  const at = c.status === 'scheduled' ? c.scheduledAt : c.startedAt
  const when = at
    ? c.local && c.localTime
      ? t(m.local, { date: shortDate(at), time: c.localTime })
      : `${shortDate(at)} · ${clock(at)}`
    : c.updatedAt !== c.createdAt && c.updatedBy
      ? t(cm.edited, { time: clock(c.updatedAt), name: c.updatedBy })
      : t(cm.created, { time: clock(c.createdAt) })
  const name = editable(c.status) ? (
    <Link to={p.edit} params={params} className={cls}>
      {c.name}
    </Link>
  ) : c.status === 'scheduled' ? (
    <Link to={p.review} params={params} className={cls}>
      {c.name}
    </Link>
  ) : (
    <Link to={p.report} params={params} className={cls}>
      {c.name}
    </Link>
  )
  return (
    <tr>
      <Td>
        <Truncate text={c.name}>{name}</Truncate>
        <div className="k">
          <ChannelTag channel={channel} />
          {[when, ...c.notes].join(' · ')}
        </div>
      </Td>
      <Td>
        <CampaignStatusBadge status={c.status} blocked={c.blocked} />
      </Td>
      {web ? <Td>{web}</Td> : null}
      <Td className={c.audience ? undefined : 'k'}>{c.audience || t(cm.notSet)}</Td>
      <Td align="right" num={c.recipients != null} className={c.recipients == null ? 'k' : undefined}>
        {c.recipients != null ? count(c.recipients) : '–'}
      </Td>
      <Td align="right" num={c.delivered != null} className={c.delivered == null ? 'k' : undefined}>
        <span title={channel === 'push' && c.delivered != null ? t(m.shownTitle) : undefined}>{c.delivered != null ? pct(c.delivered) : '–'}</span>
      </Td>
      <Td align="right" className="k">
        <span title={t(m.noOpens)}>–</span>
      </Td>
      <Td align="right" num={c.clicks != null} className={c.clicks == null ? 'k' : undefined}>
        {c.clicks != null ? pct(c.clicks) : '–'}
      </Td>
      <Td>
        {c.routes.length ? (
          <span className="inline-flex flex-wrap gap-x-1.5 gap-y-1 align-middle">
            {c.routes.map((r, i) => (
              <Badge key={r.providerId} tone={i === 0 ? 'purple' : 'neutral'}>
                {r.short} {count(r.sent || r.planned)}
              </Badge>
            ))}
            {channel === 'whatsapp' && c.cost != null ? <span className="k">{money(c.cost)}</span> : null}
          </span>
        ) : (
          <span className="k">{t(m.notSite)}</span>
        )}
      </Td>
      <Td>
        <div className="row-actions flex flex-wrap justify-end gap-1.5">
          <Actions c={c} channel={channel} />
        </div>
      </Td>
    </tr>
  )
}

function Actions({ c, channel }: { c: Item; channel: Channel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const p = paths(channel)
  const params = { workspace: slug, campaignId: c.id }
  if (editable(c.status) || c.status === 'scheduled') {
    const r = c.readiness
    return (
      <>
        {r && r.verdict !== 'ready' ? (
          <ButtonLink size="sm" to={p.review} params={params}>
            {r.verdict === 'blocked' ? t(cm.blockingIssues, { count: r.issues }) : t(cm.readiness, { count: r.issues })}
          </ButtonLink>
        ) : null}
        <ButtonLink size="sm" to={c.status === 'scheduled' ? p.review : p.edit} params={params}>
          {t(cm.edit)}
        </ButtonLink>
      </>
    )
  }
  return (
    <>
      <ButtonLink size="sm" to={p.report} params={params}>
        {t(cm.report)}
      </ButtonLink>
      <DuplicateButton c={c} channel={channel} />
    </>
  )
}

function DuplicateButton({ c, channel }: { c: Item; channel: Channel }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const reason = usePermissionReason('campaigns:edit')
  const push = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push-campaigns/{campaignId}/duplicate')
  const wa = $api.useMutation('post', '/api/v1/workspaces/{workspace}/whatsapp-campaigns/{campaignId}/duplicate')
  const busy = push.isPending || wa.isPending
  const go = (d: { id: string; name: string }) => {
    toast({ kind: 'success', title: t(m.duplicated, { name: d.name }) })
    void navigate({ to: paths(channel).edit, params: { workspace: slug, campaignId: d.id } })
  }
  const fail = (e: unknown) => toast({ kind: 'error', title: t(m.duplicateFailed), detail: errorMessage(e) })
  return (
    <Button
      size="sm"
      disabledReason={reason}
      loading={busy}
      onClick={() =>
        channel === 'push'
          ? push.mutate({ params: { path: { workspace: slug, campaignId: c.id } } }, { onSuccess: go, onError: fail })
          : wa.mutate({ params: { path: { workspace: slug, campaignId: c.id } } }, { onSuccess: go, onError: fail })
      }
    >
      {t(cm.duplicate)}
    </Button>
  )
}

/** New campaign's push and WhatsApp options, once the channel is set up. */
export function useChannelOptions(): { push: boolean; whatsapp: boolean } {
  return { push: useChannelSetUp('push'), whatsapp: useChannelSetUp('whatsapp') }
}

export function ChannelOptions({ kind, onKind, show }: { kind: string; onKind: (k: Channel) => void; show: { push: boolean; whatsapp: boolean } }) {
  const t = useT()
  return (
    <>
      {show.push ? (
        <OptionCard name="campaign-kind" value="push" checked={kind === 'push'} onChange={() => onKind('push')} title={t(m.newPush)} description={t(m.newPushBody)}>
          <span className="k">{t(m.newPushHint)}</span>
        </OptionCard>
      ) : null}
      {show.whatsapp ? (
        <OptionCard name="campaign-kind" value="whatsapp" checked={kind === 'whatsapp'} onChange={() => onKind('whatsapp')} title={t(m.newWhatsApp)} description={t(m.newWhatsAppBody)}>
          <span className="k">{t(m.newWhatsAppHint)}</span>
        </OptionCard>
      ) : null}
    </>
  )
}

/** Creates a push or WhatsApp campaign and opens it; a failure throws for the dialog to show. */
export function useCreateChannelCampaign(): (channel: Channel) => Promise<void> {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  return async (channel) => {
    if (channel === 'push') {
      const { data } = await http.POST('/api/v1/workspaces/{workspace}/push-campaigns', { params: { path: { workspace: slug } }, body: { name: t(m.untitledPush) } })
      void navigate({ to: '/w/$workspace/campaigns/$campaignId/push', params: { workspace: slug, campaignId: data!.id } })
      return
    }
    const { data } = await http.POST('/api/v1/workspaces/{workspace}/whatsapp-campaigns', { params: { path: { workspace: slug } }, body: { name: t(m.untitledWhatsApp) } })
    void navigate({ to: '/w/$workspace/campaigns/$campaignId/whatsapp', params: { workspace: slug, campaignId: data!.id } })
  }
}
