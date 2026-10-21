import { $api } from '@/api/client'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './contact.messages'

/**
 * The contacts filters "Push subscribed" (a linked browser subscription)
 * and "WhatsApp consent" (on a list), once the channel is set up (X22).
 */
export function ChannelChips({
  push,
  onPush,
  whatsapp,
  onWhatsApp,
}: {
  push: boolean
  onPush: (on: boolean) => void
  whatsapp: boolean
  onWhatsApp: (on: boolean) => void
}) {
  const t = useT()
  const { slug } = useWorkspace()
  const q = $api.useQuery('get', '/api/v1/workspaces/{workspace}/channels', { params: { path: { workspace: slug } } }, { staleTime: 60_000 })
  const set = new Set((q.data?.channels ?? []).filter((c) => c.setUp).map((c) => c.channel))
  const showPush = set.has('push') || push
  const showWhatsApp = set.has('whatsapp') || whatsapp
  if (!showPush && !showWhatsApp) return null
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t(m.filterLabel)}>
      <span className="k mr-1">{t(m.filterLabel)}</span>
      {showPush ? (
        <button type="button" className="vchip" aria-pressed={push} onClick={() => onPush(!push)}>
          {t(m.filterPush)}
        </button>
      ) : null}
      {showWhatsApp ? (
        <button type="button" className="vchip" aria-pressed={whatsapp} onClick={() => onWhatsApp(!whatsapp)}>
          {t(m.filterWhatsApp)}
        </button>
      ) : null}
    </div>
  )
}
