import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { UsagePage } from '@/features/agency/UsagePage'

const search = z.object({
  month: z
    .string()
    .regex(/^\d{4}-\d{2}$/)
    .optional()
    .catch(undefined),
  open: z.string().max(64).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/agency/usage')({
  validateSearch: search,
  component: UsagePage,
})
