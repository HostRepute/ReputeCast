import { useQueryClient } from '@tanstack/react-query'
import { forwardRef, useState } from 'react'
import { $api, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { Field, Select } from '@/components/ui/Field'
import { toast } from '@/components/ui/Toast'
import { count, shortDate } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { t as tt, useT } from '@/i18n'
import { m } from './messages'

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
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const [days, setDays] = useState('0')
  const create = $api.useMutation('post', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares')
  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      tone="info"
      badge={t(m.shareBadge)}
      title={t(m.shareTitle)}
      description={t(m.shareBody)}
      confirmLabel={t(m.createLink)}
      busyLabel={t(m.creating)}
      onConfirm={async () => {
        const s = await create.mutateAsync({ params: { path: { workspace: slug, campaignId } }, body: { expiresInDays: Number(days) } })
        await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares'] })
        const copied = s.url ? await copyText(s.url) : false
        toast({
          kind: 'success',
          title: copied ? tt(m.linkCreatedCopied) : tt(m.linkCreated),
          detail: s.expiresAt ? tt(m.stopsOn, { date: shortDate(s.expiresAt) }) : tt(m.worksUntilRevoked),
        })
        onCreated?.(s)
      }}
    >
      <Field label={t(m.linkExpires)} hint={t(m.linkExpiresHint)}>
        <Select value={days} onChange={(e) => setDays(e.target.value)}>
          <option value="0">{t(m.expiresNever)}</option>
          <option value="7">{t(m.expiresIn, { days: 7 })}</option>
          <option value="30">{t(m.expiresIn, { days: 30 })}</option>
          <option value="90">{t(m.expiresIn, { days: 90 })}</option>
        </Select>
      </Field>
    </ConfirmDialog>
  )
}

/** "Share this report": the active link with Copy and Revoke, or a way to make one. */
export const ShareCard = forwardRef<HTMLElement, { campaignId: string; onCreate: () => void }>(function ShareCard({ campaignId, onCreate }, ref) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('reports:share')
  const shares = useShares(campaignId, !reason)
  const revoke = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares/{shareId}')
  const [revoking, setRevoking] = useState<Share>()
  const link = shares.data?.items[0]
  const more = (shares.data?.items.length ?? 0) - 1
  const lead = [t(m.shareLead), link?.expiresAt ? t(m.stopsOn, { date: shortDate(link.expiresAt) }) : t(m.revokeAnyTime)]
  if (more > 0) lead.push(t(m.olderLinks, { n: more, count: count(more) }))
  return (
    <section ref={ref} tabIndex={-1} className="rep-share card flex flex-wrap items-center gap-4 px-5 py-4" aria-labelledby="rep-share-h">
      <div className="flex flex-[1_1_300px] flex-col gap-0.5">
        <span className="font-semibold" id="rep-share-h">
          {t(m.shareTitle)}
        </span>
        <span className="k">{lead.join(' ')}</span>
      </div>
      {link ? (
        <>
          <input
            className="mono h-9 flex-[1_1_300px] rounded-lg border border-line-strong bg-ground-3 px-2.5 text-[13px] max-md:min-w-0"
            value={link.url}
            readOnly
            aria-label={t(m.shareBadge)}
            onFocus={(e) => e.currentTarget.select()}
          />
          <div className="flex gap-2">
            <Button
              onClick={async () => {
                const ok = await copyText(link.url)
                toast(ok ? { kind: 'success', title: tt(m.linkCopied) } : { kind: 'error', title: tt(m.copyFailed), detail: tt(m.copyYourself) })
              }}
            >
              {t(m.copyLink)}
            </Button>
            <Button onClick={() => setRevoking(link)}>{t(m.revoke)}</Button>
          </div>
        </>
      ) : (
        <Button disabledReason={reason} loading={shares.isLoading} onClick={onCreate}>
          {t(m.createReadOnly)}
        </Button>
      )}
      {revoking ? (
        <ConfirmDialog
          open
          onOpenChange={(o) => !o && setRevoking(undefined)}
          tone="bad"
          badge={t(m.revokeBadge)}
          title={t(m.revokeTitle)}
          description={t(m.revokeBody, { n: revoking.views, count: count(revoking.views) })}
          confirmLabel={t(m.revokeBadge)}
          busyLabel={t(m.revoking)}
          onConfirm={async () => {
            await revoke.mutateAsync({ params: { path: { workspace: slug, campaignId, shareId: revoking.id } } })
            await qc.invalidateQueries({ queryKey: ['get', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares'] })
            toast({ kind: 'deleted', title: tt(m.linkRevoked), detail: tt(m.linkRevokedBody) })
          }}
        />
      ) : null}
    </section>
  )
})
