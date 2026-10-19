import { $api } from '@/api/client'
import { defineMessages, T } from '@/i18n'
import { useCan, useWorkspace } from '@/lib/session'
import { GROW, mono } from './shared'

const mf = defineMessages('grow.forms', {
  refNote: 'Referral links can point here. <mono>?ref=</mono> codes are recorded.',
})

/**
 * The line under a signup form's embed code (brief §6: FormBuilder): a page
 * opened with a referral code (?ref=K7Q2MX) credits the referrer when the
 * form is used. Shown once the workspace has a referral program.
 */
export function FormRefNote() {
  const { slug } = useWorkspace()
  const can = useCan('grow:view')
  const q = $api.useQuery('get', `${GROW}/referrals`, { params: { path: { workspace: slug } } }, { enabled: can, staleTime: 60_000 })
  if (!q.data?.items.length) return null
  return (
    <span className="k">
      <T msg={mf.refNote} values={{ mono }} />
    </span>
  )
}
