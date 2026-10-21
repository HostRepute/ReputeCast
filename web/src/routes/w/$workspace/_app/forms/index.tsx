import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { FormBuilderPage } from '@/features/forms/FormBuilderPage'
import { PushPromptPage } from '@/features/push/PromptPage'
import { FormsList } from '@/features/surveys/FormsList'

const search = z.object({
  form: z.string().uuid().optional().catch(undefined),
  // A push prompt in its designer (design expansion phase 8, X22).
  prompt: z.string().uuid().optional().catch(undefined),
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
    // Forms & pages › Forms lists signup forms and surveys (design
    // expansion, X11); a form, or a language view, opens in the builder.
    const { form, language, prompt } = Route.useSearch()
    if (prompt) return <PushPromptPage key={prompt} formId={prompt} />
    return form || language ? <FormBuilderPage formId={form} language={language} /> : <FormsList />
  },
})
