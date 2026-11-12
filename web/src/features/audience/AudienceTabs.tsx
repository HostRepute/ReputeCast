import { useQuery } from '@tanstack/react-query'
import { NavTabs } from '@/components/ui/Tabs'
import { m as scoring } from '@/features/scoring/messages'
import { useT } from '@/i18n'
import { count } from '@/lib/format'
import { m as replies } from '@/features/replies/messages'
import { useCan, useWorkspace } from '@/lib/session'
import { m as lf } from './listsfields.messages'
import { m } from './messages'
import { audienceSummaryQuery } from './queries'

/** The audience summary behind the tabs and the empty state. */
export function useAudienceSummary() {
  const { slug } = useWorkspace()
  return useQuery(audienceSummaryQuery(slug))
}

/**
 * Contacts · Segments · Lists & fields · List health · Suppression list ·
 * Replies (M58, with replies:view),
 * inline after the screen title. Counts appear once the summary has loaded
 * (and not at all while the workspace has no contacts, as on the empty-state
 * board).
 */
export function AudienceTabs() {
  const { slug } = useWorkspace()
  const t = useT()
  const summary = useAudienceSummary()
  const s = summary.data
  const show = s && s.contacts > 0
  // While the summary loads, the three tabs that show a count keep room for it.
  const reserve = (ch: number) => (summary.isPending ? ch : undefined)
  // Replies (M58), for people who can read them.
  const canReplies = useCan('replies:view')
  return (
    <NavTabs
      label={t(m.sections)}
      className="ml-3 max-md:ml-0"
      tabs={[
        { label: show ? t(m.tabContactsCount, { count: count(s.contacts) }) : t(m.tabContacts), to: '/w/$workspace/audience', params: { workspace: slug }, exact: true, reserve: reserve(6) },
        { label: show ? t(m.tabSegmentsCount, { count: count(s.segments) }) : t(m.tabSegments), to: '/w/$workspace/audience/segments', params: { workspace: slug }, reserve: reserve(2.5) },
        { label: t(lf.tab), to: '/w/$workspace/audience/lists', params: { workspace: slug } },
        { label: t(m.tabHealth), to: '/w/$workspace/audience/health', params: { workspace: slug } },
        // Scoring and Growth (X13).
        { label: t(scoring.tabScoring), to: '/w/$workspace/audience/scoring', params: { workspace: slug } },
        { label: t(scoring.tabGrowth), to: '/w/$workspace/audience/growth', params: { workspace: slug } },
        {
          label: show ? t(m.tabSuppressionsCount, { count: count(s.suppressions) }) : t(m.tabSuppressions),
          to: '/w/$workspace/audience/suppressions',
          params: { workspace: slug },
          reserve: reserve(5),
        },
        ...(canReplies ? [{ label: t(replies.tab), to: '/w/$workspace/audience/replies' as const, params: { workspace: slug } }] : []),
      ]}
    />
  )
}
