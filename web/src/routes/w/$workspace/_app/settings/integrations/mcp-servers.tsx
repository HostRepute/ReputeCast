import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { MCPServersPage } from '@/features/mcpservers/MCPServersPage'

/** The OAuth callback lands here: ?server=…&connected=1, or &error=denied|token. */
const search = z.object({
  server: z.string().uuid().optional().catch(undefined),
  connected: z.literal('1').or(z.literal(1)).optional().catch(undefined),
  error: z.enum(['denied', 'token']).optional().catch(undefined),
})

/** Settings › Integrations › MCP servers: remote servers the assistant may call (not ReputeCast's own MCP server). */
export const Route = createFileRoute('/w/$workspace/_app/settings/integrations/mcp-servers')({
  validateSearch: search,
  component: function MCPServersRoute() {
    const s = Route.useSearch()
    return <MCPServersPage callback={{ server: s.server, connected: s.connected !== undefined, error: s.error }} />
  },
})
