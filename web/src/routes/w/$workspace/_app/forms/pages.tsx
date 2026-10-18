import { createFileRoute } from '@tanstack/react-router'
import { BuiltinPagesTab } from '@/features/website/BuiltinPages'

// Forms & pages › Pages (design expansion phase 7). The page builder's list
// of the workspace's own pages comes with M29; until then the tab lists the
// built-in pages and opens their editors.
export const Route = createFileRoute('/w/$workspace/_app/forms/pages')({
  component: BuiltinPagesTab,
})
