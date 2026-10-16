import { createFileRoute } from '@tanstack/react-router'
import { TransactionalSettingsPage } from '@/features/transactional/SettingsPage'

/** Transactional › Settings: categories, recipients and retention, delivery alerts, defaults for new templates. */
export const Route = createFileRoute('/w/$workspace/_app/transactional/settings')({
  component: TransactionalSettingsPage,
})
