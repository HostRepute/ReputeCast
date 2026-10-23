import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { errorMessage, http } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { Icon } from '@/components/icons/Icon'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { clock, shortDate } from '@/lib/format'
import { roleName } from '@/lib/roles'
import { useMe, useWorkspace } from '@/lib/session'
import { useComposer } from '@/features/composer/context'
import { IssueStatusBadge, useDigest, useIssueInfo, useRefreshDigest, type IssueInfo } from '../shared'
import { isDigestTemplate } from './DigestMode'
import { dc } from './messages'

/** Whether the composer edits one of a recurring digest's issues (prepared from the template). */
export function isDigestIssue(campaign: { digestRole?: string; digestId?: string }): boolean {
  return (campaign.digestRole === 'issue' || campaign.digestRole === 'next') && Boolean(campaign.digestId)
}

/** The issue the composer edits, if it edits one. */
export function useDigestIssue(): IssueInfo | undefined {
  const c = useComposer()
  return useIssueInfo(isDigestIssue(c.campaign) ? c.campaign.id : undefined)
}

/** "Issue · Fri 17 Oct, 08:00": the header's label for an issue. */
export function IssueHeaderLabel({ fallback }: { fallback: string }) {
  const t = useT()
  const info = useDigestIssue()
  if (!info?.sendAt) return <>{fallback}</>
  return <>{t(dc.issueLabel, { date: `${shortDate(info.sendAt)}, ${clock(info.sendAt)}` })}</>
}

/** The issue's state ("Waiting for approval") in place of the save status. */
export function IssueHeaderStatus() {
  const info = useDigestIssue()
  if (!info) return null
  return <IssueStatusBadge status={info.status} />
}

/**
 * Cards above the canvas in digest mode: a prepared issue (its edits stay
 * in this issue; rebuild it or edit the template instead), and the
 * template as a member without campaigns:edit sees it.
 */
export function DigestCanvasCards({ onReview }: { onReview: () => void }) {
  const c = useComposer()
  if (isDigestIssue(c.campaign)) return <PreparedCard onReview={onReview} />
  if (isDigestTemplate(c.campaign) && !c.canEdit) return <ViewerCard />
  return null
}

function PreparedCard({ onReview }: { onReview: () => void }) {
  const t = useT()
  const c = useComposer()
  const { slug } = useWorkspace()
  const info = useDigestIssue()
  const digest = useDigest(c.campaign.digestId!).data
  const refresh = useRefreshDigest()
  const [busy, setBusy] = useState(false)
  if (!info || !info.preparedAt || !['prepared', 'waiting', 'held', 'draft'].includes(info.status)) return null
  const rebuild = async () => {
    setBusy(true)
    try {
      await http.POST('/api/v1/workspaces/{workspace}/digests/{digestId}/issues/{issueId}/rebuild', {
        params: { path: { workspace: slug, digestId: info.digestId, issueId: info.issueId } },
      })
      refresh()
      await c.reloadFromServer()
      toast({ kind: 'success', title: t(dc.rebuilt, { number: info.number }), action: { label: t(dc.checkThis), onClick: onReview } })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    } finally {
      setBusy(false)
    }
  }
  const changed = digest?.templateSavedAt && new Date(digest.templateSavedAt) > new Date(info.preparedAt) ? digest.templateSavedAt : null
  return (
    <div className="mx-auto flex w-full flex-col gap-2.5" style={{ maxWidth: 600, marginBottom: 20 }}>
      <section className="note i" role="note" aria-label={t(dc.preparedTitle)}>
        <span className="text-info-fg">
          <Icon name="info" size={16} />
        </span>
        <div className="flex min-w-0 flex-col gap-1">
          <span className="font-bold">{t(dc.preparedTitle)}</span>
          <span>{t(dc.preparedBody, { number: info.number, time: clock(info.preparedAt) })}</span>
          {c.canEdit ? (
            <span className="mt-1 flex flex-wrap gap-2">
              <Button size="sm" loading={busy} onClick={() => void rebuild()}>
                <Icon name="refresh" size={13} />
                {busy ? t(dc.rebuilding) : t(dc.rebuild)}
              </Button>
              {digest ? (
                <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: digest.templateId }} className="btn sm">
                  {t(dc.editTemplate)}
                </Link>
              ) : null}
            </span>
          ) : null}
        </div>
      </section>
      {changed && digest ? (
        <span className="k text-[12.5px]">{t(dc.templateChanged, { time: clock(changed), version: digest.templateVersion, prepared: clock(info.preparedAt) })}</span>
      ) : null}
    </div>
  )
}

function ViewerCard() {
  const t = useT()
  const c = useComposer()
  const me = useMe().data
  const { me: wsMe } = useWorkspace()
  const digest = useDigest(c.campaign.digestId!).data
  if (!me) return null
  const helpers = digest?.editHelpers ?? []
  return (
    <div className="mx-auto flex w-full flex-col gap-2.5" style={{ maxWidth: 600, marginBottom: 20 }}>
      <section className="note n" role="note" aria-label={t(dc.readOnly)}>
        <span className="text-ink-3">
          <Icon name="info" size={16} />
        </span>
        <div className="flex min-w-0 flex-col gap-1">
          <span className="font-bold">{t(dc.viewerTitle, { name: me.user.name, role: roleName({ key: wsMe.workspace.roleKey, name: wsMe.workspace.roleName }) })}</span>
          <span>{helpers.length ? t(dc.viewerBody, { names: formatList(helpers, 'disjunction') }) : t(dc.viewerBodyNoOne)}</span>
        </div>
      </section>
      <span className="k text-[12.5px]">{t(dc.viewerNote)}</span>
    </div>
  )
}
