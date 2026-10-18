import { useState } from 'react'
import { Button } from '@/components/ui/Button'
import { useT } from '@/i18n'
import { dayMonthDate } from '@/lib/format'
import { useCan, useMe } from '@/lib/session'
import { m } from './messages'
import { useOverview } from './shared'
import { tierName, VerifyDialog } from './VerifyDialog'

/** " · never checked", " · checked 14 Oct": when a tier's addresses were last verified. */
export function TierChecked({ tier }: { tier: string }) {
  const t = useT()
  const can = useCan('integrations:view')
  const q = useOverview(can)
  if (!can || !q.data || !q.data.services.some((s) => s.connected)) return null
  const name = tierName(t, tier)
  const last = q.data.uses.find((u) => u.kind === 'tier' && u.label === name && u.status === 'finished')
  return <>{last?.finishedAt ? ` · ${t(m.tierCheckedOn, { date: dayMonthDate(last.finishedAt) })}` : ` · ${t(m.tierNeverChecked)}`}</>
}

/** A tier card's Verify addresses. */
export function TierVerify({ tier }: { tier: string }) {
  const t = useT()
  const me = useMe().data
  const can = useCan('subscribers:verify')
  const [open, setOpen] = useState(false)
  if (!can) return null
  return (
    <>
      <Button size="sm" className="mt-2 self-start" onClick={() => setOpen(true)}>
        {t(m.verifyTitle)}
      </Button>
      {open ? (
        <VerifyDialog
          open
          onOpenChange={setOpen}
          target={{ kind: 'tier', tiers: [tier] }}
          origin="list_health"
          from={t(m.fromListHealthTier, { tier: t(m.tierPhrase, { tier: tierName(t, tier) }), name: me?.user.name ?? '' })}
        />
      ) : null}
    </>
  )
}

/** Verify addresses… for one list, opened from its row's menu. */
export function ListVerifyDialog({ list, onClose }: { list: { id: string; name: string }; onClose: () => void }) {
  const t = useT()
  const me = useMe().data
  return (
    <VerifyDialog
      open
      onOpenChange={(o) => !o && onClose()}
      target={{ kind: 'list', listId: list.id }}
      origin="list_health"
      from={t(m.fromListHealthList, { list: list.name, name: me?.user.name ?? '' })}
    />
  )
}

/** The row menu's label. */
export function useVerifyLabel(): string | undefined {
  const t = useT()
  return useCan('subscribers:verify') ? t(m.verifyEllipsis) : undefined
}

/** Send Readiness's "Verify N addresses first": the inactive tiers, skipping recent checks. */
export function ReadinessVerifyDialog({ campaignId, campaignName, onClose }: { campaignId: string; campaignName: string; onClose: () => void }) {
  const t = useT()
  const me = useMe().data
  return (
    <VerifyDialog
      open
      onOpenChange={(o) => !o && onClose()}
      target={{ kind: 'tier', tiers: ['at_risk', 'sunset'] }}
      origin="readiness"
      campaignId={campaignId}
      from={t(m.fromReadiness, { campaign: campaignName, name: me?.user.name ?? '' })}
      onStarted={onClose}
    />
  )
}
