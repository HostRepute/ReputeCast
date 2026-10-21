import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { SmsEventsPage, type EventSearch } from '@/features/sms/EventsPage'

const search = z.object({
  provider: z.uuid().optional().catch(undefined),
  kind: z.enum(['all', 'receipts', 'keywords', 'replies', 'failures']).optional().catch(undefined),
  page: z.coerce.number().int().min(1).optional().catch(undefined),
})

/** Sending providers › SMS › Events: raw provider events beside normalized ones. */
export const Route = createFileRoute('/w/$workspace/_app/providers/sms/events')({
  validateSearch: search,
  component: SmsEventsRoute,
})

function SmsEventsRoute() {
  const s = Route.useSearch()
  const navigate = Route.useNavigate()
  const onSearch = useCallback((next: EventSearch) => void navigate({ search: next }), [navigate])
  return <SmsEventsPage search={s} onSearch={onSearch} />
}
