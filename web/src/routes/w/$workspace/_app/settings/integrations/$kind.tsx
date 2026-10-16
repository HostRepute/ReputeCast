import { createFileRoute, notFound } from '@tanstack/react-router'
import { AppPage } from '@/features/integrations/AppPage'
import { isPlatform } from '@/features/integrations/shared'

/** Settings › Integrations › Zapier, Make or n8n. */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/$kind')({
  beforeLoad: ({ params }) => {
    if (!isPlatform(params.kind)) throw notFound()
  },
  component: function IntegrationRoute() {
    const { kind } = Route.useParams()
    return isPlatform(kind) ? <AppPage key={kind} kind={kind} /> : null
  },
})
