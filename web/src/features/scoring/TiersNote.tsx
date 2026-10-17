import { Link } from '@tanstack/react-router'
import { Icon } from '@/components/icons/Icon'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/**
 * "The score and engagement tiers do different jobs", on Audience ›
 * Scoring (linking to the tiers) and on List health (linking to scores).
 */
export function TiersNote({ on }: { on: 'scoring' | 'health' }) {
  const t = useT()
  const { slug } = useWorkspace()
  return (
    <div className="note i" role="note">
      <span className="sc-note-ic" style={{ color: '#1D4ED8' }}>
        <Icon name="info" size={16} strokeWidth={1.8} />
      </span>
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <span className="font-bold">{t(m.tiersTitle)}</span>
        <div>
          {t(m.tiersBody)}{' '}
          {on === 'scoring' ? (
            <Link to="/w/$workspace/audience/health" params={{ workspace: slug }}>
              {t(m.tiersLink)}
            </Link>
          ) : (
            <Link to="/w/$workspace/audience/scoring" params={{ workspace: slug }}>
              {t(m.tiersLinkScoring)}
            </Link>
          )}
        </div>
      </div>
    </div>
  )
}
