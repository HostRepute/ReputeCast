import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { FormBuilderPage } from '@/features/forms/FormBuilderPage'

const search = z.object({
  form: z.string().uuid().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/forms/')({
  validateSearch: search,
  component: function FormsRoute() {
    return <FormBuilderPage formId={Route.useSearch().form} />
  },
})
