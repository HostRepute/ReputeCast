import { useQueryClient } from '@tanstack/react-query'
import { forwardRef, useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Field, Select } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { count, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'

type Share = Schemas['ReportShareLink']

export function useShares(campaignId: string, enabled: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares',
    { params: { path: { workspace: slug, campaignId } } },
    { enabled },
  )
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

/** "Share read-only link": choose an expiry and create the link (it's copied). */
export function ShareDialog({ open, onOpenChange, campaignId, onCreated }: { open: boolean; onOpenChange: (o: boolean) => void; campaignId: string; onCreated?: (s: Share) => void }) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [days, setDays] = useState('0')
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares')
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="info"
      badge="Read-only link"
      title="Share this report"
      description="Anyone with the link sees the same charts and tables, without signing in. It never shows contact data, and you can revoke it any time."
      confirmLabel="Create link"
      busyLabel="Creating…"
      onConfirm={async () => {
        const s = await create.mutateAsync({ params: { path: { workspace: slug, campaignId } }, body: { expiresInDays: Number(days) } })
        await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares'] })
        const copied = s.url ? await copyText(s.url) : false
        toast({ kind: 'success', title: copied ? 'Link created and copied' : 'Link created', detail: s.expiresAt ? `It stops working on ${shortDate(s.expiresAt)}.` : 'It works until you revoke it.' })
        onCreated?.(s)
      }}
    >
      <Field label="Link expires" hint="An expiring link stops working by itself.">
        <Select value={days} onChange={(e) => setDays(e.target.value)}>
          <option value="0">Never (until revoked)</option>
          <option value="7">In 7 days</option>
          <option value="30">In 30 days</option>
          <option value="90">In 90 days</option>
        </Select>
      </Field>
    </ConfirmDialog>
  )
}

/** "Share this report": the active link with Copy and Revoke, or a way to make one. */
export const ShareCard = forwardRef<HTMLElement, { campaignId: string; onCreate: () => void }>(function ShareCard({ campaignId, onCreate }, ref) {
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('reports:share')
  const shares = useShares(campaignId, !reason)
  const revoke = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares/{shareId}')
  const [revoking, setRevoking] = useState<Share>()
  const link = shares.data?.items[0]
  const more = (shares.data?.items.length ?? 0) - 1
  const expires = link?.expiresAt ? ` It stops working on ${shortDate(link.expiresAt)}.` : ' Revoke it any time.'
  return (
    <section ref={ref} tabIndex={-1} className="rep-share card flex flex-wrap items-center gap-4 px-5 py-4" aria-labelledby="rep-share-h">
      <div className="flex flex-[1_1_300px] flex-col gap-0.5">
        <span className="font-semibold" id="rep-share-h">
          Share this report
        </span>
        <span className="k">
          A read-only link with the same charts and tables, no login needed.{link ? expires : ' Revoke it any time.'}
          {more > 0 ? ` ${count(more)} older ${more === 1 ? 'link is' : 'links are'} active too.` : ''}
        </span>
      </div>
      {link ? (
        <>
          <input
            className="mono h-9 flex-[1_1_300px] rounded-lg border border-line-strong bg-ground-3 px-2.5 text-[13px] max-md:min-w-0"
            value={link.url}
            readOnly
            aria-label="Read-only link"
            onFocus={(e) => e.currentTarget.select()}
          />
          <div className="flex gap-2">
            <Button
              onClick={async () => {
                const ok = await copyText(link.url)
                toast(ok ? { kind: 'success', title: 'Link copied' } : { kind: 'error', title: "Couldn't copy the link", detail: 'Select it in the box and copy it yourself.' })
              }}
            >
              Copy link
            </Button>
            <Button onClick={() => setRevoking(link)}>Revoke</Button>
          </div>
        </>
      ) : (
        <Button disabledReason={reason} loading={shares.isLoading} onClick={onCreate}>
          Create read-only link
        </Button>
      )}
      {revoking ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setRevoking(undefined)}
          tone="bad"
          badge="Revoke link"
          title="Revoke this read-only link?"
          description={`Anyone who opens it sees that the link doesn't work, from now on. It was opened ${count(revoking.views)} ${revoking.views === 1 ? 'time' : 'times'}. You can create a new link later.`}
          confirmLabel="Revoke link"
          busyLabel="Revoking…"
          onConfirm={async () => {
            await revoke.mutateAsync({ params: { path: { workspace: slug, campaignId, shareId: revoking.id } } })
            await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares'] })
            toast({ kind: 'deleted', title: 'Link revoked', detail: 'It no longer opens the report.' })
          }}
        />
      ) : null}
    </section>
  )
})
