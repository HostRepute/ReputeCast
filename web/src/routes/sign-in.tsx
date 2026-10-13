import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { SignInPage } from '@/features/auth/SignInPage'

const search = z.object({
  next: z.string().optional(),
  reason: z.enum(['expired']).optional(),
  // Set by the SSO and social sign-in callbacks when the identity provider
  // fails (with a reference id for the audit log) or no account matches.
  error: z.enum(['idp', 'no_account']).optional(),
  provider: z.string().optional(),
  code: z.string().optional(),
  ref: z.string().optional(),
  email: z.string().optional(),
  // Set after a password reset.
  reset: z.enum(['done']).optional(),
})

export const Route = createFileRoute('/sign-in')({
  validateSearch: search,
  component: function SignInRoute() {
    return <SignInPage search={Route.useSearch()} />
  },
})
