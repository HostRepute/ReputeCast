import { createFileRoute } from '@tanstack/react-router'
import { MemberPage } from '@/features/team/MemberPage'

export const Route = createFileRoute('/w/$workspace/_app/settings/team/$memberId')({
  component: function MemberRoute() {
    const { memberId } = Route.useParams()
    return <MemberPage key={memberId} memberId={memberId} />
  },
})
