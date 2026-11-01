import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { CalendarPage } from '@/features/campaigns/CalendarPage'

const search = z.object({
  view: z.enum(['month', 'week']).optional().catch(undefined),
  date: z
    .string()
    .regex(/^\d{4}-\d{2}-\d{2}$/)
    .optional()
    .catch(undefined),
})

// Review 2026-11 CA-3: Campaigns › Calendar (CampaignsCalendar board).
export const Route = createFileRoute('/w/$workspace/_app/campaigns/calendar')({
  validateSearch: search,
  component: function CampaignsCalendarRoute() {
    return <CalendarPage search={Route.useSearch()} />
  },
})
