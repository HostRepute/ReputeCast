import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { NewTransactionalTemplatePage } from '@/features/composer/transactional/TransactionalPage'

const search = z.object({
  /** A starter's key (password-reset, magic-link…). */
  starter: z.string().max(40).optional().catch(undefined),
  /** Another template to copy (its ID or UUID). */
  from: z.string().max(64).optional().catch(undefined),
  name: z.string().max(120).optional().catch(undefined),
})

/** Creates a transactional template, then opens it in the editor. */
export const Route = createFileRoute('/w/$workspace/transactional/templates/new')({
  validateSearch: search,
  component: function NewTransactionalTemplateRoute() {
    const s = Route.useSearch()
    return <NewTransactionalTemplatePage starter={s.starter} from={s.from} name={s.name} />
  },
})
