import { Badge } from '@/components/ui/Badge'
import { useT } from '@/i18n'
import { mc } from './composer.messages'

/** "Paid only": a campaign for paid members only (the campaigns list, the composer's header). */
export function PaidOnlyBadge({ className }: { className?: string }) {
  const t = useT()
  return (
    <Badge tone="info" className={className}>
      {t(mc.paidOnly)}
    </Badge>
  )
}

/** The Blocks panel's line under the Earn group (ComposerEarnBlocks). */
export function EarnBlocksHint() {
  const t = useT()
  return <p className="k blk-hint">{t(mc.blocksHint)}</p>
}
