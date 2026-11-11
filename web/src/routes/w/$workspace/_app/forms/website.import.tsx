import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ImportPage } from '@/features/website/import/ImportPage'
import { WithWebsite } from '@/features/website/WebsiteFrame'

const search = z.object({
  /** The import being reviewed, run or reported on. */
  import: z.uuid().optional().catch(undefined),
})

/** Forms & pages › Website › Posts › Import posts (WebsiteImport, WebsiteImportStates). */
export const Route = createFileRoute('/w/$workspace/_app/forms/website/import')({
  validateSearch: search,
  component: function ImportRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return <WithWebsite>{(ov) => <ImportPage overview={ov} importId={s.import} onImport={(id) => void navigate({ search: { import: id } })} />}</WithWebsite>
  },
})
