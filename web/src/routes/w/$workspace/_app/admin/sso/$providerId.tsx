import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SSOSetupPage } from '@/features/admin/sso/SSOSetupPage'

const search = z.object({
  // Set when a test sign-in returns from the identity provider.
  test: z.enum(['done']).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/admin/sso/$providerId')({
  validateSearch: search,
  component: function SSOProviderRoute() {
    const { providerId } = Route.useParams()
    const { test } = Route.useSearch()
    return <SSOSetupPage key={providerId} providerId={providerId} test={test} />
  },
})
