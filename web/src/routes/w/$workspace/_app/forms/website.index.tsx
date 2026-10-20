import { createFileRoute } from '@tanstack/react-router'
import { useCallback } from 'react'
import { z } from 'zod'
import type { PostsSearch } from '@/features/website/PostsPage'
import { WebsiteTab } from '@/features/website/WebsiteTab'

const search = z.object({
  status: z.enum(['published', 'scheduled', 'draft', 'unpublished']).optional().catch(undefined),
  q: z.string().max(200).optional().catch(undefined),
  visibility: z.enum(['public', 'members', 'paid']).optional().catch(undefined),
  tag: z.string().max(80).optional().catch(undefined),
  source: z.enum(['campaign', 'web']).optional().catch(undefined),
  page: z.number().int().min(1).max(100000).optional().catch(undefined),
  /** Publish past campaigns is open. */
  past: z.boolean().optional().catch(undefined),
  /** A post's card (waiting for approval, unpublished, scheduled). */
  post: z.uuid().optional().catch(undefined),
})

/** Forms & pages › Website: off, turning on, or the posts (WebsitePosts, WebsiteStates). */
export const Route = createFileRoute('/w/$workspace/_app/forms/website/')({
  validateSearch: search,
  component: function WebsiteRoute() {
    const s = Route.useSearch()
    const navigate = Route.useNavigate()
    const onSearch = useCallback((next: PostsSearch, replace?: boolean) => void navigate({ search: next as z.infer<typeof search>, replace }), [navigate])
    return <WebsiteTab search={s} onSearch={onSearch} />
  },
})
