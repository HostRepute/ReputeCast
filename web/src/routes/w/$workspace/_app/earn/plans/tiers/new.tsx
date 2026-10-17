import { createFileRoute } from '@tanstack/react-router'
import { TierEditorPage } from '@/features/earn/TierEditor'

/** A new tier (EarnPlanEditors). */
export const Route = createFileRoute('/w/$workspace/_app/earn/plans/tiers/new')({
  component: function NewTierRoute() {
    return <TierEditorPage />
  },
})
