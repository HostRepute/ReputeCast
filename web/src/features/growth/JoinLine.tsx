import { $api } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { isImport, sourceName } from './sourceText'

/**
 * The contact profile's join line: "Joined 13 Oct 2026 · Newsletter signup ·
 * linkedin / social / flags-ga · from docs.acme.io/flags" for a signup Growth
 * saw, the stored source otherwise ("Joined 18 Sep 2026 · Brevo import").
 */
export function JoinLine({ contactId, day, sourceLabel }: { contactId: string; day: string; sourceLabel: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/contacts/{contact}/growth', { params: { path: { workspace: slug, contact: contactId } } }, { retry: false })
  const j = q.data
  if (!j || !j.attributed || isImport(j.source) || j.source.kind === 'other') return <>{t(m.joinedLine, { day, source: sourceLabel })}</>
  const parts = [t(m.joinedLine, { day, source: sourceName(j.source) })]
  if (j.utmSource || j.utmMedium || j.utmCampaign) parts.push([j.utmSource || '–', j.utmMedium || '–', j.utmCampaign || '–'].join(' / '))
  if (j.referrer) parts.push(t(m.joinedFrom, { page: j.referrer }))
  return <>{parts.join(' · ')}</>
}
