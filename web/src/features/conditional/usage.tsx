import { useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, http, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { formatList, useT } from '@/i18n'
import { shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { m } from './messages'

type Usage = Schemas['VisibilityUsage']

/** Conditional blocks whose rules use a segment, list, field or tag (subscribers:view). */
export function useVisibilityUsage(kind: 'segment' | 'list' | 'field' | 'tag', ref: string, enabled = true) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', '/api/v1/workspaces/{workspace}/visibility/usage', { params: { path: { workspace: slug }, query: { kind, ref } } }, { enabled: enabled && !!ref, retry: 0 })
}

function quoted(names: string[]) {
  return formatList(names.map((n) => `“${n}”`))
}

/** "Used by 1 conditional block in Weekly Digest #143: “Annual billing offer”." */
function usedBy(t: ReturnType<typeof useT>, u: Usage): string {
  const e = u.emails[0]
  if (u.emails.length === 1 && e) return t(m.usedByIn, { count: u.blocks, email: e.name, blocks: quoted(e.blocks) })
  return t(m.usedByEmails, { count: u.blocks, emails: u.emails.length })
}

/** The segment builder's "Where this segment is used", when a Visibility rule uses it. */
export function SegmentUsageCard({ segmentId }: { segmentId: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const usage = useVisibilityUsage('segment', segmentId, segmentId !== 'new')
  const u = usage.data
  if (!u || !u.blocks) return null
  return (
    <section className="card flex flex-col gap-2 px-4.5 py-4" aria-labelledby="seg-used-h">
      <h2 className="h2" id="seg-used-h">
        {t(m.whereUsed)}
      </h2>
      <p className="m-0">
        {usedBy(t, u)} {t(m.changingChanges, { count: u.blocks })}
      </p>
      <ul className="m-0 flex list-none flex-col gap-1 p-0">
        {u.emails.map((e) => (
          <li key={e.campaignId} className="k">
            <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: e.campaignId }}>
              {e.name}
            </Link>
            {' · '}
            {e.blocks.join(', ')}
          </li>
        ))}
      </ul>
    </section>
  )
}

/** "Delete segment", which warns first when a Visibility rule uses it. */
export function DeleteSegmentButton({ segmentId, name }: { segmentId: string; name: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const reason = usePermissionReason('segments:edit')
  const [open, setOpen] = useState(false)
  const usage = useVisibilityUsage('segment', segmentId, open)
  const u = usage.data
  const e = u?.emails[0]
  if (segmentId === 'new') return null
  return (
    <>
      <Button variant="danger" disabledReason={reason} onClick={() => setOpen(true)}>
        {t(m.deleteSegment)}
      </Button>
      <ConfirmDialog
        open={open}
        onOpenChange={setOpen}
        badge={t(m.deleteSegment)}
        title={t(m.deleteSegmentTitle, { name })}
        description={
          u?.blocks && e
            ? u.emails.length === 1
              ? e.scheduledAt
                ? t(m.deleteUsedScheduled, { count: u.blocks, blocks: quoted(e.blocks), email: e.name, date: shortDate(e.scheduledAt) })
                : t(m.deleteUsed, { count: u.blocks, blocks: quoted(e.blocks), email: e.name })
              : t(m.deleteUsedMany, { count: u.blocks, emails: u.emails.length })
            : t(m.deleteUnused)
        }
        secondary={
          u?.blocks && e ? (
            <Button
              onClick={() => {
                setOpen(false)
                void navigate({ to: '/w/$workspace/campaigns/$campaignId/edit', params: { workspace: slug, campaignId: e.campaignId } })
              }}
            >
              {t(m.openEmail, { name: e.name })}
            </Button>
          ) : undefined
        }
        confirmLabel={t(m.deleteSegment)}
        onConfirm={async () => {
          await http.DELETE('/api/v1/workspaces/{workspace}/segments/{segment}', { params: { path: { workspace: slug, segment: segmentId } } })
          await qc.invalidateQueries({ predicate: (q) => String((q.queryKey as unknown[])[1] ?? '').includes('/segments') })
          toast({ kind: 'success', title: t(m.segmentDeleted, { name }) })
          void navigate({ to: '/w/$workspace/audience/segments', params: { workspace: slug } })
        }}
      />
    </>
  )
}
