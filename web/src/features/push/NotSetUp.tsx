import { $api, errorMessage } from '@/api/client'
import { Button, ButtonLink } from '@/components/ui/Button'
import { toast } from '@/components/ui/Toast'
import { useT } from '@/i18n'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { bareHost, useInvalidatePush, type PushSetup } from './common'
import { m } from './messages'

/**
 * Push not set up (board PushStates): what it takes, and the one button
 * that generates the keys. Turning push on asks nobody anything.
 */
export function NotSetUp({ data }: { data: PushSetup }) {
  const t = useT()
  const { slug } = useWorkspace()
  const invalidate = useInvalidatePush()
  const reason = usePermissionReason('providers:manage', data.helpers)
  const enable = $api.useMutation('post', '/api/v1/workspaces/{workspace}/push/enable')
  const website = data.website ? bareHost(data.website) : data.sites.find((s) => s.kind === 'publication')?.host
  return (
    <section className="card px-empty" aria-labelledby="push-empty-title">
      <svg width="64" height="56" viewBox="0 0 64 56" aria-hidden="true">
        <rect x="8" y="10" width="40" height="30" rx="4" fill="none" stroke="#9AA3B2" strokeWidth="2" />
        <path d="M8 17h40" stroke="#9AA3B2" strokeWidth="2" />
        <path d="M46 30v-6a8 8 0 0 0-16 0v6l-3 3h22z" fill="#fff" stroke="#0F766E" strokeWidth="2.2" strokeLinejoin="round" />
        <path d="M35 37a3 3 0 0 0 6 0" fill="none" stroke="#0F766E" strokeWidth="2.2" />
      </svg>
      <h2 className="tt" id="push-empty-title">
        {t(m.notSetUpTitle)}
      </h2>
      <p className="ds">{website ? t(m.notSetUpText, { website }) : t(m.notSetUpTextNoWebsite)}</p>
      <ol>
        {[m.stepKeys, m.stepSites, m.stepPrompt, m.stepFirst].map((s, i) => (
          <li key={s.id} className="chip">
            <span className="num">{i + 1}</span>
            {t(s)}
          </li>
        ))}
      </ol>
      <div className="mt-2 flex flex-wrap justify-center gap-2.5">
        <Button
          variant="primary"
          size="lg"
          disabledReason={reason}
          loading={enable.isPending}
          busyLabel={t(m.turningOn)}
          onClick={() =>
            enable.mutate(
              { params: { path: { workspace: slug } }, body: {} },
              {
                onSuccess: () => {
                  void invalidate()
                  toast({ kind: 'success', title: t(m.pushOn) })
                },
                onError: (e) => toast({ kind: 'error', title: errorMessage(e) }),
              },
            )
          }
        >
          {t(m.turnOn)}
        </Button>
        <ButtonLink size="lg" to="/w/$workspace/forms" params={{ workspace: slug }}>
          {t(m.seeDesigner)}
        </ButtonLink>
      </div>
      <span className="k">
        {t(m.turnOnNote)}
      </span>
    </section>
  )
}
