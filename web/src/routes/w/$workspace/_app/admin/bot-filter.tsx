import { createFileRoute } from '@tanstack/react-router'
import { BotFilterPage } from '@/features/admin/botfilter/BotFilterPage'

export const Route = createFileRoute('/w/$workspace/_app/admin/bot-filter')({
  component: BotFilterPage,
})
