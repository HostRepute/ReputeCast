import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { ConsentPage } from '@/features/mcpconnect/ConsentPage'

export const Route = createFileRoute('/mcp-consent')({
  validateSearch: z.object({ request: z.string().max(100).catch('') }),
  component: function ConsentRoute() { return <ConsentPage key={Route.useSearch().request} request={Route.useSearch().request} /> },
})
