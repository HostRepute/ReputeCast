import { useNavigate } from '@tanstack/react-router'
import { http } from '@/api/client'
import { OptionCard } from '@/components/ui/Field'
import { useChannelSetUp } from '@/features/channels/hooks'
import { useT } from '@/i18n'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'

/**
 * "New campaign" offers a text message (phase 8) next to Regular and
 * Recurring digest, once texting is set up.
 */
export function useTextOption(): boolean {
  return useChannelSetUp('sms')
}

export function TextOption({ checked, onChange }: { checked: boolean; onChange: () => void }) {
  const t = useT()
  return (
    <OptionCard name="campaign-kind" value="sms" checked={checked} onChange={onChange} title={t(m.newText)} description={t(m.newTextBody)}>
      <span className="k">{t(m.newTextHint)}</span>
    </OptionCard>
  )
}

/** Creates a text campaign and opens its composer; a failure throws for the dialog to show. */
export function useCreateText(): () => Promise<void> {
  const t = useT()
  const { slug } = useWorkspace()
  const navigate = useNavigate()
  return async () => {
    const { data } = await http.POST('/api/v1/workspaces/{workspace}/sms-campaigns', { params: { path: { workspace: slug } }, body: { name: t(m.untitledText) } })
    void navigate({ to: '/w/$workspace/campaigns/$campaignId/sms', params: { workspace: slug, campaignId: data!.id } })
  }
}
