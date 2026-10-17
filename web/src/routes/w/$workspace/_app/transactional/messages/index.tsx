import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { TransactionalMessagesPage, type LogSearch } from '@/features/transactional/MessagesPage'
import { logSearch } from './-search'

/** Transactional › Messages: the log, live, with its filters in the URL. */
export const Route = createFileRoute('/w/$workspace/_app/transactional/messages/')({
  validateSearch: logSearch,
  component: function TransactionalMessagesRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: LogSearch) => void navigate({ search: next, replace: true }), [navigate])
    return <TransactionalMessagesPage search={s} onSearch={onSearch} />
  },
})
