import { createFileRoute } from '@tanstack/react-router'
import { ConnectionsPage } from '@/features/mcpconnect/ConnectionsPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/connections')({ component: ConnectionsPage })
