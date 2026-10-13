import { createFileRoute } from '@tanstack/react-router'
import { McpPage } from '@/features/keys/McpPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/mcp')({
  component: McpPage,
})
