import { createFileRoute } from '@tanstack/react-router'
import { TransactionalMessagePage } from '@/features/transactional/MessagePage'
import { logSearch } from './-search'

/** One transactional message: its summary, timeline, request and content, next to the log. */
export const Route = createFileRoute('/w/$workspace/_app/transactional/messages/$messageId')({
  validateSearch: logSearch,
  component: function TransactionalMessageRoute() {
    const { messageId } = Route.useParams()
    const s = Route.useSearch()
    return <TransactionalMessagePage messageId={messageId} search={s} />
  },
})
