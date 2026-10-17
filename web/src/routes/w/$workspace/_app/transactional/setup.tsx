import { createFileRoute } from '@tanstack/react-router'
import { TransactionalSetupPage } from '@/features/transactional/SetupPage'

/** Transactional › Setup: the HTTP API, the keys that can send, code samples, the SMTP relay and the API's errors. */
export const Route = createFileRoute('/w/$workspace/_app/transactional/setup')({
  component: TransactionalSetupPage,
})
