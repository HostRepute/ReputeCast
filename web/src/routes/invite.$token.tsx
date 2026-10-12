import { createFileRoute } from '@tanstack/react-router'
import { InvitePage } from '@/features/auth/InvitePage'

export const Route = createFileRoute('/invite/$token')({
  component: function InviteRoute() {
    const { token } = Route.useParams()
    return <InvitePage token={token} />
  },
})
