import { Link } from '@tanstack/react-router'
import { useState } from 'react'
import { $api, errorMessage, http } from '@/api/client'
import { Icon } from '@/components/icons/Icon'
import { toast } from '@/components/ui/Toast'
import { T, useT } from '@/i18n'
import { orList } from '@/lib/format'
import { useWorkspace } from '@/lib/session'
import { m } from './messages'
import { base, money, useRefreshVerification, type FormSetting } from './shared'

function useFormSetting(formKey: string) {
  const { slug } = useWorkspace()
  return $api.useQuery('get', `${base}/forms/{form}`, { params: { path: { workspace: slug, form: formKey } } })
}

/** Form builder › Spam protection: check each address with the forms' service as people sign up. */
export function FormAddressCheck({ formKey, formName }: { formKey: string; formName: string }) {
  const t = useT()
  const { slug } = useWorkspace()
  const refresh = useRefreshVerification()
  const q = useFormSetting(formKey)
  const [busy, setBusy] = useState(false)
  const s = q.data
  if (!s) return null
  if (!s.service) {
    return (
      <div className="flex flex-col gap-1 rounded-lg border border-line px-3 py-2.5">
        <span className="text-[13px]">{t(m.formCheckNone)}</span>
        <span className="k">
          <T
            msg={m.formCheckNoneBody}
            values={{
              link: (c) => (
                <Link to="/w/$workspace/settings/integrations/verification" params={{ workspace: slug }}>
                  {c}
                </Link>
              ),
            }}
          />
        </span>
      </div>
    )
  }
  const name = s.serviceName ?? ''
  const why = s.canChange ? undefined : s.helpers.length ? t(m.formCheckCant, { helpers: orList(s.helpers) }) : t(m.formCheckCantNobody)
  const toggle = async (enabled: boolean) => {
    setBusy(true)
    const res = await http.PUT(`${base}/forms/{form}`, { params: { path: { workspace: slug, form: formKey }, query: { name: formName } }, body: { enabled } })
    setBusy(false)
    if (!res.data) {
      toast({ kind: 'error', title: errorMessage(res.error) })
      return
    }
    void refresh()
    toast({ kind: 'success', title: t(enabled ? m.formCheckOn : m.formCheckOff, { form: formName }) })
  }
  return (
    <>
      <div className={s.enabled ? 'flex flex-col gap-2 rounded-lg border border-accent bg-accent-tint p-3' : 'flex flex-col gap-2 rounded-lg border border-line p-3'}>
        <div className="fb-check start">
          <input
            id={`${formKey}-check`}
            type="checkbox"
            checked={s.enabled}
            disabled={busy || Boolean(why)}
            aria-describedby={`${formKey}-check-h${why ? ` ${formKey}-why` : ''}`}
            onChange={(e) => void toggle(e.target.checked)}
          />
          <span className="flex flex-col gap-px">
            <label htmlFor={`${formKey}-check`}>{t(m.formCheckTitle, { name })}</label>
            <span className="k" id={`${formKey}-check-h`}>
              <T msg={m.formCheckBody} values={{ name, price: money(s.price, true), code: (c) => <span className="mono">{c}</span> }} />
            </span>
          </span>
        </div>
        <span className="k pl-[25px] text-ink-nav!">{t(m.formCheckRules, { name })}</span>
        {why ? (
          <span className="k pl-[25px]" id={`${formKey}-why`}>
            {why}
          </span>
        ) : null}
      </div>
      <span className="k">{t(m.formCheckFoot, { name })}</span>
    </>
  )
}

/** The preview's amber note while the service isn't answering. */
export function FormChecksPaused({ formKey }: { formKey: string }) {
  const t = useT()
  const q = useFormSetting(formKey)
  const s = q.data
  if (!s?.enabled || !s.down) return null
  return (
    <div className="note w" role="note">
      <Icon name="alertCircle" size={16} className="mt-px flex-none text-warn-bar" />
      <div className="flex min-w-0 flex-[1_1_320px] flex-col gap-1">
        <b className="font-bold">{t(m.formPaused)}</b>
        <span>{t(m.formPausedBody, { name: s.serviceName ?? '', count: s.waiting })}</span>
      </div>
    </div>
  )
}

/** "Asked to fix their address": a number beside the form's views and submissions. */
export function FormAskedToFix({ formKey }: { formKey: string }) {
  const t = useT()
  const q = useFormSetting(formKey)
  const s: FormSetting | undefined = q.data
  if (!s?.enabled) return null
  return (
    <div>
      <div className="k">{t(m.formStatAsked)}</div>
      <div className="n text-[18px]">{s.askedToFix.toLocaleString('en-US')}</div>
    </div>
  )
}
