import { useState } from 'react'
import { $api, errorMessage, type Schemas } from '@/api/client'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Feedback'
import { Grid } from '@/components/ui/Layout'
import { toast } from '@/components/ui/Toast'
import { count } from '@/lib/format'
import { usePermissionReason, useWorkspace } from '@/lib/session'
import { t as tt, useT } from '@/i18n'
import { useInvalidateForms } from './common'
import { m } from './builder.messages'

/** "Loading · building the preview": two cards shaped like the builder. */
export function FormsLoading({ label }: { label?: string }) {
  const t = useT()
  return (
    <Grid min={300} gap={16} aria-busy="true">
      <span className="vh" role="status">
        {label ?? t(m.loadingForms)}
      </span>
      <div className="card flex flex-col gap-2.5 p-4">
        <Skeleton w={140} h={16} />
        <Skeleton h={34} />
        <Skeleton h={34} />
        <Skeleton h={46} w="60%" />
      </div>
      <div className="card flex flex-col gap-2.5 p-4">
        <Skeleton w={120} h={16} />
        <Skeleton h={80} />
        <Skeleton h={30} w="40%" />
      </div>
    </Grid>
  )
}

/** "No signup forms yet": the hosted page already works; make a form for your site. */
export function FormsEmpty({ hostedUrl, hostedLabel, onCreate, createReason }: { hostedUrl: string; hostedLabel: string; onCreate: () => void; createReason?: string }) {
  const t = useT()
  return (
    <section className="card flex flex-col items-center gap-2.5 border-dashed px-7 py-8 text-center">
      <h2 className="m-0 text-[17px] font-bold">{t(m.emptyTitle)}</h2>
      <p className="max-w-[520px] text-ink-2">{t(m.emptyBody, { host: hostedLabel })}</p>
      <div className="flex flex-wrap justify-center gap-2.5">
        <Button variant="primary" onClick={onCreate} disabledReason={createReason}>
          {t(m.createAForm)}
        </Button>
        <a className="btn" href={hostedUrl} target="_blank" rel="noreferrer">
          {t(m.openHosted)}
        </a>
      </div>
    </section>
  )
}

/**
 * "41 submissions from staging.acme.io were rejected today": allow the site,
 * or keep it blocked (the alert goes away until it submits again tomorrow).
 */
export function BlockedOriginAlert({ form, blocked }: { form: Schemas['SignupForm']; blocked: Schemas['SignupFormBlockedOrigin'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('forms:edit')
  const invalidate = useInvalidateForms()
  const params = { path: { workspace: slug, form: form.id } }
  const allow = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/allowed-origins')
  const keep = $api.useMutation('post', '/api/v1/workspaces/{workspace}/forms/{form}/blocked-origins/keep')
  const run = async (which: 'allow' | 'keep') => {
    try {
      if (which === 'allow') {
        await allow.mutateAsync({ params, body: { origin: blocked.origin } })
        toast({ kind: 'success', title: tt(m.canSubmit, { origin: blocked.origin, name: form.name }), detail: tt(m.addedToAllowed) })
      } else {
        await keep.mutateAsync({ params, body: { origin: blocked.origin } })
        toast({ kind: 'success', title: tt(m.staysBlocked, { origin: blocked.origin }), detail: tt(m.comesBack) })
      }
      await invalidate()
    } catch (e) {
      toast({ kind: 'error', title: tt(m.domainsFailed), detail: errorMessage(e), retry: () => void run(which) })
    }
  }
  const n = blocked.count
  return (
    <div role="alert" className="card fb-alert alert-bad">
      <div className="flex items-center gap-2.5 max-md:flex-wrap">
        <Badge tone="bad">{t(m.blocked)}</Badge>
        <span className="font-semibold">{t(m.rejectedToday, { n, count: count(n), origin: blocked.origin })}</span>
      </div>
      <span className="k">{t(m.notAllowed, { origin: blocked.origin, name: form.name })}</span>
      <div className="flex gap-2 max-md:flex-wrap">
        <Button size="sm" variant="primary" onClick={() => void run('allow')} loading={allow.isPending} disabledReason={reason}>
          {t(m.allow, { origin: blocked.origin })}
        </Button>
        <Button size="sm" onClick={() => void run('keep')} loading={keep.isPending} disabledReason={reason}>
          {t(m.keepBlocked)}
        </Button>
      </div>
    </div>
  )
}

/**
 * "Cloudflare rejected the site key (error 110200)": spam protection fell
 * back to the honeypot; paste the right key. The secret stays as stored.
 */
export function TurnstileAlert({ turnstile }: { turnstile: Schemas['PublicTurnstile'] }) {
  const t = useT()
  const { slug } = useWorkspace()
  const reason = usePermissionReason('forms:edit')
  const invalidate = useInvalidateForms()
  const save = $api.useMutation('put', '/api/v1/workspaces/{workspace}/public-pages/turnstile')
  const [key, setKey] = useState('')
  const [error, setError] = useState<string>()
  const submit = async () => {
    const siteKey = key.trim()
    if (!siteKey || save.isPending) return
    setError(undefined)
    try {
      const out = await save.mutateAsync({ params: { path: { workspace: slug } }, body: { siteKey } })
      await invalidate()
      if (out.turnstile.status === 'ok') {
        setKey('')
        toast({ kind: 'success', title: tt(m.turnstileSaved), detail: tt(m.turnstileAccepted) })
      } else {
        setError(tt(m.checkKey, { error: out.turnstile.error ?? '' }))
      }
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <div role="alert" className="card fb-alert alert-warn">
      <div className="flex items-center gap-2.5 max-md:flex-wrap">
        <Badge tone="warn">{'Turnstile'}</Badge>
        <span className="font-semibold">{turnstile.error || t(m.turnstileRejected)}</span>
      </div>
      <span className="k">{t(m.turnstileFellBack)}</span>
      <label className="lbl">
        {t(m.siteKey)}
        <input
          className="fb-in mono in-warn"
          placeholder={'0x4AAAAAAA…'}
          value={key}
          disabled={Boolean(reason)}
          title={reason}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? 'turnstile-key-err' : undefined}
          onChange={(e) => setKey(e.target.value)}
          onBlur={() => void submit()}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              void submit()
            }
          }}
        />
      </label>
      {error ? (
        <span className="err" id="turnstile-key-err" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  )
}
