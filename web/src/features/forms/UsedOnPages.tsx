import { Link } from '@tanstack/react-router'
import { $api, ApiError, errorMessage, type Schemas } from '@/api/client'
import { Badge, type Tone } from '@/components/ui/Badge'
import { T, useT, type MessageDescriptor } from '@/i18n'
import { andList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { builderLink } from '@/features/pages/format'
import type { SignupForm } from './common'
import { m } from './leadmagnet.messages'
import './leadmagnet.css'

type FormPage = Schemas['FormPage']

const statusBadge: Record<FormPage['status'], [Tone, MessageDescriptor]> = {
  published: ['ok', m.statusPublished],
  scheduled: ['info', m.statusScheduled],
  draft: ['neutral', m.statusDraft],
  unpublished: ['neutral', m.statusUnpublished],
}

/**
 * "Used on pages" (§6 Form builder): the pages whose signup form block uses
 * this form, each opening in the page builder. A published page whose
 * live version doesn't use the form yet says so.
 */
export function FormPagesSection({ form }: { form: SignupForm }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/forms/{form}/pages', { params: { path: { workspace: slug, form: form.id } } })
  // Without subscribers:view the list isn't readable: the section stays out.
  if (q.error instanceof ApiError && q.error.status === 403) return null
  const items = q.data?.items
  return (
    <div className="fb-sec">
      <span className="fb-title">{t(m.pagesTitle)}</span>
      {!items ? (
        q.isError ? <span className="k">{t(m.pagesFailed, { error: errorMessage(q.error) })}</span> : null
      ) : items.length === 0 ? (
        <span className="k">
          <T
            msg={m.pagesNone}
            values={{
              link: (c) => (
                <Link to="/w/$workspace/forms/pages" params={{ workspace: slug }}>
                  {c}
                </Link>
              ),
            }}
          />
        </span>
      ) : (
        <ul className="fl-pages">
          {items.map((p) => {
            const [tone, label] = statusBadge[p.status]
            return (
              <li key={p.id}>
                <span className="nm">
                  <Link {...builderLink(slug, p.id)} aria-label={t(m.pageOpen, { name: p.name })}>
                    {p.name}
                  </Link>
                  <span className="k mono">{p.status === 'published' && !p.live ? t(m.draftOnly, { path: p.path }) : p.path}</span>
                </span>
                <Badge tone={tone}>{t(label)}</Badge>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

/** "Your own pages on news.acme.io and … are always allowed." (the server's own origins). */
export function OwnOriginsNote({ origins }: { origins: string[] }) {
  const t = useT()
  if (!origins.length) return null
  return <span className="k">{t(m.ownOrigins, { hosts: andList(origins) })}</span>
}
