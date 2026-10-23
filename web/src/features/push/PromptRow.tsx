import { Link } from '@tanstack/react-router'
import { $api, type Schemas } from '@/api/client'
import { useT } from '@/i18n'
import { m } from './prompt.messages'

/**
 * A push prompt in Forms & pages › Forms (X22): a form type with its own
 * designer, so its row opens there, names its site, and counts readers who
 * granted permission rather than submissions.
 */
export function PushPromptRow({ f, slug, status }: { f: Schemas['SignupFormSummary']; slug: string; status: string }) {
  const t = useT()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/push/prompts/{formId}', { params: { path: { workspace: slug, formId: f.id } } }, { staleTime: 60_000, retry: false })
  const d = q.data
  return (
    <tr>
      <td className="font-semibold">
        <Link to="/w/$workspace/forms" params={{ workspace: slug }} search={{ prompt: f.id }} className="sv-name">
          {f.name}
        </Link>
      </td>
      <td>
        <span className="badge info">{t(m.typePush)}</span>
      </td>
      <td className="sv-where">{d ? (d.site?.host ?? t(m.noSite)) : <span className="k">…</span>}</td>
      <td>
        <span className={`badge ${f.status === 'live' ? 'ok' : 'mut'}`}>{status}</span>
      </td>
      <td className="nw sv-activity">{d ? t(m.listGranted, { count: d.stats.granted }) : <span className="k">…</span>}</td>
      <td className="k">—</td>
    </tr>
  )
}
