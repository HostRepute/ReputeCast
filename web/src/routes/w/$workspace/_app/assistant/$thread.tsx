import { createFileRoute } from '@tanstack/react-router'
import { AssistantPage } from '@/features/assistant/AssistantPage'

export const Route = createFileRoute('/w/$workspace/_app/assistant/$thread')({
  component: function AssistantRoute() {
    const { thread } = Route.useParams()
    return <AssistantPage key={thread} thread={thread} />
  },
})
