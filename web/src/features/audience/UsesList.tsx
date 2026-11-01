import { Link } from '@tanstack/react-router'
import type { Schemas } from '@/api/client'
import { t, useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './listsfields.messages'

type Use = Schemas['Use']

/** Where each kind of use opens. */
function UseLink({ u }: { u: Use }) {
  const tr = useT()
  const { slug } = useWorkspace()
  switch (u.kind) {
    case 'segment':
      return (
        <Link to="/w/$workspace/audience/segments/$segmentId" params={{ workspace: slug, segmentId: u.id }}>
          {tr(m.open)}
        </Link>
      )
    case 'automation':
      return (
        <Link to="/w/$workspace/automations/$automationId" params={{ workspace: slug, automationId: u.id }}>
          {tr(m.open)}
        </Link>
      )
    case 'campaign':
      return (
        <Link to="/w/$workspace/campaigns/$campaignId/edit" params={{ workspace: slug, campaignId: u.id }}>
          {tr(m.open)}
        </Link>
      )
    case 'digest':
      return (
        <Link to="/w/$workspace/campaigns/digests/$digestId" params={{ workspace: slug, digestId: u.id }}>
          {tr(m.open)}
        </Link>
      )
    case 'scoring':
      return (
        <Link to="/w/$workspace/audience/scoring" params={{ workspace: slug }}>
          {tr(m.open)}
        </Link>
      )
    case 'form':
      return (
        <Link to="/w/$workspace/forms" params={{ workspace: slug }}>
          {tr(m.open)}
        </Link>
      )
    // A saved view's own address (review 2026-11 AU-19).
    case 'view':
      return (
        <Link to="/w/$workspace/audience" params={{ workspace: slug }} search={{ view: u.id }}>
          {tr(m.open)}
        </Link>
      )
    case 'transactional':
      return (
        <Link to="/w/$workspace/transactional/templates/$templateId" params={{ workspace: slug, templateId: u.id }}>
          {tr(m.open)}
        </Link>
      )
  }
  return null
}

const kindLabel: Record<Use['kind'], () => string> = {
  segment: () => t(m.kindSegment),
  form: () => t(m.kindForm),
  scoring: () => t(m.kindScoring),
  automation: () => t(m.kindAutomation),
  campaign: () => t(m.kindCampaign),
  digest: () => t(m.kindDigest),
  view: () => t(m.kindView),
  transactional: () => t(m.kindTransactional),
}

/** What a field or segment is used by, before it's deleted (AU-11). */
export function UsesList({ uses }: { uses: Use[] }) {
  return (
    <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
      {uses.map((u) => (
        <li key={`${u.kind}-${u.id}`} className="flex items-center gap-2.5 rounded-lg border border-line-2 px-2.5 py-2 text-[13px]">
          <span className="min-w-0 flex-1">
            <b>{kindLabel[u.kind]()}</b>
            {u.name ? ` · ${u.name}` : ''}
          </span>
          <UseLink u={u} />
        </li>
      ))}
    </ul>
  )
}
