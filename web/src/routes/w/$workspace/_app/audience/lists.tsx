import { createFileRoute } from '@tanstack/react-router'
import { ListsFieldsPage } from '@/features/audience/ListsFieldsPage'

export const Route = createFileRoute('/w/$workspace/_app/audience/lists')({
  component: ListsFieldsPage,
})
