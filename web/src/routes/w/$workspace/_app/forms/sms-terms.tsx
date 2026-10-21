import { createFileRoute } from '@tanstack/react-router'
import { SmsTermsPage } from '@/features/sms/TermsPage'

/** Forms & pages › Text message terms (SMS). */
export const Route = createFileRoute('/w/$workspace/_app/forms/sms-terms')({
  component: SmsTermsPage,
})
