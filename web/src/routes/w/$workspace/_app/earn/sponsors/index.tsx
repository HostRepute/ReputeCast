import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { SponsorsPage, type SponsorsSearch } from '@/features/earn/sponsors/SponsorsPage'

const search = z.object({
  tab: z.enum(['bookings', 'sponsors', 'slots']).optional().catch(undefined),
  /** The slot whose issues the Bookings tab shows. */
  slot: z.uuid().optional().catch(undefined),
  range: z.enum(['around', 'upcoming', 'recent', 'all']).optional().catch(undefined),
  /** The booking shown in the panel. */
  booking: z.uuid().optional().catch(undefined),
  /** An open issue of the slot shown in the panel. */
  issue: z.number().int().min(1).max(1_000_000).optional().catch(undefined),
})

/** Earn › Sponsors (EarnSponsors) and the booking panel (EarnBooking). */
export const Route = createFileRoute('/w/$workspace/_app/earn/sponsors/')({
  validateSearch: search,
  component: function SponsorsRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: SponsorsSearch, replace?: boolean) => void navigate({ search: next as z.infer<typeof search>, replace }), [navigate])
    return <SponsorsPage search={s} onSearch={onSearch} />
  },
})
