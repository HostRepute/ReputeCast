import { createFileRoute } from '@tanstack/react-router'
import { TierEditorPage } from '@/features/earn/TierEditor'

/** A tier's editor (EarnPlanEditors). */
export const Route = createFileRoute('/w/$workspace/_app/earn/plans/tiers/$tierId')({
  component: function TierRoute() {
    const { tierId } = Route.useParams()
    return <TierEditorPage key={tierId} tierId={tierId} />
  },
})
