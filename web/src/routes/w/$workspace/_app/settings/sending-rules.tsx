import { createFileRoute } from '@tanstack/react-router'
import { SendingRulesPage } from '@/features/sending-rules/RulesPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/sending-rules')({
  component: SendingRulesPage,
})
