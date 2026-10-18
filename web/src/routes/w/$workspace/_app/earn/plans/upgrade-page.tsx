import { createFileRoute } from '@tanstack/react-router'
import { UpgradePageEditorPage } from '@/features/earn/UpgradePageEditor'

/** The upgrade page editor (EarnUpgradeEditor). */
export const Route = createFileRoute('/w/$workspace/_app/earn/plans/upgrade-page')({
  component: UpgradePageEditorPage,
})
