import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { LibraryPage } from '@/features/agency/LibraryPage'

const search = z.object({
  kind: z.enum(['section', 'template', 'automation', 'sms', 'transactional']).optional().catch(undefined),
  item: z.string().max(64).optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/agency/library')({
  validateSearch: search,
  component: LibraryPage,
})
