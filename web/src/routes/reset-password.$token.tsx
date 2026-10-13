import { createFileRoute } from '@tanstack/react-router'
import { ResetTokenPage } from '@/features/auth/ResetPages'

export const Route = createFileRoute('/reset-password/$token')({
  component: function ResetTokenRoute() {
    const { token } = Route.useParams()
    return <ResetTokenPage token={token} />
  },
})
