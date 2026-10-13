import { createFileRoute } from '@tanstack/react-router'
import { ContactProfilePage } from '@/features/audience/ContactProfilePage'

export const Route = createFileRoute('/w/$workspace/_app/audience/contacts/$contactId')({
  component: function ContactRoute() {
    const { contactId } = Route.useParams()
    return <ContactProfilePage key={contactId} contactId={contactId} />
  },
})
