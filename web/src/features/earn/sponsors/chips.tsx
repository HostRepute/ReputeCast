import { $api } from '@/api/client'
import { defineMessages, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'

const mc = defineMessages('earn.sponsorChip', {
  chip: 'Sponsored · {sponsor}',
})

/** The sponsor of each campaign with a booking, by campaign id (the Campaigns list's chips). */
export function useSponsorChips(): Map<string, string> {
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/sponsors/campaign-chips', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  return new Map((q.data?.items ?? []).map((c) => [c.campaignId, c.sponsor]))
}

/** "Sponsored · Northwind Analytics" (EarnInReport). */
export function SponsorChip({ sponsor }: { sponsor: string }) {
  const t = useT()
  const text = t(mc.chip, { sponsor })
  // One line: a long sponsor name ends in an ellipsis, whole on hover.
  return (
    <span className="chip max-w-full" style={{ height: 22, fontSize: 12, fontWeight: 600, color: 'var(--ink-nav)' }} title={text}>
      <span className="truncate">{text}</span>
    </span>
  )
}
