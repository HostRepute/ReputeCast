import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SubscribePageTab } from '@/features/forms/PageTabs'

// ?language= edits the page's text in another workspace language.
const search = z.object({
  language: z
    .string()
    .regex(/^[a-z]{2,3}$/)
    .optional()
    .catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/forms/subscribe')({
  validateSearch: search,
  component: SubscribePageTab,
})
