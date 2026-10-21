import { createFileRoute } from '@tanstack/react-router'
import { SmsRulesPage } from '@/features/sms/RulesPage'

/** Sending providers › SMS › Routing & rules. */
export const Route = createFileRoute('/w/$workspace/_app/providers/sms/rules')({
  component: SmsRulesPage,
})
