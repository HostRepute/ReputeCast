import { useQueryClient } from '@tanstack/react-query'
import { forwardRef, useImperativeHandle, useRef, useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Button } from '@/components/ui/Button'
import { ConfirmDialog } from '@/components/ui/ConfirmDialog'
import { toast } from '@/components/ui/Toast'
import { andList, count, dayMonthDate, money } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { t as tt, useT } from '@/i18n'
import { m } from './messages'
import type { CampaignReport } from './words'

type Share = Schemas['ReportShareLink']

const sharesPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares' as const

export function useShares(campaignId: string, enabled: boolean) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', sharesPath, { params: { path: { workspace: slug, campaignId } } }, { enabled })
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}

/** The link as the board shows it: "reputecast.acme.io/r/8f2c1a9d". */
function shortURL(url: string): string {
  return url.replace(/^https?:\/\//, '')
}

/** "Created by Linh Nguyen on 21 Oct · expires 20 Nov · with the email". */
export function linkMeta(s: Share): string {
  const meta = s.expiresAt
    ? tt(m.linkMeta, { name: s.createdByName, date: dayMonthDate(s.createdAt), expires: dayMonthDate(s.expiresAt) })
    : tt(m.linkMetaNoExpiry, { name: s.createdByName, date: dayMonthDate(s.createdAt) })
  const parts = [s.includes.email ? tt(m.partEmail) : '', s.includes.cost ? tt(m.partCost) : '', s.includes.warmup ? tt(m.partWarmup) : ''].filter(Boolean)
  return parts.length ? tt(m.linkWith, { meta, parts: andList(parts) }) : meta
}

/** What the share card's focus moves to from the header's "Share read-only link". */
export interface ShareCardHandle {
  focus(): void
}

/**
 * "Share this report" (review AN-16): choose what a new read-only link
 * includes beyond the numbers and when it expires, then the active links,
 * each copied or revoked on its own.
 */
export const ShareCard = forwardRef<ShareCardHandle, { r: CampaignReport; campaignId: string }>(function ShareCard({ r, campaignId }, ref) {
  const t = useT()
  const { slug } = useWorkspace()
  const qc = useQueryClient()
  const reason = usePermissionReason('reports:share')
  const shares = useShares(campaignId, !reason)
  const create = $api.useMutation('post', sharesPath)
  const revoke = $api.useMutation('delete', '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/report/shares/{shareId}')
  const [cost, setCost] = useState(false)
  const [warmup, setWarmup] = useState(false)
  const [email, setEmail] = useState(true)
  const [days, setDays] = useState('30')
  const [revoking, setRevoking] = useState<Share>()
  const section = useRef<HTMLElement>(null)
  const first = useRef<HTMLInputElement>(null)
  useImperativeHandle(ref, () => ({
    focus() {
      section.current?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      first.current?.focus({ preventScroll: true })
    },
  }))
  const cols = r.providers.columns.filter((c) => c.sent > 0)
  const costs = cols.filter((c) => c.cost !== undefined).map((c) => t(m.costOn, { cost: money(c.cost), provider: c.name }))
  const warming = cols.filter((c) => c.warmupDay).map((c) => c.short || c.name)
  const links = shares.data?.items ?? []
  const make = async () => {
    try {
      const s = await create.mutateAsync({
        params: { path: { workspace: slug, campaignId } },
        body: { expiresInDays: Number(days), includeCost: cost, includeWarmup: warmup, includeEmail: email },
      })
      await qc.invalidateQueries({ queryKey: ['get', sharesPath] })
      const copied = s.url ? await copyText(s.url) : false
      toast({
        kind: 'success',
        title: copied ? tt(m.linkCreatedCopied) : tt(m.linkCreated),
        detail: s.expiresAt ? tt(m.stopsOn, { date: dayMonthDate(s.expiresAt) }) : tt(m.worksUntilRevoked),
      })
    } catch (e) {
      toast({ kind: 'error', title: errorMessage(e) })
    }
  }
  return (
    <section ref={section} tabIndex={-1} id="rep-share" className="rep-share card flex flex-col" aria-labelledby="rep-share-h">
      <div className="sec">
        <h2 className="h2" id="rep-share-h">
          {t(m.shareTitle)}
        </h2>
        <span className="k">{t(m.shareCardNote)}</span>
      </div>
      <div className="rep-panes">
        <div className="rep-pane gap-3">
          <h3 className="m-0 text-[13px] font-semibold">{t(m.newLink)}</h3>
          <fieldset className="m-0 flex min-w-0 flex-col border-0 p-0" disabled={Boolean(reason)}>
            <legend className="k mb-1 p-0">{t(m.alwaysIn)}</legend>
            <label className="rep-opt">
              <input ref={first} type="checkbox" checked={cost} onChange={(e) => setCost(e.target.checked)} />
              <span>
                {t(m.includeCost)}
                {costs.length ? <span className="k">{andList(costs)}</span> : null}
              </span>
            </label>
            <label className="rep-opt">
              <input type="checkbox" checked={warmup} onChange={(e) => setWarmup(e.target.checked)} />
              <span>
                {t(m.includeWarmup)}
                <span className="k">{warming.length ? t(m.warmupOf, { providers: andList(warming) }) : t(m.warmupEach)}</span>
              </span>
            </label>
            <label className="rep-opt">
              <input type="checkbox" checked={email} onChange={(e) => setEmail(e.target.checked)} />
              <span>
                {t(m.includeEmail)}
                <span className="k">{t(m.includeEmailSub)}</span>
              </span>
            </label>
          </fieldset>
          <label className="lbl max-w-[220px]">
            {t(m.expires)}
            <select className="in" value={days} disabled={Boolean(reason)} onChange={(e) => setDays(e.target.value)}>
              {[7, 30, 90].map((d) => (
                <option key={d} value={String(d)}>
                  {t(m.expiresDays, { n: d })}
                </option>
              ))}
            </select>
          </label>
          <div>
            <Button disabledReason={reason} loading={create.isPending} onClick={() => void make()}>
              {t(m.createLink)}
            </Button>
          </div>
        </div>
        <div className="rep-pane flex-[1_1_340px]">
          <h3 className="m-0 mb-1 text-[13px] font-semibold">{t(m.activeLinks, { count: count(links.length) })}</h3>
          {shares.isError ? (
            <span className="k">
              {t(m.sharesFailed)}: {errorMessage(shares.error)}
            </span>
          ) : links.length ? (
            links.map((s) => (
              <div key={s.id} className="rep-lrow">
                <div className="flex min-w-0 flex-[1_1_220px] flex-col gap-0.5">
                  <span className="mono text-[13px] [overflow-wrap:anywhere]">{shortURL(s.url)}</span>
                  <span className="k">{linkMeta(s)}</span>
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    aria-label={t(m.copyLinkBy, { name: s.createdByName })}
                    onClick={async () => {
                      const ok = await copyText(s.url)
                      toast(ok ? { kind: 'success', title: tt(m.linkCopied) } : { kind: 'error', title: tt(m.copyFailed), detail: s.url })
                    }}
                  >
                    {t(m.copyLink)}
                  </Button>
                  <Button size="sm" aria-label={t(m.revokeLinkBy, { name: s.createdByName })} onClick={() => setRevoking(s)}>
                    {t(m.revoke)}
                  </Button>
                </div>
              </div>
            ))
          ) : (
            <span className="k">{reason ?? t(m.noActiveLinks)}</span>
          )}
        </div>
      </div>
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
            await qc.invalidateQueries({ queryKey: ['get', sharesPath] })
            toast({ kind: 'deleted', title: tt(m.linkRevoked), detail: tt(m.linkRevokedBody) })
          }}
        />
      ) : null}
    </section>
  )
})
