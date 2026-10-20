import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import { PostComposerPage, type PostComposerSearch } from '@/features/website/composer/PostComposerPage'

const search = z.object({
  /** Edit or Preview on web. */
  view: z.enum(['preview']).optional().catch(undefined),
  /** The previewed reader. */
  as: z.enum(['visitor', 'free', 'paid']).optional().catch(undefined),
  device: z.enum(['desktop', 'mobile']).optional().catch(undefined),
  /** The right panel's tab: Style, Settings or Web. */
  panel: z.enum(['style', 'settings', 'web']).optional().catch(undefined),
  /** Opens the publish dialog (Publish, or Request approval). */
  publish: z.boolean().optional().catch(undefined),
})

/**
 * A web-only post in the composer's post mode (ComposerPublishToWeb): full
 * screen like the campaign composer. Approval and notification links open
 * it here.
 */
export const Route = createFileRoute('/w/$workspace/forms/website/posts/$post')({
  validateSearch: search,
  component: function PostComposerRoute() {
    const { post } = Route.useParams()
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: PostComposerSearch) => void navigate({ search: next as z.infer<typeof search>, replace: true }), [navigate])
    return <PostComposerPage postId={post} search={s} onSearch={onSearch} />
  },
})
