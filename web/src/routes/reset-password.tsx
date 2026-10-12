import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'

import { NotBuilt } from '@/components/shell/NotBuilt'

export const Route = createFileRoute('/reset-password')({
  validateSearch: z.object({ email: z.string().optional() }),
  component: () => <NotBuilt screen="Reset password" />,
})
