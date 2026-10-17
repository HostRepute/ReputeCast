import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { FormBuilderPage } from '@/features/forms/FormBuilderPage'
import { FormsList } from '@/features/surveys/FormsList'

const search = z.object({
  form: z.string().uuid().optional().catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/forms/')({
  validateSearch: search,
  component: function FormsRoute() {
    // Forms & pages › Forms lists signup forms and surveys (design
    // expansion, X11); a form opens in the builder.
    const form = Route.useSearch().form
    return form ? <FormBuilderPage formId={form} /> : <FormsList />
  },
})
