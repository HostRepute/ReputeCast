import { createFileRoute } from '@tanstack/react-router'
import { z } from 'zod'
import { TransactionalTemplatePage } from '@/features/composer/transactional/TransactionalPage'

const search = z.object({
  view: z.enum(['visual', 'mjml', 'html', 'text']).optional().catch(undefined),
  device: z.enum(['desktop', 'mobile']).optional().catch(undefined),
  tab: z.enum(['design', 'variables', 'testdata', 'checks', 'api']).optional().catch(undefined),
  /** A language version (its code, such as es); none is the source. */
  language: z
    .string()
    .regex(/^[a-z]{2,3}$/)
    .optional()
    .catch(undefined),
  /** Opens the publish dialog: the approval notification links here with publish=1. */
  publish: z
    .union([z.literal('1'), z.literal(1), z.literal(true)])
    .optional()
    .catch(undefined),
})

/** The transactional template editor: full screen, outside the app shell, like the composer. */
export const Route = createFileRoute('/w/$workspace/transactional/templates/$templateId')({
  validateSearch: search,
  component: function TransactionalTemplateRoute() {
    const { templateId } = Route.useParams()
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    return (
      <TransactionalTemplatePage
        templateId={templateId}
        search={{ view: s.view, device: s.device, tab: s.tab, publish: s.publish !== undefined, language: s.language }}
        onSearch={(next) =>
          void navigate({
            search: { view: next.view, device: next.device, tab: next.tab, publish: next.publish ? '1' : undefined, language: next.language },
            replace: true,
            ignoreBlocker: true,
          })
        }
      />
    )
  },
})
