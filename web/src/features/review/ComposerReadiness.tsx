import { Link } from '@tanstack/react-router'
import { $api, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { mh } from '@/features/composer/chrome.messages'
import { useT } from '@/i18n'
import { clock } from '@/lib/format'
import { m } from './messages'

/*
 * What the composer's Checks tab shows from Send Readiness (Composer board):
 * the workspace's sending rules and the sender's reputation, from the
 * campaign's latest run (it isn't run again here). Each is a summary row,
 * and each warning or block a card that opens Review & send.
 */

type Findings = Schemas['ComposerReadiness']
type Group = Schemas['ComposerReadinessGroup']

const findingsPath = '/api/v1/workspaces/{workspace}/campaigns/{campaignId}/readiness/composer' as const

function useFindings(slug: string, campaignId: string) {
  return $api.useQuery('get', findingsPath, { params: { path: { workspace: slug, campaignId } } }, { retry: false, staleTime: 60_000 })
}

/** The Workspace rules and Sender reputation rows of the Checks summary. */
export function ReadinessRows({ slug, campaignId }: { slug: string; campaignId: string }) {
  const t = useT()
  const d = useFindings(slug, campaignId).data
  if (!d?.ranAt) return null
  const from = d.checkedAt ? t(m.composerFromReviewAt, { time: clock(d.checkedAt) }) : t(m.composerFromReview)
  return (
    <>
      {d.rules ? <Row title={t(m.composerRules)} group={d.rules} caption={d.rules.checks.map((c) => c.label || c.title).join(' · ')} /> : null}
      {d.reputation ? <Row title={t(m.composerReputation)} group={d.reputation} caption={from} /> : null}
    </>
  )
}

function Row({ title, group, caption }: { title: string; group: Group; caption: string }) {
  const t = useT()
  const issues = group.blocks + group.warnings
  return (
    <div className="cksr">
      <span className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5">
        <span className="font-semibold">{title}</span>
        {issues ? <span className="k">{caption}</span> : null}
      </span>
      {issues ? (
        <Badge tone={group.blocks ? 'bad' : 'warn'} dot>
          {t(mh.summaryIssues, { blocks: group.blocks, warnings: group.warnings, both: group.blocks && group.warnings ? 'yes' : 'no' })}
        </Badge>
      ) : (
        <Badge tone="ok" dot>
          {t(mh.pass)}
        </Badge>
      )}
    </div>
  )
}

/** The rules' and reputation's warnings and blocks, each opening Review & send. */
export function ReadinessIssues({ slug, campaignId }: { slug: string; campaignId: string }) {
  const t = useT()
  const d: Findings | undefined = useFindings(slug, campaignId).data
  if (!d?.ranAt) return null
  const cards = [
    ...(d.rules?.checks ?? []).map((c) => ({ c, area: t(m.composerRules) })),
    ...(d.reputation?.checks ?? []).map((c) => ({ c, area: t(m.composerReputation) })),
  ]
  return cards.map(({ c, area }) => (
    <div key={c.key} className={`iss ${c.status === 'block' ? 'is-block' : 'is-warn'}`}>
      <span className="iss-area">{area}</span>
      <div className="t">
        <Badge tone={c.status === 'block' ? 'bad' : 'warn'}>{c.status === 'block' ? t(m.statusBlock) : t(m.statusWarn)}</Badge>
        <span>{c.title}</span>
      </div>
      {c.detail ? <span className="k">{c.detail}</span> : null}
      <div className="flex flex-wrap gap-1.5">
        <Link to="/w/$workspace/campaigns/$campaignId/review" params={{ workspace: slug, campaignId }} className="btn sm">
          {t(m.openInReview)}
        </Link>
      </div>
    </div>
  ))
}
