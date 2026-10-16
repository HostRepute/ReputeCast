import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { FormBuilderPage } from '@/features/forms/FormBuilderPage'

const search = z.object({
  form: z.string().uuid().optional().catch(undefined),
  // The language view (FormBuilderLanguages): the copy in one language.
  language: z
    .string()
    .regex(/^[a-z]{2,3}$/)
    .optional()
    .catch(undefined),
})

export const Route = createFileRoute('/w/$workspace/_app/forms/')({
  validateSearch: search,
  component: function FormsRoute() {
    const { form, language } = Route.useSearch()
    return <FormBuilderPage formId={form} language={language} />
  },
})
