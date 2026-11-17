import { useRouterState } from '@tanstack/react-router'
import { $api } from '@/api/client'
import type { MessageDescriptor } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/**
 * What the assistant knows of the screen it opened on (assistant brief
 * §2, "Attached context"): the screen's key (internal/ai/assistant's
 * screens.go), and on a campaign's report or review, that campaign as the
 * suggested context. Nothing is shared until the person attaches it.
 */
export interface ScreenContext {
  /** A key the server knows, or undefined. */
  screen?: string
  /** The suggestions' set. */
  kind: 'report' | 'campaign' | 'campaigns' | 'overview' | 'audience' | 'automations' | 'deliverability' | 'settings' | 'other'
  resource?: { kind: 'report' | 'campaign'; id: string }
}

const routes: [RegExp, (m: RegExpMatchArray) => ScreenContext][] = [
  [/^\/w\/[^/]+\/campaigns\/([^/]+)\/(?:sms-|push-|whatsapp-)?report$/, (x) => ({ screen: 'campaign.report', kind: 'report', resource: { kind: 'report', id: x[1]! } })],
  [/^\/w\/[^/]+\/campaigns\/([^/]+)\/(?:sms-|push-|whatsapp-)?review$/, (x) => ({ screen: 'campaign.review', kind: 'campaign', resource: { kind: 'campaign', id: x[1]! } })],
  [/^\/w\/[^/]+\/campaigns\/([^/]+)\/live$/, () => ({ screen: 'campaign.live', kind: 'campaigns' })],
  [/^\/w\/[^/]+\/campaigns\/?$/, () => ({ screen: 'campaigns', kind: 'campaigns' })],
  [/^\/w\/[^/]+\/approvals\/?/, () => ({ screen: 'approvals', kind: 'campaigns' })],
  [/^\/w\/[^/]+\/audience\/segments\/[^/]+$/, () => ({ screen: 'audience.segment', kind: 'audience' })],
  [/^\/w\/[^/]+\/audience\/segments\/?$/, () => ({ screen: 'audience.segments', kind: 'audience' })],
  [/^\/w\/[^/]+\/audience\/suppressions/, () => ({ screen: 'audience.suppressions', kind: 'audience' })],
  [/^\/w\/[^/]+\/audience/, () => ({ screen: 'audience', kind: 'audience' })],
  [/^\/w\/[^/]+\/automations\/[^/]+$/, () => ({ screen: 'automation', kind: 'automations' })],
  [/^\/w\/[^/]+\/automations\/?$/, () => ({ screen: 'automations', kind: 'automations' })],
  [/^\/w\/[^/]+\/deliverability/, () => ({ screen: 'deliverability', kind: 'deliverability' })],
  [/^\/w\/[^/]+\/providers/, () => ({ screen: 'providers', kind: 'deliverability' })],
  [/^\/w\/[^/]+\/settings\/senders/, () => ({ screen: 'settings.senders', kind: 'deliverability' })],
  [/^\/w\/[^/]+\/settings\/integrations\/ai/, () => ({ screen: 'settings.ai', kind: 'settings' })],
  [/^\/w\/[^/]+\/settings\/team/, () => ({ screen: 'settings.team', kind: 'settings' })],
  [/^\/w\/[^/]+\/settings/, () => ({ kind: 'settings' })],
  [/^\/w\/[^/]+\/agent\/new$/, () => ({ screen: 'agent.new', kind: 'other' })],
  [/^\/w\/[^/]+\/?$/, () => ({ screen: 'overview', kind: 'overview' })],
]

export function screenOf(pathname: string): ScreenContext {
  for (const [re, fn] of routes) {
    const x = pathname.match(re)
    if (x) return fn(x)
  }
  return { kind: 'other' }
}

export function useScreenContext(): ScreenContext {
  const pathname = useRouterState({ select: (s) => s.location.pathname })
  return screenOf(pathname)
}

/** The suggested resource's name (and whether a campaign is still a draft). */
export function useResourceTitle(resource: ScreenContext['resource']) {
  const { slug } = useWorkspace()
  const q = $api.useQuery(
    'get',
    '/api/v1/workspaces/{workspace}/campaigns/{campaignId}',
    { params: { path: { workspace: slug, campaignId: resource?.id ?? '' } } },
    { enabled: Boolean(resource), staleTime: 60_000, retry: false },
  )
  if (!resource || !q.data) return undefined
  // A campaign is attached as a draft; one that isn't any more isn't suggested.
  if (resource.kind === 'campaign' && q.data.status !== 'draft') return undefined
  return q.data.name
}

/** The three fixed suggestions of an empty thread, by screen (brief §3.3). */
export const suggestions: Record<ScreenContext['kind'], [MessageDescriptor, MessageDescriptor, MessageDescriptor]> = {
  report: [m.suggestReport1, m.suggestReport2, m.suggestReport3],
  campaign: [m.suggestCampaign1, m.suggestCampaign2, m.suggestCampaign3],
  campaigns: [m.suggestCampaigns1, m.suggestCampaigns2, m.suggestCampaigns3],
  overview: [m.suggestOverview1, m.suggestOverview2, m.suggestOverview3],
  audience: [m.suggestAudience1, m.suggestAudience2, m.suggestAudience3],
  automations: [m.suggestAutomations1, m.suggestAutomations2, m.suggestAutomations3],
  deliverability: [m.suggestDeliverability1, m.suggestDeliverability2, m.suggestDeliverability3],
  settings: [m.suggestSettings1, m.suggestSettings2, m.suggestSettings3],
  other: [m.suggestDefault1, m.suggestDefault2, m.suggestDefault3],
}

export function placeholderOf(kind: ScreenContext['kind']): MessageDescriptor {
  return kind === 'report' ? m.placeholderReport : kind === 'campaign' ? m.placeholderCampaign : m.placeholder
}
