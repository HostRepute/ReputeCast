import { Link, useNavigate } from '@tanstack/react-router'
import type { ReactNode } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button, ButtonLink } from '@/components/ui/Button'
import { Td, Th } from '@/components/ui/Table'
import { toast } from '@/components/ui/Toast'
import { Truncate } from '@/components/ui/Truncate'
import { m as cm } from '@/features/campaigns/messages'
import { CampaignStatusBadge } from '@/features/campaigns/StatusBadge'
import { ChannelTag } from '@/features/channels/ChannelFilter'
import { useT } from '@/i18n'
import { clock, count, money, pct, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { useRefreshSms } from './common'
import { m } from './messages'

type Item = Schemas['CampaignListItem']

const editable = (s: Item['status']) => s === 'draft' || s === 'approval' || s === 'approved'

/** Where a text campaign's name leads: the composer, Review & send, or the report. */
function NameLink({ c }: { c: Item }) {
  const { slug } = useWorkspace()
  const params = { workspace: slug, campaignId: c.id }
  const cls = 'font-semibold text-ink no-underline'
  if (editable(c.status))
    return (
      <Link to="/w/$workspace/campaigns/$campaignId/sms" params={params} className={cls}>
        {c.name}
      </Link>
    )
  if (c.status === 'scheduled')
    return (
      <Link to="/w/$workspace/campaigns/$campaignId/sms-review" params={params} className={cls}>
        {c.name}
      </Link>
    )
  return (
    <Link to="/w/$workspace/campaigns/$campaignId/sms-report" params={params} className={cls}>
      {c.name}
    </Link>
  )
}

/** The campaigns table's columns with SMS chosen: opt-outs and cost instead of opens (brief §6). */
export function SmsCampaignHead() {
  const t = useT()
  return (
    <tr>
      <Th>{t(cm.campaign)}</Th>
      <Th>{t(cm.status)}</Th>
      <Th>{t(cm.audience)}</Th>
      <Th align="right">{t(cm.recipients)}</Th>
      <Th align="right">{t(cm.delivered)}</Th>
      <Th align="right">{t(cm.clicks)}</Th>
      <Th align="right">{t(m.listOptOuts)}</Th>
      <Th align="right">{t(m.listCost)}</Th>
      <Th>{t(cm.sendingProviders)}</Th>
      <Th>
        <span className="vh">{t(cm.actions)}</span>
      </Th>
    </tr>
  )
}

/**
 * A text campaign in the campaigns list (Campaigns, phase 8): the SMS tag,
 * the local send time, texts per provider ("Twilio 659 · Vonage 195") and
 * no opens, since texts have none. With SMS chosen, the opt-outs and the
 * cost take the opens' place.
 */
export function SmsCampaignRow({ c, textColumns, web }: { c: Item; textColumns?: boolean; web?: ReactNode }) {
  const t = useT()
  const at = c.status === 'scheduled' ? c.scheduledAt : c.startedAt
  const when = at
    ? c.local && c.localTime
      ? t(m.listLocal, { date: shortDate(at), time: c.localTime })
      : `${shortDate(at)} · ${clock(at)}`
    : c.updatedAt !== c.createdAt && c.updatedBy
          ? t(cm.edited, { time: clock(c.updatedAt), name: c.updatedBy })
          : t(cm.created, { time: clock(c.createdAt) })
  return (
    <tr>
      <Td>
        <Truncate text={c.name}>
          <NameLink c={c} />
        </Truncate>
        <div className="k">
          <ChannelTag channel="sms" />
          {[when, ...c.notes].join(' · ')}
        </div>
      </Td>
      <Td>
        <CampaignStatusBadge status={c.status} blocked={c.blocked} />
      </Td>
      {/* The website's Web column (X18), while it's on: texts aren't posts. */}
      {web ? <Td>{web}</Td> : null}
      <Td className={c.audience ? undefined : 'k'}>{c.audience || t(cm.notSet)}</Td>
      <Td align="right" num={c.recipients != null} className={c.recipients == null ? 'k' : undefined}>
        {c.recipients != null ? count(c.recipients) : '–'}
      </Td>
      <Td align="right" num={c.delivered != null} className={c.delivered == null ? 'k' : undefined}>
        {c.delivered != null ? pct(c.delivered) : '–'}
      </Td>
      {textColumns ? null : (
        <Td align="right" className="k">
          <span title={t(m.listNoOpens)}>–</span>
        </Td>
      )}
      <Td align="right" num={c.clicks != null} className={c.clicks == null ? 'k' : undefined}>
        {c.clicks != null ? pct(c.clicks) : '–'}
      </Td>
      {textColumns ? (
        <>
          <Td align="right" num={c.optOuts != null} className={c.optOuts == null ? 'k' : undefined}>
            {c.optOuts != null ? count(c.optOuts) : '–'}
          </Td>
          <Td align="right" num={c.cost != null} className={c.cost == null ? 'k' : undefined}>
            {c.cost != null ? money(c.cost) : '–'}
          </Td>
        </>
      ) : null}
      <Td>
        {c.routes.length ? (
          <span className="inline-flex flex-wrap gap-x-1.5 gap-y-1 align-middle">
            {c.routes.map((r, i) => (
              <Badge key={r.providerId} tone={i === 0 ? 'purple' : 'neutral'}>
                {r.short} {count(r.sent || r.planned)}
              </Badge>
            ))}
          </span>
        ) : (
          <span className="k">{t(m.listNoRoute)}</span>
        )}
      </Td>
      <Td>
        <div className="row-actions flex flex-wrap justify-end gap-1.5">
          <Actions c={c} />
        </div>
      </Td>
    </tr>
  )
}

/** Duplicate: a new draft with the same text and audience, opened in the composer. */
export function DuplicateTextButton({ campaignId, label, small = true, primary }: { campaignId: string; label?: string; small?: boolean; primary?: boolean }) {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  const refresh = useRefreshSms()
  const editReason = usePermissionReason('campaigns:edit')
  const duplicate = $api.useMutation('post', '/api/v1/workspaces/{workspace}/sms-campaigns/{campaignId}/duplicate', {
    onSuccess: (d) => {
      void refresh()
      toast({ kind: 'success', title: t(m.duplicated, { name: d.name }) })
      void navigate({ to: '/w/$workspace/campaigns/$campaignId/sms', params: { workspace: slug, campaignId: d.id } })
    },
    onError: (e) => toast({ kind: 'error', title: t(m.duplicateFailed), detail: errorMessage(e) }),
  })
  return (
    <Button size={small ? 'sm' : undefined} variant={primary ? 'primary' : undefined} disabledReason={editReason} loading={duplicate.isPending} onClick={() => duplicate.mutate({ params: { path: { workspace: slug, campaignId } } })}>
      {label ?? t(cm.duplicate)}
    </Button>
  )
}

function Actions({ c }: { c: Item }) {
  const t = useT()
  const { slug } = useWorkspace()
  const params = { workspace: slug, campaignId: c.id }
  if (editable(c.status) || c.status === 'scheduled') {
    const r = c.readiness
    return (
      <>
        {r && r.verdict !== 'ready' ? (
          <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/sms-review" params={params}>
            {r.verdict === 'blocked' ? t(cm.blockingIssues, { count: r.issues }) : t(cm.readiness, { count: r.issues })}
          </ButtonLink>
        ) : null}
        <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/sms" params={params}>
          {t(cm.edit)}
        </ButtonLink>
      </>
    )
  }
  if (c.status === 'sending' || c.status === 'paused')
    return (
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/sms-report" params={params}>
        {t(cm.liveMonitor)}
      </ButtonLink>
    )
  return (
    <>
      <ButtonLink size="sm" to="/w/$workspace/campaigns/$campaignId/sms-report" params={params}>
        {t(cm.report)}
      </ButtonLink>
      <DuplicateTextButton campaignId={c.id} />
    </>
  )
}
